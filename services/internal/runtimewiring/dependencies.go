package runtimewiring

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"airpath/services/internal/application"
	"airpath/services/internal/awsintegration"
	"airpath/services/internal/domain"
	"airpath/services/internal/flightaware"
	"airpath/services/internal/httpapi"
	"airpath/services/internal/runtimeconfig"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

var (
	flightAwareRateLimitStateOnce sync.Once
	flightAwareRateLimitState     *flightaware.MemoryRateLimitState
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
	if explicit := strings.ToLower(strings.TrimSpace(lookup(RuntimeBackendEnv))); explicit == string(BackendMemory) {
		return BackendMemory
	}
	if lookup(AWSLambdaFunctionNameEnv) == "" {
		return BackendMemory
	}
	return BackendAWS
}

func DynamoDBTablesFromEnv(lookup EnvLookup) awsintegration.DynamoDBTables {
	return awsintegration.DynamoDBTables{
		Flights:         envOrDefault(lookup, FlightsTableNameEnv, "flights"),
		FlightLookup:    envOrDefault(lookup, FlightLookupTableNameEnv, "flight-lookup"),
		FlightPositions: envOrDefault(lookup, FlightPositionsTableNameEnv, "flight-positions"),
		UsageBudget:     envOrDefault(lookup, UsageBudgetTableNameEnv, "usage-budget"),
	}
}

func UsageBudgetScopeFromEnv(lookup EnvLookup, now time.Time) application.UsageBudgetScope {
	return application.UsageBudgetScope{
		Environment: envOrDefault(lookup, AirpathEnvironmentEnv, runtimeconfig.DefaultEnvironment),
		Month:       now.UTC().Format("2006-01"),
	}
}

func GeoJSONBucketFromEnv(lookup EnvLookup) string {
	return envOrDefault(lookup, GeoJSONBucketNameEnv, "geojson")
}

func FetchTaskQueueURLFromEnv(lookup EnvLookup) string {
	return envOrDefault(lookup, FetchTaskQueueURLEnv, "memory")
}

func FetchTaskDiagnosticQueueURLFromEnv(lookup EnvLookup) string {
	return strings.TrimSpace(lookup(FetchTaskDiagnosticQueueURLEnv))
}

func FlightAwareCredentialConfigured(lookup EnvLookup) bool {
	return strings.TrimSpace(lookup(awsintegration.FlightAwareAPIKeyEnv)) != "" ||
		strings.TrimSpace(lookup(awsintegration.FlightAwareAPIKeySecretEnv)) != ""
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
	app := application.New(application.Config{
		Flights:    repository,
		MapData:    awsintegration.NewS3GeoJSONRepository(dependencies.Objects, GeoJSONBucketFromEnv(lookup)),
		Positions:  repository,
		FetchTasks: awsintegration.NewPersistentSQSFetchTaskQueue(dependencies.Queues, FetchTaskQueueURLFromEnv(lookup), repository),
		UsageGuard: RuntimeUsageGuard{
			Upstream:       repository,
			Config:         runtimeConfig,
			UsageScope:     usageScope,
			RateLimitState: sharedFlightAwareRateLimitState(),
		},
		FetchPolicy: runtimeFetchPolicy(runtimeConfig),
	})
	return httpapi.NewAdapter(app), nil
}

