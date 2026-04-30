package flightaware

import "context"

type MaxPagesClient struct {
	ClientDecorator
}

func NewMaxPagesClient(upstream Client) *MaxPagesClient {
	return &MaxPagesClient{ClientDecorator: NewClientDecorator(upstream)}
}

func (c *MaxPagesClient) SearchFlights(ctx context.Context, request SearchFlightsRequest) (SearchFlightsResponse, error) {
	maxPages, err := normalizeMaxPages(EndpointSearch, request.MaxPages)
	if err != nil {
		return SearchFlightsResponse{}, err
	}
	request.MaxPages = maxPages
	return c.ClientDecorator.SearchFlights(ctx, request)
}

func (c *MaxPagesClient) GetSchedules(ctx context.Context, request SchedulesRequest) (SchedulesResponse, error) {
	maxPages, err := normalizeMaxPages(EndpointSchedule, request.MaxPages)
	if err != nil {
		return SchedulesResponse{}, err
	}
	request.MaxPages = maxPages
	return c.ClientDecorator.GetSchedules(ctx, request)
}
