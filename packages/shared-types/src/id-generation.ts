import type {
  AirportCode,
  FAFlightID,
  InternalFlightLegID,
  ISODateTimeString,
  ProvisionalFlightLegID,
} from "./domain-types.js";
import { hashStableParts } from "./stable-hash.js";

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
  return `sched_${hashStableParts([
    input.ident,
    input.originCode,
    input.destinationCode,
    input.scheduledOut,
  ])}`;
}

export function generateInternalFlightLegId(input: InternalFlightLegIDInput): InternalFlightLegID {
  return `iflg_${hashStableParts([
    input.faFlightId,
    input.originCode,
    input.destinationCode,
    input.scheduledOut,
    input.legIndex,
  ])}`;
}
