package display

import (
	"testing"

	"airpath/services/internal/domain"
)

func TestNullableDisplayTextKeepsMissingValuesExplicit(t *testing.T) {
	if got := NullableDateTimeDisplayText(nil, domain.MissingValueReasonNotAnnounced); got != "未発表" {
		t.Fatalf("NullableDateTimeDisplayText(nil) = %q, want %q", got, "未発表")
	}
	if got := NullableTextDisplayText(nil, domain.MissingValueReasonNotAcquired); got != "未取得" {
		t.Fatalf("NullableTextDisplayText(nil) = %q, want %q", got, "未取得")
	}
	if got := NullableTextDisplayText(nil, domain.MissingValueReasonUnavailable); got != "取得不可" {
		t.Fatalf("NullableTextDisplayText(nil unavailable) = %q, want %q", got, "取得不可")
	}
	if got := NullableTextDisplayText(ptr("B789"), domain.MissingValueReasonNotAcquired); got != "B789" {
		t.Fatalf("NullableTextDisplayText(B789) = %q, want %q", got, "B789")
	}
}

func TestNullableProgressDisplayTextDoesNotTreatZeroAsMissing(t *testing.T) {
	if got := NullableProgressDisplayText(nil, domain.MissingValueReasonNotAcquired); got != "未取得" {
		t.Fatalf("NullableProgressDisplayText(nil) = %q, want %q", got, "未取得")
	}
	if got := NullableProgressDisplayText(ptr(0), domain.MissingValueReasonNotAcquired); got != "0%" {
		t.Fatalf("NullableProgressDisplayText(0) = %q, want %q", got, "0%%")
	}
}

func TestNullableAirportDisplayTextKeepsKnownCodes(t *testing.T) {
	airport := domain.Airport{Code: "RJTT"}
	if got := NullableAirportDisplayText(nil, domain.MissingValueReasonNotAcquired); got != "未取得" {
		t.Fatalf("NullableAirportDisplayText(nil) = %q, want %q", got, "未取得")
	}
	if got := NullableAirportDisplayText(&airport, domain.MissingValueReasonNotAcquired); got != "RJTT" {
		t.Fatalf("NullableAirportDisplayText(code only) = %q, want %q", got, "RJTT")
	}
	airport.Name = ptr("Tokyo Haneda")
	if got := NullableAirportDisplayText(&airport, domain.MissingValueReasonNotAcquired); got != "RJTT - Tokyo Haneda" {
		t.Fatalf("NullableAirportDisplayText(with name) = %q, want %q", got, "RJTT - Tokyo Haneda")
	}
}

func TestNullableDisplayValueCarriesStructuredMissingMetadata(t *testing.T) {
	missing := ToNullableDisplayValue[string](nil, domain.MissingValueReasonNotApplicable)
	if missing.Kind != NullableDisplayKindMissing {
		t.Fatalf("missing.Kind = %q, want %q", missing.Kind, NullableDisplayKindMissing)
	}
	if missing.Reason == nil || *missing.Reason != domain.MissingValueReasonNotApplicable {
		t.Fatalf("missing.Reason = %v, want %q", missing.Reason, domain.MissingValueReasonNotApplicable)
	}
	if missing.Label == nil || *missing.Label != "対象外" {
		t.Fatalf("missing.Label = %v, want %q", missing.Label, "対象外")
	}

	value := "2026-04-25T10:00:00Z"
	available := ToNullableDisplayValue(&value, domain.MissingValueReasonNotAnnounced)
	if available.Kind != NullableDisplayKindAvailable {
		t.Fatalf("available.Kind = %q, want %q", available.Kind, NullableDisplayKindAvailable)
	}
	if available.Value == nil || *available.Value != value {
		t.Fatalf("available.Value = %v, want %q", available.Value, value)
	}
}

func ptr[T any](value T) *T {
	return &value
}
