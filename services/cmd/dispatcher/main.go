package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"airpath/services/internal/application"
	"airpath/services/internal/cmdsupport"
	"airpath/services/internal/runtimewiring"
	"github.com/aws/aws-lambda-go/lambda"
)

type runtimeDependencies = runtimewiring.Dependencies

// errFetchTaskQueueNotConfigured fails fast in AWS so EventBridge invocations
// cannot silently drop due polling work when queue wiring is missing.
var errFetchTaskQueueNotConfigured = errors.New("fetch task queue url is required in aws runtime")

// pollingDispatcher is the application use-case boundary needed by this Lambda.
type pollingDispatcher interface {
	Dispatch(context.Context, application.DispatchPollInput) (application.DispatchPollResult, error)
}

var newPollingDispatcher = buildPollingDispatcher

type dispatcherResponse struct {
	Service          string `json:"service"`
	Environment      string `json:"environment"`
	Mode             string `json:"mode"`
	NoopFetchEnabled bool   `json:"noopFetchEnabled"`
	QueueConfigured  bool   `json:"queueConfigured"`
	EnqueueAttempted bool   `json:"enqueueAttempted"`
}

func main() {
	lambda.Start(handleDispatcherEvent)
}

// handleDispatcherEvent ignores the EventBridge payload because dispatching is
// based on persisted poll schedules and environment-driven runtime mode.
func handleDispatcherEvent(ctx context.Context, _ json.RawMessage) (dispatcherResponse, error) {
	response := dispatcherResponse{
		Service:          "dispatcher",
		Environment:      os.Getenv("AIRPATH_ENVIRONMENT"),
		Mode:             os.Getenv("DISPATCHER_MODE"),
		NoopFetchEnabled: strings.EqualFold(os.Getenv("NOOP_FETCH_ENABLED"), "true"),
		QueueConfigured:  strings.TrimSpace(os.Getenv("FETCH_TASK_QUEUE_URL")) != "",
		EnqueueAttempted: false,
	}
	if response.NoopFetchEnabled {
		return response, nil
	}
	backend := cmdsupport.RuntimeBackend(os.Getenv)
	if backend == runtimewiring.BackendAWS && !response.QueueConfigured {
		// The memory backend can run local tests without SQS, but AWS must have a
		// queue URL before any dispatch work is attempted.
		return response, errFetchTaskQueueNotConfigured
	}

	dependencies, err := runtimewiring.NewDependencies(ctx, backend)
	if err != nil {
		return dispatcherResponse{}, err
	}
	dispatcher, err := newPollingDispatcher(ctx, dependencies)
	if err != nil {
		return dispatcherResponse{}, err
	}
	result, err := dispatcher.Dispatch(ctx, application.DispatchPollInput{
		Now:   cmdsupport.NowUTC(),
		Limit: 25,
	})
	if err != nil {
		return dispatcherResponse{}, err
	}
	response.EnqueueAttempted = result.EnqueuedTasks > 0
	return response, nil
}

func buildPollingDispatcher(_ context.Context, dependencies runtimeDependencies) (pollingDispatcher, error) {
	return runtimewiring.NewPollingDispatcher(dependencies, os.Getenv, cmdsupport.NowUTC()), nil
}
