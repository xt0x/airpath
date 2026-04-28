package awsintegration

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
)

var ErrObjectNotFound = errors.New("object not found")
var ErrSecretNotFound = errors.New("secret not found")

type DynamoDBClient interface {
	PutItem(context.Context, string, map[string]any) error
	GetItem(context.Context, string, string, string) (map[string]any, bool, error)
	QueryByPrefix(context.Context, string, string, string) ([]map[string]any, error)
}

type MemoryDynamoDBClient struct {
	mu     sync.RWMutex
	tables map[string]map[string]map[string]any
}

func NewMemoryDynamoDBClient() *MemoryDynamoDBClient {
	return &MemoryDynamoDBClient{tables: map[string]map[string]map[string]any{}}
}

func (c *MemoryDynamoDBClient) PutItem(_ context.Context, table string, item map[string]any) error {
	key := itemKey(item)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tables[table] == nil {
		c.tables[table] = map[string]map[string]any{}
	}
	c.tables[table][key] = cloneItem(item)
	return nil
}

func (c *MemoryDynamoDBClient) GetItem(_ context.Context, table string, hashName string, hashValue string) (map[string]any, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, item := range c.tables[table] {
		if stringValue(item[hashName]) == hashValue {
			return cloneItem(item), true, nil
		}
	}
	return nil, false, nil
}

func (c *MemoryDynamoDBClient) QueryByPrefix(_ context.Context, table string, keyName string, prefix string) ([]map[string]any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	items := []map[string]any{}
	for _, item := range c.tables[table] {
		if strings.HasPrefix(stringValue(item[keyName]), prefix) {
			items = append(items, cloneItem(item))
		}
	}
	sort.Slice(items, func(left, right int) bool {
		return itemKey(items[left]) < itemKey(items[right])
	})
	return items, nil
}

func itemKey(item map[string]any) string {
	for _, name := range []string{"flightId", "lookupKey", "budgetScope"} {
		if value := stringValue(item[name]); value != "" {
			if timestamp := stringValue(item["timestamp"]); timestamp != "" && name == "flightId" {
				return value + "#" + timestamp
			}
			return value
		}
	}
	return ""
}

func cloneItem(item map[string]any) map[string]any {
	cloned := make(map[string]any, len(item))
	for key, value := range item {
		cloned[key] = value
	}
	return cloned
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

type ObjectClient interface {
	PutObject(context.Context, string, string, []byte) error
	GetObject(context.Context, string, string) ([]byte, error)
}

type MemoryObjectClient struct {
	mu      sync.RWMutex
	objects map[string][]byte
}

func NewMemoryObjectClient() *MemoryObjectClient {
	return &MemoryObjectClient{objects: map[string][]byte{}}
}

func (c *MemoryObjectClient) PutObject(_ context.Context, bucket string, key string, body []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.objects[bucket+"/"+key] = append([]byte(nil), body...)
	return nil
}

func (c *MemoryObjectClient) GetObject(_ context.Context, bucket string, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	body, ok := c.objects[bucket+"/"+key]
	if !ok {
		return nil, ErrObjectNotFound
	}
	return append([]byte(nil), body...), nil
}

type QueueMessage struct {
	QueueURL string
	Body     string
}

type QueueClient interface {
	SendMessage(context.Context, string, string) error
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

type SecretsClient interface {
	GetSecretValue(context.Context, string) (string, error)
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
