package awsintegration

import (
	"context"
	"encoding/json"
	"sync"

	"airpath/services/internal/application"
)

type SQSFetchTaskQueue struct {
	client   QueueClient
	queueURL string
	mu       sync.Mutex
	seen     map[string]struct{}
}

func NewSQSFetchTaskQueue(client QueueClient, queueURL string) *SQSFetchTaskQueue {
	return &SQSFetchTaskQueue{
		client:   client,
		queueURL: queueURL,
		seen:     map[string]struct{}{},
	}
}

func (q *SQSFetchTaskQueue) EnqueueFetchTask(ctx context.Context, task application.FetchTask) (bool, error) {
	q.mu.Lock()
	if _, ok := q.seen[task.IdempotencyKey]; ok {
		q.mu.Unlock()
		return false, nil
	}
	q.seen[task.IdempotencyKey] = struct{}{}
	q.mu.Unlock()

	body, err := json.Marshal(task)
	if err != nil {
		return false, err
	}
	if err := q.client.SendMessage(ctx, q.queueURL, string(body)); err != nil {
		return false, err
	}
	return true, nil
}
