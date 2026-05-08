import { describe, expect, it } from "vitest";

import {
  airportLabel,
  airportOptionByCode,
  formatDateOnly,
  parseDateOnly,
} from "@/features/map-workspace/lib/airport-board-controls";

describe("airport board controls", () => {
  it("resolves airport options by ICAO or IATA code", () => {
    expect(airportOptionByCode("RJTT")).toMatchObject({
      icao: "RJTT",
      iata: "HND",
      name: "Tokyo Haneda",
    });
    expect(airportOptionByCode("hnd")).toMatchObject({
      icao: "RJTT",
      iata: "HND",
    });
    expect(airportOptionByCode("XXXX")).toBeNull();
  });

  it("does not treat metropolitan area codes as airport selections", () => {
    expect(airportOptionByCode("TYO")).toBeNull();
  });

  it("formats known and unknown airport labels", () => {
    expect(airportLabel(airportOptionByCode("RJTT"), "RJTT")).toBe("Tokyo Haneda · HND / RJTT");
    expect(airportLabel(null, " kjfk ")).toBe("KJFK");
  });

  it("parses and formats board dates without timezone conversion", () => {
    const date = parseDateOnly("2026-05-05");

    expect(date).toEqual(new Date(2026, 4, 5));
    expect(formatDateOnly(new Date(2026, 4, 5))).toBe("2026-05-05");
    expect(parseDateOnly("bad-date")).toBeUndefined();
  });

  it("rejects malformed and impossible board dates", () => {
    expect(parseDateOnly("2026-02-31")).toBeUndefined();
    expect(parseDateOnly("2026-05-05-extra")).toBeUndefined();
    expect(parseDateOnly("2026-5-5")).toBeUndefined();
  });
});
