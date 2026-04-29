package runtimewiring

import (
	"testing"
	"time"
)

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
