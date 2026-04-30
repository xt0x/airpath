package awsintegration

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"
	"airpath/services/internal/flightaware"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestDynamoDBRepositoriesReadAndWriteFlightsLookupPositionsAndUsageBudget(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryDynamoDBClient()
	repo := NewScopedDynamoDBRepository(client, DynamoDBTables{
		Flights:         "Flights",
		FlightLookup:    "FlightLookup",
		FlightPositions: "FlightPositions",
		UsageBudget:     "UsageBudget",
	}, application.UsageBudgetScope{
		Environment: "dev",
		Month:       "2026-04",
	})

	flight := testFlight("iflg_aws_1", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	if err := repo.PutFlightLookup(ctx, "ident#ANA110", flight.FlightID); err != nil {
		t.Fatalf("PutFlightLookup() error = %v", err)
	}
	position := domain.FlightPosition{
		FlightID:  flight.FlightID,
		Latitude:  45.123,
		Longitude: 160.456,
		Timestamp: "2026-04-29T00:05:00Z",
		Source:    domain.PositionSourceFlightAwarePosition,
	}
	olderPosition := position
	olderPosition.Timestamp = "2026-04-29T00:01:00Z"
	olderPosition.Latitude = 44.5
	if err := repo.PutPosition(ctx, olderPosition); err != nil {
		t.Fatalf("PutPosition(older) error = %v", err)
	}
	if err := repo.PutPosition(ctx, position); err != nil {
		t.Fatalf("PutPosition() error = %v", err)
	}
	usage := application.UsageStatus{
		Budget: application.UsageBudgetStatus{
			Currency:                 "USD",
			EstimatedMonthToDateCost: 2.5,
			SoftStopThreshold:        4,
			Stopped:                  false,
		},
		FetchingEnabled: true,
	}
	if err := repo.PutMonthlyUsageStatus(ctx, application.UsageBudgetScope{Environment: "dev", Month: "2026-04"}, usage); err != nil {
		t.Fatalf("PutMonthlyUsageStatus() error = %v", err)
	}

	gotFlight, cache, err := repo.GetFlight(ctx, flight.FlightID)
	if err != nil {
		t.Fatalf("GetFlight() error = %v", err)
	}
	if gotFlight.Ident != "ANA110" || cache.Source != application.CacheSourceCache {
		t.Fatalf("GetFlight() = %#v, cache=%#v", gotFlight, cache)
	}

	search, _, err := repo.SearchByIdent(ctx, "ANA110")
	if err != nil {
		t.Fatalf("SearchByIdent() error = %v", err)
	}
	if len(search) != 1 || search[0].FlightID != flight.FlightID {
		t.Fatalf("SearchByIdent() = %#v", search)
	}

	latest, _, err := repo.GetLatestPosition(ctx, flight.FlightID)
	if err != nil {
		t.Fatalf("GetLatestPosition() error = %v", err)
	}
	if latest == nil || latest.Timestamp != "2026-04-29T00:05:00Z" {
		t.Fatalf("latest = %#v", latest)
	}
	history, _, err := repo.ListPositions(ctx, flight.FlightID, ptrString("2026-04-29T00:02:00Z"), 10)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != 1 || history[0].Timestamp != "2026-04-29T00:05:00Z" {
		t.Fatalf("history = %#v, want only the newer position", history)
	}

	gotUsage, err := repo.GetUsageStatus(ctx)
	if err != nil {
		t.Fatalf("GetUsageStatus() error = %v", err)
	}
	if gotUsage.Budget.EstimatedMonthToDateCost != 2.5 || !gotUsage.FetchingEnabled {
		t.Fatalf("usage = %#v", gotUsage)
	}
}

func TestDynamoDBRepositorySearchByIdentSkipsStaleLookupRows(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{
		Flights:      "Flights",
		FlightLookup: "FlightLookup",
	})
	if err := repo.PutFlightLookup(ctx, "ident#ANA110", "iflg_expired"); err != nil {
		t.Fatalf("PutFlightLookup() error = %v", err)
	}

	flights, cache, err := repo.SearchByIdent(ctx, "ANA110")
	if err != nil {
		t.Fatalf("SearchByIdent() error = %v, want stale lookup row skipped", err)
	}
	if len(flights) != 0 {
		t.Fatalf("SearchByIdent() = %#v, want no flights for stale lookup row", flights)
	}
	if cache.Freshness != application.CacheFreshnessMiss {
		t.Fatalf("cache freshness = %q, want miss", cache.Freshness)
	}
}

func TestDynamoDBRepositoryTreatsTTLExpiredFlightsAsCacheMisses(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{
		Flights:      "Flights",
		FlightLookup: "FlightLookup",
	})
	repo.now = func() time.Time { return time.Date(2026, 4, 29, 0, 5, 0, 0, time.UTC) }
	expiredTTL := domain.EpochSeconds(time.Date(2026, 4, 29, 0, 4, 59, 0, time.UTC).Unix())
	flight := testFlight("iflg_expired_ttl", "ANA110")
	flight.TTL = &expiredTTL
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	if err := repo.PutFlightLookup(ctx, "ident#ANA110", flight.FlightID); err != nil {
		t.Fatalf("PutFlightLookup() error = %v", err)
	}

	_, cache, err := repo.GetFlight(ctx, flight.FlightID)
	if !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("GetFlight(expired) error = %v, want ErrNotFound", err)
	}
	if cache.Freshness != application.CacheFreshnessMiss {
		t.Fatalf("GetFlight(expired) cache freshness = %q, want miss", cache.Freshness)
	}

	flights, cache, err := repo.SearchByIdent(ctx, "ANA110")
	if err != nil {
		t.Fatalf("SearchByIdent(expired) error = %v", err)
	}
	if len(flights) != 0 {
		t.Fatalf("SearchByIdent(expired) = %#v, want expired target skipped", flights)
	}
	if cache.Freshness != application.CacheFreshnessMiss {
		t.Fatalf("SearchByIdent(expired) cache freshness = %q, want miss", cache.Freshness)
	}
}

func TestDynamoDBRepositorySkipsTTLExpiredPositions(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	repo.now = func() time.Time { return time.Date(2026, 4, 29, 0, 5, 0, 0, time.UTC) }
	flightID := domain.FlightID("iflg_position_ttl")
	expiredTTL := domain.EpochSeconds(time.Date(2026, 4, 29, 0, 4, 59, 0, time.UTC).Unix())
	validTTL := domain.EpochSeconds(time.Date(2026, 4, 29, 0, 6, 0, 0, time.UTC).Unix())
	expiredLatest := domain.FlightPosition{
		FlightID:  flightID,
		Latitude:  45.123,
		Longitude: 160.456,
		Timestamp: "2026-04-29T00:05:00Z",
		Source:    domain.PositionSourceFlightAwarePosition,
		TTL:       &expiredTTL,
	}
	validOlder := domain.FlightPosition{
		FlightID:  flightID,
		Latitude:  44.5,
		Longitude: 159.5,
		Timestamp: "2026-04-29T00:04:00Z",
		Source:    domain.PositionSourceFlightAwareTrack,
		TTL:       &validTTL,
	}
	if err := repo.PutPosition(ctx, expiredLatest); err != nil {
		t.Fatalf("PutPosition(expiredLatest) error = %v", err)
	}
	if err := repo.PutPosition(ctx, validOlder); err != nil {
		t.Fatalf("PutPosition(validOlder) error = %v", err)
	}

	latest, _, err := repo.GetLatestPosition(ctx, flightID)
	if err != nil {
		t.Fatalf("GetLatestPosition() error = %v", err)
	}
	if latest == nil || latest.Timestamp != validOlder.Timestamp {
		t.Fatalf("latest = %#v, want unexpired older position", latest)
	}
	history, cache, err := repo.ListPositions(ctx, flightID, nil, 10)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != 1 || history[0].Timestamp != validOlder.Timestamp {
		t.Fatalf("history = %#v, want only unexpired position", history)
	}
	if cache.Freshness != application.CacheFreshnessFresh {
		t.Fatalf("cache freshness = %q, want fresh for remaining unexpired position", cache.Freshness)
	}
}

func TestDynamoDBRepositorySummaryWriteDoesNotClearActiveLeaseAfterStaleRead(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{Flights: "Flights"})
	flight := testFlight("iflg_summary_stale_lease", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	staleObserved := flight
	if _, leased, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z"); err != nil || !leased {
		t.Fatalf("AcquireFetchLease() leased=%v error=%v, want lease", leased, err)
	}

	summary := staleObserved
	summary.Status = "Delayed"
	summary.UpdatedAt = "2026-04-29T00:06:00Z"
	stored, err := repo.PutFlightSummary(ctx, &staleObserved, summary)
	if err != nil {
		t.Fatalf("PutFlightSummary() error = %v", err)
	}
	if stored {
		t.Fatal("PutFlightSummary() stored stale summary over an active lease")
	}
	got, _, err := repo.GetFlight(ctx, flight.FlightID)
	if err != nil {
		t.Fatalf("GetFlight() error = %v", err)
	}
	if got.FetchOwner == nil || *got.FetchOwner != "worker-1" || got.FetchLeaseUntil == nil || *got.FetchLeaseUntil != "2026-04-29T00:10:00Z" {
		t.Fatalf("lease fields = owner %v until %v, want active lease retained", got.FetchOwner, got.FetchLeaseUntil)
	}
	if got.Status != flight.Status {
		t.Fatalf("Status = %q, want stale summary skipped", got.Status)
	}
}

func TestDynamoDBRepositoryAppendsPositionOnlyWhileFetchLeaseIsHeld(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	repo.now = func() time.Time { return time.Date(2026, 4, 29, 0, 1, 0, 0, time.UTC) }
	flight := testFlight("iflg_guarded_position", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	if _, leased, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:05:00Z", "2026-04-29T00:00:00Z"); err != nil || !leased {
		t.Fatalf("AcquireFetchLease() leased=%v error=%v, want lease", leased, err)
	}

	position := domain.FlightPosition{
		FlightID:  flight.FlightID,
		Latitude:  45.123,
		Longitude: 160.456,
		Timestamp: "2026-04-29T00:01:00Z",
		Source:    domain.PositionSourceFlightAwarePosition,
	}
	appended, err := repo.AppendPositionIfLeaseHeld(ctx, position, "worker-2", "2026-04-29T00:10:00Z")
	if err != nil {
		t.Fatalf("AppendPositionIfLeaseHeld(wrong owner) error = %v", err)
	}
	if appended {
		t.Fatal("AppendPositionIfLeaseHeld(wrong owner) appended = true, want false")
	}
	history, _, err := repo.ListPositions(ctx, flight.FlightID, nil, 10)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("history count = %d, want none for lost lease", len(history))
	}

	appended, err = repo.AppendPositionIfLeaseHeld(ctx, position, "worker-1", "2026-04-29T00:05:00Z")
	if err != nil {
		t.Fatalf("AppendPositionIfLeaseHeld(owner) error = %v", err)
	}
	if !appended {
		t.Fatal("AppendPositionIfLeaseHeld(owner) appended = false, want true")
	}
	history, _, err = repo.ListPositions(ctx, flight.FlightID, nil, 10)
	if err != nil {
		t.Fatalf("ListPositions(after append) error = %v", err)
	}
	if len(history) != 1 || history[0].Timestamp != position.Timestamp {
		t.Fatalf("history = %#v, want guarded append", history)
	}
}

