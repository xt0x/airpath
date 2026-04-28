package domain

type ISODateTimeString = string
type EpochSeconds = int64
type FlightID = string
type InternalFlightLegID = string
type ProvisionalFlightLegID = string
type FAFlightID = string
type UserID = string
type ConnectionID = string
type AirportCode = string

type FlightIDType string

const (
	FlightIDTypeInternal    FlightIDType = "internal"
	FlightIDTypeProvisional FlightIDType = "provisional"
)

type FlightPollState string

const (
	FlightPollStateScheduled FlightPollState = "scheduled"
	FlightPollStatePreparing FlightPollState = "preparing"
	FlightPollStateActive    FlightPollState = "active"
	FlightPollStateArrived   FlightPollState = "arrived"
	FlightPollStateCancelled FlightPollState = "cancelled"
	FlightPollStateDiverted  FlightPollState = "diverted"
	FlightPollStateCompleted FlightPollState = "completed"
	FlightPollStateError     FlightPollState = "error"
)

type PositionSource string

const (
	PositionSourceFlightAwarePosition PositionSource = "flightaware_position"
	PositionSourceFlightAwareTrack    PositionSource = "flightaware_track"
	PositionSourceAlert               PositionSource = "alert"
	PositionSourceManual              PositionSource = "manual"
)

type PositionUpdateType string

const (
	PositionUpdateTypeActual    PositionUpdateType = "actual"
	PositionUpdateTypeEstimated PositionUpdateType = "estimated"
	PositionUpdateTypePredicted PositionUpdateType = "predicted"
	PositionUpdateTypeSurface   PositionUpdateType = "surface"
)

type AltitudeChange string

const (
	AltitudeChangeClimbing   AltitudeChange = "climbing"
	AltitudeChangeDescending AltitudeChange = "descending"
	AltitudeChangeLevel      AltitudeChange = "level"
)

type FlightEventType string

const (
	FlightEventTypeDeparture      FlightEventType = "departure"
	FlightEventTypeOff            FlightEventType = "off"
	FlightEventTypeOn             FlightEventType = "on"
	FlightEventTypeArrival        FlightEventType = "arrival"
	FlightEventTypeCancelled      FlightEventType = "cancelled"
	FlightEventTypeDiverted       FlightEventType = "diverted"
	FlightEventTypeDepartureDelay FlightEventType = "departure_delay"
	FlightEventTypeArrivalDelay   FlightEventType = "arrival_delay"
	FlightEventTypeStatusUpdated  FlightEventType = "status_updated"
)

type FlightEventSource string

const (
	FlightEventSourcePolling FlightEventSource = "polling"
	FlightEventSourceAlert   FlightEventSource = "alert"
)

type WatchState string

const (
	WatchStateActive  WatchState = "active"
	WatchStatePaused  WatchState = "paused"
	WatchStateRemoved WatchState = "removed"
)

type SubscriptionType string

const (
	SubscriptionTypeSession SubscriptionType = "session"
)

type Airport struct {
	Code     AirportCode `json:"code"`
	Name     *string     `json:"name"`
	Timezone *string     `json:"timezone"`
}

type FlightTimes struct {
	ScheduledOut *ISODateTimeString `json:"scheduledOut"`
	EstimatedOut *ISODateTimeString `json:"estimatedOut"`
	ActualOut    *ISODateTimeString `json:"actualOut"`
	ScheduledOff *ISODateTimeString `json:"scheduledOff"`
	EstimatedOff *ISODateTimeString `json:"estimatedOff"`
	ActualOff    *ISODateTimeString `json:"actualOff"`
	ScheduledOn  *ISODateTimeString `json:"scheduledOn"`
	EstimatedOn  *ISODateTimeString `json:"estimatedOn"`
	ActualOn     *ISODateTimeString `json:"actualOn"`
	ScheduledIn  *ISODateTimeString `json:"scheduledIn"`
	EstimatedIn  *ISODateTimeString `json:"estimatedIn"`
	ActualIn     *ISODateTimeString `json:"actualIn"`
}

