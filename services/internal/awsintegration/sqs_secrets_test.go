package awsintegration

import (
	"context"
	"strings"
	"testing"

	"airpath/services/internal/application"
)

func TestSQSFetchTaskQueueDedupesBeforeSending(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryQueueClient()
	queue := NewSQSFetchTaskQueue(client, "https://sqs.example/queue")
	task := application.FetchTask{
		SchemaVersion:  1,
		TaskID:         "task-1",
		TaskType:       application.FetchTaskRoute,
		FlightID:       "iflg_1",
		RequestedAt:    "2026-04-29T00:00:00Z",
		Reason:         application.FetchReasonUserManualRefresh,
		IdempotencyKey: "iflg_1:route",
	}

	first, err := queue.EnqueueFetchTask(ctx, task)
	if err != nil {
		t.Fatalf("first EnqueueFetchTask() error = %v", err)
	}
	second, err := queue.EnqueueFetchTask(ctx, task)
	if err != nil {
		t.Fatalf("second EnqueueFetchTask() error = %v", err)
	}

	if !first || second {
		t.Fatalf("dedupe results = %v, %v; want true, false", first, second)
	}
	if len(client.Messages()) != 1 {
		t.Fatalf("message count = %d, want 1", len(client.Messages()))
	}
	if !strings.Contains(client.Messages()[0].Body, `"taskType":"route"`) {
		t.Fatalf("message body = %s", client.Messages()[0].Body)
	}
}

func TestSecretsAdapterLoadsFlightAwareKeyWithoutLoggingSecretValue(t *testing.T) {
	ctx := context.Background()
	client := NewMemorySecretsClient(map[string]string{"flightaware/api-key": "super-secret-key"})
	logger := &memoryLogger{}
	adapter := NewSecretsAdapter(client, logger)

	key, err := adapter.GetFlightAwareAPIKey(ctx, "flightaware/api-key")
	if err != nil {
		t.Fatalf("GetFlightAwareAPIKey() error = %v", err)
	}
	if key != "super-secret-key" {
		t.Fatalf("key = %q, want raw secret", key)
	}
	if strings.Contains(logger.String(), "super-secret-key") {
		t.Fatalf("logger leaked secret: %s", logger.String())
	}
	if !strings.Contains(logger.String(), "flightaware/api-key") {
		t.Fatalf("logger did not include secret reference: %s", logger.String())
	}
}

type memoryLogger struct {
	lines []string
}

func (l *memoryLogger) Info(message string, fields map[string]string) {
	l.lines = append(l.lines, message+" "+fields["secretRef"])
}

func (l *memoryLogger) String() string {
	return strings.Join(l.lines, "\n")
}
