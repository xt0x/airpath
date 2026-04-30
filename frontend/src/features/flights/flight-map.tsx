import { buildDeckGlLayers } from "@airpath/map-rendering";

import type { FlightMapDataResponse } from "./types";

export function FlightMap({ mapData }: { mapData: FlightMapDataResponse | null }) {
  const deckLayers = buildDeckGlLayers(mapData);
  const plannedPath = pathForLayer(mapData?.planned.geojson ?? null);
  const actualPath = pathForLayer(mapData?.actual.geojson ?? null);
  const currentPoint = pointForLayer(mapData?.current.geojson ?? null);

  return (
    <section
      className="flight-dashboard__map-surface"
      data-deck-layer-count={deckLayers.length}
      data-renderer="mapbox-deckgl-compatible"
    >
      <svg aria-label="Flight map" role="img" viewBox="0 0 100 56" preserveAspectRatio="none">
        <rect width="100" height="56" rx="2" />
        {plannedPath !== "" ? <polyline data-layer="planned" points={plannedPath} /> : null}
        {actualPath !== "" ? <polyline data-layer="actual" points={actualPath} /> : null}
        {currentPoint !== null ? (
          <circle data-layer="current" cx={currentPoint.x} cy={currentPoint.y} r="1.8" />
        ) : null}
      </svg>
      <div className="flight-dashboard__layer-strip">
        <LayerState label="Planned" layer={mapData?.planned ?? null} />
        <LayerState label="Actual" layer={mapData?.actual ?? null} />
        <LayerState label="Current" layer={mapData?.current ?? null} />
      </div>
    </section>
  );
}

function LayerState({
  label,
  layer,
}: {
  label: string;
  layer: FlightMapDataResponse["planned"] | null;
}) {
  return (
    <span
      className={
        layer?.available
          ? "flight-dashboard__layer-state available"
          : "flight-dashboard__layer-state"
      }
    >
      {label}
    </span>
  );
}

function pathForLayer(feature: { geometry: { coordinates: unknown } } | null) {
  const coordinates = firstLineCoordinates(feature?.geometry.coordinates);
  return coordinates
    .map(([longitude, latitude]) => `${scaleLongitude(longitude)},${scaleLatitude(latitude)}`)
    .join(" ");
}

function pointForLayer(feature: { geometry: { coordinates: unknown } } | null) {
  const coordinates = feature?.geometry.coordinates;
  if (!Array.isArray(coordinates) || coordinates.length < 2) {
    return null;
  }
  const longitude = Number(coordinates[0]);
  const latitude = Number(coordinates[1]);
  if (!Number.isFinite(longitude) || !Number.isFinite(latitude)) {
    return null;
  }
  return { x: scaleLongitude(longitude), y: scaleLatitude(latitude) };
}

function firstLineCoordinates(value: unknown): Array<[number, number]> {
  if (!Array.isArray(value)) {
    return [];
  }
  const line = Array.isArray(value[0]?.[0]) ? value[0] : value;
  if (!Array.isArray(line)) {
    return [];
  }
  return line.reduce<Array<[number, number]>>((coordinates, point: unknown) => {
    if (!Array.isArray(point) || point.length < 2) {
      return coordinates;
    }
    const longitude = Number(point[0]);
    const latitude = Number(point[1]);
    if (Number.isFinite(longitude) && Number.isFinite(latitude)) {
      coordinates.push([longitude, latitude]);
    }
    return coordinates;
  }, []);
}

function scaleLongitude(longitude: number) {
  return ((longitude + 180) / 360) * 100;
}

function scaleLatitude(latitude: number) {
  return ((90 - latitude) / 180) * 56;
}
