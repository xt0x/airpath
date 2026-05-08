import type { FlightDetail, FlightMapDataResponse, GeoJSONFeature } from "@/features/flights/types";

import {
  collectFiniteCoordinatePairs,
  coordinatePairFromValue,
} from "@/features/mapbox/lib/geojson-coordinates";
import { flightAwarePlannedRouteFeature } from "@/features/mapbox/lib/map-layer-availability";
import {
  drawableRouteLineFeature,
  drawableRouteLineFeatureWithAppendedCoordinate,
} from "@/features/mapbox/lib/route-line-feature";

export function actualTrackFeatureForDisplay(
  feature: GeoJSONFeature | null,
  currentFeature: GeoJSONFeature | null,
): GeoJSONFeature | null {
  if (feature === null) {
    return null;
  }

  const currentCoordinate = pointCoordinateFromFeature(currentFeature);
  if (currentCoordinate === null) {
    return drawableRouteLineFeature(feature);
  }

  return drawableRouteLineFeatureWithAppendedCoordinate(feature, currentCoordinate);
}

export function routeEndpointFeatureCollection(
  mapData: FlightMapDataResponse | null,
  flight: FlightDetail | null,
): GeoJSON.FeatureCollection {
  const plannedRouteFeature = drawableRouteLineFeature(flightAwarePlannedRouteFeature(mapData));
  if (flight === null || plannedRouteFeature === null) {
    return emptyFeatureCollection();
  }

  const coordinates = coordinatesFromFeature(plannedRouteFeature);
  const originPosition = coordinates[0];
  const destinationPosition = coordinates[coordinates.length - 1];
  if (coordinates.length < 2 || originPosition === undefined || destinationPosition === undefined) {
    return emptyFeatureCollection();
  }

  return {
    type: "FeatureCollection",
    features: [
      routeEndpointFeature("origin", flight.origin.code, originPosition),
      routeEndpointFeature("destination", flight.destination.code, destinationPosition),
    ],
  };
}

function routeEndpointFeature(
  role: "origin" | "destination",
  code: string,
  coordinates: [number, number],
): GeoJSON.Feature {
  return {
    type: "Feature",
    geometry: { type: "Point", coordinates },
    properties: { code, role },
  };
}

function coordinatesFromFeature(feature: GeoJSONFeature | null): [number, number][] {
  return collectFiniteCoordinatePairs(feature?.geometry.coordinates);
}

function pointCoordinateFromFeature(feature: GeoJSONFeature | null): [number, number] | null {
  if (feature?.geometry.type !== "Point") {
    return null;
  }
  const coordinates = coordinatePairFromValue(feature.geometry.coordinates);
  return coordinates;
}

function emptyFeatureCollection(): GeoJSON.FeatureCollection {
  return { type: "FeatureCollection", features: [] };
}
