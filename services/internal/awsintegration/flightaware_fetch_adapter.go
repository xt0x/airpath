package awsintegration

import (
	"context"
	"errors"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"
	"airpath/services/internal/flightaware"
)

type FlightAwareFetchClient interface {
	SearchFlights(context.Context, flightaware.SearchFlightsRequest) (flightaware.SearchFlightsResponse, error)
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

func (a *FlightAwareFetchAdapter) SearchFlights(ctx context.Context, ident string) (application.ExternalSearchFlightsResponse, error) {
	response, err := a.client.SearchFlights(ctx, flightaware.SearchFlightsRequest{Ident: ident})
	if err != nil {
		return application.ExternalSearchFlightsResponse{}, mapFlightAwareError(err)
	}
	flights := make([]application.ExternalFlightSummary, 0, len(response.Flights))
	for _, flight := range response.Flights {
		flights = append(flights, externalFlightSummary(flight))
	}
	return application.ExternalSearchFlightsResponse{Flights: flights}, nil
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
		FAFlightID:           response.FAFlightID,
		Latitude:             response.Latitude,
		Longitude:            response.Longitude,
		AltitudeHundredsFeet: response.AltitudeHundredsFeet,
		AltitudeChange:       altitudeChangePtr(response.AltitudeChange),
		GroundspeedKnots:     response.GroundspeedKnots,
		HeadingDegrees:       response.HeadingDegrees,
		Timestamp:            response.Timestamp,
		UpdateType:           positionUpdateTypePtr(response.UpdateType),
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

func externalFlightSummary(summary flightaware.FlightSummary) application.ExternalFlightSummary {
	return application.ExternalFlightSummary{
		FAFlightID:      summary.FAFlightID,
		Ident:           summary.Ident,
		IdentIATA:       summary.IdentIATA,
		AircraftType:    summary.AircraftType,
		Registration:    summary.Registration,
		Origin:          summary.Origin,
		Destination:     summary.Destination,
		ScheduledOut:    isoDateTimePtr(summary.ScheduledOut),
		EstimatedOut:    isoDateTimePtr(summary.EstimatedOut),
		ActualOut:       isoDateTimePtr(summary.ActualOut),
		ScheduledOff:    isoDateTimePtr(summary.ScheduledOff),
		EstimatedOff:    isoDateTimePtr(summary.EstimatedOff),
		ActualOff:       isoDateTimePtr(summary.ActualOff),
		ScheduledOn:     isoDateTimePtr(summary.ScheduledOn),
		EstimatedOn:     isoDateTimePtr(summary.EstimatedOn),
		ActualOn:        isoDateTimePtr(summary.ActualOn),
		ScheduledIn:     isoDateTimePtr(summary.ScheduledIn),
		EstimatedIn:     isoDateTimePtr(summary.EstimatedIn),
		ActualIn:        isoDateTimePtr(summary.ActualIn),
		Status:          summary.Status,
		ProgressPercent: summary.ProgressPercent,
		FiledEteSeconds: summary.FiledEteSeconds,
	}
}

func isoDateTimePtr(value *string) *domain.ISODateTimeString {
	if value == nil {
		return nil
	}
	converted := domain.ISODateTimeString(*value)
	return &converted
}

func externalRouteResponse(response flightaware.RouteResponse) application.ExternalRouteResponse {
	fixes := make([]application.ExternalRouteFix, 0, len(response.Fixes))
	for _, fix := range response.Fixes {
		fixes = append(fixes, application.ExternalRouteFix{
			Name:      fix.Name,
			Latitude:  fix.Latitude,
			Longitude: fix.Longitude,
		})
	}
	return application.ExternalRouteResponse{RouteText: response.RouteText, Fixes: fixes}
}

func externalTrackResponse(response flightaware.TrackResponse) application.ExternalTrackResponse {
	positions := make([]application.ExternalTrackPoint, 0, len(response.Positions))
	for _, position := range response.Positions {
		positions = append(positions, application.ExternalTrackPoint{
			Latitude:             position.Latitude,
			Longitude:            position.Longitude,
			AltitudeHundredsFeet: position.AltitudeHundredsFeet,
			AltitudeChange:       altitudeChangePtr(position.AltitudeChange),
			GroundspeedKnots:     position.GroundspeedKnots,
			HeadingDegrees:       position.HeadingDegrees,
			Timestamp:            position.Timestamp,
			UpdateType:           positionUpdateTypePtr(position.UpdateType),
		})
	}
	return application.ExternalTrackResponse{Positions: positions}
}

func altitudeChangePtr(value *string) *domain.AltitudeChange {
	if value == nil || *value == "" {
		return nil
	}
	converted := domain.AltitudeChange(*value)
	return &converted
}

func positionUpdateTypePtr(value *string) *domain.PositionUpdateType {
	if value == nil || *value == "" {
		return nil
	}
	converted := domain.PositionUpdateType(*value)
	return &converted
}
