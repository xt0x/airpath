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

func TestDynamoDBRepositoryStoresFlightAwarePositionResponse(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{FlightPositions: "FlightPositions"})

	if err := repo.PutFlightAwarePosition(ctx, "iflg_1", flightaware.PositionResponse{
		FAFlightID: "fa_1",
		Latitude:   ptrFloat64(45.12),
		Longitude:  ptrFloat64(160.45),
		Timestamp:  "2026-06-20T09:30:00Z",
	}); err != nil {
		t.Fatalf("PutFlightAwarePosition() error = %v", err)
	}

	position, _, err := repo.GetLatestPosition(ctx, "iflg_1")
	if err != nil {
		t.Fatalf("GetLatestPosition() error = %v", err)
	}
	if position == nil || position.Latitude != 45.12 || position.Longitude != 160.45 || position.Source != domain.PositionSourceFlightAwarePosition {
		t.Fatalf("position = %#v", position)
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
