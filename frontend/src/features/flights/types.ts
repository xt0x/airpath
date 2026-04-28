import type { ApiErrorCode, FetchTaskType } from "@airpath/shared-types";

export type { FetchTaskType };

export type CacheFreshness = "fresh" | "stale" | "miss";
export type CacheSource = "cache" | "flightaware" | "local_accounting" | "derived";

export interface CacheMetadata {
  freshness: CacheFreshness;
  source: CacheSource;
  stale: boolean;
  checkedAt: string;
  fetchedAt?: string | null;
  staleAt?: string | null;
  expiresAt?: string | null;
}

export interface FlightSummaryItem {
  flightId: string;
  flightIdType: "internal" | "provisional";
  provisionalFlightLegId: string | null;
  faFlightId: string | null;
  ident: string;
  identIata: string | null;
  origin: string;
  destination: string;
  scheduledOut: string | null;
  legIndex: number | null;
  status: string;
}

export interface Airport {
  code: string;
  name: string | null;
  timezone: string | null;
}

export interface FlightTimes {
  scheduledOut: string | null;
  estimatedOut: string | null;
  actualOut: string | null;
  scheduledOff: string | null;
  estimatedOff: string | null;
  actualOff: string | null;
  scheduledOn: string | null;
  estimatedOn: string | null;
  actualOn: string | null;
  scheduledIn: string | null;
  estimatedIn: string | null;
  actualIn: string | null;
}

export interface FlightDetail {
  flightId: string;
  flightIdType: "internal" | "provisional";
  provisionalFlightLegId: string | null;
  faFlightId: string | null;
  ident: string;
  identIata: string | null;
  aircraftType: string | null;
  registration: string | null;
  origin: Airport;
  destination: Airport;
  legIndex: number | null;
  status: string;
  progressPercent: number | null;
  times: FlightTimes;
}

export type MapSource =
  | "flightaware_route"
  | "airport_great_circle_fallback"
  | "flightaware_track"
  | "flightaware_track_and_position"
  | "flightaware_position";

export interface MapLayer {
  source: MapSource;
  available: boolean;
  geojson: GeoJSONFeature | null;
  unavailableReason?: string | null;
}

export interface GeoJSONFeature {
  type: "Feature";
  geometry: {
    type: "Point" | "LineString" | "MultiLineString";
    coordinates: unknown;
  };
  properties: Record<string, unknown>;
}

export interface Position {
  latitude: number;
  longitude: number;
  altitudeHundredsFeet: number | null;
  altitudeFeet: number | null;
  groundspeedKnots: number | null;
  headingDegrees: number | null;
  timestamp: string;
  source: "flightaware_position" | "flightaware_track" | "manual";
}

export interface FlightSearchResponse {
  items: FlightSummaryItem[];
  cache: CacheMetadata;
}

export interface FlightDetailResponse {
  flight: FlightDetail;
  route?: MapLayer;
  track?: MapLayer;
  current?: Position | null;
  cache: CacheMetadata;
}

export interface FlightMapDataResponse {
  flightId: string;
  faFlightId: string | null;
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
  };
  rateLimit: {
    limited: boolean;
    resetAt: string | null;
  };
  fetchingEnabled: boolean;
  cache: CacheMetadata;
}

export interface ApiError {
  code: ApiErrorCode;
  message: string;
  retryable: boolean;
  requestId: string;
  retryAfterSeconds?: number | null;
  staleCacheAvailable?: boolean;
}
