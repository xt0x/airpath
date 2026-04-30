package awsintegration

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"airpath/services/internal/application"
)

type SQSFetchTaskQueue struct {
	client      QueueClient
	queueURL    string
	idempotency FetchTaskIdempotencyStore
	mu          sync.Mutex
	seen        map[string]int64
	now         func() time.Time
}

type SQSDiagnosticQueue struct {
	client   QueueClient
	queueURL string
}

type FetchTaskIdempotencyStore interface {
	ReserveFetchTaskIdempotency(context.Context, string) (bool, error)
	ReleaseFetchTaskIdempotency(context.Context, string) error
}

func NewSQSFetchTaskQueue(client QueueClient, queueURL string) *SQSFetchTaskQueue {
	return &SQSFetchTaskQueue{
		client:   client,
		queueURL: queueURL,
		seen:     map[string]int64{},
		now:      time.Now,
	}
}

func NewPersistentSQSFetchTaskQueue(client QueueClient, queueURL string, idempotency FetchTaskIdempotencyStore) *SQSFetchTaskQueue {
	queue := NewSQSFetchTaskQueue(client, queueURL)
	queue.idempotency = idempotency
	return queue
}

func (q *SQSFetchTaskQueue) EnqueueFetchTask(ctx context.Context, task application.FetchTask) (bool, error) {
	if task.IdempotencyKey == "" {
		return false, application.ErrValidation
	}
	now := q.now().UTC()
	if !q.reserveLocal(task.IdempotencyKey, now) {
		return false, nil
	}

	reserved := false
	if q.idempotency != nil {
		// Local idempotency protects this process; persistent idempotency protects
		// concurrent dispatchers and retrying Lambda invocations.
		var err error
		reserved, err = q.idempotency.ReserveFetchTaskIdempotency(ctx, task.IdempotencyKey)
		if err != nil {
			q.forget(ctx, task.IdempotencyKey, false)
			return false, err
		}
		if !reserved {
			q.forget(ctx, task.IdempotencyKey, false)
			return false, nil
		}
	}

	body, err := json.Marshal(task)
	if err != nil {
		q.forget(ctx, task.IdempotencyKey, reserved)
		return false, err
	}
	if err := q.client.SendMessage(ctx, q.queueURL, string(body)); err != nil {
		q.forget(ctx, task.IdempotencyKey, reserved)
		return false, err
	}
	return true, nil
}

func (q *SQSFetchTaskQueue) reserveLocal(idempotencyKey string, now time.Time) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	expiresAtUnix, ok := q.seen[idempotencyKey]
	if ok && expiresAtUnix > now.Unix() {
		return false
	}
	q.seen[idempotencyKey] = now.Add(fetchTaskIdempotencyTTL).Unix()
	return true
}

func (q *SQSFetchTaskQueue) forget(ctx context.Context, idempotencyKey string, reserved bool) {
	q.mu.Lock()
	delete(q.seen, idempotencyKey)
	q.mu.Unlock()
	if reserved && q.idempotency != nil {
		_ = q.idempotency.ReleaseFetchTaskIdempotency(ctx, idempotencyKey)
	}
}

func NewSQSDiagnosticQueue(client QueueClient, queueURL string) *SQSDiagnosticQueue {
	return &SQSDiagnosticQueue{
		client:   client,
		queueURL: queueURL,
	}
}

func (q *SQSDiagnosticQueue) RecordFetchTaskDiagnostic(ctx context.Context, diagnostic application.FetchTaskDiagnostic) error {
	body, err := json.Marshal(diagnostic)
	if err != nil {
		return err
	}
	return q.client.SendMessage(ctx, q.queueURL, string(body))
}
