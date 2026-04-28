package flightaware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://aeroapi.flightaware.com/aeroapi"

type HTTPClientConfig struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

type HTTPClient struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
}

func NewHTTPClient(config HTTPClientConfig) (*HTTPClient, error) {
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, errors.New("flightaware api key is required")
	}
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, err
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &HTTPClient{baseURL: parsed, apiKey: config.APIKey, httpClient: httpClient}, nil
}

func (c *HTTPClient) SearchFlights(ctx context.Context, request SearchFlightsRequest) (SearchFlightsResponse, error) {
	maxPages, err := normalizeMaxPages(EndpointSearch, request.MaxPages)
	if err != nil {
		return SearchFlightsResponse{}, err
	}
	var raw struct {
		Flights []rawFlightSummary `json:"flights"`
	}
	if err := c.get(ctx, EndpointSearch, []string{"flights", request.Ident}, map[string]string{"max_pages": fmt.Sprint(maxPages)}, &raw); err != nil {
		return SearchFlightsResponse{}, err
	}
	flights := make([]FlightSummary, 0, len(raw.Flights))
	for _, flight := range raw.Flights {
		flights = append(flights, flight.normalize())
	}
	return SearchFlightsResponse{Flights: flights}, nil
}

func (c *HTTPClient) GetFlightSummary(ctx context.Context, request FlightSummaryRequest) (FlightSummary, error) {
	var raw rawFlightSummary
	if err := c.get(ctx, EndpointSummary, []string{"flights", request.FAFlightID}, nil, &raw); err != nil {
		return FlightSummary{}, err
	}
	return raw.normalize(), nil
}

func (c *HTTPClient) GetFlightRoute(ctx context.Context, request FlightRouteRequest) (RouteResponse, error) {
	var raw struct {
		RouteText *string    `json:"route"`
		Fixes     []RouteFix `json:"fixes"`
	}
	if err := c.get(ctx, EndpointRoute, []string{"flights", request.FAFlightID, "route"}, nil, &raw); err != nil {
		return RouteResponse{}, err
	}
	return RouteResponse{RouteText: raw.RouteText, Fixes: raw.Fixes}, nil
}

func (c *HTTPClient) GetFlightPosition(ctx context.Context, request FlightPositionRequest) (PositionResponse, error) {
	var raw struct {
		FAFlightID string   `json:"fa_flight_id"`
		Latitude   *float64 `json:"latitude"`
		Longitude  *float64 `json:"longitude"`
		Timestamp  string   `json:"timestamp"`
	}
	if err := c.get(ctx, EndpointPosition, []string{"flights", request.FAFlightID, "position"}, nil, &raw); err != nil {
		return PositionResponse{}, err
	}
	return PositionResponse{
		FAFlightID: raw.FAFlightID,
		Latitude:   raw.Latitude,
		Longitude:  raw.Longitude,
		Timestamp:  raw.Timestamp,
	}, nil
}

func (c *HTTPClient) GetFlightTrack(ctx context.Context, request FlightTrackRequest) (TrackResponse, error) {
	var raw struct {
		Positions []TrackPoint `json:"positions"`
	}
	if err := c.get(ctx, EndpointTrack, []string{"flights", request.FAFlightID, "track"}, nil, &raw); err != nil {
		return TrackResponse{}, err
	}
	return TrackResponse{Positions: raw.Positions}, nil
}

func (c *HTTPClient) GetSchedules(ctx context.Context, request SchedulesRequest) (SchedulesResponse, error) {
	maxPages, err := normalizeMaxPages(EndpointSchedule, request.MaxPages)
	if err != nil {
		return SchedulesResponse{}, err
	}
	var raw struct {
		Scheduled []rawFlightSummary `json:"scheduled"`
	}
	if err := c.get(ctx, EndpointSchedule, []string{"schedules", request.StartDate, request.EndDate}, map[string]string{"max_pages": fmt.Sprint(maxPages)}, &raw); err != nil {
		return SchedulesResponse{}, err
	}
	scheduled := make([]FlightSummary, 0, len(raw.Scheduled))
	for _, flight := range raw.Scheduled {
		scheduled = append(scheduled, flight.normalize())
	}
	return SchedulesResponse{Scheduled: scheduled}, nil
}

func (c *HTTPClient) GetAccountUsage(ctx context.Context) (UsageResponse, error) {
	var raw rawUsageResponse
	if err := c.get(ctx, EndpointUsage, []string{"account", "usage"}, nil, &raw); err != nil {
		return UsageResponse{}, err
	}
	return raw.normalize(), nil
}

func (c *HTTPClient) get(ctx context.Context, endpoint Endpoint, pathParts []string, query map[string]string, target any) error {
	requestURL := *c.baseURL
	escaped := make([]string, 0, len(pathParts))
	for _, part := range pathParts {
		escaped = append(escaped, url.PathEscape(part))
	}
	requestURL.Path = strings.TrimRight(requestURL.Path, "/") + "/" + strings.Join(escaped, "/")
	values := requestURL.Query()
	for key, value := range query {
		values.Set(key, value)
	}
	requestURL.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("x-apikey", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return NewRateLimitedError(endpoint, resp.StatusCode, "FlightAware rate limit response")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &ClientError{
			Code:       "upstream_status",
			Endpoint:   endpoint,
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("FlightAware %s returned HTTP %d", endpoint, resp.StatusCode),
		}
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

type rawFlightSummary struct {
	FAFlightID  *string        `json:"fa_flight_id"`
	Ident       string         `json:"ident"`
	Origin      airportCodeRaw `json:"origin"`
	Destination airportCodeRaw `json:"destination"`
}

func (f rawFlightSummary) normalize() FlightSummary {
	return FlightSummary{
		FAFlightID:  f.FAFlightID,
		Ident:       f.Ident,
		Origin:      f.Origin.String(),
		Destination: f.Destination.String(),
	}
}

type airportCodeRaw string

func (a *airportCodeRaw) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*a = airportCodeRaw(text)
		return nil
	}
	var object struct {
		Code     string `json:"code"`
		CodeICAO string `json:"code_icao"`
		CodeIATA string `json:"code_iata"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	switch {
	case object.Code != "":
		*a = airportCodeRaw(object.Code)
	case object.CodeICAO != "":
		*a = airportCodeRaw(object.CodeICAO)
	default:
		*a = airportCodeRaw(object.CodeIATA)
	}
	return nil
}

func (a airportCodeRaw) String() string {
	return string(a)
}

type rawUsageResponse struct {
	Currency    string `json:"currency"`
	MonthToDate struct {
		EstimatedCostUSD float64 `json:"estimated_cost_usd"`
		ResultSets       int     `json:"result_sets"`
	} `json:"month_to_date"`
}

func (r rawUsageResponse) normalize() UsageResponse {
	return UsageResponse{
		Currency: r.Currency,
		MonthToDate: UsageAmount{
			EstimatedCostUSD: r.MonthToDate.EstimatedCostUSD,
			ResultSets:       r.MonthToDate.ResultSets,
		},
	}
}
