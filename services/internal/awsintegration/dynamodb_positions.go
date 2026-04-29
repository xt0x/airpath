package awsintegration

import (
	"context"
	"encoding/json"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"
	"airpath/services/internal/flightaware"
)

func (r *DynamoDBRepository) PutPosition(ctx context.Context, position domain.FlightPosition) error {
	return r.client.PutItem(ctx, r.tables.FlightPositions, map[string]any{
		"flightId":  position.FlightID,
		"timestamp": position.Timestamp,
		"body":      mustJSON(position),
	})
}

func (r *DynamoDBRepository) AppendPosition(ctx context.Context, position domain.FlightPosition) error {
	return r.PutPosition(ctx, position)
}

func (r *DynamoDBRepository) PutFlightAwarePosition(ctx context.Context, flightID domain.FlightID, response flightaware.PositionResponse) error {
	if response.Latitude == nil || response.Longitude == nil || response.Timestamp == "" {
		return application.ErrValidation
	}
	return r.PutPosition(ctx, domain.FlightPosition{
		FlightID:  flightID,
		Latitude:  *response.Latitude,
		Longitude: *response.Longitude,
		Timestamp: response.Timestamp,
		Source:    domain.PositionSourceFlightAwarePosition,
	})
}

func (r *DynamoDBRepository) GetLatestPosition(ctx context.Context, flightID domain.FlightID) (*domain.FlightPosition, application.CacheMetadata, error) {
	items, err := r.client.QueryByPrefix(ctx, r.tables.FlightPositions, "flightId", string(flightID))
	if err != nil {
		return nil, application.CacheMetadata{}, err
	}
	if len(items) == 0 {
		return nil, application.CacheMetadata{Freshness: application.CacheFreshnessMiss, Source: application.CacheSourceCache}, nil
	}
	var position domain.FlightPosition
	if err := json.Unmarshal([]byte(stringValue(items[len(items)-1]["body"])), &position); err != nil {
		return nil, application.CacheMetadata{}, err
	}
	return &position, cacheFor(true), nil
}
