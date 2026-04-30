package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"

	"airpath/services/internal/application"
	"airpath/services/internal/domain"

	"github.com/aws/aws-lambda-go/events"
)

func TestBackendRoutesAlignWithSharedTypeScriptTemplates(t *testing.T) {
	source, err := os.ReadFile("../../../packages/shared-types/src/api-routes.ts")
	if err != nil {
		t.Fatalf("read shared API routes: %v", err)
	}
	sharedRoutes := string(source)
	for _, route := range []string{
		routeFlightSearch,
		"/v1/flights/{flightId}",
		"/v1/flights/{flightId}/" + flightMapDataSuffix,
		"/v1/flights/{flightId}/" + flightPositionsSuffix,
		"/v1/flights/{flightId}/" + flightRefreshSuffix,
		routeUsageStatus,
	} {
		if !strings.Contains(sharedRoutes, `"`+route+`"`) {
			t.Fatalf("shared TypeScript API route templates do not contain %q", route)
		}
	}
}

func TestAdapterMapsAPIGatewaySearchEventToApplicationResponse(t *testing.T) {
	app := &stubApplication{
		searchResponse: application.FlightSearchResponse{
			Items: []application.FlightSummaryItem{{FlightID: "iflg_1", Ident: "ANA110", Origin: "RJTT", Destination: "KJFK"}},
			Cache: application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
		},
	}
	adapter := NewAdapter(app)

	response, err := adapter.Handle(context.Background(), events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "req-1",
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
			},
		},
		RawPath: "/v1/flights/search",
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
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "req-budget",
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
			},
		},
		RawPath: "/v1/flights/search",
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

func TestAdapterRejectsWhitespaceSearchIdentWithoutCallingApplication(t *testing.T) {
	app := &stubApplication{
		searchResponse: application.FlightSearchResponse{
			Items: []application.FlightSummaryItem{{FlightID: "iflg_1"}},
		},
	}
	adapter := NewAdapter(app)

	response, err := adapter.Handle(context.Background(), events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "req-blank",
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
			},
		},
		RawPath: routeFlightSearch,
		QueryStringParameters: map[string]string{
			"ident": "   ",
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", response.StatusCode, response.Body)
	}
	if errorCode(t, response.Body) != string(application.ApiErrorValidationFailed) {
		t.Fatalf("error code body = %s, want validation_failed", response.Body)
	}
	if app.searchInput.Ident != "" {
		t.Fatalf("SearchFlights was called with input %#v", app.searchInput)
	}
}

func TestAdapterRejectsShortSearchIdentWithoutCallingApplication(t *testing.T) {
	app := &stubApplication{
		searchResponse: application.FlightSearchResponse{
			Items: []application.FlightSummaryItem{{FlightID: "iflg_1"}},
		},
	}
	adapter := NewAdapter(app)

	response, err := adapter.Handle(context.Background(), events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "req-short",
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
			},
		},
		RawPath: routeFlightSearch,
		QueryStringParameters: map[string]string{
			"ident": " A ",
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", response.StatusCode, response.Body)
	}
	if app.searchInput.Ident != "" {
		t.Fatalf("SearchFlights was called with input %#v", app.searchInput)
	}
}

func TestAdapterRejectsSearchDateWithoutCallingApplication(t *testing.T) {
	app := &stubApplication{
		searchResponse: application.FlightSearchResponse{
			Items: []application.FlightSummaryItem{{FlightID: "iflg_1"}},
		},
	}
	adapter := NewAdapter(app)

	response, err := adapter.Handle(context.Background(), events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "req-date",
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
			},
		},
		RawPath: routeFlightSearch,
		QueryStringParameters: map[string]string{
			"ident": "ANA110",
			"date":  "2026-04-29",
		},
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", response.StatusCode, response.Body)
	}
	if app.searchInput.Ident != "" {
		t.Fatalf("SearchFlights was called with input %#v", app.searchInput)
	}
}

