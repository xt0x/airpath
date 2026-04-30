package runtimewiring

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"airpath/services/internal/application"
	"airpath/services/internal/awsintegration"
	"airpath/services/internal/domain"
	"airpath/services/internal/flightaware"
	"airpath/services/internal/runtimeconfig"
)

func TestRuntimeWiringEnvNamesAreCentralized(t *testing.T) {
	source, err := os.ReadFile("dependencies.go")
	if err != nil {
		t.Fatalf("read dependencies.go: %v", err)
	}
	for _, envName := range []string{
		"AIRPATH_RUNTIME_BACKEND",
		"AWS_LAMBDA_FUNCTION_NAME",
		"FLIGHTS_TABLE_NAME",
		"FLIGHT_LOOKUP_TABLE_NAME",
		"FLIGHT_POSITIONS_TABLE_NAME",
		"USAGE_BUDGET_TABLE_NAME",
		"GEOJSON_BUCKET_NAME",
		"FETCH_TASK_QUEUE_URL",
		"FETCH_TASK_DIAGNOSTIC_QUEUE_URL",
	} {
		if strings.Contains(string(source), `"`+envName+`"`) {
			t.Fatalf("dependencies.go contains raw env name %q; use runtime wiring env constants", envName)
		}
	}
}

func TestBackendFromEnvDefaultsToMemoryOutsideLambda(t *testing.T) {
	if got := BackendFromEnv(func(string) string { return "" }); got != BackendMemory {
		t.Fatalf("BackendFromEnv(outside lambda) = %q, want memory", got)
	}
}

func TestBackendFromEnvUsesAWSInsideLambdaUnlessMemoryIsExplicit(t *testing.T) {
	insideLambda := func(name string) string {
		if name == "AWS_LAMBDA_FUNCTION_NAME" {
			return "airpath-dev-api"
		}
		return ""
	}
	if got := BackendFromEnv(insideLambda); got != BackendAWS {
		t.Fatalf("BackendFromEnv(lambda) = %q, want aws", got)
	}

	explicitMemory := func(name string) string {
		if name == "AIRPATH_RUNTIME_BACKEND" {
			return "memory"
		}
		return insideLambda(name)
	}
	if got := BackendFromEnv(explicitMemory); got != BackendMemory {
		t.Fatalf("BackendFromEnv(explicit memory) = %q, want memory", got)
	}
}

func TestRuntimeTableAndScopeConfigAreResolvedCentrally(t *testing.T) {
	lookup := func(name string) string {
		values := map[string]string{
			"AIRPATH_ENVIRONMENT":             "dev",
			"FLIGHTS_TABLE_NAME":              "airpath-dev-flights",
			"FLIGHT_LOOKUP_TABLE_NAME":        "airpath-dev-flight-lookup",
			"FLIGHT_POSITIONS_TABLE_NAME":     "airpath-dev-flight-positions",
			"USAGE_BUDGET_TABLE_NAME":         "airpath-dev-usage-budget",
			"GEOJSON_BUCKET_NAME":             "airpath-dev-geojson",
			"FETCH_TASK_QUEUE_URL":            "https://sqs.example/fetch",
			"FETCH_TASK_DIAGNOSTIC_QUEUE_URL": "https://sqs.example/diagnostic",
		}
		return values[name]
	}

	tables := DynamoDBTablesFromEnv(lookup)
	if tables.Flights != "airpath-dev-flights" || tables.UsageBudget != "airpath-dev-usage-budget" {
		t.Fatalf("tables = %#v", tables)
	}
	scope := UsageBudgetScopeFromEnv(lookup, time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC))
	if scope.Environment != "dev" || scope.Month != "2026-04" {
		t.Fatalf("scope = %#v", scope)
	}
	if got := GeoJSONBucketFromEnv(lookup); got != "airpath-dev-geojson" {
		t.Fatalf("GeoJSONBucketFromEnv() = %q", got)
	}
	if got := FetchTaskQueueURLFromEnv(lookup); got != "https://sqs.example/fetch" {
		t.Fatalf("FetchTaskQueueURLFromEnv() = %q", got)
	}
	if got := FetchTaskDiagnosticQueueURLFromEnv(lookup); got != "https://sqs.example/diagnostic" {
		t.Fatalf("FetchTaskDiagnosticQueueURLFromEnv() = %q", got)
	}
}

