import type {
  AirportCode,
  FAFlightID,
  InternalFlightLegID,
  ISODateTimeString,
  ProvisionalFlightLegID,
} from "./index.js";

export interface ProvisionalFlightLegIDInput {
  ident: string;
  originCode: AirportCode;
  destinationCode: AirportCode;
  scheduledOut: ISODateTimeString;
}

export interface InternalFlightLegIDInput {
  faFlightId: FAFlightID;
  originCode: AirportCode;
  destinationCode: AirportCode;
  scheduledOut: ISODateTimeString;
  legIndex: number;
}

export function generateProvisionalFlightLegId(
  input: ProvisionalFlightLegIDInput,
): ProvisionalFlightLegID {
  return `sched_${hashFlightIDParts([
    input.ident,
    input.originCode,
    input.destinationCode,
    input.scheduledOut,
  ])}`;
}

export function generateInternalFlightLegId(input: InternalFlightLegIDInput): InternalFlightLegID {
  return `iflg_${hashFlightIDParts([
    input.faFlightId,
    input.originCode,
    input.destinationCode,
    input.scheduledOut,
    input.legIndex,
  ])}`;
}

function hashFlightIDParts(parts: readonly (string | number)[]): string {
  let hash = 0xcbf29ce484222325n;
  const prime = 0x100000001b3n;
  const mask = 0xffffffffffffffffn;
  const bytes = new TextEncoder().encode(parts.map(String).join("\u001f"));

  for (const byte of bytes) {
    hash ^= BigInt(byte);
    hash = (hash * prime) & mask;
  }

  return hash.toString(16).padStart(16, "0").slice(0, 12);
}
