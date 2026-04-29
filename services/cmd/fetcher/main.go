package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"airpath/services/internal/application"
	"airpath/services/internal/awsintegration"
	"airpath/services/internal/flightaware"
	"airpath/services/internal/runtimeconfig"
	"airpath/services/internal/runtimewiring"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type runtimeDependencies = runtimewiring.Dependencies

type fetchProcessor interface {
	Process(context.Context, application.FetchTask, application.ProcessFetchInput) (application.ProcessFetchResult, error)
}

var newFetchProcessor = buildFetchProcessor

type fetcherResponse struct {
	Service                     string `json:"service"`
	Environment                 string `json:"environment"`
	Mode                        string `json:"mode"`
	RecordsReceived             int    `json:"recordsReceived"`
	FlightAwareSecretReady      bool   `json:"flightawareSecretReady"`
	FlightAwareFetchEnabled     bool   `json:"flightawareFetchEnabled"`
	FlightAwareRealCallsEnabled bool   `json:"flightawareRealCallsEnabled"`
	ExternalFetchAllowed        bool   `json:"externalFetchAllowed"`
	ExternalFetchAttempted      bool   `json:"externalFetchAttempted"`
}

func main() {
	lambda.Start(handleFetchEvent)
}

func handleFetchEvent(event events.SQSEvent) (fetcherResponse, error) {
	config, err := runtimeconfig.LoadFlightAwareRuntimeConfig(os.Getenv)
	if err != nil {
		return fetcherResponse{}, err
	}
	response := fetcherResponse{
		Service:                     "fetcher",
		Environment:                 config.Environment,
		Mode:                        os.Getenv("FETCHER_MODE"),
		RecordsReceived:             len(event.Records),
		FlightAwareSecretReady:      os.Getenv("FLIGHTAWARE_API_KEY_SECRET_ARN") != "",
		FlightAwareFetchEnabled:     config.FetchEnabled,
		FlightAwareRealCallsEnabled: config.RealCallsEnabled,
		ExternalFetchAllowed:        config.ExternalCallsAllowed(),
		ExternalFetchAttempted:      false,
	}
	if !config.ExternalCallsAllowed() || len(event.Records) == 0 {
		return response, nil
	}

	ctx := context.Background()
	dependencies, err := runtimewiring.NewDependencies(ctx, runtimeBackend(os.Getenv))
	if err != nil {
		return fetcherResponse{}, err
	}
	processor, err := newFetchProcessor(ctx, dependencies)
	if err != nil {
		return fetcherResponse{}, err
	}
	for _, record := range event.Records {
		var task application.FetchTask
		if err := json.Unmarshal([]byte(record.Body), &task); err != nil {
			return fetcherResponse{}, err
		}
		result, err := processor.Process(ctx, task, application.ProcessFetchInput{
			Now:      time.Now().UTC(),
			WorkerID: "fetcher",
		})
		if err != nil {
			return fetcherResponse{}, err
		}
		response.ExternalFetchAttempted = response.ExternalFetchAttempted || result.ExternalFetchAttempted
	}
	return response, nil
}

func buildFetchProcessor(ctx context.Context, dependencies runtimeDependencies) (fetchProcessor, error) {
	tables := awsintegration.DynamoDBTables{
		Flights:         envOrDefault("FLIGHTS_TABLE_NAME", "flights"),
		FlightLookup:    envOrDefault("FLIGHT_LOOKUP_TABLE_NAME", "flight-lookup"),
		FlightPositions: envOrDefault("FLIGHT_POSITIONS_TABLE_NAME", "flight-positions"),
		UsageBudget:     envOrDefault("USAGE_BUDGET_TABLE_NAME", "usage-budget"),
	}
	repository := awsintegration.NewScopedDynamoDBRepository(dependencies.DynamoDB, tables, usageScope())
	artifacts := awsintegration.NewS3GeoJSONRepository(dependencies.Objects, envOrDefault("GEOJSON_BUCKET_NAME", "geojson"))
	diagnostics := awsintegration.NewSQSDiagnosticQueue(dependencies.Queues, envOrDefault("FETCH_TASK_DIAGNOSTIC_QUEUE_URL", envOrDefault("FETCH_TASK_QUEUE_URL", "memory")))
	secrets := awsintegration.NewSecretsAdapter(dependencies.Secrets, nil)
	apiKey, err := awsintegration.LoadFlightAwareAPIKey(ctx, os.Getenv, secrets)
	if err != nil {
		return nil, err
	}
	httpClient, err := flightaware.NewHTTPClient(flightaware.HTTPClientConfig{APIKey: apiKey})
	if err != nil {
		return nil, err
	}
	flightAware := flightaware.NewMaxPagesClient(
		flightaware.NewRateLimitedClient(
			flightaware.NewUsageAccountingClient(httpClient, repository, flightaware.StaticUsageEstimator{}),
			flightaware.NewMemoryRateLimitState(),
			15*time.Minute,
		),
	)
	return application.NewFetchProcessor(application.FetchProcessorConfig{
		Flights:     repository,
		Positions:   repository,
		Artifacts:   artifacts,
		FlightAware: flightAware,
		Diagnostics: diagnostics,
		Schedule:    application.DefaultPollSchedulePolicy(),
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
