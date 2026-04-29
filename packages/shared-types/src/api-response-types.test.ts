import { describe, expect, it } from "vitest";

import type {
  FlightMapDataResponse,
  FlightSearchResponse,
  UsageStatusResponse,
} from "./api-contract.js";

describe("API response types", () => {
  it("export source-of-truth TypeScript shapes for frontend API clients", () => {
    const cache = {
      freshness: "fresh",
      source: "cache",
      stale: false,
      checkedAt: "2026-04-29T00:00:00Z",
    } satisfies FlightSearchResponse["cache"];

    const search = {
      items: [],
      cache,
    } satisfies FlightSearchResponse;

    const mapData = {
      flightId: "iflg_1",
      faFlightId: "fa_1",
      planned: {
        source: "flightaware_route",
        available: false,
        geojson: null,
        unavailableReason: "not fetched",
      },
      actual: {
        source: "flightaware_track",
        available: false,
        geojson: null,
      },
      current: {
        source: "flightaware_position",
        available: false,
        geojson: null,
      },
      cache,
    } satisfies FlightMapDataResponse;

    const usage = {
      budget: {
        environment: "dev",
        month: "2026-04",
        currency: "USD",
        estimatedMonthToDateCost: 1,
        softStopThreshold: 4,
        stopped: false,
      },
      rateLimit: {
        limited: false,
        resetAt: null,
      },
      fetchingEnabled: true,
      cache,
    } satisfies UsageStatusResponse;

    expect(search.items).toEqual([]);
    expect(mapData.current.source).toBe("flightaware_position");
    expect(usage.budget.currency).toBe("USD");
  });
});
