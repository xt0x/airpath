package main

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestHandleAPIRequestReturnsHealthResponseWithoutSecrets(t *testing.T) {
	t.Setenv("AIRPATH_ENVIRONMENT", "dev")
	t.Setenv("FLIGHTAWARE_FETCH_ENABLED", "false")
	t.Setenv("FLIGHTAWARE_API_KEY_SECRET_ARN", "arn:aws:secretsmanager:ap-northeast-1:123456789012:secret:flightaware")

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
	if body["flightawareApiKeySecretArn"] != nil {
		t.Fatalf("body leaked secret ARN: %v", body["flightawareApiKeySecretArn"])
	}
}

func TestBoolEnvDefaultsToFalseUnlessExplicitlyTrue(t *testing.T) {
	t.Setenv("BOOL_ENV_EXPLICIT_TRUE", "true")
	t.Setenv("BOOL_ENV_UPPER_TRUE", "TRUE")
	t.Setenv("BOOL_ENV_FALSE", "false")

	if !boolEnv("BOOL_ENV_EXPLICIT_TRUE") {
		t.Fatal("boolEnv(true) = false, want true")
	}
	if !boolEnv("BOOL_ENV_UPPER_TRUE") {
		t.Fatal("boolEnv(TRUE) = false, want true")
	}
	if boolEnv("BOOL_ENV_FALSE") {
		t.Fatal("boolEnv(false) = true, want false")
	}
	if boolEnv("BOOL_ENV_MISSING") {
		t.Fatal("boolEnv(missing) = true, want false")
	}
}
