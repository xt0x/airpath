package application

import (
	"context"
	"errors"
	"time"

	"airpath/services/internal/domain"
	"airpath/services/internal/flightaware"
)

const (
	defaultPositionPollInterval = 15 * time.Minute
	defaultTrackPollInterval    = 45 * time.Minute
	defaultRoutePollInterval    = 6 * time.Hour
	defaultManualKeepAlive      = 30 * time.Minute
	defaultIdleStopAfter        = 10 * time.Minute
	defaultFetchLeaseDuration   = 5 * time.Minute
)

type PollSchedulePolicy struct {
	PositionPollInterval time.Duration
	TrackPollInterval    time.Duration
	RoutePollInterval    time.Duration
	ManualKeepAlive      time.Duration
	IdleStopAfter        time.Duration
}

type PollActivitySignal struct {
	ActiveViewer        bool
	LastManualRequestAt *time.Time
	AllowRouteWindow    bool
	AllowTrackWindow    bool
}

func DefaultPollSchedulePolicy() PollSchedulePolicy {
	return PollSchedulePolicy{
		PositionPollInterval: defaultPositionPollInterval,
		TrackPollInterval:    defaultTrackPollInterval,
		RoutePollInterval:    defaultRoutePollInterval,
		ManualKeepAlive:      defaultManualKeepAlive,
		IdleStopAfter:        defaultIdleStopAfter,
	}
}

func (p PollSchedulePolicy) Compute(flight domain.Flight, now time.Time, signal PollActivitySignal) domain.Flight {
	p = p.withDefaults()
	if !signal.activeAt(now, p.ManualKeepAlive) {
		return p.computeIdle(flight, now)
	}

	activeState := domain.FlightPollStateActive
	flight.PollState = &activeState
	flight.IdleSince = nil
	flight.NextPositionPollAt = isoPtr(now.Add(p.PositionPollInterval))
	if flight.PlannedRouteS3Key == nil || signal.AllowRouteWindow {
		flight.NextRoutePollAt = isoPtr(now.Add(p.RoutePollInterval))
	} else {
		flight.NextRoutePollAt = nil
	}
	if flight.ActualTrackS3Key == nil || signal.AllowTrackWindow {
		flight.NextTrackPollAt = isoPtr(now.Add(p.TrackPollInterval))
	} else {
		flight.NextTrackPollAt = nil
	}
	return flight
}

func (p PollSchedulePolicy) withDefaults() PollSchedulePolicy {
	if p.PositionPollInterval == 0 {
		p.PositionPollInterval = defaultPositionPollInterval
	}
	if p.TrackPollInterval == 0 {
		p.TrackPollInterval = defaultTrackPollInterval
	}
	if p.RoutePollInterval == 0 {
		p.RoutePollInterval = defaultRoutePollInterval
	}
	if p.ManualKeepAlive == 0 {
		p.ManualKeepAlive = defaultManualKeepAlive
	}
	if p.IdleStopAfter == 0 {
		p.IdleStopAfter = defaultIdleStopAfter
	}
	return p
}

func (p PollSchedulePolicy) computeIdle(flight domain.Flight, now time.Time) domain.Flight {
	if flight.IdleSince == nil {
		flight.IdleSince = isoPtr(now)
		return flight
	}
	idleSince, err := time.Parse(time.RFC3339, *flight.IdleSince)
	if err != nil {
		flight.IdleSince = isoPtr(now)
		return flight
	}
	if now.Sub(idleSince) < p.IdleStopAfter {
		return flight
	}
	completedState := domain.FlightPollStateCompleted
	flight.PollState = &completedState
	flight.NextPositionPollAt = nil
	flight.NextTrackPollAt = nil
	flight.NextRoutePollAt = nil
	return flight
}

func (s PollActivitySignal) activeAt(now time.Time, manualKeepAlive time.Duration) bool {
	if s.ActiveViewer {
		return true
	}
	if s.LastManualRequestAt == nil {
		return false
	}
	return !s.LastManualRequestAt.After(now) && now.Sub(*s.LastManualRequestAt) <= manualKeepAlive
}

type PollFlightStore interface {
	ListPollableFlights(context.Context, string, int) ([]domain.Flight, error)
	UpdatePollSchedule(context.Context, domain.Flight) error
}

type PollActivityStore interface {
	ActivityForFlight(context.Context, domain.FlightID) (PollActivitySignal, error)
}

type PollingDispatcherConfig struct {
	Flights     PollFlightStore
	Activities  PollActivityStore
	FetchTasks  FetchTaskQueue
	UsageGuard  UsageGuard
	FetchPolicy FetchPolicy
	Schedule    PollSchedulePolicy
}

