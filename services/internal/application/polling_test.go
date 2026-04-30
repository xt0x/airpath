package application

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"airpath/services/internal/domain"
)

func TestPollSchedulePolicyComputesConservativeNextPolls(t *testing.T) {
	now := mustTime(t, "2026-04-29T00:00:00Z")
	policy := DefaultPollSchedulePolicy()
	flight := testFlight("iflg_poll_1", "ANA110")

	scheduled := policy.Compute(flight, now, PollActivitySignal{ActiveViewer: true})

	assertTimePtr(t, scheduled.NextPositionPollAt, "2026-04-29T00:15:00Z")
	assertTimePtr(t, scheduled.NextTrackPollAt, "2026-04-29T00:45:00Z")
	assertTimePtr(t, scheduled.NextRoutePollAt, "2026-04-29T06:00:00Z")
	if scheduled.IdleSince != nil {
		t.Fatalf("IdleSince = %v, want nil for active flight", *scheduled.IdleSince)
	}

	routeKey := "routes/iflg_poll_1.json"
	trackKey := "tracks/iflg_poll_1.json"
	flight.PlannedRouteS3Key = &routeKey
	flight.ActualTrackS3Key = &trackKey
	scheduled = policy.Compute(flight, now, PollActivitySignal{ActiveViewer: true})

	if scheduled.NextRoutePollAt != nil {
		t.Fatalf("NextRoutePollAt = %v, want nil after cached route without explicit route window", *scheduled.NextRoutePollAt)
	}
	if scheduled.NextTrackPollAt != nil {
		t.Fatalf("NextTrackPollAt = %v, want nil after cached track without explicit track window", *scheduled.NextTrackPollAt)
	}

	scheduled = policy.Compute(flight, now, PollActivitySignal{ActiveViewer: true, AllowTrackWindow: true})
	assertTimePtr(t, scheduled.NextTrackPollAt, "2026-04-29T00:45:00Z")
}

func TestDispatcherSelectsDueFlightsOnlyWhenBudgetAllows(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	due := testFlight("iflg_due", "ANA110")
	due.NextPositionPollAt = ptr("2026-04-28T23:59:00Z")
	future := testFlight("iflg_future", "ANA111")
	future.NextPositionPollAt = ptr("2026-04-29T00:30:00Z")

	flights := &memoryPollFlightStore{flights: map[domain.FlightID]domain.Flight{
		due.FlightID:    due,
		future.FlightID: future,
	}}
	queue := &memoryFetchTaskQueue{}
	guard := &memoryUsageGuard{fetchingEnabled: false}
	dispatcher := NewPollingDispatcher(PollingDispatcherConfig{
		Flights:    flights,
		FetchTasks: queue,
		UsageGuard: guard,
	})

	stopped, err := dispatcher.Dispatch(ctx, DispatchPollInput{Now: now, Limit: 10})
	if err != nil {
		t.Fatalf("Dispatch(disabled) error = %v", err)
	}
	if !stopped.Stopped || len(queue.tasks) != 0 {
		t.Fatalf("disabled dispatch = %#v tasks=%#v, want stopped with no tasks", stopped, queue.tasks)
	}

	guard.fetchingEnabled = true
	result, err := dispatcher.Dispatch(ctx, DispatchPollInput{Now: now, Limit: 10})
	if err != nil {
		t.Fatalf("Dispatch(enabled) error = %v", err)
	}
	if result.Stopped || result.DueFlights != 1 || result.EnqueuedTasks != 1 {
		t.Fatalf("dispatch result = %#v, want one due task", result)
	}
	if len(queue.tasks) != 1 || queue.tasks[0].FlightID != due.FlightID || queue.tasks[0].TaskType != FetchTaskPosition {
		t.Fatalf("queued tasks = %#v, want due position task only", queue.tasks)
	}
}

func TestDispatcherEnqueuesDueTasksBeforeRefreshingActivitySchedule(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	due := testFlight("iflg_due_activity", "ANA110")
	due.NextPositionPollAt = ptr("2026-04-28T23:59:00Z")
	flights := &memoryPollFlightStore{flights: map[domain.FlightID]domain.Flight{
		due.FlightID: due,
	}}
	queue := &memoryFetchTaskQueue{}
	dispatcher := NewPollingDispatcher(PollingDispatcherConfig{
		Flights:    flights,
		Activities: memoryPollActivityStore{signal: PollActivitySignal{ActiveViewer: true}},
		FetchTasks: queue,
		UsageGuard: &memoryUsageGuard{fetchingEnabled: true},
	})

	result, err := dispatcher.Dispatch(ctx, DispatchPollInput{Now: now, Limit: 10})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if result.EnqueuedTasks != 1 || len(queue.tasks) != 1 {
		t.Fatalf("dispatch result = %#v tasks=%#v, want due task enqueued before schedule refresh", result, queue.tasks)
	}
	updated := flights.flights[due.FlightID]
	assertTimePtr(t, updated.NextPositionPollAt, "2026-04-29T00:15:00Z")
}

func TestDispatcherDoesNotAdvanceDueScheduleWhenEnqueueFails(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	due := testFlight("iflg_due_enqueue_fails", "ANA110")
	due.NextPositionPollAt = ptr("2026-04-28T23:59:00Z")
	flights := &memoryPollFlightStore{flights: map[domain.FlightID]domain.Flight{
		due.FlightID: due,
	}}
	queue := &memoryFetchTaskQueue{enqueueErr: errQueueUnavailable()}
	dispatcher := NewPollingDispatcher(PollingDispatcherConfig{
		Flights:    flights,
		Activities: memoryPollActivityStore{signal: PollActivitySignal{ActiveViewer: true}},
		FetchTasks: queue,
		UsageGuard: &memoryUsageGuard{fetchingEnabled: true},
	})

	_, err := dispatcher.Dispatch(ctx, DispatchPollInput{Now: now, Limit: 10})
	if err == nil {
		t.Fatal("Dispatch() error = nil, want enqueue failure")
	}
	updated := flights.flights[due.FlightID]
	assertTimePtr(t, updated.NextPositionPollAt, "2026-04-28T23:59:00Z")
}

func TestDispatcherDoesNotAdvanceDueScheduleWhenPolicyBlocksDueTask(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	due := testFlight("iflg_due_policy_blocks", "ANA110")
	due.NextRoutePollAt = ptr("2026-04-28T23:59:00Z")
	flights := &memoryPollFlightStore{flights: map[domain.FlightID]domain.Flight{
		due.FlightID: due,
	}}
	dispatcher := NewPollingDispatcher(PollingDispatcherConfig{
		Flights:    flights,
		Activities: memoryPollActivityStore{signal: PollActivitySignal{ActiveViewer: true}},
		FetchTasks: &memoryFetchTaskQueue{},
		UsageGuard: &memoryUsageGuard{fetchingEnabled: true},
		FetchPolicy: NewRuntimeFetchPolicy(RuntimeFetchConfig{
			ExternalFetchEnabled:   true,
			RouteFetchEnabled:      false,
			TrackFetchEnabled:      true,
			BackgroundFetchEnabled: true,
		}),
	})

	result, err := dispatcher.Dispatch(ctx, DispatchPollInput{Now: now, Limit: 10})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if result.EnqueuedTasks != 0 {
		t.Fatalf("EnqueuedTasks = %d, want none while route fetch is disabled", result.EnqueuedTasks)
	}
	updated := flights.flights[due.FlightID]
	assertTimePtr(t, updated.NextRoutePollAt, "2026-04-28T23:59:00Z")
}

func TestDispatcherAdvancesAllowedDueScheduleWhenAnotherTaskIsPolicyBlocked(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	due := testFlight("iflg_due_policy_blocks_route_only", "ANA110")
	due.NextPositionPollAt = ptr("2026-04-28T23:58:00Z")
	due.NextRoutePollAt = ptr("2026-04-28T23:59:00Z")
	flights := &memoryPollFlightStore{flights: map[domain.FlightID]domain.Flight{
		due.FlightID: due,
	}}
	queue := &memoryFetchTaskQueue{}
	dispatcher := NewPollingDispatcher(PollingDispatcherConfig{
		Flights:    flights,
		Activities: memoryPollActivityStore{signal: PollActivitySignal{ActiveViewer: true}},
		FetchTasks: queue,
		UsageGuard: &memoryUsageGuard{fetchingEnabled: true},
		FetchPolicy: NewRuntimeFetchPolicy(RuntimeFetchConfig{
			ExternalFetchEnabled:   true,
			RouteFetchEnabled:      false,
			TrackFetchEnabled:      true,
			BackgroundFetchEnabled: true,
		}),
	})

	result, err := dispatcher.Dispatch(ctx, DispatchPollInput{Now: now, Limit: 10})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if result.EnqueuedTasks != 1 || len(queue.tasks) != 1 {
		t.Fatalf("dispatch result = %#v tasks=%#v, want one allowed position task", result, queue.tasks)
	}
	if queue.tasks[0].TaskType != FetchTaskPosition {
		t.Fatalf("queued task type = %q, want position", queue.tasks[0].TaskType)
	}
	updated := flights.flights[due.FlightID]
	assertTimePtr(t, updated.NextPositionPollAt, "2026-04-29T00:15:00Z")
	assertTimePtr(t, updated.NextRoutePollAt, "2026-04-28T23:59:00Z")
}

