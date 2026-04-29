package runtimewiring

import (
	"context"
	"errors"
	"strings"
	"time"

	"airpath/services/internal/application"
	"airpath/services/internal/awsintegration"
	"airpath/services/internal/flightaware"
	"airpath/services/internal/httpapi"
	"airpath/services/internal/runtimeconfig"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type Backend string

const (
	BackendAWS    Backend = "aws"
	BackendMemory Backend = "memory"
)

type Dependencies struct {
	DynamoDB awsintegration.DynamoDBClient
	Objects  awsintegration.ObjectClient
	Queues   awsintegration.QueueClient
	Secrets  awsintegration.SecretsClient
}

type EnvLookup func(string) string

func BackendFromEnv(lookup EnvLookup) Backend {
	if explicit := strings.ToLower(strings.TrimSpace(lookup("AIRPATH_RUNTIME_BACKEND"))); explicit == string(BackendMemory) {
		return BackendMemory
	}
	if lookup("AWS_LAMBDA_FUNCTION_NAME") == "" {
		return BackendMemory
	}
	return BackendAWS
}

func DynamoDBTablesFromEnv(lookup EnvLookup) awsintegration.DynamoDBTables {
	return awsintegration.DynamoDBTables{
		Flights:         envOrDefault(lookup, "FLIGHTS_TABLE_NAME", "flights"),
		FlightLookup:    envOrDefault(lookup, "FLIGHT_LOOKUP_TABLE_NAME", "flight-lookup"),
		FlightPositions: envOrDefault(lookup, "FLIGHT_POSITIONS_TABLE_NAME", "flight-positions"),
		UsageBudget:     envOrDefault(lookup, "USAGE_BUDGET_TABLE_NAME", "usage-budget"),
	}
}

func UsageBudgetScopeFromEnv(lookup EnvLookup, now time.Time) application.UsageBudgetScope {
	return application.UsageBudgetScope{
		Environment: envOrDefault(lookup, "AIRPATH_ENVIRONMENT", "local"),
		Month:       now.UTC().Format("2006-01"),
	}
}

func GeoJSONBucketFromEnv(lookup EnvLookup) string {
	return envOrDefault(lookup, "GEOJSON_BUCKET_NAME", "geojson")
}

func FetchTaskQueueURLFromEnv(lookup EnvLookup) string {
	return envOrDefault(lookup, "FETCH_TASK_QUEUE_URL", "memory")
}

func FetchTaskDiagnosticQueueURLFromEnv(lookup EnvLookup) string {
	return envOrDefault(lookup, "FETCH_TASK_DIAGNOSTIC_QUEUE_URL", FetchTaskQueueURLFromEnv(lookup))
}

func NewHTTPAdapter(ctx context.Context, runtimeConfig runtimeconfig.FlightAwareRuntimeConfig, lookup EnvLookup) (*httpapi.Adapter, error) {
	dependencies, err := NewDependencies(ctx, BackendFromEnv(lookup))
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	tables := DynamoDBTablesFromEnv(lookup)
	usageScope := application.UsageBudgetScope{
		Environment: runtimeConfig.Environment,
		Month:       now.Format("2006-01"),
	}
	repository := awsintegration.NewScopedDynamoDBRepository(dependencies.DynamoDB, tables, usageScope)
	fetchPolicy := application.NewRuntimeFetchPolicy(application.RuntimeFetchConfig{
		RouteFetchEnabled:      runtimeConfig.FetchEnabled,
		TrackFetchEnabled:      runtimeConfig.FetchEnabled,
		BackgroundFetchEnabled: runtimeConfig.FetchEnabled,
	})
	app := application.New(application.Config{
		Flights:     repository,
		MapData:     awsintegration.NewS3GeoJSONRepository(dependencies.Objects, GeoJSONBucketFromEnv(lookup)),
		Positions:   repository,
		FetchTasks:  awsintegration.NewSQSFetchTaskQueue(dependencies.Queues, FetchTaskQueueURLFromEnv(lookup)),
		UsageGuard:  RuntimeUsageGuard{Upstream: repository, Config: runtimeConfig, UsageScope: usageScope},
		FetchPolicy: fetchPolicy,
	})
	return httpapi.NewAdapter(app), nil
}

func NewFetchProcessor(ctx context.Context, dependencies Dependencies, lookup EnvLookup, now time.Time) (*application.FetchProcessor, error) {
	tables := DynamoDBTablesFromEnv(lookup)
	repository := awsintegration.NewScopedDynamoDBRepository(dependencies.DynamoDB, tables, UsageBudgetScopeFromEnv(lookup, now))
	artifacts := awsintegration.NewS3GeoJSONRepository(dependencies.Objects, GeoJSONBucketFromEnv(lookup))
	diagnostics := awsintegration.NewSQSDiagnosticQueue(dependencies.Queues, FetchTaskDiagnosticQueueURLFromEnv(lookup))
	secrets := awsintegration.NewSecretsAdapter(dependencies.Secrets, nil)
	apiKey, err := awsintegration.LoadFlightAwareAPIKey(ctx, awsintegration.EnvLookup(lookup), secrets)
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
		FlightAware: awsintegration.NewFlightAwareFetchAdapter(flightAware),
		Diagnostics: diagnostics,
		Schedule:    application.DefaultPollSchedulePolicy(),
	}), nil
}

