package main

import (
	"encoding/json"
	"testing"
)

func TestHandleDispatcherEventReportsNoopState(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("DISPATCHER_MODE", "noop")
	t.Setenv("NOOP_FETCH_ENABLED", "true")
	t.Setenv("FETCH_TASK_QUEUE_URL", "https://sqs.ap-northeast-1.amazonaws.com/123456789012/airpath-dev-fetch-task")

	body, err := handleDispatcherEvent(json.RawMessage(`{"source":"aws.events"}`))
	if err != nil {
		t.Fatalf("handleDispatcherEvent() error = %v", err)
	}
	if body.Service != "dispatcher" {
		t.Fatalf("Service = %q, want dispatcher", body.Service)
	}
	if body.Environment != "dev" {
		t.Fatalf("Environment = %q, want dev", body.Environment)
	}
	if body.Mode != "noop" {
		t.Fatalf("Mode = %q, want noop", body.Mode)
	}
	if !body.NoopFetchEnabled {
		t.Fatal("NoopFetchEnabled = false, want true")
	}
	if body.EnqueueAttempted {
		t.Fatal("EnqueueAttempted = true, want false for F8 shell")
	}
}
