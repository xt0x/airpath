package flightaware

import (
	"context"
	"errors"
	"testing"
)

func TestClientInterfaceCoversPersonalMVPEndpoints(t *testing.T) {
	var client Client = NewFixtureClient(FixtureData{
		SearchResponses: map[string]SearchFlightsResponse{
			"ANA110": {Flights: []FlightSummary{{FAFlightID: ptr("flight-1"), Ident: "ANA110", Origin: "RJTT", Destination: "KJFK"}}},
		},
		SummaryResponses: map[string]FlightSummary{
			"flight-1": {FAFlightID: ptr("flight-1"), Ident: "ANA110", Origin: "RJTT", Destination: "KJFK"},
		},
		RouteResponses: map[string]RouteResponse{
			"flight-1": {RouteText: ptr("RJTT FIXA KJFK")},
		},
		PositionResponses: map[string]PositionResponse{
			"flight-1": {FAFlightID: "flight-1", Latitude: ptrFloat64(45.123), Longitude: ptrFloat64(160.456)},
		},
		TrackResponses: map[string]TrackResponse{
			"flight-1": {Positions: []TrackPoint{{Latitude: 45.123, Longitude: 160.456}}},
		},
		ScheduleResponses: map[ScheduleKey]SchedulesResponse{
			{StartDate: "2026-06-20", EndDate: "2026-06-20"}: {Scheduled: []FlightSummary{{Ident: "ANA110"}}},
		},
		UsageResponse: UsageResponse{Currency: "USD", MonthToDate: UsageAmount{EstimatedCostUSD: 1.25, ResultSets: 12}},
	})

	ctx := context.Background()
	if got, err := client.SearchFlights(ctx, SearchFlightsRequest{Ident: "ANA110", MaxPages: 1}); err != nil || len(got.Flights) != 1 {
		t.Fatalf("SearchFlights() = %#v, %v", got, err)
	}
	if got, err := client.GetFlightSummary(ctx, FlightSummaryRequest{FAFlightID: "flight-1"}); err != nil || got.Origin != "RJTT" {
		t.Fatalf("GetFlightSummary() = %#v, %v", got, err)
	}
	if got, err := client.GetFlightRoute(ctx, FlightRouteRequest{FAFlightID: "flight-1"}); err != nil || got.RouteText == nil {
		t.Fatalf("GetFlightRoute() = %#v, %v", got, err)
	}
	if got, err := client.GetFlightPosition(ctx, FlightPositionRequest{FAFlightID: "flight-1"}); err != nil || got.Latitude == nil {
		t.Fatalf("GetFlightPosition() = %#v, %v", got, err)
	}
	if got, err := client.GetFlightTrack(ctx, FlightTrackRequest{FAFlightID: "flight-1"}); err != nil || len(got.Positions) != 1 {
		t.Fatalf("GetFlightTrack() = %#v, %v", got, err)
	}
	if got, err := client.GetSchedules(ctx, SchedulesRequest{StartDate: "2026-06-20", EndDate: "2026-06-20", MaxPages: 1}); err != nil || len(got.Scheduled) != 1 {
		t.Fatalf("GetSchedules() = %#v, %v", got, err)
	}
	if got, err := client.GetAccountUsage(ctx); err != nil || got.MonthToDate.ResultSets != 12 {
		t.Fatalf("GetAccountUsage() = %#v, %v", got, err)
	}
}

func TestFixtureClientReturnsTypedNotFoundAndEnforcesBoundedPages(t *testing.T) {
	client := NewFixtureClient(FixtureData{})

	_, err := client.GetFlightRoute(context.Background(), FlightRouteRequest{FAFlightID: "missing"})
	if !errors.Is(err, ErrFixtureNotFound) {
		t.Fatalf("GetFlightRoute() error = %v, want ErrFixtureNotFound", err)
	}

	_, err = client.SearchFlights(context.Background(), SearchFlightsRequest{Ident: "ANA110", MaxPages: 2})
	if !errors.Is(err, ErrMaxPagesExceeded) {
		t.Fatalf("SearchFlights(max_pages=2) error = %v, want ErrMaxPagesExceeded", err)
	}
}

func TestMaxPagesClientDefaultsListCallsToOnePageAndRejectsUnboundedRequests(t *testing.T) {
	upstream := &capturingListClient{}
	client := NewMaxPagesClient(upstream)

	_, err := client.SearchFlights(context.Background(), SearchFlightsRequest{Ident: "ANA110"})
	if err != nil {
		t.Fatalf("SearchFlights(default max_pages) error = %v", err)
	}
	if upstream.searchMaxPages != DefaultMaxPages {
		t.Fatalf("search max_pages = %d, want %d", upstream.searchMaxPages, DefaultMaxPages)
	}

	_, err = client.GetSchedules(context.Background(), SchedulesRequest{StartDate: "2026-06-20", EndDate: "2026-06-20"})
	if err != nil {
		t.Fatalf("GetSchedules(default max_pages) error = %v", err)
	}
	if upstream.scheduleMaxPages != DefaultMaxPages {
		t.Fatalf("schedule max_pages = %d, want %d", upstream.scheduleMaxPages, DefaultMaxPages)
	}

	_, err = client.SearchFlights(context.Background(), SearchFlightsRequest{Ident: "ANA110", MaxPages: 99})
	if !errors.Is(err, ErrMaxPagesExceeded) {
		t.Fatalf("SearchFlights(max_pages=99) error = %v, want ErrMaxPagesExceeded", err)
	}
	if upstream.searchCalls != 1 {
		t.Fatalf("upstream search calls = %d, want only the bounded default call", upstream.searchCalls)
	}
}

func ptr(value string) *string {
	return &value
}

func ptrFloat64(value float64) *float64 {
	return &value
}

type capturingListClient struct {
	Client
	searchCalls      int
	searchMaxPages   int
	scheduleMaxPages int
}

func (c *capturingListClient) SearchFlights(_ context.Context, request SearchFlightsRequest) (SearchFlightsResponse, error) {
	c.searchCalls++
	c.searchMaxPages = request.MaxPages
	return SearchFlightsResponse{}, nil
}

func (c *capturingListClient) GetSchedules(_ context.Context, request SchedulesRequest) (SchedulesResponse, error) {
	c.scheduleMaxPages = request.MaxPages
	return SchedulesResponse{}, nil
}
