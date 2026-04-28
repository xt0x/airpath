import { describe, expect, it } from "vitest";

import trackCases from "./flight-track-cases.json";

describe("flight track response fixtures", () => {
  it("captures a /track response with an ordered position list", () => {
    const currentTrack = trackCases.cases.find((trackCase) => trackCase.kind === "current_track");

    expect(trackCases.source.fixtureName).toBe("flight-track-response-cases");
    expect(trackCases.source).not.toHaveProperty("planItem");
    expect(currentTrack?.source.endpoint).toBe("GET /flights/{id}/track");
    expect(currentTrack?.trackResponse.status).toBe(200);
    expect(currentTrack?.trackResponse.body.positions.length).toBeGreaterThan(2);
    expect(currentTrack?.trackResponse.body.positions.map((position) => position.timestamp)).toEqual(
      [
        "2026-06-20T08:30:00Z",
        "2026-06-20T09:00:00Z",
        "2026-06-20T09:30:00Z",
      ],
    );
  });

  it("keeps coordinate-bearing track points displayable", () => {
    const currentTrack = trackCases.cases.find((trackCase) => trackCase.kind === "current_track");

    expect(
      currentTrack?.trackResponse.body.positions.every(
        (position) =>
          typeof position.latitude === "number" && typeof position.longitude === "number",
      ),
    ).toBe(true);
  });

  it("captures unavailable track responses separately", () => {
    const unavailable = trackCases.cases.find((trackCase) => trackCase.kind === "unavailable");

    expect(unavailable?.trackResponse.status).toBe(404);
    expect(unavailable?.trackResponse.body).toMatchObject({
      title: "Not Found",
      reason: "NO_TRACK",
      status: 404,
    });
  });
});
