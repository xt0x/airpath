package awsintegration

import (
	"context"
	"testing"

	"airpath/services/internal/domain"
	"airpath/services/internal/flightaware"
)

func TestFlightAwareFetchAdapterMapsPositionMetrics(t *testing.T) {
	altitudeChange := "level"
	updateType := "estimated"
	client := &stubFlightAwareFetchClient{
		position: flightaware.PositionResponse{
			FAFlightID:           "fa_1",
			Latitude:             ptrFloat64(45.12),
			Longitude:            ptrFloat64(160.45),
			AltitudeHundredsFeet: ptrInt(370),
			AltitudeChange:       &altitudeChange,
			GroundspeedKnots:     ptrInt(488),
			HeadingDegrees:       ptrInt(275),
			Timestamp:            "2026-06-20T09:30:00Z",
			UpdateType:           &updateType,
		},
	}
	adapter := NewFlightAwareFetchAdapter(client)

	position, err := adapter.GetFlightPosition(context.Background(), "fa_1")
	if err != nil {
		t.Fatalf("GetFlightPosition() error = %v", err)
	}

	if position.AltitudeHundredsFeet == nil || *position.AltitudeHundredsFeet != 370 {
		t.Fatalf("AltitudeHundredsFeet = %v, want 370", position.AltitudeHundredsFeet)
	}
	if position.AltitudeChange == nil || *position.AltitudeChange != domain.AltitudeChangeLevel {
		t.Fatalf("AltitudeChange = %v, want level", position.AltitudeChange)
	}
	if position.GroundspeedKnots == nil || *position.GroundspeedKnots != 488 {
		t.Fatalf("GroundspeedKnots = %v, want 488", position.GroundspeedKnots)
	}
	if position.HeadingDegrees == nil || *position.HeadingDegrees != 275 {
		t.Fatalf("HeadingDegrees = %v, want 275", position.HeadingDegrees)
	}
	if position.UpdateType == nil || *position.UpdateType != domain.PositionUpdateTypeEstimated {
		t.Fatalf("UpdateType = %v, want estimated", position.UpdateType)
	}
}

func TestFlightAwareFetchAdapterMapsTrackMetrics(t *testing.T) {
	altitudeChange := "climbing"
	updateType := "actual"
	client := &stubFlightAwareFetchClient{
		track: flightaware.TrackResponse{Positions: []flightaware.TrackPoint{{
			Latitude:             35.55,
			Longitude:            139.78,
			AltitudeHundredsFeet: ptrInt(330),
			AltitudeChange:       &altitudeChange,
			GroundspeedKnots:     ptrInt(472),
			HeadingDegrees:       ptrInt(84),
			Timestamp:            "2026-06-20T09:00:00Z",
			UpdateType:           &updateType,
		}}},
	}
	adapter := NewFlightAwareFetchAdapter(client)

	track, err := adapter.GetFlightTrack(context.Background(), "fa_1")
	if err != nil {
		t.Fatalf("GetFlightTrack() error = %v", err)
	}
	if len(track.Positions) != 1 {
		t.Fatalf("track positions = %#v, want one position", track.Positions)
	}
	position := track.Positions[0]
	if position.AltitudeHundredsFeet == nil || *position.AltitudeHundredsFeet != 330 ||
		position.AltitudeChange == nil || *position.AltitudeChange != domain.AltitudeChangeClimbing ||
		position.GroundspeedKnots == nil || *position.GroundspeedKnots != 472 ||
		position.HeadingDegrees == nil || *position.HeadingDegrees != 84 ||
		position.UpdateType == nil || *position.UpdateType != domain.PositionUpdateTypeActual {
		t.Fatalf("track position metrics = %#v, want mapped FlightAware metrics", position)
	}
}

type stubFlightAwareFetchClient struct {
	position flightaware.PositionResponse
	track    flightaware.TrackResponse
}

func (c *stubFlightAwareFetchClient) SearchFlights(context.Context, flightaware.SearchFlightsRequest) (flightaware.SearchFlightsResponse, error) {
	return flightaware.SearchFlightsResponse{}, nil
}

func (c *stubFlightAwareFetchClient) GetFlightRoute(context.Context, flightaware.FlightRouteRequest) (flightaware.RouteResponse, error) {
	return flightaware.RouteResponse{}, nil
}

func (c *stubFlightAwareFetchClient) GetFlightPosition(context.Context, flightaware.FlightPositionRequest) (flightaware.PositionResponse, error) {
	return c.position, nil
}

func (c *stubFlightAwareFetchClient) GetFlightTrack(context.Context, flightaware.FlightTrackRequest) (flightaware.TrackResponse, error) {
	return c.track, nil
}

func ptrInt(value int) *int {
	return &value
}
