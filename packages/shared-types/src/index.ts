export type ISODateTimeString = string;
export type EpochSeconds = number;
export type FlightID = string;
export type InternalFlightLegID = string;
export type ProvisionalFlightLegID = string;
export type FAFlightID = string;
export type UserID = string;
export type ConnectionID = string;
export type AirportCode = string;

export type FlightIDType = "internal" | "provisional";

export type FlightPollState =
  | "scheduled"
  | "preparing"
  | "active"
  | "arrived"
  | "cancelled"
  | "diverted"
  | "completed"
  | "error";

export type PositionSource = "flightaware_position" | "flightaware_track" | "alert" | "manual";

export type PositionUpdateType = "actual" | "estimated" | "predicted" | "surface";

export type AltitudeChange = "climbing" | "descending" | "level";

export type FlightEventType =
  | "departure"
  | "off"
  | "on"
  | "arrival"
  | "cancelled"
  | "diverted"
  | "departure_delay"
  | "arrival_delay"
  | "status_updated";

export type FlightEventSource = "polling" | "alert";

export type WatchState = "active" | "paused" | "removed";

export type SubscriptionType = "session";

export interface Airport {
  code: AirportCode;
  name: string | null;
  timezone: string | null;
}

export interface FlightTimes {
  scheduledOut: ISODateTimeString | null;
  estimatedOut: ISODateTimeString | null;
  actualOut: ISODateTimeString | null;
  scheduledOff: ISODateTimeString | null;
  estimatedOff: ISODateTimeString | null;
  actualOff: ISODateTimeString | null;
  scheduledOn: ISODateTimeString | null;
  estimatedOn: ISODateTimeString | null;
  actualOn: ISODateTimeString | null;
  scheduledIn: ISODateTimeString | null;
  estimatedIn: ISODateTimeString | null;
  actualIn: ISODateTimeString | null;
}

export interface Flight {
  flightId: FlightID;
  flightIdType: FlightIDType;
  internalFlightLegId: InternalFlightLegID | null;
  provisionalFlightLegId: ProvisionalFlightLegID | null;
  faFlightId: FAFlightID | null;
  ident: string;
  identIata: string | null;
  operator: string | null;
  aircraftType: string | null;
  registration: string | null;
  origin: Airport;
  destination: Airport;
  originalDestination: Airport | null;
  diverted: boolean;
  legIndex: number | null;
  status: string;
  progressPercent: number | null;
  times: FlightTimes;
  filedEteSeconds: number | null;
  plannedRouteS3Key: string | null;
  actualTrackS3Key: string | null;
  latestPositionTimestamp: ISODateTimeString | null;
  latestPositionSource: PositionSource | null;
  pollState: FlightPollState | null;
  watcherCount: number;
  sessionSubscriberCount: number;
  persistentWatcherCount: number;
  fetchLeaseUntil: ISODateTimeString | null;
  fetchOwner: string | null;
  nextSummaryPollAt: ISODateTimeString | null;
  nextPositionPollAt: ISODateTimeString | null;
  nextTrackPollAt: ISODateTimeString | null;
  nextRoutePollAt: ISODateTimeString | null;
  idleSince: ISODateTimeString | null;
  updatedAt: ISODateTimeString;
  ttl: EpochSeconds | null;
}

export interface FlightPosition {
  flightId: FlightID;
  internalFlightLegId: InternalFlightLegID | null;
  provisionalFlightLegId: ProvisionalFlightLegID | null;
  faFlightId: FAFlightID | null;
  latitude: number;
  longitude: number;
  altitudeHundredsFeet: number | null;
  altitudeFeet: number | null;
  altitudeChange: AltitudeChange | null;
  groundspeedKnots: number | null;
  headingDegrees: number | null;
  timestamp: ISODateTimeString;
  updateType: PositionUpdateType | null;
  source: PositionSource;
  ttl: EpochSeconds | null;
}

export interface FlightEvent {
  flightId: FlightID;
  internalFlightLegId: InternalFlightLegID | null;
  provisionalFlightLegId: ProvisionalFlightLegID | null;
  faFlightId: FAFlightID | null;
  dedupeKey: string;
  eventType: FlightEventType;
  eventTimestamp: ISODateTimeString;
  payload: Record<string, unknown>;
  source: FlightEventSource;
  appliedToFlightState: boolean;
  createdAt: ISODateTimeString;
}

export interface UserWatch {
  userId: UserID;
  flightId: FlightID;
  faFlightId: FAFlightID | null;
  flightIdType: FlightIDType;
  watchState: WatchState;
  createdAt: ISODateTimeString;
  updatedAt: ISODateTimeString;
  ttl: EpochSeconds | null;
}

export interface FlightSubscription {
  flightId: FlightID;
  connectionId: ConnectionID;
  userId: UserID;
  subscriptionType: SubscriptionType;
  createdAt: ISODateTimeString;
  lastSeenAt: ISODateTimeString;
  ttl: EpochSeconds | null;
}
