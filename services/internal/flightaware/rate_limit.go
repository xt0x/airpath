package flightaware

import (
	"context"
	"sync"
	"time"
)

type RateLimitState interface {
	IsStopped(Endpoint) bool
	Stop(Endpoint, string)
	MarkRateLimited(Endpoint, time.Time)
}

type MemoryRateLimitState struct {
	mu      sync.RWMutex
	stopped map[Endpoint]string
	resetAt map[Endpoint]time.Time
}

func NewMemoryRateLimitState() *MemoryRateLimitState {
	return &MemoryRateLimitState{
		stopped: map[Endpoint]string{},
		resetAt: map[Endpoint]time.Time{},
	}
}

func (s *MemoryRateLimitState) IsStopped(endpoint Endpoint) bool {
	return s.isStoppedAt(endpoint, time.Now())
}

func (s *MemoryRateLimitState) isStoppedAt(endpoint Endpoint, now time.Time) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	until, hasResetAt := s.resetAt[endpoint]
	if hasResetAt && now.After(until) {
		return false
	}
	_, ok := s.stopped[endpoint]
	return ok
}

func (s *MemoryRateLimitState) Stop(endpoint Endpoint, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped[endpoint] = reason
	delete(s.resetAt, endpoint)
}

func (s *MemoryRateLimitState) MarkRateLimited(endpoint Endpoint, until time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.markStopped(endpoint, until)
	// A single 429 means low-priority enrichment calls should also pause, while
	// summary and position calls can resume independently when their own state allows.
	for _, lowPriorityEndpoint := range lowPriorityEndpoints() {
		s.markStopped(lowPriorityEndpoint, until)
	}
}

func (s *MemoryRateLimitState) ResetAt(endpoint Endpoint) *time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	until, ok := s.resetAt[endpoint]
	if !ok {
		return nil
	}
	return &until
}

func (s *MemoryRateLimitState) markStopped(endpoint Endpoint, until time.Time) {
	s.stopped[endpoint] = until.Format(time.RFC3339)
	s.resetAt[endpoint] = until
}

func lowPriorityEndpoints() []Endpoint {
	return []Endpoint{EndpointRoute, EndpointTrack, EndpointSchedule}
}

type RateLimitedClient struct {
	upstream Client
	state    RateLimitState
	backoff  time.Duration
}

func NewRateLimitedClient(upstream Client, state RateLimitState, backoff time.Duration) *RateLimitedClient {
	return &RateLimitedClient{upstream: upstream, state: state, backoff: backoff}
}

func (c *RateLimitedClient) SearchFlights(ctx context.Context, request SearchFlightsRequest) (SearchFlightsResponse, error) {
	return withRateLimit(ctx, c, EndpointSearch, func() (SearchFlightsResponse, error) {
		return c.upstream.SearchFlights(ctx, request)
	})
}

func (c *RateLimitedClient) GetFlightSummary(ctx context.Context, request FlightSummaryRequest) (FlightSummary, error) {
	return withRateLimit(ctx, c, EndpointSummary, func() (FlightSummary, error) {
		return c.upstream.GetFlightSummary(ctx, request)
	})
}

func (c *RateLimitedClient) GetFlightRoute(ctx context.Context, request FlightRouteRequest) (RouteResponse, error) {
	return withRateLimit(ctx, c, EndpointRoute, func() (RouteResponse, error) {
		return c.upstream.GetFlightRoute(ctx, request)
	})
}

func (c *RateLimitedClient) GetFlightPosition(ctx context.Context, request FlightPositionRequest) (PositionResponse, error) {
	return withRateLimit(ctx, c, EndpointPosition, func() (PositionResponse, error) {
		return c.upstream.GetFlightPosition(ctx, request)
	})
}

func (c *RateLimitedClient) GetFlightTrack(ctx context.Context, request FlightTrackRequest) (TrackResponse, error) {
	return withRateLimit(ctx, c, EndpointTrack, func() (TrackResponse, error) {
		return c.upstream.GetFlightTrack(ctx, request)
	})
}

func (c *RateLimitedClient) GetSchedules(ctx context.Context, request SchedulesRequest) (SchedulesResponse, error) {
	return withRateLimit(ctx, c, EndpointSchedule, func() (SchedulesResponse, error) {
		return c.upstream.GetSchedules(ctx, request)
	})
}

func (c *RateLimitedClient) GetAccountUsage(ctx context.Context) (UsageResponse, error) {
	return c.upstream.GetAccountUsage(ctx)
}

func withRateLimit[T any](ctx context.Context, client *RateLimitedClient, endpoint Endpoint, call func() (T, error)) (T, error) {
	if client.state.IsStopped(endpoint) {
		var zero T
		return zero, &ClientError{Code: "fetch_disabled", Endpoint: endpoint, Message: "FlightAware fetch is disabled", Err: ErrFlightAwareFetchDisabled}
	}

	result, err := call()
	if err != nil && errorCode(err) == "rate_limited" {
		client.state.MarkRateLimited(endpoint, time.Now().Add(client.backoff))
	}
	_ = ctx
	return result, err
}
