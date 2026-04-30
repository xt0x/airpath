package domain

import (
	"encoding/json"
	"os"
	"testing"
)

type domainHelperGolden struct {
	IDGeneration struct {
		Provisional struct {
			Input struct {
				Ident           string            `json:"ident"`
				OriginCode      AirportCode       `json:"originCode"`
				DestinationCode AirportCode       `json:"destinationCode"`
				ScheduledOut    ISODateTimeString `json:"scheduledOut"`
			} `json:"input"`
			Expected ProvisionalFlightLegID `json:"expected"`
		} `json:"provisional"`
		Internal struct {
			Input struct {
				FAFlightID      FAFlightID        `json:"faFlightId"`
				OriginCode      AirportCode       `json:"originCode"`
				DestinationCode AirportCode       `json:"destinationCode"`
				ScheduledOut    ISODateTimeString `json:"scheduledOut"`
				LegIndex        int               `json:"legIndex"`
			} `json:"input"`
			Expected InternalFlightLegID `json:"expected"`
		} `json:"internal"`
	} `json:"idGeneration"`
	TimeNormalization struct {
		UTCInput    string `json:"utcInput"`
		LocalInput  NormalizeLocalDateTimeInput
		ExpectedUTC ISODateTimeString `json:"expectedUtc"`
	} `json:"timeNormalization"`
	NullableDisplay struct {
		MissingReason        MissingValueReason `json:"missingReason"`
		ExpectedMissingLabel string             `json:"expectedMissingLabel"`
		Progress             int                `json:"progress"`
		ExpectedProgress     string             `json:"expectedProgress"`
	} `json:"nullableDisplay"`
	PositionMetrics struct {
		Input struct {
			AltitudeHundredsFeet int `json:"altitudeHundredsFeet"`
			GroundspeedKnots     int `json:"groundspeedKnots"`
			HeadingDegrees       int `json:"headingDegrees"`
		} `json:"input"`
		Expected struct {
			AltitudeHundredsFeet int `json:"altitudeHundredsFeet"`
			AltitudeFeet         int `json:"altitudeFeet"`
			GroundspeedKnots     int `json:"groundspeedKnots"`
			HeadingDegrees       int `json:"headingDegrees"`
		} `json:"expected"`
	} `json:"positionMetrics"`
}

func TestDomainHelperGoldenFixture(t *testing.T) {
	body, err := os.ReadFile("../../../packages/shared-types/fixtures/domain/domain-helper-golden.json")
	if err != nil {
		t.Fatalf("read golden fixture: %v", err)
	}
	var golden domainHelperGolden
	if err := json.Unmarshal(body, &golden); err != nil {
		t.Fatalf("unmarshal golden fixture: %v", err)
	}

	provisional := GenerateProvisionalFlightLegID(ProvisionalFlightLegIDInput{
		Ident:           golden.IDGeneration.Provisional.Input.Ident,
		OriginCode:      golden.IDGeneration.Provisional.Input.OriginCode,
		DestinationCode: golden.IDGeneration.Provisional.Input.DestinationCode,
		ScheduledOut:    golden.IDGeneration.Provisional.Input.ScheduledOut,
	})
	if provisional != golden.IDGeneration.Provisional.Expected {
		t.Fatalf("GenerateProvisionalFlightLegID() = %q, want %q", provisional, golden.IDGeneration.Provisional.Expected)
	}

	internal := GenerateInternalFlightLegID(InternalFlightLegIDInput{
		FAFlightID:      golden.IDGeneration.Internal.Input.FAFlightID,
		OriginCode:      golden.IDGeneration.Internal.Input.OriginCode,
		DestinationCode: golden.IDGeneration.Internal.Input.DestinationCode,
		ScheduledOut:    golden.IDGeneration.Internal.Input.ScheduledOut,
		LegIndex:        golden.IDGeneration.Internal.Input.LegIndex,
	})
	if internal != golden.IDGeneration.Internal.Expected {
		t.Fatalf("GenerateInternalFlightLegID() = %q, want %q", internal, golden.IDGeneration.Internal.Expected)
	}

	utc, err := NormalizeUTCISODateTime(golden.TimeNormalization.UTCInput)
	if err != nil {
		t.Fatalf("NormalizeUTCISODateTime() error = %v", err)
	}
	if utc != golden.TimeNormalization.ExpectedUTC {
		t.Fatalf("NormalizeUTCISODateTime() = %q, want %q", utc, golden.TimeNormalization.ExpectedUTC)
	}
	local, err := NormalizeLocalDateTimeToUTCISO(golden.TimeNormalization.LocalInput)
	if err != nil {
		t.Fatalf("NormalizeLocalDateTimeToUTCISO() error = %v", err)
	}
	if local != golden.TimeNormalization.ExpectedUTC {
		t.Fatalf("NormalizeLocalDateTimeToUTCISO() = %q, want %q", local, golden.TimeNormalization.ExpectedUTC)
	}

	metrics := NormalizeFlightPositionMetrics(FlightPositionMetricsInput{
		AltitudeHundredsFeet: &golden.PositionMetrics.Input.AltitudeHundredsFeet,
		GroundspeedKnots:     &golden.PositionMetrics.Input.GroundspeedKnots,
		HeadingDegrees:       &golden.PositionMetrics.Input.HeadingDegrees,
	})
	if metrics.AltitudeHundredsFeet == nil || *metrics.AltitudeHundredsFeet != golden.PositionMetrics.Expected.AltitudeHundredsFeet ||
		metrics.AltitudeFeet == nil || *metrics.AltitudeFeet != golden.PositionMetrics.Expected.AltitudeFeet ||
		metrics.GroundspeedKnots == nil || *metrics.GroundspeedKnots != golden.PositionMetrics.Expected.GroundspeedKnots ||
		metrics.HeadingDegrees == nil || *metrics.HeadingDegrees != golden.PositionMetrics.Expected.HeadingDegrees {
		t.Fatalf("NormalizeFlightPositionMetrics() = %#v, want %#v", metrics, golden.PositionMetrics.Expected)
	}
}
