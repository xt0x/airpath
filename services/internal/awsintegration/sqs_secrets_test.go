package awsintegration

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"airpath/services/internal/application"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
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

func TestSQSFetchTaskQueueRejectsEmptyIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryQueueClient()
	queue := NewSQSFetchTaskQueue(client, "https://sqs.example/queue")

	enqueued, err := queue.EnqueueFetchTask(ctx, application.FetchTask{
		SchemaVersion: 1,
		TaskID:        "task-1",
		TaskType:      application.FetchTaskRoute,
		FlightID:      "iflg_1",
		RequestedAt:   "2026-04-29T00:00:00Z",
		Reason:        application.FetchReasonUserManualRefresh,
	})
	if !errors.Is(err, application.ErrValidation) {
		t.Fatalf("EnqueueFetchTask() error = %v, want ErrValidation", err)
	}
	if enqueued {
		t.Fatal("EnqueueFetchTask() enqueued = true, want false for empty idempotency key")
	}
	if len(client.Messages()) != 0 {
		t.Fatalf("message count = %d, want 0", len(client.Messages()))
	}
}

func TestSQSFetchTaskQueueDedupesAcrossQueueInstancesWithPersistentStore(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryQueueClient()
	store := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{FlightLookup: "FlightLookup"})
	firstQueue := NewPersistentSQSFetchTaskQueue(client, "https://sqs.example/queue", store)
	secondQueue := NewPersistentSQSFetchTaskQueue(client, "https://sqs.example/queue", store)
	task := application.FetchTask{
		SchemaVersion:  1,
		TaskID:         "task-1",
		TaskType:       application.FetchTaskRoute,
		FlightID:       "iflg_1",
		RequestedAt:    "2026-04-29T00:00:00Z",
		Reason:         application.FetchReasonUserManualRefresh,
		IdempotencyKey: "refresh:iflg_1:route:2026-04-29T00:00:00Z",
	}

	first, err := firstQueue.EnqueueFetchTask(ctx, task)
	if err != nil {
		t.Fatalf("first EnqueueFetchTask() error = %v", err)
	}
	second, err := secondQueue.EnqueueFetchTask(ctx, task)
	if err != nil {
		t.Fatalf("second EnqueueFetchTask() error = %v", err)
	}

	if !first || second {
		t.Fatalf("dedupe results = %v, %v; want true, false across queue instances", first, second)
	}
	if len(client.Messages()) != 1 {
		t.Fatalf("message count = %d, want 1", len(client.Messages()))
	}
}

func TestPersistentSQSFetchTaskQueueAllowsExpiredReservationsToBeRequeued(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryQueueClient()
	dynamo := NewMemoryDynamoDBClient()
	store := NewDynamoDBRepository(dynamo, DynamoDBTables{FlightLookup: "FlightLookup"})
	queue := NewPersistentSQSFetchTaskQueue(client, "https://sqs.example/queue", store)
	task := application.FetchTask{
		SchemaVersion:  1,
		TaskID:         "task-1",
		TaskType:       application.FetchTaskPosition,
		FlightID:       "iflg_1",
		RequestedAt:    "2026-04-29T00:00:00Z",
		Reason:         application.FetchReasonLowFrequencyPoll,
		IdempotencyKey: "poll:iflg_1:position:2026-04-29T00:00:00Z",
	}
	if err := dynamo.PutItem(ctx, "FlightLookup", map[string]any{
		"lookupType": "fetchTaskIdempotency",
		"lookupKey":  "fetchTaskIdempotency#" + task.IdempotencyKey,
		"flightId":   task.IdempotencyKey,
		"ttl":        int64(1),
	}); err != nil {
		t.Fatalf("seed expired idempotency row: %v", err)
	}

	enqueued, err := queue.EnqueueFetchTask(ctx, task)
	if err != nil {
		t.Fatalf("EnqueueFetchTask() error = %v", err)
	}
	if !enqueued {
		t.Fatal("EnqueueFetchTask() enqueued = false, want expired reservation to be replaced")
	}
	if len(client.Messages()) != 1 {
		t.Fatalf("message count = %d, want 1", len(client.Messages()))
	}
	item, ok, err := dynamo.GetItem(ctx, "FlightLookup", "lookupKey", "fetchTaskIdempotency#"+task.IdempotencyKey)
	if err != nil {
		t.Fatalf("GetItem() error = %v", err)
	}
	if !ok {
		t.Fatal("idempotency row not found")
	}
	if ttl := numberValue(item["ttl"]); ttl <= 1 {
		t.Fatalf("ttl = %v, want refreshed future expiry", ttl)
	}
}

func TestSQSFetchTaskQueueLocalDedupeExpires(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryQueueClient()
	queue := NewSQSFetchTaskQueue(client, "https://sqs.example/queue")
	now := time.Unix(100, 0).UTC()
	queue.now = func() time.Time { return now }
	task := application.FetchTask{
		SchemaVersion:  1,
		TaskID:         "task-1",
		TaskType:       application.FetchTaskPosition,
		FlightID:       "iflg_1",
		RequestedAt:    "2026-04-29T00:00:00Z",
		Reason:         application.FetchReasonLowFrequencyPoll,
		IdempotencyKey: "poll:iflg_1:position:2026-04-29T00:00:00Z",
	}

	first, err := queue.EnqueueFetchTask(ctx, task)
	if err != nil {
		t.Fatalf("first EnqueueFetchTask() error = %v", err)
	}
	second, err := queue.EnqueueFetchTask(ctx, task)
	if err != nil {
		t.Fatalf("second EnqueueFetchTask() error = %v", err)
	}
	now = now.Add(fetchTaskIdempotencyTTL + time.Second)
	third, err := queue.EnqueueFetchTask(ctx, task)
	if err != nil {
		t.Fatalf("third EnqueueFetchTask() error = %v", err)
	}
	if !first || second || !third {
		t.Fatalf("dedupe results = %v, %v, %v; want true, false, true after expiry", first, second, third)
	}
	if len(client.Messages()) != 2 {
		t.Fatalf("message count = %d, want 2", len(client.Messages()))
	}
}

