import { describe, expect, it } from "vitest";

import {
  apiContract,
  apiErrorCodes,
  apiRouteBuilders,
  apiRouteTemplates,
  fetchTaskSchema,
  fetchTaskTypes,
  mvpExcludedContracts,
  refreshTaskTypes,
} from "../src/api-contract.js";

describe("MVP API contract", () => {
  it("defines the HTTP API paths required by the free-allowance MVP", () => {
    const expectedPaths = [
      "/v1/airports/{airportCode}/departures",
      "/v1/airports/{airportCode}/arrivals",
      "/v1/flights/search",
      "/v1/flights/{flightId}",
      "/v1/flights/{flightId}/map-data",
      "/v1/flights/{flightId}/positions",
      "/v1/flights/{flightId}/refresh",
      "/v1/usage/status",
    ];

    expect(Object.keys(apiContract.paths)).toEqual(expectedPaths);
    expect(Object.values(apiRouteTemplates)).toEqual(expectedPaths);

    expect(apiContract.paths["/v1/airports/{airportCode}/departures"]).toHaveProperty("get");
    expect(apiContract.paths["/v1/airports/{airportCode}/arrivals"]).toHaveProperty("get");
    expect(apiContract.paths["/v1/flights/search"]).toHaveProperty("get");
    expect(apiContract.paths["/v1/flights/{flightId}"]).toHaveProperty("get");
    expect(apiContract.paths["/v1/flights/{flightId}/map-data"]).toHaveProperty("get");
    expect(apiContract.paths["/v1/flights/{flightId}/positions"]).toHaveProperty("get");
    expect(apiContract.paths["/v1/flights/{flightId}/refresh"]).toHaveProperty("post");
    expect(apiContract.paths["/v1/usage/status"]).toHaveProperty("get");

    const departureParameters =
      apiContract.paths["/v1/airports/{airportCode}/departures"].get.parameters;
    expect(departureParameters.map((parameter) => parameter.name)).toEqual(["airportCode", "date"]);

    const searchParameters = apiContract.paths["/v1/flights/search"].get.parameters;
    expect(searchParameters.map((parameter) => parameter.name)).toEqual(["ident"]);

    const mapDataParameters = apiContract.paths["/v1/flights/{flightId}/map-data"].get.parameters;
    expect(mapDataParameters.map((parameter) => parameter.name)).toEqual(["flightId"]);

    const positionsParameters =
      apiContract.paths["/v1/flights/{flightId}/positions"].get.parameters;
    expect(positionsParameters.map((parameter) => parameter.name)).toEqual([
      "flightId",
      "since",
      "limit",
    ]);
  });

  it("builds concrete API routes from the shared route templates", () => {
    expect(apiRouteBuilders.airportDepartures("rjtt", { date: "2026-05-04" })).toBe(
      "/v1/airports/RJTT/departures?date=2026-05-04",
    );
    expect(apiRouteBuilders.airportArrivals("kjfk", { date: "2026-05-04" })).toBe(
      "/v1/airports/KJFK/arrivals?date=2026-05-04",
    );
    expect(apiRouteBuilders.searchFlights("ANA110")).toBe("/v1/flights/search?ident=ANA110");
    expect(apiRouteBuilders.flightDetail("iflg_1/segment")).toBe("/v1/flights/iflg_1%2Fsegment");
    expect(apiRouteBuilders.flightMapData("iflg_1")).toBe("/v1/flights/iflg_1/map-data");
    expect(apiRouteBuilders.flightPositions("iflg_1", { since: "2026-04-29T00:00:00Z" })).toBe(
      "/v1/flights/iflg_1/positions?since=2026-04-29T00%3A00%3A00Z",
    );
    expect(apiRouteBuilders.flightRefresh("iflg_1")).toBe("/v1/flights/iflg_1/refresh");
    expect(apiRouteBuilders.usageStatus()).toBe("/v1/usage/status");
  });

  it("wraps every success response with cache metadata", () => {
    const successResponseSchemas = [
      "FlightSearchResponse",
      "AirportBoardResponse",
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

  it("keeps flight detail schema aligned with the backend response shape", () => {
    const schema = apiContract.components.schemas.FlightDetailResponse;

    expect(schema.required).toEqual(["flight", "route", "track", "current", "cache"]);
    expect(schema.properties).toMatchObject({
      flight: { $ref: "#/components/schemas/FlightDetail" },
      route: { $ref: "#/components/schemas/MapLayer" },
      track: { $ref: "#/components/schemas/MapLayer" },
      current: {
        anyOf: [{ $ref: "#/components/schemas/Position" }, { type: "null" }],
      },
      cache: { $ref: "#/components/schemas/CacheMetadata" },
    });
  });

  it("defines typed error responses for budget, rate-limit, stale-cache, and upstream failures", () => {
    expect(apiErrorCodes).toEqual([
      "validation_failed",
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
    expect(refreshTaskTypes).toEqual(["position", "route", "track", "final_track"]);
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

  it("exposes monthly usage budget scope in usage status", () => {
    const usageBudgetSchema = apiContract.components.schemas.UsageStatusResponse.properties.budget;

    expect(usageBudgetSchema.required).toEqual([
      "environment",
      "month",
      "currency",
      "estimatedMonthToDateCost",
      "softStopThreshold",
      "stopped",
      "dailyUsage",
    ]);
    expect(usageBudgetSchema.properties).toMatchObject({
      environment: { type: "string", minLength: 1 },
      month: { type: "string", pattern: "^\\d{4}-\\d{2}$" },
      dailyUsage: {
        type: "array",
        items: {
          type: "object",
          additionalProperties: false,
          required: ["date", "estimatedCostUSD", "estimatedCallCount"],
          properties: {
            date: { type: "string", format: "date" },
            estimatedCostUSD: { type: "number", minimum: 0 },
            estimatedCallCount: { type: "integer", minimum: 0 },
          },
        },
      },
    });
  });

  it("explicitly keeps WebSocket and FlightAware Alerts outside the MVP contract", () => {
    expect(mvpExcludedContracts).toEqual(["websocket", "flightaware_alerts"]);
    expect(Object.keys(apiContract.paths).join("\n")).not.toMatch(/realtime|websocket|alerts/i);
  });
});
