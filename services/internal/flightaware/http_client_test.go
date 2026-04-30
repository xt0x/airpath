package flightaware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPClientSearchFlightsSendsAPIKeyAndMaxPagesOne(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/flights/ANA110" {
			t.Fatalf("path = %q, want /flights/ANA110", r.URL.Path)
		}
		if r.URL.Query().Get("ident_type") != "designator" {
			t.Fatalf("ident_type = %q, want designator", r.URL.Query().Get("ident_type"))
		}
		if r.URL.Query().Get("max_pages") != "1" {
			t.Fatalf("max_pages = %q, want 1", r.URL.Query().Get("max_pages"))
		}
		if r.Header.Get("x-apikey") != "test-key" {
			t.Fatalf("x-apikey header = %q, want test-key", r.Header.Get("x-apikey"))
		}
		writeJSON(t, w, map[string]any{
			"flights": []map[string]any{
				{
					"fa_flight_id": "fa_1",
					"ident":        "ANA110",
					"origin":       map[string]any{"code": "RJTT"},
					"destination":  map[string]any{"code": "KJFK"},
				},
			},
		})
	}))
	defer server.Close()

	client, err := NewHTTPClient(HTTPClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	response, err := client.SearchFlights(context.Background(), SearchFlightsRequest{Ident: "ANA110"})
	if err != nil {
		t.Fatalf("SearchFlights() error = %v", err)
	}
	if len(response.Flights) != 1 || response.Flights[0].Origin != "RJTT" || response.Flights[0].Destination != "KJFK" {
		t.Fatalf("response = %#v", response)
	}
}

func TestHTTPClientGetFlightSummaryDecodesFlightsWrapper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/flights/fa_1" {
			t.Fatalf("path = %q, want /flights/fa_1", r.URL.Path)
		}
		if r.URL.Query().Get("ident_type") != "fa_flight_id" {
			t.Fatalf("ident_type = %q, want fa_flight_id", r.URL.Query().Get("ident_type"))
		}
		writeJSON(t, w, map[string]any{
			"flights": []map[string]any{
				{
					"fa_flight_id": "fa_1",
					"ident":        "ANA110",
					"origin":       map[string]any{"code": "RJTT"},
					"destination":  map[string]any{"code": "KJFK"},
				},
			},
		})
	}))
	defer server.Close()

	client, err := NewHTTPClient(HTTPClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	summary, err := client.GetFlightSummary(context.Background(), FlightSummaryRequest{FAFlightID: "fa_1"})
	if err != nil {
		t.Fatalf("GetFlightSummary() error = %v", err)
	}
	if summary.FAFlightID == nil || *summary.FAFlightID != "fa_1" || summary.Ident != "ANA110" || summary.Origin != "RJTT" || summary.Destination != "KJFK" {
		t.Fatalf("summary = %#v, want first wrapped flight normalized", summary)
	}
}

func TestHTTPClientGetFlightSummaryRejectsEmptyFlightsWrapper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"flights": []map[string]any{}})
	}))
	defer server.Close()

	client, err := NewHTTPClient(HTTPClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	_, err = client.GetFlightSummary(context.Background(), FlightSummaryRequest{FAFlightID: "missing"})
	var clientErr *ClientError
	if !errors.As(err, &clientErr) || clientErr.Code != "upstream_empty_response" {
		t.Fatalf("GetFlightSummary(empty wrapper) error = %v, want upstream_empty_response ClientError", err)
	}
}

