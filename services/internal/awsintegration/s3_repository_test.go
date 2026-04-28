package awsintegration

import (
	"context"
	"testing"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"
	"airpath/services/internal/flightaware"
)

func TestS3GeoJSONRepositoryStoresAndLoadsRouteAndTrackArtifacts(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryObjectClient()
	repo := NewS3GeoJSONRepository(client, "geojson-bucket")

	route := application.MapLayer{
		Source:    application.MapSourceFlightAwareRoute,
		Available: true,
		GeoJSON:   map[string]any{"type": "Feature", "kind": "planned"},
	}
	track := application.MapLayer{
		Source:    application.MapSourceFlightAwareTrack,
		Available: true,
		GeoJSON:   map[string]any{"type": "Feature", "kind": "track"},
	}

	routeKey, err := repo.StoreRoute(ctx, "iflg_1", route)
	if err != nil {
		t.Fatalf("StoreRoute() error = %v", err)
	}
	trackKey, err := repo.StoreTrack(ctx, "iflg_1", track)
	if err != nil {
		t.Fatalf("StoreTrack() error = %v", err)
	}

	flight := domain.Flight{
		FlightID:          "iflg_1",
		PlannedRouteS3Key: &routeKey,
		ActualTrackS3Key:  &trackKey,
	}
	gotRoute, err := repo.GetPlannedRoute(ctx, flight)
	if err != nil {
		t.Fatalf("GetPlannedRoute() error = %v", err)
	}
	gotTrack, err := repo.GetActualTrack(ctx, flight)
	if err != nil {
		t.Fatalf("GetActualTrack() error = %v", err)
	}
	if gotRoute.Source != application.MapSourceFlightAwareRoute || gotRoute.GeoJSON["kind"] != "planned" {
		t.Fatalf("route = %#v", gotRoute)
	}
	if gotTrack.Source != application.MapSourceFlightAwareTrack || gotTrack.GeoJSON["kind"] != "track" {
		t.Fatalf("track = %#v", gotTrack)
	}
}

func TestS3GeoJSONRepositoryStoresFlightAwareRouteAndTrackResponses(t *testing.T) {
	ctx := context.Background()
	repo := NewS3GeoJSONRepository(NewMemoryObjectClient(), "geojson-bucket")

	routeKey, err := repo.StoreFlightAwareRoute(ctx, "iflg_1", flightaware.RouteResponse{
		Fixes: []flightaware.RouteFix{
			{Name: "RJTT", Latitude: ptrFloat64(35.55), Longitude: ptrFloat64(139.78)},
			{Name: "KJFK", Latitude: ptrFloat64(40.64), Longitude: ptrFloat64(-73.78)},
		},
	})
	if err != nil {
		t.Fatalf("StoreFlightAwareRoute() error = %v", err)
	}
	trackKey, err := repo.StoreFlightAwareTrack(ctx, "iflg_1", flightaware.TrackResponse{
		Positions: []flightaware.TrackPoint{
			{Latitude: 35.55, Longitude: 139.78, Timestamp: "2026-06-20T08:30:00Z"},
			{Latitude: 40.64, Longitude: -73.78, Timestamp: "2026-06-20T17:30:00Z"},
		},
	})
	if err != nil {
		t.Fatalf("StoreFlightAwareTrack() error = %v", err)
	}

	flight := domain.Flight{FlightID: "iflg_1", PlannedRouteS3Key: &routeKey, ActualTrackS3Key: &trackKey}
	route, err := repo.GetPlannedRoute(ctx, flight)
	if err != nil {
		t.Fatalf("GetPlannedRoute() error = %v", err)
	}
	track, err := repo.GetActualTrack(ctx, flight)
	if err != nil {
		t.Fatalf("GetActualTrack() error = %v", err)
	}

	if route.Source != application.MapSourceFlightAwareRoute || !route.Available {
		t.Fatalf("route layer = %#v", route)
	}
	if route.GeoJSON["type"] != "Feature" {
		t.Fatalf("route geojson = %#v", route.GeoJSON)
	}
	if track.Source != application.MapSourceFlightAwareTrack || !track.Available {
		t.Fatalf("track layer = %#v", track)
	}
}

func ptrFloat64(value float64) *float64 {
	return &value
}
