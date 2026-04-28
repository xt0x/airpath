package awsintegration

import (
	"context"
	"encoding/json"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"
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