func TestAdapterMapsFlightDetailMapRefreshAndUsageRoutes(t *testing.T) {
	app := &stubApplication{
		detailResponse: application.FlightDetailResponse{
			Flight: testFlightDetail("iflg_1"),
			Cache:  application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
		},
		mapResponse: application.FlightMapDataResponse{
			FlightID: "iflg_1",
			Planned:  application.MapLayer{Source: application.MapSourceFlightAwareRoute, Available: false},
			Actual:   application.MapLayer{Source: application.MapSourceFlightAwareTrack, Available: false},
			Current:  application.MapLayer{Source: application.MapSourceFlightAwarePosition, Available: false},
			Cache:    application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
		},
		positionsResponse: application.FlightPositionsResponse{
			FlightID: "iflg_1",
			Items: []application.Position{{
				Latitude:  45.1,
				Longitude: 160.4,
				Timestamp: "2026-04-29T00:05:00Z",
				Source:    domain.PositionSourceFlightAwarePosition,
			}},
			Cache: application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
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

	positions, err := adapter.Handle(context.Background(), request(http.MethodGet, "/v1/flights/iflg_1/positions", ""))
	if err != nil {
		t.Fatalf("positions Handle() error = %v", err)
	}
	if positions.StatusCode != 200 || !jsonContains(positions.Body, "items") || app.positionsInput.Limit != 200 {
		t.Fatalf("positions response = %#v input=%#v", positions, app.positionsInput)
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

func TestAdapterDecodesFlightIDPathSegmentBeforeCallingApplication(t *testing.T) {
	const encodedFlightID = "iflg_1%2Fsegment"
	const decodedFlightID = "iflg_1/segment"

	for _, testCase := range []struct {
		name   string
		method string
		path   string
		body   string
		assert func(*testing.T, *stubApplication)
	}{
		{
			name:   "detail",
			method: http.MethodGet,
			path:   "/v1/flights/" + encodedFlightID,
			assert: func(t *testing.T, app *stubApplication) {
				t.Helper()
				if app.detailInput.FlightID != decodedFlightID {
					t.Fatalf("FlightID = %q, want %q", app.detailInput.FlightID, decodedFlightID)
				}
			},
		},
		{
			name:   "map",
			method: http.MethodGet,
			path:   "/v1/flights/" + encodedFlightID + "/map-data",
			assert: func(t *testing.T, app *stubApplication) {
				t.Helper()
				if app.mapInput.FlightID != decodedFlightID {
					t.Fatalf("FlightID = %q, want %q", app.mapInput.FlightID, decodedFlightID)
				}
			},
		},
		{
			name:   "positions",
			method: http.MethodGet,
			path:   "/v1/flights/" + encodedFlightID + "/positions",
			assert: func(t *testing.T, app *stubApplication) {
				t.Helper()
				if app.positionsInput.FlightID != decodedFlightID {
					t.Fatalf("FlightID = %q, want %q", app.positionsInput.FlightID, decodedFlightID)
				}
			},
		},
		{
			name:   "refresh",
			method: http.MethodPost,
			path:   "/v1/flights/" + encodedFlightID + "/refresh",
			body:   `{"taskTypes":["route"],"clientReason":"user_manual_refresh"}`,
			assert: func(t *testing.T, app *stubApplication) {
				t.Helper()
				if app.refreshInput.FlightID != decodedFlightID {
					t.Fatalf("FlightID = %q, want %q", app.refreshInput.FlightID, decodedFlightID)
				}
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			app := &stubApplication{
				detailResponse: application.FlightDetailResponse{
					Flight: testFlightDetail(decodedFlightID),
					Cache:  application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
				},
				mapResponse: application.FlightMapDataResponse{
					FlightID: decodedFlightID,
					Cache:    application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
				},
				positionsResponse: application.FlightPositionsResponse{
					FlightID: decodedFlightID,
					Cache:    application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
				},
				refreshResponse: application.FlightRefreshResponse{
					FlightID: decodedFlightID,
					Cache:    application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
				},
			}
			adapter := NewAdapter(app)

			response, err := adapter.Handle(context.Background(), request(testCase.method, testCase.path, testCase.body))
			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			if response.StatusCode < 200 || response.StatusCode >= 300 {
				t.Fatalf("status = %d, want success body=%s", response.StatusCode, response.Body)
			}
			testCase.assert(t, app)
		})
	}
}

func TestAdapterValidatesPositionsQueryBeforeCallingApplication(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		query map[string]string
	}{
		{
			name: "malformed since",
			query: map[string]string{
				"since": "not-a-date",
			},
		},
		{
			name: "blank since",
			query: map[string]string{
				"since": "   ",
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			app := &stubApplication{
				positionsResponse: application.FlightPositionsResponse{
					FlightID: "iflg_1",
					Cache:    application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
				},
			}
			adapter := NewAdapter(app)
			event := request(http.MethodGet, "/v1/flights/iflg_1/positions", "")
			event.QueryStringParameters = testCase.query

			response, err := adapter.Handle(context.Background(), event)
			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 body=%s", response.StatusCode, response.Body)
			}
			if app.positionsInput.FlightID != "" {
				t.Fatalf("GetFlightPositions was called with input %#v", app.positionsInput)
			}
		})
	}
}

func TestAdapterParsesValidPositionsQuery(t *testing.T) {
	app := &stubApplication{
		positionsResponse: application.FlightPositionsResponse{
			FlightID: "iflg_1",
			Cache:    application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
		},
	}
	adapter := NewAdapter(app)
	event := request(http.MethodGet, "/v1/flights/iflg_1/positions", "")
	event.QueryStringParameters = map[string]string{
		"since": " 2026-04-29T00:05:00Z ",
		"limit": "501",
	}

	response, err := adapter.Handle(context.Background(), event)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", response.StatusCode, response.Body)
	}
	if app.positionsInput.Since == nil || *app.positionsInput.Since != "2026-04-29T00:05:00Z" {
		t.Fatalf("Since = %#v, want trimmed RFC3339 value", app.positionsInput.Since)
	}
	if app.positionsInput.Limit != 500 {
		t.Fatalf("Limit = %d, want capped 500", app.positionsInput.Limit)
	}
}

func TestAdapterNormalizesPositionsSinceToUTCBeforeCallingApplication(t *testing.T) {
	app := &stubApplication{
		positionsResponse: application.FlightPositionsResponse{
			FlightID: "iflg_1",
			Cache:    application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
		},
	}
	adapter := NewAdapter(app)
	event := request(http.MethodGet, "/v1/flights/iflg_1/positions", "")
	event.QueryStringParameters = map[string]string{
		"since": "2026-04-29T09:05:00.987654321+09:00",
	}

	response, err := adapter.Handle(context.Background(), event)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", response.StatusCode, response.Body)
	}
	if app.positionsInput.Since == nil || *app.positionsInput.Since != "2026-04-29T00:05:00Z" {
		t.Fatalf("Since = %#v, want canonical UTC value", app.positionsInput.Since)
	}
}

