package display

import (
	"strconv"

	"airpath/services/internal/domain"
)

type NullableDisplayKind string

const (
	NullableDisplayKindAvailable NullableDisplayKind = "available"
	NullableDisplayKindMissing   NullableDisplayKind = "missing"
)

type NullableDisplayValue[T any] struct {
	Kind   NullableDisplayKind        `json:"kind"`
	Value  *T                         `json:"value,omitempty"`
	Reason *domain.MissingValueReason `json:"reason,omitempty"`
	Label  *string                    `json:"label,omitempty"`
}

func ToNullableDisplayValue[T any](value *T, reason domain.MissingValueReason) NullableDisplayValue[T] {
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

func MissingValueLabel(reason domain.MissingValueReason) string {
	switch reason {
	case domain.MissingValueReasonNotAcquired:
		return "未取得"
	case domain.MissingValueReasonNotAnnounced:
		return "未発表"
	case domain.MissingValueReasonNotApplicable:
		return "対象外"
	case domain.MissingValueReasonUnavailable:
		return "取得不可"
	default:
		return "未取得"
	}
}

func NullableTextDisplayText(value *string, reason domain.MissingValueReason) string {
	displayValue := ToNullableDisplayValue(value, reason)
	if displayValue.Kind == NullableDisplayKindMissing {
		return *displayValue.Label
	}

	return *displayValue.Value
}

func NullableDateTimeDisplayText(value *domain.ISODateTimeString, reason domain.MissingValueReason) string {
	displayValue := ToNullableDisplayValue(value, reason)
	if displayValue.Kind == NullableDisplayKindMissing {
		return *displayValue.Label
	}

	return string(*displayValue.Value)
}

func NullableProgressDisplayText(value *int, reason domain.MissingValueReason) string {
	displayValue := ToNullableDisplayValue(value, reason)
	if displayValue.Kind == NullableDisplayKindMissing {
		return *displayValue.Label
	}

	return strconv.Itoa(*displayValue.Value) + "%"
}

func NullableAirportDisplayText(airport *domain.Airport, reason domain.MissingValueReason) string {
	displayValue := ToNullableDisplayValue(airport, reason)
	if displayValue.Kind == NullableDisplayKindMissing {
		return *displayValue.Label
	}

	if displayValue.Value.Name == nil {
		return string(displayValue.Value.Code)
	}

	return string(displayValue.Value.Code) + " - " + *displayValue.Value.Name
}
