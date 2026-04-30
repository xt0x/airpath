package main

import (
	"context"
	"errors"
	"testing"

	"airpath/services/internal/application"
	"github.com/aws/aws-lambda-go/events"
)

// The entrypoint tests replace only the fetch processor seam so SQS decoding,
// partial batch failure reporting, and runtime flags are still exercised here.
func TestHandleFetchEventSkipsValidSQSRecordsWithoutRetryingWhenFetchingIsDisabled(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("AIRPATH_RUNTIME_BACKEND", "memory")
	t.Setenv("FETCHER_MODE", "mock")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "false")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "false")

	body, err := handleFetchEvent(context.Background(), events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "task-1", Body: `{"schemaVersion":1,"taskId":"task-1","taskType":"summary","flightId":"ANA110","requestedAt":"2026-04-29T00:00:00Z","reason":"search_result_seed","idempotencyKey":"task-1"}`},
			{MessageId: "task-2", Body: `{"schemaVersion":1,"taskId":"task-2","taskType":"position","flightId":"iflg_1","faFlightId":"fa_1","requestedAt":"2026-04-29T00:00:00Z","reason":"low_frequency_poll","idempotencyKey":"task-2"}`},
		},
	})
	if err != nil {
		t.Fatalf("handleFetchEvent(disabled records) error = %v", err)
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
		t.Fatal("ExternalFetchAttempted = true, want false while external fetch is disabled")
	}
	if len(body.BatchItemFailures) != 0 {
		t.Fatalf("BatchItemFailures = %#v, want none for policy-skipped valid records", body.BatchItemFailures)
	}
}

func TestHandleFetchEventReportsDisabledStateWithoutRetryingEmptyBatch(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FETCHER_MODE", "mock")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "false")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "false")

	body, err := handleFetchEvent(context.Background(), events.SQSEvent{})
	if err != nil {
		t.Fatalf("handleFetchEvent(empty disabled batch) error = %v", err)
	}
	if body.RecordsReceived != 0 {
		t.Fatalf("RecordsReceived = %d, want 0", body.RecordsReceived)
	}
	if body.ExternalFetchAllowed {
		t.Fatal("ExternalFetchAllowed = true, want false")
	}
}

func TestHandleFetchEventReportsFlightAwareCredentialReadyFromEnvironmentKey(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FETCHER_MODE", "mock")
	t.Setenv("FLIGHTAWARE_API_KEY", "local-api-key")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "false")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "false")

	body, err := handleFetchEvent(context.Background(), events.SQSEvent{})
	if err != nil {
		t.Fatalf("handleFetchEvent(empty disabled batch) error = %v", err)
	}
	if !body.FlightAwareSecretReady {
		t.Fatal("FlightAwareSecretReady = false, want true for FLIGHTAWARE_API_KEY source")
	}
}

func TestHandleFetchEventRequiresBothRuntimeFlagsToAllowExternalFetches(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FETCHER_MODE", "real-opt-in")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "true")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "true")

	body, err := handleFetchEvent(context.Background(), events.SQSEvent{})
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
		t.Fatal("ExternalFetchAttempted = true, want false without task records")
	}
}

