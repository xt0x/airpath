import { describe, expect, it } from "vitest";

import { buildDeckGlLayers, mapboxStyleURL } from "./index.js";

describe("Mapbox/deck.gl layer adapter", () => {
  it("builds planned route, actual track, and current-position deck.gl layers", () => {
    const layers = buildDeckGlLayers({
      flightId: "iflg_1",
      faFlightId: "fa_1",
      planned: lineLayer("flightaware_route"),
      actual: lineLayer("flightaware_track"),
      current: {
        source: "flightaware_position",
        available: true,
        geojson: {
          type: "Feature",
          geometry: { type: "Point", coordinates: [139.7, 35.6] },
          properties: { timestamp: "2026-04-29T00:00:00Z" },
        },
      },
      cache: {
        freshness: "fresh",
        source: "cache",
        stale: false,
        checkedAt: "2026-04-29T00:00:00Z",
      },
    });

    expect(layers.map((layer) => layer.id)).toEqual([
      "planned-route",
      "actual-track",
      "current-position",
    ]);
  });

  it("pins the Mapbox style used by the MVP map", () => {
    expect(mapboxStyleURL()).toBe("mapbox://styles/mapbox/light-v11");
  });
});

function lineLayer(source: "flightaware_route" | "flightaware_track") {
  return {
    source,
    available: true,
    geojson: {
      type: "Feature" as const,
      geometry: {
        type: "LineString" as const,
        coordinates: [
          [139.7, 35.6],
          [-73.8, 40.6],
        ],
      },
      properties: {},
    },
  };
}