func TestUsageBudgetScopeFromEnvDefaultsBlankEnvironmentToLocal(t *testing.T) {
	lookup := func(name string) string {
		if name == AirpathEnvironmentEnv {
			return "   "
		}
		return ""
	}

	scope := UsageBudgetScopeFromEnv(lookup, time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC))
	if scope.Environment != runtimeconfig.DefaultEnvironment {
		t.Fatalf("Environment = %q, want %q", scope.Environment, runtimeconfig.DefaultEnvironment)
	}
	if scope.Month != "2026-04" {
		t.Fatalf("Month = %q, want 2026-04", scope.Month)
	}
}

func TestFetchTaskDiagnosticQueueURLDoesNotFallBackToFetchQueue(t *testing.T) {
	lookup := func(name string) string {
		if name == FetchTaskQueueURLEnv {
			return "https://sqs.example/fetch"
		}
		return ""
	}

	if got := FetchTaskDiagnosticQueueURLFromEnv(lookup); got != "" {
		t.Fatalf("FetchTaskDiagnosticQueueURLFromEnv() = %q, want empty when diagnostic queue is not configured", got)
	}
}

func TestFlightAwareCredentialConfiguredAcceptsEnvironmentKeyOrSecretReference(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string
		want   bool
	}{
		{name: "missing", values: map[string]string{}, want: false},
		{name: "environment key", values: map[string]string{awsintegration.FlightAwareAPIKeyEnv: "local-key"}, want: true},
		{name: "secret reference", values: map[string]string{awsintegration.FlightAwareAPIKeySecretEnv: "secret-ref"}, want: true},
		{name: "blank values", values: map[string]string{awsintegration.FlightAwareAPIKeyEnv: " ", awsintegration.FlightAwareAPIKeySecretEnv: " "}, want: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := FlightAwareCredentialConfigured(func(name string) string {
				return testCase.values[name]
			})
			if got != testCase.want {
				t.Fatalf("FlightAwareCredentialConfigured() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestFlightAwareUsageEstimatorFromEnvDefaultsToNonZeroCost(t *testing.T) {
	estimator := FlightAwareUsageEstimatorFromEnv(func(string) string { return "" })

	for _, endpoint := range []flightaware.Endpoint{
		flightaware.EndpointSearch,
		flightaware.EndpointSummary,
		flightaware.EndpointRoute,
		flightaware.EndpointPosition,
		flightaware.EndpointTrack,
		flightaware.EndpointSchedule,
		flightaware.EndpointUsage,
	} {
		resultSets, costUSD := estimator.Estimate(endpoint)
		if resultSets < 1 || costUSD <= 0 {
			t.Fatalf("Estimate(%s) = resultSets %d cost %v, want non-zero conservative estimate", endpoint, resultSets, costUSD)
		}
	}
}

func TestFlightAwareUsageEstimatorFromEnvAllowsCostOverride(t *testing.T) {
	estimator := FlightAwareUsageEstimatorFromEnv(func(name string) string {
		if name == FlightAwareEstimatedCostUSDPerCallEnv {
			return "0.25"
		}
		return ""
	})

	_, costUSD := estimator.Estimate("route")
	if costUSD != 0.25 {
		t.Fatalf("route cost = %v, want env override 0.25", costUSD)
	}
}

func TestFlightAwareRateLimitStateIsSharedAcrossFetchProcessorWiring(t *testing.T) {
	first := sharedFlightAwareRateLimitState()
	second := sharedFlightAwareRateLimitState()

	if first == nil {
		t.Fatal("sharedFlightAwareRateLimitState() = nil")
	}
	if first != second {
		t.Fatal("FlightAware rate-limit state is not shared across processor wiring")
	}
}

func TestPollingDispatcherRuntimeWiringHonorsDisabledFetchPolicy(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 29, 0, 5, 0, 0, time.UTC)
	dynamo := awsintegration.NewMemoryDynamoDBClient()
	queue := awsintegration.NewMemoryQueueClient()
	tables := awsintegration.DynamoDBTables{
		Flights:     "Flights",
		UsageBudget: "UsageBudget",
	}
	repository := awsintegration.NewDynamoDBRepository(dynamo, tables)
	faFlightID := domain.FAFlightID("fa_poll_1")
	dueAt := domain.ISODateTimeString("2026-04-29T00:00:00Z")
	if err := repository.PutFlight(ctx, domain.Flight{
		FlightID:           "iflg_poll_1",
		FlightIDType:       domain.FlightIDTypeInternal,
		FAFlightID:         &faFlightID,
		Ident:              "ANA110",
		Origin:             domain.Airport{Code: "RJTT"},
		Destination:        domain.Airport{Code: "KJFK"},
		Status:             "En Route",
		NextPositionPollAt: &dueAt,
		UpdatedAt:          "2026-04-29T00:00:00Z",
	}); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}

	lookup := func(name string) string {
		values := map[string]string{
			AirpathEnvironmentEnv:           "dev",
			FlightsTableNameEnv:             tables.Flights,
			UsageBudgetTableNameEnv:         tables.UsageBudget,
			FetchTaskQueueURLEnv:            "memory-fetch",
			FlightAwareFetchEnabledEnv:      "false",
			DispatcherAssumeActiveViewerEnv: "true",
		}
		return values[name]
	}
	dispatcher := NewPollingDispatcher(Dependencies{
		DynamoDB: dynamo,
		Queues:   queue,
	}, lookup, now)

	result, err := dispatcher.Dispatch(ctx, application.DispatchPollInput{Now: now, Limit: 10})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if result.EnqueuedTasks != 0 {
		t.Fatalf("EnqueuedTasks = %d, want 0 while FlightAware fetches are disabled", result.EnqueuedTasks)
	}
	if messages := queue.Messages(); len(messages) != 0 {
		t.Fatalf("queued messages = %#v, want none while FlightAware fetches are disabled", messages)
	}
}

func TestPollingDispatcherRuntimeWiringRequiresRealCallOptIn(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 29, 0, 5, 0, 0, time.UTC)
	dynamo := awsintegration.NewMemoryDynamoDBClient()
	queue := awsintegration.NewMemoryQueueClient()
	tables := awsintegration.DynamoDBTables{
		Flights:     "Flights",
		UsageBudget: "UsageBudget",
	}
	repository := awsintegration.NewDynamoDBRepository(dynamo, tables)
	faFlightID := domain.FAFlightID("fa_poll_real_opt_in")
	dueAt := domain.ISODateTimeString("2026-04-29T00:00:00Z")
	if err := repository.PutFlight(ctx, domain.Flight{
		FlightID:           "iflg_poll_real_opt_in",
		FlightIDType:       domain.FlightIDTypeInternal,
		FAFlightID:         &faFlightID,
		Ident:              "ANA110",
		Origin:             domain.Airport{Code: "RJTT"},
		Destination:        domain.Airport{Code: "KJFK"},
		Status:             "En Route",
		NextPositionPollAt: &dueAt,
		UpdatedAt:          "2026-04-29T00:00:00Z",
	}); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}

	lookup := func(name string) string {
		values := map[string]string{
			AirpathEnvironmentEnv:           "dev",
			FlightsTableNameEnv:             tables.Flights,
			UsageBudgetTableNameEnv:         tables.UsageBudget,
			FetchTaskQueueURLEnv:            "memory-fetch",
			FlightAwareFetchEnabledEnv:      "true",
			FlightAwareRealCallsEnabledEnv:  "false",
			DispatcherAssumeActiveViewerEnv: "true",
		}
		return values[name]
	}
	dispatcher := NewPollingDispatcher(Dependencies{
		DynamoDB: dynamo,
		Queues:   queue,
	}, lookup, now)

	result, err := dispatcher.Dispatch(ctx, application.DispatchPollInput{Now: now, Limit: 10})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if result.EnqueuedTasks != 0 {
		t.Fatalf("EnqueuedTasks = %d, want 0 without real-call opt-in", result.EnqueuedTasks)
	}
	if messages := queue.Messages(); len(messages) != 0 {
		t.Fatalf("queued messages = %#v, want none without real-call opt-in", messages)
	}
}

