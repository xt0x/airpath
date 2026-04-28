import { describe, expect, it } from "vitest";

import {
  apiContract,
  apiErrorCodes,
  fetchTaskSchema,
  fetchTaskTypes,
  mvpExcludedContracts,
} from "./api-contract.js";

describe("F4 MVP API contract", () => {
  it("defines the HTTP API paths required by the free-allowance MVP", () => {
    expect(Object.keys(apiContract.paths)).toEqual([
      "/v1/flights/search",
      "/v1/flights/{flightId}",
      "/v1/flights/{flightId}/map-data",
      "/v1/flights/{flightId}/positions",
      "/v1/flights/{flightId}/refresh",
      "/v1/usage/status",
    ]);

    expect(apiContract.paths["/v1/flights/search"]).toHaveProperty("get");
    expect(apiContract.paths["/v1/flights/{flightId}"]).toHaveProperty("get");
    expect(apiContract.paths["/v1/flights/{flightId}/map-data"]).toHaveProperty("get");
    expect(apiContract.paths["/v1/flights/{flightId}/positions"]).toHaveProperty("get");
    expect(apiContract.paths["/v1/flights/{flightId}/refresh"]).toHaveProperty("post");
    expect(apiContract.paths["/v1/usage/status"]).toHaveProperty("get");
  });

  it("wraps every success response with cache metadata", () => {
    const successResponseSchemas = [
      "FlightSearchResponse",
      "FlightDetailResponse",
      "FlightMapDataResponse",
      "FlightPositionsResponse",
      "FlightRefreshResponse",
      "UsageStatusResponse",
    ] as const;

    for (const schemaName of successResponseSchemas) {
      const schema = apiContract.components.schemas[schemaName];

      expect(schema.required).toContain("cache");
      expect(schema.properties).toHaveProperty("cache");
      expect(schema.properties.cache).toEqual({ $ref: "#/components/schemas/CacheMetadata" });
    }

    expect(apiContract.components.schemas.CacheMetadata.properties).toMatchObject({
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
    });
  });

  it("defines typed error responses for budget, rate-limit, stale-cache, and upstream failures", () => {
    expect(apiErrorCodes).toEqual([
      "flightaware_budget_exceeded",
      "flightaware_rate_limited",
      "stale_cache_unavailable",
      "upstream_failure",
      "flightaware_fetch_disabled",
    ]);

    expect(apiContract.components.schemas.ApiError.properties.code.enum).toEqual(apiErrorCodes);

    const searchResponses = apiContract.paths["/v1/flights/search"].get.responses;
    expect(searchResponses).toHaveProperty("429");
    expect(searchResponses).toHaveProperty("502");
    expect(searchResponses).toHaveProperty("503");
  });

  it("limits FetchTask messages to the Personal MVP polling tasks", () => {
    expect(fetchTaskTypes).toEqual(["summary", "position", "route", "track", "final_track"]);
    expect(fetchTaskSchema.properties.taskType.enum).toEqual(fetchTaskTypes);
    expect(fetchTaskSchema.required).toEqual([
      "schemaVersion",
      "taskId",
      "taskType",
      "flightId",
      "requestedAt",
      "reason",
      "idempotencyKey",
    ]);
  });

  it("explicitly keeps WebSocket and FlightAware Alerts outside the MVP contract", () => {
    expect(mvpExcludedContracts).toEqual(["websocket", "flightaware_alerts"]);
    expect(Object.keys(apiContract.paths).join("\n")).not.toMatch(/realtime|websocket|alerts/i);
  });
});
