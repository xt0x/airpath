import { apiRouteBuilders } from "@airpath/shared-types";

import type {
  ApiError,
  FetchTaskType,
  FlightDetailResponse,
  FlightMapDataResponse,
  FlightRefreshResponse,
  FlightSearchResponse,
  FlightSummaryItem,
  MapLayer,
  UsageStatusResponse,
} from "./types";

type FlightDetailPayload = FlightDetailResponse["flight"];
type PositionPayload = NonNullable<FlightDetailResponse["current"]>;

export interface AirpathApiClientConfig {
  basePath?: string;
  fetcher?: typeof fetch;
}

export class AirpathApiError extends Error {
  readonly error: ApiError;
  readonly status: number;

  constructor(error: ApiError, status: number) {
    super(error.message);
    this.name = "AirpathApiError";
    this.error = error;
    this.status = status;
  }
}

export class AirpathApiClient {
  private readonly basePath: string;
  private readonly fetcher: typeof fetch;

  constructor(config: AirpathApiClientConfig = {}) {
    this.basePath = (config.basePath ?? process.env.NEXT_PUBLIC_API_BASE_URL ?? "").replace(
      /\/$/,
      "",
    );
    this.fetcher = config.fetcher ?? fetch;
  }

  searchFlights(ident: string): Promise<FlightSearchResponse> {
    return this.request(apiRouteBuilders.searchFlights(ident), isFlightSearchResponse);
  }

  getFlightDetail(flightId: string): Promise<FlightDetailResponse> {
    return this.request(apiRouteBuilders.flightDetail(flightId), isFlightDetailResponse);
  }

  getFlightMapData(flightId: string): Promise<FlightMapDataResponse> {
    return this.request(apiRouteBuilders.flightMapData(flightId), isFlightMapDataResponse);
  }

  requestFlightRefresh(
    flightId: string,
    taskTypes: FetchTaskType[],
  ): Promise<FlightRefreshResponse> {
    return this.request(apiRouteBuilders.flightRefresh(flightId), isFlightRefreshResponse, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        taskTypes,
        clientReason: "user_manual_refresh",
      }),
    });
  }

  getUsageStatus(): Promise<UsageStatusResponse> {
    return this.request(apiRouteBuilders.usageStatus(), isUsageStatusResponse);
  }

  private async request<TResponse>(
    path: string,
    validate: (value: unknown) => value is TResponse,
    init?: RequestInit,
  ): Promise<TResponse> {
    const response = await this.fetcher(`${this.basePath}${path}`, {
      ...init,
      headers: {
        accept: "application/json",
        ...init?.headers,
      },
    });
    const payload = (await response.json()) as unknown;

    if (!response.ok) {
      const fallback: ApiError = {
        code: "upstream_failure",
        message: `Request failed with HTTP ${response.status}`,
        retryable: response.status >= 500,
        requestId: "unknown",
      };
      throw new AirpathApiError(
        isApiErrorResponse(payload) ? payload.error : fallback,
        response.status,
      );
    }

    if (!validate(payload)) {
      throw new AirpathApiError(
        {
          code: "upstream_failure",
          message: `Invalid response payload for ${path.split("?")[0]}`,
          retryable: true,
          requestId: "client-validation",
        },
        502,
      );
    }

    return payload;
  }
}

function isApiErrorResponse(value: unknown): value is { error: ApiError } {
  if (!isObject(value) || !("error" in value)) {
    return false;
  }
  const error = value.error;
  return (
    isObject(error) &&
    isString(error.code) &&
    isString(error.message) &&
    isBoolean(error.retryable) &&
    isString(error.requestId)
  );
}

function isFlightSearchResponse(value: unknown): value is FlightSearchResponse {
  return (
    isObject(value) && isArrayOf(value.items, isFlightSummaryItem) && isCacheMetadata(value.cache)
  );
}

function isFlightDetailResponse(value: unknown): value is FlightDetailResponse {
  return (
    isObject(value) &&
    isFlightDetail(value.flight) &&
    isMapLayer(value.route) &&
    isMapLayer(value.track) &&
    (value.current === null || isPosition(value.current)) &&
    isCacheMetadata(value.cache)
  );
}

function isFlightMapDataResponse(value: unknown): value is FlightMapDataResponse {
  return (
    isObject(value) &&
    isString(value.flightId) &&
    isNullableString(value.faFlightId) &&
    isMapLayer(value.planned) &&
    isMapLayer(value.actual) &&
    isMapLayer(value.current) &&
    isCacheMetadata(value.cache)
  );
}

function isFlightRefreshResponse(value: unknown): value is FlightRefreshResponse {
  return (
    isObject(value) &&
    isString(value.flightId) &&
    isArrayOf(value.acceptedTasks, isFetchTask) &&
    isCacheMetadata(value.cache)
  );
}

