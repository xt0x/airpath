package flightaware

import "context"

type FixtureData struct {
	SearchResponses   map[string]SearchFlightsResponse
	SummaryResponses  map[string]FlightSummary
	RouteResponses    map[string]RouteResponse
	PositionResponses map[string]PositionResponse
	TrackResponses    map[string]TrackResponse
	ScheduleResponses map[ScheduleKey]SchedulesResponse
	UsageResponse     UsageResponse
}

type FixtureClient struct {
	data FixtureData
}

func NewFixtureClient(data FixtureData) *FixtureClient {
	return &FixtureClient{data: data}
}

func (c *FixtureClient) SearchFlights(_ context.Context, request SearchFlightsRequest) (SearchFlightsResponse, error) {
	if _, err := normalizeMaxPages(EndpointSearch, request.MaxPages); err != nil {
		return SearchFlightsResponse{}, err
	}
	response, ok := c.data.SearchResponses[request.Ident]
	if !ok {
		return SearchFlightsResponse{}, newFixtureNotFound(EndpointSearch, request.Ident)
	}
	return response, nil
}

func (c *FixtureClient) GetFlightSummary(_ context.Context, request FlightSummaryRequest) (FlightSummary, error) {
	response, ok := c.data.SummaryResponses[request.FAFlightID]
	if !ok {
		return FlightSummary{}, newFixtureNotFound(EndpointSummary, request.FAFlightID)
	}
	return response, nil
}

func (c *FixtureClient) GetFlightRoute(_ context.Context, request FlightRouteRequest) (RouteResponse, error) {
	response, ok := c.data.RouteResponses[request.FAFlightID]
	if !ok {
		return RouteResponse{}, newFixtureNotFound(EndpointRoute, request.FAFlightID)
	}
	return response, nil
}

func (c *FixtureClient) GetFlightPosition(_ context.Context, request FlightPositionRequest) (PositionResponse, error) {
	response, ok := c.data.PositionResponses[request.FAFlightID]
	if !ok {
		return PositionResponse{}, newFixtureNotFound(EndpointPosition, request.FAFlightID)
	}
	return response, nil
}

func (c *FixtureClient) GetFlightTrack(_ context.Context, request FlightTrackRequest) (TrackResponse, error) {
	response, ok := c.data.TrackResponses[request.FAFlightID]
	if !ok {
		return TrackResponse{}, newFixtureNotFound(EndpointTrack, request.FAFlightID)
	}
	return response, nil
}

func (c *FixtureClient) GetSchedules(_ context.Context, request SchedulesRequest) (SchedulesResponse, error) {
	if _, err := normalizeMaxPages(EndpointSchedule, request.MaxPages); err != nil {
		return SchedulesResponse{}, err
	}
	key := ScheduleKey{StartDate: request.StartDate, EndDate: request.EndDate}
	response, ok := c.data.ScheduleResponses[key]
	if !ok {
		return SchedulesResponse{}, newFixtureNotFound(EndpointSchedule, key.StartDate+".."+key.EndDate)
	}
	return response, nil
}

func (c *FixtureClient) GetAccountUsage(context.Context) (UsageResponse, error) {
	return c.data.UsageResponse, nil
}

func normalizeMaxPages(endpoint Endpoint, maxPages int) (int, error) {
	if maxPages == 0 {
		return DefaultMaxPages, nil
	}
	if maxPages != DefaultMaxPages {
		return 0, maxPagesError(endpoint, maxPages)
	}
	return maxPages, nil
}
