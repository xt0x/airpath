import { describe, expect, it } from "vitest";

import {
  firstOrderedSearchResultFlightId,
  sortFlightSearchResponseByScheduledOut,
} from "@/features/map-workspace/lib/flight-search-selection";
import type { FlightSearchResponse } from "@/features/flights/types";

describe("flight search selection", () => {
  it("selects the first already ordered flight-number search result for route loading", () => {
    expect(
      firstOrderedSearchResultFlightId(
        flightSearchResponse([
          { flightId: "iflg_first", ident: "ANA110", scheduledOut: "2026-05-04T10:00:00Z" },
          { flightId: "iflg_second", ident: "ANA110A", scheduledOut: "2026-05-04T08:00:00Z" },
        ]),
      ),
    ).toBe("iflg_first");
  });

  it("returns null when a flight-number search finds no flights", () => {
    expect(firstOrderedSearchResultFlightId(flightSearchResponse([]))).toBeNull();
  });

  it("sorts flight-number search results by scheduledOut before selection", () => {
    const response = flightSearchResponse([
      { flightId: "iflg_missing", ident: "ANA110", scheduledOut: null },
      { flightId: "iflg_later", ident: "ANA110", scheduledOut: "2026-05-04T10:00:00Z" },
      { flightId: "iflg_earlier", ident: "ANA110", scheduledOut: "2026-05-04T08:00:00Z" },
    ]);

    expect(
      sortFlightSearchResponseByScheduledOut(response).items.map((item) => item.flightId),
    ).toEqual(["iflg_earlier", "iflg_later", "iflg_missing"]);
  });

  it("keeps invalid scheduledOut values after valid times without mutating the response", () => {
    const response = flightSearchResponse([
      { flightId: "iflg_invalid", ident: "ANA110", scheduledOut: "not-a-date" },
      { flightId: "iflg_valid", ident: "ANA110", scheduledOut: "2026-05-04T08:00:00Z" },
    ]);

    const sorted = sortFlightSearchResponseByScheduledOut(response);

    expect(sorted.items.map((item) => item.flightId)).toEqual(["iflg_valid", "iflg_invalid"]);
    expect(response.items.map((item) => item.flightId)).toEqual(["iflg_invalid", "iflg_valid"]);
  });
});

function flightSearchResponse(
  items: Array<
    Pick<FlightSearchResponse["items"][number], "flightId" | "ident"> &
      Partial<Pick<FlightSearchResponse["items"][number], "scheduledOut">>
  >,
): FlightSearchResponse {
  return {
    items: items.map((item) => ({
      flightId: item.flightId,
      flightIdType: "internal",
      provisionalFlightLegId: null,
      faFlightId: "fa_1",
      ident: item.ident,
      identIata: null,
      origin: "RJTT",
      destination: "KJFK",
      scheduledOut: "scheduledOut" in item ? (item.scheduledOut ?? null) : "2026-05-04T01:00:00Z",
      legIndex: 0,
      status: "Scheduled",
    })),
    cache: {
      freshness: "fresh",
      source: "cache",
      stale: false,
      checkedAt: "2026-04-29T00:06:00Z",
      fetchedAt: "2026-04-29T00:05:00Z",
      staleAt: null,
      expiresAt: null,
    },
  };
}
