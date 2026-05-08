import { describe, expect, it } from "vitest";

import {
  collectFiniteCoordinatePairs,
  coordinatePairFromValue,
} from "@/features/mapbox/lib/geojson-coordinates";

describe("geojson coordinates", () => {
  it("reads a finite longitude latitude pair from array-like GeoJSON coordinates", () => {
    expect(coordinatePairFromValue([139.7, 35.6])).toEqual([139.7, 35.6]);
    expect(coordinatePairFromValue([139.7, 35.6, 1200])).toEqual([139.7, 35.6]);
  });

  it("rejects missing or non-finite coordinate pairs", () => {
    expect(coordinatePairFromValue([139.7])).toBeNull();
    expect(coordinatePairFromValue([139.7, Number.NaN])).toBeNull();
    expect(coordinatePairFromValue(["139.7", 35.6])).toBeNull();
    expect(coordinatePairFromValue(null)).toBeNull();
  });

  it("rejects coordinates outside Mapbox longitude and latitude bounds", () => {
    expect(coordinatePairFromValue([181, 35.6])).toBeNull();
    expect(coordinatePairFromValue([-181, 35.6])).toBeNull();
    expect(coordinatePairFromValue([139.7, 91])).toBeNull();
    expect(coordinatePairFromValue([139.7, -91])).toBeNull();
  });

  it("collects finite coordinate pairs from nested GeoJSON coordinate arrays", () => {
    expect(
      collectFiniteCoordinatePairs([
        [
          [139.7, 35.6],
          [140.2, 35.9],
        ],
        [[-73.8, 40.6]],
      ]),
    ).toEqual([
      [139.7, 35.6],
      [140.2, 35.9],
      [-73.8, 40.6],
    ]);
  });
});
