package main

import (
	"context"
	"encoding/json"
	"os"

	"airpath/services/internal/application"
	"airpath/services/internal/cmdsupport"
	"airpath/services/internal/runtimeconfig"
	"airpath/services/internal/runtimewiring"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type runtimeDependencies = runtimewiring.Dependencies

// fetchProcessor is the narrow use-case boundary this Lambda needs, which keeps
// SQS batch handling testable without constructing real runtime adapters.
type fetchProcessor interface {
	Process(context.Context, application.FetchTask, application.ProcessFetchInput) (application.ProcessFetchResult, error)
}

var newFetchProcessor = buildFetchProcessor

// fetcherResponse is returned to Lambda logs and tests as operational telemetry;
// it reports credential readiness but never includes credential values.
type fetcherResponse struct {
	Service                     string                       `json:"service"`
	Environment                 string                       `json:"environment"`
	Mode                        string                       `json:"mode"`
	RecordsReceived             int                          `json:"recordsReceived"`
	FlightAwareSecretReady      bool                         `json:"flightawareSecretReady"`
	FlightAwareFetchEnabled     bool                         `json:"flightawareFetchEnabled"`
	FlightAwareRealCallsEnabled bool                         `json:"flightawareRealCallsEnabled"`
	ExternalFetchAllowed        bool                         `json:"externalFetchAllowed"`
	ExternalFetchAttempted      bool                         `json:"externalFetchAttempted"`
	BatchItemFailures           []events.SQSBatchItemFailure `json:"batchItemFailures,omitempty"`
}

func main() {
	lambda.Start(handleFetchEvent)
}

func handleFetchEvent(ctx context.Context, event events.SQSEvent) (fetcherResponse, error) {
	config, err := runtimeconfig.LoadFlightAwareRuntimeConfig(os.Getenv)
	if err != nil {
		return fetcherResponse{}, err
	}
	response := fetcherResponse{
		Service:                     "fetcher",
		Environment:                 config.Environment,
		Mode:                        os.Getenv("FETCHER_MODE"),
		RecordsReceived:             len(event.Records),
		FlightAwareSecretReady:      runtimewiring.FlightAwareCredentialConfigured(os.Getenv),
		FlightAwareFetchEnabled:     config.FetchEnabled,
		FlightAwareRealCallsEnabled: config.RealCallsEnabled,
		ExternalFetchAllowed:        config.ExternalCallsAllowed(),
		ExternalFetchAttempted:      false,
	}
	if len(event.Records) == 0 {
		return response, nil
	}

	dependencies, err := runtimewiring.NewDependencies(ctx, cmdsupport.RuntimeBackend(os.Getenv))
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
			// Malformed messages should be retried or moved to the DLQ without
			// replaying records that already decoded successfully.
			response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
			continue
		}
		result, err := processor.Process(ctx, task, application.ProcessFetchInput{
			Now:      cmdsupport.NowUTC(),
			WorkerID: fetcherWorkerID(record, task),
		})
		response.ExternalFetchAttempted = response.ExternalFetchAttempted || result.ExternalFetchAttempted
		if err != nil {
			// Lambda partial batch failure requires message IDs, not task IDs.
			response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
			continue
		}
	}
	return response, nil
}

func buildFetchProcessor(ctx context.Context, dependencies runtimeDependencies) (fetchProcessor, error) {
	return runtimewiring.NewFetchProcessor(ctx, dependencies, os.Getenv, cmdsupport.NowUTC())
}

// fetcherWorkerID gives downstream idempotency and diagnostics a stable
// invocation-local owner while preferring the SQS message identifier.
func fetcherWorkerID(record events.SQSMessage, task application.FetchTask) string {
	if record.MessageId != "" {
		return "fetcher:" + record.MessageId
	}
	if task.TaskID != "" {
		return "fetcher:" + task.TaskID
	}
	return "fetcher"
}