func TestDynamoDBRepositoryRejectsGuardedPositionAppendAfterLeaseExpires(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	repo.now = func() time.Time { return time.Date(2026, 4, 29, 0, 6, 0, 0, time.UTC) }
	flight := testFlight("iflg_guarded_position_expired", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	if _, leased, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:05:00Z", "2026-04-29T00:00:00Z"); err != nil || !leased {
		t.Fatalf("AcquireFetchLease() leased=%v error=%v, want lease", leased, err)
	}

	held, err := repo.FetchLeaseHeld(ctx, flight.FlightID, "worker-1", "2026-04-29T00:05:00Z")
	if err != nil {
		t.Fatalf("FetchLeaseHeld(expired) error = %v", err)
	}
	if held {
		t.Fatal("FetchLeaseHeld(expired) = true, want false")
	}

	position := domain.FlightPosition{
		FlightID:  flight.FlightID,
		Latitude:  45.123,
		Longitude: 160.456,
		Timestamp: "2026-04-29T00:06:00Z",
		Source:    domain.PositionSourceFlightAwarePosition,
	}
	appended, err := repo.AppendPositionIfLeaseHeld(ctx, position, "worker-1", "2026-04-29T00:05:00Z")
	if err != nil {
		t.Fatalf("AppendPositionIfLeaseHeld(expired) error = %v", err)
	}
	if appended {
		t.Fatal("AppendPositionIfLeaseHeld(expired) appended = true, want false")
	}
	history, _, err := repo.ListPositions(ctx, flight.FlightID, nil, 10)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("history count after expired append = %d, want 0", len(history))
	}
}

func TestDynamoDBRepositoryUpdatesFetchedFlightAndPositionTogether(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	flight := testFlight("iflg_position_transaction", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	leasedFlight, leased, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:05:00Z", "2026-04-29T00:00:00Z")
	if err != nil || !leased {
		t.Fatalf("AcquireFetchLease() leased=%v error=%v, want lease", leased, err)
	}
	timestamp := domain.ISODateTimeString("2026-04-29T00:01:00Z")
	leasedFlight.LatestPositionTimestamp = &timestamp
	position := domain.FlightPosition{
		FlightID:  flight.FlightID,
		Latitude:  45.123,
		Longitude: 160.456,
		Timestamp: timestamp,
		Source:    domain.PositionSourceFlightAwarePosition,
	}

	updated, err := repo.UpdateFetchedFlightWithPosition(ctx, leasedFlight, position, "worker-2", "2026-04-29T00:10:00Z", "2026-04-29T00:01:00Z")
	if err != nil {
		t.Fatalf("UpdateFetchedFlightWithPosition(wrong owner) error = %v", err)
	}
	if updated {
		t.Fatal("UpdateFetchedFlightWithPosition(wrong owner) updated = true, want false")
	}
	stored, _, err := repo.GetFlight(ctx, flight.FlightID)
	if err != nil {
		t.Fatalf("GetFlight() error = %v", err)
	}
	if stored.LatestPositionTimestamp != nil {
		t.Fatalf("LatestPositionTimestamp = %v, want unchanged after failed transaction", stored.LatestPositionTimestamp)
	}
	history, _, err := repo.ListPositions(ctx, flight.FlightID, nil, 10)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("history count = %d, want no position after failed transaction", len(history))
	}

	updated, err = repo.UpdateFetchedFlightWithPosition(ctx, leasedFlight, position, "worker-1", "2026-04-29T00:05:00Z", "2026-04-29T00:01:00Z")
	if err != nil {
		t.Fatalf("UpdateFetchedFlightWithPosition(owner) error = %v", err)
	}
	if !updated {
		t.Fatal("UpdateFetchedFlightWithPosition(owner) updated = false, want true")
	}
	stored, _, err = repo.GetFlight(ctx, flight.FlightID)
	if err != nil {
		t.Fatalf("GetFlight(after update) error = %v", err)
	}
	if stored.LatestPositionTimestamp == nil || *stored.LatestPositionTimestamp != timestamp {
		t.Fatalf("LatestPositionTimestamp = %v, want %s", stored.LatestPositionTimestamp, timestamp)
	}
	history, _, err = repo.ListPositions(ctx, flight.FlightID, nil, 10)
	if err != nil {
		t.Fatalf("ListPositions(after update) error = %v", err)
	}
	if len(history) != 1 || history[0].Timestamp != timestamp {
		t.Fatalf("history = %#v, want transactional position write", history)
	}
}

func TestDynamoDBRepositoryProjectsDomainTTLAttributes(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryDynamoDBClient()
	repo := NewDynamoDBRepository(client, DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	flight := testFlight("iflg_ttl", "ANA110")
	flightTTL := domain.EpochSeconds(1800000000)
	flight.TTL = &flightTTL
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	positionTTL := domain.EpochSeconds(1800000100)
	position := domain.FlightPosition{
		FlightID:  flight.FlightID,
		Latitude:  45.123,
		Longitude: 160.456,
		Timestamp: "2026-04-29T00:01:00Z",
		Source:    domain.PositionSourceFlightAwarePosition,
		TTL:       &positionTTL,
	}
	if err := repo.PutPosition(ctx, position); err != nil {
		t.Fatalf("PutPosition() error = %v", err)
	}

	flightItem, ok, err := client.GetItem(ctx, "Flights", "flightId", string(flight.FlightID))
	if err != nil || !ok {
		t.Fatalf("GetItem(flight) = %v, %v", ok, err)
	}
	if ttl := numberValue(flightItem["ttl"]); ttl != float64(flightTTL) {
		t.Fatalf("flight ttl = %v, want %d", ttl, flightTTL)
	}
	positionItem, ok, err := client.GetItem(ctx, "FlightPositions", "flightId", string(flight.FlightID))
	if err != nil || !ok {
		t.Fatalf("GetItem(position) = %v, %v", ok, err)
	}
	if ttl := numberValue(positionItem["ttl"]); ttl != float64(positionTTL) {
		t.Fatalf("position ttl = %v, want %d", ttl, positionTTL)
	}
}

func TestDynamoDBRepositoryStoresMonthlyBudgetStateByEnvironmentAndMonth(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryDynamoDBClient()
	repo := NewScopedDynamoDBRepository(client, DynamoDBTables{UsageBudget: "UsageBudget"}, application.UsageBudgetScope{
		Environment: "dev",
		Month:       "2026-04",
	})

	devApril := application.UsageStatus{
		Budget: application.UsageBudgetStatus{
			Currency:                 "USD",
			EstimatedMonthToDateCost: 1.25,
		},
		FetchingEnabled: true,
	}
	prodApril := application.UsageStatus{
		Budget: application.UsageBudgetStatus{
			Currency:                 "USD",
			EstimatedMonthToDateCost: 3.75,
		},
		FetchingEnabled: true,
	}

	if err := repo.PutMonthlyUsageStatus(ctx, application.UsageBudgetScope{Environment: "dev", Month: "2026-04"}, devApril); err != nil {
		t.Fatalf("PutMonthlyUsageStatus(dev) error = %v", err)
	}
	if err := repo.PutMonthlyUsageStatus(ctx, application.UsageBudgetScope{Environment: "prod", Month: "2026-04"}, prodApril); err != nil {
		t.Fatalf("PutMonthlyUsageStatus(prod) error = %v", err)
	}

	got, err := repo.GetUsageStatus(ctx)
	if err != nil {
		t.Fatalf("GetUsageStatus() error = %v", err)
	}
	if got.Budget.Environment != "dev" || got.Budget.Month != "2026-04" {
		t.Fatalf("scope = %s/%s, want dev/2026-04", got.Budget.Environment, got.Budget.Month)
	}
	if got.Budget.EstimatedMonthToDateCost != 1.25 {
		t.Fatalf("estimated cost = %v, want 1.25", got.Budget.EstimatedMonthToDateCost)
	}

	prodGot, err := repo.GetMonthlyUsageStatus(ctx, application.UsageBudgetScope{Environment: "prod", Month: "2026-04"})
	if err != nil {
		t.Fatalf("GetMonthlyUsageStatus(prod) error = %v", err)
	}
	if prodGot.Budget.EstimatedMonthToDateCost != 3.75 {
		t.Fatalf("prod estimated cost = %v, want 3.75", prodGot.Budget.EstimatedMonthToDateCost)
	}
}

func TestDynamoDBRepositorySoftStopThresholdDisablesNewFetches(t *testing.T) {
	ctx := context.Background()
	repo := NewScopedDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{UsageBudget: "UsageBudget"}, application.UsageBudgetScope{
		Environment: "dev",
		Month:       "2026-04",
	})

	if err := repo.PutMonthlyUsageStatus(ctx, application.UsageBudgetScope{Environment: "dev", Month: "2026-04"}, application.UsageStatus{
		Budget: application.UsageBudgetStatus{
			Currency:                 "USD",
			EstimatedMonthToDateCost: application.DefaultSoftStopThresholdUSD,
		},
		FetchingEnabled: true,
	}); err != nil {
		t.Fatalf("PutMonthlyUsageStatus() error = %v", err)
	}

	allowed, err := repo.FetchingAllowed(ctx)
	if err != nil {
		t.Fatalf("FetchingAllowed() error = %v", err)
	}
	if allowed {
		t.Fatal("FetchingAllowed() = true, want false at the default soft stop threshold")
	}

	status, err := repo.GetUsageStatus(ctx)
	if err != nil {
		t.Fatalf("GetUsageStatus() error = %v", err)
	}
	if status.Budget.SoftStopThreshold != application.DefaultSoftStopThresholdUSD || !status.Budget.Stopped || status.FetchingEnabled {
		t.Fatalf("usage status = %#v, want default threshold stop", status)
	}
}

func TestDynamoDBRepositoryRecordsUsageEstimatesIntoMonthlyBudgetState(t *testing.T) {
	ctx := context.Background()
	scope := application.UsageBudgetScope{Environment: "dev", Month: "2026-04"}
	repo := NewScopedDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{UsageBudget: "UsageBudget"}, scope)

	if err := repo.RecordFlightAwareCall(ctx, flightaware.UsageCallRecord{
		Endpoint:            flightaware.EndpointSearch,
		Phase:               flightaware.UsageRecordPhaseBefore,
		EstimatedResultSets: 1,
		EstimatedCostUSD:    0.75,
	}); err != nil {
		t.Fatalf("RecordFlightAwareCall(before) error = %v", err)
	}
	if err := repo.RecordFlightAwareCall(ctx, flightaware.UsageCallRecord{
		Endpoint:         flightaware.EndpointSearch,
		Phase:            flightaware.UsageRecordPhaseAfter,
		EstimatedCostUSD: 0.75,
		Succeeded:        true,
	}); err != nil {
		t.Fatalf("RecordFlightAwareCall(after) error = %v", err)
	}

	status, err := repo.GetMonthlyUsageStatus(ctx, scope)
	if err != nil {
		t.Fatalf("GetMonthlyUsageStatus() error = %v", err)
	}
	if status.Budget.EstimatedMonthToDateCost != 0.75 {
		t.Fatalf("estimated cost = %v, want 0.75", status.Budget.EstimatedMonthToDateCost)
	}
	if !status.FetchingEnabled || status.Budget.SoftStopThreshold != application.DefaultSoftStopThresholdUSD {
		t.Fatalf("usage status = %#v", status)
	}
}

