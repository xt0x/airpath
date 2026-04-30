import type { ISODateTimeString } from "./domain-types.js";

export interface NormalizeLocalDateTimeInput {
  localDateTime: string;
  timeZone: string;
}

const utcIsoDateTimePattern =
  /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/;
const localDateTimePattern = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/;

export function normalizeUtcIsoDateTime(value: string): ISODateTimeString {
  if (!utcIsoDateTimePattern.test(value)) {
    throw new Error("Invalid ISO 8601 date-time with timezone");
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    throw new Error("Invalid ISO 8601 date-time with timezone");
  }

  return formatUtcDate(parsed.getTime());
}

export function normalizeLocalDateTimeToUtcIso(
  input: NormalizeLocalDateTimeInput,
): ISODateTimeString {
  assertValidTimeZone(input.timeZone);
  const parts = parseLocalDateTime(input.localDateTime);
  const localAsUtc = Date.UTC(
    parts.year,
    parts.month - 1,
    parts.day,
    parts.hour,
    parts.minute,
    parts.second,
  );
  const firstOffset = timeZoneOffsetMilliseconds(input.timeZone, localAsUtc);
  const firstUtc = localAsUtc - firstOffset;
  const refinedOffset = timeZoneOffsetMilliseconds(input.timeZone, firstUtc);

  return formatUtcDate(localAsUtc - refinedOffset);
}

function parseLocalDateTime(value: string): {
  year: number;
  month: number;
  day: number;
  hour: number;
  minute: number;
  second: number;
} {
  const match = localDateTimePattern.exec(value);
  if (match === null) {
    throw new Error("Invalid local ISO 8601 date-time");
  }

  const [, year, month, day, hour, minute, second = "00"] = match;
  return {
    year: Number(year),
    month: Number(month),
    day: Number(day),
    hour: Number(hour),
    minute: Number(minute),
    second: Number(second),
  };
}

function assertValidTimeZone(timeZone: string): void {
  try {
    new Intl.DateTimeFormat("en-US", { timeZone }).format(new Date(0));
  } catch {
    throw new Error("Invalid IANA timezone");
  }
}

function timeZoneOffsetMilliseconds(timeZone: string, utcMilliseconds: number): number {
  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }).formatToParts(new Date(utcMilliseconds));
  const values = Object.fromEntries(parts.map((part) => [part.type, part.value]));
  const hour = values["hour"] === "24" ? 0 : Number(values["hour"]);
  const localAsUtc = Date.UTC(
    Number(values["year"]),
    Number(values["month"]) - 1,
    Number(values["day"]),
    hour,
    Number(values["minute"]),
    Number(values["second"]),
  );

  return localAsUtc - utcMilliseconds;
}

function formatUtcDate(milliseconds: number): ISODateTimeString {
  return new Date(milliseconds).toISOString().replace(/\.\d{3}Z$/, "Z");
}
