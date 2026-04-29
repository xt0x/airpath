package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"

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

func TestAdapterMapsFlightDetailMapRefreshAndUsageRoutes(t *testing.T) {
	app := &stubApplication{
		detailResponse: application.FlightDetailResponse{
			Flight: testFlight("iflg_1"),
			Cache:  application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
		},
		mapResponse: application.FlightMapDataResponse{
			FlightID: "iflg_1",
			Planned:  application.MapLayer{Source: application.MapSourceFlightAwareRoute, Available: false},
			Actual:   application.MapLayer{Source: application.MapSourceFlightAwareTrack, Available: false},
			Current:  application.MapLayer{Source: application.MapSourceFlightAwarePosition, Available: false},
			Cache:    application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
		},
		refreshResponse: application.FlightRefreshResponse{
			FlightID:      "iflg_1",
			AcceptedTasks: []application.FetchTask{},
			Cache:         application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
		},
		usageResponse: application.UsageStatus{
			Budget:          application.UsageBudgetStatus{Environment: "dev", Month: "2026-04", Currency: "USD", SoftStopThreshold: 4},
			FetchingEnabled: true,
		},
	}
	adapter := NewAdapter(app)

	detail, err := adapter.Handle(context.Background(), request(http.MethodGet, "/v1/flights/iflg_1", ""))
	if err != nil {
		t.Fatalf("detail Handle() error = %v", err)
	}
	if detail.StatusCode != 200 || !jsonContains(detail.Body, "iflg_1") {
		t.Fatalf("detail response = %#v", detail)
	}

	mapData, err := adapter.Handle(context.Background(), request(http.MethodGet, "/v1/flights/iflg_1/map-data", ""))
	if err != nil {
		t.Fatalf("map Handle() error = %v", err)
	}
	if mapData.StatusCode != 200 || !jsonContains(mapData.Body, "planned") {
		t.Fatalf("map response = %#v", mapData)
	}

	refresh, err := adapter.Handle(context.Background(), request(http.MethodPost, "/v1/flights/iflg_1/refresh", `{"taskTypes":["route"],"clientReason":"user_manual_refresh"}`))
	if err != nil {
		t.Fatalf("refresh Handle() error = %v", err)
	}
	if refresh.StatusCode != 202 || app.refreshInput.TaskTypes[0] != application.FetchTaskRoute {
		t.Fatalf("refresh response = %#v input=%#v", refresh, app.refreshInput)
	}

	usage, err := adapter.Handle(context.Background(), request(http.MethodGet, "/v1/usage/status", ""))
	if err != nil {
		t.Fatalf("usage Handle() error = %v", err)
	}
	if usage.StatusCode != 200 || !jsonContains(usage.Body, "fetchingEnabled") {
		t.Fatalf("usage response = %#v", usage)
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
	searchResponse  application.FlightSearchResponse
	searchErr       error
	detailResponse  application.FlightDetailResponse
	mapResponse     application.FlightMapDataResponse
	refreshResponse application.FlightRefreshResponse
	refreshInput    application.FlightRefreshInput
	usageResponse   application.UsageStatus
}

func (a *stubApplication) SearchFlights(_ context.Context, input application.SearchFlightsInput) (application.FlightSearchResponse, error) {
	if input.Ident == "" {
		return application.FlightSearchResponse{}, errors.New("missing ident")
	}
	return a.searchResponse, a.searchErr
}

func (a *stubApplication) GetFlightDetail(context.Context, application.FlightDetailInput) (application.FlightDetailResponse, error) {
	if a.detailResponse.Flight.FlightID == "" {
		return application.FlightDetailResponse{}, application.ErrNotFound
	}
	return a.detailResponse, nil
}

func (a *stubApplication) GetFlightMapData(context.Context, application.FlightMapDataInput) (application.FlightMapDataResponse, error) {
	if a.mapResponse.FlightID == "" {
		return application.FlightMapDataResponse{}, application.ErrNotFound
	}
	return a.mapResponse, nil
}

func (a *stubApplication) RequestFlightRefresh(_ context.Context, input application.FlightRefreshInput) (application.FlightRefreshResponse, error) {
	a.refreshInput = input
	if a.refreshResponse.FlightID == "" {
		return application.FlightRefreshResponse{}, application.ErrNotFound
	}
	return a.refreshResponse, nil
}

func (a *stubApplication) GetUsageStatus(context.Context, application.UsageStatusInput) (application.UsageStatus, error) {
	if a.usageResponse.Budget.Environment == "" {
		return application.UsageStatus{}, application.ErrNotFound
	}
	return a.usageResponse, nil
}

func request(method string, path string, body string) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{
		RawPath: path,
		Body:    body,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "req-1",
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: method,
			},
		},
	}
}

func testFlight(flightID string) domain.Flight {
	return domain.Flight{
		FlightID:     domain.FlightID(flightID),
		FlightIDType: domain.FlightIDTypeInternal,
		Ident:        "ANA110",
		Origin:       domain.Airport{Code: "RJTT"},
		Destination:  domain.Airport{Code: "KJFK"},
		Status:       "En Route",
		UpdatedAt:    "2026-04-29T00:00:00Z",
	}
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
