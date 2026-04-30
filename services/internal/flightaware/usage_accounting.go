package flightaware

import (
	"context"
	"errors"
)

type UsageRecordPhase string

const (
	UsageRecordPhaseBefore UsageRecordPhase = "before"
	UsageRecordPhaseAfter  UsageRecordPhase = "after"
)

type UsageCallRecord struct {
	Endpoint            Endpoint
	Phase               UsageRecordPhase
	EstimatedResultSets int
	EstimatedCostUSD    float64
	Succeeded           bool
	ErrorCode           string
}

type UsageRecorder interface {
	RecordFlightAwareCall(context.Context, UsageCallRecord) error
}

type StaticUsageEstimator struct {
	ResultSetsByEndpoint map[Endpoint]int
	CostUSDByEndpoint    map[Endpoint]float64
}

func (e StaticUsageEstimator) Estimate(endpoint Endpoint) (int, float64) {
	return e.ResultSetsByEndpoint[endpoint], e.CostUSDByEndpoint[endpoint]
}

type UsageAccountingClient struct {
	upstream  Client
	recorder  UsageRecorder
	estimator StaticUsageEstimator
}

func NewUsageAccountingClient(upstream Client, recorder UsageRecorder, estimator StaticUsageEstimator) *UsageAccountingClient {
	return &UsageAccountingClient{
		upstream:  upstream,
		recorder:  recorder,
		estimator: estimator,
	}
}

func (c *UsageAccountingClient) SearchFlights(ctx context.Context, request SearchFlightsRequest) (SearchFlightsResponse, error) {
	return withUsageAccounting(ctx, c, EndpointSearch, func() (SearchFlightsResponse, error) {
		return c.upstream.SearchFlights(ctx, request)
	})
}

func (c *UsageAccountingClient) GetFlightSummary(ctx context.Context, request FlightSummaryRequest) (FlightSummary, error) {
	return withUsageAccounting(ctx, c, EndpointSummary, func() (FlightSummary, error) {
		return c.upstream.GetFlightSummary(ctx, request)
	})
}

func (c *UsageAccountingClient) GetFlightRoute(ctx context.Context, request FlightRouteRequest) (RouteResponse, error) {
	return withUsageAccounting(ctx, c, EndpointRoute, func() (RouteResponse, error) {
		return c.upstream.GetFlightRoute(ctx, request)
	})
}

func (c *UsageAccountingClient) GetFlightPosition(ctx context.Context, request FlightPositionRequest) (PositionResponse, error) {
	return withUsageAccounting(ctx, c, EndpointPosition, func() (PositionResponse, error) {
		return c.upstream.GetFlightPosition(ctx, request)
	})
}

func (c *UsageAccountingClient) GetFlightTrack(ctx context.Context, request FlightTrackRequest) (TrackResponse, error) {
	return withUsageAccounting(ctx, c, EndpointTrack, func() (TrackResponse, error) {
		return c.upstream.GetFlightTrack(ctx, request)
	})
}

func (c *UsageAccountingClient) GetSchedules(ctx context.Context, request SchedulesRequest) (SchedulesResponse, error) {
	return withUsageAccounting(ctx, c, EndpointSchedule, func() (SchedulesResponse, error) {
		return c.upstream.GetSchedules(ctx, request)
	})
}

func (c *UsageAccountingClient) GetAccountUsage(ctx context.Context) (UsageResponse, error) {
	return withUsageAccounting(ctx, c, EndpointUsage, func() (UsageResponse, error) {
		return c.upstream.GetAccountUsage(ctx)
	})
}

func withUsageAccounting[T any](ctx context.Context, client *UsageAccountingClient, endpoint Endpoint, call func() (T, error)) (T, error) {
	resultSets, costUSD := client.estimator.Estimate(endpoint)
	before := UsageCallRecord{
		Endpoint:            endpoint,
		Phase:               UsageRecordPhaseBefore,
		EstimatedResultSets: resultSets,
		EstimatedCostUSD:    costUSD,
	}
	if err := client.recorder.RecordFlightAwareCall(ctx, before); err != nil {
		var zero T
		return zero, err
	}

	// The before-call reservation is authoritative for budget enforcement; the
	// upstream request is skipped if the recorder rejects the estimated cost.
	result, upstreamErr := call()
	after := before
	after.Phase = UsageRecordPhaseAfter
	after.Succeeded = upstreamErr == nil
	if upstreamErr != nil {
		after.ErrorCode = errorCode(upstreamErr)
	}
	if err := client.recorder.RecordFlightAwareCall(ctx, after); err != nil {
		if upstreamErr != nil {
			return result, errors.Join(upstreamErr, err)
		}
		return result, err
	}

	return result, upstreamErr
}
