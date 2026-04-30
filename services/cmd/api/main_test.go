package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"airpath/services/internal/cmdsupport"
	"github.com/aws/aws-lambda-go/events"
)

// These tests exercise the Lambda entrypoint directly because this package owns
// API Gateway method handling, CORS headers, and runtime configuration loading.
func TestHandleAPIRequestReturnsHealthResponseWithoutSecrets(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "false")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "false")
	t.Setenv("FLIGHTAWARE_API_KEY_SECRET_ARN", "arn:aws:secretsmanager:ap-northeast-1:123456789012:secret:flightaware")
	t.Setenv("AIRPATH_PERSONAL_DEMO_NOTICE", "personal non-commercial low-frequency dev")

	response, err := handleAPIRequest(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath: "/v1/health",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: "GET"},
		},
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

func TestHandleAPIRequestReportsFlightAwareCredentialPresentFromEnvironmentKey(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FLIGHTAWARE_API_KEY", "local-api-key")

	response, err := handleAPIRequest(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath: "/v1/health",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: "GET"},
		},
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
	if body["flightawareSecretPresent"] != true {
		t.Fatalf("flightawareSecretPresent = %v, want true for FLIGHTAWARE_API_KEY source", body["flightawareSecretPresent"])
	}
	if strings.Contains(response.Body, "local-api-key") {
		t.Fatalf("health body leaked API key: %s", response.Body)
	}
}

func TestHandleAPIRequestReportsManualFetchControlConfig(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "false")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "true")
	t.Setenv("FLIGHTAWARE_FETCH_DISABLED_REASON", "operator_budget_stop")

	response, err := handleAPIRequest(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath: "/v1/admin/fetch-control",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: "GET"},
		},
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
	response, err = handleAPIRequest(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath: "/v1/admin/fetch-control",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: "GET"},
		},
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

func TestHandleAPIRequestGatesStaticReadOnlyRoutesByMethodAndCORS(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")

	tests := []struct {
		name   string
		path   string
		method string
		status int
	}{
		{name: "health options", path: "/v1/health", method: "OPTIONS", status: 204},
		{name: "fetch control options", path: "/v1/admin/fetch-control", method: "OPTIONS", status: 204},
		{name: "health post", path: "/v1/health", method: "POST", status: 404},
		{name: "fetch control delete", path: "/v1/admin/fetch-control", method: "DELETE", status: 404},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			response, err := handleAPIRequest(context.Background(), events.APIGatewayV2HTTPRequest{
				RawPath: testCase.path,
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					RequestID: "req-static-route",
					HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
						Method: testCase.method,
					},
				},
			})
			if err != nil {
				t.Fatalf("handleAPIRequest() error = %v", err)
			}
			if response.StatusCode != testCase.status {
				t.Fatalf("StatusCode = %d, want %d body=%s", response.StatusCode, testCase.status, response.Body)
			}
			if response.Headers["access-control-allow-origin"] != "*" {
				t.Fatalf("access-control-allow-origin = %q, want *", response.Headers["access-control-allow-origin"])
			}
			if response.Headers["access-control-allow-methods"] == "" {
				t.Fatal("access-control-allow-methods header is empty")
			}
		})
	}
}

func TestHandleAPIRequestRoutesFlightSearchThroughHTTPAdapter(t *testing.T) {
	t.Setenv("AIRPATH_RUNTIME_BACKEND", "memory")
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "true")
	t.Setenv("FLIGHTAWARE_REAL_CALLS_ENABLED", "true")

	response, err := handleAPIRequest(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath: "/v1/flights/search",
		QueryStringParameters: map[string]string{
			"ident": "ANA110",
		},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "req-search",
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "GET",
			},
		},
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
	if got := cmdsupport.RuntimeBackend(func(string) string { return "" }); got != runtimeBackendMemory {
		t.Fatalf("runtimeBackend(outside lambda) = %q, want memory", got)
	}

	got := cmdsupport.RuntimeBackend(func(name string) string {
		if name == "AWS_LAMBDA_FUNCTION_NAME" {
			return "airpath-dev-api"
		}
		return ""
	})
	if got != runtimeBackendAWS {
		t.Fatalf("runtimeBackend(inside lambda) = %q, want aws", got)
	}

	got = cmdsupport.RuntimeBackend(func(name string) string {
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
