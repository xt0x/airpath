package domain

import "strconv"

type MissingValueReason string

const (
	MissingValueReasonNotAcquired   MissingValueReason = "not_acquired"
	MissingValueReasonNotAnnounced  MissingValueReason = "not_announced"
	MissingValueReasonNotApplicable MissingValueReason = "not_applicable"
	MissingValueReasonUnavailable   MissingValueReason = "unavailable"
)

type NullableDisplayKind string

const (
	NullableDisplayKindAvailable NullableDisplayKind = "available"
	NullableDisplayKindMissing   NullableDisplayKind = "missing"
)

type NullableDisplayValue[T any] struct {
	Kind   NullableDisplayKind `json:"kind"`
	Value  *T                  `json:"value,omitempty"`
	Reason *MissingValueReason `json:"reason,omitempty"`
	Label  *string             `json:"label,omitempty"`
}

func ToNullableDisplayValue[T any](value *T, reason MissingValueReason) NullableDisplayValue[T] {
	if value == nil {
		label := MissingValueLabel(reason)
		return NullableDisplayValue[T]{
			Kind:   NullableDisplayKindMissing,
			Reason: &reason,
			Label:  &label,
		}
	}

	return NullableDisplayValue[T]{
		Kind:  NullableDisplayKindAvailable,
		Value: value,
	}
}

func MissingValueLabel(reason MissingValueReason) string {
	switch reason {
	case MissingValueReasonNotAcquired:
		return "未取得"
	case MissingValueReasonNotAnnounced:
		return "未発表"
	case MissingValueReasonNotApplicable:
		return "対象外"
	case MissingValueReasonUnavailable:
		return "取得不可"
	default:
		return "未取得"
	}
}

func NullableTextDisplayText(value *string, reason MissingValueReason) string {
	displayValue := ToNullableDisplayValue(value, reason)
	if displayValue.Kind == NullableDisplayKindMissing {
		return *displayValue.Label
	}

	return *displayValue.Value
}

func NullableDateTimeDisplayText(value *ISODateTimeString, reason MissingValueReason) string {
	displayValue := ToNullableDisplayValue(value, reason)
	if displayValue.Kind == NullableDisplayKindMissing {
		return *displayValue.Label
	}

	return string(*displayValue.Value)
}

func NullableProgressDisplayText(value *int, reason MissingValueReason) string {
	displayValue := ToNullableDisplayValue(value, reason)
	if displayValue.Kind == NullableDisplayKindMissing {
		return *displayValue.Label
	}

	return strconv.Itoa(*displayValue.Value) + "%"
}

func NullableAirportDisplayText(airport *Airport, reason MissingValueReason) string {
	displayValue := ToNullableDisplayValue(airport, reason)
	if displayValue.Kind == NullableDisplayKindMissing {
		return *displayValue.Label
	}

	if displayValue.Value.Name == nil {
		return string(displayValue.Value.Code)
	}

	return string(displayValue.Value.Code) + " - " + *displayValue.Value.Name
}
