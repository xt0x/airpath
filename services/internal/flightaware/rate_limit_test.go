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
