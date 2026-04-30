package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"airpath/services/internal/application"
)

// These tests keep the Lambda entrypoint under test while replacing only the
// dispatcher use-case boundary.
func TestHandleDispatcherEventReportsNoopState(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("DISPATCHER_MODE", "noop")
	t.Setenv("NOOP_FETCH_ENABLED", "true")
	t.Setenv("FETCH_TASK_QUEUE_URL", "https://sqs.ap-northeast-1.amazonaws.com/123456789012/airpath-dev-fetch-task")

	body, err := handleDispatcherEvent(context.Background(), json.RawMessage(`{"source":"aws.events"}`))
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
	ctx := context.WithValue(context.Background(), contextKey("dispatcher-test"), "invocation")
	newPollingDispatcher = func(ctx context.Context, _ runtimeDependencies) (pollingDispatcher, error) {
		if ctx.Value(contextKey("dispatcher-test")) != "invocation" {
			t.Fatalf("builder context value = %v, want invocation context", ctx.Value(contextKey("dispatcher-test")))
		}
		return pollingDispatcherFunc(func(ctx context.Context, _ application.DispatchPollInput) (application.DispatchPollResult, error) {
			if ctx.Value(contextKey("dispatcher-test")) != "invocation" {
				t.Fatalf("dispatch context value = %v, want invocation context", ctx.Value(contextKey("dispatcher-test")))
			}
			called = true
			return application.DispatchPollResult{DueFlights: 1, EnqueuedTasks: 2}, nil
		}), nil
	}

	body, err := handleDispatcherEvent(ctx, json.RawMessage(`{"source":"aws.events"}`))
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

func TestHandleDispatcherEventRequiresQueueURLInAWSRuntime(t *testing.T) {
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "airpath-dev-dispatcher")
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("DISPATCHER_MODE", "active")
	t.Setenv("NOOP_FETCH_ENABLED", "false")
	t.Setenv("FETCH_TASK_QUEUE_URL", "   ")

	original := newPollingDispatcher
	t.Cleanup(func() { newPollingDispatcher = original })
	newPollingDispatcher = func(context.Context, runtimeDependencies) (pollingDispatcher, error) {
		t.Fatal("polling dispatcher should not be built without FETCH_TASK_QUEUE_URL in AWS runtime")
		return nil, nil
	}

	body, err := handleDispatcherEvent(context.Background(), json.RawMessage(`{"source":"aws.events"}`))
	if !errors.Is(err, errFetchTaskQueueNotConfigured) {
		t.Fatalf("handleDispatcherEvent() error = %v, want errFetchTaskQueueNotConfigured", err)
	}
	if body.QueueConfigured {
		t.Fatal("QueueConfigured = true, want false")
	}
	if body.EnqueueAttempted {
		t.Fatal("EnqueueAttempted = true, want false")
	}
}

type pollingDispatcherFunc func(context.Context, application.DispatchPollInput) (application.DispatchPollResult, error)

func (f pollingDispatcherFunc) Dispatch(ctx context.Context, input application.DispatchPollInput) (application.DispatchPollResult, error) {
	return f(ctx, input)
}

// contextKey is private to tests so context propagation checks cannot collide
// with production context keys.
type contextKey string
