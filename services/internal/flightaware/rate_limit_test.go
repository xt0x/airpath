package flightaware

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRateLimitGuardStopsCallsWhenStopFlagIsActive(t *testing.T) {
	state := NewMemoryRateLimitState()
	state.Stop(EndpointSearch, "budget stop")
	client := NewRateLimitedClient(NewFixtureClient(FixtureData{}), state, time.Minute)

	_, err := client.SearchFlights(context.Background(), SearchFlightsRequest{Ident: "ANA110", MaxPages: 1})
	if !errors.Is(err, ErrFlightAwareFetchDisabled) {
		t.Fatalf("SearchFlights() error = %v, want ErrFlightAwareFetchDisabled", err)
	}
}

func TestRateLimitGuardMarks429AndBlocksLaterCalls(t *testing.T) {
	state := NewMemoryRateLimitState()
	client := NewRateLimitedClient(&rateLimitedOnceClient{}, state, time.Minute)

	_, err := client.GetFlightTrack(context.Background(), FlightTrackRequest{FAFlightID: "flight-1"})
	if !errors.Is(err, ErrFlightAwareRateLimited) {
		t.Fatalf("first GetFlightTrack() error = %v, want ErrFlightAwareRateLimited", err)
	}
	if !state.IsStopped(EndpointTrack) {
		t.Fatal("track endpoint was not marked stopped after 429")
	}

	_, err = client.GetFlightTrack(context.Background(), FlightTrackRequest{FAFlightID: "flight-1"})
	if !errors.Is(err, ErrFlightAwareFetchDisabled) {
		t.Fatalf("second GetFlightTrack() error = %v, want ErrFlightAwareFetchDisabled", err)
	}
}

func TestRateLimitGuardExtendsBackoffAndStopsLowPriorityFetchesAfter429(t *testing.T) {
	state := NewMemoryRateLimitState()
	client := NewRateLimitedClient(&positionRateLimitedClient{}, state, 5*time.Minute)

	_, err := client.GetFlightPosition(context.Background(), FlightPositionRequest{FAFlightID: "flight-1"})
	if !errors.Is(err, ErrFlightAwareRateLimited) {
		t.Fatalf("GetFlightPosition() error = %v, want ErrFlightAwareRateLimited", err)
	}

	resetAt := state.ResetAt(EndpointPosition)
	if resetAt == nil || time.Until(*resetAt) < 4*time.Minute {
		t.Fatalf("position resetAt = %v, want roughly five minutes in the future", resetAt)
	}

	_, err = client.GetFlightRoute(context.Background(), FlightRouteRequest{FAFlightID: "flight-1"})
	if !errors.Is(err, ErrFlightAwareFetchDisabled) {
		t.Fatalf("GetFlightRoute() after position 429 error = %v, want ErrFlightAwareFetchDisabled", err)
	}
	if state.ResetAt(EndpointRoute) == nil || state.ResetAt(EndpointTrack) == nil || state.ResetAt(EndpointSchedule) == nil {
		t.Fatalf("low-priority endpoints were not stopped after 429")
	}
}

func TestRateLimitGuardAllowsUsageEndpointWhenOtherEndpointStopped(t *testing.T) {
	state := NewMemoryRateLimitState()
	state.Stop(EndpointRoute, "manual stop")
	client := NewRateLimitedClient(NewFixtureClient(FixtureData{}), state, time.Minute)

	_, err := client.GetAccountUsage(context.Background())
	if err != nil {
		t.Fatalf("GetAccountUsage() error = %v", err)
	}
}

type rateLimitedOnceClient struct {
	Client
}

func (c *rateLimitedOnceClient) GetFlightTrack(context.Context, FlightTrackRequest) (TrackResponse, error) {
	return TrackResponse{}, NewRateLimitedError(EndpointTrack, 429, "too many requests")
}

type positionRateLimitedClient struct {
	Client
}

func (c *positionRateLimitedClient) GetFlightPosition(context.Context, FlightPositionRequest) (PositionResponse, error) {
	return PositionResponse{}, NewRateLimitedError(EndpointPosition, 429, "too many requests")
}

func (c *positionRateLimitedClient) GetFlightRoute(context.Context, FlightRouteRequest) (RouteResponse, error) {
	return RouteResponse{}, nil
}
