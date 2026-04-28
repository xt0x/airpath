package main

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestHandleFetchEventReportsProcessedSQSRecordsWithoutFetching(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FETCHER_MODE", "mock")
	t.Setenv("FLIGHTAWARE_API_KEY_SECRET_ARN", "arn:aws:secretsmanager:ap-northeast-1:123456789012:secret:flightaware")

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
	if body.ExternalFetchAttempted {
		t.Fatal("ExternalFetchAttempted = true, want false for F8 shell")
	}
}
