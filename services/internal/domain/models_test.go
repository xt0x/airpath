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

func TestGenerateProvisionalFlightLegID(t *testing.T) {
	input := ProvisionalFlightLegIDInput{
		Ident:           "ANA110",
		OriginCode:      "RJTT",
		DestinationCode: "KJFK",
		ScheduledOut:    "2026-08-01T01:00:00Z",
	}

	got := GenerateProvisionalFlightLegID(input)
	if got != "sched_c9df8a3ee088" {
		t.Fatalf("GenerateProvisionalFlightLegID() = %q, want %q", got, "sched_c9df8a3ee088")
	}
	if got != GenerateProvisionalFlightLegID(input) {
		t.Fatal("GenerateProvisionalFlightLegID() is not stable for identical input")
	}

	input.DestinationCode = "KLAX"
	if got := GenerateProvisionalFlightLegID(input); got != "sched_b76cfb4f33db" {
		t.Fatalf("GenerateProvisionalFlightLegID() with changed destination = %q, want %q", got, "sched_b76cfb4f33db")
	}
}

func TestGenerateInternalFlightLegID(t *testing.T) {
	input := InternalFlightLegIDInput{
		FAFlightID:      "UAL1234-1234567890-airline-0123",
		OriginCode:      "KSFO",
		DestinationCode: "RJTT",
		ScheduledOut:    "2026-04-25T10:00:00Z",
		LegIndex:        0,
	}

	got := GenerateInternalFlightLegID(input)
	if got != "iflg_de338117b7ac" {
		t.Fatalf("GenerateInternalFlightLegID() = %q, want %q", got, "iflg_de338117b7ac")
	}
	if got != GenerateInternalFlightLegID(input) {
		t.Fatal("GenerateInternalFlightLegID() is not stable for identical input")
	}

	input.LegIndex = 1
	if got := GenerateInternalFlightLegID(input); got != "iflg_de338217b7ac" {
		t.Fatalf("GenerateInternalFlightLegID() with changed legIndex = %q, want %q", got, "iflg_de338217b7ac")
	}
}

func TestNullableDisplayTextKeepsMissingValuesExplicit(t *testing.T) {
	if got := NullableDateTimeDisplayText(nil, MissingValueReasonNotAnnounced); got != "未発表" {
		t.Fatalf("NullableDateTimeDisplayText(nil) = %q, want %q", got, "未発表")
	}
	if got := NullableTextDisplayText(nil, MissingValueReasonNotAcquired); got != "未取得" {
		t.Fatalf("NullableTextDisplayText(nil) = %q, want %q", got, "未取得")
	}
	if got := NullableTextDisplayText(nil, MissingValueReasonUnavailable); got != "取得不可" {
		t.Fatalf("NullableTextDisplayText(nil unavailable) = %q, want %q", got, "取得不可")
	}
	if got := NullableTextDisplayText(ptr("B789"), MissingValueReasonNotAcquired); got != "B789" {
		t.Fatalf("NullableTextDisplayText(B789) = %q, want %q", got, "B789")
	}
	if got := NullableTextDisplayText(ptr("N12345"), MissingValueReasonNotAcquired); got != "N12345" {
		t.Fatalf("NullableTextDisplayText(N12345) = %q, want %q", got, "N12345")
	}
}

func TestNullableProgressDisplayTextDoesNotTreatZeroAsMissing(t *testing.T) {
	if got := NullableProgressDisplayText(nil, MissingValueReasonNotAcquired); got != "未取得" {
		t.Fatalf("NullableProgressDisplayText(nil) = %q, want %q", got, "未取得")
	}
	if got := NullableProgressDisplayText(ptr(0), MissingValueReasonNotAcquired); got != "0%" {
		t.Fatalf("NullableProgressDisplayText(0) = %q, want %q", got, "0%%")
	}
	if got := NullableProgressDisplayText(ptr(62), MissingValueReasonNotAcquired); got != "62%" {
		t.Fatalf("NullableProgressDisplayText(62) = %q, want %q", got, "62%%")
	}
}

