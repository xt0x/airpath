package awsintegration

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type QueueMessage struct {
	QueueURL string
	Body     string
}

type QueueClient interface {
	SendMessage(context.Context, string, string) error
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

type MemoryQueueClient struct {
	mu       sync.RWMutex
	messages []QueueMessage
}

func NewMemoryQueueClient() *MemoryQueueClient {
	return &MemoryQueueClient{}
}

func (c *MemoryQueueClient) SendMessage(_ context.Context, queueURL string, body string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, QueueMessage{QueueURL: queueURL, Body: body})
	return nil
}

func (c *MemoryQueueClient) Messages() []QueueMessage {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]QueueMessage(nil), c.messages...)
}
