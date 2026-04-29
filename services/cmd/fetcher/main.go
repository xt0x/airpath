package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"airpath/services/internal/application"
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
	return runtimewiring.NewFetchProcessor(ctx, dependencies, os.Getenv, time.Now().UTC())
}

func runtimeBackend(lookup func(string) string) runtimewiring.Backend {
	return runtimewiring.BackendFromEnv(lookup)
}