func TestAdapterRejectsUnsupportedMapDataQueryWithoutCallingApplication(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		query map[string]string
	}{
		{
			name: "include",
			query: map[string]string{
				"include": "current",
			},
		},
		{
			name: "simplify",
			query: map[string]string{
				"simplify": "false",
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			app := &stubApplication{
				mapResponse: application.FlightMapDataResponse{
					FlightID: "iflg_1",
				},
			}
			adapter := NewAdapter(app)
			event := request(http.MethodGet, "/v1/flights/iflg_1/map-data", "")
			event.QueryStringParameters = testCase.query

			response, err := adapter.Handle(context.Background(), event)
			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 body=%s", response.StatusCode, response.Body)
			}
			if app.mapCalled {
				t.Fatal("GetFlightMapData was called")
			}
		})
	}
}

func TestAdapterRejectsUnsupportedPositionsQualityWithoutCallingApplication(t *testing.T) {
	app := &stubApplication{
		positionsResponse: application.FlightPositionsResponse{
			FlightID: "iflg_1",
			Cache:    application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
		},
	}
	adapter := NewAdapter(app)
	event := request(http.MethodGet, "/v1/flights/iflg_1/positions", "")
	event.QueryStringParameters = map[string]string{
		"quality": "raw",
	}

	response, err := adapter.Handle(context.Background(), event)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", response.StatusCode, response.Body)
	}
	if app.positionsInput.FlightID != "" {
		t.Fatalf("GetFlightPositions was called with input %#v", app.positionsInput)
	}
}

func TestAdapterRejectsEmptyPositionsQueryValuesWithoutCallingApplication(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		query map[string]string
	}{
		{
			name: "empty since",
			query: map[string]string{
				"since": "",
			},
		},
		{
			name: "empty limit",
			query: map[string]string{
				"limit": "",
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			app := &stubApplication{
				positionsResponse: application.FlightPositionsResponse{
					FlightID: "iflg_1",
					Cache:    application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
				},
			}
			adapter := NewAdapter(app)
			event := request(http.MethodGet, "/v1/flights/iflg_1/positions", "")
			event.QueryStringParameters = testCase.query

			response, err := adapter.Handle(context.Background(), event)
			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 body=%s", response.StatusCode, response.Body)
			}
			if app.positionsInput.FlightID != "" {
				t.Fatalf("GetFlightPositions was called with input %#v", app.positionsInput)
			}
		})
	}
}

