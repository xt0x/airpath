import { describe, expect, it } from "vitest";

import type { GeoJSONFeature } from "@/features/flights/types";

import {
  drawableRouteLineFeature,
  drawableRouteLineFeatureWithAppendedCoordinate,
} from "@/features/mapbox/lib/route-line-feature";

describe("route line feature", () => {
  it("returns LineString features with finite display coordinates", () => {
    const feature = lineFeature([
      [139.7, 35.6],
      [Number.NaN, 36],
      [140.1, 35.9],
    ]);

    expect(drawableRouteLineFeature(feature)).toEqual(
      lineFeature([
        [139.7, 35.6],
        [140.1, 35.9],
      ]),
    );
  });

  it("returns MultiLineString features with only drawable finite-coordinate segments", () => {
    const feature = multiLineFeature([
      [[Number.NaN, 35.6]],
      [
        [139.7, 35.6],
        [Infinity, 36],
        [140.1, 35.9],
      ],
    ]);

    expect(drawableRouteLineFeature(feature)).toEqual(
      multiLineFeature([
        [
          [139.7, 35.6],
          [140.1, 35.9],
        ],
      ]),
    );
  });

  it("rejects point features, null features, and line features without two finite coordinates", () => {
    expect(drawableRouteLineFeature(pointFeature([139.7, 35.6]))).toBeNull();
    expect(drawableRouteLineFeature(null)).toBeNull();
    expect(drawableRouteLineFeature(lineFeature([[139.7, 35.6]]))).toBeNull();
    expect(
      drawableRouteLineFeature(
        lineFeature([
          [139.7, 35.6],
          [Number.NaN, 35.9],
        ]),
      ),
    ).toBeNull();
  });

  it("returns a drawable LineString after appending a coordinate to one finite track point", () => {
    expect(
      drawableRouteLineFeatureWithAppendedCoordinate(
        lineFeature([
          [139.7, 35.6],
          [Number.NaN, 35.7],
        ]),
        [140.2, 35.9],
      ),
    ).toEqual(
      lineFeature([
        [139.7, 35.6],
        [140.2, 35.9],
      ]),
    );
  });

  it("rejects appended coordinates that Mapbox cannot draw", () => {
    expect(
      drawableRouteLineFeatureWithAppendedCoordinate(lineFeature([[139.7, 35.6]]), [
        Number.NaN,
        35.9,
      ]),
    ).toBeNull();
    expect(
      drawableRouteLineFeatureWithAppendedCoordinate(
        lineFeature([
          [139.7, 35.6],
          [140.1, 35.8],
        ]),
        [181, 35.9],
      ),
    ).toBeNull();
  });

  it("appends to the last drawable MultiLineString segment without duplicating its endpoint", () => {
    expect(
      drawableRouteLineFeatureWithAppendedCoordinate(
        multiLineFeature([
          [
            [139.7, 35.6],
            [139.8, 35.7],
          ],
          [
            [140.1, 35.8],
            [140.2, 35.9],
          ],
        ]),
        [140.2, 35.9],
      ),
    ).toEqual(
      multiLineFeature([
        [
          [139.7, 35.6],
          [139.8, 35.7],
        ],
        [
          [140.1, 35.8],
          [140.2, 35.9],
        ],
      ]),
    );
  });
});

function lineFeature(coordinates: unknown[]): GeoJSONFeature {
  return { type: "Feature", geometry: { type: "LineString", coordinates }, properties: {} };
}

function multiLineFeature(coordinates: unknown[]): GeoJSONFeature {
  return { type: "Feature", geometry: { type: "MultiLineString", coordinates }, properties: {} };
}

function pointFeature(coordinates: [number, number]): GeoJSONFeature {
  return { type: "Feature", geometry: { type: "Point", coordinates }, properties: {} };
}
