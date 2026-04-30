import { apiErrorCodes, fetchTaskTypes } from "./api-types.js";

const cacheMetadataSchema = {
  type: "object",
  additionalProperties: false,
  required: ["freshness", "source", "stale", "checkedAt"],
  properties: {
    freshness: {
      type: "string",
      enum: ["fresh", "stale", "miss"],
    },
    source: {
      type: "string",
      enum: ["cache", "flightaware", "local_accounting", "derived"],
    },
    stale: {
      type: "boolean",
    },
    checkedAt: {
      type: "string",
      format: "date-time",
    },
    fetchedAt: {
      type: ["string", "null"],
      format: "date-time",
    },
    staleAt: {
      type: ["string", "null"],
      format: "date-time",
    },
    expiresAt: {
      type: ["string", "null"],
      format: "date-time",
    },
  },
} as const;

const apiErrorSchema = {
  type: "object",
  additionalProperties: false,
  required: ["code", "message", "retryable", "requestId"],
  properties: {
    code: {
      type: "string",
      enum: apiErrorCodes,
    },
    message: {
      type: "string",
    },
    retryable: {
      type: "boolean",
    },
    requestId: {
      type: "string",
    },
    retryAfterSeconds: {
      type: ["integer", "null"],
      minimum: 0,
    },
    staleCacheAvailable: {
      type: "boolean",
    },
  },
} as const;

const apiErrorResponseSchema = {
  type: "object",
  additionalProperties: false,
  required: ["error"],
  properties: {
    error: {
      $ref: "#/components/schemas/ApiError",
    },
  },
} as const;

const airportSchema = {
  type: "object",
  additionalProperties: false,
  required: ["code", "name", "timezone"],
  properties: {
    code: {
      type: "string",
    },
    name: {
      type: ["string", "null"],
    },
    timezone: {
      type: ["string", "null"],
    },
  },
} as const;

const flightTimesSchema = {
  type: "object",
  additionalProperties: false,
  required: [
    "scheduledOut",
    "estimatedOut",
    "actualOut",
    "scheduledOff",
    "estimatedOff",
    "actualOff",
    "scheduledOn",
    "estimatedOn",
    "actualOn",
    "scheduledIn",
    "estimatedIn",
    "actualIn",
  ],
  properties: {
    scheduledOut: { type: ["string", "null"], format: "date-time" },
    estimatedOut: { type: ["string", "null"], format: "date-time" },
    actualOut: { type: ["string", "null"], format: "date-time" },
    scheduledOff: { type: ["string", "null"], format: "date-time" },
    estimatedOff: { type: ["string", "null"], format: "date-time" },
    actualOff: { type: ["string", "null"], format: "date-time" },
    scheduledOn: { type: ["string", "null"], format: "date-time" },
    estimatedOn: { type: ["string", "null"], format: "date-time" },
    actualOn: { type: ["string", "null"], format: "date-time" },
    scheduledIn: { type: ["string", "null"], format: "date-time" },
    estimatedIn: { type: ["string", "null"], format: "date-time" },
    actualIn: { type: ["string", "null"], format: "date-time" },
  },
} as const;

const flightSummarySchema = {
  type: "object",
  additionalProperties: false,
  required: [
    "flightId",
    "flightIdType",
    "provisionalFlightLegId",
    "faFlightId",
    "ident",
    "identIata",
    "origin",
    "destination",
    "scheduledOut",
    "legIndex",
    "status",
  ],
  properties: {
    flightId: { type: "string" },
    flightIdType: { type: "string", enum: ["internal", "provisional"] },
    provisionalFlightLegId: { type: ["string", "null"] },
    faFlightId: { type: ["string", "null"] },
    ident: { type: "string" },
    identIata: { type: ["string", "null"] },
    origin: { type: "string" },
    destination: { type: "string" },
    scheduledOut: { type: ["string", "null"], format: "date-time" },
    legIndex: { type: ["integer", "null"], minimum: 0 },
    status: { type: "string" },
  },
} as const;