function isUsageStatusResponse(value: unknown): value is UsageStatusResponse {
  return (
    isObject(value) &&
    isObject(value.budget) &&
    isString(value.budget.environment) &&
    isString(value.budget.month) &&
    value.budget.currency === "USD" &&
    isNumber(value.budget.estimatedMonthToDateCost) &&
    isNumber(value.budget.softStopThreshold) &&
    isBoolean(value.budget.stopped) &&
    isObject(value.rateLimit) &&
    isBoolean(value.rateLimit.limited) &&
    isNullableString(value.rateLimit.resetAt) &&
    isBoolean(value.fetchingEnabled) &&
    isCacheMetadata(value.cache)
  );
}

function isFlightSummaryItem(value: unknown): value is FlightSummaryItem {
  return (
    isObject(value) &&
    isString(value.flightId) &&
    isString(value.flightIdType) &&
    isNullableString(value.provisionalFlightLegId) &&
    isNullableString(value.faFlightId) &&
    isString(value.ident) &&
    isNullableString(value.identIata) &&
    isString(value.origin) &&
    isString(value.destination) &&
    isNullableString(value.scheduledOut) &&
    isNullableNumber(value.legIndex) &&
    isString(value.status)
  );
}

function isFlightDetail(value: unknown): value is FlightDetailPayload {
  return (
    isFlightSummaryLike(value) &&
    isNullableString(value.aircraftType) &&
    isNullableString(value.registration) &&
    isAirport(value.origin) &&
    isAirport(value.destination) &&
    isNullableNumber(value.progressPercent) &&
    isFlightTimes(value.times)
  );
}

function isFlightSummaryLike(value: unknown): value is Record<string, unknown> {
  return (
    isObject(value) &&
    isString(value.flightId) &&
    isString(value.flightIdType) &&
    isNullableString(value.provisionalFlightLegId) &&
    isNullableString(value.faFlightId) &&
    isString(value.ident) &&
    isNullableString(value.identIata) &&
    isNullableNumber(value.legIndex) &&
    isString(value.status)
  );
}

function isAirport(value: unknown): boolean {
  return (
    isObject(value) &&
    isString(value.code) &&
    isNullableString(value.name) &&
    isNullableString(value.timezone)
  );
}

function isFlightTimes(value: unknown): boolean {
  const names = [
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
  ];
  return isObject(value) && names.every((name) => isNullableString(value[name]));
}

function isMapLayer(value: unknown): value is MapLayer {
  return (
    isObject(value) &&
    isString(value.source) &&
    isBoolean(value.available) &&
    (value.geojson === null || isGeoJSONFeature(value.geojson)) &&
    (value.unavailableReason === undefined || isNullableString(value.unavailableReason))
  );
}

function isGeoJSONFeature(value: unknown): boolean {
  return (
    isObject(value) &&
    value.type === "Feature" &&
    isObject(value.geometry) &&
    isString(value.geometry.type) &&
    "coordinates" in value.geometry &&
    isObject(value.properties)
  );
}

function isPosition(value: unknown): value is PositionPayload {
  return (
    isObject(value) &&
    isNumber(value.latitude) &&
    isNumber(value.longitude) &&
    isNullableNumber(value.altitudeHundredsFeet) &&
    isNullableNumber(value.altitudeFeet) &&
    isNullableString(value.altitudeChange) &&
    isNullableNumber(value.groundspeedKnots) &&
    isNullableNumber(value.headingDegrees) &&
    isString(value.timestamp) &&
    isNullableString(value.updateType) &&
    isString(value.source)
  );
}

function isFetchTask(value: unknown): value is FlightRefreshResponse["acceptedTasks"][number] {
  return (
    isObject(value) &&
    value.schemaVersion === 1 &&
    isString(value.taskId) &&
    isString(value.taskType) &&
    isString(value.flightId) &&
    (value.faFlightId === undefined || isNullableString(value.faFlightId)) &&
    isString(value.requestedAt) &&
    isString(value.reason) &&
    isString(value.idempotencyKey) &&
    (value.notBefore === undefined || isNullableString(value.notBefore)) &&
    (value.attempt === undefined || isNumber(value.attempt))
  );
}

function isCacheMetadata(value: unknown): boolean {
  return (
    isObject(value) &&
    isString(value.freshness) &&
    isString(value.source) &&
    isBoolean(value.stale) &&
    isString(value.checkedAt) &&
    (value.fetchedAt === undefined || isNullableString(value.fetchedAt)) &&
    (value.staleAt === undefined || isNullableString(value.staleAt)) &&
    (value.expiresAt === undefined || isNullableString(value.expiresAt))
  );
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isString(value: unknown): value is string {
  return typeof value === "string";
}

function isNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value);
}

function isBoolean(value: unknown): value is boolean {
  return typeof value === "boolean";
}

function isNullableString(value: unknown): value is string | null {
  return value === null || isString(value);
}

function isNullableNumber(value: unknown): value is number | null {
  return value === null || isNumber(value);
}

function isArrayOf<T>(value: unknown, validate: (item: unknown) => item is T): value is T[] {
  return Array.isArray(value) && value.every(validate);
}