type Flight struct {
	FlightID                FlightID                `json:"flightId"`
	FlightIDType            FlightIDType            `json:"flightIdType"`
	InternalFlightLegID     *InternalFlightLegID    `json:"internalFlightLegId"`
	ProvisionalFlightLegID  *ProvisionalFlightLegID `json:"provisionalFlightLegId"`
	FAFlightID              *FAFlightID             `json:"faFlightId"`
	Ident                   string                  `json:"ident"`
	IdentIATA               *string                 `json:"identIata"`
	Operator                *string                 `json:"operator"`
	AircraftType            *string                 `json:"aircraftType"`
	Registration            *string                 `json:"registration"`
	Origin                  Airport                 `json:"origin"`
	Destination             Airport                 `json:"destination"`
	OriginalDestination     *Airport                `json:"originalDestination"`
	Diverted                bool                    `json:"diverted"`
	LegIndex                *int                    `json:"legIndex"`
	Status                  string                  `json:"status"`
	ProgressPercent         *int                    `json:"progressPercent"`
	Times                   FlightTimes             `json:"times"`
	FiledEteSeconds         *int                    `json:"filedEteSeconds"`
	PlannedRouteS3Key       *string                 `json:"plannedRouteS3Key"`
	ActualTrackS3Key        *string                 `json:"actualTrackS3Key"`
	LatestPositionTimestamp *ISODateTimeString      `json:"latestPositionTimestamp"`
	LatestPositionSource    *PositionSource         `json:"latestPositionSource"`
	PollState               *FlightPollState        `json:"pollState"`
	WatcherCount            int                     `json:"watcherCount"`
	SessionSubscriberCount  int                     `json:"sessionSubscriberCount"`
	PersistentWatcherCount  int                     `json:"persistentWatcherCount"`
	FetchLeaseUntil         *ISODateTimeString      `json:"fetchLeaseUntil"`
	FetchOwner              *string                 `json:"fetchOwner"`
	NextSummaryPollAt       *ISODateTimeString      `json:"nextSummaryPollAt"`
	NextPositionPollAt      *ISODateTimeString      `json:"nextPositionPollAt"`
	NextTrackPollAt         *ISODateTimeString      `json:"nextTrackPollAt"`
	NextRoutePollAt         *ISODateTimeString      `json:"nextRoutePollAt"`
	IdleSince               *ISODateTimeString      `json:"idleSince"`
	UpdatedAt               ISODateTimeString       `json:"updatedAt"`
	TTL                     *EpochSeconds           `json:"ttl"`
}

type FlightPosition struct {
	FlightID               FlightID                `json:"flightId"`
	InternalFlightLegID    *InternalFlightLegID    `json:"internalFlightLegId"`
	ProvisionalFlightLegID *ProvisionalFlightLegID `json:"provisionalFlightLegId"`
	FAFlightID             *FAFlightID             `json:"faFlightId"`
	Latitude               float64                 `json:"latitude"`
	Longitude              float64                 `json:"longitude"`
	AltitudeHundredsFeet   *int                    `json:"altitudeHundredsFeet"`
	AltitudeFeet           *int                    `json:"altitudeFeet"`
	AltitudeChange         *AltitudeChange         `json:"altitudeChange"`
	GroundspeedKnots       *int                    `json:"groundspeedKnots"`
	HeadingDegrees         *int                    `json:"headingDegrees"`
	Timestamp              ISODateTimeString       `json:"timestamp"`
	UpdateType             *PositionUpdateType     `json:"updateType"`
	Source                 PositionSource          `json:"source"`
	TTL                    *EpochSeconds           `json:"ttl"`
}

type FlightEvent struct {
	FlightID               FlightID                `json:"flightId"`
	InternalFlightLegID    *InternalFlightLegID    `json:"internalFlightLegId"`
	ProvisionalFlightLegID *ProvisionalFlightLegID `json:"provisionalFlightLegId"`
	FAFlightID             *FAFlightID             `json:"faFlightId"`
	DedupeKey              string                  `json:"dedupeKey"`
	EventType              FlightEventType         `json:"eventType"`
	EventTimestamp         ISODateTimeString       `json:"eventTimestamp"`
	Payload                map[string]any          `json:"payload"`
	Source                 FlightEventSource       `json:"source"`
	AppliedToFlightState   bool                    `json:"appliedToFlightState"`
	CreatedAt              ISODateTimeString       `json:"createdAt"`
}

type UserWatch struct {
	UserID       UserID            `json:"userId"`
	FlightID     FlightID          `json:"flightId"`
	FAFlightID   *FAFlightID       `json:"faFlightId"`
	FlightIDType FlightIDType      `json:"flightIdType"`
	WatchState   WatchState        `json:"watchState"`
	CreatedAt    ISODateTimeString `json:"createdAt"`
	UpdatedAt    ISODateTimeString `json:"updatedAt"`
	TTL          *EpochSeconds     `json:"ttl"`
}

type FlightSubscription struct {
	FlightID         FlightID          `json:"flightId"`
	ConnectionID     ConnectionID      `json:"connectionId"`
	UserID           UserID            `json:"userId"`
	SubscriptionType SubscriptionType  `json:"subscriptionType"`
	CreatedAt        ISODateTimeString `json:"createdAt"`
	LastSeenAt       ISODateTimeString `json:"lastSeenAt"`
	TTL              *EpochSeconds     `json:"ttl"`
}
