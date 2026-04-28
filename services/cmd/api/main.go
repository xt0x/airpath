package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type apiHealthResponse struct {
	Service                  string `json:"service"`
	Environment              string `json:"environment"`
	FlightAwareFetchEnabled  bool   `json:"flightawareFetchEnabled"`
	FlightAwareSecretPresent bool   `json:"flightawareSecretPresent"`
	Path                     string `json:"path"`
}

type fetchControlResponse struct {
	Service         string `json:"service"`
	Environment     string `json:"environment"`
	FetchingEnabled bool   `json:"fetchingEnabled"`
	Reason          string `json:"reason"`
	Path            string `json:"path"`
}

func main() {
	lambda.Start(handleAPIRequest)
}

func handleAPIRequest(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if request.RawPath == "/v1/admin/fetch-control" {
		return jsonBody(fetchControlResponse{
			Service:         "api",
			Environment:     os.Getenv("AIRPATH_ENVIRONMENT"),
			FetchingEnabled: boolEnv("FLIGHTAWARE_FETCH_ENABLED"),
			Reason:          os.Getenv("FLIGHTAWARE_FETCH_DISABLED_REASON"),
			Path:            request.RawPath,
		})
	}

	return jsonBody(apiHealthResponse{
		Service:                  "api",
		Environment:              os.Getenv("AIRPATH_ENVIRONMENT"),
		FlightAwareFetchEnabled:  boolEnv("FLIGHTAWARE_FETCH_ENABLED"),
		FlightAwareSecretPresent: os.Getenv("FLIGHTAWARE_API_KEY_SECRET_ARN") != "",
		Path:                     request.RawPath,
	})
}

func jsonBody(body any) (events.APIGatewayV2HTTPResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"content-type": "application/json",
		},
		Body: string(payload),
	}, nil
}

func boolEnv(name string) bool {
	return strings.EqualFold(os.Getenv(name), "true")
}
