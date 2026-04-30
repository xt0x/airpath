package awsintegration

import (
	"context"
	"os"
	"testing"
	"time"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"
	"airpath/services/internal/flightaware"
)

func TestOptInFlightAwareSearchRealCallUsesMaxPagesOne(t *testing.T) {
	client := newOptInFlightAwareClient(t)
	ident := requiredOptInEnv(t, "FLIGHTAWARE_TEST_IDENT")

	response, err := client.SearchFlights(context.Background(), flightaware.SearchFlightsRequest{Ident: ident})
	if err != nil {
		t.Fatalf("SearchFlights(real) error = %v", err)
	}
	if len(response.Flights) == 0 {
		t.Fatalf("SearchFlights(real) returned no flights for %s", ident)
	}
}

func TestOptInFlightAwareRouteTrackAndPositionRealCallsNormalizeResponses(t *testing.T) {
	client := newOptInFlightAwareClient(t)
	faFlightID := optionalOptInEnv(t, "FLIGHTAWARE_TEST_FA_FLIGHT_ID")
	if faFlightID == "" {
		t.Skip("FLIGHTAWARE_TEST_FA_FLIGHT_ID is required for route/track/position real-call verification")
	}
	flightID := domain.FlightID("iflg_opt_in")
	adapter := NewFlightAwareFetchAdapter(client)
	geoRepo := NewS3GeoJSONRepository(NewMemoryObjectClient(), "geojson")
	dynamoRepo := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{FlightPositions: "FlightPositions"})

	route, err := adapter.GetFlightRoute(context.Background(), faFlightID)
	if err != nil {
		t.Fatalf("GetFlightRoute(real) error = %v", err)
	}
	if route.RouteText == nil && len(route.Fixes) == 0 {
		t.Fatalf("GetFlightRoute(real) returned neither route text nor fixes: %#v", route)
	}
	if len(route.Fixes) >= 2 {
		if _, err := geoRepo.StoreRouteArtifact(context.Background(), flightID, route); err != nil {
			t.Fatalf("StoreRouteArtifact(real) error = %v", err)
		}
	}

	track, err := adapter.GetFlightTrack(context.Background(), faFlightID)
	if err != nil {
		t.Fatalf("GetFlightTrack(real) error = %v", err)
	}
	if len(track.Positions) == 0 {
		t.Fatalf("GetFlightTrack(real) returned no positions")
	}
	if len(track.Positions) >= 2 {
		if _, err := geoRepo.StoreTrackArtifact(context.Background(), flightID, track); err != nil {
			t.Fatalf("StoreTrackArtifact(real) error = %v", err)
		}
	}

	position, err := adapter.GetFlightPosition(context.Background(), faFlightID)
	if err != nil {
		t.Fatalf("GetFlightPosition(real) error = %v", err)
	}
	if position.Latitude == nil || position.Longitude == nil || position.Timestamp == "" {
		t.Fatalf("GetFlightPosition(real) did not normalize current position: %#v", position)
	}
	storedPosition := domain.FlightPosition{
		FlightID:  flightID,
		Latitude:  *position.Latitude,
		Longitude: *position.Longitude,
		Timestamp: position.Timestamp,
		Source:    domain.PositionSourceFlightAwarePosition,
	}
	if position.FAFlightID != "" {
		faFlightID := domain.FAFlightID(position.FAFlightID)
		storedPosition.FAFlightID = &faFlightID
	}
	if err := dynamoRepo.PutPosition(context.Background(), storedPosition); err != nil {
		t.Fatalf("PutPosition(real) error = %v", err)
	}
}

func TestOptInFlightAwareUsageRealCallReconcilesLocalBudget(t *testing.T) {
	client := newOptInFlightAwareClient(t)
	usage, err := client.GetAccountUsage(context.Background())
	if err != nil {
		t.Fatalf("GetAccountUsage(real) error = %v", err)
	}

	scope := application.UsageBudgetScope{Environment: "opt-in", Month: time.Now().UTC().Format("2006-01")}
	repo := NewScopedDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{UsageBudget: "UsageBudget"}, scope)
	if err := repo.ReconcileAccountUsage(context.Background(), scope, usage); err != nil {
		t.Fatalf("ReconcileAccountUsage(real) error = %v", err)
	}
	status, err := repo.GetMonthlyUsageStatus(context.Background(), scope)
	if err != nil {
		t.Fatalf("GetMonthlyUsageStatus(real) error = %v", err)
	}
	if status.Budget.EstimatedMonthToDateCost < usage.MonthToDate.EstimatedCostUSD {
		t.Fatalf("reconciled budget = %#v, usage = %#v", status.Budget, usage)
	}
}

func newOptInFlightAwareClient(t *testing.T) flightaware.Client {
	t.Helper()
	if os.Getenv("AIRPATH_FLIGHTAWARE_REAL_TESTS") != "true" {
		t.Skip("set AIRPATH_FLIGHTAWARE_REAL_TESTS=true to run real FlightAware AeroAPI tests")
	}
	key := requiredOptInEnv(t, "FLIGHTAWARE_API_KEY")
	httpClient, err := flightaware.NewHTTPClient(flightaware.HTTPClientConfig{APIKey: key})
	if err != nil {
		t.Fatalf("NewHTTPClient(real) error = %v", err)
	}
	return flightaware.NewMaxPagesClient(
		flightaware.NewRateLimitedClient(httpClient, flightaware.NewMemoryRateLimitState(), 5*time.Minute),
	)
}

func requiredOptInEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is required when AIRPATH_FLIGHTAWARE_REAL_TESTS=true", name)
	}
	return value
}

func optionalOptInEnv(t *testing.T, name string) string {
	t.Helper()
	return os.Getenv(name)
}
