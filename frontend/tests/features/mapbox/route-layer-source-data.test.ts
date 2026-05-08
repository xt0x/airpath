import { describe, expect, it } from "vitest";

import type { FlightDetail, FlightMapDataResponse, GeoJSONFeature } from "@/features/flights/types";

import { routeLayerSourceData } from "@/features/mapbox/lib/route-layer-source-data";

describe("route layer source data", () => {
  it("builds source feature collections for exact planned routes, actual tracks, and endpoints", () => {
    expect(
      routeLayerSourceData(
        mapData({
          planned: {
            source: "flightaware_route",
            available: true,
            geojson: lineFeature([
              [139.7, 35.6],
              [-73.8, 40.6],
            ]),
          },
          actual: {
            source: "flightaware_track",
            available: true,
            geojson: lineFeature([
              [139.7, 35.6],
              [140.1, 35.9],
            ]),
          },
          current: {
            source: "flightaware_position",
            available: true,
            geojson: pointFeature([140.2, 36]),
          },
        }),
        flightDetail(),
      ),
    ).toEqual({
      actualTrack: {
        type: "FeatureCollection",
        features: [
          lineFeature([
            [139.7, 35.6],
            [140.1, 35.9],
            [140.2, 36],
          ]),
        ],
      },
      plannedRoute: {
        type: "FeatureCollection",
        features: [
          lineFeature([
            [139.7, 35.6],
            [-73.8, 40.6],
          ]),
        ],
      },
      routeEndpoints: {
        type: "FeatureCollection",
        features: [
          {
            type: "Feature",
            geometry: { type: "Point", coordinates: [139.7, 35.6] },
            properties: { code: "RJTT", role: "origin" },
          },
          {
            type: "Feature",
            geometry: { type: "Point", coordinates: [-73.8, 40.6] },
            properties: { code: "KJFK", role: "destination" },
          },
        ],
      },
    });
  });

  it("clears unavailable and fallback route data before it reaches Mapbox sources", () => {
    expect(
      routeLayerSourceData(
        mapData({
          planned: {
            source: "airport_great_circle_fallback",
            available: true,
            geojson: lineFeature([
              [139.7, 35.6],
              [-73.8, 40.6],
            ]),
          },
          actual: { source: "flightaware_track", available: false, geojson: lineFeature([]) },
        }),
        flightDetail(),
      ),
    ).toEqual({
      actualTrack: emptyFeatureCollection(),
      plannedRoute: emptyFeatureCollection(),
      routeEndpoints: emptyFeatureCollection(),
    });
  });

  it("clears non-line route features before they reach Mapbox line sources", () => {
    expect(
      routeLayerSourceData(
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
            geojson: pointFeature([140.2, 36]),
          },
        }),
        flightDetail(),
      ),
    ).toEqual({
      actualTrack: emptyFeatureCollection(),
      plannedRoute: emptyFeatureCollection(),
      routeEndpoints: emptyFeatureCollection(),
    });
  });

  it("keeps MultiLineString actual tracks as route line source data", () => {
    expect(
      routeLayerSourceData(
        mapData({
          actual: {
            source: "flightaware_track",
            available: true,
            geojson: multiLineFeature([
              [
                [139.7, 35.6],
                [139.8, 35.7],
              ],
              [
                [139.9, 35.8],
                [140.1, 35.9],
              ],
            ]),
          },
        }),
        flightDetail(),
      ).actualTrack,
    ).toEqual({
      type: "FeatureCollection",
      features: [
        multiLineFeature([
          [
            [139.7, 35.6],
            [139.8, 35.7],
          ],
          [
            [139.9, 35.8],
            [140.1, 35.9],
          ],
        ]),
      ],
    });
  });

  it("does not extend the actual track with an unavailable current-position feature", () => {
    expect(
      routeLayerSourceData(
        mapData({
          actual: {
            source: "flightaware_track",
            available: true,
            geojson: lineFeature([
              [139.7, 35.6],
              [140.1, 35.9],
            ]),
          },
          current: {
            source: "flightaware_position",
            available: false,
            geojson: pointFeature([141, 36.2]),
          },
        }),
        flightDetail(),
      ).actualTrack,
    ).toEqual({
      type: "FeatureCollection",
      features: [
        lineFeature([
          [139.7, 35.6],
          [140.1, 35.9],
        ]),
      ],
    });
  });

  it("writes actual track source data after discarding invalid coordinates and appending current position", () => {
    expect(
      routeLayerSourceData(
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
            geojson: pointFeature([140.2, 36]),
          },
        }),
        flightDetail(),
      ).actualTrack,
    ).toEqual({
      type: "FeatureCollection",
      features: [
        lineFeature([
          [139.7, 35.6],
          [140.2, 36],
        ]),
      ],
    });
  });
});

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

function flightDetail(): FlightDetail {
  return {
    flightId: "iflg_1",
    flightIdType: "internal",
    provisionalFlightLegId: null,
    faFlightId: "fa_1",
    ident: "ANA110",
    identIata: "NH110",
    aircraftType: "B77W",
    registration: "JA777A",
    origin: { code: "RJTT", name: "Tokyo Haneda", timezone: "Asia/Tokyo" },
    destination: { code: "KJFK", name: "New York JFK", timezone: "America/New_York" },
    legIndex: 0,
    status: "En Route",
    progressPercent: 48,
    times: {
      scheduledOut: "2026-04-29T00:00:00Z",
      estimatedOut: null,
      actualOut: null,
      scheduledOff: null,
      estimatedOff: null,
      actualOff: null,
      scheduledOn: null,
      estimatedOn: null,
      actualOn: null,
      scheduledIn: null,
      estimatedIn: null,
      actualIn: null,
    },
  };
}

function emptyFeatureCollection(): GeoJSON.FeatureCollection {
  return { type: "FeatureCollection", features: [] };
}
