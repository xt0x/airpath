import { describe, expect, it } from "vitest";

import {
  formatAircraftAltitude,
  formatAircraftSpeed,
  formatAircraftTimestamp,
} from "@/features/mapbox/lib/aircraft-hover-card-formatters";

describe("aircraft hover-card formatters", () => {
  it("formats present aircraft position metrics with display units", () => {
    expect(formatAircraftAltitude(37000)).toBe("37,000 ft");
    expect(formatAircraftSpeed(488)).toBe("488 kt");
    expect(formatAircraftTimestamp("2026-04-29T00:05:00Z")).not.toContain("T00:05:00Z");
  });

  it("labels missing aircraft position metrics as unavailable", () => {
    expect(formatAircraftAltitude(null)).toBe("Unavailable");
    expect(formatAircraftSpeed(null)).toBe("Unavailable");
  });
});
