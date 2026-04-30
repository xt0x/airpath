import { describe, expect, it } from "vitest";

import searchResponse from "./flight-ident-search.json";

describe("flight ident search fixture", () => {
  it("captures a bounded /flights/{ident} response for the free-allowance MVP", () => {
    expect(searchResponse.source.fixtureName).toBe("flight-ident-search-response");
    expect(searchResponse.source.endpoint).toBe("GET /flights/{ident}");
    expect(searchResponse.source.query.max_pages).toBe(1);
    expect(searchResponse.source).not.toHaveProperty("planItem");
    expect(searchResponse.response.num_pages).toBe(1);
    expect(searchResponse.response.flights.length).toBeGreaterThan(0);
  });

  it("keeps the fields needed to create or resolve internal flight records", () => {
    const [flight] = searchResponse.response.flights;

    expect(flight).toMatchObject({
      fa_flight_id: "anon_search_current",
      ident: "ANON123",
      ident_iata: "ZZ123",
      origin: {
        code: "AAAA",
        code_icao: "AAAA",
        code_iata: "AAA",
      },
      destination: {
        code: "BBBB",
        code_icao: "BBBB",
        code_iata: "BBB",
      },
      scheduled_out: "2026-06-20T08:15:00Z",
      status: "En Route",
    });
  });

  it("keeps nullable FlightAware fields explicit", () => {
    const [, scheduledFlight] = searchResponse.response.flights;
    if (scheduledFlight === undefined) {
      throw new Error("search fixture requires a scheduled flight");
    }

    expect(scheduledFlight.fa_flight_id).toBeNull();
    expect(scheduledFlight.actual_out).toBeNull();
    expect(scheduledFlight.progress_percent).toBeNull();
  });

  it("keeps fixture data anonymized", () => {
    const serialized = JSON.stringify(searchResponse);

    expect(serialized).not.toMatch(/[A-Z]{3}\d{1,4}-\d{8,}/);
    expect(serialized).not.toMatch(/\b(UAL|ANA|JAL|DAL|AAL)\d{1,4}\b/);
    expect(serialized).not.toMatch(/\bN\d{2,}[A-Z0-9]*\b/);
  });
});
