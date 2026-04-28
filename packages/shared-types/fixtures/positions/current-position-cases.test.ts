import { describe, expect, it } from "vitest";

import positionCases from "./current-position-cases.json";

describe("current position response fixtures", () => {
  it("captures a current /position response with displayable coordinates", () => {
    const current = positionCases.cases.find((positionCase) => positionCase.kind === "current");

    expect(positionCases.source.fixtureName).toBe("current-position-response-cases");
    expect(positionCases.source).not.toHaveProperty("planItem");
    expect(current?.source.endpoint).toBe("GET /flights/{id}/position");
    expect(current?.positionResponse.status).toBe(200);
    expect(typeof current?.positionResponse.body.latitude).toBe("number");
    expect(typeof current?.positionResponse.body.longitude).toBe("number");
    expect(current?.positionResponse.body.timestamp).toMatch(/Z$/);
  });

  it("captures FlightAware metric fields without converting units in the fixture", () => {
    const current = positionCases.cases.find((positionCase) => positionCase.kind === "current");

    expect(current?.positionResponse.body.altitude).toBe(370);
    expect(current?.positionResponse.body.groundspeed).toBe(488);
    expect(current?.positionResponse.body.heading).toBe(275);
  });

  it("captures unavailable current-position responses separately", () => {
    const unavailable = positionCases.cases.find(
      (positionCase) => positionCase.kind === "unavailable",
    );

    expect(unavailable?.positionResponse.status).toBe(404);
    expect(unavailable?.positionResponse.body).toMatchObject({
      title: "Not Found",
      reason: "NO_POSITION",
      status: 404,
    });
  });
});
