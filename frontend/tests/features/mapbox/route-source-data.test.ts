import { describe, expect, it } from "vitest";

import type { FlightDetail, FlightMapDataResponse, GeoJSONFeature } from "@/features/flights/types";

import {
  actualTrackFeatureForDisplay,
  routeEndpointFeatureCollection,
} from "@/features/mapbox/lib/route-source-data";

describe("route source data", () => {
  it("extends the actual track display line to the current aircraft position", () => {
    const feature = actualTrackFeature();
    const current = currentPositionFeature([140.2, 35.9]);

    const result = actualTrackFeatureForDisplay(feature, current);

    expect(result?.geometry.coordinates).toEqual([
      [139.7, 35.6],
      [139.9, 35.7],
      [140.2, 35.9],
    ]);
    expect(result?.properties).toEqual(feature.properties);
  });

  it("does not duplicate the current aircraft position when it is already the track endpoint", () => {
    const feature = actualTrackFeature();

    const result = actualTrackFeatureForDisplay(feature, currentPositionFeature([139.9, 35.7]));

    expect(result?.geometry.coordinates).toEqual([
      [139.7, 35.6],
      [139.9, 35.7],
    ]);
  });

  it("extends the actual track after discarding invalid track coordinates", () => {
    const feature = actualTrackFeature({
      coordinates: [
        [139.7, 35.6],
        [Number.NaN, 35.7],
      ],
    });

    const result = actualTrackFeatureForDisplay(feature, currentPositionFeature([140.2, 35.9]));

    expect(result?.geometry.coordinates).toEqual([
      [139.7, 35.6],
      [140.2, 35.9],
    ]);
  });

  it("extends the last MultiLineString actual track segment to the current aircraft position", () => {
    const feature = actualTrackFeature({
      type: "MultiLineString",
      coordinates: [
        [
          [139.7, 35.6],
          [139.8, 35.65],
        ],
        [
          [139.9, 35.7],
          [140, 35.8],
        ],
      ],
    });

    const result = actualTrackFeatureForDisplay(feature, currentPositionFeature([140.2, 35.9]));

    expect(result?.geometry.coordinates).toEqual([
      [
        [139.7, 35.6],
        [139.8, 35.65],
      ],
      [
        [139.9, 35.7],
        [140, 35.8],
        [140.2, 35.9],
      ],
    ]);
  });

  it("returns null when actual track geometry is unavailable", () => {
    expect(actualTrackFeatureForDisplay(null, currentPositionFeature([140.2, 35.9]))).toBeNull();
  });

  it("returns a drawable actual track line when current position geometry is unavailable", () => {
    const feature = actualTrackFeature({
      coordinates: [
        [139.7, 35.6],
        [Number.NaN, 35.65],
        [139.9, 35.7],
      ],
    });

    expect(actualTrackFeatureForDisplay(feature, null)).toEqual(
      actualTrackFeature({
        coordinates: [
          [139.7, 35.6],
          [139.9, 35.7],
        ],
      }),
    );
  });

  it("returns null when actual track geometry is not a drawable route line", () => {
    expect(
      actualTrackFeatureForDisplay(
        currentPositionFeature([139.7, 35.6]),
        currentPositionFeature([140.2, 35.9]),
      ),
    ).toBeNull();
    expect(
      actualTrackFeatureForDisplay(actualTrackFeature({ coordinates: [[139.7, 35.6]] }), null),
    ).toBeNull();
  });

  it("builds route endpoint point features from exact planned route geometry", () => {
    const result = routeEndpointFeatureCollection(mapData(), flightDetail());

    expect(result).toEqual({
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
    });
  });

  it("does not build endpoint features for unavailable or fallback planned routes", () => {
    expect(
      routeEndpointFeatureCollection(
        mapData({
          planned: {
            source: "airport_great_circle_fallback",
            available: true,
            geojson: actualTrackFeature(),
          },
        }),
        flightDetail(),
      ),
    ).toEqual(emptyFeatureCollection());
    expect(routeEndpointFeatureCollection(null, flightDetail())).toEqual(emptyFeatureCollection());
    expect(routeEndpointFeatureCollection(mapData(), null)).toEqual(emptyFeatureCollection());
  });

  it("does not build endpoint features from degenerate planned route geometry", () => {
    expect(
      routeEndpointFeatureCollection(
        mapData({
          planned: {
            source: "flightaware_route",
            available: true,
            geojson: currentPositionFeature([139.7, 35.6]),
          },
        }),
        flightDetail(),
      ),
    ).toEqual(emptyFeatureCollection());
    expect(
      routeEndpointFeatureCollection(
        mapData({
          planned: {
            source: "flightaware_route",
            available: true,
            geojson: multiLineFeature([[[139.7, 35.6]], [[-73.8, 40.6]]]),
          },
        }),
        flightDetail(),
      ),
    ).toEqual(emptyFeatureCollection());
  });
});

function actualTrackFeature(override: Partial<GeoJSONFeature["geometry"]> = {}): GeoJSONFeature {
  return {
    type: "Feature",
    geometry: {
      type: override.type ?? "LineString",
      coordinates: override.coordinates ?? [
        [139.7, 35.6],
        [139.9, 35.7],
      ],
    },
    properties: { kind: "actual_track" },
  };
}

function currentPositionFeature(coordinates: [number, number]): GeoJSONFeature {
  return {
    type: "Feature",
    geometry: {
      type: "Point",
      coordinates,
    },
    properties: { kind: "current_position" },
  };
}

function multiLineFeature(coordinates: [number, number][][]): GeoJSONFeature {
  return {
    type: "Feature",
    geometry: { type: "MultiLineString", coordinates },
    properties: {},
  };
}

function mapData(override: Partial<FlightMapDataResponse> = {}): FlightMapDataResponse {
  return {
    flightId: "iflg_1",
    faFlightId: "fa_1",
    planned: {
      source: "flightaware_route",
      available: true,
      geojson: {
        type: "Feature",
        geometry: {
          type: "LineString",
          coordinates: [
            [139.7, 35.6],
            [170.1, 48.2],
            [-73.8, 40.6],
          ],
        },
        properties: {},
      },
    },
    actual: {
      source: "flightaware_track",
      available: false,
      geojson: null,
    },
    current: {
      source: "flightaware_position",
      available: false,
      geojson: null,
    },
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
