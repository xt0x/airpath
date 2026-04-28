package awsintegration

import (
	"context"
	"encoding/json"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"
	"airpath/services/internal/flightaware"
)

type S3GeoJSONRepository struct {
	client ObjectClient
	bucket string
}

func NewS3GeoJSONRepository(client ObjectClient, bucket string) *S3GeoJSONRepository {
	return &S3GeoJSONRepository{client: client, bucket: bucket}
}

func (r *S3GeoJSONRepository) StoreRoute(ctx context.Context, flightID domain.FlightID, layer application.MapLayer) (string, error) {
	key := "routes/" + string(flightID) + ".json"
	return key, r.putLayer(ctx, key, layer)
}

func (r *S3GeoJSONRepository) StoreTrack(ctx context.Context, flightID domain.FlightID, layer application.MapLayer) (string, error) {
	key := "tracks/" + string(flightID) + ".json"
	return key, r.putLayer(ctx, key, layer)
}

func (r *S3GeoJSONRepository) StoreFlightAwareRoute(ctx context.Context, flightID domain.FlightID, response flightaware.RouteResponse) (string, error) {
	return r.StoreRoute(ctx, flightID, routeLayerFromFlightAware(response))
}

func (r *S3GeoJSONRepository) StoreFlightAwareTrack(ctx context.Context, flightID domain.FlightID, response flightaware.TrackResponse) (string, error) {
	return r.StoreTrack(ctx, flightID, trackLayerFromFlightAware(response))
}

func (r *S3GeoJSONRepository) GetPlannedRoute(ctx context.Context, flight domain.Flight) (application.MapLayer, error) {
	if flight.PlannedRouteS3Key == nil {
		return application.MapLayer{Source: application.MapSourceAirportGreatCircleFallback, Available: false}, nil
	}
	return r.getLayer(ctx, *flight.PlannedRouteS3Key)
}

func (r *S3GeoJSONRepository) GetActualTrack(ctx context.Context, flight domain.Flight) (application.MapLayer, error) {
	if flight.ActualTrackS3Key == nil {
		return application.MapLayer{Source: application.MapSourceFlightAwareTrack, Available: false}, nil
	}
	return r.getLayer(ctx, *flight.ActualTrackS3Key)
}

func (r *S3GeoJSONRepository) putLayer(ctx context.Context, key string, layer application.MapLayer) error {
	body, err := json.Marshal(layer)
	if err != nil {
		return err
	}
	return r.client.PutObject(ctx, r.bucket, key, body)
}

func (r *S3GeoJSONRepository) getLayer(ctx context.Context, key string) (application.MapLayer, error) {
	body, err := r.client.GetObject(ctx, r.bucket, key)
	if err != nil {
		return application.MapLayer{}, err
	}
	var layer application.MapLayer
	if err := json.Unmarshal(body, &layer); err != nil {
		return application.MapLayer{}, err
	}
	return layer, nil
}

func routeLayerFromFlightAware(response flightaware.RouteResponse) application.MapLayer {
	coordinates := make([][]float64, 0, len(response.Fixes))
	for _, fix := range response.Fixes {
		if fix.Latitude == nil || fix.Longitude == nil {
			continue
		}
		coordinates = append(coordinates, []float64{*fix.Longitude, *fix.Latitude})
	}
	if len(coordinates) < 2 {
		return application.MapLayer{Source: application.MapSourceFlightAwareRoute, Available: false, UnavailableReason: ptrString("route coordinates unavailable")}
	}
	return lineLayer(application.MapSourceFlightAwareRoute, "planned_route", coordinates)
}

func trackLayerFromFlightAware(response flightaware.TrackResponse) application.MapLayer {
	coordinates := make([][]float64, 0, len(response.Positions))
	for _, position := range response.Positions {
		coordinates = append(coordinates, []float64{position.Longitude, position.Latitude})
	}
	if len(coordinates) < 2 {
		return application.MapLayer{Source: application.MapSourceFlightAwareTrack, Available: false, UnavailableReason: ptrString("track coordinates unavailable")}
	}
	return lineLayer(application.MapSourceFlightAwareTrack, "actual_track", coordinates)
}

func lineLayer(source application.MapSource, kind string, coordinates [][]float64) application.MapLayer {
	return application.MapLayer{
		Source:    source,
		Available: true,
		GeoJSON: map[string]any{
			"type": "Feature",
			"geometry": map[string]any{
				"type":        "LineString",
				"coordinates": coordinates,
			},
			"properties": map[string]any{"kind": kind},
		},
	}
}

func ptrString(value string) *string {
	return &value
}
