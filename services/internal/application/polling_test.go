package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"airpath/services/internal/domain"
	"airpath/services/internal/flightaware"
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

func TestFetchProcessorUsesFlightLeaseToPreventDuplicateFlightAwareCalls(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_lease", "ANA110")
	flight.FetchLeaseUntil = ptr("2026-04-29T00:05:00Z")
	flight.FetchOwner = ptr("other-worker")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{
		position: flightaware.PositionResponse{
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

func TestFetchProcessorUpdatesLatestPositionAndHistory(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_position", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{
		position: flightaware.PositionResponse{
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

func TestFetchProcessorStoresRouteAndTrackOnlyForExplicitTasks(t *testing.T) {
	ctx := context.Background()
	now := mustTime(t, "2026-04-29T00:00:00Z")
	flight := testFlight("iflg_layers", "ANA110")
	store := newMemoryFetchFlightStore(flight)
	client := &countingFlightAwareClient{
		route: flightaware.RouteResponse{Fixes: []flightaware.RouteFix{
			{Name: "RJTT", Latitude: ptrFloat64(35.55), Longitude: ptrFloat64(139.78)},
			{Name: "KJFK", Latitude: ptrFloat64(40.64), Longitude: ptrFloat64(-73.78)},
		}},
		track: flightaware.TrackResponse{Positions: []flightaware.TrackPoint{
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
	client := &countingFlightAwareClient{err: flightaware.NewRateLimitedError(flightaware.EndpointPosition, 429, "limited")}
	processor := NewFetchProcessor(FetchProcessorConfig{
		Flights:     store,
		Positions:   store,
		Artifacts:   store,
		FlightAware: client,
		Diagnostics: store,
	})

	_, err := processor.Process(ctx, taskFor(flight, FetchTaskPosition), ProcessFetchInput{Now: now, WorkerID: "worker-1"})
	if !errors.Is(err, flightaware.ErrFlightAwareRateLimited) {
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

type memoryFetchFlightStore struct {
	flights     map[domain.FlightID]domain.Flight
	positions   map[domain.FlightID][]domain.FlightPosition
	diagnostics []FetchTaskDiagnostic
	routeCount  int
	trackCount  int
}

func newMemoryFetchFlightStore(flights ...domain.Flight) *memoryFetchFlightStore {
	store := &memoryFetchFlightStore{
		flights:   map[domain.FlightID]domain.Flight{},
		positions: map[domain.FlightID][]domain.FlightPosition{},
	}
	for _, flight := range flights {
		store.flights[flight.FlightID] = flight
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

func (s *memoryFetchFlightStore) UpdateFetchedFlight(_ context.Context, flight domain.Flight) error {
	s.flights[flight.FlightID] = flight
	return nil
}

func (s *memoryFetchFlightStore) ReleaseFetchLease(_ context.Context, flightID domain.FlightID, owner string, _ string) error {
	flight := s.flights[flightID]
	if flight.FetchOwner != nil && *flight.FetchOwner == owner {
		flight.FetchOwner = nil
		flight.FetchLeaseUntil = nil
	}
	s.flights[flightID] = flight
	return nil
}

func (s *memoryFetchFlightStore) AppendPosition(_ context.Context, position domain.FlightPosition) error {
	s.positions[position.FlightID] = append(s.positions[position.FlightID], position)
	return nil
}

func (s *memoryFetchFlightStore) StoreFlightAwareRoute(_ context.Context, flightID domain.FlightID, _ flightaware.RouteResponse) (string, error) {
	s.routeCount++
	return "routes/" + string(flightID) + ".json", nil
}

func (s *memoryFetchFlightStore) StoreFlightAwareTrack(_ context.Context, flightID domain.FlightID, _ flightaware.TrackResponse) (string, error) {
	s.trackCount++
	return "tracks/" + string(flightID) + ".json", nil
}

func (s *memoryFetchFlightStore) RecordFetchTaskDiagnostic(_ context.Context, diagnostic FetchTaskDiagnostic) error {
	s.diagnostics = append(s.diagnostics, diagnostic)
	return nil
}

type countingFlightAwareClient struct {
	position      flightaware.PositionResponse
	route         flightaware.RouteResponse
	track         flightaware.TrackResponse
	err           error
	positionCalls int
	routeCalls    int
	trackCalls    int
}

func (c *countingFlightAwareClient) GetFlightPosition(context.Context, flightaware.FlightPositionRequest) (flightaware.PositionResponse, error) {
	c.positionCalls++
	if c.err != nil {
		return flightaware.PositionResponse{}, c.err
	}
	return c.position, nil
}

func (c *countingFlightAwareClient) GetFlightRoute(context.Context, flightaware.FlightRouteRequest) (flightaware.RouteResponse, error) {
	c.routeCalls++
	if c.err != nil {
		return flightaware.RouteResponse{}, c.err
	}
	return c.route, nil
}

func (c *countingFlightAwareClient) GetFlightTrack(context.Context, flightaware.FlightTrackRequest) (flightaware.TrackResponse, error) {
	c.trackCalls++
	if c.err != nil {
		return flightaware.TrackResponse{}, c.err
	}
	return c.track, nil
}
