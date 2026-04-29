package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"airpath/services/internal/application"
	"airpath/services/internal/awsintegration"
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
	dependencies, err := runtimewiring.NewDependencies(ctx, runtimeBackend(os.Getenv))
	if err != nil {
		return nil, err
	}
	tables := awsintegration.DynamoDBTables{
		Flights:         envOrDefault("FLIGHTS_TABLE_NAME", "flights"),
		FlightLookup:    envOrDefault("FLIGHT_LOOKUP_TABLE_NAME", "flight-lookup"),
		FlightPositions: envOrDefault("FLIGHT_POSITIONS_TABLE_NAME", "flight-positions"),
		UsageBudget:     envOrDefault("USAGE_BUDGET_TABLE_NAME", "usage-budget"),
	}
	usageScope := application.UsageBudgetScope{
		Environment: config.Environment,
		Month:       time.Now().UTC().Format("2006-01"),
	}
	repository := awsintegration.NewScopedDynamoDBRepository(dependencies.DynamoDB, tables, usageScope)
	fetchPolicy := application.NewRuntimeFetchPolicy(application.RuntimeFetchConfig{
		RouteFetchEnabled:      config.FetchEnabled,
		TrackFetchEnabled:      config.FetchEnabled,
		BackgroundFetchEnabled: config.FetchEnabled,
	})
	app := application.New(application.Config{
		Flights:     repository,
		MapData:     awsintegration.NewS3GeoJSONRepository(dependencies.Objects, envOrDefault("GEOJSON_BUCKET_NAME", "geojson")),
		Positions:   repository,
		FetchTasks:  awsintegration.NewSQSFetchTaskQueue(dependencies.Queues, envOrDefault("FETCH_TASK_QUEUE_URL", "memory")),
		UsageGuard:  runtimeUsageGuard{upstream: repository, config: config, usageScope: usageScope},
		FetchPolicy: fetchPolicy,
	})
	return httpapi.NewAdapter(app), nil
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

type runtimeUsageGuard struct {
	upstream   application.UsageGuard
	config     runtimeconfig.FlightAwareRuntimeConfig
	usageScope application.UsageBudgetScope
}

func (g runtimeUsageGuard) FetchingAllowed(ctx context.Context) (bool, error) {
	if !g.config.FetchEnabled {
		return false, nil
	}
	return g.upstream.FetchingAllowed(ctx)
}

func (g runtimeUsageGuard) GetUsageStatus(ctx context.Context) (application.UsageStatus, error) {
	status, err := g.upstream.GetUsageStatus(ctx)
	if err != nil {
		if !errors.Is(err, application.ErrNotFound) {
			return application.UsageStatus{}, err
		}
		status = application.UsageStatus{
			Budget: application.UsageBudgetStatus{
				Environment:       g.usageScope.Environment,
				Month:             g.usageScope.Month,
				Currency:          "USD",
				SoftStopThreshold: application.DefaultSoftStopThresholdUSD,
			},
			FetchingEnabled: true,
		}
	}
	status = application.NormalizeUsageStatus(status)
	status.FetchingEnabled = g.config.FetchEnabled && status.FetchingEnabled && !status.Budget.Stopped
	return status, nil
}

func envOrDefault(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
