package main

import (
	"context"
	"encoding/json"
	"testing"

	"airpath/services/internal/application"
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
		t.Fatal("EnqueueAttempted = true, want false while noop mode is enabled")
	}
}

func TestHandleDispatcherEventRunsPollingDispatcherWhenNoopIsDisabled(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("DISPATCHER_MODE", "active")
	t.Setenv("NOOP_FETCH_ENABLED", "false")
	t.Setenv("FETCH_TASK_QUEUE_URL", "https://sqs.ap-northeast-1.amazonaws.com/123456789012/airpath-dev-fetch-task")

	original := newPollingDispatcher
	t.Cleanup(func() { newPollingDispatcher = original })
	called := false
	newPollingDispatcher = func(context.Context, runtimeDependencies) (pollingDispatcher, error) {
		return pollingDispatcherFunc(func(context.Context, application.DispatchPollInput) (application.DispatchPollResult, error) {
			called = true
			return application.DispatchPollResult{DueFlights: 1, EnqueuedTasks: 2}, nil
		}), nil
	}

	body, err := handleDispatcherEvent(json.RawMessage(`{"source":"aws.events"}`))
	if err != nil {
		t.Fatalf("handleDispatcherEvent() error = %v", err)
	}
	if !called {
		t.Fatal("polling dispatcher was not called")
	}
	if !body.EnqueueAttempted {
		t.Fatal("EnqueueAttempted = false, want true")
	}
}

type pollingDispatcherFunc func(context.Context, application.DispatchPollInput) (application.DispatchPollResult, error)

func (f pollingDispatcherFunc) Dispatch(ctx context.Context, input application.DispatchPollInput) (application.DispatchPollResult, error) {
	return f(ctx, input)
}