func TestHandleFetchEventProcessesFetchTasksWhenExternalFetchIsAllowed(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FETCHER_MODE", "real-opt-in")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "true")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "true")

	original := newFetchProcessor
	t.Cleanup(func() { newFetchProcessor = original })
	processed := 0
	ctx := context.WithValue(context.Background(), fetcherContextKey("fetcher-test"), "invocation")
	newFetchProcessor = func(ctx context.Context, _ runtimeDependencies) (fetchProcessor, error) {
		if ctx.Value(fetcherContextKey("fetcher-test")) != "invocation" {
			t.Fatalf("builder context value = %v, want invocation context", ctx.Value(fetcherContextKey("fetcher-test")))
		}
		return fetchProcessorFunc(func(ctx context.Context, task application.FetchTask, input application.ProcessFetchInput) (application.ProcessFetchResult, error) {
			if ctx.Value(fetcherContextKey("fetcher-test")) != "invocation" {
				t.Fatalf("process context value = %v, want invocation context", ctx.Value(fetcherContextKey("fetcher-test")))
			}
			processed++
			if task.TaskID != "task-1" {
				t.Fatalf("TaskID = %q, want task-1", task.TaskID)
			}
			if input.WorkerID == "" || input.WorkerID == "fetcher" {
				t.Fatalf("WorkerID = %q, want record-specific fetcher worker ID", input.WorkerID)
			}
			return application.ProcessFetchResult{ExternalFetchAttempted: true, UpdatedPositionCount: 1}, nil
		}), nil
	}

	body, err := handleFetchEvent(ctx, events.SQSEvent{
		Records: []events.SQSMessage{{
			MessageId: "task-1",
			Body:      `{"schemaVersion":1,"taskId":"task-1","taskType":"position","flightId":"iflg_1","faFlightId":"fa_1","requestedAt":"2026-04-29T00:00:00Z","reason":"low_frequency_poll","idempotencyKey":"task-1"}`,
		}},
	})
	if err != nil {
		t.Fatalf("handleFetchEvent() error = %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	if !body.ExternalFetchAttempted {
		t.Fatal("ExternalFetchAttempted = false, want true")
	}
}

func TestHandleFetchEventUsesRecordSpecificWorkerIDs(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FETCHER_MODE", "real-opt-in")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "true")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "true")

	original := newFetchProcessor
	t.Cleanup(func() { newFetchProcessor = original })
	workerIDs := []string{}
	newFetchProcessor = func(context.Context, runtimeDependencies) (fetchProcessor, error) {
		return fetchProcessorFunc(func(_ context.Context, _ application.FetchTask, input application.ProcessFetchInput) (application.ProcessFetchResult, error) {
			workerIDs = append(workerIDs, input.WorkerID)
			return application.ProcessFetchResult{ExternalFetchAttempted: true}, nil
		}), nil
	}

	_, err := handleFetchEvent(context.Background(), events.SQSEvent{
		Records: []events.SQSMessage{
			{
				MessageId: "message-one",
				Body:      `{"schemaVersion":1,"taskId":"task-one","taskType":"position","flightId":"iflg_1","faFlightId":"fa_1","requestedAt":"2026-04-29T00:00:00Z","reason":"low_frequency_poll","idempotencyKey":"task-one"}`,
			},
			{
				MessageId: "message-two",
				Body:      `{"schemaVersion":1,"taskId":"task-two","taskType":"position","flightId":"iflg_1","faFlightId":"fa_1","requestedAt":"2026-04-29T00:00:00Z","reason":"low_frequency_poll","idempotencyKey":"task-two"}`,
			},
		},
	})
	if err != nil {
		t.Fatalf("handleFetchEvent() error = %v", err)
	}
	if len(workerIDs) != 2 {
		t.Fatalf("workerIDs = %#v, want two processed records", workerIDs)
	}
	if workerIDs[0] == "" || workerIDs[1] == "" || workerIDs[0] == workerIDs[1] {
		t.Fatalf("workerIDs = %#v, want distinct non-empty worker IDs", workerIDs)
	}
}

func TestHandleFetchEventReturnsPartialBatchFailuresWithoutRetryingSuccessfulRecords(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FETCHER_MODE", "real-opt-in")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "true")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "true")

	original := newFetchProcessor
	t.Cleanup(func() { newFetchProcessor = original })
	processed := []string{}
	newFetchProcessor = func(context.Context, runtimeDependencies) (fetchProcessor, error) {
		return fetchProcessorFunc(func(_ context.Context, task application.FetchTask, _ application.ProcessFetchInput) (application.ProcessFetchResult, error) {
			processed = append(processed, task.TaskID)
			if task.TaskID == "task-fails" {
				return application.ProcessFetchResult{ExternalFetchAttempted: true}, errors.New("upstream unavailable")
			}
			return application.ProcessFetchResult{ExternalFetchAttempted: true, UpdatedPositionCount: 1}, nil
		}), nil
	}

	body, err := handleFetchEvent(context.Background(), events.SQSEvent{
		Records: []events.SQSMessage{
			{
				MessageId: "message-ok",
				Body:      `{"schemaVersion":1,"taskId":"task-ok","taskType":"position","flightId":"iflg_1","faFlightId":"fa_1","requestedAt":"2026-04-29T00:00:00Z","reason":"low_frequency_poll","idempotencyKey":"task-ok"}`,
			},
			{
				MessageId: "message-fails",
				Body:      `{"schemaVersion":1,"taskId":"task-fails","taskType":"position","flightId":"iflg_2","faFlightId":"fa_2","requestedAt":"2026-04-29T00:00:00Z","reason":"low_frequency_poll","idempotencyKey":"task-fails"}`,
			},
		},
	})
	if err != nil {
		t.Fatalf("handleFetchEvent() error = %v, want nil partial batch response", err)
	}
	if len(processed) != 2 || processed[0] != "task-ok" || processed[1] != "task-fails" {
		t.Fatalf("processed tasks = %#v, want both records attempted", processed)
	}
	if len(body.BatchItemFailures) != 1 || body.BatchItemFailures[0].ItemIdentifier != "message-fails" {
		t.Fatalf("batch failures = %#v, want only failed message", body.BatchItemFailures)
	}
	if !body.ExternalFetchAttempted {
		t.Fatal("ExternalFetchAttempted = false, want true after successful first record")
	}
}

type fetchProcessorFunc func(context.Context, application.FetchTask, application.ProcessFetchInput) (application.ProcessFetchResult, error)

func (f fetchProcessorFunc) Process(ctx context.Context, task application.FetchTask, input application.ProcessFetchInput) (application.ProcessFetchResult, error) {
	return f(ctx, task, input)
}

// fetcherContextKey avoids collisions with production context values while
// verifying that invocation context reaches the processor builder and process call.
type fetcherContextKey string