func TestDynamoDBRepositoryStopsUsageAccountingWhenEstimateCrossesBudget(t *testing.T) {
	ctx := context.Background()
	scope := application.UsageBudgetScope{Environment: "dev", Month: "2026-04"}
	repo := NewScopedDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{UsageBudget: "UsageBudget"}, scope)
	if err := repo.PutMonthlyUsageStatus(ctx, scope, application.UsageStatus{
		Budget: application.UsageBudgetStatus{
			Environment:              "dev",
			Month:                    "2026-04",
			Currency:                 "USD",
			EstimatedMonthToDateCost: 3.95,
			SoftStopThreshold:        4,
		},
		FetchingEnabled: true,
	}); err != nil {
		t.Fatalf("PutMonthlyUsageStatus() error = %v", err)
	}

	err := repo.RecordFlightAwareCall(ctx, flightaware.UsageCallRecord{
		Endpoint:            flightaware.EndpointPosition,
		Phase:               flightaware.UsageRecordPhaseBefore,
		EstimatedResultSets: 1,
		EstimatedCostUSD:    0.10,
	})
	if !errors.Is(err, application.ErrBudgetExceeded) {
		t.Fatalf("RecordFlightAwareCall() error = %v, want ErrBudgetExceeded", err)
	}
	status, err := repo.GetMonthlyUsageStatus(ctx, scope)
	if err != nil {
		t.Fatalf("GetMonthlyUsageStatus() error = %v", err)
	}
	if status.Budget.EstimatedMonthToDateCost != 4.05 || !status.Budget.Stopped || status.FetchingEnabled {
		t.Fatalf("usage status = %#v, want stopped after reserved estimate", status)
	}
}

func TestDynamoDBRepositoryBlocksUsageAccountingFromAtomicUpdatedBudgetState(t *testing.T) {
	ctx := context.Background()
	scope := application.UsageBudgetScope{Environment: "dev", Month: "2026-04"}
	client := &staleUsageReadDynamoDBClient{MemoryDynamoDBClient: NewMemoryDynamoDBClient()}
	repo := NewScopedDynamoDBRepository(client, DynamoDBTables{UsageBudget: "UsageBudget"}, scope)
	if err := repo.PutMonthlyUsageStatus(ctx, scope, application.UsageStatus{
		Budget: application.UsageBudgetStatus{
			Environment:              "dev",
			Month:                    "2026-04",
			Currency:                 "USD",
			EstimatedMonthToDateCost: 3.95,
			SoftStopThreshold:        4,
		},
		FetchingEnabled: true,
	}); err != nil {
		t.Fatalf("PutMonthlyUsageStatus() error = %v", err)
	}

	err := repo.RecordFlightAwareCall(ctx, flightaware.UsageCallRecord{
		Endpoint:            flightaware.EndpointPosition,
		Phase:               flightaware.UsageRecordPhaseBefore,
		EstimatedResultSets: 1,
		EstimatedCostUSD:    0.10,
	})
	if !errors.Is(err, application.ErrBudgetExceeded) {
		t.Fatalf("RecordFlightAwareCall() error = %v, want ErrBudgetExceeded from updated write result", err)
	}
}

func TestDynamoDBRepositoryRecordsConcurrentUsageEstimatesAtomically(t *testing.T) {
	ctx := context.Background()
	scope := application.UsageBudgetScope{Environment: "dev", Month: "2026-04"}
	repo := NewScopedDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{UsageBudget: "UsageBudget"}, scope)
	if err := repo.PutMonthlyUsageStatus(ctx, scope, application.UsageStatus{
		Budget: application.UsageBudgetStatus{
			Environment:       "dev",
			Month:             "2026-04",
			Currency:          "USD",
			SoftStopThreshold: 10,
		},
		FetchingEnabled: true,
	}); err != nil {
		t.Fatalf("PutMonthlyUsageStatus() error = %v", err)
	}

	const workers = 50
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- repo.RecordFlightAwareCall(ctx, flightaware.UsageCallRecord{
				Endpoint:            flightaware.EndpointPosition,
				Phase:               flightaware.UsageRecordPhaseBefore,
				EstimatedResultSets: 1,
				EstimatedCostUSD:    0.10,
			})
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("RecordFlightAwareCall(concurrent) error = %v", err)
		}
	}

	status, err := repo.GetMonthlyUsageStatus(ctx, scope)
	if err != nil {
		t.Fatalf("GetMonthlyUsageStatus() error = %v", err)
	}
	if math.Abs(status.Budget.EstimatedMonthToDateCost-5.0) > 0.000001 {
		t.Fatalf("estimated cost = %v, want 5.0", status.Budget.EstimatedMonthToDateCost)
	}
}

func TestDynamoDBRepositoryReconcilesAccountUsageWithoutLoweringLocalEstimate(t *testing.T) {
	ctx := context.Background()
	scope := application.UsageBudgetScope{Environment: "dev", Month: "2026-04"}
	repo := NewScopedDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{UsageBudget: "UsageBudget"}, scope)

	if err := repo.PutMonthlyUsageStatus(ctx, scope, application.UsageStatus{
		Budget: application.UsageBudgetStatus{
			Currency:                 "USD",
			EstimatedMonthToDateCost: 3.50,
			SoftStopThreshold:        application.DefaultSoftStopThresholdUSD,
		},
		FetchingEnabled: true,
	}); err != nil {
		t.Fatalf("PutMonthlyUsageStatus() error = %v", err)
	}

	if err := repo.ReconcileAccountUsage(ctx, scope, flightaware.UsageResponse{
		Currency:    "USD",
		MonthToDate: flightaware.UsageAmount{EstimatedCostUSD: 2.25, ResultSets: 10},
	}); err != nil {
		t.Fatalf("ReconcileAccountUsage(lower remote) error = %v", err)
	}
	status, err := repo.GetMonthlyUsageStatus(ctx, scope)
	if err != nil {
		t.Fatalf("GetMonthlyUsageStatus() error = %v", err)
	}
	if status.Budget.EstimatedMonthToDateCost != 3.50 {
		t.Fatalf("estimated cost after lower remote = %v, want local 3.50", status.Budget.EstimatedMonthToDateCost)
	}

	if err := repo.ReconcileAccountUsage(ctx, scope, flightaware.UsageResponse{
		Currency:    "USD",
		MonthToDate: flightaware.UsageAmount{EstimatedCostUSD: 4.25, ResultSets: 20},
	}); err != nil {
		t.Fatalf("ReconcileAccountUsage(higher remote) error = %v", err)
	}
	status, err = repo.GetMonthlyUsageStatus(ctx, scope)
	if err != nil {
		t.Fatalf("GetMonthlyUsageStatus() after higher remote error = %v", err)
	}
	if status.Budget.EstimatedMonthToDateCost != 4.25 || !status.Budget.Stopped || status.FetchingEnabled {
		t.Fatalf("usage status after higher remote = %#v, want stopped at 4.25", status)
	}
}

func TestDynamoDBRepositoryReconcileDoesNotOverwriteNewerLocalEstimate(t *testing.T) {
	ctx := context.Background()
	scope := application.UsageBudgetScope{Environment: "dev", Month: "2026-04"}
	client := &concurrentUsageReconcileDynamoDBClient{MemoryDynamoDBClient: NewMemoryDynamoDBClient()}
	repo := NewScopedDynamoDBRepository(client, DynamoDBTables{UsageBudget: "UsageBudget"}, scope)

	if err := repo.PutMonthlyUsageStatus(ctx, scope, application.UsageStatus{
		Budget: application.UsageBudgetStatus{
			Currency:                 "USD",
			EstimatedMonthToDateCost: 3.50,
			SoftStopThreshold:        10,
		},
		FetchingEnabled: true,
	}); err != nil {
		t.Fatalf("PutMonthlyUsageStatus() error = %v", err)
	}
	client.raiseCostBeforeNextReconcileWrite = true

	if err := repo.ReconcileAccountUsage(ctx, scope, flightaware.UsageResponse{
		Currency:    "USD",
		MonthToDate: flightaware.UsageAmount{EstimatedCostUSD: 4.25, ResultSets: 20},
	}); err != nil {
		t.Fatalf("ReconcileAccountUsage() error = %v", err)
	}
	status, err := repo.GetMonthlyUsageStatus(ctx, scope)
	if err != nil {
		t.Fatalf("GetMonthlyUsageStatus() error = %v", err)
	}
	if status.Budget.EstimatedMonthToDateCost != 5.00 {
		t.Fatalf("estimated cost after concurrent reconcile = %v, want newer local 5.00", status.Budget.EstimatedMonthToDateCost)
	}
}

func TestDynamoDBRepositoryListsDuePollFlightsAndUsesFlightLease(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryDynamoDBClient()
	repo := NewDynamoDBRepository(client, DynamoDBTables{
		Flights: "Flights",
	})
	due := testFlight("iflg_due", "ANA110")
	due.NextPositionPollAt = ptrISO("2026-04-29T00:00:00Z")
	future := testFlight("iflg_future", "ANA111")
	future.NextPositionPollAt = ptrISO("2026-04-29T00:30:00Z")
	if err := repo.PutFlight(ctx, due); err != nil {
		t.Fatalf("PutFlight(due) error = %v", err)
	}
	if err := repo.PutFlight(ctx, future); err != nil {
		t.Fatalf("PutFlight(future) error = %v", err)
	}

	flights, err := repo.ListPollableFlights(ctx, "2026-04-29T00:05:00Z", 10)
	if err != nil {
		t.Fatalf("ListPollableFlights() error = %v", err)
	}
	if len(flights) != 1 || flights[0].FlightID != due.FlightID {
		t.Fatalf("due flights = %#v, want only due flight", flights)
	}

	leased, ok, err := repo.AcquireFetchLease(ctx, due.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if err != nil {
		t.Fatalf("AcquireFetchLease(first) error = %v", err)
	}
	if !ok || leased.FetchOwner == nil || *leased.FetchOwner != "worker-1" {
		t.Fatalf("first lease = %#v ok=%v", leased, ok)
	}
	_, ok, err = repo.AcquireFetchLease(ctx, due.FlightID, "worker-2", "2026-04-29T00:10:00Z", "2026-04-29T00:06:00Z")
	if err != nil {
		t.Fatalf("AcquireFetchLease(second) error = %v", err)
	}
	if ok {
		t.Fatal("second lease acquired while first lease is active")
	}
	_, ok, err = repo.AcquireFetchLease(ctx, due.FlightID, "worker-1", "2026-04-29T00:15:00Z", "2026-04-29T00:06:00Z")
	if err != nil {
		t.Fatalf("AcquireFetchLease(same owner) error = %v", err)
	}
	if ok {
		t.Fatal("same owner acquired a second lease while the first lease is active")
	}
	if client.WasConditionalPutUsed("Flights", string(due.FlightID)) == false {
		t.Fatal("AcquireFetchLease did not use a conditional DynamoDB write")
	}
	if err := repo.ReleaseFetchLease(ctx, due.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:07:00Z"); err != nil {
		t.Fatalf("ReleaseFetchLease() error = %v", err)
	}
	released, _, err := repo.GetFlight(ctx, due.FlightID)
	if err != nil {
		t.Fatalf("GetFlight(released) error = %v", err)
	}
	if released.FetchOwner != nil || released.FetchLeaseUntil != nil {
		t.Fatalf("released lease fields = owner %v until %v, want nil", released.FetchOwner, released.FetchLeaseUntil)
	}
}

func TestDynamoDBRepositorySkipsTTLExpiredPollFlights(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{
		Flights: "Flights",
	})
	expired := testFlight("iflg_expired_poll", "ANA110")
	expired.NextPositionPollAt = ptrISO("2026-04-29T00:00:00Z")
	expiredTTL := domain.EpochSeconds(time.Date(2026, 4, 29, 0, 4, 59, 0, time.UTC).Unix())
	expired.TTL = &expiredTTL
	due := testFlight("iflg_due_poll", "ANA111")
	due.NextPositionPollAt = ptrISO("2026-04-29T00:00:00Z")
	dueTTL := domain.EpochSeconds(time.Date(2026, 4, 29, 0, 6, 0, 0, time.UTC).Unix())
	due.TTL = &dueTTL
	if err := repo.PutFlight(ctx, expired); err != nil {
		t.Fatalf("PutFlight(expired) error = %v", err)
	}
	if err := repo.PutFlight(ctx, due); err != nil {
		t.Fatalf("PutFlight(due) error = %v", err)
	}

	flights, err := repo.ListPollableFlights(ctx, "2026-04-29T00:05:00Z", 10)
	if err != nil {
		t.Fatalf("ListPollableFlights() error = %v", err)
	}
	if len(flights) != 1 || flights[0].FlightID != due.FlightID {
		t.Fatalf("pollable flights = %#v, want only unexpired due flight", flights)
	}
}

