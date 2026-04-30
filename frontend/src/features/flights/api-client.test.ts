import { describe, expect, it } from "vitest";

import { AirpathApiClient, AirpathApiError } from "./api-client";
import type { FlightSearchResponse, UsageStatusResponse } from "./types";

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
    await client.getFlightDetail("iflg_1");
    await client.getFlightMapData("iflg_1");
    await client.requestFlightRefresh("iflg_1", ["route", "position"]);
    await client.getUsageStatus();

    expect(calls.map((call) => call.url)).toEqual([
      "https://api.example.test/v1/flights/search?ident=ANA110",
      "https://api.example.test/v1/flights/iflg_1",
      "https://api.example.test/v1/flights/iflg_1/map-data",
      "https://api.example.test/v1/flights/iflg_1/refresh",
      "https://api.example.test/v1/usage/status",
    ]);
    expect(calls[3]?.init?.method).toBe("POST");
    expect(JSON.parse(String(calls[3]?.init?.body))).toEqual({
      taskTypes: ["route", "position"],
      clientReason: "user_manual_refresh",
    });
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
});

function responseFor(
  url: string,
): FlightSearchResponse | UsageStatusResponse | Record<string, unknown> {
  const cache = validCache();
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
      },
      rateLimit: { limited: false, resetAt: null },
      fetchingEnabled: true,
      cache,
    };
  }
  if (url.includes("/refresh")) {
    return { flightId: "iflg_1", acceptedTasks: [], cache };
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
