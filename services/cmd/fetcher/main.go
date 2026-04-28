package main

import (
	"os"

	"airpath/services/internal/runtimeconfig"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

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

	return fetcherResponse{
		Service:                     "fetcher",
		Environment:                 config.Environment,
		Mode:                        os.Getenv("FETCHER_MODE"),
		RecordsReceived:             len(event.Records),
		FlightAwareSecretReady:      os.Getenv("FLIGHTAWARE_API_KEY_SECRET_ARN") != "",
		FlightAwareFetchEnabled:     config.FetchEnabled,
		FlightAwareRealCallsEnabled: config.RealCallsEnabled,
		ExternalFetchAllowed:        config.ExternalCallsAllowed(),
		ExternalFetchAttempted:      false,
	}, nil
}
