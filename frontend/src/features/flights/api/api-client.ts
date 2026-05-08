import { apiRouteBuilders, refreshTaskTypes, validateApiSchema } from "@airpath/shared-types";

import type {
  ApiError,
  ApiSchemaName,
  AirportBoardDirection,
  AirportBoardResponse,
  FlightDetailResponse,
  FlightMapDataResponse,
  FlightPositionsResponse,
  FlightRefreshResponse,
  FlightSearchResponse,
  RefreshTaskType,
  UsageStatusResponse,
} from "@/features/flights/types";

type UsageStatusPayload = Omit<UsageStatusResponse, "budget"> & {
  budget: Omit<UsageStatusResponse["budget"], "dailyUsage"> & {
    dailyUsage?: UsageStatusResponse["budget"]["dailyUsage"];
  };
};

const publicRefreshTaskTypes = refreshTaskTypes satisfies readonly RefreshTaskType[];

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
    this.basePath = normalizeBasePath(
      config.basePath ?? process.env.NEXT_PUBLIC_API_BASE_URL ?? "",
    );
    this.fetcher = config.fetcher ?? ((url, init) => globalThis.fetch(url, init));
  }

  searchFlights(ident: string): Promise<FlightSearchResponse> {
    return this.request(apiRouteBuilders.searchFlights(ident), "FlightSearchResponse");
  }

  getAirportBoard(
    airportCode: string,
    direction: AirportBoardDirection,
    query: { date?: string } = {},
  ): Promise<AirportBoardResponse> {
    const path =
      direction === "arrivals"
        ? apiRouteBuilders.airportArrivals(airportCode, query)
        : apiRouteBuilders.airportDepartures(airportCode, query);
    return this.request(path, "AirportBoardResponse");
  }

  getFlightDetail(flightId: string): Promise<FlightDetailResponse> {
    return this.request(apiRouteBuilders.flightDetail(flightId), "FlightDetailResponse");
  }

  getFlightMapData(flightId: string): Promise<FlightMapDataResponse> {
    return this.request(apiRouteBuilders.flightMapData(flightId), "FlightMapDataResponse");
  }

  getFlightPositions(
    flightId: string,
    query: { since?: string; limit?: number } = {},
  ): Promise<FlightPositionsResponse> {
    return this.request(
      apiRouteBuilders.flightPositions(flightId, query),
      "FlightPositionsResponse",
    );
  }

  requestFlightRefresh(
    flightId: string,
    taskTypes: readonly RefreshTaskType[],
  ): Promise<FlightRefreshResponse> {
    const validationErrorMessage = validateRefreshTaskTypes(taskTypes);
    if (validationErrorMessage !== null) {
      return Promise.reject(refreshValidationError(validationErrorMessage));
    }

    return this.request(apiRouteBuilders.flightRefresh(flightId), "FlightRefreshResponse", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        taskTypes,
        clientReason: "user_manual_refresh",
      }),
    });
  }

  getUsageStatus(): Promise<UsageStatusResponse> {
    return this.request(
      apiRouteBuilders.usageStatus(),
      "UsageStatusResponse",
      undefined,
      normalizeUsageStatusResponse,
      normalizeLegacyUsageStatusForValidation,
    );
  }

  private async request<TPayload, TResponse = TPayload>(
    path: string,
    responseSchemaName: ApiSchemaName,
    init?: RequestInit,
    normalize?: (payload: TPayload) => TResponse,
    normalizeBeforeValidation?: (payload: unknown) => unknown,
  ): Promise<TResponse> {
    const requestURL = `${this.basePath}${path}`;
    let response: Response;
    try {
      response = await this.fetcher(requestURL, {
        ...init,
        headers: {
          accept: "application/json",
          ...init?.headers,
        },
      });
    } catch (error) {
      if (error instanceof AirpathApiError) {
        throw error;
      }
      throw new AirpathApiError(
        {
          code: "upstream_failure",
          message: `Unable to reach Airpath API at ${this.apiOriginLabel()}. Start the local API with \`pnpm dev:api\`, then restart the frontend if NEXT_PUBLIC_API_BASE_URL changed.`,
          retryable: true,
          requestId: "client-network",
        },
        0,
      );
    }
    const payload = await this.readJsonPayload(response, path);

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

    const validationPayload = normalizeBeforeValidation?.(payload) ?? payload;
    if (!isApiSchemaPayload<TPayload>(responseSchemaName, validationPayload)) {
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

    return normalize === undefined
      ? (validationPayload as unknown as TResponse)
      : normalize(validationPayload);
  }

  private apiOriginLabel(): string {
    return this.basePath === "" ? "the same-origin /v1 proxy" : this.basePath;
  }

  private async readJsonPayload(response: Response, path: string): Promise<unknown> {
    const contentType = response.headers.get("content-type")?.split(";")[0]?.trim() ?? "";
    if (contentType !== "application/json") {
      throw new AirpathApiError(
        {
          code: "upstream_failure",
          message: `Expected JSON from ${path.split("?")[0]} but received ${contentType || "unknown content type"}; configure NEXT_PUBLIC_API_BASE_URL or a same-origin /v1 API proxy.`,
          retryable: true,
          requestId: "client-non-json",
        },
        response.status,
      );
    }

    try {
      return (await response.json()) as unknown;
    } catch {
      throw new AirpathApiError(
        {
          code: "upstream_failure",
          message: `Invalid JSON response for ${path.split("?")[0]}`,
          retryable: true,
          requestId: "client-invalid-json",
        },
        response.status,
      );
    }
  }
}