func TestDispatcherUsesDueTimeForPollTaskIdempotencyAcrossTicks(t *testing.T) {
	ctx := context.Background()
	firstTick := mustTime(t, "2026-04-29T00:00:00Z")
	secondTick := mustTime(t, "2026-04-29T00:01:00Z")
	due := testFlight("iflg_due_stable_key", "ANA110")
	due.NextPositionPollAt = ptr("2026-04-28T23:59:00Z")
	flights := &memoryPollFlightStore{flights: map[domain.FlightID]domain.Flight{
		due.FlightID: due,
	}}
	queue := &memoryFetchTaskQueue{}
	dispatcher := NewPollingDispatcher(PollingDispatcherConfig{
		Flights:    flights,
		FetchTasks: queue,
		UsageGuard: &memoryUsageGuard{fetchingEnabled: true},
	})

	first, err := dispatcher.Dispatch(ctx, DispatchPollInput{Now: firstTick, Limit: 10})
	if err != nil {
		t.Fatalf("first Dispatch() error = %v", err)
	}
	second, err := dispatcher.Dispatch(ctx, DispatchPollInput{Now: secondTick, Limit: 10})
	if err != nil {
		t.Fatalf("second Dispatch() error = %v", err)
	}

	if first.EnqueuedTasks != 1 || second.EnqueuedTasks != 0 || len(queue.tasks) != 1 {
		t.Fatalf("dispatch results first=%#v second=%#v tasks=%#v, want one stable enqueue", first, second, queue.tasks)
	}
	wantKey := "poll:iflg_due_stable_key:position:2026-04-28T23:59:00Z"
	if queue.tasks[0].IdempotencyKey != wantKey || queue.tasks[0].TaskID != wantKey {
		t.Fatalf("task key/id = %q/%q, want %q", queue.tasks[0].IdempotencyKey, queue.tasks[0].TaskID, wantKey)
	}
}

func TestDispatcherDoesNotEnqueueIdleStoppedFlights(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:20:00Z")
	due := testFlight("iflg_idle_due", "ANA110")
	due.IdleSince = ptr("2026-04-29T00:00:00Z")
	due.NextPositionPollAt = ptr("2026-04-29T00:15:00Z")
	flights := &memoryPollFlightStore{flights: map[domain.FlightID]domain.Flight{
		due.FlightID: due,
	}}
	queue := &memoryFetchTaskQueue{}
	dispatcher := NewPollingDispatcher(PollingDispatcherConfig{
		Flights:    flights,
		Activities: memoryPollActivityStore{signal: PollActivitySignal{}},
		FetchTasks: queue,
		UsageGuard: &memoryUsageGuard{fetchingEnabled: true},
	})

	result, err := dispatcher.Dispatch(ctx, DispatchPollInput{Now: now, Limit: 10})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}

	if result.EnqueuedTasks != 0 || len(queue.tasks) != 0 {
		t.Fatalf("dispatch result = %#v tasks=%#v, want no idle tasks", result, queue.tasks)
	}
	updated := flights.flights[due.FlightID]
	if updated.PollState == nil || *updated.PollState != domain.FlightPollStateCompleted {
		t.Fatalf("poll state = %v, want completed", updated.PollState)
	}
}

func TestFetchProcessorUsesFlightLeaseToPreventDuplicateFlightAwareCalls(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_lease", "ANA110")
	flight.FetchLeaseUntil = ptr("2026-04-29T00:05:00Z")
	flight.FetchOwner = ptr("other-worker")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{
		position: ExternalPositionResponse{
			FAFlightID: "fa_1",
			Latitude:   ptrFloat64(45.1),
			Longitude:  ptrFloat64(160.2),
			Timestamp:  "2026-04-29T00:01:00Z",
		},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	result, err := processor.Process(ctx, FetchTask{
		SchemaVersion: 1,
		TaskID:        "task-1",
		TaskType:      FetchTaskPosition,
		FlightID:      flight.FlightID,
		FAFlightID:    flight.FAFlightID,
		Reason:        FetchReasonLowFrequencyPoll,
		RequestedAt:   "2026-04-29T00:00:00Z",
	}, ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process(leased) error = %v", err)
	}
	if !result.Skipped || result.SkipReason != "lease_held" || client.positionCalls != 0 {
		t.Fatalf("leased result = %#v position calls=%d, want skipped without call", result, client.positionCalls)
	}
}

func TestFetchProcessorInvalidTaskDoesNotReportExternalFetchAttempt(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_invalid_task", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	result, err := processor.Process(ctx, FetchTask{
		SchemaVersion:  1,
		TaskID:         "task-invalid",
		TaskType:       FetchTaskType("bogus"),
		FlightID:       flight.FlightID,
		FAFlightID:     flight.FAFlightID,
		Reason:         FetchReasonLowFrequencyPoll,
		RequestedAt:    "2026-04-29T00:00:00Z",
		IdempotencyKey: "task-invalid",
	}, ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Process(invalid task) error = %v, want ErrValidation", err)
	}
	if result.ExternalFetchAttempted {
		t.Fatalf("ExternalFetchAttempted = true, want false for invalid task before FlightAware call")
	}
	if client.positionCalls != 0 || client.routeCalls != 0 || client.trackCalls != 0 || client.searchCalls != 0 {
		t.Fatalf("client calls search=%d position=%d route=%d track=%d, want none", client.searchCalls, client.positionCalls, client.routeCalls, client.trackCalls)
	}
}

func TestFetchProcessorInvalidReasonDoesNotCallFlightAware(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	store := newMemoryFetchFlightStore()
	client := &countingFlightAwareClient{}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	result, err := processor.Process(ctx, FetchTask{
		SchemaVersion:  1,
		TaskID:         "task-invalid-reason",
		TaskType:       FetchTaskSummary,
		FlightID:       "ANA110",
		Reason:         FetchReason("bogus"),
		RequestedAt:    "2026-04-29T00:00:00Z",
		IdempotencyKey: "task-invalid-reason",
	}, ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Process(invalid reason) error = %v, want ErrValidation", err)
	}
	if result.ExternalFetchAttempted {
		t.Fatalf("ExternalFetchAttempted = true, want false for invalid reason before FlightAware call")
	}
	if client.searchCalls != 0 || client.positionCalls != 0 || client.routeCalls != 0 || client.trackCalls != 0 {
		t.Fatalf("client calls search=%d position=%d route=%d track=%d, want none", client.searchCalls, client.positionCalls, client.routeCalls, client.trackCalls)
	}
}

func TestFetchProcessorInvalidTaskReasonCombinationDoesNotCallFlightAware(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_invalid_reason_combo", "ANA110")

	tests := []struct {
		name string
		task FetchTask
	}{
		{
			name: "summary manual refresh",
			task: FetchTask{
				SchemaVersion:  1,
				TaskID:         "task-summary-manual",
				TaskType:       FetchTaskSummary,
				FlightID:       "ANA110",
				Reason:         FetchReasonUserManualRefresh,
				RequestedAt:    "2026-04-29T00:00:00Z",
				IdempotencyKey: "task-summary-manual",
			},
		},
		{
			name: "position search seed",
			task: FetchTask{
				SchemaVersion:  1,
				TaskID:         "task-position-seed",
				TaskType:       FetchTaskPosition,
				FlightID:       flight.FlightID,
				FAFlightID:     flight.FAFlightID,
				Reason:         FetchReasonSearchResultSeed,
				RequestedAt:    "2026-04-29T00:00:00Z",
				IdempotencyKey: "task-position-seed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMemoryFetchFlightStore(flight)
			client := &countingFlightAwareClient{}
			processor := NewFetchProcessor(FetchProcessorConfig{
				Flights:     store,
				Positions:   store,
				Artifacts:   store,
				FlightAware: client,
				Diagnostics: store,
			})

			result, err := processor.Process(ctx, tt.task, ProcessFetchInput{Now: now, WorkerID: "worker-1"})
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("Process() error = %v, want ErrValidation", err)
			}
			if result.ExternalFetchAttempted {
				t.Fatalf("ExternalFetchAttempted = true, want false for invalid task/reason combination")
			}
			if client.searchCalls != 0 || client.positionCalls != 0 || client.routeCalls != 0 || client.trackCalls != 0 {
				t.Fatalf("client calls search=%d position=%d route=%d track=%d, want none", client.searchCalls, client.positionCalls, client.routeCalls, client.trackCalls)
			}
		})
	}
}

