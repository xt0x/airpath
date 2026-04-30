package awsintegration

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	secretsmanagertypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

var ErrSecretNotFound = errors.New("secret not found")

type SecretsClient interface {
	GetSecretValue(context.Context, string) (string, error)
}

type AWSSecretsClient struct {
	client *secretsmanager.Client
}

func NewAWSSecretsClient(client *secretsmanager.Client) *AWSSecretsClient {
	return &AWSSecretsClient{client: client}
}

func (c *AWSSecretsClient) GetSecretValue(ctx context.Context, secretRef string) (string, error) {
	output, err := c.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretRef),
	})
	if err != nil {
		var notFound *secretsmanagertypes.ResourceNotFoundException
		if errors.As(err, &notFound) {
			return "", ErrSecretNotFound
		}
		return "", err
	}
	return aws.ToString(output.SecretString), nil
}

type MemorySecretsClient struct {
	secrets map[string]string
}

func NewMemorySecretsClient(secrets map[string]string) *MemorySecretsClient {
	return &MemorySecretsClient{secrets: secrets}
}

func (c *MemorySecretsClient) GetSecretValue(_ context.Context, secretRef string) (string, error) {
	value, ok := c.secrets[secretRef]
	if !ok {
		return "", ErrSecretNotFound
	}
	return value, nil
}
