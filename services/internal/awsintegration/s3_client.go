package awsintegration

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

var ErrObjectNotFound = errors.New("object not found")

type ObjectClient interface {
	PutObject(context.Context, string, string, []byte) error
	GetObject(context.Context, string, string) ([]byte, error)
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
		if isS3ObjectNotFound(err) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}
	defer output.Body.Close()
	return io.ReadAll(output.Body)
}

func isS3ObjectNotFound(err error) bool {
	var noSuchKey *s3types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return true
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.ErrorCode() {
		case "NoSuchKey", "NotFound":
			return true
		}
	}
	return false
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