func TestFetchProcessorSkipsExternalCallWhenUsageBudgetStopsFetching(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_budget_stopped", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{
		position: ExternalPositionResponse{
			FAFlightID: "fa_1",
			Latitude:   ptrFloat64(45.1),
			Longitude:  ptrFloat64(160.2),
			Timestamp:  "2026-04-29T00:01:00Z",
		},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		UsageGuard:  &memoryUsageGuard{fetchingEnabled: false},
	})

	result, err := processor.Process(ctx, taskFor(flight, FetchTaskPosition), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process(position) error = %v", err)
	}
	if !result.Skipped || result.SkipReason != "budget_stopped" || result.ExternalFetchAttempted {
		t.Fatalf("result = %#v, want budget_stopped skip before external fetch", result)
	}
	if client.positionCalls != 0 {
		t.Fatalf("position calls = %d, want no FlightAware call after budget stop", client.positionCalls)
	}
	if got := store.flights[flight.FlightID]; got.FetchOwner != nil || got.FetchLeaseUntil != nil {
		t.Fatalf("lease = owner %v until %v, want no lease acquired when budget stopped", got.FetchOwner, got.FetchLeaseUntil)
	}
}

func TestFetchProcessorSkipsSummaryFetchWhenUsageBudgetStopsFetching(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	store := newMemoryFetchFlightStore()
	client := &countingFlightAwareClient{}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		UsageGuard:  &memoryUsageGuard{fetchingEnabled: false},
	})

	result, err := processor.Process(ctx, FetchTask{
		SchemaVersion:  1,
		TaskID:         "search-ANA110",
		TaskType:       FetchTaskSummary,
		FlightID:       "ANA110",
		Reason:         FetchReasonSearchResultSeed,
		RequestedAt:    "2026-04-29T00:00:00Z",
		IdempotencyKey: "search-ANA110",
	}, ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process(summary) error = %v", err)
	}
	if !result.Skipped || result.SkipReason != "budget_stopped" || result.ExternalFetchAttempted {
		t.Fatalf("result = %#v, want budget_stopped skip before external fetch", result)
	}
	if client.searchCalls != 0 {
		t.Fatalf("search calls = %d, want no FlightAware call after budget stop", client.searchCalls)
	}
}

func TestFetchProcessorSkipsQueuedTaskWhenRuntimePolicyDisallowsIt(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_policy_disabled_worker", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{
		route: ExternalRouteResponse{Fixes: []ExternalRouteFix{
			{Name: "RJTT", Latitude: ptrFloat64(35.55), Longitude: ptrFloat64(139.78)},
			{Name: "KJFK", Latitude: ptrFloat64(40.64), Longitude: ptrFloat64(-73.78)},
		}},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		UsageGuard:  &memoryUsageGuard{fetchingEnabled: true},
		FetchPolicy: NewRuntimeFetchPolicy(RuntimeFetchConfig{
			ExternalFetchEnabled:   true,
			RouteFetchEnabled:      false,
			TrackFetchEnabled:      true,
			BackgroundFetchEnabled: true,
		}),
	})

	result, err := processor.Process(ctx, taskFor(flight, FetchTaskRoute), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process(route) error = %v", err)
	}
	if !result.Skipped || result.SkipReason != "fetch_disabled" || result.ExternalFetchAttempted {
		t.Fatalf("result = %#v, want fetch_disabled skip before external fetch", result)
	}
	if client.routeCalls != 0 {
		t.Fatalf("route calls = %d, want no FlightAware call after policy disabled", client.routeCalls)
	}
	if got := store.flights[flight.FlightID]; got.FetchOwner != nil || got.FetchLeaseUntil != nil {
		t.Fatalf("lease = owner %v until %v, want no lease acquired when policy disabled", got.FetchOwner, got.FetchLeaseUntil)
	}
}

func TestFetchProcessorSkipsAllQueuedFlightAwareTasksWhenRuntimePolicyDisablesExternalFetches(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	policy := NewRuntimeFetchPolicy(RuntimeFetchConfig{ExternalFetchEnabled: false})

	t.Run("summary", func(t *testing.T) {
		store := newMemoryFetchFlightStore()
		client := &countingFlightAwareClient{
			search: ExternalSearchFlightsResponse{Flights: []ExternalFlightSummary{
				{Ident: "ANA110", Origin: "RJTT", Destination: "KJFK"},
			}},
		}
		processor := NewFetchProcessor(FetchProcessorConfig{
			Flights:     store,
			Positions:   store,
			Artifacts:   store,
			FlightAware: client,
			UsageGuard:  &memoryUsageGuard{fetchingEnabled: true},
			FetchPolicy: policy,
		})

		result, err := processor.Process(ctx, FetchTask{
			SchemaVersion:  1,
			TaskID:         "search-ANA110",
			TaskType:       FetchTaskSummary,
			FlightID:       "ANA110",
			Reason:         FetchReasonSearchResultSeed,
			RequestedAt:    "2026-04-29T00:00:00Z",
			IdempotencyKey: "search-ANA110",
		}, ProcessFetchInput{Now: now, WorkerID: "worker-1"})
		if err != nil {
			t.Fatalf("Process(summary) error = %v", err)
		}
		if !result.Skipped || result.SkipReason != "fetch_disabled" || result.ExternalFetchAttempted {
			t.Fatalf("result = %#v, want fetch_disabled skip before external fetch", result)
		}
		if client.searchCalls != 0 {
			t.Fatalf("search calls = %d, want no FlightAware call after policy disabled", client.searchCalls)
		}
	})

	t.Run("manual position", func(t *testing.T) {
		flight := testFlight("iflg_policy_disabled_position", "ANA110")
		store := newMemoryFetchFlightStore(flight)
		client := &countingFlightAwareClient{
			position: ExternalPositionResponse{
				FAFlightID: "fa_1",
				Latitude:   ptrFloat64(45.1),
				Longitude:  ptrFloat64(160.2),
				Timestamp:  "2026-04-29T00:01:00Z",
			},
		}
		processor := NewFetchProcessor(FetchProcessorConfig{
			Flights:     store,
			Positions:   store,
			Artifacts:   store,
			FlightAware: client,
			UsageGuard:  &memoryUsageGuard{fetchingEnabled: true},
			FetchPolicy: policy,
		})
		task := taskFor(flight, FetchTaskPosition)
		task.Reason = FetchReasonUserManualRefresh

		result, err := processor.Process(ctx, task, ProcessFetchInput{Now: now, WorkerID: "worker-1"})
		if err != nil {
			t.Fatalf("Process(position) error = %v", err)
		}
		if !result.Skipped || result.SkipReason != "fetch_disabled" || result.ExternalFetchAttempted {
			t.Fatalf("result = %#v, want fetch_disabled skip before external fetch", result)
		}
		if client.positionCalls != 0 {
			t.Fatalf("position calls = %d, want no FlightAware call after policy disabled", client.positionCalls)
		}
		if got := store.flights[flight.FlightID]; got.FetchOwner != nil || got.FetchLeaseUntil != nil {
			t.Fatalf("lease = owner %v until %v, want no lease acquired when policy disabled", got.FetchOwner, got.FetchLeaseUntil)
		}
	})
}