func TestPollingDispatcherRuntimeWiringEnqueuesWhenRuntimeFlagsAllowRealCalls(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 29, 0, 5, 0, 0, time.UTC)
	dynamo := awsintegration.NewMemoryDynamoDBClient()
	queue := awsintegration.NewMemoryQueueClient()
	tables := awsintegration.DynamoDBTables{
		Flights:     "Flights",
		UsageBudget: "UsageBudget",
	}
	repository := awsintegration.NewDynamoDBRepository(dynamo, tables)
	faFlightID := domain.FAFlightID("fa_poll_real_enabled")
	dueAt := domain.ISODateTimeString("2026-04-29T00:00:00Z")
	if err := repository.PutFlight(ctx, domain.Flight{
		FlightID:           "iflg_poll_real_enabled",
		FlightIDType:       domain.FlightIDTypeInternal,
		FAFlightID:         &faFlightID,
		Ident:              "ANA110",
		Origin:             domain.Airport{Code: "RJTT"},
		Destination:        domain.Airport{Code: "KJFK"},
		Status:             "En Route",
		NextPositionPollAt: &dueAt,
		UpdatedAt:          "2026-04-29T00:00:00Z",
	}); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}

	lookup := func(name string) string {
		values := map[string]string{
			AirpathEnvironmentEnv:           "dev",
			FlightsTableNameEnv:             tables.Flights,
			UsageBudgetTableNameEnv:         tables.UsageBudget,
			FetchTaskQueueURLEnv:            "memory-fetch",
			FlightAwareFetchEnabledEnv:      "true",
			FlightAwareRealCallsEnabledEnv:  "true",
			DispatcherAssumeActiveViewerEnv: "true",
		}
		return values[name]
	}
	dispatcher := NewPollingDispatcher(Dependencies{
		DynamoDB: dynamo,
		Queues:   queue,
	}, lookup, now)

	result, err := dispatcher.Dispatch(ctx, application.DispatchPollInput{Now: now, Limit: 10})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if result.EnqueuedTasks != 1 {
		t.Fatalf("EnqueuedTasks = %d, want 1 when fetch and real-call flags are enabled", result.EnqueuedTasks)
	}
	if messages := queue.Messages(); len(messages) != 1 {
		t.Fatalf("queued messages = %#v, want one task when fetch and real-call flags are enabled", messages)
	}
}

