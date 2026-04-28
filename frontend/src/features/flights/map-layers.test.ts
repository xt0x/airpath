import { describe, expect, it } from "vitest";

import { buildDeckGlLayers, mapboxStyleURL } from "./map-layers";
import { sampleMapData } from "./sample-data";

describe("Mapbox/deck.gl layer adapter", () => {
  it("builds planned route, actual track, and current-position deck.gl layers", () => {
    const layers = buildDeckGlLayers(sampleMapData);

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