func TestSQSFetchTaskQueueRetriesAfterLocalSendFailure(t *testing.T) {
	ctx := context.Background()
	client := &flakyQueueClient{err: errors.New("temporary sqs failure")}
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
	if err == nil || first {
		t.Fatalf("first EnqueueFetchTask() = %v, %v; want send error", first, err)
	}
	client.err = nil
	second, err := queue.EnqueueFetchTask(ctx, task)
	if err != nil {
		t.Fatalf("second EnqueueFetchTask() error = %v", err)
	}
	if !second || len(client.messages) != 1 {
		t.Fatalf("retry result = %v messages=%d, want one successful retry", second, len(client.messages))
	}
}

func TestPersistentSQSFetchTaskQueueReleasesReservationAfterSendFailure(t *testing.T) {
	ctx := context.Background()
	client := &flakyQueueClient{err: errors.New("temporary sqs failure")}
	store := NewDynamoDBRepository(NewMemoryDynamoDBClient(), DynamoDBTables{FlightLookup: "FlightLookup"})
	queue := NewPersistentSQSFetchTaskQueue(client, "https://sqs.example/queue", store)
	task := application.FetchTask{
		SchemaVersion:  1,
		TaskID:         "task-1",
		TaskType:       application.FetchTaskRoute,
		FlightID:       "iflg_1",
		RequestedAt:    "2026-04-29T00:00:00Z",
		Reason:         application.FetchReasonUserManualRefresh,
		IdempotencyKey: "refresh:iflg_1:route:2026-04-29T00:00:00Z",
	}

	first, err := queue.EnqueueFetchTask(ctx, task)
	if err == nil || first {
		t.Fatalf("first EnqueueFetchTask() = %v, %v; want send error", first, err)
	}
	client.err = nil
	second, err := queue.EnqueueFetchTask(ctx, task)
	if err != nil {
		t.Fatalf("second EnqueueFetchTask() error = %v", err)
	}
	if !second || len(client.messages) != 1 {
		t.Fatalf("retry result = %v messages=%d, want one successful retry", second, len(client.messages))
	}
}

func TestSQSDiagnosticQueueSendsSafeFetchFailureMetadata(t *testing.T) {
	ctx := context.Background()
	client := NewMemoryQueueClient()
	queue := NewSQSDiagnosticQueue(client, "https://sqs.example/fetch-task-dlq")

	if err := queue.RecordFetchTaskDiagnostic(ctx, application.FetchTaskDiagnostic{
		TaskID:    "task-1",
		TaskType:  application.FetchTaskPosition,
		FlightID:  "iflg_1",
		ErrorCode: "rate_limited",
		Message:   "FlightAware rate limit is active",
		FailedAt:  "2026-04-29T00:00:00Z",
	}); err != nil {
		t.Fatalf("RecordFetchTaskDiagnostic() error = %v", err)
	}

	messages := client.Messages()
	if len(messages) != 1 {
		t.Fatalf("message count = %d, want 1", len(messages))
	}
	if messages[0].QueueURL != "https://sqs.example/fetch-task-dlq" {
		t.Fatalf("queue URL = %q", messages[0].QueueURL)
	}
	if strings.Contains(messages[0].Body, "fa_") || strings.Contains(messages[0].Body, "x-apikey") {
		t.Fatalf("diagnostic body contains unsafe metadata: %s", messages[0].Body)
	}
	if !strings.Contains(messages[0].Body, `"errorCode":"rate_limited"`) {
		t.Fatalf("diagnostic body = %s, want typed error code", messages[0].Body)
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

func TestAWSSecretsClientMapsMissingSecretToSecretNotFound(t *testing.T) {
	client := secretsmanager.New(secretsmanager.Options{
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider("key", "secret", ""),
		BaseEndpoint: aws.String("https://secretsmanager.test"),
		HTTPClient: roundTripClient(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Header: http.Header{
					"Content-Type":         []string{"application/x-amz-json-1.1"},
					"X-Amzn-Errortype":     []string{"ResourceNotFoundException"},
					"X-Amzn-Requestid":     []string{"request-1"},
					"X-Amz-Request-Id":     []string{"request-1"},
					"X-Amzn-Trace-Id":      []string{"Root=1-request"},
					"X-Amz-Target":         []string{"secretsmanager.GetSecretValue"},
					"X-Amz-Content-Sha256": []string{"UNSIGNED-PAYLOAD"},
				},
				Body:    io.NopCloser(strings.NewReader(`{"__type":"ResourceNotFoundException","message":"missing"}`)),
				Request: request,
			}, nil
		}),
	})
	secrets := NewAWSSecretsClient(client)

	_, err := secrets.GetSecretValue(context.Background(), "missing-secret")
	if !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("GetSecretValue() error = %v, want ErrSecretNotFound", err)
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

type flakyQueueClient struct {
	err      error
	messages []QueueMessage
}

func (c *flakyQueueClient) SendMessage(_ context.Context, queueURL string, body string) error {
	if c.err != nil {
		return c.err
	}
	c.messages = append(c.messages, QueueMessage{QueueURL: queueURL, Body: body})
	return nil
}
