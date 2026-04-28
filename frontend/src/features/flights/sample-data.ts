import type {
  FlightDetailResponse,
  FlightMapDataResponse,
  FlightSearchResponse,
  UsageStatusResponse,
} from "./types";

const freshCache = {
  freshness: "fresh" as const,
  source: "cache" as const,
  stale: false,
  checkedAt: "2026-04-29T00:00:00Z",
  fetchedAt: "2026-04-29T00:00:00Z",
};

export const sampleSearch: FlightSearchResponse = {
  items: [
    {
      flightId: "iflg_1",
      flightIdType: "internal",
      provisionalFlightLegId: null,
      faFlightId: "fa_1",
      ident: "ANA110",
      identIata: "NH110",
      origin: "RJTT",
      destination: "KJFK",
      scheduledOut: "2026-04-29T00:00:00Z",
      legIndex: 0,
      status: "En Route",
    },
  ],
  cache: freshCache,
};

export const sampleDetail: FlightDetailResponse = {
  flight: {
    flightId: "iflg_1",
    flightIdType: "internal",
    provisionalFlightLegId: null,
    faFlightId: "fa_1",
    ident: "ANA110",
    identIata: "NH110",
    aircraftType: "B789",
    registration: null,
    origin: { code: "RJTT", name: "Tokyo Haneda", timezone: "Asia/Tokyo" },
    destination: {
      code: "KJFK",
      name: "John F. Kennedy International",
      timezone: "America/New_York",
    },
    legIndex: 0,
    status: "En Route",
    progressPercent: 42,
    times: {
      scheduledOut: "2026-04-29T00:00:00Z",
      estimatedOut: "2026-04-29T00:04:00Z",
      actualOut: null,
      scheduledOff: null,
      estimatedOff: null,
      actualOff: null,
      scheduledOn: null,
      estimatedOn: "2026-04-29T12:20:00Z",
      actualOn: null,
      scheduledIn: "2026-04-29T12:45:00Z",
      estimatedIn: null,
      actualIn: null,
    },
  },
  cache: freshCache,
};

export const sampleMapData: FlightMapDataResponse = {
  flightId: "iflg_1",
  faFlightId: "fa_1",
  planned: {
    source: "flightaware_route",
    available: true,
    geojson: {
      type: "Feature",
      geometry: {
        type: "MultiLineString",
        coordinates: [
          [
            [139.78, 35.55],
            [160.45, 45.12],
            [-73.78, 40.64],
          ],
        ],
      },
      properties: { kind: "planned_route" },
    },
  },
  actual: {
    source: "flightaware_track",
    available: true,
    geojson: {
      type: "Feature",
      geometry: {
        type: "MultiLineString",
        coordinates: [
          [
            [139.78, 35.55],
            [150.25, 41.12],
            [160.45, 45.12],
          ],
        ],
      },
      properties: { kind: "actual_track" },
    },
  },
  current: {
    source: "flightaware_position",
    available: true,
    geojson: {
      type: "Feature",
      geometry: { type: "Point", coordinates: [160.45, 45.12] },
      properties: { kind: "current_position", timestamp: "2026-04-29T03:30:00Z" },
    },
  },
  cache: freshCache,
};

export const sampleUsage: UsageStatusResponse = {
  budget: {
    environment: "dev",
    month: "2026-04",
    currency: "USD",
    estimatedMonthToDateCost: 1,
    softStopThreshold: 4,
    stopped: false,
  },
  rateLimit: { limited: false, resetAt: null },
  fetchingEnabled: true,
  cache: {
    freshness: "fresh",
    source: "local_accounting",
    stale: false,
    checkedAt: "2026-04-29T00:00:00Z",
  },
};
