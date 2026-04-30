import type { FlightTimes, ISODateTimeString } from "./domain-types.js";
import { normalizeUtcIsoDateTime } from "./time-normalization.js";

export type FlightDurationKind = "actual" | "estimated" | "scheduled" | "filed" | "gate_actual";

export interface FlightDurationInput {
  times: FlightTimes;
  filedEteSeconds: number | null | undefined;
}

export interface FlightDuration {
  kind: FlightDurationKind;
  seconds: number;
  startAt: ISODateTimeString | null;
  endAt: ISODateTimeString | null;
}

export function calculateFlightDuration(input: FlightDurationInput): FlightDuration | null {
  return (
    durationFromPair("actual", input.times.actualOff, input.times.actualOn) ??
    durationFromPair("estimated", input.times.estimatedOff, input.times.estimatedOn) ??
    durationFromPair("scheduled", input.times.scheduledOff, input.times.scheduledOn) ??
    durationFromFiledEte(input.filedEteSeconds) ??
    durationFromPair("gate_actual", input.times.actualOut, input.times.actualIn)
  );
}

function durationFromPair(
  kind: Exclude<FlightDurationKind, "filed">,
  startAt: ISODateTimeString | null,
  endAt: ISODateTimeString | null,
): FlightDuration | null {
  if (startAt === null || endAt === null) {
    return null;
  }

  const normalizedStartAt = normalizeUtcIsoDateTime(startAt);
  const normalizedEndAt = normalizeUtcIsoDateTime(endAt);
  const seconds = Math.floor((Date.parse(normalizedEndAt) - Date.parse(normalizedStartAt)) / 1000);
  if (seconds <= 0) {
    return null;
  }

  return {
    kind,
    seconds,
    startAt: normalizedStartAt,
    endAt: normalizedEndAt,
  };
}

function durationFromFiledEte(filedEteSeconds: number | null | undefined): FlightDuration | null {
  if (filedEteSeconds === null || filedEteSeconds === undefined || filedEteSeconds <= 0) {
    return null;
  }

  return {
    kind: "filed",
    seconds: filedEteSeconds,
    startAt: null,
    endAt: null,
  };
}
