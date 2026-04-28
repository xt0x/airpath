package application

import (
	"errors"

	"airpath/services/internal/domain"
)

type CacheFreshness string

const (
	CacheFreshnessFresh CacheFreshness = "fresh"
	CacheFreshnessStale CacheFreshness = "stale"
	CacheFreshnessMiss  CacheFreshness = "miss"
)

type CacheSource string

const (
	CacheSourceCache           CacheSource = "cache"
	CacheSourceFlightAware     CacheSource = "flightaware"
	CacheSourceLocalAccounting CacheSource = "local_accounting"
	CacheSourceDerived         CacheSource = "derived"
)

type CacheMetadata struct {
	Freshness CacheFreshness `json:"freshness"`
	Source    CacheSource    `json:"source"`
	Stale     bool           `json:"stale"`
	CheckedAt string         `json:"checkedAt"`
	FetchedAt *string        `json:"fetchedAt,omitempty"`
	StaleAt   *string        `json:"staleAt,omitempty"`
	ExpiresAt *string        `json:"expiresAt,omitempty"`
}

type FetchTaskType string

const (
	FetchTaskSummary    FetchTaskType = "summary"
	FetchTaskPosition   FetchTaskType = "position"
	FetchTaskRoute      FetchTaskType = "route"
	FetchTaskTrack      FetchTaskType = "track"
	FetchTaskFinalTrack FetchTaskType = "final_track"
)

type FetchReason string

const (
	FetchReasonSearchResultSeed    FetchReason = "search_result_seed"
	FetchReasonUserOpenedDetail    FetchReason = "user_opened_flight_detail"
	FetchReasonUserManualRefresh   FetchReason = "user_manual_refresh"
	FetchReasonLowFrequencyPoll    FetchReason = "low_frequency_poll"
	FetchReasonArrivalFinalization FetchReason = "arrival_finalization"
	FetchReasonUsageReconciliation FetchReason = "usage_reconciliation"
)

type FetchTask struct {
	SchemaVersion  int                `json:"schemaVersion"`
	TaskID         string             `json:"taskId"`
	TaskType       FetchTaskType      `json:"taskType"`
	FlightID       domain.FlightID    `json:"flightId"`
	FAFlightID     *domain.FAFlightID `json:"faFlightId"`
	RequestedAt    string             `json:"requestedAt"`
	Reason         FetchReason        `json:"reason"`
	IdempotencyKey string             `json:"idempotencyKey"`
	NotBefore      *string            `json:"notBefore,omitempty"`
	Attempt        int                `json:"attempt"`
}

type MapSource string

const (
	MapSourceFlightAwareRoute            MapSource = "flightaware_route"
	MapSourceAirportGreatCircleFallback  MapSource = "airport_great_circle_fallback"
	MapSourceFlightAwareTrack            MapSource = "flightaware_track"
	MapSourceFlightAwareTrackAndPosition MapSource = "flightaware_track_and_position"
	MapSourceFlightAwarePosition         MapSource = "flightaware_position"
)

type MapLayer struct {
	Source            MapSource      `json:"source"`
	Available         bool           `json:"available"`
	GeoJSON           map[string]any `json:"geojson"`
	UnavailableReason *string        `json:"unavailableReason,omitempty"`
}

type Position struct {
	Latitude             float64               `json:"latitude"`
	Longitude            float64               `json:"longitude"`
	AltitudeHundredsFeet *int                  `json:"altitudeHundredsFeet"`
	AltitudeFeet         *int                  `json:"altitudeFeet"`
	GroundspeedKnots     *int                  `json:"groundspeedKnots"`
	HeadingDegrees       *int                  `json:"headingDegrees"`
	Timestamp            string                `json:"timestamp"`
	Source               domain.PositionSource `json:"source"`
}