func TestDynamoDBRepositoryPollLimitCountsUnexpiredFlights(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{
		Flights: "Flights",
	})
	expiredTTL := domain.EpochSeconds(time.Date(2026, 4, 29, 0, 4, 59, 0, time.UTC).Unix())
	validTTL := domain.EpochSeconds(time.Date(2026, 4, 29, 0, 6, 0, 0, time.UTC).Unix())
	expired := testFlight("iflg_expired_poll_limit", "ANA110")
	expired.NextPositionPollAt = ptrISO("2026-04-29T00:00:00Z")
	expired.TTL = &expiredTTL
	valid := testFlight("iflg_valid_poll_limit", "ANA111")
	valid.NextPositionPollAt = ptrISO("2026-04-29T00:00:01Z")
	valid.TTL = &validTTL
	if err := repo.PutFlight(ctx, expired); err != nil {
		t.Fatalf("PutFlight(expired) error = %v", err)
	}
	if err := repo.PutFlight(ctx, valid); err != nil {
		t.Fatalf("PutFlight(valid) error = %v", err)
	}

	flights, err := repo.ListPollableFlights(ctx, "2026-04-29T00:05:00Z", 1)
	if err != nil {
		t.Fatalf("ListPollableFlights() error = %v", err)
	}
	if len(flights) != 1 || flights[0].FlightID != valid.FlightID {
		t.Fatalf("pollable flights with limit = %#v, want first unexpired due flight", flights)
	}
}

func TestDynamoDBRepositoryPollLimitUsesBoundedPagesUntilEnoughValidFlights(t *testing.T) {
	ctx := context.Background()
	client := &countingPollPageDynamoDBClient{MemoryDynamoDBClient: NewMemoryDynamoDBClient()}
	repo := NewDynamoDBRepository(client, DynamoDBTables{
		Flights: "Flights",
	})
	expiredTTL := domain.EpochSeconds(time.Date(2026, 4, 29, 0, 4, 59, 0, time.UTC).Unix())
	validTTL := domain.EpochSeconds(time.Date(2026, 4, 29, 0, 6, 0, 0, time.UTC).Unix())
	for index := 0; index < pollQueryPageLimit+3; index++ {
		expired := testFlight(domain.FlightID(fmt.Sprintf("iflg_expired_poll_page_%02d", index)), "ANA110")
		expired.NextPositionPollAt = ptrISO(fmt.Sprintf("2026-04-29T00:00:%02dZ", index))
		expired.TTL = &expiredTTL
		if err := repo.PutFlight(ctx, expired); err != nil {
			t.Fatalf("PutFlight(expired %d) error = %v", index, err)
		}
	}
	valid := testFlight("iflg_valid_poll_page", "ANA111")
	valid.NextPositionPollAt = ptrISO("2026-04-29T00:00:40Z")
	valid.TTL = &validTTL
	if err := repo.PutFlight(ctx, valid); err != nil {
		t.Fatalf("PutFlight(valid) error = %v", err)
	}

	flights, err := repo.ListPollableFlights(ctx, "2026-04-29T00:05:00Z", 1)
	if err != nil {
		t.Fatalf("ListPollableFlights() error = %v", err)
	}
	if len(flights) != 1 || flights[0].FlightID != valid.FlightID {
		t.Fatalf("pollable flights with paged limit = %#v, want first unexpired due flight", flights)
	}
	if client.sawUnboundedQuery {
		t.Fatal("ListPollableFlights used an unbounded due-poll query")
	}
	if client.pageCalls < 2 {
		t.Fatalf("page calls = %d, want multiple bounded pages after expired rows", client.pageCalls)
	}
	if client.maxPageLimit > pollQueryPageLimit {
		t.Fatalf("max page limit = %d, want at most %d", client.maxPageLimit, pollQueryPageLimit)
	}
}

func TestDynamoDBRepositoryRejectsTTLExpiredFlightLeaseAcquisition(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{
		Flights: "Flights",
	})
	flight := testFlight("iflg_expired_lease_acquire", "ANA110")
	expiredTTL := domain.EpochSeconds(time.Date(2026, 4, 29, 0, 4, 59, 0, time.UTC).Unix())
	flight.TTL = &expiredTTL
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}

	_, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("AcquireFetchLease(expired) error = %v, want ErrNotFound", err)
	}
	if ok {
		t.Fatal("AcquireFetchLease(expired) acquired lease")
	}
}

func TestDynamoDBRepositoryDoesNotClearActiveLeaseWhenUpdatingPollSchedule(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{Flights: "Flights"})
	flight := testFlight("iflg_poll_schedule_lease", "ANA110")
	flight.NextPositionPollAt = ptrISO("2026-04-29T00:00:00Z")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	staleDispatcherSnapshot := flight

	leased, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease() = %v, %v", ok, err)
	}
	timestamp := domain.ISODateTimeString("2026-04-29T00:06:00Z")
	leased.LatestPositionTimestamp = &timestamp
	if updated, err := repo.UpdateFetchedFlight(ctx, leased, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:06:00Z"); err != nil || !updated {
		t.Fatalf("UpdateFetchedFlight() = %v, %v", updated, err)
	}

	staleDispatcherSnapshot.NextPositionPollAt = ptrISO("2026-04-29T00:20:00Z")
	if err := repo.UpdatePollSchedule(ctx, staleDispatcherSnapshot); err != nil {
		t.Fatalf("UpdatePollSchedule(stale snapshot) error = %v", err)
	}

	got, _, err := repo.GetFlight(ctx, flight.FlightID)
	if err != nil {
		t.Fatalf("GetFlight() error = %v", err)
	}
	if got.FetchOwner == nil || *got.FetchOwner != "worker-1" || got.FetchLeaseUntil == nil || *got.FetchLeaseUntil != "2026-04-29T00:10:00Z" {
		t.Fatalf("lease fields = owner %v until %v, want active worker lease retained", got.FetchOwner, got.FetchLeaseUntil)
	}
	if got.LatestPositionTimestamp == nil || *got.LatestPositionTimestamp != timestamp {
		t.Fatalf("LatestPositionTimestamp = %v, want fetched metadata retained", got.LatestPositionTimestamp)
	}
	if got.NextPositionPollAt == nil || *got.NextPositionPollAt != "2026-04-29T00:00:00Z" {
		t.Fatalf("NextPositionPollAt = %v, want stale schedule update ignored while lease changed", got.NextPositionPollAt)
	}
}

func TestDynamoDBRepositoryReleaseFetchLeaseDoesNotOverwriteFetchedMetadataFromStaleRead(t *testing.T) {
	ctx := context.Background()
	client := &staleFlightReadDynamoDBClient{MemoryDynamoDBClient: NewMemoryDynamoDBClient()}
	repo := NewDynamoDBRepository(client, DynamoDBTables{Flights: "Flights"})
	flight := testFlight("iflg_release_stale_read", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	leased, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease() = %v, %v", ok, err)
	}
	staleLeasedItem := flightItem(leased)
	routeKey := "routes/iflg_release_stale_read/hash.json"
	leased.PlannedRouteS3Key = &routeKey
	if updated, err := repo.UpdateFetchedFlight(ctx, leased, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:06:00Z"); err != nil || !updated {
		t.Fatalf("UpdateFetchedFlight() = %v, %v", updated, err)
	}
	client.staleNextFlightGet = staleLeasedItem

	if err := repo.ReleaseFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:07:00Z"); err != nil {
		t.Fatalf("ReleaseFetchLease() error = %v", err)
	}
	client.staleNextFlightGet = nil
	got, _, err := repo.GetFlight(ctx, flight.FlightID)
	if err != nil {
		t.Fatalf("GetFlight() error = %v", err)
	}
	if got.PlannedRouteS3Key == nil || *got.PlannedRouteS3Key != routeKey {
		t.Fatalf("PlannedRouteS3Key = %v, want fetched route metadata retained", got.PlannedRouteS3Key)
	}
	if got.FetchOwner != nil || got.FetchLeaseUntil != nil {
		t.Fatalf("lease fields = owner %v until %v, want released lease", got.FetchOwner, got.FetchLeaseUntil)
	}
}

func TestDynamoDBRepositoryPollScheduleDoesNotOverwriteFetchedMetadataAfterStaleRead(t *testing.T) {
	ctx := context.Background()
	client := &staleFlightReadDynamoDBClient{MemoryDynamoDBClient: NewMemoryDynamoDBClient()}
	repo := NewDynamoDBRepository(client, DynamoDBTables{Flights: "Flights"})
	flight := testFlight("iflg_poll_stale_read", "ANA110")
	flight.NextPositionPollAt = ptrISO("2026-04-29T00:00:00Z")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	staleItem := flightItem(flight)
	leased, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease() = %v, %v", ok, err)
	}
	routeKey := "routes/iflg_poll_stale_read/hash.json"
	leased.PlannedRouteS3Key = &routeKey
	if updated, err := repo.UpdateFetchedFlight(ctx, leased, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:06:00Z"); err != nil || !updated {
		t.Fatalf("UpdateFetchedFlight() = %v, %v", updated, err)
	}
	if err := repo.ReleaseFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:07:00Z"); err != nil {
		t.Fatalf("ReleaseFetchLease() error = %v", err)
	}
	client.staleNextFlightGet = staleItem

	scheduled := flight
	scheduled.NextPositionPollAt = ptrISO("2026-04-29T00:20:00Z")
	if err := repo.UpdatePollSchedule(ctx, scheduled); err != nil {
		t.Fatalf("UpdatePollSchedule() error = %v", err)
	}
	client.staleNextFlightGet = nil
	got, _, err := repo.GetFlight(ctx, flight.FlightID)
	if err != nil {
		t.Fatalf("GetFlight() error = %v", err)
	}
	if got.PlannedRouteS3Key == nil || *got.PlannedRouteS3Key != routeKey {
		t.Fatalf("PlannedRouteS3Key = %v, want fetched route metadata retained", got.PlannedRouteS3Key)
	}
	if got.NextPositionPollAt == nil || *got.NextPositionPollAt != "2026-04-29T00:00:00Z" {
		t.Fatalf("NextPositionPollAt = %v, want stale schedule update skipped", got.NextPositionPollAt)
	}
}