function normalizeBasePath(basePath: string): string {
  return basePath.replace(/\/+$/, "");
}

function isRefreshTaskType(value: unknown): value is RefreshTaskType {
  return isOneOf(value, publicRefreshTaskTypes);
}

function validateRefreshTaskTypes(taskTypes: unknown): string | null {
  if (!Array.isArray(taskTypes)) {
    return "Refresh task types must be an array.";
  }
  if (taskTypes.length === 0) {
    return "Refresh task types must include at least one task.";
  }
  const invalidTaskType = taskTypes.find((taskType) => !isRefreshTaskType(taskType));
  if (invalidTaskType !== undefined) {
    return `Invalid refresh task type: ${String(invalidTaskType)}`;
  }
  if (new Set(taskTypes).size !== taskTypes.length) {
    return "Refresh task types must be unique.";
  }
  return null;
}

function refreshValidationError(message: string): AirpathApiError {
  return new AirpathApiError(
    {
      code: "validation_failed",
      message,
      retryable: false,
      requestId: "client-validation",
    },
    400,
  );
}

function isApiErrorResponse(value: unknown): value is { error: ApiError } {
  return isApiSchemaPayload<{ error: ApiError }>("ApiErrorResponse", value);
}

function normalizeUsageStatusResponse(value: UsageStatusPayload): UsageStatusResponse {
  return {
    ...value,
    budget: {
      ...value.budget,
      dailyUsage: value.budget.dailyUsage ?? [],
    },
  };
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isOneOf<TValue extends string>(
  value: unknown,
  allowedValues: readonly TValue[],
): value is TValue {
  return isString(value) && (allowedValues as readonly string[]).includes(value);
}

function isString(value: unknown): value is string {
  return typeof value === "string";
}

function isApiSchemaPayload<TPayload>(
  schemaName: ApiSchemaName,
  value: unknown,
): value is TPayload {
  return validateApiSchema(schemaName, value).valid;
}

function normalizeLegacyUsageStatusForValidation(payload: unknown): unknown {
  if (!isObject(payload) || !isObject(payload.budget) || "dailyUsage" in payload.budget) {
    return payload;
  }

  return {
    ...payload,
    budget: {
      ...payload.budget,
      dailyUsage: [],
    },
  };
}
