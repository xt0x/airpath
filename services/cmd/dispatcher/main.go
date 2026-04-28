package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
)

type dispatcherResponse struct {
	Service          string `json:"service"`
	Environment      string `json:"environment"`
	Mode             string `json:"mode"`
	NoopFetchEnabled bool   `json:"noopFetchEnabled"`
	QueueConfigured  bool   `json:"queueConfigured"`
	EnqueueAttempted bool   `json:"enqueueAttempted"`
}

func main() {
	lambda.Start(handleDispatcherEvent)
}

func handleDispatcherEvent(json.RawMessage) (dispatcherResponse, error) {
	return dispatcherResponse{
		Service:          "dispatcher",
		Environment:      os.Getenv("AIRPATH_ENVIRONMENT"),
		Mode:             os.Getenv("DISPATCHER_MODE"),
		NoopFetchEnabled: strings.EqualFold(os.Getenv("NOOP_FETCH_ENABLED"), "true"),
		QueueConfigured:  os.Getenv("FETCH_TASK_QUEUE_URL") != "",
		EnqueueAttempted: false,
	}, nil
}