func NewFetchProcessor(ctx context.Context, dependencies Dependencies, lookup EnvLookup, now time.Time) (*application.FetchProcessor, error) {
	tables := DynamoDBTablesFromEnv(lookup)
	repository := awsintegration.NewScopedDynamoDBRepository(dependencies.DynamoDB, tables, UsageBudgetScopeFromEnv(lookup, now))
	artifacts := awsintegration.NewS3GeoJSONRepository(dependencies.Objects, GeoJSONBucketFromEnv(lookup))
	var diagnostics application.FetchTaskDiagnosticStore
	if diagnosticQueueURL := FetchTaskDiagnosticQueueURLFromEnv(lookup); diagnosticQueueURL != "" {
		diagnostics = awsintegration.NewSQSDiagnosticQueue(dependencies.Queues, diagnosticQueueURL)
	}
	runtimeConfig := runtimeconfig.FlightAwareRuntimeConfig{
		FetchEnabled:     runtimeconfig.BoolEnv(runtimeconfig.EnvLookup(lookup), FlightAwareFetchEnabledEnv),
		RealCallsEnabled: runtimeconfig.BoolEnv(runtimeconfig.EnvLookup(lookup), FlightAwareRealCallsEnabledEnv),
	}
	flightAware, err := newRuntimeFlightAwareFetchClient(ctx, dependencies, lookup, repository, runtimeConfig)
	if err != nil {
		return nil, err
	}
	return application.NewFetchProcessor(application.FetchProcessorConfig{
		Flights:     repository,
		Positions:   repository,
		Artifacts:   artifacts,
		FlightAware: awsintegration.NewFlightAwareFetchAdapter(flightAware),
		UsageGuard:  repository,
		FetchPolicy: runtimeFetchPolicy(runtimeConfig),
		Diagnostics: diagnostics,
		Schedule:    application.DefaultPollSchedulePolicy(),
	}), nil
}

func newRuntimeFlightAwareFetchClient(ctx context.Context, dependencies Dependencies, lookup EnvLookup, repository *awsintegration.DynamoDBRepository, runtimeConfig runtimeconfig.FlightAwareRuntimeConfig) (awsintegration.FlightAwareFetchClient, error) {
	if !runtimeConfig.ExternalCallsAllowed() {
		// Disabled runtimes avoid resolving credentials entirely; queued work is
		// skipped by policy before this inert client can be called.
		return disabledFlightAwareClient{}, nil
	}
	secrets := awsintegration.NewSecretsAdapter(dependencies.Secrets, nil)
	apiKey, err := awsintegration.LoadFlightAwareAPIKey(ctx, awsintegration.EnvLookup(lookup), secrets)
	if err != nil {
		return nil, err
	}
	httpClient, err := flightaware.NewHTTPClient(flightaware.HTTPClientConfig{APIKey: apiKey})
	if err != nil {
		return nil, err
	}
	return flightaware.NewMaxPagesClient(
		flightaware.NewRateLimitedClient(
			flightaware.NewUsageAccountingClient(httpClient, repository, FlightAwareUsageEstimatorFromEnv(lookup)),
			sharedFlightAwareRateLimitState(),
			15*time.Minute,
		),
	), nil
}

type disabledFlightAwareClient struct{}

func (disabledFlightAwareClient) SearchFlights(context.Context, flightaware.SearchFlightsRequest) (flightaware.SearchFlightsResponse, error) {
	return flightaware.SearchFlightsResponse{}, application.ErrUpstreamFetchDisabled
}

func (disabledFlightAwareClient) GetFlightRoute(context.Context, flightaware.FlightRouteRequest) (flightaware.RouteResponse, error) {
	return flightaware.RouteResponse{}, application.ErrUpstreamFetchDisabled
}

func (disabledFlightAwareClient) GetFlightPosition(context.Context, flightaware.FlightPositionRequest) (flightaware.PositionResponse, error) {
	return flightaware.PositionResponse{}, application.ErrUpstreamFetchDisabled
}

func (disabledFlightAwareClient) GetFlightTrack(context.Context, flightaware.FlightTrackRequest) (flightaware.TrackResponse, error) {
	return flightaware.TrackResponse{}, application.ErrUpstreamFetchDisabled
}

