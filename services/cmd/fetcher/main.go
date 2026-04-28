package main

import (
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type fetcherResponse struct {
	Service                string `json:"service"`
	Environment            string `json:"environment"`
	Mode                   string `json:"mode"`
	RecordsReceived        int    `json:"recordsReceived"`
	FlightAwareSecretReady bool   `json:"flightawareSecretReady"`
	ExternalFetchAttempted bool   `json:"externalFetchAttempted"`
}

func main() {
	lambda.Start(handleFetchEvent)
}

func handleFetchEvent(event events.SQSEvent) (fetcherResponse, error) {
	return fetcherResponse{
		Service:                "fetcher",
		Environment:            os.Getenv("AIRPATH_ENVIRONMENT"),
		Mode:                   os.Getenv("FETCHER_MODE"),
		RecordsReceived:        len(event.Records),
		FlightAwareSecretReady: os.Getenv("FLIGHTAWARE_API_KEY_SECRET_ARN") != "",
		ExternalFetchAttempted: false,
	}, nil
}
