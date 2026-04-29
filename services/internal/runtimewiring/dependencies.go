package runtimewiring

import (
	"context"

	"airpath/services/internal/awsintegration"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
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
