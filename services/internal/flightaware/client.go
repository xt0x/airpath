package flightaware

import "context"

type Client interface {
	SearchFlights(context.Context, SearchFlightsRequest) (SearchFlightsResponse, error)
	GetFlightSummary(context.Context, FlightSummaryRequest) (FlightSummary, error)
	GetFlightRoute(context.Context, FlightRouteRequest) (RouteResponse, error)
	GetFlightPosition(context.Context, FlightPositionRequest) (PositionResponse, error)
	GetFlightTrack(context.Context, FlightTrackRequest) (TrackResponse, error)
	GetSchedules(context.Context, SchedulesRequest) (SchedulesResponse, error)
	GetAccountUsage(context.Context) (UsageResponse, error)
}
