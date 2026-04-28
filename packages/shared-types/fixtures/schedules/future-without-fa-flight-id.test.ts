import { describe, expect, it } from "vitest";

import scheduleResponse from "./future-without-fa-flight-id.json";

describe("L3-02 future schedule fixture", () => {
  it("captures a future /schedules response where FlightAware has not issued fa_flight_id", () => {
    expect(scheduleResponse.source.endpoint).toBe("GET /schedules/{date_start}/{date_end}");
    expect(scheduleResponse.source.planItem).toBe("L3-02");
    expect(scheduleResponse.response.scheduled).toHaveLength(2);

    for (const flight of scheduleResponse.response.scheduled) {
      expect(flight.fa_flight_id).toBeNull();
      expect(flight.ident).toMatch(/^ANON/);
      expect(flight.scheduled_out).toMatch(/Z$/);
      expect(flight.scheduled_in).toMatch(/Z$/);
      expect(flight.origin).toMatch(/^A[A-Z0-9]{3}$/);
      expect(flight.destination).toMatch(/^B[A-Z0-9]{3}$/);
    }
  });

  it("keeps the fields needed to create provisionalFlightLegId before fa_flight_id exists", () => {
    const [primaryFlight] = scheduleResponse.response.scheduled;

    expect(primaryFlight).toMatchObject({
      ident: "ANON123",
      origin: "AAAA",
      origin_icao: "AAAA",
      origin_iata: "AAA",
      destination: "BBBB",
      destination_icao: "BBBB",
      destination_iata: "BBB",
      scheduled_out: "2026-06-20T08:15:00Z",
      fa_flight_id: null,
    });
  });

  it("keeps nullable codeshare and airport-code variants explicit", () => {
    const [primaryFlight, codeshareFlight] = scheduleResponse.response.scheduled;

    expect(primaryFlight.actual_ident).toBeNull();
    expect(primaryFlight.origin_lid).toBeNull();
    expect(primaryFlight.destination_lid).toBeNull();

    expect(codeshareFlight.actual_ident).toBe("ANON789");
    expect(codeshareFlight.actual_ident_icao).toBe("ANON789");
    expect(codeshareFlight.actual_ident_iata).toBe("ZZ789");
  });
});
