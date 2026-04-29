package awsintegration

import (
	"bytes"
	"context"
	"io"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type AWSDynamoDBClient struct {
	client *dynamodb.Client
}

func NewAWSDynamoDBClient(client *dynamodb.Client) *AWSDynamoDBClient {
	return &AWSDynamoDBClient{client: client}
}

func (c *AWSDynamoDBClient) PutItem(ctx context.Context, table string, item map[string]any) error {
	encoded := make(map[string]ddbtypes.AttributeValue, len(item))
	for key, value := range item {
		encoded[key] = encodeAttributeValue(value)
	}
	_, err := c.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      encoded,
	})
	return err
}

func (c *AWSDynamoDBClient) GetItem(ctx context.Context, table string, hashName string, hashValue string) (map[string]any, bool, error) {
	output, err := c.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(table),
		Key: map[string]ddbtypes.AttributeValue{
			hashName: &ddbtypes.AttributeValueMemberS{Value: hashValue},
		},
	})
	if err != nil {
		return nil, false, err
	}
	if len(output.Item) == 0 {
		return nil, false, nil
	}
	return decodeItem(output.Item), true, nil
}

func (c *AWSDynamoDBClient) QueryByPrefix(ctx context.Context, table string, keyName string, prefix string) ([]map[string]any, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(table),
	}
	if prefix != "" {
		input.ExpressionAttributeNames = map[string]string{"#key": keyName}
		input.ExpressionAttributeValues = map[string]ddbtypes.AttributeValue{
			":prefix": &ddbtypes.AttributeValueMemberS{Value: prefix},
		}
		input.FilterExpression = aws.String("begins_with(#key, :prefix)")
	}

	items := []map[string]any{}
	for {
		output, err := c.client.Scan(ctx, input)
		if err != nil {
			return nil, err
		}
		for _, item := range output.Items {
			items = append(items, decodeItem(item))
		}
		if len(output.LastEvaluatedKey) == 0 {
			return items, nil
		}
		input.ExclusiveStartKey = output.LastEvaluatedKey
	}
}

func encodeAttributeValue(value any) ddbtypes.AttributeValue {
	switch typed := value.(type) {
	case string:
		return &ddbtypes.AttributeValueMemberS{Value: typed}
	case bool:
		return &ddbtypes.AttributeValueMemberBOOL{Value: typed}
	case int:
		return &ddbtypes.AttributeValueMemberN{Value: strconv.Itoa(typed)}
	case int64:
		return &ddbtypes.AttributeValueMemberN{Value: strconv.FormatInt(typed, 10)}
	case float64:
		return &ddbtypes.AttributeValueMemberN{Value: strconv.FormatFloat(typed, 'f', -1, 64)}
	case nil:
		return &ddbtypes.AttributeValueMemberNULL{Value: true}
	default:
		return &ddbtypes.AttributeValueMemberS{Value: stringValue(value)}
	}
}

func decodeItem(item map[string]ddbtypes.AttributeValue) map[string]any {
	decoded := make(map[string]any, len(item))
	for key, value := range item {
		decoded[key] = decodeAttributeValue(value)
	}
	return decoded
}

func decodeAttributeValue(value ddbtypes.AttributeValue) any {
	switch typed := value.(type) {
	case *ddbtypes.AttributeValueMemberS:
		return typed.Value
	case *ddbtypes.AttributeValueMemberN:
		if integer, err := strconv.ParseInt(typed.Value, 10, 64); err == nil {
			return integer
		}
		if number, err := strconv.ParseFloat(typed.Value, 64); err == nil {
			return number
		}
		return typed.Value
	case *ddbtypes.AttributeValueMemberBOOL:
		return typed.Value
	default:
		return nil
	}
}

type AWSS3ObjectClient struct {
	client *s3.Client
}

func NewAWSS3ObjectClient(client *s3.Client) *AWSS3ObjectClient {
	return &AWSS3ObjectClient{client: client}
}

func (c *AWSS3ObjectClient) PutObject(ctx context.Context, bucket string, key string, body []byte) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})
	return err
}

func (c *AWSS3ObjectClient) GetObject(ctx context.Context, bucket string, key string) ([]byte, error) {
	output, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer output.Body.Close()
	return io.ReadAll(output.Body)
}

type AWSSQSQueueClient struct {
	client *sqs.Client
}

func NewAWSSQSQueueClient(client *sqs.Client) *AWSSQSQueueClient {
	return &AWSSQSQueueClient{client: client}
}

func (c *AWSSQSQueueClient) SendMessage(ctx context.Context, queueURL string, body string) error {
	_, err := c.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(body),
	})
	return err
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
		return "", err
	}
	return aws.ToString(output.SecretString), nil
}
