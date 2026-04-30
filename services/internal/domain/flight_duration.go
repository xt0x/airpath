package domain

import "time"

type FlightDurationKind string

const (
	FlightDurationKindActual     FlightDurationKind = "actual"
	FlightDurationKindEstimated  FlightDurationKind = "estimated"
	FlightDurationKindScheduled  FlightDurationKind = "scheduled"
	FlightDurationKindFiled      FlightDurationKind = "filed"
	FlightDurationKindGateActual FlightDurationKind = "gate_actual"
)

type FlightDurationInput struct {
	Times           FlightTimes
	FiledEteSeconds *int
}

type FlightDuration struct {
	Kind    FlightDurationKind `json:"kind"`
	Seconds int                `json:"seconds"`
	StartAt *ISODateTimeString `json:"startAt"`
	EndAt   *ISODateTimeString `json:"endAt"`
}

func CalculateFlightDuration(input FlightDurationInput) *FlightDuration {
	candidates := []*FlightDuration{
		durationFromPair(FlightDurationKindActual, input.Times.ActualOff, input.Times.ActualOn),
		durationFromPair(FlightDurationKindEstimated, input.Times.EstimatedOff, input.Times.EstimatedOn),
		durationFromPair(FlightDurationKindScheduled, input.Times.ScheduledOff, input.Times.ScheduledOn),
		durationFromFiledEte(input.FiledEteSeconds),
		durationFromPair(FlightDurationKindGateActual, input.Times.ActualOut, input.Times.ActualIn),
	}

	for _, candidate := range candidates {
		if candidate != nil {
			return candidate
		}
	}

	return nil
}

func durationFromPair(kind FlightDurationKind, startAt *ISODateTimeString, endAt *ISODateTimeString) *FlightDuration {
	if startAt == nil || endAt == nil {
		return nil
	}

	normalizedStartAt, err := NormalizeUTCISODateTime(string(*startAt))
	if err != nil {
		return nil
	}
	normalizedEndAt, err := NormalizeUTCISODateTime(string(*endAt))
	if err != nil {
		return nil
	}
	parsedStartAt, err := time.Parse("2006-01-02T15:04:05Z", string(normalizedStartAt))
	if err != nil {
		return nil
	}
	parsedEndAt, err := time.Parse("2006-01-02T15:04:05Z", string(normalizedEndAt))
	if err != nil {
		return nil
	}

	seconds := int(parsedEndAt.Sub(parsedStartAt).Seconds())
	if seconds <= 0 {
		return nil
	}

	return &FlightDuration{
		Kind:    kind,
		Seconds: seconds,
		StartAt: &normalizedStartAt,
		EndAt:   &normalizedEndAt,
	}
}

func durationFromFiledEte(filedEteSeconds *int) *FlightDuration {
	if filedEteSeconds == nil || *filedEteSeconds <= 0 {
		return nil
	}

	return &FlightDuration{
		Kind:    FlightDurationKindFiled,
		Seconds: *filedEteSeconds,
	}
}