type PollingDispatcher struct {
	flights     PollFlightStore
	activities  PollActivityStore
	fetchTasks  FetchTaskQueue
	usageGuard  UsageGuard
	fetchPolicy FetchPolicy
	schedule    PollSchedulePolicy
}

type DispatchPollInput struct {
	Now   time.Time
	Limit int
}

type DispatchPollResult struct {
	Stopped       bool
	DueFlights    int
	EnqueuedTasks int
}

func NewPollingDispatcher(config PollingDispatcherConfig) *PollingDispatcher {
	return &PollingDispatcher{
		flights:     config.Flights,
		activities:  config.Activities,
		fetchTasks:  config.FetchTasks,
		usageGuard:  config.UsageGuard,
		fetchPolicy: config.FetchPolicy,
		schedule:    config.Schedule.withDefaults(),
	}
}

func (d *PollingDispatcher) Dispatch(ctx context.Context, input DispatchPollInput) (DispatchPollResult, error) {
	if d.usageGuard != nil {
		allowed, err := d.usageGuard.FetchingAllowed(ctx)
		if err != nil {
			return DispatchPollResult{}, err
		}
		if !allowed {
			return DispatchPollResult{Stopped: true}, nil
		}
	}
	nowString := formatISO(input.Now)
	flights, err := d.flights.ListPollableFlights(ctx, nowString, input.Limit)
	if err != nil {
		return DispatchPollResult{}, err
	}
	result := DispatchPollResult{}
	for _, flight := range flights {
		signal := PollActivitySignal{ActiveViewer: true}
		if d.activities != nil {
			signal, err = d.activities.ActivityForFlight(ctx, flight.FlightID)
			if err != nil {
				return DispatchPollResult{}, err
			}
			updated := d.schedule.Compute(flight, input.Now, signal)
			if err := d.flights.UpdatePollSchedule(ctx, updated); err != nil {
				return DispatchPollResult{}, err
			}
			flight = updated
		}
		tasks := dueTasks(flight, input.Now)
		if len(tasks) == 0 {
			continue
		}
		result.DueFlights++
		for _, taskType := range tasks {
			allowed, err := d.taskAllowed(ctx, taskType)
			if err != nil {
				return DispatchPollResult{}, err
			}
			if !allowed {
				continue
			}
			task := FetchTask{
				SchemaVersion:  1,
				TaskID:         stableTaskID("poll", flight.FlightID, taskType, nowString),
				TaskType:       taskType,
				FlightID:       flight.FlightID,
				FAFlightID:     flight.FAFlightID,
				RequestedAt:    nowString,
				Reason:         FetchReasonLowFrequencyPoll,
				IdempotencyKey: stableTaskID("poll", flight.FlightID, taskType, nowString),
				NotBefore:      ptr(nowString),
			}
			enqueued, err := d.fetchTasks.EnqueueFetchTask(ctx, task)
			if err != nil {
				return DispatchPollResult{}, err
			}
			if enqueued {
				result.EnqueuedTasks++
			}
		}
	}
	return result, nil
}

func (d *PollingDispatcher) taskAllowed(ctx context.Context, taskType FetchTaskType) (bool, error) {
	if d.fetchPolicy == nil {
		return true, nil
	}
	return d.fetchPolicy.FetchTaskAllowed(ctx, taskType, FetchReasonLowFrequencyPoll)
}

func dueTasks(flight domain.Flight, now time.Time) []FetchTaskType {
	tasks := []FetchTaskType{}
	if dueAt(flight.NextPositionPollAt, now) {
		tasks = append(tasks, FetchTaskPosition)
	}
	if dueAt(flight.NextRoutePollAt, now) {
		tasks = append(tasks, FetchTaskRoute)
	}
	if dueAt(flight.NextTrackPollAt, now) {
		tasks = append(tasks, FetchTaskTrack)
	}
	return tasks
}

func dueAt(value *domain.ISODateTimeString, now time.Time) bool {
	if value == nil {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return false
	}
	return !parsed.After(now)
}

type FetchFlightStore interface {
	AcquireFetchLease(context.Context, domain.FlightID, string, string, string) (domain.Flight, bool, error)
	UpdateFetchedFlight(context.Context, domain.Flight) error
	ReleaseFetchLease(context.Context, domain.FlightID, string, string) error
}

type PositionHistoryStore interface {
	AppendPosition(context.Context, domain.FlightPosition) error
}

type FlightArtifactStore interface {
	StoreFlightAwareRoute(context.Context, domain.FlightID, flightaware.RouteResponse) (string, error)
	StoreFlightAwareTrack(context.Context, domain.FlightID, flightaware.TrackResponse) (string, error)
}

