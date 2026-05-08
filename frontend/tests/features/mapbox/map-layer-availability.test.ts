import { describe, expect, it } from "vitest";

import type { FlightMapDataResponse, GeoJSONFeature, MapLayer } from "@/features/flights/types";

import {
  availableActualTrackFeature,
  availableCurrentPositionFeature,
  availableLayerFeature,
  flightAwarePlannedRouteFeature,
} from "@/features/mapbox/lib/map-layer-availability";

describe("map layer availability", () => {
  it("returns layer GeoJSON only when the layer is available", () => {
    const feature = lineFeature([
      [139.7, 35.6],
      [140.1, 35.9],
    ]);

    expect(availableLayerFeature(layer({ available: true, geojson: feature }))).toEqual(feature);
    expect(availableLayerFeature(layer({ available: false, geojson: feature }))).toBeNull();
    expect(availableLayerFeature(null)).toBeNull();
    expect(availableLayerFeature(undefined)).toBeNull();
  });

  it("accepts only exact FlightAware planned route geometry for planned-route display", () => {
    const feature = lineFeature([
      [139.7, 35.6],
      [-73.8, 40.6],
    ]);

    expect(
      flightAwarePlannedRouteFeature(
        mapData({
          planned: layer({ source: "flightaware_route", available: true, geojson: feature }),
        }),
      ),
    ).toEqual(feature);
    expect(
      flightAwarePlannedRouteFeature(
        mapData({
          planned: layer({
            source: "airport_great_circle_fallback",
            available: true,
            geojson: feature,
          }),
        }),
      ),
    ).toBeNull();
  });

  it("exposes available actual-track and current-position features", () => {
    const actual = lineFeature([
      [139.7, 35.6],
      [140.1, 35.9],
    ]);
    const current = pointFeature([140.2, 36]);
    const data = mapData({
      actual: layer({ source: "flightaware_track", available: true, geojson: actual }),
      current: layer({ source: "flightaware_position", available: true, geojson: current }),
    });

    expect(availableActualTrackFeature(data)).toEqual(actual);
    expect(availableCurrentPositionFeature(data)).toEqual(current);
  });
});

function layer(override: Partial<MapLayer> = {}): MapLayer {
  return {
    source: "flightaware_route",
    available: false,
    geojson: null,
    ...override,
  };
}

function lineFeature(coordinates: [number, number][]): GeoJSONFeature {
  return { type: "Feature", geometry: { type: "LineString", coordinates }, properties: {} };
}

function pointFeature(coordinates: [number, number]): GeoJSONFeature {
  return { type: "Feature", geometry: { type: "Point", coordinates }, properties: {} };
}

function mapData(override: Partial<FlightMapDataResponse> = {}): FlightMapDataResponse {
  return {
    flightId: "iflg_1",
    faFlightId: "fa_1",
    planned: layer(),
    actual: layer({ source: "flightaware_track" }),
    current: layer({ source: "flightaware_position" }),
    cache: {
      freshness: "fresh",
      source: "cache",
      stale: false,
      checkedAt: "2026-04-29T00:00:00Z",
    },
    ...override,
  };
}
