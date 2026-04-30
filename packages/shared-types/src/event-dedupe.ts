import type {
  FAFlightID,
  FlightEventType,
  ISODateTimeString,
  ProvisionalFlightLegID,
} from "./domain-types.js";
import { hashStableParts } from "./stable-hash.js";

export interface FlightEventDedupeKeyInput {
  eventType: FlightEventType;
  faFlightId: FAFlightID | null;
  provisionalFlightLegId: ProvisionalFlightLegID | null;
  eventTimestamp: ISODateTimeString;
}

export function generateFlightEventDedupeKey(input: FlightEventDedupeKeyInput): string {
  return hashStableParts([
    "polling",
    input.eventType,
    requireFlightKey(input.faFlightId, input.provisionalFlightLegId),
    input.eventTimestamp,
  ]);
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
