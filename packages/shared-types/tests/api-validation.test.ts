import { describe, expect, it } from "vitest";

import { validateApiSchema } from "../src/api-contract.js";

describe("API schema validation", () => {
  it("validates successful response payloads from the shared contract schemas", () => {
    expect(
      validateApiSchema("FlightSearchResponse", {
        items: [],
        cache: validCache(),
      }),
    ).toEqual({ valid: true });
  });

  it("rejects payload values that violate shared enum and numeric ranges", () => {
    expect(
      validateApiSchema("FlightPositionsResponse", {
        flightId: "iflg_1",
        items: [
          {
            latitude: 45.1,
            longitude: 160.4,
            altitudeHundredsFeet: 370,
            altitudeFeet: 37000,
            altitudeChange: "level",
            groundspeedKnots: 488,
            headingDegrees: 360,
            timestamp: "2026-04-29T00:05:00Z",
            updateType: "estimated",
            source: "flightaware_position",
          },
        ],
        cache: validCache(),
      }),
    ).toMatchObject({ valid: false });

    expect(
      validateApiSchema("FlightMapDataResponse", {
        flightId: "iflg_1",
        faFlightId: "fa_1",
        planned: {
          source: "computed_route",
          available: false,
          geojson: null,
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
        cache: validCache(),
      }),
    ).toMatchObject({ valid: false });
  });

  it("rejects additional properties when the API contract closes an object shape", () => {
    expect(
      validateApiSchema("FlightSearchResponse", {
        items: [],
        cache: validCache(),
        diagnostics: {},
      }),
    ).toMatchObject({ valid: false });
  });

  it("rejects refresh response task metadata outside the shared fetch task schema", () => {
    expect(
      validateApiSchema("FlightRefreshResponse", {
        flightId: "iflg_1",
        acceptedTasks: [
          {
            schemaVersion: 1,
            taskId: "task_1",
            taskType: "position",
            flightId: "iflg_1",
            requestedAt: "2026-04-29T00:00:00Z",
            reason: "browser_debug_override",
            idempotencyKey: "refresh:iflg_1:position",
          },
        ],
        cache: validCache(),
      }),
    ).toMatchObject({ valid: false });
  });
});

function validCache() {
  return {
    freshness: "fresh",
    source: "cache",
    stale: false,
    checkedAt: "2026-04-29T00:00:00Z",
  };
}