type FetchTaskDiagnosticStore interface {
	RecordFetchTaskDiagnostic(context.Context, FetchTaskDiagnostic) error
}

type FetchFlightAwareClient interface {
	GetFlightRoute(context.Context, flightaware.FlightRouteRequest) (flightaware.RouteResponse, error)
	GetFlightPosition(context.Context, flightaware.FlightPositionRequest) (flightaware.PositionResponse, error)
	GetFlightTrack(context.Context, flightaware.FlightTrackRequest) (flightaware.TrackResponse, error)
}

type FetchTaskDiagnostic struct {
	TaskID     string          `json:"taskId"`
	TaskType   FetchTaskType   `json:"taskType"`
	FlightID   domain.FlightID `json:"flightId"`
	FAFlightID string          `json:"faFlightId,omitempty"`
	ErrorCode  string          `json:"errorCode"`
	Message    string          `json:"message"`
	FailedAt   string          `json:"failedAt"`
}

type FetchProcessorConfig struct {
	Flights       FetchFlightStore
	Positions     PositionHistoryStore
	Artifacts     FlightArtifactStore
	FlightAware   FetchFlightAwareClient
	Diagnostics   FetchTaskDiagnosticStore
	Schedule      PollSchedulePolicy
	LeaseDuration time.Duration
}

type FetchProcessor struct {
	flights       FetchFlightStore
	positions     PositionHistoryStore
	artifacts     FlightArtifactStore
	flightAware   FetchFlightAwareClient
	diagnostics   FetchTaskDiagnosticStore
	schedule      PollSchedulePolicy
	leaseDuration time.Duration
}

type ProcessFetchInput struct {
	Now      time.Time
	WorkerID string
}

type ProcessFetchResult struct {
	ExternalFetchAttempted bool
	Skipped                bool
	SkipReason             string
	UpdatedPositionCount   int
	UpdatedRoute           bool
	UpdatedTrack           bool
}

func NewFetchProcessor(config FetchProcessorConfig) *FetchProcessor {
	leaseDuration := config.LeaseDuration
	if leaseDuration == 0 {
		leaseDuration = defaultFetchLeaseDuration
	}
	return &FetchProcessor{
		flights:       config.Flights,
		positions:     config.Positions,
		artifacts:     config.Artifacts,
		flightAware:   config.FlightAware,
		diagnostics:   config.Diagnostics,
		schedule:      config.Schedule.withDefaults(),
		leaseDuration: leaseDuration,
	}
}

func (p *FetchProcessor) Process(ctx context.Context, task FetchTask, input ProcessFetchInput) (ProcessFetchResult, error) {
	workerID := input.WorkerID
	if workerID == "" {
		workerID = "fetcher"
	}
	nowString := formatISO(input.Now)
	leaseUntil := formatISO(input.Now.Add(p.leaseDuration))
	flight, leased, err := p.flights.AcquireFetchLease(ctx, task.FlightID, workerID, leaseUntil, nowString)
	if err != nil {
		return ProcessFetchResult{}, err
	}
	if !leased {
		return ProcessFetchResult{Skipped: true, SkipReason: "lease_held"}, nil
	}
	defer func() {
		_ = p.flights.ReleaseFetchLease(ctx, task.FlightID, workerID, nowString)
	}()

	if flight.FAFlightID == nil || *flight.FAFlightID == "" {
		err := ErrValidation
		_ = p.recordDiagnostic(ctx, task, input.Now, err)
		return ProcessFetchResult{}, err
	}

	result := ProcessFetchResult{ExternalFetchAttempted: true}
	switch task.TaskType {
	case FetchTaskPosition:
		result, err = p.processPosition(ctx, flight, task, input.Now, result)
	case FetchTaskRoute:
		result, err = p.processRoute(ctx, flight, task, input.Now, result)
	case FetchTaskTrack, FetchTaskFinalTrack:
		result, err = p.processTrack(ctx, flight, task, input.Now, result)
	default:
		err = ErrValidation
	}
	if err != nil {
		_ = p.recordDiagnostic(ctx, task, input.Now, err)
	}
	return result, err
}

func (p *FetchProcessor) processPosition(ctx context.Context, flight domain.Flight, task FetchTask, now time.Time, result ProcessFetchResult) (ProcessFetchResult, error) {
	response, err := p.flightAware.GetFlightPosition(ctx, flightaware.FlightPositionRequest{FAFlightID: string(*flight.FAFlightID)})
	if err != nil {
		return result, err
	}
	position, ok := positionFromFlightAware(flight, response)
	if !ok {
		return result, ErrValidation
	}
	if err := p.positions.AppendPosition(ctx, position); err != nil {
		return result, err
	}
	flight.LatestPositionTimestamp = &position.Timestamp
	flight.LatestPositionSource = &position.Source
	flight = p.schedule.Compute(flight, now, PollActivitySignal{ActiveViewer: true})
	if err := p.flights.UpdateFetchedFlight(ctx, flight); err != nil {
		return result, err
	}
	result.UpdatedPositionCount = 1
	_ = task
	return result, nil
}

