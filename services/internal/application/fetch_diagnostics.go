package application

import (
	"context"
	"errors"
	"time"

	"airpath/services/internal/domain"
)

type FetchTaskDiagnosticStore interface {
	RecordFetchTaskDiagnostic(context.Context, FetchTaskDiagnostic) error
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

func fetchErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrUpstreamRateLimited):
		return "rate_limited"
	case errors.Is(err, ErrUpstreamFetchDisabled):
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
