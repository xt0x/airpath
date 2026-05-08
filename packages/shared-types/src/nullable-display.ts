import type { Airport, ISODateTimeString } from "./domain-types.js";

export type MissingValueReason =
  | "not_acquired"
  | "not_announced"
  | "not_applicable"
  | "unavailable";

export type NullableDisplayKind = "available" | "missing";

export type NullableDisplayValue<T> =
  | {
      kind: "available";
      value: T;
    }
  | {
      kind: "missing";
      reason: MissingValueReason;
      label: MissingValueLabel;
    };

export type MissingValueLabel = (typeof missingValueLabels)[MissingValueReason];

export const missingValueLabels = {
  not_acquired: "Not acquired",
  not_announced: "Not announced",
  not_applicable: "Not applicable",
  unavailable: "Unavailable",
} as const;

export function toNullableDisplayValue<T>(
  value: T | null | undefined,
  reason: MissingValueReason,
): NullableDisplayValue<T> {
  if (value === null || value === undefined) {
    return {
      kind: "missing",
      reason,
      label: missingValueLabels[reason],
    };
  }

  return {
    kind: "available",
    value,
  };
}

export function nullableTextDisplayText(
  value: string | null | undefined,
  reason: MissingValueReason,
): string {
  const displayValue = toNullableDisplayValue(value, reason);
  return displayValue.kind === "available" ? displayValue.value : displayValue.label;
}

export function nullableDateTimeDisplayText(
  value: ISODateTimeString | null | undefined,
  reason: MissingValueReason,
): string {
  return nullableTextDisplayText(value, reason);
}

export function nullableProgressDisplayText(
  value: number | null | undefined,
  reason: MissingValueReason,
): string {
  const displayValue = toNullableDisplayValue(value, reason);
  return displayValue.kind === "available" ? `${displayValue.value}%` : displayValue.label;
}

export function nullableAirportDisplayText(
  airport: Airport | null | undefined,
  reason: MissingValueReason,
): string {
  const displayValue = toNullableDisplayValue(airport, reason);
  if (displayValue.kind === "missing") {
    return displayValue.label;
  }

  const { code, name } = displayValue.value;
  return name === null ? code : `${code} - ${name}`;
}
