package awsintegration

import (
	"context"
	"testing"
)

func TestLoadFlightAwareAPIKeyPrefersEnvironmentValue(t *testing.T) {
	key, err := LoadFlightAwareAPIKey(context.Background(), mapEnvLookup(map[string]string{
		"FLIGHTAWARE_API_KEY":            "env-key",
		"FLIGHTAWARE_API_KEY_SECRET_ARN": "secret-ref",
	}), NewSecretsAdapter(NewMemorySecretsClient(map[string]string{"secret-ref": "secret-key"}), nil))
	if err != nil {
		t.Fatalf("LoadFlightAwareAPIKey() error = %v", err)
	}
	if key != "env-key" {
		t.Fatalf("key = %q, want env-key", key)
	}
}

func TestLoadFlightAwareAPIKeyFallsBackToSecretsManagerReference(t *testing.T) {
	key, err := LoadFlightAwareAPIKey(context.Background(), mapEnvLookup(map[string]string{
		"FLIGHTAWARE_API_KEY_SECRET_ARN": "secret-ref",
	}), NewSecretsAdapter(NewMemorySecretsClient(map[string]string{"secret-ref": "secret-key"}), nil))
	if err != nil {
		t.Fatalf("LoadFlightAwareAPIKey() error = %v", err)
	}
	if key != "secret-key" {
		t.Fatalf("key = %q, want secret-key", key)
	}
}

func TestLoadFlightAwareAPIKeyRequiresRuntimeSource(t *testing.T) {
	_, err := LoadFlightAwareAPIKey(context.Background(), mapEnvLookup(map[string]string{}), nil)
	if err == nil {
		t.Fatal("LoadFlightAwareAPIKey() error = nil, want missing runtime source")
	}
}

func mapEnvLookup(values map[string]string) EnvLookup {
	return func(name string) string {
		return values[name]
	}
}
