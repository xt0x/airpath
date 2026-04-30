package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"airpath/services/internal/application"
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

// fetchControlResponse exposes operator-visible fetch gates without returning
// credentials or other deployment secrets.
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

func handleAPIRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	config, err := runtimeconfig.LoadFlightAwareRuntimeConfig(os.Getenv)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	// Keep read-only control-plane routes in this package so health checks do
	// not need to initialize storage or external-service adapters.
	if request.RawPath == "/v1/admin/fetch-control" {
		if response, handled, err := handleStaticReadRoute(request); handled {
			return response, err
		}
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
		if response, handled, err := handleStaticReadRoute(request); handled {
			return response, err
		}
		return jsonBody(apiHealthResponse{
			Service:                         "api",
			Environment:                     config.Environment,
			PersonalDemoNotice:              config.PersonalDemoNotice,
			FlightAwareFetchEnabled:         config.FetchEnabled,
			FlightAwareRealCallsEnabled:     config.RealCallsEnabled,
			ExternalFlightAwareCallsAllowed: config.ExternalCallsAllowed(),
			FlightAwareSecretPresent:        runtimewiring.FlightAwareCredentialConfigured(os.Getenv),
			Path:                            request.RawPath,
		})
	}

	adapter, err := newHTTPAdapter(ctx, config)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return adapter.Handle(ctx, request)
}

// handleStaticReadRoute applies the method contract shared by local static
// endpoints before the request falls through to response construction.
func handleStaticReadRoute(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, bool, error) {
	method := request.RequestContext.HTTP.Method
	if method == http.MethodOptions {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusNoContent,
			Headers:    responseHeaders(""),
		}, true, nil
	}
	if method != http.MethodGet {
		apiErr := application.MapApplicationError(application.ErrNotFound, request.RequestContext.RequestID)
		response, err := jsonBodyWithStatus(http.StatusNotFound, map[string]any{"error": apiErr})
		return response, true, err
	}
	return events.APIGatewayV2HTTPResponse{}, false, nil
}

func jsonBody(body any) (events.APIGatewayV2HTTPResponse, error) {
	return jsonBodyWithStatus(http.StatusOK, body)
}

func jsonBodyWithStatus(statusCode int, body any) (events.APIGatewayV2HTTPResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers:    responseHeaders("application/json"),
		Body:       string(payload),
	}, nil
}

// responseHeaders is intentionally small and shared by all local responses so
// CORS behavior stays consistent with the adapter-backed routes.
func responseHeaders(contentType string) map[string]string {
	headers := map[string]string{
		"access-control-allow-origin":  "*",
		"access-control-allow-methods": strings.Join([]string{http.MethodGet, http.MethodPost, http.MethodOptions}, ","),
		"access-control-allow-headers": "accept,content-type",
	}
	if contentType != "" {
		headers["content-type"] = contentType
	}
	return headers
}

func newHTTPAdapter(ctx context.Context, config runtimeconfig.FlightAwareRuntimeConfig) (*httpapi.Adapter, error) {
	return runtimewiring.NewHTTPAdapter(ctx, config, os.Getenv)
}
