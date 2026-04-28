package awsintegration

import (
	"context"
	"testing"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"
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
