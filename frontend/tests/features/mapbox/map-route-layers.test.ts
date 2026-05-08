import { describe, expect, it } from "vitest";

import type { FlightDetail, FlightMapDataResponse, GeoJSONFeature } from "@/features/flights/types";

import {
  AIRPATH_ACTUAL_TRACK_LAYER_ID,
  AIRPATH_ACTUAL_TRACK_SOURCE_ID,
  AIRPATH_PLANNED_ROUTE_LAYER_ID,
  AIRPATH_PLANNED_ROUTE_SOURCE_ID,
  AIRPATH_ROUTE_ENDPOINT_LABEL_LAYER_ID,
  AIRPATH_ROUTE_ENDPOINT_SOURCE_ID,
  applyNativeRouteLayerPaintProperties,
  ensureNativeRouteLayers,
  updateNativeRouteLayers,
} from "@/features/mapbox/lib/map-route-layers";
import {
  actualTrackLinePaint,
  plannedRouteLinePaint,
  routeEndpointLabelPaint,
  routeLinePaintProperties,
} from "@/features/mapbox/lib/map-theme";

describe("map route layers", () => {
  it("ensures the native Mapbox sources and layers for routes and endpoint labels", () => {
    const map = new FakeRouteMap();

    ensureNativeRouteLayers(map.asMapboxMap());

    expect(Array.from(map.sources.keys())).toEqual([
      AIRPATH_PLANNED_ROUTE_SOURCE_ID,
      AIRPATH_ACTUAL_TRACK_SOURCE_ID,
      AIRPATH_ROUTE_ENDPOINT_SOURCE_ID,
    ]);
    expect(map.layers.get(AIRPATH_PLANNED_ROUTE_LAYER_ID)).toMatchObject({
      id: AIRPATH_PLANNED_ROUTE_LAYER_ID,
      type: "line",
      source: AIRPATH_PLANNED_ROUTE_SOURCE_ID,
      paint: plannedRouteLinePaint(),
    });
    expect(map.layers.get(AIRPATH_ACTUAL_TRACK_LAYER_ID)).toMatchObject({
      id: AIRPATH_ACTUAL_TRACK_LAYER_ID,
      type: "line",
      source: AIRPATH_ACTUAL_TRACK_SOURCE_ID,
      paint: actualTrackLinePaint(),
    });
    expect(map.layers.get(AIRPATH_ROUTE_ENDPOINT_LABEL_LAYER_ID)).toMatchObject({
      id: AIRPATH_ROUTE_ENDPOINT_LABEL_LAYER_ID,
      type: "symbol",
      source: AIRPATH_ROUTE_ENDPOINT_SOURCE_ID,
      paint: routeEndpointLabelPaint(),
    });
  });

  it("is idempotent when route sources and layers already exist", () => {
    const map = new FakeRouteMap();

    ensureNativeRouteLayers(map.asMapboxMap());
    ensureNativeRouteLayers(map.asMapboxMap());

    expect(map.addedSources).toHaveLength(3);
    expect(map.addedLayers).toHaveLength(3);
  });

  it("updates planned route, actual track, and route endpoint source data", () => {
    const map = new FakeRouteMap();
    ensureNativeRouteLayers(map.asMapboxMap());

    updateNativeRouteLayers(
      map.asMapboxMap(),
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
    );

    expect(map.sourceData(AIRPATH_PLANNED_ROUTE_SOURCE_ID)).toEqual({
      type: "FeatureCollection",
      features: [
        lineFeature([
          [139.7, 35.6],
          [-73.8, 40.6],
        ]),
      ],
    });
    expect(map.sourceData(AIRPATH_ACTUAL_TRACK_SOURCE_ID)).toEqual({
      type: "FeatureCollection",
      features: [
        lineFeature([
          [139.7, 35.6],
          [140.1, 35.9],
          [140.2, 36],
        ]),
      ],
    });
    expect(map.sourceData(AIRPATH_ROUTE_ENDPOINT_SOURCE_ID)).toEqual({
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

  it("does not reapply identical route source data", () => {
    const map = new FakeRouteMap();
    const data = mapData({
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
    });
    ensureNativeRouteLayers(map.asMapboxMap());

    updateNativeRouteLayers(map.asMapboxMap(), data, flightDetail());
    const updateCountsAfterFirstWrite = map.sourceUpdateCounts();
    updateNativeRouteLayers(map.asMapboxMap(), data, flightDetail());

    expect(map.sourceUpdateCounts()).toEqual(updateCountsAfterFirstWrite);
  });

  it("reapplies route source data after Mapbox recreates a source", () => {
    const map = new FakeRouteMap();
    const data = mapData({
      planned: {
        source: "flightaware_route",
        available: true,
        geojson: lineFeature([
          [139.7, 35.6],
          [-73.8, 40.6],
        ]),
      },
    });
    ensureNativeRouteLayers(map.asMapboxMap());
    updateNativeRouteLayers(map.asMapboxMap(), data, flightDetail());

    map.deleteSource(AIRPATH_PLANNED_ROUTE_SOURCE_ID);
    ensureNativeRouteLayers(map.asMapboxMap());
    updateNativeRouteLayers(map.asMapboxMap(), data, flightDetail());

    expect(map.sourceData(AIRPATH_PLANNED_ROUTE_SOURCE_ID)).toEqual({
      type: "FeatureCollection",
      features: [
        lineFeature([
          [139.7, 35.6],
          [-73.8, 40.6],
        ]),
      ],
    });
    expect(map.sources.get(AIRPATH_PLANNED_ROUTE_SOURCE_ID)?.updates).toBe(1);
  });

  it("clears unavailable and fallback route data from native sources", () => {
    const map = new FakeRouteMap();
    ensureNativeRouteLayers(map.asMapboxMap());

    updateNativeRouteLayers(
      map.asMapboxMap(),
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
    );

    expect(map.sourceData(AIRPATH_PLANNED_ROUTE_SOURCE_ID)).toEqual(emptyFeatureCollection());
    expect(map.sourceData(AIRPATH_ACTUAL_TRACK_SOURCE_ID)).toEqual(emptyFeatureCollection());
    expect(map.sourceData(AIRPATH_ROUTE_ENDPOINT_SOURCE_ID)).toEqual(emptyFeatureCollection());
  });

  it("reapplies shared route line paint properties only when they drift", () => {
    const map = new FakeRouteMap();
    ensureNativeRouteLayers(map.asMapboxMap());
    map.paint.set(`${AIRPATH_PLANNED_ROUTE_LAYER_ID}:line-color`, "#000000");

    applyNativeRouteLayerPaintProperties(map.asMapboxMap());

    expect(map.paintUpdates).toEqual([
      {
        layerId: AIRPATH_PLANNED_ROUTE_LAYER_ID,
        property: "line-color",
        value: routeLinePaintProperties()["line-color"],
      },
    ]);
  });
});

class FakeRouteMap {
  readonly addedLayers: unknown[] = [];
  readonly addedSources: unknown[] = [];
  readonly layers = new Map<
    string,
    { paint?: Record<string, unknown> } & Record<string, unknown>
  >();
  readonly paint = new Map<string, unknown>();
  readonly paintUpdates: Array<{ layerId: string; property: string; value: unknown }> = [];
  readonly sources = new Map<
    string,
    { data: unknown; setData: (data: unknown) => void; updates: number }
  >();

  addLayer(layer: { id: string; paint?: Record<string, unknown> } & Record<string, unknown>) {
    this.addedLayers.push(layer);
    this.layers.set(layer.id, layer);
    for (const [property, value] of Object.entries(layer.paint ?? {})) {
      this.paint.set(`${layer.id}:${property}`, value);
    }
  }

  addSource(sourceId: string, source: { data: unknown }) {
    this.addedSources.push({ source, sourceId });
    this.sources.set(sourceId, {
      data: source.data,
      updates: 0,
      setData(data: unknown) {
        this.data = data;
        this.updates += 1;
      },
    });
  }

  deleteSource(sourceId: string) {
    this.sources.delete(sourceId);
  }

  asMapboxMap() {
    return this as unknown as import("mapbox-gl").Map;
  }

  getLayer(layerId: string) {
    return this.layers.get(layerId);
  }

  getPaintProperty(layerId: string, property: string) {
    return this.paint.get(`${layerId}:${property}`);
  }

  getSource(sourceId: string) {
    return this.sources.get(sourceId);
  }

  setPaintProperty(layerId: string, property: string, value: unknown) {
    this.paintUpdates.push({ layerId, property, value });
    this.paint.set(`${layerId}:${property}`, value);
  }

  sourceData(sourceId: string) {
    return this.sources.get(sourceId)?.data;
  }

  sourceUpdateCounts() {
    return Object.fromEntries(
      Array.from(this.sources.entries()).map(([sourceId, source]) => [sourceId, source.updates]),
    );
  }
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