func NewPollingDispatcher(dependencies Dependencies, lookup EnvLookup, now time.Time) *application.PollingDispatcher {
	tables := DynamoDBTablesFromEnv(lookup)
	repository := awsintegration.NewScopedDynamoDBRepository(dependencies.DynamoDB, tables, UsageBudgetScopeFromEnv(lookup, now))
	runtimeConfig := runtimeconfig.FlightAwareRuntimeConfig{
		FetchEnabled:     runtimeconfig.BoolEnv(runtimeconfig.EnvLookup(lookup), FlightAwareFetchEnabledEnv),
		RealCallsEnabled: runtimeconfig.BoolEnv(runtimeconfig.EnvLookup(lookup), FlightAwareRealCallsEnabledEnv),
	}
	return application.NewPollingDispatcher(application.PollingDispatcherConfig{
		Flights:     repository,
		Activities:  PollActivityStoreFromEnv(lookup),
		FetchTasks:  awsintegration.NewPersistentSQSFetchTaskQueue(dependencies.Queues, FetchTaskQueueURLFromEnv(lookup), repository),
		UsageGuard:  repository,
		FetchPolicy: runtimeFetchPolicy(runtimeConfig),
		Schedule:    application.DefaultPollSchedulePolicy(),
	})
}

func PollActivityStoreFromEnv(lookup EnvLookup) application.PollActivityStore {
	return runtimePollActivityStore{
		assumeActiveViewer: runtimeconfig.BoolEnv(runtimeconfig.EnvLookup(lookup), DispatcherAssumeActiveViewerEnv),
	}
}

type runtimePollActivityStore struct {
	assumeActiveViewer bool
}

func (s runtimePollActivityStore) ActivityForFlight(context.Context, domain.FlightID) (application.PollActivitySignal, error) {
	return application.PollActivitySignal{ActiveViewer: s.assumeActiveViewer}, nil
}

func runtimeFetchPolicy(runtimeConfig runtimeconfig.FlightAwareRuntimeConfig) *application.RuntimeFetchPolicy {
	externalCallsAllowed := runtimeConfig.ExternalCallsAllowed()
	return application.NewRuntimeFetchPolicy(application.RuntimeFetchConfig{
		ExternalFetchEnabled:   externalCallsAllowed,
		RouteFetchEnabled:      externalCallsAllowed,
		TrackFetchEnabled:      externalCallsAllowed,
		BackgroundFetchEnabled: externalCallsAllowed,
	})
}

type RuntimeUsageGuard struct {
	Upstream       application.UsageGuard
	Config         runtimeconfig.FlightAwareRuntimeConfig
	UsageScope     application.UsageBudgetScope
	RateLimitState rateLimitStatusProvider
}

type rateLimitStatusProvider interface {
	IsStopped(flightaware.Endpoint) bool
	ResetAt(flightaware.Endpoint) *time.Time
}

func (g RuntimeUsageGuard) FetchingAllowed(ctx context.Context) (bool, error) {
	if !g.Config.ExternalCallsAllowed() {
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
	status.FetchingEnabled = g.Config.ExternalCallsAllowed() && status.FetchingEnabled && !status.Budget.Stopped
	status.RateLimit = mergeRateLimitStatus(status.RateLimit, g.RateLimitState)
	return status, nil
}

func mergeRateLimitStatus(current application.RateLimitStatus, state rateLimitStatusProvider) application.RateLimitStatus {
	if state == nil {
		return current
	}
	for _, endpoint := range []flightaware.Endpoint{
		flightaware.EndpointSearch,
		flightaware.EndpointSummary,
		flightaware.EndpointRoute,
		flightaware.EndpointPosition,
		flightaware.EndpointTrack,
		flightaware.EndpointSchedule,
	} {
		if !state.IsStopped(endpoint) {
			continue
		}
		current.Limited = true
		resetAt := state.ResetAt(endpoint)
		if resetAt == nil {
			continue
		}
		formatted := resetAt.UTC().Format(time.RFC3339)
		if current.ResetAt == nil || formatted > *current.ResetAt {
			current.ResetAt = &formatted
		}
	}
	return current
}

func sharedFlightAwareRateLimitState() *flightaware.MemoryRateLimitState {
	flightAwareRateLimitStateOnce.Do(func() {
		// Lambda execution environments can be reused, so keep rate-limit backoff in
		// process memory across warm invocations.
		flightAwareRateLimitState = flightaware.NewMemoryRateLimitState()
	})
	return flightAwareRateLimitState
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
	value := strings.TrimSpace(lookup(name))
	if value == "" {
		return fallback
	}
	return value
}
