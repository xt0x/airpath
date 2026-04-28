package domain

import (
	"encoding/json"
	"testing"
)

func TestFlightModelKeepsNullableFlightAwareFields(t *testing.T) {
	flight := Flight{
		FlightID:               "sched_3f5a7c8d91ab",
		FlightIDType:           FlightIDTypeProvisional,
		InternalFlightLegID:    nil,
		ProvisionalFlightLegID: ptr("sched_3f5a7c8d91ab"),
		FAFlightID:             nil,
		Ident:                  "ANA110",
		IdentIATA:              ptr("NH110"),
		Operator:               ptr("ANA"),
		AircraftType:           nil,
		Registration:           nil,
		Origin: Airport{
			Code:     "RJTT",
			Name:     ptr("Tokyo Haneda"),
			Timezone: ptr("Asia/Tokyo"),
		},
		Destination: Airport{
			Code:     "KJFK",
			Name:     ptr("John F. Kennedy International Airport"),
			Timezone: ptr("America/New_York"),
		},
		OriginalDestination: nil,
		Diverted:            false,
		LegIndex:            nil,
		Status:              "Scheduled",
		ProgressPercent:     nil,
		Times: FlightTimes{
			ScheduledOut: ptr("2026-08-01T01:00:00Z"),
		},
		FiledEteSeconds:         nil,
		PlannedRouteS3Key:       nil,
		ActualTrackS3Key:        nil,
		LatestPositionTimestamp: nil,
		LatestPositionSource:    nil,
		PollState:               ptr(FlightPollStateScheduled),
		WatcherCount:            0,
		SessionSubscriberCount:  0,
		PersistentWatcherCount:  0,
		FetchLeaseUntil:         nil,
		FetchOwner:              nil,
		NextSummaryPollAt:       nil,
		NextPositionPollAt:      nil,
		NextTrackPollAt:         nil,
		NextRoutePollAt:         nil,
		IdleSince:               nil,
		UpdatedAt:               "2026-04-25T15:00:00Z",
		TTL:                     nil,
	}

	body, err := json.Marshal(flight)
	if err != nil {
		t.Fatalf("marshal flight: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal flight: %v", err)
	}
	if got["faFlightId"] != nil {
		t.Fatalf("faFlightId = %v, want nil", got["faFlightId"])
	}
	if got["aircraftType"] != nil {
		t.Fatalf("aircraftType = %v, want nil", got["aircraftType"])
	}
	if got["registration"] != nil {
		t.Fatalf("registration = %v, want nil", got["registration"])
	}
}

func TestPositionAndEventModels(t *testing.T) {
	position := FlightPosition{
		FlightID:               "iflg_8d2a2a3a6c4f",
		InternalFlightLegID:    ptr("iflg_8d2a2a3a6c4f"),
		ProvisionalFlightLegID: nil,
		FAFlightID:             ptr("UAL1234-1234567890-airline-0123"),
		Latitude:               45.123,
		Longitude:              160.456,
		AltitudeHundredsFeet:   ptr(370),
		AltitudeFeet:           ptr(37000),
		AltitudeChange:         ptr(AltitudeChangeLevel),
		GroundspeedKnots:       ptr(488),
		HeadingDegrees:         ptr(275),
		Timestamp:              "2026-04-25T15:05:00Z",
		UpdateType:             ptr(PositionUpdateTypeEstimated),
		Source:                 PositionSourceFlightAwarePosition,
		TTL:                    nil,
	}
	event := FlightEvent{
		FlightID:               "iflg_8d2a2a3a6c4f",
		InternalFlightLegID:    ptr("iflg_8d2a2a3a6c4f"),
		ProvisionalFlightLegID: nil,
		FAFlightID:             ptr("UAL1234-1234567890-airline-0123"),
		DedupeKey:              "dedupe_123",
		EventType:              FlightEventTypeDeparture,
		EventTimestamp:         "2026-04-25T10:08:00Z",
		Payload:                map[string]any{"sourceStatus": "En Route"},
		Source:                 FlightEventSourcePolling,
		AppliedToFlightState:   true,
		CreatedAt:              "2026-04-25T15:05:01Z",
	}

	if *position.AltitudeFeet != 37000 {
		t.Fatalf("AltitudeFeet = %d, want 37000", *position.AltitudeFeet)
	}
	if position.Source != PositionSourceFlightAwarePosition {
		t.Fatalf("Source = %q", position.Source)
	}
	if event.Payload["sourceStatus"] != "En Route" {
		t.Fatalf("payload sourceStatus = %v", event.Payload["sourceStatus"])
	}
}

func TestWatchAndSubscriptionModelsStaySeparate(t *testing.T) {
	watch := UserWatch{
		UserID:       "user_123",
		FlightID:     "iflg_8d2a2a3a6c4f",
		FAFlightID:   ptr("UAL1234-1234567890-airline-0123"),
		FlightIDType: FlightIDTypeInternal,
		WatchState:   WatchStateActive,
		CreatedAt:    "2026-04-25T15:00:00Z",
		UpdatedAt:    "2026-04-25T15:00:00Z",
		TTL:          nil,
	}
	subscription := FlightSubscription{
		FlightID:         "iflg_8d2a2a3a6c4f",
		ConnectionID:     "abc123",
		UserID:           "user_123",
		SubscriptionType: SubscriptionTypeSession,
		CreatedAt:        "2026-04-25T15:01:00Z",
		LastSeenAt:       "2026-04-25T15:02:00Z",
		TTL:              ptr(int64(1770000000)),
	}

	if watch.WatchState != WatchStateActive {
		t.Fatalf("WatchState = %q", watch.WatchState)
	}
	if subscription.SubscriptionType != SubscriptionTypeSession {
		t.Fatalf("SubscriptionType = %q", subscription.SubscriptionType)
	}
}

func ptr[T any](value T) *T {
	return &value
}
