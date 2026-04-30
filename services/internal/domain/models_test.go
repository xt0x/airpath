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
		{
			input: NormalizeLocalDateTimeInput{
				LocalDateTime: "2026-04-25T19:00",
				TimeZone:      "Asia/Tokyo",
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

func TestNormalizeAltitudeFeet(t *testing.T) {
	if got := NormalizeAltitudeFeet(ptr(370)); got == nil || *got != 37000 {
		t.Fatalf("NormalizeAltitudeFeet(370) = %v, want 37000", got)
	}
	if got := NormalizeAltitudeFeet(ptr(0)); got == nil || *got != 0 {
		t.Fatalf("NormalizeAltitudeFeet(0) = %v, want 0", got)
	}
	if got := NormalizeAltitudeFeet(nil); got != nil {
		t.Fatalf("NormalizeAltitudeFeet(nil) = %v, want nil", got)
	}
}

func TestNormalizeSpeedAndHeadingKeepMissingValues(t *testing.T) {
	if got := NormalizeGroundspeedKnots(ptr(488)); got == nil || *got != 488 {
		t.Fatalf("NormalizeGroundspeedKnots(488) = %v, want 488", got)
	}
	if got := NormalizeGroundspeedKnots(nil); got != nil {
		t.Fatalf("NormalizeGroundspeedKnots(nil) = %v, want nil", got)
	}
	if got := NormalizeHeadingDegrees(ptr(275)); got == nil || *got != 275 {
		t.Fatalf("NormalizeHeadingDegrees(275) = %v, want 275", got)
	}
	if got := NormalizeHeadingDegrees(nil); got != nil {
		t.Fatalf("NormalizeHeadingDegrees(nil) = %v, want nil", got)
	}
}

func TestNormalizeHeadingDegreesTreats360AsZero(t *testing.T) {
	if got := NormalizeHeadingDegrees(ptr(0)); got == nil || *got != 0 {
		t.Fatalf("NormalizeHeadingDegrees(0) = %v, want 0", got)
	}
	if got := NormalizeHeadingDegrees(ptr(360)); got == nil || *got != 0 {
		t.Fatalf("NormalizeHeadingDegrees(360) = %v, want 0", got)
	}
}

func TestNormalizeFlightPositionMetrics(t *testing.T) {
	converted := NormalizeFlightPositionMetrics(FlightPositionMetricsInput{
		AltitudeHundredsFeet: ptr(370),
		GroundspeedKnots:     ptr(488),
		HeadingDegrees:       ptr(360),
	})
	if converted.AltitudeHundredsFeet == nil || *converted.AltitudeHundredsFeet != 370 {
		t.Fatalf("AltitudeHundredsFeet = %v, want 370", converted.AltitudeHundredsFeet)
	}
	if converted.AltitudeFeet == nil || *converted.AltitudeFeet != 37000 {
		t.Fatalf("AltitudeFeet = %v, want 37000", converted.AltitudeFeet)
	}
	if converted.GroundspeedKnots == nil || *converted.GroundspeedKnots != 488 {
		t.Fatalf("GroundspeedKnots = %v, want 488", converted.GroundspeedKnots)
	}
	if converted.HeadingDegrees == nil || *converted.HeadingDegrees != 0 {
		t.Fatalf("HeadingDegrees = %v, want 0", converted.HeadingDegrees)
	}

	missing := NormalizeFlightPositionMetrics(FlightPositionMetricsInput{})
	if missing.AltitudeHundredsFeet != nil {
		t.Fatalf("missing AltitudeHundredsFeet = %v, want nil", missing.AltitudeHundredsFeet)
	}
	if missing.AltitudeFeet != nil {
		t.Fatalf("missing AltitudeFeet = %v, want nil", missing.AltitudeFeet)
	}
	if missing.GroundspeedKnots != nil {
		t.Fatalf("missing GroundspeedKnots = %v, want nil", missing.GroundspeedKnots)
	}
	if missing.HeadingDegrees != nil {
		t.Fatalf("missing HeadingDegrees = %v, want nil", missing.HeadingDegrees)
	}
}

func TestCalculateFlightDurationUsesActualRunwayTimesFirst(t *testing.T) {
	got := CalculateFlightDuration(FlightDurationInput{
		Times:           baseFlightDurationTimes(),
		FiledEteSeconds: ptr(42000),
	})
	if got == nil {
		t.Fatal("CalculateFlightDuration() = nil, want actual duration")
	}
	assertFlightDuration(t, *got, FlightDurationKindActual, 40920, ptr(ISODateTimeString("2026-04-25T10:08:00Z")), ptr(ISODateTimeString("2026-04-25T21:30:00Z")))
}

func TestCalculateFlightDurationFallsBackByPriority(t *testing.T) {
	times := baseFlightDurationTimes()
	times.ActualOff = nil
	times.ActualOn = nil
	got := CalculateFlightDuration(FlightDurationInput{
		Times:           times,
		FiledEteSeconds: ptr(42000),
	})
	if got == nil {
		t.Fatal("estimated duration = nil")
	}
	assertFlightDuration(t, *got, FlightDurationKindEstimated, 41400, ptr(ISODateTimeString("2026-04-25T10:05:00Z")), ptr(ISODateTimeString("2026-04-25T21:35:00Z")))

	times.EstimatedOff = nil
	times.EstimatedOn = nil
	got = CalculateFlightDuration(FlightDurationInput{
		Times:           times,
		FiledEteSeconds: ptr(42000),
	})
	if got == nil {
		t.Fatal("scheduled duration = nil")
	}
	assertFlightDuration(t, *got, FlightDurationKindScheduled, 42000, ptr(ISODateTimeString("2026-04-25T10:00:00Z")), ptr(ISODateTimeString("2026-04-25T21:40:00Z")))

	times.ScheduledOff = nil
	times.ScheduledOn = nil
	got = CalculateFlightDuration(FlightDurationInput{
		Times:           times,
		FiledEteSeconds: ptr(42000),
	})
	if got == nil {
		t.Fatal("filed duration = nil")
	}
	assertFlightDuration(t, *got, FlightDurationKindFiled, 42000, nil, nil)

	got = CalculateFlightDuration(FlightDurationInput{
		Times:           times,
		FiledEteSeconds: nil,
	})
	if got == nil {
		t.Fatal("gate duration = nil")
	}
	assertFlightDuration(t, *got, FlightDurationKindGateActual, 42480, ptr(ISODateTimeString("2026-04-25T10:02:00Z")), ptr(ISODateTimeString("2026-04-25T21:50:00Z")))
}

func TestCalculateFlightDurationReturnsNilWithoutCompleteSource(t *testing.T) {
	got := CalculateFlightDuration(FlightDurationInput{
		Times:           FlightTimes{},
		FiledEteSeconds: nil,
	})
	if got != nil {
		t.Fatalf("CalculateFlightDuration() = %+v, want nil", got)
	}
}

func TestCalculateFlightDurationSkipsNonPositiveTimestampPairs(t *testing.T) {
	times := baseFlightDurationTimes()
	times.ActualOff = ptr(ISODateTimeString("2026-04-25T21:30:00Z"))
	times.ActualOn = ptr(ISODateTimeString("2026-04-25T10:08:00Z"))
	times.EstimatedOff = ptr(ISODateTimeString("2026-04-25T10:05:00Z"))
	times.EstimatedOn = ptr(ISODateTimeString("2026-04-25T10:05:00Z"))

	got := CalculateFlightDuration(FlightDurationInput{
		Times:           times,
		FiledEteSeconds: ptr(42000),
	})
	if got == nil {
		t.Fatal("CalculateFlightDuration() = nil, want scheduled duration")
	}
	assertFlightDuration(t, *got, FlightDurationKindScheduled, 42000, ptr(ISODateTimeString("2026-04-25T10:00:00Z")), ptr(ISODateTimeString("2026-04-25T21:40:00Z")))
}

func TestCalculateFlightDurationSkipsNonPositiveFiledEte(t *testing.T) {
	times := baseFlightDurationTimes()
	times.ActualOff = nil
	times.ActualOn = nil
	times.EstimatedOff = nil
	times.EstimatedOn = nil
	times.ScheduledOff = nil
	times.ScheduledOn = nil

	got := CalculateFlightDuration(FlightDurationInput{
		Times:           times,
		FiledEteSeconds: ptr(0),
	})
	if got == nil {
		t.Fatal("CalculateFlightDuration() = nil, want gate duration")
	}
	assertFlightDuration(t, *got, FlightDurationKindGateActual, 42480, ptr(ISODateTimeString("2026-04-25T10:02:00Z")), ptr(ISODateTimeString("2026-04-25T21:50:00Z")))

	got = CalculateFlightDuration(FlightDurationInput{
		Times:           times,
		FiledEteSeconds: ptr(-1),
	})
	if got == nil {
		t.Fatal("CalculateFlightDuration() = nil, want gate duration")
	}
	assertFlightDuration(t, *got, FlightDurationKindGateActual, 42480, ptr(ISODateTimeString("2026-04-25T10:02:00Z")), ptr(ISODateTimeString("2026-04-25T21:50:00Z")))
}

func TestGenerateFlightEventDedupeKeyUsesProvisionalFlightLegID(t *testing.T) {
	got := GenerateFlightEventDedupeKey(FlightEventDedupeKeyInput{
		EventType:              FlightEventTypeDeparture,
		FAFlightID:             nil,
		ProvisionalFlightLegID: ptr(ProvisionalFlightLegID("sched_3f5a7c8d91ab")),
		EventTimestamp:         "2026-04-25T10:08:00Z",
	})
	if got != "13b84e62b4d0" {
		t.Fatalf("GenerateFlightEventDedupeKey(provisional) = %q, want %q", got, "13b84e62b4d0")
	}
}

func TestGenerateFlightEventDedupeKeyForPolling(t *testing.T) {
	got := GenerateFlightEventDedupeKey(FlightEventDedupeKeyInput{
		EventType:              FlightEventTypeArrival,
		FAFlightID:             ptr(FAFlightID("iflg_8d2a2a3a6c4f")),
		ProvisionalFlightLegID: nil,
		EventTimestamp:         "2026-04-25T21:50:00Z",
	})
	if got != "b22238bbb5ee" {
		t.Fatalf("GenerateFlightEventDedupeKey(polling) = %q, want %q", got, "b22238bbb5ee")
	}
}

func TestGenerateFlightEventDedupeKeyRequiresFlightKey(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("GenerateFlightEventDedupeKey() did not panic without a flight key")
		}
	}()

	GenerateFlightEventDedupeKey(FlightEventDedupeKeyInput{
		EventType:              FlightEventTypeStatusUpdated,
		FAFlightID:             nil,
		ProvisionalFlightLegID: nil,
		EventTimestamp:         "2026-04-25T21:50:00Z",
	})
}

