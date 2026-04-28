package flightaware

import "context"

type MaxPagesClient struct {
	upstream Client
}

func NewMaxPagesClient(upstream Client) *MaxPagesClient {
	return &MaxPagesClient{upstream: upstream}
}

func (c *MaxPagesClient) SearchFlights(ctx context.Context, request SearchFlightsRequest) (SearchFlightsResponse, error) {
	maxPages, err := normalizeMaxPages(EndpointSearch, request.MaxPages)
	if err != nil {
		return SearchFlightsResponse{}, err
	}
	request.MaxPages = maxPages
	return c.upstream.SearchFlights(ctx, request)
}

func (c *MaxPagesClient) GetFlightSummary(ctx context.Context, request FlightSummaryRequest) (FlightSummary, error) {
	return c.upstream.GetFlightSummary(ctx, request)
}

func (c *MaxPagesClient) GetFlightRoute(ctx context.Context, request FlightRouteRequest) (RouteResponse, error) {
	return c.upstream.GetFlightRoute(ctx, request)
}

func (c *MaxPagesClient) GetFlightPosition(ctx context.Context, request FlightPositionRequest) (PositionResponse, error) {
	return c.upstream.GetFlightPosition(ctx, request)
}

func (c *MaxPagesClient) GetFlightTrack(ctx context.Context, request FlightTrackRequest) (TrackResponse, error) {
	return c.upstream.GetFlightTrack(ctx, request)
}

func (c *MaxPagesClient) GetSchedules(ctx context.Context, request SchedulesRequest) (SchedulesResponse, error) {
	maxPages, err := normalizeMaxPages(EndpointSchedule, request.MaxPages)
	if err != nil {
		return SchedulesResponse{}, err
	}
	request.MaxPages = maxPages
	return c.upstream.GetSchedules(ctx, request)
}

func (c *MaxPagesClient) GetAccountUsage(ctx context.Context) (UsageResponse, error) {
	return c.upstream.GetAccountUsage(ctx)
}
