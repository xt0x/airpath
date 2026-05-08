import { describe, expect, it } from "vitest";

import { AirpathApiClient, AirpathApiError } from "@/features/flights/api/api-client";
import type {
  AirportBoardResponse,
  FlightPositionsResponse,
  FlightSearchResponse,
  RefreshTaskType,
  UsageStatusResponse,
} from "@/features/flights/types";

describe("AirpathApiClient", () => {
  it("uses NEXT_PUBLIC_API_BASE_URL as an origin/base URL without duplicating the API prefix", async () => {
    const originalBaseURL = process.env.NEXT_PUBLIC_API_BASE_URL;
    process.env.NEXT_PUBLIC_API_BASE_URL = "https://api.example.test/";
    const calls: string[] = [];
    try {
      const client = new AirpathApiClient({
        fetcher: async (url) => {
          calls.push(String(url));
          return jsonResponse(responseFor(String(url)));
        },
      });

      await client.getUsageStatus();

      expect(calls).toEqual(["https://api.example.test/v1/usage/status"]);
    } finally {
      if (originalBaseURL === undefined) {
        delete process.env.NEXT_PUBLIC_API_BASE_URL;
      } else {
        process.env.NEXT_PUBLIC_API_BASE_URL = originalBaseURL;
      }
    }
  });

  it("calls the default global fetch without rebinding it to the API client instance", async () => {
    const originalFetch = globalThis.fetch;
    const calls: string[] = [];
    try {
      globalThis.fetch = function fetchWithWindowReceiver(this: typeof globalThis, url) {
        if (this !== globalThis) {
          throw new TypeError("Illegal invocation");
        }
        calls.push(String(url));
        return Promise.resolve(jsonResponse(responseFor(String(url))));
      } as typeof fetch;

      const client = new AirpathApiClient({ basePath: "https://api.example.test" });

      await client.getUsageStatus();

      expect(calls).toEqual(["https://api.example.test/v1/usage/status"]);
    } finally {
      globalThis.fetch = originalFetch;
    }
  });

  it("calls the MVP flight and usage endpoints with typed responses", async () => {
    const calls: Array<{ url: string; init: RequestInit | undefined }> = [];
    const client = new AirpathApiClient({
      basePath: "https://api.example.test",
      fetcher: async (url, init) => {
        calls.push({ url: String(url), init });
        return jsonResponse(responseFor(String(url)));
      },
    });

    await client.searchFlights("ANA110");
    await client.getAirportBoard("rjtt", "departures", { date: "2026-05-04" });
    await client.getAirportBoard("kjfk", "arrivals", { date: "2026-05-04" });
    await client.getFlightDetail("iflg_1");
    await client.getFlightMapData("iflg_1");
    await client.getFlightPositions("iflg_1", {
      since: "2026-04-29T00:00:00Z",
      limit: 50,
    });
    await client.requestFlightRefresh("iflg_1", ["route", "position"]);
    await client.getUsageStatus();

    expect(calls.map((call) => call.url)).toEqual([
      "https://api.example.test/v1/flights/search?ident=ANA110",
      "https://api.example.test/v1/airports/RJTT/departures?date=2026-05-04",
      "https://api.example.test/v1/airports/KJFK/arrivals?date=2026-05-04",
      "https://api.example.test/v1/flights/iflg_1",
      "https://api.example.test/v1/flights/iflg_1/map-data",
      "https://api.example.test/v1/flights/iflg_1/positions?since=2026-04-29T00%3A00%3A00Z&limit=50",
      "https://api.example.test/v1/flights/iflg_1/refresh",
      "https://api.example.test/v1/usage/status",
    ]);
    expect(calls[6]?.init?.method).toBe("POST");
    expect(JSON.parse(String(calls[6]?.init?.body))).toEqual({
      taskTypes: ["route", "position"],
      clientReason: "user_manual_refresh",
    });
  });

  it("strips every trailing slash from the configured API base path", async () => {
    const calls: string[] = [];
    const client = new AirpathApiClient({
      basePath: "https://api.example.test///",
      fetcher: async (url) => {
        calls.push(String(url));
        return jsonResponse(responseFor(String(url)));
      },
    });

    await client.getUsageStatus();

    expect(calls).toEqual(["https://api.example.test/v1/usage/status"]);
  });

  it("rejects refresh task values outside the browser refresh contract before making a request", async () => {
    const calls: string[] = [];
    const client = new AirpathApiClient({
      basePath: "https://api.example.test",
      fetcher: async (url) => {
        calls.push(String(url));
        return jsonResponse(responseFor(String(url)));
      },
    });

    await expectRefreshInputRejected(client, ["summary"] as unknown as readonly RefreshTaskType[]);

    expect(calls).toEqual([]);
  });

  it("rejects refresh task arrays that violate the request body schema before making a request", async () => {
    const calls: string[] = [];
    const client = new AirpathApiClient({
      basePath: "https://api.example.test",
      fetcher: async (url) => {
        calls.push(String(url));
        return jsonResponse(responseFor(String(url)));
      },
    });

    await expectRefreshInputRejected(client, null as unknown as readonly RefreshTaskType[]);
    await expectRefreshInputRejected(client, []);
    await expectRefreshInputRejected(client, ["route", "route"]);

    expect(calls).toEqual([]);
  });

  it("throws typed API errors", async () => {
    const client = new AirpathApiClient({
      fetcher: async () =>
        jsonResponse(
          {
            error: {
              code: "flightaware_budget_exceeded",
              message: "budget stopped",
              retryable: false,
              requestId: "req-1",
            },
          },
          503,
        ),
    });

    await expect(client.searchFlights("ANA110")).rejects.toMatchObject({
      name: "AirpathApiError",
      error: { code: "flightaware_budget_exceeded", requestId: "req-1" },
    } satisfies {
      name: string;
      error: Pick<AirpathApiError["error"], "code" | "requestId">;
    });
  });

  it("accepts legacy usage status responses without daily usage and normalizes them to an empty chart series", async () => {
    const client = new AirpathApiClient({
      fetcher: async () => {
        const response = responseFor("/v1/usage/status") as UsageStatusResponse;
        const budget = Object.fromEntries(
          Object.entries(response.budget).filter(([key]) => key !== "dailyUsage"),
        );
        return jsonResponse({
          ...response,
          budget,
        });
      },
    });

    await expect(client.getUsageStatus()).resolves.toMatchObject({
      budget: {
        dailyUsage: [],
      },
    });
  });

  it("rejects malformed success responses before feature state consumes them", async () => {
    const client = new AirpathApiClient({
      fetcher: async () => jsonResponse({ cache: validCache() }),
    });

    await expect(client.searchFlights("ANA110")).rejects.toMatchObject({
      name: "AirpathApiError",
      status: 502,
      error: {
        code: "upstream_failure",
        message: "Invalid response payload for /v1/flights/search",
        retryable: true,
        requestId: "client-validation",
      },
    });
  });

  it("turns non-JSON API responses into typed configuration errors", async () => {
    const client = new AirpathApiClient({
      fetcher: async () =>
        new Response("<!DOCTYPE html><h1>Not Found</h1>", {
          status: 404,
          headers: { "content-type": "text/html; charset=utf-8" },
        }),
    });

    await expect(client.getUsageStatus()).rejects.toMatchObject({
      name: "AirpathApiError",
      status: 404,
      error: {
        code: "upstream_failure",
        message:
          "Expected JSON from /v1/usage/status but received text/html; configure NEXT_PUBLIC_API_BASE_URL or a same-origin /v1 API proxy.",
        retryable: true,
        requestId: "client-non-json",
      },
    });
  });

  it("turns network failures into actionable local API errors", async () => {
    const client = new AirpathApiClient({
      basePath: "http://localhost:8080",
      fetcher: async () => {
        throw new TypeError("fetch failed");
      },
    });

    await expect(client.searchFlights("ANA110")).rejects.toMatchObject({
      name: "AirpathApiError",
      status: 0,
      error: {
        code: "upstream_failure",
        message:
          "Unable to reach Airpath API at http://localhost:8080. Start the local API with `pnpm dev:api`, then restart the frontend if NEXT_PUBLIC_API_BASE_URL changed.",
        retryable: true,
        requestId: "client-network",
      },
    });
  });
});

