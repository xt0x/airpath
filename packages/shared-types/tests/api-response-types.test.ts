import { describe, expect, it } from "vitest";

import type {
  Airport,
  AirportBoardResponse,
  FlightDetailResponse,
  FlightMapDataResponse,
  FlightPositionsResponse,
  FlightTimes,
  FlightSearchResponse,
  UsageStatusResponse,
} from "../src/api-contract.js";
import type { Airport as DomainAirport, FlightTimes as DomainFlightTimes } from "../src/index.js";

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

    const departures = {
      airportCode: "RJTT",
      direction: "departures",
      date: "2026-05-04",
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
          scheduledOut: "2026-05-04T01:00:00Z",
          legIndex: 0,
          status: "Scheduled",
        },
      ],
      cache,
    } satisfies AirportBoardResponse;

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

    const detail = {
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
        destination: { code: "KJFK", name: "John F. Kennedy", timezone: "America/New_York" },
        legIndex: null,
        status: "En Route",
        progressPercent: 62,
        times: {
          scheduledOut: null,
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
      route: mapData.planned,
      track: mapData.actual,
      current: null,
      cache,
    } satisfies FlightDetailResponse;

    const positions = {
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
    } satisfies FlightPositionsResponse;

    const usage = {
      budget: {
        environment: "dev",
        month: "2026-04",
        currency: "USD",
        estimatedMonthToDateCost: 1,
        softStopThreshold: 4,
        stopped: false,
        dailyUsage: [{ date: "2026-04-29", estimatedCostUSD: 0.25, estimatedCallCount: 25 }],
      },
      rateLimit: {
        limited: false,
        resetAt: null,
      },
      fetchingEnabled: true,
      cache,
    } satisfies UsageStatusResponse;

    expect(search.items).toEqual([]);
    expect(departures.direction).toBe("departures");
    expect(detail.route.source).toBe("flightaware_route");
    expect(mapData.current.source).toBe("flightaware_position");
    expect(positions.items[0]?.altitudeFeet).toBe(37000);
    expect(usage.budget.currency).toBe("USD");
    expect(usage.budget.dailyUsage[0]?.estimatedCallCount).toBe(25);
  });

  it("derives duplicated API response value objects from the shared domain model", () => {
    const airport = {
      code: "RJTT",
      name: "Tokyo Haneda",
      timezone: "Asia/Tokyo",
    } satisfies Airport satisfies DomainAirport;

    const times = {
      scheduledOut: null,
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
    } satisfies FlightTimes satisfies DomainFlightTimes;

    expect(airport.code).toBe("RJTT");
    expect(times.actualIn).toBeNull();
  });
});
