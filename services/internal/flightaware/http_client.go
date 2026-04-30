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
	if err := c.get(ctx, EndpointSearch, []string{"flights", request.Ident}, map[string]string{
		"ident_type": "designator",
		"max_pages":  fmt.Sprint(maxPages),
	}, &raw); err != nil {
		return SearchFlightsResponse{}, err
	}
	flights := make([]FlightSummary, 0, len(raw.Flights))
	for _, flight := range raw.Flights {
		flights = append(flights, flight.normalize())
	}
	return SearchFlightsResponse{Flights: flights}, nil
}

func (c *HTTPClient) GetFlightSummary(ctx context.Context, request FlightSummaryRequest) (FlightSummary, error) {
	var raw struct {
		Flights []rawFlightSummary `json:"flights"`
	}
	if err := c.get(ctx, EndpointSummary, []string{"flights", request.FAFlightID}, map[string]string{"ident_type": "fa_flight_id"}, &raw); err != nil {
		return FlightSummary{}, err
	}
	if len(raw.Flights) == 0 {
		return FlightSummary{}, &ClientError{
			Code:     "upstream_empty_response",
			Endpoint: EndpointSummary,
			Message:  "FlightAware summary response contained no flights",
		}
	}
	return raw.Flights[0].normalize(), nil
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
		FAFlightID   string `json:"fa_flight_id"`
		LastPosition *struct {
			FAFlightID           *string  `json:"fa_flight_id"`
			Latitude             *float64 `json:"latitude"`
			Longitude            *float64 `json:"longitude"`
			AltitudeHundredsFeet *int     `json:"altitude"`
			AltitudeChange       *string  `json:"altitude_change"`
			GroundspeedKnots     *int     `json:"groundspeed"`
			HeadingDegrees       *int     `json:"heading"`
			Timestamp            string   `json:"timestamp"`
			UpdateType           *string  `json:"update_type"`
		} `json:"last_position"`
	}
	if err := c.get(ctx, EndpointPosition, []string{"flights", request.FAFlightID, "position"}, nil, &raw); err != nil {
		return PositionResponse{}, err
	}
	if raw.LastPosition == nil {
		return PositionResponse{}, &ClientError{
			Code:     "upstream_empty_response",
			Endpoint: EndpointPosition,
			Message:  "FlightAware position response contained no last_position",
		}
	}
	faFlightID := raw.FAFlightID
	if raw.LastPosition.FAFlightID != nil && *raw.LastPosition.FAFlightID != "" {
		faFlightID = *raw.LastPosition.FAFlightID
	}
	return PositionResponse{
		FAFlightID:           faFlightID,
		Latitude:             raw.LastPosition.Latitude,
		Longitude:            raw.LastPosition.Longitude,
		AltitudeHundredsFeet: raw.LastPosition.AltitudeHundredsFeet,
		AltitudeChange:       normalizeAltitudeChange(raw.LastPosition.AltitudeChange),
		GroundspeedKnots:     raw.LastPosition.GroundspeedKnots,
		HeadingDegrees:       raw.LastPosition.HeadingDegrees,
		Timestamp:            raw.LastPosition.Timestamp,
		UpdateType:           normalizeUpdateType(raw.LastPosition.UpdateType),
	}, nil
}

func (c *HTTPClient) GetFlightTrack(ctx context.Context, request FlightTrackRequest) (TrackResponse, error) {
	var raw struct {
		Positions []rawTrackPoint `json:"positions"`
	}
	if err := c.get(ctx, EndpointTrack, []string{"flights", request.FAFlightID, "track"}, nil, &raw); err != nil {
		return TrackResponse{}, err
	}
	positions := make([]TrackPoint, 0, len(raw.Positions))
	for _, position := range raw.Positions {
		positions = append(positions, position.normalize())
	}
	return TrackResponse{Positions: positions}, nil
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
	FAFlightID      *string        `json:"fa_flight_id"`
	Ident           string         `json:"ident"`
	IdentIATA       *string        `json:"ident_iata"`
	AircraftType    *string        `json:"aircraft_type"`
	Registration    *string        `json:"registration"`
	Origin          airportCodeRaw `json:"origin"`
	Destination     airportCodeRaw `json:"destination"`
	ScheduledOut    *string        `json:"scheduled_out"`
	EstimatedOut    *string        `json:"estimated_out"`
	ActualOut       *string        `json:"actual_out"`
	ScheduledOff    *string        `json:"scheduled_off"`
	EstimatedOff    *string        `json:"estimated_off"`
	ActualOff       *string        `json:"actual_off"`
	ScheduledOn     *string        `json:"scheduled_on"`
	EstimatedOn     *string        `json:"estimated_on"`
	ActualOn        *string        `json:"actual_on"`
	ScheduledIn     *string        `json:"scheduled_in"`
	EstimatedIn     *string        `json:"estimated_in"`
	ActualIn        *string        `json:"actual_in"`
	Status          string         `json:"status"`
	ProgressPercent *int           `json:"progress_percent"`
	FiledEteSeconds *int           `json:"filed_ete"`
}

