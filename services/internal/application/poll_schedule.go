package application

import (
	"time"

	"airpath/services/internal/domain"
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
		// The first inactive observation starts a grace period. Polls are stopped
		// only after the flight remains inactive across the idle window.
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

func formatISO(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func isoPtr(value time.Time) *domain.ISODateTimeString {
	formatted := formatISO(value)
	return &formatted
}