func TestPollingDispatcherRuntimeWiringStopsIdleFlightsWithoutActivity(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 29, 0, 20, 0, 0, time.UTC)
	dynamo := awsintegration.NewMemoryDynamoDBClient()
	queue := awsintegration.NewMemoryQueueClient()
	tables := awsintegration.DynamoDBTables{
		Flights:     "Flights",
		UsageBudget: "UsageBudget",
	}
	repository := awsintegration.NewDynamoDBRepository(dynamo, tables)
	faFlightID := domain.FAFlightID("fa_poll_idle")
	idleSince := domain.ISODateTimeString("2026-04-29T00:00:00Z")
	dueAt := domain.ISODateTimeString("2026-04-29T00:15:00Z")
	if err := repository.PutFlight(ctx, domain.Flight{
		FlightID:           "iflg_poll_idle",
		FlightIDType:       domain.FlightIDTypeInternal,
		FAFlightID:         &faFlightID,
		Ident:              "ANA110",
		Origin:             domain.Airport{Code: "RJTT"},
		Destination:        domain.Airport{Code: "KJFK"},
		Status:             "En Route",
		IdleSince:          &idleSince,
		NextPositionPollAt: &dueAt,
		UpdatedAt:          "2026-04-29T00:00:00Z",
	}); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}

	lookup := func(name string) string {
		values := map[string]string{
			AirpathEnvironmentEnv:          "dev",
			FlightsTableNameEnv:            tables.Flights,
			UsageBudgetTableNameEnv:        tables.UsageBudget,
			FetchTaskQueueURLEnv:           "memory-fetch",
			FlightAwareFetchEnabledEnv:     "true",
			FlightAwareRealCallsEnabledEnv: "true",
		}
		return values[name]
	}
	dispatcher := NewPollingDispatcher(Dependencies{
		DynamoDB: dynamo,
		Queues:   queue,
	}, lookup, now)

	result, err := dispatcher.Dispatch(ctx, application.DispatchPollInput{Now: now, Limit: 10})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if result.EnqueuedTasks != 0 {
		t.Fatalf("EnqueuedTasks = %d, want 0 for idle flight without activity", result.EnqueuedTasks)
	}
	if messages := queue.Messages(); len(messages) != 0 {
		t.Fatalf("queued messages = %#v, want none for idle flight without activity", messages)
	}
	updated, _, err := repository.GetFlight(ctx, "iflg_poll_idle")
	if err != nil {
		t.Fatalf("GetFlight() error = %v", err)
	}
	if updated.PollState == nil || *updated.PollState != domain.FlightPollStateCompleted {
		t.Fatalf("PollState = %v, want completed", updated.PollState)
	}
	if updated.NextPositionPollAt != nil {
		t.Fatalf("NextPositionPollAt = %v, want nil after idle stop", updated.NextPositionPollAt)
	}
}

