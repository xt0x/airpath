package awsintegration

import "context"

type Logger interface {
	Info(message string, fields map[string]string)
}

type SecretsAdapter struct {
	client SecretsClient
	logger Logger
}

func NewSecretsAdapter(client SecretsClient, logger Logger) *SecretsAdapter {
	return &SecretsAdapter{client: client, logger: logger}
}

func (a *SecretsAdapter) GetFlightAwareAPIKey(ctx context.Context, secretRef string) (string, error) {
	value, err := a.client.GetSecretValue(ctx, secretRef)
	if err != nil {
		return "", err
	}
	if a.logger != nil {
		a.logger.Info("loaded FlightAware API key secret", map[string]string{"secretRef": secretRef})
	}
	return value, nil
}
