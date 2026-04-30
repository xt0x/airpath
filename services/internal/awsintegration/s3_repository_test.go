package awsintegration

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

func TestS3GeoJSONRepositoryStoresExternalRouteAndTrackArtifacts(t *testing.T) {
	ctx := context.Background()
	repo := NewS3GeoJSONRepository(NewMemoryObjectClient(), "geojson-bucket")

	routeKey, err := repo.StoreRouteArtifact(ctx, "iflg_1", application.ExternalRouteResponse{
		Fixes: []application.ExternalRouteFix{
			{Name: "RJTT", Latitude: ptrFloat64(35.55), Longitude: ptrFloat64(139.78)},
			{Name: "KJFK", Latitude: ptrFloat64(40.64), Longitude: ptrFloat64(-73.78)},
		},
	})
	if err != nil {
		t.Fatalf("StoreRouteArtifact() error = %v", err)
	}
	if routeKey == "routes/iflg_1.json" {
		t.Fatalf("route artifact key = %q, want versioned key that cannot overwrite the published route object", routeKey)
	}
	trackKey, err := repo.StoreTrackArtifact(ctx, "iflg_1", application.ExternalTrackResponse{
		Positions: []application.ExternalTrackPoint{
			{Latitude: 35.55, Longitude: 139.78, Timestamp: "2026-06-20T08:30:00Z"},
			{Latitude: 40.64, Longitude: -73.78, Timestamp: "2026-06-20T17:30:00Z"},
		},
	})
	if err != nil {
		t.Fatalf("StoreTrackArtifact() error = %v", err)
	}
	if trackKey == "tracks/iflg_1.json" {
		t.Fatalf("track artifact key = %q, want versioned key that cannot overwrite the published track object", trackKey)
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

func TestAWSS3ObjectClientMapsNoSuchKeyToObjectNotFound(t *testing.T) {
	client := s3.New(s3.Options{
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider("key", "secret", ""),
		BaseEndpoint: aws.String("https://s3.test"),
		HTTPClient: roundTripClient(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     http.Header{"Content-Type": []string{"application/xml"}},
				Body:       io.NopCloser(strings.NewReader(`<Error><Code>NoSuchKey</Code><Message>missing</Message></Error>`)),
				Request:    request,
			}, nil
		}),
	})
	objectClient := NewAWSS3ObjectClient(client)

	_, err := objectClient.GetObject(context.Background(), "bucket", "missing.json")
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("GetObject() error = %v, want ErrObjectNotFound", err)
	}
}

func ptrFloat64(value float64) *float64 {
	return &value
}

type roundTripClient func(*http.Request) (*http.Response, error)

func (c roundTripClient) Do(request *http.Request) (*http.Response, error) {
	return c(request)
}
