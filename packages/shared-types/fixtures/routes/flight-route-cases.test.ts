import { describe, expect, it } from "vitest";

import routeCases from "./flight-route-cases.json";

describe("flight route response fixtures", () => {
  it("uses a descriptive fixture name instead of an implementation-plan item number", () => {
    expect(routeCases.source.fixtureName).toBe("flight-route-response-cases");
    expect(routeCases.source).not.toHaveProperty("planItem");
  });

  it("captures a decoded /route response with coordinates", () => {
    const decoded = routeCases.cases.find(
      (routeCase) => routeCase.kind === "decoded_with_coordinates",
    );

    expect(decoded?.source.endpoint).toBe("GET /flights/{id}/route");
    expect(decoded?.routeResponse.status).toBe(200);
    expect(decoded?.routeResponse.body.fixes.length).toBeGreaterThan(2);
    expect(decoded?.routeResponse.body.fixes.every((fix) => typeof fix.latitude === "number")).toBe(
      true,
    );
    expect(
      decoded?.routeResponse.body.fixes.every((fix) => typeof fix.longitude === "number"),
    ).toBe(true);
  });

  it("captures the route-string-only fallback when decoded coordinates are not available", () => {
    const stringOnly = routeCases.cases.find((routeCase) => routeCase.kind === "route_string_only");

    expect(stringOnly?.flightSummary.route).toBe("ANON1 ANON2 ANON3 ANON4");
    expect(stringOnly?.routeResponse.status).toBe(200);
    expect(stringOnly?.routeResponse.body.fixes).toEqual([
      {
        name: "ANON1",
        latitude: null,
        longitude: null,
        distance_from_origin: null,
        distance_this_leg: null,
        distance_to_destination: null,
        outbound_course: null,
        type: "UNKNOWN",
      },
    ]);
  });

  it("captures a route-unavailable response separately from a route string fallback", () => {
    const unavailable = routeCases.cases.find((routeCase) => routeCase.kind === "unavailable");

    expect(unavailable?.routeResponse.status).toBe(404);
    expect(unavailable?.routeResponse.body).toMatchObject({
      title: "Not Found",
      reason: "NO_ROUTE",
      status: 404,
    });
    expect(unavailable?.flightSummary.route).toBeNull();
  });

  it("keeps fixture data anonymized", () => {
    const serialized = JSON.stringify(routeCases);

    expect(serialized).not.toMatch(/[A-Z]{3}\d{1,4}-\d{8,}/);
    expect(serialized).not.toMatch(/UAL|ANA|JAL|DAL|AAL|N\d{2,}/);
  });
});
