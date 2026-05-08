import { describe, expect, it } from "vitest";

import type { FlightMapDataResponse, GeoJSONFeature } from "@/features/flights/types";

import { boundsFromMapData } from "@/features/mapbox/lib/map-camera";

describe("map camera", () => {
  it("returns null when map data or Mapbox GL is unavailable", () => {
    expect(boundsFromMapData(null, fakeMapboxGL())).toBeNull();
    expect(boundsFromMapData(mapData(), null)).toBeNull();
  });

  it("returns null when no finite route coordinates are available", () => {
    expect(
      boundsFromMapData(
        mapData({
          planned: { source: "flightaware_route", available: true, geojson: null },
          actual: { source: "flightaware_track", available: true, geojson: null },
          current: { source: "flightaware_position", available: true, geojson: null },
        }),
        fakeMapboxGL(),
      ),
    ).toBeNull();
  });

  it("builds wrapped bounds from planned, actual, and current route coordinates", () => {
    const bounds = boundsFromMapData(
      mapData({
        planned: {
          source: "flightaware_route",
          available: true,
          geojson: lineFeature([
            [139.7, 35.6],
            [170.1, 48.2],
          ]),
        },
        actual: {
          source: "flightaware_track",
          available: true,
          geojson: multiLineFeature([
            [
              [-120.3, 51.2],
              [-90.4, 45.1],
            ],
          ]),
        },
        current: {
          source: "flightaware_position",
          available: true,
          geojson: pointFeature([-73.8, 40.6]),
        },
      }),
      fakeMapboxGL(),
    );

    expect(bounds).toEqual({
      extended: [
        [139.7, 35.6],
        [170.1, 48.2],
        [239.7, 51.2],
        [269.6, 45.1],
        [286.2, 40.6],
      ],
      first: [139.7, 35.6],
    });
  });

  it("keeps antimeridian-crossing route focus bounds on the shorter wrapped span", () => {
    const bounds = boundsFromMapData(
      mapData({
        planned: {
          source: "flightaware_route",
          available: true,
          geojson: lineFeature([
            [170, 35.6],
            [-170, 40.6],
          ]),
        },
      }),
      fakeMapboxGL(),
    );

    expect(bounds).toEqual({
      extended: [
        [170, 35.6],
        [190, 40.6],
      ],
      first: [170, 35.6],
    });
  });

  it("ignores unavailable and fallback map layers when calculating route focus bounds", () => {
    expect(
      boundsFromMapData(
        mapData({
          planned: {
            source: "airport_great_circle_fallback",
            available: true,
            geojson: lineFeature([
              [139.7, 35.6],
              [-73.8, 40.6],
            ]),
          },
          actual: {
            source: "flightaware_track",
            available: false,
            geojson: lineFeature([
              [140.1, 35.9],
              [141.2, 36.4],
            ]),
          },
          current: {
            source: "flightaware_position",
            available: false,
            geojson: pointFeature([142.4, 37.1]),
          },
        }),
        fakeMapboxGL(),
      ),
    ).toBeNull();
  });

  it("uses available current-position GeoJSON for route focus when route lines are unavailable", () => {
    const bounds = boundsFromMapData(
      mapData({
        current: {
          source: "flightaware_position",
          available: true,
          geojson: pointFeature([142.4, 37.1]),
        },
      }),
      fakeMapboxGL(),
    );

    expect(bounds).toEqual({
      extended: [[142.4, 37.1]],
      first: [142.4, 37.1],
    });
  });

  it("builds bounds from the actual track after discarding invalid coordinates and appending current position", () => {
    const bounds = boundsFromMapData(
      mapData({
        actual: {
          source: "flightaware_track",
          available: true,
          geojson: lineFeature([
            [139.7, 35.6],
            [Number.NaN, 35.7],
          ]),
        },
        current: {
          source: "flightaware_position",
          available: true,
          geojson: pointFeature([140.2, 35.9]),
        },
      }),
      fakeMapboxGL(),
    );

    expect(bounds).toEqual({
      extended: [
        [139.7, 35.6],
        [140.2, 35.9],
      ],
      first: [139.7, 35.6],
    });
  });

  it("ignores non-line planned and actual route features when calculating route focus bounds", () => {
    const bounds = boundsFromMapData(
      mapData({
        planned: {
          source: "flightaware_route",
          available: true,
          geojson: pointFeature([139.7, 35.6]),
        },
        actual: {
          source: "flightaware_track",
          available: true,
          geojson: pointFeature([140.1, 35.9]),
        },
        current: {
          source: "flightaware_position",
          available: true,
          geojson: pointFeature([142.4, 37.1]),
        },
      }),
      fakeMapboxGL(),
    );

    expect(bounds).toEqual({
      extended: [[142.4, 37.1]],
      first: [142.4, 37.1],
    });
  });
});

function fakeMapboxGL() {
  return {
    LngLatBounds: class FakeLngLatBounds {
      first: [number, number];
      extended: [number, number][];

      constructor(first: [number, number]) {
        this.first = first;
        this.extended = [first];
      }

      extend(coordinate: [number, number]) {
        this.extended.push(coordinate);
      }
    },
  } as unknown as (typeof import("mapbox-gl"))["default"];
}

function lineFeature(coordinates: [number, number][]): GeoJSONFeature {
  return { type: "Feature", geometry: { type: "LineString", coordinates }, properties: {} };
}

function multiLineFeature(coordinates: [number, number][][]): GeoJSONFeature {
  return { type: "Feature", geometry: { type: "MultiLineString", coordinates }, properties: {} };
}

function pointFeature(coordinates: [number, number]): GeoJSONFeature {
  return { type: "Feature", geometry: { type: "Point", coordinates }, properties: {} };
}

function mapData(override: Partial<FlightMapDataResponse> = {}): FlightMapDataResponse {
  return {
    flightId: "iflg_1",
    faFlightId: "fa_1",
    planned: { source: "flightaware_route", available: false, geojson: null },
    actual: { source: "flightaware_track", available: false, geojson: null },
    current: { source: "flightaware_position", available: false, geojson: null },
    cache: {
      freshness: "fresh",
      source: "cache",
      stale: false,
      checkedAt: "2026-04-29T00:00:00Z",
    },
    ...override,
  };
}