func NewPollingDispatcher(dependencies Dependencies, lookup EnvLookup, now time.Time) *application.PollingDispatcher {
	tables := DynamoDBTablesFromEnv(lookup)
	repository := awsintegration.NewScopedDynamoDBRepository(dependencies.DynamoDB, tables, UsageBudgetScopeFromEnv(lookup, now))
	return application.NewPollingDispatcher(application.PollingDispatcherConfig{
		Flights:    repository,
		FetchTasks: awsintegration.NewSQSFetchTaskQueue(dependencies.Queues, FetchTaskQueueURLFromEnv(lookup)),
		UsageGuard: repository,
		Schedule:   application.DefaultPollSchedulePolicy(),
	})
}

type RuntimeUsageGuard struct {
	Upstream   application.UsageGuard
	Config     runtimeconfig.FlightAwareRuntimeConfig
	UsageScope application.UsageBudgetScope
}

func (g RuntimeUsageGuard) FetchingAllowed(ctx context.Context) (bool, error) {
	if !g.Config.FetchEnabled {
		return false, nil
	}
	return g.Upstream.FetchingAllowed(ctx)
}

func (g RuntimeUsageGuard) GetUsageStatus(ctx context.Context) (application.UsageStatus, error) {
	status, err := g.Upstream.GetUsageStatus(ctx)
	if err != nil {
		if !errors.Is(err, application.ErrNotFound) {
			return application.UsageStatus{}, err
		}
		status = application.UsageStatus{
			Budget: application.UsageBudgetStatus{
				Environment:       g.UsageScope.Environment,
				Month:             g.UsageScope.Month,
				Currency:          "USD",
				SoftStopThreshold: application.DefaultSoftStopThresholdUSD,
			},
			FetchingEnabled: true,
		}
	}
	status = application.NormalizeUsageStatus(status)
	status.FetchingEnabled = g.Config.FetchEnabled && status.FetchingEnabled && !status.Budget.Stopped
	return status, nil
}

func NewDependencies(ctx context.Context, backend Backend) (Dependencies, error) {
	if backend == BackendMemory {
		return Dependencies{
			DynamoDB: awsintegration.NewMemoryDynamoDBClient(),
			Objects:  awsintegration.NewMemoryObjectClient(),
			Queues:   awsintegration.NewMemoryQueueClient(),
			Secrets:  awsintegration.NewMemorySecretsClient(map[string]string{}),
		}, nil
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return Dependencies{}, err
	}
	return Dependencies{
		DynamoDB: awsintegration.NewAWSDynamoDBClient(dynamodb.NewFromConfig(cfg)),
		Objects:  awsintegration.NewAWSS3ObjectClient(s3.NewFromConfig(cfg)),
		Queues:   awsintegration.NewAWSSQSQueueClient(sqs.NewFromConfig(cfg)),
		Secrets:  awsintegration.NewAWSSecretsClient(secretsmanager.NewFromConfig(cfg)),
	}, nil
}

func envOrDefault(lookup EnvLookup, name string, fallback string) string {
	value := lookup(name)
	if value == "" {
		return fallback
	}
	return value
}
