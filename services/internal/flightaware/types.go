package flightaware

import (
	"errors"
	"fmt"
)

type Endpoint string

const (
	EndpointSearch   Endpoint = "search"
	EndpointSummary  Endpoint = "summary"
	EndpointRoute    Endpoint = "route"
	EndpointPosition Endpoint = "position"
	EndpointTrack    Endpoint = "track"
	EndpointSchedule Endpoint = "schedules"
	EndpointUsage    Endpoint = "usage"
)

var (
	ErrFixtureNotFound          = errors.New("flightaware fixture not found")
	ErrMaxPagesExceeded         = errors.New("flightaware max_pages must be exactly 1")
	ErrFlightAwareRateLimited   = errors.New("flightaware rate limited")
	ErrFlightAwareFetchDisabled = errors.New("flightaware fetch disabled")
)

type ClientError struct {
	Code       string
	Endpoint   Endpoint
	StatusCode int
	Message    string
	Err        error
}

func (e *ClientError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Code)
}

func (e *ClientError) Unwrap() error {
	return e.Err
}

func errorCode(err error) string {
	var clientErr *ClientError
	if errors.As(err, &clientErr) && clientErr.Code != "" {
		return clientErr.Code
	}
	switch {
	case errors.Is(err, ErrFixtureNotFound):
		return "fixture_not_found"
	case errors.Is(err, ErrMaxPagesExceeded):
		return "max_pages_exceeded"
	case errors.Is(err, ErrFlightAwareRateLimited):
		return "rate_limited"
	case errors.Is(err, ErrFlightAwareFetchDisabled):
		return "fetch_disabled"
	default:
		return "unknown"
	}
}

func NewRateLimitedError(endpoint Endpoint, statusCode int, message string) error {
	return &ClientError{
		Code:       "rate_limited",
		Endpoint:   endpoint,
		StatusCode: statusCode,
		Message:    message,
		Err:        ErrFlightAwareRateLimited,
	}
}

func newFixtureNotFound(endpoint Endpoint, key string) error {
	return &ClientError{
		Code:     "fixture_not_found",
		Endpoint: endpoint,
		Message:  fmt.Sprintf("missing FlightAware fixture for %s: %s", endpoint, key),
		Err:      ErrFixtureNotFound,
	}
}

func maxPagesError(endpoint Endpoint, maxPages int) error {
	return &ClientError{
		Code:     "max_pages_exceeded",
		Endpoint: endpoint,
		Message:  fmt.Sprintf("%s max_pages=%d is outside the free-allowance boundary", endpoint, maxPages),
		Err:      ErrMaxPagesExceeded,
	}
}

type SearchFlightsRequest struct {
	Ident    string
	MaxPages int
}

type FlightSummaryRequest struct {
	FAFlightID string
}

type FlightRouteRequest struct {
	FAFlightID string
}

type FlightPositionRequest struct {
	FAFlightID string
}

type FlightTrackRequest struct {
	FAFlightID string
}

type SchedulesRequest struct {
	StartDate string
	EndDate   string
	MaxPages  int
}

type SearchFlightsResponse struct {
	Flights []FlightSummary
}

type SchedulesResponse struct {
	Scheduled []FlightSummary
}

type FlightSummary struct {
	FAFlightID  *string
	Ident       string
	Origin      string
	Destination string
}

type RouteResponse struct {
	RouteText *string
	Fixes     []RouteFix
}

type RouteFix struct {
	Name      string
	Latitude  *float64
	Longitude *float64
}

type PositionResponse struct {
	FAFlightID string
	Latitude   *float64
	Longitude  *float64
	Timestamp  string
}

type TrackResponse struct {
	Positions []TrackPoint
}

type TrackPoint struct {
	Latitude  float64
	Longitude float64
	Timestamp string
}

type UsageResponse struct {
	Currency    string
	MonthToDate UsageAmount
}

type UsageAmount struct {
	EstimatedCostUSD float64
	ResultSets       int
}

type ScheduleKey struct {
	StartDate string
	EndDate   string
}
