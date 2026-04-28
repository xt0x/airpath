package awsintegration

import (
	"context"
	"testing"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"
	"airpath/services/internal/flightaware"
)

func TestDynamoDBRepositoriesReadAndWriteFlightsLookupPositionsAndUsageBudget(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryDynamoDBClient()
	repo := NewDynamoDBRepository(client, DynamoDBTables{
		Flights:         "Flights",
		FlightLookup:    "FlightLookup",
		FlightPositions: "FlightPositions",
		UsageBudget:     "UsageBudget",
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
	if err := repo.PutUsageStatus(ctx, "dev#2026-04", usage); err != nil {
		t.Fatalf("PutUsageStatus() error = %v", err)
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

func TestDynamoDBRepositoryListsDuePollFlightsAndUsesFlightLease(t *testing.T) {
	ctx := context.Background()
	repo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{
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
	if err := repo.ReleaseFetchLease(ctx, due.FlightID, "worker-1", "2026-04-29T00:07:00Z"); err != nil {
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