type FlightSummaryItem struct {
	FlightID               domain.FlightID                `json:"flightId"`
	FlightIDType           domain.FlightIDType            `json:"flightIdType"`
	ProvisionalFlightLegID *domain.ProvisionalFlightLegID `json:"provisionalFlightLegId"`
	FAFlightID             *domain.FAFlightID             `json:"faFlightId"`
	Ident                  string                         `json:"ident"`
	IdentIATA              *string                        `json:"identIata"`
	Origin                 domain.AirportCode             `json:"origin"`
	Destination            domain.AirportCode             `json:"destination"`
	ScheduledOut           *domain.ISODateTimeString      `json:"scheduledOut"`
	LegIndex               *int                           `json:"legIndex"`
	Status                 string                         `json:"status"`
}

type FlightSearchResponse struct {
	Items []FlightSummaryItem `json:"items"`
	Cache CacheMetadata       `json:"cache"`
}

type FlightDetailResponse struct {
	Flight  domain.Flight `json:"flight"`
	Route   MapLayer      `json:"route"`
	Track   MapLayer      `json:"track"`
	Current *Position     `json:"current"`
	Cache   CacheMetadata `json:"cache"`
}

type FlightMapDataResponse struct {
	FlightID   domain.FlightID    `json:"flightId"`
	FAFlightID *domain.FAFlightID `json:"faFlightId"`
	Planned    MapLayer           `json:"planned"`
	Actual     MapLayer           `json:"actual"`
	Current    MapLayer           `json:"current"`
	Cache      CacheMetadata      `json:"cache"`
}

type FlightRefreshResponse struct {
	FlightID      domain.FlightID `json:"flightId"`
	AcceptedTasks []FetchTask     `json:"acceptedTasks"`
	Cache         CacheMetadata   `json:"cache"`
}

type UsageBudgetStatus struct {
	Currency                 string  `json:"currency"`
	EstimatedMonthToDateCost float64 `json:"estimatedMonthToDateCost"`
	SoftStopThreshold        float64 `json:"softStopThreshold"`
	Stopped                  bool    `json:"stopped"`
}

type RateLimitStatus struct {
	Limited bool    `json:"limited"`
	ResetAt *string `json:"resetAt"`
}

type UsageStatus struct {
	Budget          UsageBudgetStatus `json:"budget"`
	RateLimit       RateLimitStatus   `json:"rateLimit"`
	FetchingEnabled bool              `json:"fetchingEnabled"`
	Cache           CacheMetadata     `json:"cache"`
}

type SearchFlightsInput struct {
	Ident     string
	CheckedAt string
}

type FlightDetailInput struct {
	FlightID  domain.FlightID
	CheckedAt string
}

type FlightMapDataInput struct {
	FlightID  domain.FlightID
	CheckedAt string
}

type FlightRefreshInput struct {
	FlightID      domain.FlightID
	TaskTypes     []FetchTaskType
	ClientReason  FetchReason
	RequestedAt   string
	IdempotencyID string
}

type UsageStatusInput struct {
	CheckedAt string
}

type ApiErrorCode string

const (
	ApiErrorFlightAwareBudgetExceeded ApiErrorCode = "flightaware_budget_exceeded"
	ApiErrorFlightAwareRateLimited    ApiErrorCode = "flightaware_rate_limited"
	ApiErrorStaleCacheUnavailable     ApiErrorCode = "stale_cache_unavailable"
	ApiErrorUpstreamFailure           ApiErrorCode = "upstream_failure"
	ApiErrorFlightAwareFetchDisabled  ApiErrorCode = "flightaware_fetch_disabled"
)

type APIError struct {
	Code                ApiErrorCode `json:"code"`
	Message             string       `json:"message"`
	Retryable           bool         `json:"retryable"`
	RequestID           string       `json:"requestId"`
	RetryAfterSeconds   *int         `json:"retryAfterSeconds,omitempty"`
	StaleCacheAvailable bool         `json:"staleCacheAvailable"`
}

var (
	ErrBudgetExceeded        = errors.New("flightaware budget exceeded")
	ErrStaleCacheUnavailable = errors.New("stale cache unavailable")
	ErrNotFound              = errors.New("not found")
	ErrValidation            = errors.New("validation failed")
)

func ptr(value string) *string {
	return &value
}
