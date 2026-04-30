package application

import (
	"context"
	"time"

	"airpath/services/internal/domain"
)

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
	mustProvide("PollingDispatcherConfig.Flights", config.Flights)
	mustProvide("PollingDispatcherConfig.FetchTasks", config.FetchTasks)
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
		tasks := dueTasks(flight, input.Now)
		signal := PollActivitySignal{ActiveViewer: true}
		shouldRefreshSchedule := false
		if d.activities != nil {
			signal, err = d.activities.ActivityForFlight(ctx, flight.FlightID)
			if err != nil {
				return DispatchPollResult{}, err
			}
			if !signal.activeAt(input.Now, d.schedule.ManualKeepAlive) {
				updated := d.schedule.Compute(flight, input.Now, signal)
				if err := d.flights.UpdatePollSchedule(ctx, updated); err != nil {
					return DispatchPollResult{}, err
				}
				continue
			}
			shouldRefreshSchedule = true
		}
		if len(tasks) == 0 {
			if shouldRefreshSchedule {
				updated := d.schedule.Compute(flight, input.Now, signal)
				if err := d.flights.UpdatePollSchedule(ctx, updated); err != nil {
					return DispatchPollResult{}, err
				}
			}
			continue
		}
		result.DueFlights++
		blockedTaskTypes := map[FetchTaskType]struct{}{}
		dispatchedTask := false
		for _, dueTask := range tasks {
			allowed, err := d.taskAllowed(ctx, dueTask.taskType)
			if err != nil {
				return DispatchPollResult{}, err
			}
			if !allowed {
				blockedTaskTypes[dueTask.taskType] = struct{}{}
				continue
			}
			taskID := stableTaskID("poll", flight.FlightID, dueTask.taskType, dueTask.dueAt)
			task := FetchTask{
				SchemaVersion:  1,
				TaskID:         taskID,
				TaskType:       dueTask.taskType,
				FlightID:       flight.FlightID,
				FAFlightID:     flight.FAFlightID,
				RequestedAt:    nowString,
				Reason:         FetchReasonLowFrequencyPoll,
				IdempotencyKey: taskID,
				NotBefore:      ptr(dueTask.dueAt),
			}
			enqueued, err := d.fetchTasks.EnqueueFetchTask(ctx, task)
			if err != nil {
				return DispatchPollResult{}, err
			}
			dispatchedTask = true
			if enqueued {
				result.EnqueuedTasks++
			}
		}
		if shouldRefreshSchedule && (len(blockedTaskTypes) == 0 || dispatchedTask) {
			updated := d.schedule.Compute(flight, input.Now, signal)
			// Runtime policy may temporarily block only some task types. Preserve
			// their due timestamps so they retry when the policy is enabled again.
			updated = preserveBlockedPollTimes(updated, flight, blockedTaskTypes)
			if err := d.flights.UpdatePollSchedule(ctx, updated); err != nil {
				return DispatchPollResult{}, err
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

func preserveBlockedPollTimes(updated domain.Flight, original domain.Flight, blockedTaskTypes map[FetchTaskType]struct{}) domain.Flight {
	for taskType := range blockedTaskTypes {
		switch taskType {
		case FetchTaskPosition:
			updated.NextPositionPollAt = original.NextPositionPollAt
		case FetchTaskRoute:
			updated.NextRoutePollAt = original.NextRoutePollAt
		case FetchTaskTrack:
			updated.NextTrackPollAt = original.NextTrackPollAt
		}
	}
	return updated
}

type dueFetchTask struct {
	taskType FetchTaskType
	dueAt    string
}

func dueTasks(flight domain.Flight, now time.Time) []dueFetchTask {
	tasks := []dueFetchTask{}
	if dueAt(flight.NextPositionPollAt, now) {
		tasks = append(tasks, dueFetchTask{taskType: FetchTaskPosition, dueAt: *flight.NextPositionPollAt})
	}
	if dueAt(flight.NextRoutePollAt, now) {
		tasks = append(tasks, dueFetchTask{taskType: FetchTaskRoute, dueAt: *flight.NextRoutePollAt})
	}
	if dueAt(flight.NextTrackPollAt, now) {
		tasks = append(tasks, dueFetchTask{taskType: FetchTaskTrack, dueAt: *flight.NextTrackPollAt})
	}
	return tasks
}