func TestHTTPClientNormalizesRouteTrackPositionAndUsageResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/flights/fa_1/route":
			writeJSON(t, w, map[string]any{
				"route": "RJTT FIXA KJFK",
				"fixes": []map[string]any{
					{"name": "RJTT", "latitude": 35.55, "longitude": 139.78},
					{"name": "KJFK", "latitude": 40.64, "longitude": -73.78},
				},
			})
		case "/flights/fa_1/track":
			writeJSON(t, w, map[string]any{
				"positions": []map[string]any{
					{"latitude": 35.55, "longitude": 139.78, "altitude": 0, "altitude_change": "C", "groundspeed": 0, "heading": 72, "timestamp": "2026-06-20T08:30:00Z", "update_type": "A"},
					{"latitude": 40.64, "longitude": -73.78, "altitude": 370, "altitude_change": "-", "groundspeed": 488, "heading": 360, "timestamp": "2026-06-20T17:30:00Z", "update_type": "P"},
				},
			})
		case "/flights/fa_1/position":
			writeJSON(t, w, map[string]any{
				"fa_flight_id": "fa_1",
				"last_position": map[string]any{
					"latitude":        45.12,
					"longitude":       160.45,
					"altitude":        370,
					"altitude_change": "D",
					"groundspeed":     488,
					"heading":         275,
					"timestamp":       "2026-06-20T09:30:00Z",
					"update_type":     "Z",
				},
			})
		case "/account/usage":
			writeJSON(t, w, map[string]any{
				"total_cost":          1.50,
				"total_discount_cost": 1.25,
				"total_pages":         12,
			})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewHTTPClient(HTTPClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	route, err := client.GetFlightRoute(context.Background(), FlightRouteRequest{FAFlightID: "fa_1"})
	if err != nil {
		t.Fatalf("GetFlightRoute() error = %v", err)
	}
	if route.RouteText == nil || *route.RouteText != "RJTT FIXA KJFK" || len(route.Fixes) != 2 {
		t.Fatalf("route = %#v", route)
	}

	track, err := client.GetFlightTrack(context.Background(), FlightTrackRequest{FAFlightID: "fa_1"})
	if err != nil {
		t.Fatalf("GetFlightTrack() error = %v", err)
	}
	if len(track.Positions) != 2 || track.Positions[1].Timestamp != "2026-06-20T17:30:00Z" {
		t.Fatalf("track = %#v", track)
	}
	if track.Positions[0].AltitudeChange == nil || *track.Positions[0].AltitudeChange != "climbing" ||
		track.Positions[0].UpdateType == nil || *track.Positions[0].UpdateType != "actual" {
		t.Fatalf("first track code metrics = %#v, want climbing and actual", track.Positions[0])
	}
	if track.Positions[1].AltitudeHundredsFeet == nil || *track.Positions[1].AltitudeHundredsFeet != 370 ||
		track.Positions[1].AltitudeChange == nil || *track.Positions[1].AltitudeChange != "level" ||
		track.Positions[1].GroundspeedKnots == nil || *track.Positions[1].GroundspeedKnots != 488 ||
		track.Positions[1].HeadingDegrees == nil || *track.Positions[1].HeadingDegrees != 360 ||
		track.Positions[1].UpdateType == nil || *track.Positions[1].UpdateType != "predicted" {
		t.Fatalf("track metrics = %#v, want FlightAware position metrics preserved", track.Positions[1])
	}

	position, err := client.GetFlightPosition(context.Background(), FlightPositionRequest{FAFlightID: "fa_1"})
	if err != nil {
		t.Fatalf("GetFlightPosition() error = %v", err)
	}
	if position.Latitude == nil || *position.Latitude != 45.12 || position.Timestamp == "" {
		t.Fatalf("position = %#v", position)
	}
	if position.AltitudeHundredsFeet == nil || *position.AltitudeHundredsFeet != 370 ||
		position.AltitudeChange == nil || *position.AltitudeChange != "descending" ||
		position.GroundspeedKnots == nil || *position.GroundspeedKnots != 488 ||
		position.HeadingDegrees == nil || *position.HeadingDegrees != 275 ||
		position.UpdateType == nil || *position.UpdateType != "actual" {
		t.Fatalf("position metrics = %#v, want FlightAware position metrics preserved", position)
	}

	usage, err := client.GetAccountUsage(context.Background())
	if err != nil {
		t.Fatalf("GetAccountUsage() error = %v", err)
	}
	if usage.Currency != "USD" || usage.MonthToDate.EstimatedCostUSD != 1.25 || usage.MonthToDate.ResultSets != 12 {
		t.Fatalf("usage = %#v", usage)
	}
}

func TestHTTPClientGetAccountUsagePreservesDiscountedZeroCost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/usage" {
			t.Fatalf("path = %q, want /account/usage", r.URL.Path)
		}
		writeJSON(t, w, map[string]any{
			"total_cost":          10.00,
			"total_discount_cost": 0,
			"total_pages":         4,
		})
	}))
	defer server.Close()

	client, err := NewHTTPClient(HTTPClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	usage, err := client.GetAccountUsage(context.Background())
	if err != nil {
		t.Fatalf("GetAccountUsage() error = %v", err)
	}
	if usage.MonthToDate.EstimatedCostUSD != 0 || usage.MonthToDate.ResultSets != 4 {
		t.Fatalf("usage = %#v, want discounted zero cost preserved", usage)
	}
}

func TestHTTPClientGetAccountUsageFallsBackToTotalCostWhenDiscountCostMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/usage" {
			t.Fatalf("path = %q, want /account/usage", r.URL.Path)
		}
		writeJSON(t, w, map[string]any{
			"total_cost":  3.75,
			"total_pages": 2,
		})
	}))
	defer server.Close()

	client, err := NewHTTPClient(HTTPClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	usage, err := client.GetAccountUsage(context.Background())
	if err != nil {
		t.Fatalf("GetAccountUsage() error = %v", err)
	}
	if usage.MonthToDate.EstimatedCostUSD != 3.75 || usage.MonthToDate.ResultSets != 2 {
		t.Fatalf("usage = %#v, want total_cost fallback", usage)
	}
}

func TestHTTPClientGetFlightPositionRejectsMissingLastPosition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{"fa_flight_id": "fa_1"})
	}))
	defer server.Close()

	client, err := NewHTTPClient(HTTPClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	_, err = client.GetFlightPosition(context.Background(), FlightPositionRequest{FAFlightID: "fa_1"})
	var clientErr *ClientError
	if !errors.As(err, &clientErr) || clientErr.Code != "upstream_empty_response" {
		t.Fatalf("GetFlightPosition(missing last_position) error = %v, want upstream_empty_response ClientError", err)
	}
}

func TestHTTPClientMaps429ToTypedRateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"title":"rate limited"}`, http.StatusTooManyRequests)
	}))
	defer server.Close()

	client, err := NewHTTPClient(HTTPClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	_, err = client.GetFlightTrack(context.Background(), FlightTrackRequest{FAFlightID: "fa_1"})
	if !errors.Is(err, ErrFlightAwareRateLimited) {
		t.Fatalf("GetFlightTrack() error = %v, want ErrFlightAwareRateLimited", err)
	}
}

func TestHTTPClientRejectsMissingAPIKey(t *testing.T) {
	_, err := NewHTTPClient(HTTPClientConfig{APIKey: ""})
	if err == nil || !strings.Contains(err.Error(), "api key") {
		t.Fatalf("NewHTTPClient(empty key) error = %v, want API key validation", err)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