function responseFor(
  url: string,
):
  | AirportBoardResponse
  | FlightSearchResponse
  | FlightPositionsResponse
  | UsageStatusResponse
  | Record<string, unknown> {
  const cache = validCache();
  if (url.includes("/airports/")) {
    const direction = url.includes("/arrivals") ? "arrivals" : "departures";
    return {
      airportCode: url.includes("KJFK") ? "KJFK" : "RJTT",
      direction,
      date: "2026-05-04",
      items: [flightSummary()],
      cache,
    };
  }
  if (url.includes("/search")) {
    return { items: [], cache };
  }
  if (url.includes("/usage/status")) {
    return {
      budget: {
        environment: "dev",
        month: "2026-04",
        currency: "USD",
        estimatedMonthToDateCost: 1,
        softStopThreshold: 4,
        stopped: false,
        dailyUsage: [{ date: "2026-04-29", estimatedCostUSD: 0.2, estimatedCallCount: 20 }],
      },
      rateLimit: { limited: false, resetAt: null },
      fetchingEnabled: true,
      cache,
    };
  }
  if (url.includes("/refresh")) {
    return { flightId: "iflg_1", acceptedTasks: [], cache };
  }
  if (url.includes("/positions")) {
    return {
      flightId: "iflg_1",
      items: [
        {
          latitude: 45.1,
          longitude: 160.4,
          altitudeHundredsFeet: 370,
          altitudeFeet: 37000,
          altitudeChange: "level",
          groundspeedKnots: 488,
          headingDegrees: 275,
          timestamp: "2026-04-29T00:05:00Z",
          updateType: "estimated",
          source: "flightaware_position",
        },
      ],
      cache,
    };
  }
  if (url.includes("/map-data")) {
    const unavailable = {
      source: "flightaware_route",
      available: false,
      geojson: null,
      unavailableReason: "not fetched",
    };
    return {
      flightId: "iflg_1",
      faFlightId: "fa_1",
      planned: unavailable,
      actual: { ...unavailable, source: "flightaware_track" },
      current: { ...unavailable, source: "flightaware_position" },
      cache,
    };
  }
  return {
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
        estimatedOut: null,
        actualOut: null,
        scheduledOff: null,
        estimatedOff: null,
        actualOff: null,
        scheduledOn: null,
        estimatedOn: null,
        actualOn: null,
        scheduledIn: null,
        estimatedIn: null,
        actualIn: null,
      },
    },
    route: {
      source: "flightaware_route",
      available: false,
      geojson: null,
      unavailableReason: "not fetched",
    },
    track: {
      source: "flightaware_track",
      available: false,
      geojson: null,
      unavailableReason: "not fetched",
    },
    current: null,
    cache,
  };
}

function flightSummary(): FlightSearchResponse["items"][number] {
  return {
    flightId: "iflg_1",
    flightIdType: "internal",
    provisionalFlightLegId: null,
    faFlightId: "fa_1",
    ident: "ANA110",
    identIata: "NH110",
    origin: "RJTT",
    destination: "KJFK",
    scheduledOut: "2026-05-04T01:00:00Z",
    legIndex: 0,
    status: "Scheduled",
  };
}

function validCache() {
  return {
    freshness: "fresh" as const,
    source: "cache" as const,
    stale: false,
    checkedAt: "2026-04-29T00:00:00Z",
  };
}

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

async function expectRefreshInputRejected(
  client: AirpathApiClient,
  taskTypes: readonly RefreshTaskType[],
): Promise<void> {
  await expect(client.requestFlightRefresh("iflg_1", taskTypes)).rejects.toMatchObject({
    name: "AirpathApiError",
    status: 400,
    error: {
      code: "validation_failed",
      retryable: false,
      requestId: "client-validation",
    },
  });
}
