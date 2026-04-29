package main

import (
	"context"
	"encoding/json"
	"os"

	"airpath/services/internal/httpapi"
	"airpath/services/internal/runtimeconfig"
	"airpath/services/internal/runtimewiring"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

const (
	runtimeBackendAWS    = runtimewiring.BackendAWS
	runtimeBackendMemory = runtimewiring.BackendMemory
)

type apiHealthResponse struct {
	Service                         string `json:"service"`
	Environment                     string `json:"environment"`
	PersonalDemoNotice              string `json:"personalDemoNotice"`
	FlightAwareFetchEnabled         bool   `json:"flightawareFetchEnabled"`
	FlightAwareRealCallsEnabled     bool   `json:"flightawareRealCallsEnabled"`
	ExternalFlightAwareCallsAllowed bool   `json:"externalFlightAwareCallsAllowed"`
	FlightAwareSecretPresent        bool   `json:"flightawareSecretPresent"`
	Path                            string `json:"path"`
}

type fetchControlResponse struct {
	Service              string `json:"service"`
	Environment          string `json:"environment"`
	PersonalDemoNotice   string `json:"personalDemoNotice"`
	FetchingEnabled      bool   `json:"fetchingEnabled"`
	RealCallsEnabled     bool   `json:"realCallsEnabled"`
	ExternalCallsAllowed bool   `json:"externalCallsAllowed"`
	Reason               string `json:"reason"`
	Path                 string `json:"path"`
}

func main() {
	lambda.Start(handleAPIRequest)
}

func handleAPIRequest(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	config, err := runtimeconfig.LoadFlightAwareRuntimeConfig(os.Getenv)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	if request.RawPath == "/v1/admin/fetch-control" {
		return jsonBody(fetchControlResponse{
			Service:              "api",
			Environment:          config.Environment,
			PersonalDemoNotice:   config.PersonalDemoNotice,
			FetchingEnabled:      config.FetchEnabled,
			RealCallsEnabled:     config.RealCallsEnabled,
			ExternalCallsAllowed: config.ExternalCallsAllowed(),
			Reason:               config.DisabledReason,
			Path:                 request.RawPath,
		})
	}
	if request.RawPath == "/v1/health" {
		return jsonBody(apiHealthResponse{
			Service:                         "api",
			Environment:                     config.Environment,
			PersonalDemoNotice:              config.PersonalDemoNotice,
			FlightAwareFetchEnabled:         config.FetchEnabled,
			FlightAwareRealCallsEnabled:     config.RealCallsEnabled,
			ExternalFlightAwareCallsAllowed: config.ExternalCallsAllowed(),
			FlightAwareSecretPresent:        os.Getenv("FLIGHTAWARE_API_KEY_SECRET_ARN") != "",
			Path:                            request.RawPath,
		})
	}

	adapter, err := newHTTPAdapter(context.Background(), config)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return adapter.Handle(context.Background(), request)
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

func newHTTPAdapter(ctx context.Context, config runtimeconfig.FlightAwareRuntimeConfig) (*httpapi.Adapter, error) {
	return runtimewiring.NewHTTPAdapter(ctx, config, os.Getenv)
}

func runtimeBackend(lookup func(string) string) runtimewiring.Backend {
	return runtimewiring.BackendFromEnv(lookup)
}
