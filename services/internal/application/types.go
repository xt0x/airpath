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
	Freshness CacheFreshness
	Source    CacheSource
	Stale     bool
	CheckedAt string
	FetchedAt *string
	StaleAt   *string
	ExpiresAt *string
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
	SchemaVersion  int
	TaskID         string
	TaskType       FetchTaskType
	FlightID       domain.FlightID
	FAFlightID     *domain.FAFlightID
	RequestedAt    string
	Reason         FetchReason
	IdempotencyKey string
	NotBefore      *string
	Attempt        int
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
	Source            MapSource
	Available         bool
	GeoJSON           map[string]any
	UnavailableReason *string
}

type Position struct {
	Latitude             float64
	Longitude            float64
	AltitudeHundredsFeet *int
	AltitudeFeet         *int
	GroundspeedKnots     *int
	HeadingDegrees       *int
	Timestamp            string
	Source               domain.PositionSource
}

type FlightSummaryItem struct {
	FlightID               domain.FlightID
	FlightIDType           domain.FlightIDType
	ProvisionalFlightLegID *domain.ProvisionalFlightLegID
	FAFlightID             *domain.FAFlightID
	Ident                  string
	IdentIATA              *string
	Origin                 domain.AirportCode
	Destination            domain.AirportCode
	ScheduledOut           *domain.ISODateTimeString
	LegIndex               *int
	Status                 string
}

type FlightSearchResponse struct {
	Items []FlightSummaryItem
	Cache CacheMetadata
}

type FlightDetailResponse struct {
	Flight  domain.Flight
	Route   MapLayer
	Track   MapLayer
	Current *Position
	Cache   CacheMetadata
}

type FlightMapDataResponse struct {
	FlightID   domain.FlightID
	FAFlightID *domain.FAFlightID
	Planned    MapLayer
	Actual     MapLayer
	Current    MapLayer
	Cache      CacheMetadata
}

type FlightRefreshResponse struct {
	FlightID      domain.FlightID
	AcceptedTasks []FetchTask
	Cache         CacheMetadata
}

type UsageBudgetStatus struct {
	Currency                 string
	EstimatedMonthToDateCost float64
	SoftStopThreshold        float64
	Stopped                  bool
}

type RateLimitStatus struct {
	Limited bool
	ResetAt *string
}

type UsageStatus struct {
	Budget          UsageBudgetStatus
	RateLimit       RateLimitStatus
	FetchingEnabled bool
	Cache           CacheMetadata
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
	Code                ApiErrorCode
	Message             string
	Retryable           bool
	RequestID           string
	RetryAfterSeconds   *int
	StaleCacheAvailable bool
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