func baseFlightDurationTimes() FlightTimes {
	return FlightTimes{
		ScheduledOut: ptr(ISODateTimeString("2026-04-25T09:30:00Z")),
		ActualOut:    ptr(ISODateTimeString("2026-04-25T10:02:00Z")),
		ScheduledOff: ptr(ISODateTimeString("2026-04-25T10:00:00Z")),
		EstimatedOff: ptr(ISODateTimeString("2026-04-25T10:05:00Z")),
		ActualOff:    ptr(ISODateTimeString("2026-04-25T10:08:00Z")),
		ScheduledOn:  ptr(ISODateTimeString("2026-04-25T21:40:00Z")),
		EstimatedOn:  ptr(ISODateTimeString("2026-04-25T21:35:00Z")),
		ActualOn:     ptr(ISODateTimeString("2026-04-25T21:30:00Z")),
		ActualIn:     ptr(ISODateTimeString("2026-04-25T21:50:00Z")),
	}
}

func assertFlightDuration(t *testing.T, got FlightDuration, kind FlightDurationKind, seconds int, startAt *ISODateTimeString, endAt *ISODateTimeString) {
	t.Helper()
	if got.Kind != kind {
		t.Fatalf("Kind = %q, want %q", got.Kind, kind)
	}
	if got.Seconds != seconds {
		t.Fatalf("Seconds = %d, want %d", got.Seconds, seconds)
	}
	if !equalISODateTimePointers(got.StartAt, startAt) {
		t.Fatalf("StartAt = %v, want %v", got.StartAt, startAt)
	}
	if !equalISODateTimePointers(got.EndAt, endAt) {
		t.Fatalf("EndAt = %v, want %v", got.EndAt, endAt)
	}
}

func equalISODateTimePointers(left *ISODateTimeString, right *ISODateTimeString) bool {
	if left == nil || right == nil {
		return left == right
	}

	return *left == *right
}

func ptr[T any](value T) *T {
	return &value
}