func (p *FetchProcessor) processRoute(ctx context.Context, flight domain.Flight, task FetchTask, now time.Time, result ProcessFetchResult) (ProcessFetchResult, error) {
	response, err := p.flightAware.GetFlightRoute(ctx, flightaware.FlightRouteRequest{FAFlightID: string(*flight.FAFlightID)})
	if err != nil {
		return result, err
	}
	key, err := p.artifacts.StoreFlightAwareRoute(ctx, flight.FlightID, response)
	if err != nil {
		return result, err
	}
	flight.PlannedRouteS3Key = &key
	flight = p.schedule.Compute(flight, now, PollActivitySignal{ActiveViewer: true})
	if err := p.flights.UpdateFetchedFlight(ctx, flight); err != nil {
		return result, err
	}
	result.UpdatedRoute = true
	_ = task
	return result, nil
}

func (p *FetchProcessor) processTrack(ctx context.Context, flight domain.Flight, task FetchTask, now time.Time, result ProcessFetchResult) (ProcessFetchResult, error) {
	response, err := p.flightAware.GetFlightTrack(ctx, flightaware.FlightTrackRequest{FAFlightID: string(*flight.FAFlightID)})
	if err != nil {
		return result, err
	}
	key, err := p.artifacts.StoreFlightAwareTrack(ctx, flight.FlightID, response)
	if err != nil {
		return result, err
	}
	for _, point := range response.Positions {
		position := domain.FlightPosition{
			FlightID:   flight.FlightID,
			FAFlightID: flight.FAFlightID,
			Latitude:   point.Latitude,
			Longitude:  point.Longitude,
			Timestamp:  point.Timestamp,
			Source:     domain.PositionSourceFlightAwareTrack,
		}
		if err := p.positions.AppendPosition(ctx, position); err != nil {
			return result, err
		}
		result.UpdatedPositionCount++
		flight.LatestPositionTimestamp = &position.Timestamp
		flight.LatestPositionSource = &position.Source
	}
	flight.ActualTrackS3Key = &key
	flight = p.schedule.Compute(flight, now, PollActivitySignal{ActiveViewer: true})
	if err := p.flights.UpdateFetchedFlight(ctx, flight); err != nil {
		return result, err
	}
	result.UpdatedTrack = true
	_ = task
	return result, nil
}

func (p *FetchProcessor) recordDiagnostic(ctx context.Context, task FetchTask, failedAt time.Time, err error) error {
	if p.diagnostics == nil {
		return nil
	}
	return p.diagnostics.RecordFetchTaskDiagnostic(ctx, FetchTaskDiagnostic{
		TaskID:    task.TaskID,
		TaskType:  task.TaskType,
		FlightID:  task.FlightID,
		ErrorCode: fetchErrorCode(err),
		Message:   safeErrorMessage(err),
		FailedAt:  formatISO(failedAt),
	})
}

func positionFromFlightAware(flight domain.Flight, response flightaware.PositionResponse) (domain.FlightPosition, bool) {
	if response.Latitude == nil || response.Longitude == nil || response.Timestamp == "" {
		return domain.FlightPosition{}, false
	}
	return domain.FlightPosition{
		FlightID:   flight.FlightID,
		FAFlightID: flight.FAFlightID,
		Latitude:   *response.Latitude,
		Longitude:  *response.Longitude,
		Timestamp:  response.Timestamp,
		Source:     domain.PositionSourceFlightAwarePosition,
	}, true
}

func fetchErrorCode(err error) string {
	switch {
	case errors.Is(err, flightaware.ErrFlightAwareRateLimited):
		return "rate_limited"
	case errors.Is(err, flightaware.ErrFlightAwareFetchDisabled):
		return "fetch_disabled"
	case errors.Is(err, ErrValidation):
		return "validation_failed"
	default:
		return "upstream_failure"
	}
}

func safeErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	code := fetchErrorCode(err)
	switch code {
	case "rate_limited":
		return "FlightAware rate limit is active"
	case "fetch_disabled":
		return "FlightAware fetch is disabled"
	case "validation_failed":
		return "Fetch task validation failed"
	default:
		return "FlightAware fetch failed"
	}
}

func formatISO(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func isoPtr(value time.Time) *domain.ISODateTimeString {
	formatted := formatISO(value)
	return &formatted
}
