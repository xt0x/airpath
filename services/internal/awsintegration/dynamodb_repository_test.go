package awsintegration

import (
	"context"
	"testing"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"
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
