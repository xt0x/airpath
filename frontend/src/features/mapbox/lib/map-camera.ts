import type { FlightMapDataResponse } from "@/features/flights/types";

import { collectFiniteCoordinatePairs } from "@/features/mapbox/lib/geojson-coordinates";
import {
  availableActualTrackFeature,
  availableCurrentPositionFeature,
  flightAwarePlannedRouteFeature,
} from "@/features/mapbox/lib/map-layer-availability";
import { drawableRouteLineFeature } from "@/features/mapbox/lib/route-line-feature";
import { actualTrackFeatureForDisplay } from "@/features/mapbox/lib/route-source-data";

type MapboxGL = (typeof import("mapbox-gl"))["default"];

export function boundsFromMapData(
  mapData: FlightMapDataResponse | null,
  mapboxGL: MapboxGL | null,
) {
  if (mapData === null || mapboxGL === null) {
    return null;
  }

  const currentPositionFeature = availableCurrentPositionFeature(mapData);
  const actualTrackFeature = actualTrackFeatureForDisplay(
    availableActualTrackFeature(mapData),
    currentPositionFeature,
  );
  const currentOnlyFeature = actualTrackFeature === null ? currentPositionFeature : null;
  const coordinates = [
    ...coordinatesFromFeature(drawableRouteLineFeature(flightAwarePlannedRouteFeature(mapData))),
    ...coordinatesFromFeature(actualTrackFeature),
    ...coordinatesFromFeature(currentOnlyFeature),
  ];
  const firstCoordinate = coordinates[0];
  if (firstCoordinate === undefined) {
    return null;
  }

  const bounds = new mapboxGL.LngLatBounds(firstCoordinate, firstCoordinate);
  for (const coordinate of coordinates.slice(1)) {
    bounds.extend(coordinate);
  }
  return bounds;
}

function coordinatesFromFeature(
  feature: FlightMapDataResponse["planned"]["geojson"],
): [number, number][] {
  return collectFiniteCoordinatePairs(feature?.geometry.coordinates);
}
