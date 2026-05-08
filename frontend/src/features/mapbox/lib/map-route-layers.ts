import type { FlightDetail, FlightMapDataResponse } from "@/features/flights/types";
import type { GeoJSONSource, Map as MapboxMap } from "mapbox-gl";

import {
  actualTrackLinePaint,
  plannedRouteLinePaint,
  routeEndpointLabelPaint,
  routeLinePaintProperties,
} from "@/features/mapbox/lib/map-theme";
import {
  emptyRouteFeatureCollection,
  routeLayerSourceData,
} from "@/features/mapbox/lib/route-layer-source-data";

export const AIRPATH_PLANNED_ROUTE_SOURCE_ID = "airpath-planned-route-source";
export const AIRPATH_ACTUAL_TRACK_SOURCE_ID = "airpath-actual-track-source";
export const AIRPATH_ROUTE_ENDPOINT_SOURCE_ID = "airpath-route-endpoint-source";
export const AIRPATH_PLANNED_ROUTE_LAYER_ID = "airpath-planned-route-line";
export const AIRPATH_ACTUAL_TRACK_LAYER_ID = "airpath-actual-track-line";
export const AIRPATH_ROUTE_ENDPOINT_LABEL_LAYER_ID = "airpath-route-endpoint-label";

const routeSourceDataKeysByMap = new WeakMap<MapboxMap, Map<string, string>>();

export function ensureNativeRouteLayers(map: MapboxMap) {
  ensureRouteSource(map, AIRPATH_PLANNED_ROUTE_SOURCE_ID);
  ensureRouteSource(map, AIRPATH_ACTUAL_TRACK_SOURCE_ID);
  ensureRouteSource(map, AIRPATH_ROUTE_ENDPOINT_SOURCE_ID);

  if (!map.getLayer(AIRPATH_PLANNED_ROUTE_LAYER_ID)) {
    map.addLayer({
      id: AIRPATH_PLANNED_ROUTE_LAYER_ID,
      type: "line",
      source: AIRPATH_PLANNED_ROUTE_SOURCE_ID,
      layout: {
        "line-cap": "round",
        "line-join": "round",
      },
      paint: plannedRouteLinePaint(),
    });
  }
  if (!map.getLayer(AIRPATH_ACTUAL_TRACK_LAYER_ID)) {
    map.addLayer({
      id: AIRPATH_ACTUAL_TRACK_LAYER_ID,
      type: "line",
      source: AIRPATH_ACTUAL_TRACK_SOURCE_ID,
      layout: {
        "line-cap": "round",
        "line-join": "round",
      },
      paint: actualTrackLinePaint(),
    });
  }

  if (!map.getLayer(AIRPATH_ROUTE_ENDPOINT_LABEL_LAYER_ID)) {
    map.addLayer({
      id: AIRPATH_ROUTE_ENDPOINT_LABEL_LAYER_ID,
      type: "symbol",
      source: AIRPATH_ROUTE_ENDPOINT_SOURCE_ID,
      layout: {
        "text-allow-overlap": true,
        "text-field": ["get", "code"],
        "text-font": ["Open Sans Bold", "Arial Unicode MS Bold"],
        "text-offset": [0, -1.35],
        "text-size": 12,
      },
      paint: routeEndpointLabelPaint(),
    });
  }
  applyNativeRouteLayerPaintProperties(map);
}

export function updateNativeRouteLayers(
  map: MapboxMap,
  mapData: FlightMapDataResponse | null,
  flight: FlightDetail | null,
) {
  const sourceData = routeLayerSourceData(mapData, flight);

  setRouteSourceData(map, AIRPATH_PLANNED_ROUTE_SOURCE_ID, sourceData.plannedRoute);
  setRouteSourceData(map, AIRPATH_ACTUAL_TRACK_SOURCE_ID, sourceData.actualTrack);
  setRouteSourceData(map, AIRPATH_ROUTE_ENDPOINT_SOURCE_ID, sourceData.routeEndpoints);
}

export function applyNativeRouteLayerPaintProperties(map: MapboxMap) {
  if (map.getLayer(AIRPATH_PLANNED_ROUTE_LAYER_ID)) {
    setLineColorPaintProperty(map, AIRPATH_PLANNED_ROUTE_LAYER_ID);
  }
  if (map.getLayer(AIRPATH_ACTUAL_TRACK_LAYER_ID)) {
    setLineColorPaintProperty(map, AIRPATH_ACTUAL_TRACK_LAYER_ID);
  }
}

function ensureRouteSource(map: MapboxMap, sourceId: string) {
  if (map.getSource(sourceId)) {
    return;
  }
  map.addSource(sourceId, {
    type: "geojson",
    data: emptyRouteFeatureCollection(),
  });
  forgetRouteSourceDataKey(map, sourceId);
}

function setLineColorPaintProperty(map: MapboxMap, layerId: string) {
  const paintProperties = routeLinePaintProperties();
  const routePaintPropertyNames = Object.keys(paintProperties) as Array<
    keyof typeof paintProperties
  >;
  for (const property of routePaintPropertyNames) {
    const value = paintProperties[property];
    if (map.getPaintProperty(layerId, property) !== value) {
      map.setPaintProperty(layerId, property, value);
    }
  }
}

function setRouteSourceData(
  map: MapboxMap,
  sourceId: string,
  featureCollection: GeoJSON.FeatureCollection,
) {
  const source = map.getSource(sourceId);
  if (!isGeoJSONSource(source)) {
    return;
  }
  const nextDataKey = routeSourceDataKey(featureCollection);
  if (lastRouteSourceDataKey(map, sourceId) === nextDataKey) {
    return;
  }
  source.setData(featureCollection);
  rememberRouteSourceDataKey(map, sourceId, nextDataKey);
}

function isGeoJSONSource(source: ReturnType<MapboxMap["getSource"]>): source is GeoJSONSource {
  return source !== undefined && "setData" in source && typeof source.setData === "function";
}

function routeSourceDataKey(featureCollection: GeoJSON.FeatureCollection): string {
  return JSON.stringify(featureCollection);
}

function lastRouteSourceDataKey(map: MapboxMap, sourceId: string): string | undefined {
  return routeSourceDataKeysByMap.get(map)?.get(sourceId);
}

function rememberRouteSourceDataKey(map: MapboxMap, sourceId: string, dataKey: string) {
  let sourceDataKeys = routeSourceDataKeysByMap.get(map);
  if (sourceDataKeys === undefined) {
    sourceDataKeys = new Map();
    routeSourceDataKeysByMap.set(map, sourceDataKeys);
  }
  sourceDataKeys.set(sourceId, dataKey);
}

function forgetRouteSourceDataKey(map: MapboxMap, sourceId: string) {
  routeSourceDataKeysByMap.get(map)?.delete(sourceId);
}
