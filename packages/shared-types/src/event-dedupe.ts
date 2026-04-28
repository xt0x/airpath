import type {
  FAFlightID,
  FlightEventSource,
  FlightEventType,
  ISODateTimeString,
  ProvisionalFlightLegID,
} from "./index.js";

export interface FlightEventDedupeKeyInput {
  source: FlightEventSource;
  alertId?: string | null;
  eventType: FlightEventType;
  faFlightId: FAFlightID | null;
  provisionalFlightLegId: ProvisionalFlightLegID | null;
  eventTimestamp: ISODateTimeString;
}

export function generateFlightEventDedupeKey(input: FlightEventDedupeKeyInput): string {
  return hashDedupeKeyParts([
    input.source,
    input.source === "alert" ? requireAlertId(input.alertId) : "polling",
    input.eventType,
    requireFlightKey(input.faFlightId, input.provisionalFlightLegId),
    input.eventTimestamp,
  ]);
}

function requireAlertId(alertId: string | null | undefined): string {
  if (alertId === null || alertId === undefined || alertId.length === 0) {
    throw new Error("alertId is required for alert event dedupe keys");
  }

  return alertId;
}

function requireFlightKey(
  faFlightId: FAFlightID | null,
  provisionalFlightLegId: ProvisionalFlightLegID | null,
): string {
  const flightKey = faFlightId ?? provisionalFlightLegId;
  if (flightKey === null) {
    throw new Error("faFlightId or provisionalFlightLegId is required for event dedupe keys");
  }

  return flightKey;
}

function hashDedupeKeyParts(parts: readonly string[]): string {
  let hash = 0xcbf29ce484222325n;
  const prime = 0x100000001b3n;
  const mask = 0xffffffffffffffffn;
  const bytes = new TextEncoder().encode(parts.join("\u001f"));

  for (const byte of bytes) {
    hash ^= BigInt(byte);
    hash = (hash * prime) & mask;
  }

  return hash.toString(16).padStart(16, "0").slice(0, 12);
}
