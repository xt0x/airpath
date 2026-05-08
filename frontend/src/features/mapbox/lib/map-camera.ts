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
  const boundsCoordinates = coordinatesWithWrappedLongitudes(coordinates);
  const firstCoordinate = boundsCoordinates[0];
  if (firstCoordinate === undefined) {
    return null;
  }

  const bounds = new mapboxGL.LngLatBounds(firstCoordinate, firstCoordinate);
  for (const coordinate of boundsCoordinates.slice(1)) {
    bounds.extend(coordinate);
  }
  return bounds;
}

function coordinatesFromFeature(
  feature: FlightMapDataResponse["planned"]["geojson"],
): [number, number][] {
  return collectFiniteCoordinatePairs(feature?.geometry.coordinates);
}

function coordinatesWithWrappedLongitudes(coordinates: [number, number][]): [number, number][] {
  const firstCoordinate = coordinates[0];
  if (firstCoordinate === undefined) {
    return [];
  }

  const wrappedCoordinates: [number, number][] = [firstCoordinate];
  let previousLongitude = firstCoordinate[0];
  for (const [rawLongitude, latitude] of coordinates.slice(1)) {
    const longitude = longitudeNearestToPrevious(rawLongitude, previousLongitude);
    wrappedCoordinates.push([longitude, latitude]);
    previousLongitude = longitude;
  }
  return wrappedCoordinates;
}

function longitudeNearestToPrevious(longitude: number, previousLongitude: number): number {
  let wrappedLongitude = longitude;
  while (wrappedLongitude - previousLongitude > 180) {
    wrappedLongitude -= 360;
  }
  while (wrappedLongitude - previousLongitude < -180) {
    wrappedLongitude += 360;
  }
  return wrappedLongitude;
}