func TestFetchProcessorProcessesSummarySearchTaskIntoCachedFlights(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	existingInternalID := domain.GenerateInternalFlightLegID(domain.InternalFlightLegIDInput{
		FAFlightID:      "fa_current_1",
		OriginCode:      "RJTT",
		DestinationCode: "KJFK",
		ScheduledOut:    "2026-04-29T10:00:00Z",
		LegIndex:        0,
	})
	existingRouteKey := "routes/existing-current.json"
	existing := domain.Flight{
		FlightID:            domain.FlightID(existingInternalID),
		FlightIDType:        domain.FlightIDTypeInternal,
		InternalFlightLegID: &existingInternalID,
		Ident:               "ANA110",
		Origin:              domain.Airport{Code: "RJTT"},
		Destination:         domain.Airport{Code: "KJFK"},
		PlannedRouteS3Key:   &existingRouteKey,
	}
	store := newMemoryFetchFlightStore(existing)
	client := &countingFlightAwareClient{
		search: ExternalSearchFlightsResponse{Flights: []ExternalFlightSummary{
			{
				FAFlightID:      ptr("fa_current_1"),
				Ident:           "ANA110",
				IdentIATA:       ptr("NH110"),
				Origin:          "RJTT",
				Destination:     "KJFK",
				ScheduledOut:    ptr("2026-04-29T10:00:00Z"),
				Status:          "En Route",
				ProgressPercent: ptrInt(42),
			},
			{
				Ident:        "ANA110",
				Origin:       "RJTT",
				Destination:  "KJFK",
				ScheduledOut: ptr("2026-04-30T10:00:00Z"),
				Status:       "Scheduled",
			},
		}},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	result, err := processor.Process(ctx, FetchTask{
		SchemaVersion:  1,
		TaskID:         "search-ANA110",
		TaskType:       FetchTaskSummary,
		FlightID:       "ANA110",
		Reason:         FetchReasonSearchResultSeed,
		RequestedAt:    "2026-04-29T00:00:00Z",
		IdempotencyKey: "search-ANA110",
	}, ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process(summary) error = %v", err)
	}

	if !result.ExternalFetchAttempted || result.UpdatedSummaryCount != 2 || client.searchCalls != 1 {
		t.Fatalf("summary result = %#v searchCalls=%d, want two cached flights", result, client.searchCalls)
	}
	searchResults := store.searchResults["ANA110"]
	if len(searchResults) != 2 {
		t.Fatalf("cached search results = %#v, want two flights", searchResults)
	}
	if searchResults[0].FlightIDType != domain.FlightIDTypeInternal || searchResults[0].FAFlightID == nil {
		t.Fatalf("first cached flight = %#v, want internal FlightAware-backed flight", searchResults[0])
	}
	if searchResults[1].FlightIDType != domain.FlightIDTypeProvisional || searchResults[1].ProvisionalFlightLegID == nil {
		t.Fatalf("second cached flight = %#v, want provisional scheduled flight", searchResults[1])
	}
	if got := store.flights[searchResults[0].FlightID]; got.IdentIATA == nil || *got.IdentIATA != "NH110" || got.ProgressPercent == nil || *got.ProgressPercent != 42 {
		t.Fatalf("stored first flight = %#v, want normalized summary fields", got)
	}
	if got := store.flights[searchResults[0].FlightID]; got.PlannedRouteS3Key == nil || *got.PlannedRouteS3Key != existingRouteKey {
		t.Fatalf("stored first flight route key = %v, want existing route key preserved", got.PlannedRouteS3Key)
	}
}

func TestFetchProcessorSkipsInvalidSummaryRowsWithoutFailingTask(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	store := newMemoryFetchFlightStore()
	client := &countingFlightAwareClient{
		search: ExternalSearchFlightsResponse{Flights: []ExternalFlightSummary{
			{
				FAFlightID:   ptr("fa_current_1"),
				Ident:        "ANA110",
				Origin:       "RJTT",
				Destination:  "KJFK",
				ScheduledOut: ptr("2026-04-29T10:00:00Z"),
			},
			{
				Ident:       "ANA110",
				Origin:      "RJTT",
				Destination: "",
			},
			{
				Ident:        "ANA110",
				Origin:       "RJTT",
				Destination:  "KJFK",
				ScheduledOut: ptr("2026-04-30T10:00:00Z"),
			},
		}},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	result, err := processor.Process(ctx, FetchTask{
		SchemaVersion:  1,
		TaskID:         "search-ANA110",
		TaskType:       FetchTaskSummary,
		FlightID:       "ANA110",
		Reason:         FetchReasonSearchResultSeed,
		RequestedAt:    "2026-04-29T00:00:00Z",
		IdempotencyKey: "search-ANA110",
	}, ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process(summary with invalid row) error = %v", err)
	}

	if result.UpdatedSummaryCount != 2 {
		t.Fatalf("UpdatedSummaryCount = %d, want only valid rows cached", result.UpdatedSummaryCount)
	}
	if len(store.searchResults["ANA110"]) != 2 {
		t.Fatalf("cached search results = %#v, want only valid rows", store.searchResults["ANA110"])
	}
	if len(store.diagnostics) != 1 || store.diagnostics[0].ErrorCode != "validation_failed" {
		t.Fatalf("diagnostics = %#v, want one safe validation diagnostic for skipped row", store.diagnostics)
	}
}

func TestFetchProcessorDoesNotPersistResultsAfterLeaseIsStolen(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_stale_worker", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	store.stealLeaseBeforeUpdate = true
	client := &countingFlightAwareClient{
		position: ExternalPositionResponse{
			FAFlightID: "fa_1",
			Latitude:   ptrFloat64(45.1),
			Longitude:  ptrFloat64(160.2),
			Timestamp:  "2026-04-29T00:01:00Z",
		},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	result, err := processor.Process(ctx, taskFor(flight, FetchTaskPosition), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !result.Skipped || result.SkipReason != "lease_lost" {
		t.Fatalf("result = %#v, want lease_lost skip", result)
	}
	got := store.flights[flight.FlightID]
	if got.LatestPositionTimestamp != nil {
		t.Fatalf("LatestPositionTimestamp = %v, want stale worker update ignored", *got.LatestPositionTimestamp)
	}
	if len(store.positions[flight.FlightID]) != 0 {
		t.Fatalf("position history count = %d, want stale worker append ignored", len(store.positions[flight.FlightID]))
	}
	if got.FetchOwner == nil || *got.FetchOwner != "worker-2" {
		t.Fatalf("FetchOwner = %v, want stolen lease owner retained", got.FetchOwner)
	}
}

func TestFetchProcessorDoesNotAppendPositionWhenLeaseIsLostBeforeHistoryWrite(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_position_history_lease_lost", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	store.stealLeaseBeforeGuardedAppend = true
	client := &countingFlightAwareClient{
		position: ExternalPositionResponse{
			FAFlightID: "fa_1",
			Latitude:   ptrFloat64(45.1),
			Longitude:  ptrFloat64(160.2),
			Timestamp:  "2026-04-29T00:01:00Z",
		},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	result, err := processor.Process(ctx, taskFor(flight, FetchTaskPosition), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process(position) error = %v", err)
	}
	if !result.Skipped || result.SkipReason != "lease_lost" {
		t.Fatalf("result = %#v, want lease_lost skip", result)
	}
	if len(store.positions[flight.FlightID]) != 0 {
		t.Fatalf("position history count = %d, want stale worker append rejected", len(store.positions[flight.FlightID]))
	}
}

func TestFetchProcessorUpdatesLatestPositionAndHistory(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_position", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{
		position: ExternalPositionResponse{
			FAFlightID: "fa_1",
			Latitude:   ptrFloat64(45.1),
			Longitude:  ptrFloat64(160.2),
			Timestamp:  "2026-04-29T00:01:00Z",
		},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	result, err := processor.Process(ctx, taskFor(flight, FetchTaskPosition), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process(position) error = %v", err)
	}
	if !result.ExternalFetchAttempted || result.UpdatedPositionCount != 1 {
		t.Fatalf("position result = %#v", result)
	}
	got := store.flights[flight.FlightID]
	if got.LatestPositionTimestamp == nil || *got.LatestPositionTimestamp != "2026-04-29T00:01:00Z" {
		t.Fatalf("latest timestamp = %v, want updated timestamp", got.LatestPositionTimestamp)
	}
	if len(store.positions[flight.FlightID]) != 1 {
		t.Fatalf("position history count = %d, want 1", len(store.positions[flight.FlightID]))
	}
}

func TestFetchProcessorPersistsPositionMetrics(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_position_metrics", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{
		position: ExternalPositionResponse{
			FAFlightID:           "fa_1",
			Latitude:             ptrFloat64(45.1),
			Longitude:            ptrFloat64(160.2),
			Timestamp:            "2026-04-29T00:01:00Z",
			AltitudeHundredsFeet: ptrInt(370),
			AltitudeChange:       ptrAltitudeChange(domain.AltitudeChangeLevel),
			GroundspeedKnots:     ptrInt(488),
			HeadingDegrees:       ptrInt(360),
			UpdateType:           ptrPositionUpdateType(domain.PositionUpdateTypeEstimated),
		},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	if _, err := processor.Process(ctx, taskFor(flight, FetchTaskPosition), ProcessFetchInput{Now: now, WorkerID: "worker-1"}); err != nil {
		t.Fatalf("Process(position) error = %v", err)
	}

	history := store.positions[flight.FlightID]
	if len(history) != 1 {
		t.Fatalf("position history count = %d, want 1", len(history))
	}
	got := history[0]
	if got.AltitudeHundredsFeet == nil || *got.AltitudeHundredsFeet != 370 {
		t.Fatalf("AltitudeHundredsFeet = %v, want 370", got.AltitudeHundredsFeet)
	}
	if got.AltitudeFeet == nil || *got.AltitudeFeet != 37000 {
		t.Fatalf("AltitudeFeet = %v, want 37000", got.AltitudeFeet)
	}
	if got.AltitudeChange == nil || *got.AltitudeChange != domain.AltitudeChangeLevel {
		t.Fatalf("AltitudeChange = %v, want level", got.AltitudeChange)
	}
	if got.GroundspeedKnots == nil || *got.GroundspeedKnots != 488 {
		t.Fatalf("GroundspeedKnots = %v, want 488", got.GroundspeedKnots)
	}
	if got.HeadingDegrees == nil || *got.HeadingDegrees != 0 {
		t.Fatalf("HeadingDegrees = %v, want normalized 0", got.HeadingDegrees)
	}
	if got.UpdateType == nil || *got.UpdateType != domain.PositionUpdateTypeEstimated {
		t.Fatalf("UpdateType = %v, want estimated", got.UpdateType)
	}
}

func TestFetchProcessorDoesNotAdvanceFlightWhenPositionAppendFails(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_position_append_fails", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	store.appendPositionErr = errors.New("position write failed")
	client := &countingFlightAwareClient{
		position: ExternalPositionResponse{
			FAFlightID: "fa_1",
			Latitude:   ptrFloat64(45.1),
			Longitude:  ptrFloat64(160.2),
			Timestamp:  "2026-04-29T00:01:00Z",
		},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	_, err := processor.Process(ctx, taskFor(flight, FetchTaskPosition), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err == nil {
		t.Fatal("Process(position) error = nil, want append failure")
	}
	got := store.flights[flight.FlightID]
	if got.LatestPositionTimestamp != nil || got.NextPositionPollAt != nil {
		t.Fatalf("flight after append failure = %#v, want latest position and schedule unchanged", got)
	}
}

func TestFetchProcessorStoresRouteAndTrackOnlyForExplicitTasks(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_layers", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{
		route: ExternalRouteResponse{Fixes: []ExternalRouteFix{
			{Name: "RJTT", Latitude: ptrFloat64(35.55), Longitude: ptrFloat64(139.78)},
			{Name: "KJFK", Latitude: ptrFloat64(40.64), Longitude: ptrFloat64(-73.78)},
		}},
		track: ExternalTrackResponse{Positions: []ExternalTrackPoint{
			{Latitude: 35.55, Longitude: 139.78, Timestamp: "2026-04-29T00:00:00Z"},
			{Latitude: 40.64, Longitude: -73.78, Timestamp: "2026-04-29T12:00:00Z"},
		}},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	if _, err := processor.Process(ctx, taskFor(flight, FetchTaskRoute), ProcessFetchInput{Now: now, WorkerID: "worker-1"}); err != nil {
		t.Fatalf("Process(route) error = %v", err)
	}
	if _, err := processor.Process(ctx, taskFor(flight, FetchTaskTrack), ProcessFetchInput{Now: now.Add(time.Minute), WorkerID: "worker-1"}); err != nil {
		t.Fatalf("Process(track) error = %v", err)
	}

	got := store.flights[flight.FlightID]
	if got.PlannedRouteS3Key == nil || got.ActualTrackS3Key == nil {
		t.Fatalf("stored keys = route %v track %v, want both", got.PlannedRouteS3Key, got.ActualTrackS3Key)
	}
	if client.routeCalls != 1 || client.trackCalls != 1 || client.positionCalls != 0 {
		t.Fatalf("client calls route=%d track=%d position=%d", client.routeCalls, client.trackCalls, client.positionCalls)
	}
	if len(store.positions[flight.FlightID]) != 2 {
		t.Fatalf("track history positions = %d, want 2", len(store.positions[flight.FlightID]))
	}
}

func TestFetchProcessorStoresRouteArtifactWithOnlyValidFixes(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_route_sanitized", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{
		route: ExternalRouteResponse{Fixes: []ExternalRouteFix{
			{Name: "RJTT", Latitude: ptrFloat64(35.55), Longitude: ptrFloat64(139.78)},
			{Name: "bad", Latitude: ptrFloat64(math.NaN()), Longitude: ptrFloat64(140.00)},
			{Name: "KJFK", Latitude: ptrFloat64(40.64), Longitude: ptrFloat64(-73.78)},
		}},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	result, err := processor.Process(ctx, taskFor(flight, FetchTaskRoute), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process(route) error = %v", err)
	}

	if !result.UpdatedRoute || len(store.lastRouteArtifact.Fixes) != 2 {
		t.Fatalf("route result=%#v artifact=%#v, want only valid fixes stored", result, store.lastRouteArtifact)
	}
	for _, fix := range store.lastRouteArtifact.Fixes {
		if fix.Latitude == nil || fix.Longitude == nil || !validCoordinate(*fix.Latitude, *fix.Longitude) {
			t.Fatalf("stored invalid route fix = %#v", fix)
		}
	}
}

func TestFlightFromExternalSummaryKeepsInternalIDStableWhenResultOrderChanges(t *testing.T) {
	now := mustTime(t, "2026-04-29T00:00:00Z")
	summary := ExternalFlightSummary{
		FAFlightID:   ptr("fa_current_1"),
		Ident:        "ANA110",
		Origin:       "RJTT",
		Destination:  "KJFK",
		ScheduledOut: ptr("2026-04-29T10:00:00Z"),
	}

	first, ok := flightFromExternalSummary(summary, now, 0)
	if !ok {
		t.Fatal("flightFromExternalSummary(index 0) ok = false, want true")
	}
	second, ok := flightFromExternalSummary(summary, now, 3)
	if !ok {
		t.Fatal("flightFromExternalSummary(index 3) ok = false, want true")
	}

	if first.FlightID != second.FlightID {
		t.Fatalf("flight IDs differ by result order: first=%q second=%q", first.FlightID, second.FlightID)
	}
	if second.LegIndex == nil || *second.LegIndex != 3 {
		t.Fatalf("LegIndex = %v, want public result order preserved", second.LegIndex)
	}
}

func TestFetchProcessorIgnoresInvalidTrackPointsAndUsesLatestTimestamp(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_track_order", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{
		track: ExternalTrackResponse{Positions: []ExternalTrackPoint{
			{Latitude: 40.64, Longitude: -73.78, Timestamp: "2026-04-29T12:00:00Z"},
			{Latitude: 35.55, Longitude: 139.78, Timestamp: "2026-04-29T00:00:00Z"},
			{Latitude: 36.00, Longitude: 140.00, Timestamp: ""},
		}},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	result, err := processor.Process(ctx, taskFor(flight, FetchTaskTrack), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process(track) error = %v", err)
	}

	got := store.flights[flight.FlightID]
	if result.UpdatedPositionCount != 2 {
		t.Fatalf("UpdatedPositionCount = %d, want only valid timestamped points", result.UpdatedPositionCount)
	}
	if got.LatestPositionTimestamp == nil || *got.LatestPositionTimestamp != "2026-04-29T12:00:00Z" {
		t.Fatalf("latest timestamp = %v, want chronologically latest track point", got.LatestPositionTimestamp)
	}
	history := store.positions[flight.FlightID]
	if len(history) != 2 || history[0].Timestamp != "2026-04-29T00:00:00Z" || history[1].Timestamp != "2026-04-29T12:00:00Z" {
		t.Fatalf("history = %#v, want valid points in timestamp order", history)
	}
}

func TestFetchProcessorSortsTrackPointsChronologicallyAcrossOffsets(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_track_offsets", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{
		track: ExternalTrackResponse{Positions: []ExternalTrackPoint{
			{Latitude: 35.55, Longitude: 139.78, Timestamp: "2026-04-29T09:00:00+09:00"},
			{Latitude: 40.64, Longitude: -73.78, Timestamp: "2026-04-29T00:30:00Z"},
		}},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	result, err := processor.Process(ctx, taskFor(flight, FetchTaskTrack), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process(track) error = %v", err)
	}

	got := store.flights[flight.FlightID]
	if result.UpdatedPositionCount != 2 {
		t.Fatalf("UpdatedPositionCount = %d, want 2", result.UpdatedPositionCount)
	}
	if got.LatestPositionTimestamp == nil || *got.LatestPositionTimestamp != "2026-04-29T00:30:00Z" {
		t.Fatalf("latest timestamp = %v, want latest instant across offsets", got.LatestPositionTimestamp)
	}
	history := store.positions[flight.FlightID]
	if len(history) != 2 || history[0].Timestamp != "2026-04-29T09:00:00+09:00" || history[1].Timestamp != "2026-04-29T00:30:00Z" {
		t.Fatalf("history = %#v, want chronological order by parsed instant", history)
	}
}

func TestFetchProcessorStoresTrackArtifactFromValidatedSortedTrackPoints(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_track_artifact_validated", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{
		track: ExternalTrackResponse{Positions: []ExternalTrackPoint{
			{Latitude: 40.64, Longitude: -73.78, Timestamp: "2026-04-29T12:00:00Z"},
			{Latitude: 36.00, Longitude: 140.00, Timestamp: ""},
			{Latitude: 35.55, Longitude: 139.78, Timestamp: "2026-04-29T00:00:00Z"},
		}},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	if _, err := processor.Process(ctx, taskFor(flight, FetchTaskTrack), ProcessFetchInput{Now: now, WorkerID: "worker-1"}); err != nil {
		t.Fatalf("Process(track) error = %v", err)
	}

	if len(store.lastTrackArtifact.Positions) != 2 {
		t.Fatalf("track artifact positions = %#v, want only valid timestamped points", store.lastTrackArtifact.Positions)
	}
	if store.lastTrackArtifact.Positions[0].Timestamp != "2026-04-29T00:00:00Z" ||
		store.lastTrackArtifact.Positions[1].Timestamp != "2026-04-29T12:00:00Z" {
		t.Fatalf("track artifact positions = %#v, want timestamp order", store.lastTrackArtifact.Positions)
	}
}

func TestFetchProcessorDoesNotPublishUnavailableRouteOrTrackArtifacts(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_unavailable_artifacts", "ANA110")
	flight.NextRoutePollAt = ptr("2026-04-29T00:00:00Z")
	flight.NextTrackPollAt = ptr("2026-04-29T00:00:00Z")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{
		route: ExternalRouteResponse{Fixes: []ExternalRouteFix{
			{Name: "RJTT", Latitude: ptrFloat64(35.55), Longitude: ptrFloat64(139.78)},
		}},
		track: ExternalTrackResponse{Positions: []ExternalTrackPoint{
			{Latitude: 35.55, Longitude: 139.78, Timestamp: "2026-04-29T00:00:00Z"},
		}},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	routeResult, err := processor.Process(ctx, taskFor(flight, FetchTaskRoute), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process(route) error = %v", err)
	}
	trackResult, err := processor.Process(ctx, taskFor(flight, FetchTaskTrack), ProcessFetchInput{Now: now.Add(time.Minute), WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process(track) error = %v", err)
	}

	if !routeResult.Skipped || routeResult.SkipReason != "route_unavailable" {
		t.Fatalf("route result = %#v, want route_unavailable skip", routeResult)
	}
	if !trackResult.Skipped || trackResult.SkipReason != "track_unavailable" {
		t.Fatalf("track result = %#v, want track_unavailable skip", trackResult)
	}
	if store.routeCount != 0 || store.trackCount != 0 {
		t.Fatalf("artifact writes route=%d track=%d, want none", store.routeCount, store.trackCount)
	}
	got := store.flights[flight.FlightID]
	if got.PlannedRouteS3Key != nil || got.ActualTrackS3Key != nil {
		t.Fatalf("artifact keys = route %v track %v, want none", got.PlannedRouteS3Key, got.ActualTrackS3Key)
	}
	assertTimePtr(t, got.NextRoutePollAt, "2026-04-29T00:00:00Z")
	assertTimePtr(t, got.NextTrackPollAt, "2026-04-29T00:00:00Z")
}

func TestFetchProcessorDoesNotLeavePartialTrackHistoryWhenLeaseIsLostBeforeFlightUpdate(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_track_partial_lease", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	store.stealLeaseDuringTrackTransaction = true
	client := &countingFlightAwareClient{
		track: ExternalTrackResponse{Positions: []ExternalTrackPoint{
			{Latitude: 35.55, Longitude: 139.78, Timestamp: "2026-04-29T00:00:00Z"},
			{Latitude: 40.64, Longitude: -73.78, Timestamp: "2026-04-29T12:00:00Z"},
		}},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	result, err := processor.Process(ctx, taskFor(flight, FetchTaskTrack), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process(track) error = %v", err)
	}
	if !result.Skipped || result.SkipReason != "lease_lost" {
		t.Fatalf("result = %#v, want lease_lost", result)
	}
	if len(store.positions[flight.FlightID]) != 0 {
		t.Fatalf("track history count = %d, want no partial position writes after lease loss", len(store.positions[flight.FlightID]))
	}
	got := store.flights[flight.FlightID]
	if got.ActualTrackS3Key != nil || got.LatestPositionTimestamp != nil {
		t.Fatalf("flight after lost track transaction = %#v, want track metadata unchanged", got)
	}
}

func TestFetchProcessorRequiresAtomicTrackWriterForTrackUpdates(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_track_requires_atomic", "ANA110")
	store := newFallbackOnlyFetchStore(flight)
	client := &countingFlightAwareClient{
		track: ExternalTrackResponse{Positions: []ExternalTrackPoint{
			{Latitude: 35.55, Longitude: 139.78, Timestamp: "2026-04-29T00:00:00Z"},
			{Latitude: 40.64, Longitude: -73.78, Timestamp: "2026-04-29T12:00:00Z"},
		}},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store.inner,
	})

	result, err := processor.Process(ctx, taskFor(flight, FetchTaskTrack), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if !errors.Is(err, ErrAtomicTrackWriterRequired) {
		t.Fatalf("Process(track) error = %v, want ErrAtomicTrackWriterRequired", err)
	}
	if !result.ExternalFetchAttempted || result.UpdatedTrack || result.UpdatedPositionCount != 0 {
		t.Fatalf("result = %#v, want external attempt without persisted track updates", result)
	}
	if len(store.inner.positions[flight.FlightID]) != 0 {
		t.Fatalf("track history count = %d, want no fallback partial writes", len(store.inner.positions[flight.FlightID]))
	}
	got := store.inner.flights[flight.FlightID]
	if got.ActualTrackS3Key != nil || got.LatestPositionTimestamp != nil {
		t.Fatalf("flight after missing atomic writer = %#v, want track metadata unchanged", got)
	}
}

func TestFetchProcessorRequiresAtomicPositionWriterForPositionUpdates(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_position_requires_atomic", "ANA110")
	store := newFallbackOnlyFetchStore(flight)
	client := &countingFlightAwareClient{
		position: ExternalPositionResponse{
			FAFlightID: "fa_1",
			Latitude:   ptrFloat64(45.1),
			Longitude:  ptrFloat64(160.2),
			Timestamp:  "2026-04-29T00:01:00Z",
		},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store.inner,
	})

	result, err := processor.Process(ctx, taskFor(flight, FetchTaskPosition), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if !errors.Is(err, ErrAtomicPositionWriterRequired) {
		t.Fatalf("Process(position) error = %v, want ErrAtomicPositionWriterRequired", err)
	}
	if !result.ExternalFetchAttempted || result.UpdatedPositionCount != 0 {
		t.Fatalf("result = %#v, want external attempt without persisted position updates", result)
	}
	if len(store.inner.positions[flight.FlightID]) != 0 {
		t.Fatalf("position history count = %d, want no fallback partial writes", len(store.inner.positions[flight.FlightID]))
	}
	got := store.inner.flights[flight.FlightID]
	if got.LatestPositionTimestamp != nil || got.LatestPositionSource != nil {
		t.Fatalf("flight after missing atomic writer = %#v, want position metadata unchanged", got)
	}
}

func TestFetchProcessorDoesNotOverwritePublishedArtifactWhenLeaseIsLost(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_stale_artifact", "ANA110")
	flight.PlannedRouteS3Key = ptr("routes/iflg_stale_artifact.json")
	store := newMemoryFetchFlightStore(flight)
	store.stealLeaseBeforeUpdate = true
	client := &countingFlightAwareClient{
		route: ExternalRouteResponse{Fixes: []ExternalRouteFix{
			{Name: "RJTT", Latitude: ptrFloat64(35.55), Longitude: ptrFloat64(139.78)},
			{Name: "KJFK", Latitude: ptrFloat64(40.64), Longitude: ptrFloat64(-73.78)},
		}},
	}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	result, err := processor.Process(ctx, taskFor(flight, FetchTaskRoute), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if err != nil {
		t.Fatalf("Process(route) error = %v", err)
	}

	if !result.Skipped || result.SkipReason != "lease_lost" {
		t.Fatalf("result = %#v, want lease_lost", result)
	}
	if store.overwrotePublishedArtifact {
		t.Fatal("stale worker overwrote the previously published route artifact key")
	}
	got := store.flights[flight.FlightID]
	if got.PlannedRouteS3Key == nil || *got.PlannedRouteS3Key != "routes/iflg_stale_artifact.json" {
		t.Fatalf("published route key = %v, want original key retained", got.PlannedRouteS3Key)
	}
}

func TestPollSchedulePolicyStopsIdleFlightsQuickly(t *testing.T) {
	now := mustTime(t, "2026-04-29T00:20:00Z")
	flight := testFlight("iflg_idle", "ANA110")
	flight.IdleSince = ptr("2026-04-29T00:00:00Z")
	flight.NextPositionPollAt = ptr("2026-04-29T00:15:00Z")
	flight.NextTrackPollAt = ptr("2026-04-29T00:45:00Z")
	flight.NextRoutePollAt = ptr("2026-04-29T06:00:00Z")

	got := DefaultPollSchedulePolicy().Compute(flight, now, PollActivitySignal{})

	if got.NextPositionPollAt != nil || got.NextTrackPollAt != nil || got.NextRoutePollAt != nil {
		t.Fatalf("poll times = %v/%v/%v, want stopped", got.NextPositionPollAt, got.NextTrackPollAt, got.NextRoutePollAt)
	}
	if got.PollState == nil || *got.PollState != domain.FlightPollStateCompleted {
		t.Fatalf("poll state = %v, want completed", got.PollState)
	}
}

func TestFetchProcessorRecordsSafeDiagnosticForFailedTasks(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_fail", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{err: ErrUpstreamRateLimited}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	_, err := processor.Process(ctx, taskFor(flight, FetchTaskPosition), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if !errors.Is(err, ErrUpstreamRateLimited) {
		t.Fatalf("Process(failed) error = %v, want rate limited", err)
	}
	if len(store.diagnostics) != 1 {
		t.Fatalf("diagnostics count = %d, want 1", len(store.diagnostics))
	}
	got := store.diagnostics[0]
	if got.TaskID != "task-position" || got.TaskType != FetchTaskPosition || got.FlightID != flight.FlightID {
		t.Fatalf("diagnostic = %#v", got)
	}
	if got.FAFlightID != "" || got.ErrorCode != "rate_limited" {
		t.Fatalf("diagnostic safe fields = %#v, want redacted faFlightId and typed error", got)
	}
}

func TestPositionsFromFlightAwareTrackDedupesDuplicateTimestamps(t *testing.T) {
	flight := testFlight("iflg_track_dedupe", "ANA110")
	positions := positionsFromFlightAwareTrack(flight, ExternalTrackResponse{Positions: []ExternalTrackPoint{
		{Latitude: 35.0, Longitude: 139.0, Timestamp: "2026-04-29T00:01:00Z"},
		{Latitude: 36.0, Longitude: 140.0, Timestamp: "2026-04-29T00:01:00Z", AltitudeHundredsFeet: ptrInt(380), HeadingDegrees: ptrInt(360)},
		{Latitude: 37.0, Longitude: 141.0, Timestamp: "2026-04-29T00:02:00Z"},
	}})

	if len(positions) != 2 {
		t.Fatalf("position count = %d, want duplicate timestamps collapsed", len(positions))
	}
	if positions[0].Latitude != 36.0 || positions[0].Longitude != 140.0 {
		t.Fatalf("deduped first position = %#v, want latest point for duplicate timestamp", positions[0])
	}
	if positions[0].AltitudeFeet == nil || *positions[0].AltitudeFeet != 38000 || positions[0].HeadingDegrees == nil || *positions[0].HeadingDegrees != 0 {
		t.Fatalf("deduped first position metrics = %#v, want latest normalized metrics", positions[0])
	}
}

func taskFor(flight domain.Flight, taskType FetchTaskType) FetchTask {
	return FetchTask{
		SchemaVersion:  1,
		TaskID:         "task-" + string(taskType),
		TaskType:       taskType,
		FlightID:       flight.FlightID,
		FAFlightID:     flight.FAFlightID,
		Reason:         FetchReasonLowFrequencyPoll,
		RequestedAt:    "2026-04-29T00:00:00Z",
		IdempotencyKey: "task-" + string(taskType),
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}

func assertTimePtr(t *testing.T, got *domain.ISODateTimeString, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("time = %v, want %s", got, want)
	}
}

func ptrFloat64(value float64) *float64 {
	return &value
}

func ptrAltitudeChange(value domain.AltitudeChange) *domain.AltitudeChange {
	return &value
}

func ptrPositionUpdateType(value domain.PositionUpdateType) *domain.PositionUpdateType {
	return &value
}

type memoryPollFlightStore struct {
	flights map[domain.FlightID]domain.Flight
}

func (s *memoryPollFlightStore) ListPollableFlights(_ context.Context, _ string, limit int) ([]domain.Flight, error) {
	flights := make([]domain.Flight, 0, len(s.flights))
	for _, flight := range s.flights {
		flights = append(flights, flight)
		if limit > 0 && len(flights) >= limit {
			break
		}
	}
	return flights, nil
}

func (s *memoryPollFlightStore) UpdatePollSchedule(_ context.Context, flight domain.Flight) error {
	s.flights[flight.FlightID] = flight
	return nil
}

type memoryPollActivityStore struct {
	signal PollActivitySignal
}

func (s memoryPollActivityStore) ActivityForFlight(context.Context, domain.FlightID) (PollActivitySignal, error) {
	return s.signal, nil
}

type memoryFetchFlightStore struct {
	flights                          map[domain.FlightID]domain.Flight
	searchResults                    map[string][]domain.Flight
	publishedArtifactKeys            map[string]struct{}
	positions                        map[domain.FlightID][]domain.FlightPosition
	diagnostics                      []FetchTaskDiagnostic
	lastRouteArtifact                ExternalRouteResponse
	lastTrackArtifact                ExternalTrackResponse
	routeCount                       int
	trackCount                       int
	stealLeaseBeforeUpdate           bool
	stealLeaseBeforeGuardedAppend    bool
	stealLeaseDuringTrackTransaction bool
	overwrotePublishedArtifact       bool
	appendPositionErr                error
}

func newMemoryFetchFlightStore(flights ...domain.Flight) *memoryFetchFlightStore {
	store := &memoryFetchFlightStore{
		flights:               map[domain.FlightID]domain.Flight{},
		searchResults:         map[string][]domain.Flight{},
		publishedArtifactKeys: map[string]struct{}{},
		positions:             map[domain.FlightID][]domain.FlightPosition{},
	}
	for _, flight := range flights {
		store.flights[flight.FlightID] = flight
		if flight.PlannedRouteS3Key != nil {
			store.publishedArtifactKeys[*flight.PlannedRouteS3Key] = struct{}{}
		}
		if flight.ActualTrackS3Key != nil {
			store.publishedArtifactKeys[*flight.ActualTrackS3Key] = struct{}{}
		}
	}
	return store
}

func (s *memoryFetchFlightStore) AcquireFetchLease(_ context.Context, flightID domain.FlightID, owner string, leaseUntil string, now string) (domain.Flight, bool, error) {
	flight, ok := s.flights[flightID]
	if !ok {
		return domain.Flight{}, false, ErrNotFound
	}
	if flight.FetchLeaseUntil != nil && *flight.FetchLeaseUntil > now && (flight.FetchOwner == nil || *flight.FetchOwner != owner) {
		return flight, false, nil
	}
	flight.FetchOwner = &owner
	flight.FetchLeaseUntil = &leaseUntil
	s.flights[flightID] = flight
	return flight, true, nil
}

func (s *memoryFetchFlightStore) GetFlight(_ context.Context, flightID domain.FlightID) (domain.Flight, CacheMetadata, error) {
	flight, ok := s.flights[flightID]
	if !ok {
		return domain.Flight{}, CacheMetadata{Freshness: CacheFreshnessMiss, Source: CacheSourceCache}, ErrNotFound
	}
	return flight, CacheMetadata{Freshness: CacheFreshnessFresh, Source: CacheSourceCache}, nil
}

func (s *memoryFetchFlightStore) UpdateFetchedFlight(_ context.Context, flight domain.Flight, owner string, leaseUntil string, _ string) (bool, error) {
	if s.stealLeaseBeforeUpdate {
		stolen := s.flights[flight.FlightID]
		stolen.FetchOwner = ptr("worker-2")
		stolen.FetchLeaseUntil = ptr("2026-04-29T00:10:00Z")
		s.flights[flight.FlightID] = stolen
		s.stealLeaseBeforeUpdate = false
	}
	current := s.flights[flight.FlightID]
	if current.FetchOwner == nil || *current.FetchOwner != owner || current.FetchLeaseUntil == nil || *current.FetchLeaseUntil != leaseUntil {
		return false, nil
	}
	s.flights[flight.FlightID] = flight
	return true, nil
}

func (s *memoryFetchFlightStore) UpdateFetchedFlightWithPosition(_ context.Context, flight domain.Flight, position domain.FlightPosition, owner string, leaseUntil string, _ string) (bool, error) {
	if s.appendPositionErr != nil {
		return false, s.appendPositionErr
	}
	if s.stealLeaseBeforeUpdate || s.stealLeaseBeforeGuardedAppend {
		stolen := s.flights[flight.FlightID]
		stolen.FetchOwner = ptr("worker-2")
		stolen.FetchLeaseUntil = ptr("2026-04-29T00:10:00Z")
		s.flights[flight.FlightID] = stolen
		s.stealLeaseBeforeUpdate = false
		s.stealLeaseBeforeGuardedAppend = false
	}
	current := s.flights[flight.FlightID]
	if current.FetchOwner == nil || *current.FetchOwner != owner || current.FetchLeaseUntil == nil || *current.FetchLeaseUntil != leaseUntil {
		return false, nil
	}
	s.flights[flight.FlightID] = flight
	s.positions[position.FlightID] = append(s.positions[position.FlightID], position)
	return true, nil
}

func (s *memoryFetchFlightStore) UpdateFetchedFlightWithTrackPositions(_ context.Context, flight domain.Flight, positions []domain.FlightPosition, owner string, leaseUntil string, _ string) (bool, error) {
	if s.appendPositionErr != nil {
		return false, s.appendPositionErr
	}
	if s.stealLeaseDuringTrackTransaction {
		stolen := s.flights[flight.FlightID]
		stolen.FetchOwner = ptr("worker-2")
		stolen.FetchLeaseUntil = ptr("2026-04-29T00:10:00Z")
		s.flights[flight.FlightID] = stolen
		s.stealLeaseDuringTrackTransaction = false
		return false, nil
	}
	current := s.flights[flight.FlightID]
	if current.FetchOwner == nil || *current.FetchOwner != owner || current.FetchLeaseUntil == nil || *current.FetchLeaseUntil != leaseUntil {
		return false, nil
	}
	s.flights[flight.FlightID] = flight
	s.positions[flight.FlightID] = append(s.positions[flight.FlightID], positions...)
	return true, nil
}

func (s *memoryFetchFlightStore) FetchLeaseHeld(_ context.Context, flightID domain.FlightID, owner string, leaseUntil string) (bool, error) {
	if s.stealLeaseBeforeUpdate {
		stolen := s.flights[flightID]
		stolen.FetchOwner = ptr("worker-2")
		stolen.FetchLeaseUntil = ptr("2026-04-29T00:10:00Z")
		s.flights[flightID] = stolen
		s.stealLeaseBeforeUpdate = false
	}
	current := s.flights[flightID]
	return current.FetchOwner != nil && *current.FetchOwner == owner &&
		current.FetchLeaseUntil != nil && *current.FetchLeaseUntil == leaseUntil, nil
}

func (s *memoryFetchFlightStore) ReleaseFetchLease(_ context.Context, flightID domain.FlightID, owner string, leaseUntil string, _ string) error {
	flight := s.flights[flightID]
	if flight.FetchOwner != nil && *flight.FetchOwner == owner && flight.FetchLeaseUntil != nil && *flight.FetchLeaseUntil == leaseUntil {
		flight.FetchOwner = nil
		flight.FetchLeaseUntil = nil
	}
	s.flights[flightID] = flight
	return nil
}

func (s *memoryFetchFlightStore) PutFlight(_ context.Context, flight domain.Flight) error {
	s.flights[flight.FlightID] = flight
	return nil
}

func (s *memoryFetchFlightStore) PutFlightLookup(_ context.Context, lookupKey string, flightID domain.FlightID) error {
	flight, ok := s.flights[flightID]
	if !ok {
		return ErrNotFound
	}
	if lookupType, value, ok := strings.Cut(lookupKey, "#"); ok && lookupType == "ident" {
		for _, existing := range s.searchResults[value] {
			if existing.FlightID == flightID {
				return nil
			}
		}
		s.searchResults[value] = append(s.searchResults[value], flight)
	}
	return nil
}

func (s *memoryFetchFlightStore) AppendPosition(_ context.Context, position domain.FlightPosition) error {
	if s.appendPositionErr != nil {
		return s.appendPositionErr
	}
	s.positions[position.FlightID] = append(s.positions[position.FlightID], position)
	return nil
}

func (s *memoryFetchFlightStore) AppendPositionIfLeaseHeld(_ context.Context, position domain.FlightPosition, owner string, leaseUntil string) (bool, error) {
	if s.appendPositionErr != nil {
		return false, s.appendPositionErr
	}
	if s.stealLeaseBeforeGuardedAppend || s.stealLeaseBeforeUpdate {
		stolen := s.flights[position.FlightID]
		stolen.FetchOwner = ptr("worker-2")
		stolen.FetchLeaseUntil = ptr("2026-04-29T00:10:00Z")
		s.flights[position.FlightID] = stolen
		s.stealLeaseBeforeGuardedAppend = false
		s.stealLeaseBeforeUpdate = false
	}
	current := s.flights[position.FlightID]
	if current.FetchOwner == nil || *current.FetchOwner != owner || current.FetchLeaseUntil == nil || *current.FetchLeaseUntil != leaseUntil {
		return false, nil
	}
	s.positions[position.FlightID] = append(s.positions[position.FlightID], position)
	return true, nil
}

func (s *memoryFetchFlightStore) StoreRouteArtifact(_ context.Context, flightID domain.FlightID, response ExternalRouteResponse) (string, error) {
	s.routeCount++
	s.lastRouteArtifact = response
	key := "routes/" + string(flightID) + "/new.json"
	if _, ok := s.publishedArtifactKeys[key]; ok {
		s.overwrotePublishedArtifact = true
	}
	return key, nil
}

func (s *memoryFetchFlightStore) StoreTrackArtifact(_ context.Context, flightID domain.FlightID, response ExternalTrackResponse) (string, error) {
	s.trackCount++
	s.lastTrackArtifact = response
	key := "tracks/" + string(flightID) + "/new.json"
	if _, ok := s.publishedArtifactKeys[key]; ok {
		s.overwrotePublishedArtifact = true
	}
	return key, nil
}

func (s *memoryFetchFlightStore) RecordFetchTaskDiagnostic(_ context.Context, diagnostic FetchTaskDiagnostic) error {
	s.diagnostics = append(s.diagnostics, diagnostic)
	return nil
}

type fallbackOnlyFetchStore struct {
	inner *memoryFetchFlightStore
}

func newFallbackOnlyFetchStore(flights ...domain.Flight) *fallbackOnlyFetchStore {
	return &fallbackOnlyFetchStore{inner: newMemoryFetchFlightStore(flights...)}
}

func (s *fallbackOnlyFetchStore) AcquireFetchLease(ctx context.Context, flightID domain.FlightID, owner string, leaseUntil string, now string) (domain.Flight, bool, error) {
	return s.inner.AcquireFetchLease(ctx, flightID, owner, leaseUntil, now)
}

func (s *fallbackOnlyFetchStore) GetFlight(ctx context.Context, flightID domain.FlightID) (domain.Flight, CacheMetadata, error) {
	return s.inner.GetFlight(ctx, flightID)
}

func (s *fallbackOnlyFetchStore) UpdateFetchedFlight(ctx context.Context, flight domain.Flight, owner string, leaseUntil string, now string) (bool, error) {
	return s.inner.UpdateFetchedFlight(ctx, flight, owner, leaseUntil, now)
}

func (s *fallbackOnlyFetchStore) FetchLeaseHeld(ctx context.Context, flightID domain.FlightID, owner string, leaseUntil string) (bool, error) {
	return s.inner.FetchLeaseHeld(ctx, flightID, owner, leaseUntil)
}

func (s *fallbackOnlyFetchStore) ReleaseFetchLease(ctx context.Context, flightID domain.FlightID, owner string, leaseUntil string, now string) error {
	return s.inner.ReleaseFetchLease(ctx, flightID, owner, leaseUntil, now)
}

func (s *fallbackOnlyFetchStore) PutFlight(ctx context.Context, flight domain.Flight) error {
	return s.inner.PutFlight(ctx, flight)
}

func (s *fallbackOnlyFetchStore) PutFlightLookup(ctx context.Context, lookupKey string, flightID domain.FlightID) error {
	return s.inner.PutFlightLookup(ctx, lookupKey, flightID)
}

func (s *fallbackOnlyFetchStore) AppendPosition(ctx context.Context, position domain.FlightPosition) error {
	return s.inner.AppendPosition(ctx, position)
}

func (s *fallbackOnlyFetchStore) AppendPositionIfLeaseHeld(ctx context.Context, position domain.FlightPosition, owner string, leaseUntil string) (bool, error) {
	return s.inner.AppendPositionIfLeaseHeld(ctx, position, owner, leaseUntil)
}

func (s *fallbackOnlyFetchStore) StoreRouteArtifact(ctx context.Context, flightID domain.FlightID, response ExternalRouteResponse) (string, error) {
	return s.inner.StoreRouteArtifact(ctx, flightID, response)
}

func (s *fallbackOnlyFetchStore) StoreTrackArtifact(ctx context.Context, flightID domain.FlightID, response ExternalTrackResponse) (string, error) {
	return s.inner.StoreTrackArtifact(ctx, flightID, response)
}

type countingFlightAwareClient struct {
	search        ExternalSearchFlightsResponse
	position      ExternalPositionResponse
	route         ExternalRouteResponse
	track         ExternalTrackResponse
	err           error
	searchCalls   int
	positionCalls int
	routeCalls    int
	trackCalls    int
}

func (c *countingFlightAwareClient) SearchFlights(context.Context, string) (ExternalSearchFlightsResponse, error) {
	c.searchCalls++
	if c.err != nil {
		return ExternalSearchFlightsResponse{}, c.err
	}
	return c.search, nil
}

func (c *countingFlightAwareClient) GetFlightPosition(context.Context, string) (ExternalPositionResponse, error) {
	c.positionCalls++
	if c.err != nil {
		return ExternalPositionResponse{}, c.err
	}
	return c.position, nil
}

func (c *countingFlightAwareClient) GetFlightRoute(context.Context, string) (ExternalRouteResponse, error) {
	c.routeCalls++
	if c.err != nil {
		return ExternalRouteResponse{}, c.err
	}
	return c.route, nil
}

func (c *countingFlightAwareClient) GetFlightTrack(context.Context, string) (ExternalTrackResponse, error) {
	c.trackCalls++
	if c.err != nil {
		return ExternalTrackResponse{}, c.err
	}
	return c.track, nil
}