const flightDetailSchema = {
  type: "object",
  additionalProperties: false,
  required: [
    "flightId",
    "flightIdType",
    "provisionalFlightLegId",
    "faFlightId",
    "ident",
    "identIata",
    "aircraftType",
    "registration",
    "origin",
    "destination",
    "legIndex",
    "status",
    "progressPercent",
    "times",
  ],
  properties: {
    flightId: { type: "string" },
    flightIdType: { type: "string", enum: ["internal", "provisional"] },
    provisionalFlightLegId: { type: ["string", "null"] },
    faFlightId: { type: ["string", "null"] },
    ident: { type: "string" },
    identIata: { type: ["string", "null"] },
    aircraftType: { type: ["string", "null"] },
    registration: { type: ["string", "null"] },
    origin: { $ref: "#/components/schemas/Airport" },
    destination: { $ref: "#/components/schemas/Airport" },
    legIndex: { type: ["integer", "null"], minimum: 0 },
    status: { type: "string" },
    progressPercent: { type: ["integer", "null"], minimum: 0, maximum: 100 },
    times: { $ref: "#/components/schemas/FlightTimes" },
  },
} as const;

const geoJsonFeatureSchema = {
  type: "object",
  required: ["type", "geometry", "properties"],
  properties: {
    type: {
      type: "string",
      const: "Feature",
    },
    geometry: {
      type: "object",
    },
    properties: {
      type: "object",
    },
  },
} as const;

const mapLayerSchema = {
  type: "object",
  additionalProperties: false,
  required: ["source", "available", "geojson"],
  properties: {
    source: {
      type: "string",
      enum: [
        "flightaware_route",
        "airport_great_circle_fallback",
        "flightaware_track",
        "flightaware_track_and_position",
        "flightaware_position",
      ],
    },
    available: {
      type: "boolean",
    },
    geojson: {
      oneOf: [{ $ref: "#/components/schemas/GeoJsonFeature" }, { type: "null" }],
    },
    unavailableReason: {
      type: ["string", "null"],
    },
  },
} as const;

const positionSchema = {
  type: "object",
  additionalProperties: false,
  required: [
    "latitude",
    "longitude",
    "altitudeHundredsFeet",
    "altitudeFeet",
    "altitudeChange",
    "groundspeedKnots",
    "headingDegrees",
    "timestamp",
    "updateType",
    "source",
  ],
  properties: {
    latitude: { type: "number", minimum: -90, maximum: 90 },
    longitude: { type: "number", minimum: -180, maximum: 180 },
    altitudeHundredsFeet: { type: ["integer", "null"] },
    altitudeFeet: { type: ["integer", "null"] },
    altitudeChange: {
      type: ["string", "null"],
      enum: ["climbing", "descending", "level", null],
    },
    groundspeedKnots: { type: ["integer", "null"], minimum: 0 },
    headingDegrees: { type: ["integer", "null"], minimum: 0, maximum: 359 },
    timestamp: { type: "string", format: "date-time" },
    updateType: {
      type: ["string", "null"],
      enum: ["actual", "estimated", "predicted", "surface", null],
    },
    source: {
      type: "string",
      enum: ["flightaware_position", "flightaware_track", "manual"],
    },
  },
} as const;

export const fetchTaskSchema = {
  type: "object",
  additionalProperties: false,
  required: [
    "schemaVersion",
    "taskId",
    "taskType",
    "flightId",
    "requestedAt",
    "reason",
    "idempotencyKey",
  ],
  properties: {
    schemaVersion: {
      type: "integer",
      const: 1,
    },
    taskId: {
      type: "string",
    },
    taskType: {
      type: "string",
      enum: fetchTaskTypes,
    },
    flightId: {
      type: "string",
    },
    faFlightId: {
      type: ["string", "null"],
    },
    requestedAt: {
      type: "string",
      format: "date-time",
    },
    reason: {
      type: "string",
      enum: [
        "search_result_seed",
        "user_opened_flight_detail",
        "user_manual_refresh",
        "low_frequency_poll",
        "arrival_finalization",
        "usage_reconciliation",
      ],
    },
    idempotencyKey: {
      type: "string",
    },
    notBefore: {
      type: ["string", "null"],
      format: "date-time",
    },
    attempt: {
      type: "integer",
      minimum: 0,
      default: 0,
    },
  },
} as const;

export const successJsonResponse = {
  description: "Successful response",
  content: {
    "application/json": {
      schema: {
        type: "object",
      },
    },
  },
} as const;