func TestNullableAirportDisplayTextKeepsKnownCodes(t *testing.T) {
	airportWithMissingDetails := Airport{
		Code:     "RJTT",
		Name:     nil,
		Timezone: nil,
	}

	if got := NullableAirportDisplayText(nil, MissingValueReasonNotAcquired); got != "未取得" {
		t.Fatalf("NullableAirportDisplayText(nil) = %q, want %q", got, "未取得")
	}
	if got := NullableAirportDisplayText(&airportWithMissingDetails, MissingValueReasonNotAcquired); got != "RJTT" {
		t.Fatalf("NullableAirportDisplayText(code only) = %q, want %q", got, "RJTT")
	}
	airportWithMissingDetails.Name = ptr("Tokyo Haneda")
	if got := NullableAirportDisplayText(&airportWithMissingDetails, MissingValueReasonNotAcquired); got != "RJTT - Tokyo Haneda" {
		t.Fatalf("NullableAirportDisplayText(with name) = %q, want %q", got, "RJTT - Tokyo Haneda")
	}
}

func TestNullableDisplayValueCarriesStructuredMissingMetadata(t *testing.T) {
	missing := ToNullableDisplayValue[string](nil, MissingValueReasonNotApplicable)
	if missing.Kind != NullableDisplayKindMissing {
		t.Fatalf("missing.Kind = %q, want %q", missing.Kind, NullableDisplayKindMissing)
	}
	if missing.Reason == nil || *missing.Reason != MissingValueReasonNotApplicable {
		t.Fatalf("missing.Reason = %v, want %q", missing.Reason, MissingValueReasonNotApplicable)
	}
	if missing.Label == nil || *missing.Label != "対象外" {
		t.Fatalf("missing.Label = %v, want %q", missing.Label, "対象外")
	}

	value := "2026-04-25T10:00:00Z"
	available := ToNullableDisplayValue(&value, MissingValueReasonNotAnnounced)
	if available.Kind != NullableDisplayKindAvailable {
		t.Fatalf("available.Kind = %q, want %q", available.Kind, NullableDisplayKindAvailable)
	}
	if available.Value == nil || *available.Value != value {
		t.Fatalf("available.Value = %v, want %q", available.Value, value)
	}
}

func TestNormalizeUTCISODateTime(t *testing.T) {
	cases := map[string]ISODateTimeString{
		"2026-04-25T10:00:00Z":      "2026-04-25T10:00:00Z",
		"2026-04-25T03:00:00-07:00": "2026-04-25T10:00:00Z",
		"2026-04-25T10:00:00.123Z":  "2026-04-25T10:00:00Z",
	}

	for input, want := range cases {
		got, err := NormalizeUTCISODateTime(input)
		if err != nil {
			t.Fatalf("NormalizeUTCISODateTime(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("NormalizeUTCISODateTime(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeLocalDateTimeToUTCISO(t *testing.T) {
	cases := []struct {
		input NormalizeLocalDateTimeInput
		want  ISODateTimeString
	}{
		{
			input: NormalizeLocalDateTimeInput{
				LocalDateTime: "2026-04-25T19:00:00",
				TimeZone:      "Asia/Tokyo",
			},
			want: "2026-04-25T10:00:00Z",
		},
		{
			input: NormalizeLocalDateTimeInput{
				LocalDateTime: "2026-04-25T03:00:00",
				TimeZone:      "America/Los_Angeles",
			},
			want: "2026-04-25T10:00:00Z",
		},
	}

	for _, tc := range cases {
		got, err := NormalizeLocalDateTimeToUTCISO(tc.input)
		if err != nil {
			t.Fatalf("NormalizeLocalDateTimeToUTCISO(%+v) returned error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Fatalf("NormalizeLocalDateTimeToUTCISO(%+v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestTimeNormalizationRejectsInvalidInput(t *testing.T) {
	if _, err := NormalizeUTCISODateTime("2026-04-25 10:00:00"); err == nil {
		t.Fatal("NormalizeUTCISODateTime accepted an invalid ISO 8601 timestamp")
	}
	if _, err := NormalizeLocalDateTimeToUTCISO(NormalizeLocalDateTimeInput{
		LocalDateTime: "2026-04-25T19:00:00",
		TimeZone:      "Not/AZone",
	}); err == nil {
		t.Fatal("NormalizeLocalDateTimeToUTCISO accepted an invalid timezone")
	}
}

func ptr[T any](value T) *T {
	return &value
}
