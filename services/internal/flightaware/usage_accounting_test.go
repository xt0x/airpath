package flightaware

import (
	"context"
	"errors"
	"testing"
)

func TestUsageAccountingRecordsBeforeAndAfterEndpointCalls(t *testing.T) {
	recorder := &memoryUsageRecorder{}
	client := NewUsageAccountingClient(
		NewFixtureClient(FixtureData{
			RouteResponses: map[string]RouteResponse{"flight-1": {RouteText: ptr("AAA BBB")}},
		}),
		recorder,
		StaticUsageEstimator{
			ResultSetsByEndpoint: map[Endpoint]int{EndpointRoute: 2},
			CostUSDByEndpoint:    map[Endpoint]float64{EndpointRoute: 0.02},
		},
	)

	_, err := client.GetFlightRoute(context.Background(), FlightRouteRequest{FAFlightID: "flight-1"})
	if err != nil {
		t.Fatalf("GetFlightRoute() error = %v", err)
	}

	if len(recorder.records) != 2 {
		t.Fatalf("record count = %d, want 2", len(recorder.records))
	}
	if recorder.records[0].Phase != UsageRecordPhaseBefore || recorder.records[1].Phase != UsageRecordPhaseAfter {
		t.Fatalf("phases = %#v", recorder.records)
	}
	if recorder.records[0].Endpoint != EndpointRoute || recorder.records[0].EstimatedResultSets != 2 || recorder.records[0].EstimatedCostUSD != 0.02 {
		t.Fatalf("before record = %#v", recorder.records[0])
	}
	if !recorder.records[1].Succeeded || recorder.records[1].ErrorCode != "" {
		t.Fatalf("after record = %#v", recorder.records[1])
	}
}

func TestUsageAccountingRecordsFailureWithoutHidingUpstreamError(t *testing.T) {
	recorder := &memoryUsageRecorder{}
	client := NewUsageAccountingClient(NewFixtureClient(FixtureData{}), recorder, StaticUsageEstimator{})

	_, err := client.GetFlightPosition(context.Background(), FlightPositionRequest{FAFlightID: "missing"})
	if !errors.Is(err, ErrFixtureNotFound) {
		t.Fatalf("GetFlightPosition() error = %v, want ErrFixtureNotFound", err)
	}
	if len(recorder.records) != 2 || recorder.records[1].Succeeded {
		t.Fatalf("records = %#v, want failed after record", recorder.records)
	}
	if recorder.records[1].ErrorCode != "fixture_not_found" {
		t.Fatalf("error code = %q, want fixture_not_found", recorder.records[1].ErrorCode)
	}
}

func TestUsageAccountingDoesNotCallUpstreamWhenBeforeRecordFails(t *testing.T) {
	recorder := &memoryUsageRecorder{err: errors.New("recorder unavailable")}
	upstream := &countingClient{}
	client := NewUsageAccountingClient(upstream, recorder, StaticUsageEstimator{})

	_, err := client.GetAccountUsage(context.Background())
	if err == nil {
		t.Fatal("GetAccountUsage() error = nil, want recorder error")
	}
	if upstream.usageCalls != 0 {
		t.Fatalf("upstream usage calls = %d, want 0", upstream.usageCalls)
	}
}

type memoryUsageRecorder struct {
	records []UsageCallRecord
	err     error
}

func (r *memoryUsageRecorder) RecordFlightAwareCall(_ context.Context, record UsageCallRecord) error {
	if r.err != nil {
		return r.err
	}
	r.records = append(r.records, record)
	return nil
}

type countingClient struct {
	Client
	usageCalls int
}

func (c *countingClient) GetAccountUsage(context.Context) (UsageResponse, error) {
	c.usageCalls++
	return UsageResponse{}, nil
}
