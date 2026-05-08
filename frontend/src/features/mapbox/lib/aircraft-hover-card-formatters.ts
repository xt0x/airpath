export function formatAircraftAltitude(value: number | null): string {
  return value === null ? "Unavailable" : `${value.toLocaleString("en-US")} ft`;
}

export function formatAircraftSpeed(value: number | null): string {
  return value === null ? "Unavailable" : `${value.toLocaleString("en-US")} kt`;
}

export { formatDisplayDateTime as formatAircraftTimestamp } from "@/lib/display-date-time";
