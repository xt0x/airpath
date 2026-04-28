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