func TestAdapterRejectsInvalidRefreshRequestsWithoutCallingApplication(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "unknown task type",
			body: `{"taskTypes":["unknown"],"clientReason":"user_manual_refresh"}`,
		},
		{
			name: "summary task type",
			body: `{"taskTypes":["summary"],"clientReason":"user_manual_refresh"}`,
		},
		{
			name: "duplicate task type",
			body: `{"taskTypes":["route","route"],"clientReason":"user_manual_refresh"}`,
		},
		{
			name: "unknown reason",
			body: `{"taskTypes":["route"],"clientReason":"unknown_reason"}`,
		},
		{
			name: "internal low-frequency reason",
			body: `{"taskTypes":["route"],"clientReason":"low_frequency_poll"}`,
		},
		{
			name: "internal usage reconciliation reason",
			body: `{"taskTypes":["route"],"clientReason":"usage_reconciliation"}`,
		},
		{
			name: "unknown field",
			body: `{"taskTypes":["route"],"clientReason":"user_manual_refresh","admin":true}`,
		},
		{
			name: "trailing json value",
			body: `{"taskTypes":["route"],"clientReason":"user_manual_refresh"} {}`,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			app := &stubApplication{
				refreshResponse: application.FlightRefreshResponse{
					FlightID: "iflg_1",
				},
			}
			adapter := NewAdapter(app)

			response, err := adapter.Handle(context.Background(), request(http.MethodPost, "/v1/flights/iflg_1/refresh", testCase.body))
			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 body=%s", response.StatusCode, response.Body)
			}
			if app.refreshInput.FlightID != "" {
				t.Fatalf("application was called with refresh input %#v", app.refreshInput)
			}
		})
	}
}

