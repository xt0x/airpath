import type {
  ApiError,
  FetchTaskType,
  FlightDetailResponse,
  FlightMapDataResponse,
  FlightRefreshResponse,
  FlightSearchResponse,
  UsageStatusResponse,
} from "./types";

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
    this.basePath = config.basePath?.replace(/\/$/, "") ?? "";
    this.fetcher = config.fetcher ?? fetch;
  }

  searchFlights(ident: string): Promise<FlightSearchResponse> {
    const params = new URLSearchParams({ ident });
    return this.request(`/v1/flights/search?${params.toString()}`);
  }

  getFlightDetail(flightId: string): Promise<FlightDetailResponse> {
    return this.request(`/v1/flights/${encodeURIComponent(flightId)}`);
  }

  getFlightMapData(flightId: string): Promise<FlightMapDataResponse> {
    return this.request(`/v1/flights/${encodeURIComponent(flightId)}/map-data`);
  }

  requestFlightRefresh(
    flightId: string,
    taskTypes: FetchTaskType[],
  ): Promise<FlightRefreshResponse> {
    return this.request(`/v1/flights/${encodeURIComponent(flightId)}/refresh`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        taskTypes,
        clientReason: "user_manual_refresh",
      }),
    });
  }

  getUsageStatus(): Promise<UsageStatusResponse> {
    return this.request("/v1/usage/status");
  }

  private async request<TResponse>(path: string, init?: RequestInit): Promise<TResponse> {
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

    return payload as TResponse;
  }
}

function isApiErrorResponse(value: unknown): value is { error: ApiError } {
  return typeof value === "object" && value !== null && "error" in value;
}