type rawTrackPoint struct {
	Latitude             float64 `json:"latitude"`
	Longitude            float64 `json:"longitude"`
	AltitudeHundredsFeet *int    `json:"altitude"`
	AltitudeChange       *string `json:"altitude_change"`
	GroundspeedKnots     *int    `json:"groundspeed"`
	HeadingDegrees       *int    `json:"heading"`
	Timestamp            string  `json:"timestamp"`
	UpdateType           *string `json:"update_type"`
}

func (p rawTrackPoint) normalize() TrackPoint {
	return TrackPoint{
		Latitude:             p.Latitude,
		Longitude:            p.Longitude,
		AltitudeHundredsFeet: p.AltitudeHundredsFeet,
		AltitudeChange:       normalizeAltitudeChange(p.AltitudeChange),
		GroundspeedKnots:     p.GroundspeedKnots,
		HeadingDegrees:       p.HeadingDegrees,
		Timestamp:            p.Timestamp,
		UpdateType:           normalizeUpdateType(p.UpdateType),
	}
}

func normalizeAltitudeChange(value *string) *string {
	if value == nil {
		return nil
	}
	var normalized string
	switch *value {
	case "C", "climbing":
		normalized = "climbing"
	case "D", "descending":
		normalized = "descending"
	case "-", "level":
		normalized = "level"
	default:
		return nil
	}
	return &normalized
}

func normalizeUpdateType(value *string) *string {
	if value == nil {
		return nil
	}
	var normalized string
	switch *value {
	case "A", "D", "M", "O", "S", "Z", "actual":
		normalized = "actual"
	case "P", "predicted":
		normalized = "predicted"
	case "V", "estimated":
		normalized = "estimated"
	case "X", "surface":
		normalized = "surface"
	default:
		return nil
	}
	return &normalized
}

func (f rawFlightSummary) normalize() FlightSummary {
	return FlightSummary{
		FAFlightID:      f.FAFlightID,
		Ident:           f.Ident,
		IdentIATA:       f.IdentIATA,
		AircraftType:    f.AircraftType,
		Registration:    f.Registration,
		Origin:          f.Origin.String(),
		Destination:     f.Destination.String(),
		ScheduledOut:    f.ScheduledOut,
		EstimatedOut:    f.EstimatedOut,
		ActualOut:       f.ActualOut,
		ScheduledOff:    f.ScheduledOff,
		EstimatedOff:    f.EstimatedOff,
		ActualOff:       f.ActualOff,
		ScheduledOn:     f.ScheduledOn,
		EstimatedOn:     f.EstimatedOn,
		ActualOn:        f.ActualOn,
		ScheduledIn:     f.ScheduledIn,
		EstimatedIn:     f.EstimatedIn,
		ActualIn:        f.ActualIn,
		Status:          f.Status,
		ProgressPercent: f.ProgressPercent,
		FiledEteSeconds: f.FiledEteSeconds,
	}
}

type airportCodeRaw string

func (a *airportCodeRaw) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*a = airportCodeRaw(text)
		return nil
	}
	// FlightAware endpoints are inconsistent: airports may arrive as plain codes
	// or objects. Prefer the canonical code field, then ICAO, then IATA.
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
	TotalCostUSD         float64  `json:"total_cost"`
	TotalDiscountCostUSD *float64 `json:"total_discount_cost"`
	TotalPages           int      `json:"total_pages"`
}

func (r rawUsageResponse) normalize() UsageResponse {
	estimatedCostUSD := r.TotalCostUSD
	if r.TotalDiscountCostUSD != nil {
		estimatedCostUSD = *r.TotalDiscountCostUSD
	}
	return UsageResponse{
		Currency: "USD",
		MonthToDate: UsageAmount{
			EstimatedCostUSD: estimatedCostUSD,
			ResultSets:       r.TotalPages,
		},
	}
}
