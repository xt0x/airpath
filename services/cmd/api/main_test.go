package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestHandleAPIRequestReturnsHealthResponseWithoutSecrets(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "false")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "false")
	t.Setenv("FLIGHTAWARE_API_KEY_SECRET_ARN", "arn:aws:secretsmanager:ap-northeast-1:123456789012:secret:flightaware")
	t.Setenv("AIRPATH_PERSONAL_DEMO_NOTICE", "personal non-commercial low-frequency dev")

	response, err := handleAPIRequest(events.APIGatewayV2HTTPRequest{
		RawPath: "/v1/health",
	})
	if err != nil {
		t.Fatalf("handleAPIRequest() error = %v", err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("StatusCode = %d, want 200", response.StatusCode)
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(response.Body), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["service"] != "api" {
		t.Fatalf("service = %v, want api", body["service"])
	}
	if body["environment"] != "dev" {
		t.Fatalf("environment = %v, want dev", body["environment"])
	}
	if body["flightawareFetchEnabled"] != false {
		t.Fatalf("flightawareFetchEnabled = %v, want false", body["flightawareFetchEnabled"])
	}
	if body["flightawareRealCallsEnabled"] != false {
		t.Fatalf("flightawareRealCallsEnabled = %v, want false", body["flightawareRealCallsEnabled"])
	}
	if body["externalFlightAwareCallsAllowed"] != false {
		t.Fatalf("externalFlightAwareCallsAllowed = %v, want false", body["externalFlightAwareCallsAllowed"])
	}
	if body["personalDemoNotice"] != "personal non-commercial low-frequency dev" {
		t.Fatalf("personalDemoNotice = %v", body["personalDemoNotice"])
	}
	if body["flightawareApiKeySecretArn"] != nil {
		t.Fatalf("body leaked secret ARN: %v", body["flightawareApiKeySecretArn"])
	}
}

func TestHandleAPIRequestReportsManualFetchControlConfig(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "false")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "true")
	t.Setenv("FLIGHTAWARE_FETCH_DISABLED_REASON", "operator_budget_stop")

	response, err := handleAPIRequest(events.APIGatewayV2HTTPRequest{
		RawPath: "/v1/admin/fetch-control",
	})
	if err != nil {
		t.Fatalf("handleAPIRequest(fetch-control) error = %v", err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("StatusCode = %d, want 200", response.StatusCode)
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(response.Body), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["fetchingEnabled"] != false {
		t.Fatalf("fetchingEnabled = %v, want false", body["fetchingEnabled"])
	}
	if body["realCallsEnabled"] != true {
		t.Fatalf("realCallsEnabled = %v, want true", body["realCallsEnabled"])
	}
	if body["externalCallsAllowed"] != false {
		t.Fatalf("externalCallsAllowed = %v, want false", body["externalCallsAllowed"])
	}
	if body["reason"] != "operator_budget_stop" {
		t.Fatalf("reason = %v", body["reason"])
	}

	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "true")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "true")
	t.Setenv("FLIGHTAWARE_FETCH_DISABLED_REASON", "")
	response, err = handleAPIRequest(events.APIGatewayV2HTTPRequest{
		RawPath: "/v1/admin/fetch-control",
	})
	if err != nil {
		t.Fatalf("handleAPIRequest(fetch-control resumed) error = %v", err)
	}
	if err := json.Unmarshal([]byte(response.Body), &body); err != nil {
		t.Fatalf("unmarshal resumed body: %v", err)
	}
	if body["fetchingEnabled"] != true || body["realCallsEnabled"] != true || body["externalCallsAllowed"] != true || body["reason"] != "" {
		t.Fatalf("resumed body = %#v, want enabled without reason", body)
	}
}

func TestHandleAPIRequestRoutesFlightSearchThroughHTTPAdapter(t *testing.T) {
	t.Setenv("AIRPATH_RUNTIME_BACKEND", "memory")
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "true")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "false")

	response, err := handleAPIRequest(events.APIGatewayV2HTTPRequest{
		RawPath: "/v1/flights/search",
		QueryStringParameters: map[string]string{
			"ident": "ANA110",
		},
		RequestContext: events.APIGatewayV2HTTPRequestContext{RequestID: "req-search"},
	})
	if err != nil {
		t.Fatalf("handleAPIRequest(search) error = %v", err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("StatusCode = %d, want 200 body=%s", response.StatusCode, response.Body)
	}
	if !strings.Contains(response.Body, `"items"`) || !strings.Contains(response.Body, `"cache"`) {
		t.Fatalf("search body = %s, want application response shape", response.Body)
	}
}

func TestRuntimeBackendDefaultsToAWSOnlyInsideLambda(t *testing.T) {
	if got := runtimeBackend(func(string) string { return "" }); got != runtimeBackendMemory {
		t.Fatalf("runtimeBackend(outside lambda) = %q, want memory", got)
	}

	got := runtimeBackend(func(name string) string {
		if name == "AWS_LAMBDA_FUNCTION_NAME" {
			return "airpath-dev-api"
		}
		return ""
	})
	if got != runtimeBackendAWS {
		t.Fatalf("runtimeBackend(inside lambda) = %q, want aws", got)
	}

	got = runtimeBackend(func(name string) string {
		if name == "AIRPATH_RUNTIME_BACKEND" {
			return "memory"
		}
		if name == "AWS_LAMBDA_FUNCTION_NAME" {
			return "airpath-dev-api"
		}
		return ""
	})
	if got != runtimeBackendMemory {
		t.Fatalf("runtimeBackend(explicit memory) = %q, want memory", got)
	}
}
