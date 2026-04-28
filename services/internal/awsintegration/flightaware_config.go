package awsintegration

import (
	"context"
	"errors"
	"strings"
)

const (
	FlightAwareAPIKeyEnv       = "FLIGHTAWARE_API_KEY"
	FlightAwareAPIKeySecretEnv = "FLIGHTAWARE_API_KEY_SECRET_ARN"
)

type EnvLookup func(string) string

func LoadFlightAwareAPIKey(ctx context.Context, lookup EnvLookup, secrets *SecretsAdapter) (string, error) {
	if lookup == nil {
		return "", errors.New("environment lookup is required")
	}
	if key := strings.TrimSpace(lookup(FlightAwareAPIKeyEnv)); key != "" {
		return key, nil
	}
	secretRef := strings.TrimSpace(lookup(FlightAwareAPIKeySecretEnv))
	if secretRef == "" || secrets == nil {
		return "", errors.New("FlightAware API key must be provided by environment or Secrets Manager")
	}
	return secrets.GetFlightAwareAPIKey(ctx, secretRef)
}
