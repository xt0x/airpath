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
					{"latitude": 35.55, "longitude": 139.78, "timestamp": "2026-06-20T08:30:00Z"},
					{"latitude": 40.64, "longitude": -73.78, "timestamp": "2026-06-20T17:30:00Z"},
				},
			})
		case "/flights/fa_1/position":
			writeJSON(t, w, map[string]any{
				"fa_flight_id": "fa_1",
				"latitude":     45.12,
				"longitude":    160.45,
				"timestamp":    "2026-06-20T09:30:00Z",
			})
		case "/account/usage":
			writeJSON(t, w, map[string]any{
				"currency": "USD",
				"month_to_date": map[string]any{
					"estimated_cost_usd": 1.25,
					"result_sets":        12,
				},
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

	position, err := client.GetFlightPosition(context.Background(), FlightPositionRequest{FAFlightID: "fa_1"})
	if err != nil {
		t.Fatalf("GetFlightPosition() error = %v", err)
	}
	if position.Latitude == nil || *position.Latitude != 45.12 || position.Timestamp == "" {
		t.Fatalf("position = %#v", position)
	}

	usage, err := client.GetAccountUsage(context.Background())
	if err != nil {
		t.Fatalf("GetAccountUsage() error = %v", err)
	}
	if usage.Currency != "USD" || usage.MonthToDate.EstimatedCostUSD != 1.25 || usage.MonthToDate.ResultSets != 12 {
		t.Fatalf("usage = %#v", usage)
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
