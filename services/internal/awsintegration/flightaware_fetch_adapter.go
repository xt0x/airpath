package awsintegration

import (
	"context"
	"errors"

	"airpath/services/internal/application"
	"airpath/services/internal/flightaware"
)

type FlightAwareFetchClient interface {
	GetFlightRoute(context.Context, flightaware.FlightRouteRequest) (flightaware.RouteResponse, error)
	GetFlightPosition(context.Context, flightaware.FlightPositionRequest) (flightaware.PositionResponse, error)
	GetFlightTrack(context.Context, flightaware.FlightTrackRequest) (flightaware.TrackResponse, error)
}

type FlightAwareFetchAdapter struct {
	client FlightAwareFetchClient
}

func NewFlightAwareFetchAdapter(client FlightAwareFetchClient) *FlightAwareFetchAdapter {
	return &FlightAwareFetchAdapter{client: client}
}

func (a *FlightAwareFetchAdapter) GetFlightRoute(ctx context.Context, faFlightID string) (application.ExternalRouteResponse, error) {
	response, err := a.client.GetFlightRoute(ctx, flightaware.FlightRouteRequest{FAFlightID: faFlightID})
	if err != nil {
		return application.ExternalRouteResponse{}, mapFlightAwareError(err)
	}
	return externalRouteResponse(response), nil
}

func (a *FlightAwareFetchAdapter) GetFlightPosition(ctx context.Context, faFlightID string) (application.ExternalPositionResponse, error) {
	response, err := a.client.GetFlightPosition(ctx, flightaware.FlightPositionRequest{FAFlightID: faFlightID})
	if err != nil {
		return application.ExternalPositionResponse{}, mapFlightAwareError(err)
	}
	return application.ExternalPositionResponse{
		FAFlightID: response.FAFlightID,
		Latitude:   response.Latitude,
		Longitude:  response.Longitude,
		Timestamp:  response.Timestamp,
	}, nil
}

func (a *FlightAwareFetchAdapter) GetFlightTrack(ctx context.Context, faFlightID string) (application.ExternalTrackResponse, error) {
	response, err := a.client.GetFlightTrack(ctx, flightaware.FlightTrackRequest{FAFlightID: faFlightID})
	if err != nil {
		return application.ExternalTrackResponse{}, mapFlightAwareError(err)
	}
	return externalTrackResponse(response), nil
}

func mapFlightAwareError(err error) error {
	switch {
	case errors.Is(err, flightaware.ErrFlightAwareRateLimited):
		return application.ErrUpstreamRateLimited
	case errors.Is(err, flightaware.ErrFlightAwareFetchDisabled):
		return application.ErrUpstreamFetchDisabled
	default:
		return err
	}
}
