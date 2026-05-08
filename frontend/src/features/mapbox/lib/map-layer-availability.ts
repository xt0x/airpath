import type { FlightMapDataResponse, GeoJSONFeature, MapLayer } from "@/features/flights/types";

export function availableLayerFeature(layer: MapLayer | null | undefined): GeoJSONFeature | null {
  return layer?.available === true ? layer.geojson : null;
}

export function flightAwarePlannedRouteFeature(
  mapData: FlightMapDataResponse | null,
): GeoJSONFeature | null {
  const feature = availableLayerFeature(mapData?.planned);
  return mapData?.planned.source === "flightaware_route" ? feature : null;
}

export function availableActualTrackFeature(
  mapData: FlightMapDataResponse | null,
): GeoJSONFeature | null {
  return availableLayerFeature(mapData?.actual);
}

export function availableCurrentPositionFeature(
  mapData: FlightMapDataResponse | null,
): GeoJSONFeature | null {
  return availableLayerFeature(mapData?.current);
}
