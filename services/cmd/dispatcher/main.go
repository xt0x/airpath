package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"airpath/services/internal/application"
	"airpath/services/internal/awsintegration"
	"airpath/services/internal/runtimewiring"
	"github.com/aws/aws-lambda-go/lambda"
)

type runtimeDependencies = runtimewiring.Dependencies

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

func handleDispatcherEvent(json.RawMessage) (dispatcherResponse, error) {
	response := dispatcherResponse{
		Service:          "dispatcher",
		Environment:      os.Getenv("AIRPATH_ENVIRONMENT"),
		Mode:             os.Getenv("DISPATCHER_MODE"),
		NoopFetchEnabled: strings.EqualFold(os.Getenv("NOOP_FETCH_ENABLED"), "true"),
		QueueConfigured:  os.Getenv("FETCH_TASK_QUEUE_URL") != "",
		EnqueueAttempted: false,
	}
	if response.NoopFetchEnabled {
		return response, nil
	}

	ctx := context.Background()
	dependencies, err := runtimewiring.NewDependencies(ctx, runtimeBackend(os.Getenv))
	if err != nil {
		return dispatcherResponse{}, err
	}
	dispatcher, err := newPollingDispatcher(ctx, dependencies)
	if err != nil {
		return dispatcherResponse{}, err
	}
	result, err := dispatcher.Dispatch(ctx, application.DispatchPollInput{
		Now:   time.Now().UTC(),
		Limit: 25,
	})
	if err != nil {
		return dispatcherResponse{}, err
	}
	response.EnqueueAttempted = result.EnqueuedTasks > 0
	return response, nil
}

func buildPollingDispatcher(_ context.Context, dependencies runtimeDependencies) (pollingDispatcher, error) {
	tables := awsintegration.DynamoDBTables{
		Flights:         envOrDefault("FLIGHTS_TABLE_NAME", "flights"),
		FlightLookup:    envOrDefault("FLIGHT_LOOKUP_TABLE_NAME", "flight-lookup"),
		FlightPositions: envOrDefault("FLIGHT_POSITIONS_TABLE_NAME", "flight-positions"),
		UsageBudget:     envOrDefault("USAGE_BUDGET_TABLE_NAME", "usage-budget"),
	}
	repository := awsintegration.NewScopedDynamoDBRepository(dependencies.DynamoDB, tables, usageScope())
	return application.NewPollingDispatcher(application.PollingDispatcherConfig{
		Flights:    repository,
		FetchTasks: awsintegration.NewSQSFetchTaskQueue(dependencies.Queues, envOrDefault("FETCH_TASK_QUEUE_URL", "memory")),
		UsageGuard: repository,
		Schedule:   application.DefaultPollSchedulePolicy(),
	}), nil
}

func runtimeBackend(lookup func(string) string) runtimewiring.Backend {
	if explicit := strings.ToLower(strings.TrimSpace(lookup("AIRPATH_RUNTIME_BACKEND"))); explicit == string(runtimewiring.BackendMemory) {
		return runtimewiring.BackendMemory
	}
	if lookup("AWS_LAMBDA_FUNCTION_NAME") == "" {
		return runtimewiring.BackendMemory
	}
	return runtimewiring.BackendAWS
}

func usageScope() application.UsageBudgetScope {
	return application.UsageBudgetScope{
		Environment: envOrDefault("AIRPATH_ENVIRONMENT", "local"),
		Month:       time.Now().UTC().Format("2006-01"),
	}
}

func envOrDefault(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