func TestDynamoDBRepositoryAcquireFetchLeaseDoesNotOverwriteFetchedMetadataAfterStaleRead(t *testing.T) {
	ctx := context.Background()
	client := &staleFlightReadDynamoDBClient{MemoryDynamoDBClient: NewMemoryDynamoDBClient()}
	repo := NewDynamoDBRepository(client, DynamoDBTables{Flights: "Flights"})
	flight := testFlight("iflg_acquire_stale_read", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	staleItem := flightItem(flight)
	leased, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease(worker-1) = %v, %v", ok, err)
	}
	routeKey := "routes/iflg_acquire_stale_read/hash.json"
	leased.PlannedRouteS3Key = &routeKey
	if updated, err := repo.UpdateFetchedFlight(ctx, leased, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:06:00Z"); err != nil || !updated {
		t.Fatalf("UpdateFetchedFlight() = %v, %v", updated, err)
	}
	if err := repo.ReleaseFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:07:00Z"); err != nil {
		t.Fatalf("ReleaseFetchLease() error = %v", err)
	}
	client.staleNextFlightGet = staleItem

	leased, ok, err = repo.AcquireFetchLease(ctx, flight.FlightID, "worker-2", "2026-04-29T00:15:00Z", "2026-04-29T00:08:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease(worker-2) = %v, %v", ok, err)
	}
	if leased.PlannedRouteS3Key == nil || *leased.PlannedRouteS3Key != routeKey {
		t.Fatalf("leased PlannedRouteS3Key = %v, want fetched route metadata retained", leased.PlannedRouteS3Key)
	}
	client.staleNextFlightGet = nil
	got, _, err := repo.GetFlight(ctx, flight.FlightID)
	if err != nil {
		t.Fatalf("GetFlight() error = %v", err)
	}
	if got.PlannedRouteS3Key == nil || *got.PlannedRouteS3Key != routeKey {
		t.Fatalf("stored PlannedRouteS3Key = %v, want fetched route metadata retained", got.PlannedRouteS3Key)
	}
	if got.FetchOwner == nil || *got.FetchOwner != "worker-2" || got.FetchLeaseUntil == nil || *got.FetchLeaseUntil != "2026-04-29T00:15:00Z" {
		t.Fatalf("lease fields = owner %v until %v, want worker-2 lease", got.FetchOwner, got.FetchLeaseUntil)
	}
}

func TestDynamoDBRepositoryRejectsFetchedFlightUpdateAfterLeaseIsStolen(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{Flights: "Flights"})
	faFlightID := domain.FAFlightID("fa_stale_1")
	flight := domain.Flight{
		FlightID:     "iflg_stale_1",
		FlightIDType: domain.FlightIDTypeInternal,
		FAFlightID:   &faFlightID,
		Ident:        "ANA110",
		Origin:       domain.Airport{Code: "RJTT"},
		Destination:  domain.Airport{Code: "KJFK"},
		Status:       "En Route",
		UpdatedAt:    "2026-04-29T00:00:00Z",
	}
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	workerOneFlight, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:05:00Z", "2026-04-29T00:00:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease(worker-1) = %v, %v", ok, err)
	}
	_, ok, err = repo.AcquireFetchLease(ctx, flight.FlightID, "worker-2", "2026-04-29T00:11:00Z", "2026-04-29T00:06:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease(worker-2) = %v, %v", ok, err)
	}

	timestamp := domain.ISODateTimeString("2026-04-29T00:07:00Z")
	workerOneFlight.LatestPositionTimestamp = &timestamp
	updated, err := repo.UpdateFetchedFlight(ctx, workerOneFlight, "worker-1", "2026-04-29T00:05:00Z", "2026-04-29T00:07:00Z")
	if err != nil {
		t.Fatalf("UpdateFetchedFlight(stale worker) error = %v", err)
	}
	if updated {
		t.Fatal("UpdateFetchedFlight(stale worker) updated flight after lease was stolen")
	}
	got, _, err := repo.GetFlight(ctx, flight.FlightID)
	if err != nil {
		t.Fatalf("GetFlight() error = %v", err)
	}
	if got.LatestPositionTimestamp != nil {
		t.Fatalf("LatestPositionTimestamp = %v, want stale update ignored", *got.LatestPositionTimestamp)
	}
	if got.FetchOwner == nil || *got.FetchOwner != "worker-2" {
		t.Fatalf("FetchOwner = %v, want worker-2 retained", got.FetchOwner)
	}
}

func TestDynamoDBRepositoryRejectsFetchedFlightUpdateAfterLeaseExpires(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{Flights: "Flights"})
	flight := testFlight("iflg_expired_fetch_update", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	leased, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:05:00Z", "2026-04-29T00:00:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease() = %v, %v", ok, err)
	}

	timestamp := domain.ISODateTimeString("2026-04-29T00:07:00Z")
	leased.LatestPositionTimestamp = &timestamp
	updated, err := repo.UpdateFetchedFlight(ctx, leased, "worker-1", "2026-04-29T00:05:00Z", "2026-04-29T00:06:00Z")
	if err != nil {
		t.Fatalf("UpdateFetchedFlight(expired lease) error = %v", err)
	}
	if updated {
		t.Fatal("UpdateFetchedFlight(expired lease) updated flight after lease expired")
	}
	got, _, err := repo.GetFlight(ctx, flight.FlightID)
	if err != nil {
		t.Fatalf("GetFlight() error = %v", err)
	}
	if got.LatestPositionTimestamp != nil {
		t.Fatalf("LatestPositionTimestamp = %v, want expired lease update ignored", *got.LatestPositionTimestamp)
	}
}

func TestDynamoDBRepositoryRejectsFetchedPositionUpdateAfterLeaseExpires(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	flight := testFlight("iflg_expired_position_update", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	leased, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:05:00Z", "2026-04-29T00:00:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease() = %v, %v", ok, err)
	}
	position := domain.FlightPosition{
		FlightID:  flight.FlightID,
		Latitude:  35.55,
		Longitude: 139.78,
		Timestamp: "2026-04-29T00:06:00Z",
		Source:    domain.PositionSourceFlightAwarePosition,
	}
	leased.LatestPositionTimestamp = &position.Timestamp

	updated, err := repo.UpdateFetchedFlightWithPosition(ctx, leased, position, "worker-1", "2026-04-29T00:05:00Z", "2026-04-29T00:06:00Z")
	if err != nil {
		t.Fatalf("UpdateFetchedFlightWithPosition(expired lease) error = %v", err)
	}
	if updated {
		t.Fatal("UpdateFetchedFlightWithPosition(expired lease) updated after lease expired")
	}
	history, _, err := repo.ListPositions(ctx, flight.FlightID, nil, 10)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("positions written after expired lease = %d, want 0", len(history))
	}
}

func TestDynamoDBRepositoryRejectsLargeTrackCommitAfterLeaseExpires(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	flight := testFlight("iflg_expired_large_track", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	leased, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:05:00Z", "2026-04-29T00:00:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease() = %v, %v", ok, err)
	}
	positions := largeTrackPositions(flight.FlightID, 0, 120)

	updated, err := repo.UpdateFetchedFlightWithTrackPositions(ctx, leased, positions, "worker-1", "2026-04-29T00:05:00Z", "2026-04-29T00:06:00Z")
	if err != nil {
		t.Fatalf("UpdateFetchedFlightWithTrackPositions(expired lease) error = %v", err)
	}
	if updated {
		t.Fatal("UpdateFetchedFlightWithTrackPositions(expired lease) updated after lease expired")
	}
	history, _, err := repo.ListPositions(ctx, flight.FlightID, nil, 200)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("large track positions visible after expired lease = %d, want 0", len(history))
	}
}

