import type {
  AltitudeChange,
  Airport as DomainAirport,
  FAFlightID,
  Flight,
  FlightID,
  FlightIDType,
  FlightTimes as DomainFlightTimes,
  ISODateTimeString,
  PositionSource,
  PositionUpdateType,
  ProvisionalFlightLegID,
} from "./domain-types.js";

export const apiErrorCodes = [
  "validation_failed",
  "flightaware_budget_exceeded",
  "flightaware_rate_limited",
  "stale_cache_unavailable",
  "upstream_failure",
  "flightaware_fetch_disabled",
] as const;

export type ApiErrorCode = (typeof apiErrorCodes)[number];

export const fetchTaskTypes = ["summary", "position", "route", "track", "final_track"] as const;

export type FetchTaskType = (typeof fetchTaskTypes)[number];

export const refreshTaskTypes = ["position", "route", "track", "final_track"] as const;

export type RefreshTaskType = (typeof refreshTaskTypes)[number];

export const mvpExcludedContracts = ["websocket", "flightaware_alerts"] as const;

export type MvpExcludedContract = (typeof mvpExcludedContracts)[number];

export type CacheFreshness = "fresh" | "stale" | "miss";
export type CacheSource = "cache" | "flightaware" | "local_accounting" | "derived";
export type AirportBoardDirection = "departures" | "arrivals";
export type { FlightIDType, PositionSource } from "./domain-types.js";
export type MapSource =
  | "flightaware_route"
  | "airport_great_circle_fallback"
  | "flightaware_track"
  | "flightaware_track_and_position"
  | "flightaware_position";

export interface CacheMetadata {
  freshness: CacheFreshness;
  source: CacheSource;
  stale: boolean;
  checkedAt: string;
  fetchedAt?: string | null;
  staleAt?: string | null;
  expiresAt?: string | null;
}

export interface ApiError {
  code: ApiErrorCode;
  message: string;
  retryable: boolean;
  requestId: string;
  retryAfterSeconds?: number | null;
  staleCacheAvailable?: boolean;
}

export interface FlightSummaryItem {
  flightId: FlightID;
  flightIdType: FlightIDType;
  provisionalFlightLegId: ProvisionalFlightLegID | null;
  faFlightId: FAFlightID | null;
  ident: string;
  identIata: string | null;
  origin: string;
  destination: string;
  scheduledOut: ISODateTimeString | null;
  legIndex: number | null;
  status: string;
}

export type Airport = DomainAirport;
export type FlightTimes = DomainFlightTimes;

export type FlightDetail = Pick<
  Flight,
  | "flightId"
  | "flightIdType"
  | "provisionalFlightLegId"
  | "faFlightId"
  | "ident"
  | "identIata"
  | "aircraftType"
  | "registration"
  | "origin"
  | "destination"
  | "legIndex"
  | "status"
  | "progressPercent"
  | "times"
>;

export interface GeoJSONFeature {
  type: "Feature";
  geometry: {
    type: "Point" | "LineString" | "MultiLineString";
    coordinates: unknown;
  };
  properties: Record<string, unknown>;
}

export interface MapLayer {
  source: MapSource;
  available: boolean;
  geojson: GeoJSONFeature | null;
  unavailableReason?: string | null;
}

export interface Position {
  latitude: number;
  longitude: number;
  altitudeHundredsFeet: number | null;
  altitudeFeet: number | null;
  altitudeChange: AltitudeChange | null;
  groundspeedKnots: number | null;
  headingDegrees: number | null;
  timestamp: string;
  updateType: PositionUpdateType | null;
  source: PositionSource;
}

export interface FlightSearchResponse {
  items: FlightSummaryItem[];
  cache: CacheMetadata;
}

export interface AirportBoardResponse {
  airportCode: string;
  direction: AirportBoardDirection;
  date: string;
  items: FlightSummaryItem[];
  cache: CacheMetadata;
}

export interface FlightDetailResponse {
  flight: FlightDetail;
  route: MapLayer;
  track: MapLayer;
  current: Position | null;
  cache: CacheMetadata;
}

export interface FlightPositionsResponse {
  flightId: FlightID;
  items: Position[];
  cache: CacheMetadata;
}

export interface FlightMapDataResponse {
  flightId: FlightID;
  faFlightId: FAFlightID | null;
  planned: MapLayer;
  actual: MapLayer;
  current: MapLayer;
  cache: CacheMetadata;
}

export interface FetchTask {
  schemaVersion: 1;
  taskId: string;
  taskType: FetchTaskType;
  flightId: string;
  faFlightId?: string | null;
  requestedAt: string;
  reason: string;
  idempotencyKey: string;
  notBefore?: string | null;
  attempt?: number;
}

export interface FlightRefreshResponse {
  flightId: string;
  acceptedTasks: FetchTask[];
  cache: CacheMetadata;
}

export interface UsageStatusResponse {
  budget: {
    environment: string;
    month: string;
    currency: "USD";
    estimatedMonthToDateCost: number;
    softStopThreshold: number;
    stopped: boolean;
    dailyUsage: UsageDailyStatus[];
  };
  rateLimit: {
    limited: boolean;
    resetAt: string | null;
  };
  fetchingEnabled: boolean;
  cache: CacheMetadata;
}

export interface UsageDailyStatus {
  date: string;
  estimatedCostUSD: number;
  estimatedCallCount: number;
}
