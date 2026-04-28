package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"airpath/services/internal/application"

	"github.com/aws/aws-lambda-go/events"
)

func TestAdapterMapsAPIGatewaySearchEventToApplicationResponse(t *testing.T) {
	app := &stubApplication{
		searchResponse: application.FlightSearchResponse{
			Items: []application.FlightSummaryItem{{FlightID: "iflg_1", Ident: "ANA110", Origin: "RJTT", Destination: "KJFK"}},
			Cache: application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
		},
	}
	adapter := NewAdapter(app)

	response, err := adapter.Handle(context.Background(), events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{RequestID: "req-1"},
		RawPath:        "/v1/flights/search",
		QueryStringParameters: map[string]string{
			"ident": "ANA110",
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("status = %d, want 200 body=%s", response.StatusCode, response.Body)
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(response.Body), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	items := body["items"].([]any)
	first := items[0].(map[string]any)
	if first["ident"] != "ANA110" {
		t.Fatalf("first item = %#v", first)
	}
}

func TestAdapterMapsApplicationErrorToTypedResponse(t *testing.T) {
	adapter := NewAdapter(&stubApplication{searchErr: application.ErrBudgetExceeded})

	response, err := adapter.Handle(context.Background(), events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{RequestID: "req-budget"},
		RawPath:        "/v1/flights/search",
		QueryStringParameters: map[string]string{
			"ident": "ANA110",
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if response.StatusCode != 503 {
		t.Fatalf("status = %d, want 503", response.StatusCode)
	}
	if !jsonContains(response.Body, "flightaware_budget_exceeded") || !jsonContains(response.Body, "req-budget") {
		t.Fatalf("body = %s", response.Body)
	}
}

func TestAdapterRejectsUnsupportedRouteWithTypedResponse(t *testing.T) {
	adapter := NewAdapter(&stubApplication{})

	response, err := adapter.Handle(context.Background(), events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{RequestID: "req-missing"},
		RawPath:        "/v1/unknown",
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if response.StatusCode != 404 {
		t.Fatalf("status = %d, want 404", response.StatusCode)
	}
}

type stubApplication struct {
	searchResponse application.FlightSearchResponse
	searchErr      error
}

func (a *stubApplication) SearchFlights(_ context.Context, input application.SearchFlightsInput) (application.FlightSearchResponse, error) {
	if input.Ident == "" {
		return application.FlightSearchResponse{}, errors.New("missing ident")
	}
	return a.searchResponse, a.searchErr
}

func jsonContains(body string, value string) bool {
	var parsed any
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return false
	}
	encoded, _ := json.Marshal(parsed)
	return string(encoded) != "" && contains(string(encoded), value)
}

func contains(value string, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