func TestAdapterRejectsEmptyFlightIDResourceWithoutCallingApplication(t *testing.T) {
	app := &stubApplication{
		detailResponse: application.FlightDetailResponse{
			Flight: testFlightDetail("iflg_1"),
			Cache:  application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
		},
		positionsResponse: application.FlightPositionsResponse{
			FlightID: "iflg_1",
			Cache:    application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
		},
		refreshResponse: application.FlightRefreshResponse{
			FlightID: "iflg_1",
			Cache:    application.CacheMetadata{Freshness: application.CacheFreshnessFresh, Source: application.CacheSourceCache},
		},
	}
	adapter := NewAdapter(app)

	for _, testCase := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "detail", method: http.MethodGet, path: "/v1/flights/"},
		{name: "positions", method: http.MethodGet, path: "/v1/flights//positions"},
		{name: "refresh", method: http.MethodPost, path: "/v1/flights//refresh", body: `{"taskTypes":["route"],"clientReason":"user_manual_refresh"}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			response, err := adapter.Handle(context.Background(), request(testCase.method, testCase.path, testCase.body))
			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			if response.StatusCode != http.StatusNotFound {
				t.Fatalf("status = %d, want 404 body=%s", response.StatusCode, response.Body)
			}
			if app.detailCalled || app.positionsInput.FlightID != "" || app.refreshInput.FlightID != "" {
				t.Fatalf("application was called: detail=%v positions=%#v refresh=%#v", app.detailCalled, app.positionsInput, app.refreshInput)
			}
		})
	}
}

func TestAdapterAddsCORSHeadersAndHandlesPreflight(t *testing.T) {
	app := &stubApplication{
		usageResponse: application.UsageStatus{
			Budget:          application.UsageBudgetStatus{Environment: "dev"},
			FetchingEnabled: true,
		},
	}
	adapter := NewAdapter(app)

	success, err := adapter.Handle(context.Background(), request(http.MethodGet, routeUsageStatus, ""))
	if err != nil {
		t.Fatalf("Handle(success) error = %v", err)
	}
	assertCORSHeaders(t, success)

	errorResponse, err := adapter.Handle(context.Background(), request(http.MethodGet, "/v1/missing", ""))
	if err != nil {
		t.Fatalf("Handle(error) error = %v", err)
	}
	assertCORSHeaders(t, errorResponse)

	app.usageCalled = false
	preflight, err := adapter.Handle(context.Background(), request(http.MethodOptions, "/v1/flights/iflg_1/refresh", ""))
	if err != nil {
		t.Fatalf("Handle(preflight) error = %v", err)
	}
	if preflight.StatusCode != http.StatusNoContent || preflight.Body != "" {
		t.Fatalf("preflight response = %#v, want 204 with empty body", preflight)
	}
	assertCORSHeaders(t, preflight)
	if app.usageCalled {
		t.Fatal("application was called for preflight")
	}
}

func TestAdapterRejectsUnsupportedMethodsForTopLevelReadRoutes(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "search post", method: http.MethodPost, path: "/v1/flights/search"},
		{name: "search missing method", method: "", path: "/v1/flights/search"},
		{name: "usage delete", method: http.MethodDelete, path: "/v1/usage/status"},
		{name: "usage missing method", method: "", path: "/v1/usage/status"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			app := &stubApplication{
				searchResponse: application.FlightSearchResponse{
					Items: []application.FlightSummaryItem{{FlightID: "iflg_1"}},
				},
				usageResponse: application.UsageStatus{
					Budget:          application.UsageBudgetStatus{Environment: "dev"},
					FetchingEnabled: true,
				},
			}
			adapter := NewAdapter(app)

			response, err := adapter.Handle(context.Background(), request(testCase.method, testCase.path, ""))
			if err != nil {
				t.Fatalf("Handle() error = %v", err)
			}
			if response.StatusCode != http.StatusNotFound {
				t.Fatalf("status = %d, want 404 body=%s", response.StatusCode, response.Body)
			}
			if app.searchInput.Ident != "" {
				t.Fatalf("SearchFlights was called with input %#v", app.searchInput)
			}
			if app.usageCalled {
				t.Fatal("GetUsageStatus was called for unsupported method")
			}
		})
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
	searchResponse    application.FlightSearchResponse
	searchInput       application.SearchFlightsInput
	searchErr         error
	detailResponse    application.FlightDetailResponse
	detailInput       application.FlightDetailInput
	detailCalled      bool
	mapResponse       application.FlightMapDataResponse
	mapInput          application.FlightMapDataInput
	mapCalled         bool
	positionsResponse application.FlightPositionsResponse
	positionsInput    application.FlightPositionsInput
	refreshResponse   application.FlightRefreshResponse
	refreshInput      application.FlightRefreshInput
	usageResponse     application.UsageStatus
	usageCalled       bool
}

func (a *stubApplication) SearchFlights(_ context.Context, input application.SearchFlightsInput) (application.FlightSearchResponse, error) {
	a.searchInput = input
	if input.Ident == "" {
		return application.FlightSearchResponse{}, errors.New("missing ident")
	}
	return a.searchResponse, a.searchErr
}

func (a *stubApplication) GetFlightDetail(_ context.Context, input application.FlightDetailInput) (application.FlightDetailResponse, error) {
	a.detailInput = input
	a.detailCalled = true
	if a.detailResponse.Flight.FlightID == "" {
		return application.FlightDetailResponse{}, application.ErrNotFound
	}
	return a.detailResponse, nil
}

func (a *stubApplication) GetFlightMapData(_ context.Context, input application.FlightMapDataInput) (application.FlightMapDataResponse, error) {
	a.mapInput = input
	a.mapCalled = true
	if a.mapResponse.FlightID == "" {
		return application.FlightMapDataResponse{}, application.ErrNotFound
	}
	return a.mapResponse, nil
}

func (a *stubApplication) GetFlightPositions(_ context.Context, input application.FlightPositionsInput) (application.FlightPositionsResponse, error) {
	a.positionsInput = input
	if a.positionsResponse.FlightID == "" {
		return application.FlightPositionsResponse{}, application.ErrNotFound
	}
	return a.positionsResponse, nil
}

func (a *stubApplication) RequestFlightRefresh(_ context.Context, input application.FlightRefreshInput) (application.FlightRefreshResponse, error) {
	a.refreshInput = input
	if a.refreshResponse.FlightID == "" {
		return application.FlightRefreshResponse{}, application.ErrNotFound
	}
	return a.refreshResponse, nil
}

func (a *stubApplication) GetUsageStatus(context.Context, application.UsageStatusInput) (application.UsageStatus, error) {
	a.usageCalled = true
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

func testFlightDetail(flightID string) application.FlightDetail {
	return application.FlightDetail{
		FlightID:     domain.FlightID(flightID),
		FlightIDType: domain.FlightIDTypeInternal,
		Ident:        "ANA110",
		Origin:       domain.Airport{Code: "RJTT"},
		Destination:  domain.Airport{Code: "KJFK"},
		Status:       "En Route",
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

func assertCORSHeaders(t *testing.T, response events.APIGatewayV2HTTPResponse) {
	t.Helper()
	if response.Headers["access-control-allow-origin"] != "*" {
		t.Fatalf("access-control-allow-origin = %q, want *", response.Headers["access-control-allow-origin"])
	}
	if !strings.Contains(response.Headers["access-control-allow-methods"], http.MethodOptions) {
		t.Fatalf("access-control-allow-methods = %q, want OPTIONS included", response.Headers["access-control-allow-methods"])
	}
	if !strings.Contains(response.Headers["access-control-allow-headers"], "content-type") {
		t.Fatalf("access-control-allow-headers = %q, want content-type included", response.Headers["access-control-allow-headers"])
	}
}

func errorCode(t *testing.T, body string) string {
	t.Helper()
	var parsed struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	return parsed.Error.Code
}