func TestRuntimeUsageGuardRequiresRealCallOptIn(t *testing.T) {
	guard := RuntimeUsageGuard{
		Upstream: applicationUsageGuardFunc(func(context.Context) (bool, error) {
			return true, nil
		}),
		Config: runtimeconfigForTest(true, false),
	}

	allowed, err := guard.FetchingAllowed(context.Background())
	if err != nil {
		t.Fatalf("FetchingAllowed() error = %v", err)
	}
	if allowed {
		t.Fatal("FetchingAllowed() = true, want false without real-call opt-in")
	}
}

func TestRuntimeUsageGuardSurfacesSharedRateLimitState(t *testing.T) {
	resetAt := time.Now().UTC().Add(20 * time.Minute).Truncate(time.Second)
	rateLimitState := flightaware.NewMemoryRateLimitState()
	rateLimitState.MarkRateLimited(flightaware.EndpointPosition, resetAt)
	guard := RuntimeUsageGuard{
		Upstream: applicationUsageGuardFunc(func(context.Context) (bool, error) {
			return true, nil
		}),
		Config:         runtimeconfigForTest(true, true),
		UsageScope:     application.UsageBudgetScope{Environment: "dev", Month: "2026-04"},
		RateLimitState: rateLimitState,
	}

	status, err := guard.GetUsageStatus(context.Background())
	if err != nil {
		t.Fatalf("GetUsageStatus() error = %v", err)
	}

	if !status.RateLimit.Limited {
		t.Fatalf("RateLimit = %#v, want limited", status.RateLimit)
	}
	wantResetAt := resetAt.Format(time.RFC3339)
	if status.RateLimit.ResetAt == nil || *status.RateLimit.ResetAt != wantResetAt {
		t.Fatalf("ResetAt = %v, want %s", status.RateLimit.ResetAt, wantResetAt)
	}
}

type applicationUsageGuardFunc func(context.Context) (bool, error)

func (f applicationUsageGuardFunc) FetchingAllowed(ctx context.Context) (bool, error) {
	return f(ctx)
}

func (f applicationUsageGuardFunc) GetUsageStatus(context.Context) (application.UsageStatus, error) {
	return application.UsageStatus{FetchingEnabled: true}, nil
}

func runtimeconfigForTest(fetchEnabled bool, realCallsEnabled bool) runtimeconfig.FlightAwareRuntimeConfig {
	return runtimeconfig.FlightAwareRuntimeConfig{
		FetchEnabled:     fetchEnabled,
		RealCallsEnabled: realCallsEnabled,
	}
}

