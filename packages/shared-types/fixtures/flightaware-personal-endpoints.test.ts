import { describe, expect, it } from "vitest";

import endpointTable from "./flightaware-personal-endpoints.json";

describe("FlightAware Personal endpoint table", () => {
  it("uses a descriptive fixture name instead of an implementation-plan item number", () => {
    expect(endpointTable.source.fixtureName).toBe("flightaware-personal-endpoint-table");
    expect(endpointTable.source).not.toHaveProperty("planItem");
  });

  it("covers every endpoint needed by the free-allowance MVP", () => {
    expect(endpointTable.endpoints.map((endpoint) => endpoint.endpoint)).toEqual([
      "GET /flights/{ident}",
      "GET /flights/{id}/position",
      "GET /flights/{id}/route",
      "GET /flights/{id}/track",
      "GET /schedules/{date_start}/{date_end}",
      "GET /account/usage",
    ]);
  });

  it("marks paid-plan-only endpoints outside the MVP", () => {
    expect(endpointTable.excludedEndpointFamilies).toEqual([
      "FlightAware Alerts",
      "Historical Flight Data",
      "Foresight",
      "Aireon",
    ]);
  });

  it("records the free-allowance guardrails that affect implementation", () => {
    expect(endpointTable.guardrails.defaultMaxPages).toBe(1);
    expect(endpointTable.guardrails.realCallTests).toBe("opt-in");
    expect(endpointTable.guardrails.browserAccess).toBe("forbidden");
    expect(endpointTable.guardrails.backendAccess).toBe("required");
  });
});
