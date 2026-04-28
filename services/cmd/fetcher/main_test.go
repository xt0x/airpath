package main

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestHandleFetchEventReportsProcessedSQSRecordsWithoutFetching(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FETCHER_MODE", "mock")
	t.Setenv("FLIGHTAWARE_API_KEY_SECRET_ARN", "arn:aws:secretsmanager:ap-northeast-1:123456789012:secret:flightaware")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "false")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "false")

	body, err := handleFetchEvent(events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "task-1", Body: `{"kind":"summary"}`},
			{MessageId: "task-2", Body: `{"kind":"position"}`},
		},
	})
	if err != nil {
		t.Fatalf("handleFetchEvent() error = %v", err)
	}
	if body.Service != "fetcher" {
		t.Fatalf("Service = %q, want fetcher", body.Service)
	}
	if body.Environment != "dev" {
		t.Fatalf("Environment = %q, want dev", body.Environment)
	}
	if body.Mode != "mock" {
		t.Fatalf("Mode = %q, want mock", body.Mode)
	}
	if body.RecordsReceived != 2 {
		t.Fatalf("RecordsReceived = %d, want 2", body.RecordsReceived)
	}
	if body.FlightAwareFetchEnabled {
		t.Fatal("FlightAwareFetchEnabled = true, want false")
	}
	if body.FlightAwareRealCallsEnabled {
		t.Fatal("FlightAwareRealCallsEnabled = true, want false")
	}
	if body.ExternalFetchAllowed {
		t.Fatal("ExternalFetchAllowed = true, want false")
	}
	if body.ExternalFetchAttempted {
		t.Fatal("ExternalFetchAttempted = true, want false for F8 shell")
	}
}

func TestHandleFetchEventRequiresBothRuntimeFlagsToAllowExternalFetches(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FETCHER_MODE", "real-opt-in")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "true")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "true")

	body, err := handleFetchEvent(events.SQSEvent{})
	if err != nil {
		t.Fatalf("handleFetchEvent() error = %v", err)
	}
	if !body.FlightAwareFetchEnabled {
		t.Fatal("FlightAwareFetchEnabled = false, want true")
	}
	if !body.FlightAwareRealCallsEnabled {
		t.Fatal("FlightAwareRealCallsEnabled = false, want true")
	}
	if !body.ExternalFetchAllowed {
		t.Fatal("ExternalFetchAllowed = false, want true")
	}
	if body.ExternalFetchAttempted {
		t.Fatal("ExternalFetchAttempted = true, want false until real worker integration is wired")
	}
}
