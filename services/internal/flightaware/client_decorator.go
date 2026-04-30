package flightaware

import "context"

type ClientDecorator struct {
	upstream Client
}

func NewClientDecorator(upstream Client) ClientDecorator {
	return ClientDecorator{upstream: upstream}
}

func (d ClientDecorator) SearchFlights(ctx context.Context, request SearchFlightsRequest) (SearchFlightsResponse, error) {
	return d.upstream.SearchFlights(ctx, request)
}

func (d ClientDecorator) GetFlightSummary(ctx context.Context, request FlightSummaryRequest) (FlightSummary, error) {
	return d.upstream.GetFlightSummary(ctx, request)
}

func (d ClientDecorator) GetFlightRoute(ctx context.Context, request FlightRouteRequest) (RouteResponse, error) {
	return d.upstream.GetFlightRoute(ctx, request)
}

func (d ClientDecorator) GetFlightPosition(ctx context.Context, request FlightPositionRequest) (PositionResponse, error) {
	return d.upstream.GetFlightPosition(ctx, request)
}

func (d ClientDecorator) GetFlightTrack(ctx context.Context, request FlightTrackRequest) (TrackResponse, error) {
	return d.upstream.GetFlightTrack(ctx, request)
}

func (d ClientDecorator) GetSchedules(ctx context.Context, request SchedulesRequest) (SchedulesResponse, error) {
	return d.upstream.GetSchedules(ctx, request)
}

func (d ClientDecorator) GetAccountUsage(ctx context.Context) (UsageResponse, error) {
	return d.upstream.GetAccountUsage(ctx)
}
