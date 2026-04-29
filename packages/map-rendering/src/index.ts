import { GeoJsonLayer, ScatterplotLayer } from "@deck.gl/layers";

export interface GeoJSONFeature {
  type: "Feature";
  geometry: {
    type: "Point" | "LineString" | "MultiLineString";
    coordinates: unknown;
  };
  properties: Record<string, unknown>;
}

export interface MapLayer {
  source?: string;
  available: boolean;
  geojson: GeoJSONFeature | null;
}

export interface FlightMapData {
  flightId?: string;
  faFlightId?: string | null;
  planned: MapLayer;
  actual: MapLayer;
  current: MapLayer;
  cache?: unknown;
}

type DeckLayer = GeoJsonLayer<GeoJSONFeature> | ScatterplotLayer<CurrentPoint>;

interface CurrentPoint {
  position: [number, number];
  timestamp: string | null;
}

export function buildDeckGlLayers(mapData: FlightMapData | null): DeckLayer[] {
  if (mapData === null) {
    return [];
  }

  const layers: DeckLayer[] = [];
  if (mapData.planned.available && mapData.planned.geojson !== null) {
    layers.push(
      new GeoJsonLayer<GeoJSONFeature>({
        id: "planned-route",
        data: mapData.planned.geojson as never,
        getLineColor: [31, 111, 120],
        getLineWidth: 3,
        lineWidthMinPixels: 2,
        pickable: true,
      }),
    );
  }
  if (mapData.actual.available && mapData.actual.geojson !== null) {
    layers.push(
      new GeoJsonLayer<GeoJSONFeature>({
        id: "actual-track",
        data: mapData.actual.geojson as never,
        getLineColor: [111, 78, 155],
        getLineWidth: 4,
        lineWidthMinPixels: 2,
        pickable: true,
      }),
    );
  }

  const currentPoint = currentPointFromFeature(mapData.current.geojson);
  if (mapData.current.available && currentPoint !== null) {
    layers.push(
      new ScatterplotLayer<CurrentPoint>({
        id: "current-position",
        data: [currentPoint],
        getFillColor: [194, 65, 12],
        getLineColor: [255, 255, 255],
        getLineWidth: 2,
        getPosition: (point) => point.position,
        getRadius: 9000,
        lineWidthMinPixels: 1,
        pickable: true,
        stroked: true,
      }),
    );
  }

  return layers;
}

export function mapboxStyleURL(): string {
  return "mapbox://styles/mapbox/light-v11";
}

function currentPointFromFeature(feature: GeoJSONFeature | null): CurrentPoint | null {
  const coordinates = feature?.geometry.coordinates;
  if (!Array.isArray(coordinates) || coordinates.length < 2) {
    return null;
  }
  const longitude = Number(coordinates[0]);
  const latitude = Number(coordinates[1]);
  if (!Number.isFinite(longitude) || !Number.isFinite(latitude)) {
    return null;
  }
  return {
    position: [longitude, latitude],
    timestamp:
      typeof feature?.properties["timestamp"] === "string" ? feature.properties["timestamp"] : null,
  };
}