func TestFetchProcessorRuntimeWiringSkipsDiagnosticsWhenQueueIsUnconfigured(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 29, 0, 5, 0, 0, time.UTC)
	dynamo := awsintegration.NewMemoryDynamoDBClient()
	queue := awsintegration.NewMemoryQueueClient()
	tables := awsintegration.DynamoDBTables{
		Flights:         "Flights",
		FlightLookup:    "FlightLookup",
		FlightPositions: "FlightPositions",
		UsageBudget:     "UsageBudget",
	}
	repository := awsintegration.NewDynamoDBRepository(dynamo, tables)
	if err := repository.PutFlight(ctx, domain.Flight{
		FlightID:     "iflg_missing_fa",
		FlightIDType: domain.FlightIDTypeInternal,
		Ident:        "ANA110",
		Origin:       domain.Airport{Code: "RJTT"},
		Destination:  domain.Airport{Code: "KJFK"},
		Status:       "En Route",
		UpdatedAt:    "2026-04-29T00:00:00Z",
	}); err != nil {
		t.Fatalf("PutFlight() error = %v", err)
	}

	lookup := func(name string) string {
		values := map[string]string{
			FlightAwareFetchEnabledEnv:            "true",
			FlightAwareRealCallsEnabledEnv:        "true",
			AirpathEnvironmentEnv:                 "dev",
			FlightsTableNameEnv:                   tables.Flights,
			FlightLookupTableNameEnv:              tables.FlightLookup,
			FlightPositionsTableNameEnv:           tables.FlightPositions,
			UsageBudgetTableNameEnv:               tables.UsageBudget,
			GeoJSONBucketNameEnv:                  "geojson",
			FetchTaskQueueURLEnv:                  "https://sqs.example/fetch",
			"FLIGHTAWARE_API_KEY":                 "test-api-key",
			FetchTaskDiagnosticQueueURLEnv:        "",
			FlightAwareEstimatedCostUSDPerCallEnv: "0.01",
		}
		return values[name]
	}
	processor, err := NewFetchProcessor(ctx, Dependencies{
		DynamoDB: dynamo,
		Objects:  awsintegration.NewMemoryObjectClient(),
		Queues:   queue,
		Secrets:  awsintegration.NewMemorySecretsClient(map[string]string{}),
	}, lookup, now)
	if err != nil {
		t.Fatalf("NewFetchProcessor() error = %v", err)
	}

	_, err = processor.Process(ctx, application.FetchTask{
		SchemaVersion:  1,
		TaskID:         "task-missing-fa",
		TaskType:       application.FetchTaskPosition,
		FlightID:       "iflg_missing_fa",
		RequestedAt:    "2026-04-29T00:00:00Z",
		Reason:         application.FetchReasonLowFrequencyPoll,
		IdempotencyKey: "task-missing-fa",
	}, application.ProcessFetchInput{Now: now, WorkerID: "fetcher"})
	if !errors.Is(err, application.ErrValidation) {
		t.Fatalf("Process() error = %v, want ErrValidation", err)
	}
	if messages := queue.Messages(); len(messages) != 0 {
		t.Fatalf("diagnostic queue messages = %#v, want none when diagnostic queue is not configured", messages)
	}
}

func TestFetchProcessorRuntimeWiringDoesNotRequireFlightAwareCredentialWhenFetchesAreDisabled(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 29, 0, 5, 0, 0, time.UTC)
	dynamo := awsintegration.NewMemoryDynamoDBClient()
	tables := awsintegration.DynamoDBTables{
		Flights:         "Flights",
		FlightLookup:    "FlightLookup",
		FlightPositions: "FlightPositions",
		UsageBudget:     "UsageBudget",
	}
	lookup := func(name string) string {
		values := map[string]string{
			FlightAwareFetchEnabledEnv:     "false",
			FlightAwareRealCallsEnabledEnv: "false",
			AirpathEnvironmentEnv:          "dev",
			FlightsTableNameEnv:            tables.Flights,
			FlightLookupTableNameEnv:       tables.FlightLookup,
			FlightPositionsTableNameEnv:    tables.FlightPositions,
			UsageBudgetTableNameEnv:        tables.UsageBudget,
			GeoJSONBucketNameEnv:           "geojson",
		}
		return values[name]
	}

	processor, err := NewFetchProcessor(ctx, Dependencies{
		DynamoDB: dynamo,
		Objects:  awsintegration.NewMemoryObjectClient(),
		Queues:   awsintegration.NewMemoryQueueClient(),
		Secrets:  awsintegration.NewMemorySecretsClient(map[string]string{}),
	}, lookup, now)
	if err != nil {
		t.Fatalf("NewFetchProcessor(disabled without credential) error = %v", err)
	}

	result, err := processor.Process(ctx, application.FetchTask{
		SchemaVersion:  1,
		TaskID:         "task-disabled",
		TaskType:       application.FetchTaskSummary,
		FlightID:       "ANA110",
		RequestedAt:    "2026-04-29T00:00:00Z",
		Reason:         application.FetchReasonSearchResultSeed,
		IdempotencyKey: "task-disabled",
	}, application.ProcessFetchInput{Now: now, WorkerID: "fetcher"})
	if err != nil {
		t.Fatalf("Process(disabled task) error = %v", err)
	}
	if !result.Skipped || result.SkipReason != "fetch_disabled" || result.ExternalFetchAttempted {
		t.Fatalf("result = %#v, want fetch_disabled skip without external attempt", result)
	}
}