func TestDynamoDBRepositoryStoresLargeTrackPositionsWithLeaseGuard(t *testing.T) {
	ctx := context.Background()
	client := &countingCollectionDynamoDBClient{MemoryDynamoDBClient: NewMemoryDynamoDBClient()}
	repo := NewDynamoDBRepository(client, DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	flight := testFlight("iflg_large_track", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	leased, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:00:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease() = %v, %v", ok, err)
	}
	positions := make([]domain.FlightPosition, 0, 120)
	for index := 0; index < 120; index++ {
		positions = append(positions, domain.FlightPosition{
			FlightID:  flight.FlightID,
			Latitude:  float64(index),
			Longitude: float64(index),
			Timestamp: fmt.Sprintf("2026-04-29T00:%02d:%02dZ", index/60, index%60),
			Source:    domain.PositionSourceFlightAwareTrack,
		})
	}

	updated, err := repo.UpdateFetchedFlightWithTrackPositions(ctx, leased, positions, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if err != nil {
		t.Fatalf("UpdateFetchedFlightWithTrackPositions() error = %v", err)
	}
	if !updated {
		t.Fatal("UpdateFetchedFlightWithTrackPositions() updated = false, want true")
	}
	if client.collectionCalls != 0 {
		t.Fatalf("collection calls = %d, want large track to avoid partial flight publishes", client.collectionCalls)
	}
	if client.positionWrites != len(positions) {
		t.Fatalf("guarded position writes = %d, want %d", client.positionWrites, len(positions))
	}
	history, _, err := repo.ListPositions(ctx, flight.FlightID, nil, 200)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != len(positions) {
		t.Fatalf("stored positions = %d, want %d", len(history), len(positions))
	}
}

func TestDynamoDBRepositoryUpdatesExistingLargeTrackPositionRows(t *testing.T) {
	ctx := context.Background()
	client := &countingCollectionDynamoDBClient{MemoryDynamoDBClient: NewMemoryDynamoDBClient()}
	repo := NewDynamoDBRepository(client, DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	flight := testFlight("iflg_large_track_update", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	existing := domain.FlightPosition{
		FlightID:  flight.FlightID,
		Latitude:  35.55,
		Longitude: 139.78,
		Timestamp: "2026-04-29T00:00:00Z",
		Source:    domain.PositionSourceFlightAwareTrack,
	}
	if err := repo.PutPosition(ctx, existing); err != nil {
		t.Fatalf("PutPosition(existing) error = %v", err)
	}
	leased, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:00:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease() = %v, %v", ok, err)
	}
	positions := make([]domain.FlightPosition, 0, 120)
	for index := 0; index < 120; index++ {
		positions = append(positions, domain.FlightPosition{
			FlightID:  flight.FlightID,
			Latitude:  float64(index) + 0.25,
			Longitude: float64(index) + 0.5,
			Timestamp: fmt.Sprintf("2026-04-29T00:%02d:%02dZ", index/60, index%60),
			Source:    domain.PositionSourceFlightAwareTrack,
		})
	}

	updated, err := repo.UpdateFetchedFlightWithTrackPositions(ctx, leased, positions, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if err != nil {
		t.Fatalf("UpdateFetchedFlightWithTrackPositions() error = %v", err)
	}
	if !updated {
		t.Fatal("UpdateFetchedFlightWithTrackPositions() updated = false, want true")
	}
	history, _, err := repo.ListPositions(ctx, flight.FlightID, nil, 200)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != len(positions) {
		t.Fatalf("stored positions = %d, want %d", len(history), len(positions))
	}
	if history[0].Latitude != 0.25 || history[0].Longitude != 0.5 {
		t.Fatalf("first history row = %#v, want existing timestamp overwritten by refreshed track", history[0])
	}
}

func TestDynamoDBRepositoryDoesNotLeaveLargeTrackPartialsWhenLeaseIsLost(t *testing.T) {
	ctx := context.Background()
	client := &leaseLosingTrackDynamoDBClient{MemoryDynamoDBClient: NewMemoryDynamoDBClient(), loseAfterPositionWrites: 100}
	repo := NewDynamoDBRepository(client, DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	flight := testFlight("iflg_large_track_lost", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	leased, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:00:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease() = %v, %v", ok, err)
	}
	trackKey := "tracks/iflg_large_track_lost/hash.json"
	timestamp := domain.ISODateTimeString("2026-04-29T00:01:59Z")
	leased.ActualTrackS3Key = &trackKey
	leased.LatestPositionTimestamp = &timestamp
	positions := make([]domain.FlightPosition, 0, 120)
	for index := 0; index < 120; index++ {
		positions = append(positions, domain.FlightPosition{
			FlightID:  flight.FlightID,
			Latitude:  float64(index),
			Longitude: float64(index),
			Timestamp: fmt.Sprintf("2026-04-29T00:%02d:%02dZ", index/60, index%60),
			Source:    domain.PositionSourceFlightAwareTrack,
		})
	}

	updated, err := repo.UpdateFetchedFlightWithTrackPositions(ctx, leased, positions, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if err != nil {
		t.Fatalf("UpdateFetchedFlightWithTrackPositions() error = %v", err)
	}
	if updated {
		t.Fatal("UpdateFetchedFlightWithTrackPositions() updated = true after lease loss")
	}

	stored, _, err := repo.GetFlight(ctx, flight.FlightID)
	if err != nil {
		t.Fatalf("GetFlight() error = %v", err)
	}
	if stored.ActualTrackS3Key != nil || stored.LatestPositionTimestamp != nil {
		t.Fatalf("stored flight = %#v, want track metadata unchanged after lease loss", stored)
	}
	history, _, err := repo.ListPositions(ctx, flight.FlightID, nil, 200)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("stored partial positions = %d, want rollback after lease loss", len(history))
	}
}

func TestDynamoDBRepositoryDoesNotExposeLargeTrackPositionsBeforeFlightCommit(t *testing.T) {
	ctx := context.Background()
	client := &observingTrackWriteDynamoDBClient{MemoryDynamoDBClient: NewMemoryDynamoDBClient()}
	repo := NewDynamoDBRepository(client, DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	client.repo = repo
	client.ctx = ctx
	flight := testFlight("iflg_large_track_staged", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	leased, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:00:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease() = %v, %v", ok, err)
	}
	positions := make([]domain.FlightPosition, 0, 120)
	for index := 0; index < 120; index++ {
		positions = append(positions, domain.FlightPosition{
			FlightID:  flight.FlightID,
			Latitude:  float64(index),
			Longitude: float64(index),
			Timestamp: fmt.Sprintf("2026-04-29T00:%02d:%02dZ", index/60, index%60),
			Source:    domain.PositionSourceFlightAwareTrack,
		})
	}

	updated, err := repo.UpdateFetchedFlightWithTrackPositions(ctx, leased, positions, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if err != nil {
		t.Fatalf("UpdateFetchedFlightWithTrackPositions() error = %v", err)
	}
	if !updated {
		t.Fatal("UpdateFetchedFlightWithTrackPositions() updated = false, want true")
	}
	if client.visibleDuringWrite != 0 {
		t.Fatalf("visible staged positions during write = %d, want 0 before flight commit", client.visibleDuringWrite)
	}
	history, _, err := repo.ListPositions(ctx, flight.FlightID, nil, 200)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != len(positions) {
		t.Fatalf("visible positions after commit = %d, want %d", len(history), len(positions))
	}

	scheduled, _, err := repo.GetFlight(ctx, flight.FlightID)
	if err != nil {
		t.Fatalf("GetFlight() error = %v", err)
	}
	scheduled.NextPositionPollAt = ptrISO("2026-04-29T00:20:00Z")
	if err := repo.UpdatePollSchedule(ctx, scheduled); err != nil {
		t.Fatalf("UpdatePollSchedule() error = %v", err)
	}
	history, _, err = repo.ListPositions(ctx, flight.FlightID, nil, 200)
	if err != nil {
		t.Fatalf("ListPositions(after schedule update) error = %v", err)
	}
	if len(history) != len(positions) {
		t.Fatalf("visible positions after schedule update = %d, want committed large track positions preserved", len(history))
	}
}

func TestDynamoDBRepositoryClearsLargeTrackWriteTokensAfterCommit(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryDynamoDBClient()
	repo := NewDynamoDBRepository(client, DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	flight := testFlight("iflg_large_track_token_cleanup", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	leased, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:00:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease() = %v, %v", ok, err)
	}

	positions := largeTrackPositions(flight.FlightID, 0, 120)
	updated, err := repo.UpdateFetchedFlightWithTrackPositions(ctx, leased, positions, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if err != nil {
		t.Fatalf("UpdateFetchedFlightWithTrackPositions() error = %v", err)
	}
	if !updated {
		t.Fatal("UpdateFetchedFlightWithTrackPositions() updated = false, want true")
	}

	flightItem, ok, err := client.GetItem(ctx, "Flights", "flightId", string(flight.FlightID))
	if err != nil {
		t.Fatalf("GetItem(flight) error = %v", err)
	}
	if !ok {
		t.Fatal("flight item not found")
	}
	if token := stringValue(flightItem[committedPositionWriteTokenAttribute]); token != "" {
		t.Fatalf("committedPositionWriteToken = %q, want cleared after committed rows are normalized", token)
	}
	if tokens := stringValue(flightItem[committedPositionWriteTokensAttribute]); tokens != "" {
		t.Fatalf("committedPositionWriteTokens = %q, want cleared after committed rows are normalized", tokens)
	}
	positionItems, err := client.QueryRange(ctx, "FlightPositions", "", "flightId", string(flight.FlightID), "timestamp", nil, 0, false)
	if err != nil {
		t.Fatalf("QueryRange(positions) error = %v", err)
	}
	if len(positionItems) != len(positions) {
		t.Fatalf("stored position rows = %d, want %d", len(positionItems), len(positions))
	}
	for _, item := range positionItems {
		if token := stringValue(item[positionWriteTokenAttribute]); token != "" {
			t.Fatalf("position %s retained write token %q after commit cleanup", stringValue(item["timestamp"]), token)
		}
	}
}

func TestDynamoDBRepositoryReportsLargeTrackCommitCleanupLeaseMiss(t *testing.T) {
	ctx := context.Background()
	client := &leaseLosingFlightTokenCleanupDynamoDBClient{MemoryDynamoDBClient: NewMemoryDynamoDBClient()}
	repo := NewDynamoDBRepository(client, DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	flight := testFlight("iflg_large_track_cleanup_lease_miss", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	leased, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:00:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease() = %v, %v", ok, err)
	}

	positions := largeTrackPositions(flight.FlightID, 0, 120)
	updated, err := repo.UpdateFetchedFlightWithTrackPositions(ctx, leased, positions, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if err != nil {
		t.Fatalf("UpdateFetchedFlightWithTrackPositions() error = %v", err)
	}
	if updated {
		t.Fatal("UpdateFetchedFlightWithTrackPositions() updated = true after final token cleanup lease miss")
	}

	flightItem, ok, err := client.GetItem(ctx, "Flights", "flightId", string(flight.FlightID))
	if err != nil {
		t.Fatalf("GetItem(flight) error = %v", err)
	}
	if !ok {
		t.Fatal("flight item not found")
	}
	if token := stringValue(flightItem[committedPositionWriteTokenAttribute]); token == "" {
		t.Fatal("committedPositionWriteToken was cleared despite cleanup lease miss")
	}
	positionItems, err := client.QueryRange(ctx, "FlightPositions", "", "flightId", string(flight.FlightID), "timestamp", nil, 0, false)
	if err != nil {
		t.Fatalf("QueryRange(positions) error = %v", err)
	}
	for _, item := range positionItems {
		if token := stringValue(item[positionWriteTokenAttribute]); token != "" {
			t.Fatalf("position %s retained write token %q after row cleanup completed", stringValue(item["timestamp"]), token)
		}
	}
}

func TestDynamoDBRepositoryKeepsCommittedLargeTrackTokenWhenCleanupMissesRow(t *testing.T) {
	ctx := context.Background()
	client := &cleanupMissingTrackDynamoDBClient{MemoryDynamoDBClient: NewMemoryDynamoDBClient(), missCleanupOnce: true}
	repo := NewDynamoDBRepository(client, DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	flight := testFlight("iflg_large_track_cleanup_miss", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	leased, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:00:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease() = %v, %v", ok, err)
	}

	positions := largeTrackPositions(flight.FlightID, 0, 120)
	updated, err := repo.UpdateFetchedFlightWithTrackPositions(ctx, leased, positions, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if err != nil {
		t.Fatalf("UpdateFetchedFlightWithTrackPositions() error = %v", err)
	}
	if !updated {
		t.Fatal("UpdateFetchedFlightWithTrackPositions() updated = false, want true")
	}

	history, _, err := repo.ListPositions(ctx, flight.FlightID, nil, 200)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != len(positions) {
		t.Fatalf("visible positions after incomplete cleanup = %d, want %d", len(history), len(positions))
	}

	flightItem, ok, err := client.GetItem(ctx, "Flights", "flightId", string(flight.FlightID))
	if err != nil {
		t.Fatalf("GetItem(flight) error = %v", err)
	}
	if !ok {
		t.Fatal("flight item not found")
	}
	if token := stringValue(flightItem[committedPositionWriteTokenAttribute]); token == "" {
		t.Fatal("committedPositionWriteToken was cleared after incomplete cleanup")
	}

	positionItems, err := client.QueryRange(ctx, "FlightPositions", "", "flightId", string(flight.FlightID), "timestamp", nil, 0, false)
	if err != nil {
		t.Fatalf("QueryRange(positions) error = %v", err)
	}
	retainedPositionTokens := 0
	for _, item := range positionItems {
		if stringValue(item[positionWriteTokenAttribute]) != "" {
			retainedPositionTokens++
		}
	}
	if retainedPositionTokens != 1 {
		t.Fatalf("position rows with retained write tokens = %d, want 1", retainedPositionTokens)
	}
}

func TestDynamoDBRepositoryKeepsCleanupMissedLargeTrackRowsAfterNextLargeCommit(t *testing.T) {
	ctx := context.Background()
	client := &cleanupMissingTrackDynamoDBClient{MemoryDynamoDBClient: NewMemoryDynamoDBClient(), missCleanupOnce: true}
	repo := NewDynamoDBRepository(client, DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	flight := testFlight("iflg_large_track_cleanup_miss_next", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}

	firstLease, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:00:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease(first) = %v, %v", ok, err)
	}
	firstPositions := largeTrackPositions(flight.FlightID, 0, 120)
	updated, err := repo.UpdateFetchedFlightWithTrackPositions(ctx, firstLease, firstPositions, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if err != nil || !updated {
		t.Fatalf("UpdateFetchedFlightWithTrackPositions(first) = %v, %v", updated, err)
	}
	if err := repo.ReleaseFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:06:00Z"); err != nil {
		t.Fatalf("ReleaseFetchLease(first) error = %v", err)
	}

	secondLease, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-2", "2026-04-29T00:20:00Z", "2026-04-29T00:06:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease(second) = %v, %v", ok, err)
	}
	secondPositions := largeTrackPositions(flight.FlightID, 120, 120)
	updated, err = repo.UpdateFetchedFlightWithTrackPositions(ctx, secondLease, secondPositions, "worker-2", "2026-04-29T00:20:00Z", "2026-04-29T00:10:00Z")
	if err != nil || !updated {
		t.Fatalf("UpdateFetchedFlightWithTrackPositions(second) = %v, %v", updated, err)
	}

	history, _, err := repo.ListPositions(ctx, flight.FlightID, nil, 300)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != len(firstPositions)+len(secondPositions) {
		t.Fatalf("visible positions after next large commit = %d, want %d", len(history), len(firstPositions)+len(secondPositions))
	}
}

func TestDynamoDBRepositoryKeepsPreviousCommittedLargeTrackRowsAfterSecondCommit(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	flight := testFlight("iflg_large_track_second_commit", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}

	firstLease, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:00:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease(first) = %v, %v", ok, err)
	}
	firstPositions := largeTrackPositions(flight.FlightID, 0, 120)
	updated, err := repo.UpdateFetchedFlightWithTrackPositions(ctx, firstLease, firstPositions, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if err != nil || !updated {
		t.Fatalf("UpdateFetchedFlightWithTrackPositions(first) = %v, %v", updated, err)
	}
	if err := repo.ReleaseFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:06:00Z"); err != nil {
		t.Fatalf("ReleaseFetchLease(first) error = %v", err)
	}

	secondLease, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-2", "2026-04-29T00:20:00Z", "2026-04-29T00:06:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease(second) = %v, %v", ok, err)
	}
	secondPositions := largeTrackPositions(flight.FlightID, 120, 120)
	updated, err = repo.UpdateFetchedFlightWithTrackPositions(ctx, secondLease, secondPositions, "worker-2", "2026-04-29T00:20:00Z", "2026-04-29T00:10:00Z")
	if err != nil || !updated {
		t.Fatalf("UpdateFetchedFlightWithTrackPositions(second) = %v, %v", updated, err)
	}

	history, _, err := repo.ListPositions(ctx, flight.FlightID, nil, 300)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != len(firstPositions)+len(secondPositions) {
		t.Fatalf("visible positions after second commit = %d, want %d", len(history), len(firstPositions)+len(secondPositions))
	}
	if history[0].Timestamp != firstPositions[0].Timestamp {
		t.Fatalf("first visible timestamp = %q, want previous committed row %q", history[0].Timestamp, firstPositions[0].Timestamp)
	}
	if history[len(history)-1].Timestamp != secondPositions[len(secondPositions)-1].Timestamp {
		t.Fatalf("last visible timestamp = %q, want latest committed row %q", history[len(history)-1].Timestamp, secondPositions[len(secondPositions)-1].Timestamp)
	}
}

func TestDynamoDBRepositoryPreservesExistingLargeTrackHistoryOnRollback(t *testing.T) {
	ctx := context.Background()
	client := &leaseLosingTrackDynamoDBClient{MemoryDynamoDBClient: NewMemoryDynamoDBClient(), loseAfterPositionWrites: 2}
	repo := NewDynamoDBRepository(client, DynamoDBTables{
		Flights:         "Flights",
		FlightPositions: "FlightPositions",
	})
	flight := testFlight("iflg_large_track_existing", "ANA110")
	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}
	existing := []domain.FlightPosition{
		{
			FlightID:  flight.FlightID,
			Latitude:  35.55,
			Longitude: 139.78,
			Timestamp: "2026-04-29T00:00:00Z",
			Source:    domain.PositionSourceFlightAwareTrack,
		},
		{
			FlightID:  flight.FlightID,
			Latitude:  36.00,
			Longitude: 140.00,
			Timestamp: "2026-04-29T00:00:01Z",
			Source:    domain.PositionSourceFlightAwareTrack,
		},
	}
	for _, position := range existing {
		if err := repo.PutPosition(ctx, position); err != nil {
			t.Fatalf("PutPosition() error = %v", err)
		}
	}
	leased, ok, err := repo.AcquireFetchLease(ctx, flight.FlightID, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:00:00Z")
	if err != nil || !ok {
		t.Fatalf("AcquireFetchLease() = %v, %v", ok, err)
	}
	positions := make([]domain.FlightPosition, 0, 120)
	for index := 0; index < 120; index++ {
		positions = append(positions, domain.FlightPosition{
			FlightID:  flight.FlightID,
			Latitude:  float64(index),
			Longitude: float64(index),
			Timestamp: fmt.Sprintf("2026-04-29T00:%02d:%02dZ", index/60, index%60),
			Source:    domain.PositionSourceFlightAwareTrack,
		})
	}

	updated, err := repo.UpdateFetchedFlightWithTrackPositions(ctx, leased, positions, "worker-1", "2026-04-29T00:10:00Z", "2026-04-29T00:05:00Z")
	if err != nil {
		t.Fatalf("UpdateFetchedFlightWithTrackPositions() error = %v", err)
	}
	if updated {
		t.Fatal("UpdateFetchedFlightWithTrackPositions() updated = true after lease loss")
	}
	history, _, err := repo.ListPositions(ctx, flight.FlightID, nil, 200)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != len(existing) {
		t.Fatalf("stored positions = %d, want only %d pre-existing positions", len(history), len(existing))
	}
	for index, position := range existing {
		if history[index].Timestamp != position.Timestamp || history[index].Latitude != position.Latitude || history[index].Longitude != position.Longitude {
			t.Fatalf("history[%d] = %#v, want existing %#v preserved", index, history[index], position)
		}
	}
}

func largeTrackPositions(flightID domain.FlightID, startSecond int, count int) []domain.FlightPosition {
	positions := make([]domain.FlightPosition, 0, count)
	for index := 0; index < count; index++ {
		offset := startSecond + index
		positions = append(positions, domain.FlightPosition{
			FlightID:  flightID,
			Latitude:  float64(offset),
			Longitude: float64(offset),
			Timestamp: fmt.Sprintf("2026-04-29T00:%02d:%02dZ", offset/60, offset%60),
			Source:    domain.PositionSourceFlightAwareTrack,
		})
	}
	return positions
}

func TestDynamoDBRepositoryRollbackDoesNotDeleteNewerPositionRows(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryDynamoDBClient()
	repo := NewDynamoDBRepository(client, DynamoDBTables{FlightPositions: "FlightPositions"})
	staleWritten := positionItem(domain.FlightPosition{
		FlightID:  "iflg_large_track_newer",
		Latitude:  35,
		Longitude: 139,
		Timestamp: "2026-04-29T00:00:00Z",
		Source:    domain.PositionSourceFlightAwareTrack,
	})
	if err := client.PutItem(ctx, "FlightPositions", staleWritten); err != nil {
		t.Fatalf("PutItem(staleWritten) error = %v", err)
	}
	newer := domain.FlightPosition{
		FlightID:  "iflg_large_track_newer",
		Latitude:  36,
		Longitude: 140,
		Timestamp: "2026-04-29T00:00:00Z",
		Source:    domain.PositionSourceFlightAwareTrack,
	}
	if err := client.PutItem(ctx, "FlightPositions", positionItem(newer)); err != nil {
		t.Fatalf("PutItem(newer) error = %v", err)
	}

	repo.rollbackPositionItemsBestEffort(ctx, []largeTrackPositionWrite{{item: staleWritten, existed: false}})

	history, _, err := repo.ListPositions(ctx, newer.FlightID, nil, 10)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != 1 || history[0].Latitude != newer.Latitude || history[0].Longitude != newer.Longitude {
		t.Fatalf("history after rollback = %#v, want newer row preserved", history)
	}
}

func TestDynamoDBRepositoryRollbackDoesNotDeleteNewerIdenticalPositionRows(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryDynamoDBClient()
	repo := NewDynamoDBRepository(client, DynamoDBTables{FlightPositions: "FlightPositions"})

	position := domain.FlightPosition{
		FlightID:  "iflg_large_track_identical_newer",
		Latitude:  35,
		Longitude: 139,
		Timestamp: "2026-04-29T00:00:00Z",
		Source:    domain.PositionSourceFlightAwareTrack,
	}
	staleWritten := positionItem(position)
	staleWritten["positionWriteToken"] = "stale-write"
	newerWritten := positionItem(position)
	newerWritten["positionWriteToken"] = "newer-write"
	if err := client.PutItem(ctx, "FlightPositions", newerWritten); err != nil {
		t.Fatalf("PutItem(newerWritten) error = %v", err)
	}

	repo.rollbackPositionItemsBestEffort(ctx, []largeTrackPositionWrite{{item: staleWritten, existed: false}})

	item, ok, err := repo.getPositionItem(ctx, staleWritten)
	if err != nil {
		t.Fatalf("getPositionItem() error = %v", err)
	}
	if !ok {
		t.Fatal("position was deleted by stale rollback, want newer identical row preserved")
	}
	if got := stringValue(item["positionWriteToken"]); got != "newer-write" {
		t.Fatalf("positionWriteToken = %q, want newer-write", got)
	}
}

func TestDynamoDBRepositoryRollbackDoesNotOverwriteNewerPositionRows(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryDynamoDBClient()
	repo := NewDynamoDBRepository(client, DynamoDBTables{FlightPositions: "FlightPositions"})
	previous := positionItem(domain.FlightPosition{
		FlightID:  "iflg_large_track_existing_newer",
		Latitude:  35,
		Longitude: 139,
		Timestamp: "2026-04-29T00:00:00Z",
		Source:    domain.PositionSourceFlightAwareTrack,
	})
	staleWritten := positionItem(domain.FlightPosition{
		FlightID:  "iflg_large_track_existing_newer",
		Latitude:  35.5,
		Longitude: 139.5,
		Timestamp: "2026-04-29T00:00:00Z",
		Source:    domain.PositionSourceFlightAwareTrack,
	})
	if err := client.PutItem(ctx, "FlightPositions", staleWritten); err != nil {
		t.Fatalf("PutItem(staleWritten) error = %v", err)
	}
	newer := domain.FlightPosition{
		FlightID:  "iflg_large_track_existing_newer",
		Latitude:  36,
		Longitude: 140,
		Timestamp: "2026-04-29T00:00:00Z",
		Source:    domain.PositionSourceFlightAwareTrack,
	}
	if err := client.PutItem(ctx, "FlightPositions", positionItem(newer)); err != nil {
		t.Fatalf("PutItem(newer) error = %v", err)
	}

	repo.rollbackPositionItemsBestEffort(ctx, []largeTrackPositionWrite{{item: staleWritten, previous: previous, existed: true}})

	history, _, err := repo.ListPositions(ctx, newer.FlightID, nil, 10)
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if len(history) != 1 || history[0].Latitude != newer.Latitude || history[0].Longitude != newer.Longitude {
		t.Fatalf("history after rollback = %#v, want newer row preserved", history)
	}
}

func TestTransactionCanceledWithoutReasonsIsNotTreatedAsConditionFailure(t *testing.T) {
	if transactionCanceledBecauseConditionFailed(&ddbtypes.TransactionCanceledException{}) {
		t.Fatal("empty TransactionCanceledException reasons were treated as a conditional lease miss")
	}
	if !transactionCanceledBecauseConditionFailed(&ddbtypes.TransactionCanceledException{
		CancellationReasons: []ddbtypes.CancellationReason{{Code: ptrString("ConditionalCheckFailed")}},
	}) {
		t.Fatal("ConditionalCheckFailed cancellation reason was not treated as a lease miss")
	}
	if !transactionCanceledBecauseConditionFailed(&ddbtypes.TransactionCanceledException{
		CancellationReasons: []ddbtypes.CancellationReason{
			{Code: ptrString("ConditionalCheckFailed")},
			{Code: ptrString("None")},
		},
	}) {
		t.Fatal("ConditionalCheckFailed with None cancellation reason was not treated as a lease miss")
	}
	if transactionCanceledBecauseConditionFailed(&ddbtypes.TransactionCanceledException{
		CancellationReasons: []ddbtypes.CancellationReason{
			{Code: ptrString("ConditionalCheckFailed")},
			{Code: ptrString("TransactionConflict")},
		},
	}) {
		t.Fatal("mixed cancellation reasons were treated as a conditional lease miss")
	}
}

type countingCollectionDynamoDBClient struct {
	*MemoryDynamoDBClient
	maxExtraItems   int
	collectionCalls int
	positionWrites  int
}

type countingPollPageDynamoDBClient struct {
	*MemoryDynamoDBClient
	pageCalls         int
	maxPageLimit      int
	sawUnboundedQuery bool
}

func (c *countingPollPageDynamoDBClient) QueryRangeUntilPage(ctx context.Context, table string, indexName string, hashName string, hashValue string, rangeName string, until string, limit int, exclusiveStartKey map[string]any) ([]map[string]any, map[string]any, error) {
	c.pageCalls++
	if limit == 0 {
		c.sawUnboundedQuery = true
	}
	if limit > c.maxPageLimit {
		c.maxPageLimit = limit
	}
	return c.MemoryDynamoDBClient.QueryRangeUntilPage(ctx, table, indexName, hashName, hashValue, rangeName, until, limit, exclusiveStartKey)
}

func (c *countingCollectionDynamoDBClient) PutItemCollectionIfLeaseOwner(ctx context.Context, table string, item map[string]any, extraTable string, extraItems []map[string]any, owner string, leaseUntil string, now string) (bool, error) {
	c.collectionCalls++
	if len(extraItems) > c.maxExtraItems {
		c.maxExtraItems = len(extraItems)
	}
	if len(extraItems) > 99 {
		return false, fmt.Errorf("test transaction item count %d exceeds DynamoDB limit", len(extraItems)+1)
	}
	return c.MemoryDynamoDBClient.PutItemCollectionIfLeaseOwner(ctx, table, item, extraTable, extraItems, owner, leaseUntil, now)
}

func (c *countingCollectionDynamoDBClient) PutItemIfOtherItemLeaseOwner(ctx context.Context, table string, item map[string]any, conditionTable string, conditionKey map[string]any, owner string, leaseUntil string, now string) (bool, error) {
	c.positionWrites++
	return c.MemoryDynamoDBClient.PutItemIfOtherItemLeaseOwner(ctx, table, item, conditionTable, conditionKey, owner, leaseUntil, now)
}

type leaseLosingTrackDynamoDBClient struct {
	*MemoryDynamoDBClient
	positionWrites          int
	loseAfterPositionWrites int
}

func (c *leaseLosingTrackDynamoDBClient) PutItemCollectionIfLeaseOwner(ctx context.Context, table string, item map[string]any, extraTable string, extraItems []map[string]any, owner string, leaseUntil string, now string) (bool, error) {
	c.positionWrites += len(extraItems)
	if c.positionWrites >= c.loseAfterPositionWrites {
		c.stealLease(table, stringValue(item["flightId"]))
	}
	return c.MemoryDynamoDBClient.PutItemCollectionIfLeaseOwner(ctx, table, item, extraTable, extraItems, owner, leaseUntil, now)
}

func (c *leaseLosingTrackDynamoDBClient) PutItemIfOtherItemLeaseOwner(ctx context.Context, table string, item map[string]any, conditionTable string, conditionKey map[string]any, owner string, leaseUntil string, now string) (bool, error) {
	c.positionWrites++
	if c.positionWrites >= c.loseAfterPositionWrites {
		c.stealLease(conditionTable, stringValue(conditionKey["flightId"]))
	}
	return c.MemoryDynamoDBClient.PutItemIfOtherItemLeaseOwner(ctx, table, item, conditionTable, conditionKey, owner, leaseUntil, now)
}

func (c *leaseLosingTrackDynamoDBClient) stealLease(table string, flightID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item := c.tables[table][flightID]
	if item == nil {
		return
	}
	item["fetchOwner"] = "worker-2"
	item["fetchLeaseUntil"] = "2026-04-29T00:15:00Z"
}

type observingTrackWriteDynamoDBClient struct {
	*MemoryDynamoDBClient
	repo               *DynamoDBRepository
	ctx                context.Context
	positionWrites     int
	visibleDuringWrite int
}

func (c *observingTrackWriteDynamoDBClient) PutItemIfOtherItemLeaseOwner(ctx context.Context, table string, item map[string]any, conditionTable string, conditionKey map[string]any, owner string, leaseUntil string, now string) (bool, error) {
	updated, err := c.MemoryDynamoDBClient.PutItemIfOtherItemLeaseOwner(ctx, table, item, conditionTable, conditionKey, owner, leaseUntil, now)
	if err != nil || !updated {
		return updated, err
	}
	c.positionWrites++
	if c.positionWrites == 10 {
		history, _, err := c.repo.ListPositions(c.ctx, domain.FlightID(stringValue(conditionKey["flightId"])), nil, 200)
		if err != nil {
			return false, err
		}
		c.visibleDuringWrite = len(history)
	}
	return updated, nil
}

type leaseLosingFlightTokenCleanupDynamoDBClient struct {
	*MemoryDynamoDBClient
	publishedLargeTrackToken bool
	lostCleanupLease         bool
}

func (c *leaseLosingFlightTokenCleanupDynamoDBClient) PutItemIfLeaseOwner(ctx context.Context, table string, item map[string]any, owner string, leaseUntil string, now string) (bool, error) {
	if table == "Flights" && stringValue(item[committedPositionWriteTokenAttribute]) != "" {
		c.publishedLargeTrackToken = true
		return c.MemoryDynamoDBClient.PutItemIfLeaseOwner(ctx, table, item, owner, leaseUntil, now)
	}
	if table == "Flights" && c.publishedLargeTrackToken && !c.lostCleanupLease && stringValue(item[committedPositionWriteTokenAttribute]) == "" {
		c.lostCleanupLease = true
		c.stealLease(table, stringValue(item["flightId"]))
	}
	return c.MemoryDynamoDBClient.PutItemIfLeaseOwner(ctx, table, item, owner, leaseUntil, now)
}

func (c *leaseLosingFlightTokenCleanupDynamoDBClient) stealLease(table string, flightID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item := c.tables[table][flightID]
	if item == nil {
		return
	}
	item["fetchOwner"] = "worker-2"
	item["fetchLeaseUntil"] = "2026-04-29T00:15:00Z"
}

type cleanupMissingTrackDynamoDBClient struct {
	*MemoryDynamoDBClient
	missCleanupOnce bool
}

func (c *cleanupMissingTrackDynamoDBClient) PutItemIfCurrentAttributesEqual(ctx context.Context, table string, item map[string]any, expected map[string]any) (bool, error) {
	if table == "FlightPositions" && stringValue(expected[positionWriteTokenAttribute]) != "" && stringValue(item[positionWriteTokenAttribute]) == "" && c.missCleanupOnce {
		c.missCleanupOnce = false
		return false, nil
	}
	return c.MemoryDynamoDBClient.PutItemIfCurrentAttributesEqual(ctx, table, item, expected)
}

type staleUsageReadDynamoDBClient struct {
	*MemoryDynamoDBClient
	staleItem map[string]any
}

type staleFlightReadDynamoDBClient struct {
	*MemoryDynamoDBClient
	staleNextFlightGet map[string]any
}

func (c *staleFlightReadDynamoDBClient) GetItem(ctx context.Context, table string, hashName string, hashValue string) (map[string]any, bool, error) {
	if table == "Flights" && hashName == "flightId" && c.staleNextFlightGet != nil && stringValue(c.staleNextFlightGet["flightId"]) == hashValue {
		item := cloneItem(c.staleNextFlightGet)
		c.staleNextFlightGet = nil
		return item, true, nil
	}
	return c.MemoryDynamoDBClient.GetItem(ctx, table, hashName, hashValue)
}

func (c *staleUsageReadDynamoDBClient) PutItem(ctx context.Context, table string, item map[string]any) error {
	if table == "UsageBudget" {
		c.staleItem = cloneItem(item)
	}
	return c.MemoryDynamoDBClient.PutItem(ctx, table, item)
}

func (c *staleUsageReadDynamoDBClient) AddUsageEstimate(ctx context.Context, table string, budgetScope string, defaults map[string]any, delta float64) (map[string]any, error) {
	return c.MemoryDynamoDBClient.AddUsageEstimate(ctx, table, budgetScope, defaults, delta)
}

func (c *staleUsageReadDynamoDBClient) GetItem(ctx context.Context, table string, hashName string, hashValue string) (map[string]any, bool, error) {
	if table == "UsageBudget" && hashName == "budgetScope" && hashValue == "dev#2026-04" && c.staleItem != nil {
		return cloneItem(c.staleItem), true, nil
	}
	return c.MemoryDynamoDBClient.GetItem(ctx, table, hashName, hashValue)
}

type concurrentUsageReconcileDynamoDBClient struct {
	*MemoryDynamoDBClient
	raiseCostBeforeNextReconcileWrite bool
}

func (c *concurrentUsageReconcileDynamoDBClient) PutItem(ctx context.Context, table string, item map[string]any) error {
	if c.raiseCostBeforeNextReconcileWrite && table == "UsageBudget" && numberValue(item["estimatedMonthToDateCost"]) == 4.25 {
		c.raiseCostBeforeNextReconcileWrite = false
		current := cloneItem(item)
		current["estimatedMonthToDateCost"] = 5.00
		if err := c.MemoryDynamoDBClient.PutItem(ctx, table, current); err != nil {
			return err
		}
	}
	return c.MemoryDynamoDBClient.PutItem(ctx, table, item)
}

func (c *concurrentUsageReconcileDynamoDBClient) PutUsageEstimateIfHigher(ctx context.Context, table string, budgetScope string, defaults map[string]any, estimate float64) (map[string]any, bool, error) {
	if c.raiseCostBeforeNextReconcileWrite {
		c.raiseCostBeforeNextReconcileWrite = false
		current, ok, err := c.MemoryDynamoDBClient.GetItem(ctx, table, "budgetScope", budgetScope)
		if err != nil {
			return nil, false, err
		}
		if !ok {
			current = map[string]any{"budgetScope": budgetScope}
		}
		current["estimatedMonthToDateCost"] = 5.00
		if err := c.MemoryDynamoDBClient.PutItem(ctx, table, current); err != nil {
			return nil, false, err
		}
	}
	return c.MemoryDynamoDBClient.PutUsageEstimateIfHigher(ctx, table, budgetScope, defaults, estimate)
}

func TestDynamoDBRepositoryOmitsNilLeaseAttributes(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryDynamoDBClient()
	repo := NewDynamoDBRepository(client, DynamoDBTables{Flights: "Flights"})
	flight := testFlight("iflg_nil_lease", "ANA110")

	if err := repo.PutFlight(ctx, flight); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}

	item, ok, err := client.GetItem(ctx, "Flights", "flightId", string(flight.FlightID))
	if err != nil {
		t.Fatalf("GetItem() error = %v", err)
	}
	if !ok {
		t.Fatal("stored flight not found")
	}
	if _, ok := item["fetchLeaseUntil"]; ok {
		t.Fatalf("fetchLeaseUntil stored for nil lease: %#v", item["fetchLeaseUntil"])
	}
	if _, ok := item["fetchOwner"]; ok {
		t.Fatalf("fetchOwner stored for nil lease: %#v", item["fetchOwner"])
	}
}

func testFlight(id domain.FlightID, ident string) domain.Flight {
	internalID := domain.InternalFlightLegID(id)
	faFlightID := domain.FAFlightID("fa_aws_1")
	return domain.Flight{
		FlightID:            id,
		FlightIDType:        domain.FlightIDTypeInternal,
		InternalFlightLegID: &internalID,
		FAFlightID:          &faFlightID,
		Ident:               ident,
		Origin:              domain.Airport{Code: "RJTT"},
		Destination:         domain.Airport{Code: "KJFK"},
		Status:              "Scheduled",
		Times:               domain.FlightTimes{},
		UpdatedAt:           "2026-04-29T00:00:00Z",
	}
}

func ptrISO(value domain.ISODateTimeString) *domain.ISODateTimeString {
	return &value
}