export const typedErrorResponse = {
  description: "Typed API error",
  content: {
    "application/json": {
      schema: {
        $ref: "#/components/schemas/ApiErrorResponse",
      },
    },
  },
} as const;

export const flightIdParameter = {
  name: "flightId",
  in: "path",
  required: true,
  schema: {
    type: "string",
  },
} as const;

export const apiSchemas = {
  ApiError: apiErrorSchema,
  ApiErrorResponse: apiErrorResponseSchema,
  Airport: airportSchema,
  CacheMetadata: cacheMetadataSchema,
  FetchTask: fetchTaskSchema,
  FlightTimes: flightTimesSchema,
  FlightSummary: flightSummarySchema,
  FlightDetail: flightDetailSchema,
  GeoJsonFeature: geoJsonFeatureSchema,
  MapLayer: mapLayerSchema,
  Position: positionSchema,
  FlightSearchResponse: {
    type: "object",
    additionalProperties: false,
    required: ["items", "cache"],
    properties: {
      items: {
        type: "array",
        items: { $ref: "#/components/schemas/FlightSummary" },
      },
      cache: { $ref: "#/components/schemas/CacheMetadata" },
    },
  },
  FlightDetailResponse: {
    type: "object",
    additionalProperties: false,
    required: ["flight", "route", "track", "current", "cache"],
    properties: {
      flight: { $ref: "#/components/schemas/FlightDetail" },
      route: { $ref: "#/components/schemas/MapLayer" },
      track: { $ref: "#/components/schemas/MapLayer" },
      current: {
        anyOf: [{ $ref: "#/components/schemas/Position" }, { type: "null" }],
      },
      cache: { $ref: "#/components/schemas/CacheMetadata" },
    },
  },
  FlightMapDataResponse: {
    type: "object",
    additionalProperties: false,
    required: ["flightId", "faFlightId", "planned", "actual", "current", "cache"],
    properties: {
      flightId: { type: "string" },
      faFlightId: { type: ["string", "null"] },
      planned: { $ref: "#/components/schemas/MapLayer" },
      actual: { $ref: "#/components/schemas/MapLayer" },
      current: { $ref: "#/components/schemas/MapLayer" },
      cache: { $ref: "#/components/schemas/CacheMetadata" },
    },
  },
  FlightPositionsResponse: {
    type: "object",
    additionalProperties: false,
    required: ["flightId", "items", "cache"],
    properties: {
      flightId: { type: "string" },
      items: {
        type: "array",
        items: { $ref: "#/components/schemas/Position" },
      },
      cache: { $ref: "#/components/schemas/CacheMetadata" },
    },
  },
  FlightRefreshResponse: {
    type: "object",
    additionalProperties: false,
    required: ["flightId", "acceptedTasks", "cache"],
    properties: {
      flightId: { type: "string" },
      acceptedTasks: {
        type: "array",
        items: { $ref: "#/components/schemas/FetchTask" },
      },
      cache: { $ref: "#/components/schemas/CacheMetadata" },
    },
  },
  UsageStatusResponse: {
    type: "object",
    additionalProperties: false,
    required: ["budget", "rateLimit", "fetchingEnabled", "cache"],
    properties: {
      budget: {
        type: "object",
        additionalProperties: false,
        required: [
          "environment",
          "month",
          "currency",
          "estimatedMonthToDateCost",
          "softStopThreshold",
          "stopped",
        ],
        properties: {
          environment: { type: "string", minLength: 1 },
          month: { type: "string", pattern: "^\\d{4}-\\d{2}$" },
          currency: { type: "string", const: "USD" },
          estimatedMonthToDateCost: { type: "number", minimum: 0 },
          softStopThreshold: { type: "number", minimum: 0 },
          stopped: { type: "boolean" },
        },
      },
      rateLimit: {
        type: "object",
        additionalProperties: false,
        required: ["limited", "resetAt"],
        properties: {
          limited: { type: "boolean" },
          resetAt: { type: ["string", "null"], format: "date-time" },
        },
      },
      fetchingEnabled: { type: "boolean" },
      cache: { $ref: "#/components/schemas/CacheMetadata" },
    },
  },
} as const;
