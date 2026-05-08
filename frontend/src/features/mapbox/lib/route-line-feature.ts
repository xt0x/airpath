import type { GeoJSONFeature } from "@/features/flights/types";

import { coordinatePairFromValue } from "@/features/mapbox/lib/geojson-coordinates";

export function drawableRouteLineFeature(feature: GeoJSONFeature | null): GeoJSONFeature | null {
  if (feature === null) {
    return null;
  }

  if (feature.geometry.type === "LineString") {
    const coordinates = finiteLineStringCoordinates(feature.geometry.coordinates, 2);
    return coordinates === null
      ? null
      : { ...feature, geometry: { ...feature.geometry, coordinates } };
  }

  if (feature.geometry.type === "MultiLineString") {
    const coordinates = finiteMultiLineStringCoordinates(feature.geometry.coordinates, 2);
    return coordinates === null
      ? null
      : { ...feature, geometry: { ...feature.geometry, coordinates } };
  }

  return null;
}

export function drawableRouteLineFeatureWithAppendedCoordinate(
  feature: GeoJSONFeature | null,
  coordinate: [number, number],
): GeoJSONFeature | null {
  if (feature === null) {
    return null;
  }

  if (feature.geometry.type === "LineString") {
    const coordinates = finiteLineStringCoordinates(feature.geometry.coordinates, 1);
    return drawableRouteLineFeature(
      coordinates === null
        ? null
        : {
            ...feature,
            geometry: {
              ...feature.geometry,
              coordinates: appendCoordinate(coordinates, coordinate),
            },
          },
    );
  }

  if (feature.geometry.type === "MultiLineString") {
    const coordinates = finiteMultiLineStringCoordinates(feature.geometry.coordinates, 1);
    if (coordinates === null) {
      return null;
    }
    const nextCoordinates = coordinates.map((segment) => [...segment]);
    const lastSegmentIndex = nextCoordinates.length - 1;
    const lastSegment = nextCoordinates[lastSegmentIndex];
    if (lastSegment === undefined) {
      return null;
    }
    nextCoordinates[lastSegmentIndex] = appendCoordinate(lastSegment, coordinate);
    return drawableRouteLineFeature({
      ...feature,
      geometry: {
        ...feature.geometry,
        coordinates: nextCoordinates,
      },
    });
  }

  return null;
}

function finiteMultiLineStringCoordinates(
  value: unknown,
  minCoordinatesPerSegment: number,
): [number, number][][] | null {
  if (!Array.isArray(value)) {
    return null;
  }

  const segments = value.flatMap((segment) => {
    const coordinates = finiteLineStringCoordinates(segment, minCoordinatesPerSegment);
    return coordinates === null ? [] : [coordinates];
  });

  return segments.length === 0 ? null : segments;
}

function finiteLineStringCoordinates(
  value: unknown,
  minCoordinates: number,
): [number, number][] | null {
  if (!Array.isArray(value)) {
    return null;
  }

  const coordinates = value.flatMap((coordinate) => {
    const pair = coordinatePairFromValue(coordinate);
    return pair === null ? [] : [pair];
  });

  return coordinates.length < minCoordinates ? null : coordinates;
}

function appendCoordinate(
  coordinates: [number, number][],
  coordinate: [number, number],
): [number, number][] {
  const last = coordinates[coordinates.length - 1];
  if (last !== undefined && last[0] === coordinate[0] && last[1] === coordinate[1]) {
    return coordinates;
  }
  return [...coordinates, coordinate];
}
