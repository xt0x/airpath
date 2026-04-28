export type LongitudeLatitude = [longitude: number, latitude: number];

export interface GeoPoint {
  latitude: number | null;
  longitude: number | null;
}

export interface NamedGeoPoint extends GeoPoint {
  name?: string | null;
}

export interface AirportGeoPoint extends GeoPoint {
  code: string;
}

export interface TrackPoint extends GeoPoint {
  timestamp: string | null;
  altitudeFeet?: number | null;
  groundspeedKnots?: number | null;
  headingDegrees?: number | null;
  kind?: "departure" | "arrival" | "latest";
  important?: boolean;
}

export interface GeoJsonFeature<
  TGeometry extends GeoJsonGeometry = GeoJsonGeometry,
  TProperties extends Record<string, unknown> = Record<string, unknown>,
> {
  type: "Feature";
  geometry: TGeometry;
  properties: TProperties;
}

export interface GeoJsonFeatureCollection<TFeature extends GeoJsonFeature = GeoJsonFeature> {
  type: "FeatureCollection";
  features: TFeature[];
}

export type GeoJsonGeometry = PointGeometry | MultiLineStringGeometry;

export interface PointGeometry {
  type: "Point";
  coordinates: LongitudeLatitude;
}

export interface MultiLineStringGeometry {
  type: "MultiLineString";
  coordinates: LongitudeLatitude[][];
}

export type PlannedRouteSource = "flightaware_route" | "airport_great_circle_fallback";

export interface PlannedRouteProperties extends Record<string, unknown> {
  kind: "planned_route";
  source: PlannedRouteSource;
  routeText: string | null;
  pointCount: number;
  origin: string | null;
  destination: string | null;
}

export interface BuildPlannedRouteInput {
  routePoints: NamedGeoPoint[];
  routeText: string | null;
  origin: AirportGeoPoint | null;
  destination: AirportGeoPoint | null;
  fallbackStepCount?: number;
}

export interface ActualTrackProperties extends Record<string, unknown> {
  kind: "actual_track";
  source: "flightaware_track";
  lastTimestamp: string;
  pointCount: number;
}

export interface TrackPointProperties extends Record<string, unknown> {
  kind: "track_point";
  timestamp: string;
  sequence: number;
  altitudeFeet: number | null;
  groundspeedKnots: number | null;
  headingDegrees: number | null;
}

export interface CurrentPositionProperties extends Record<string, unknown> {
  kind: "current_position";
  altitudeFeet: number | null;
  groundspeedKnots: number | null;
  headingDegrees: number | null;
  timestamp: string;
}

export interface TrackFeatures {
  line: GeoJsonFeature<MultiLineStringGeometry, ActualTrackProperties> | null;
  points: GeoJsonFeatureCollection<GeoJsonFeature<PointGeometry, TrackPointProperties>>;
}

type ValidTrackPoint = TrackPoint & {
  latitude: number;
  longitude: number;
  timestamp: string;
};

const degreesToRadians = (degrees: number): number => (degrees * Math.PI) / 180;
const radiansToDegrees = (radians: number): number => (radians * 180) / Math.PI;

export function buildPlannedRouteFeature(
  input: BuildPlannedRouteInput,
): GeoJsonFeature<MultiLineStringGeometry, PlannedRouteProperties> | null {
  const decodedCoordinates = input.routePoints
    .filter(isValidGeoPoint)
    .map((point) => toCoordinate(point));

  if (decodedCoordinates.length >= 2) {
    return {
      type: "Feature",
      geometry: {
        type: "MultiLineString",
        coordinates: splitLineStringForAntimeridian(decodedCoordinates),
      },
      properties: {
        kind: "planned_route",
        source: "flightaware_route",
        routeText: input.routeText,
        pointCount: decodedCoordinates.length,
        origin: input.origin?.code ?? null,
        destination: input.destination?.code ?? null,
      },
    };
  }

  if (isValidGeoPoint(input.origin) && isValidGeoPoint(input.destination)) {
    const fallbackCoordinates = buildGreatCircleCoordinates(
      toCoordinate(input.origin),
      toCoordinate(input.destination),
      input.fallbackStepCount ?? 64,
    );

    return {
      type: "Feature",
      geometry: {
        type: "MultiLineString",
        coordinates: splitLineStringForAntimeridian(fallbackCoordinates),
      },
      properties: {
        kind: "planned_route",
        source: "airport_great_circle_fallback",
        routeText: input.routeText,
        pointCount: fallbackCoordinates.length,
        origin: input.origin.code,
        destination: input.destination.code,
      },
    };
  }

  return null;
}

export function buildTrackFeatures(points: TrackPoint[]): TrackFeatures {
  const validPoints = points.filter(isValidTrackPoint);
  const pointFeatures = validPoints.map((point, sequence) =>
    buildTrackPointFeature(point, sequence),
  );

  return {
    line:
      validPoints.length >= 2
        ? {
            type: "Feature",
            geometry: {
              type: "MultiLineString",
              coordinates: splitLineStringForAntimeridian(
                validPoints.map((point) => toCoordinate(point)),
              ),
            },
            properties: {
              kind: "actual_track",
              source: "flightaware_track",
              lastTimestamp: validPoints[validPoints.length - 1]?.timestamp ?? "",
              pointCount: validPoints.length,
            },
          }
        : null,
    points: {
      type: "FeatureCollection",
      features: pointFeatures,
    },
  };
}

export function buildCurrentPositionFeature(
  position: TrackPoint,
): GeoJsonFeature<PointGeometry, CurrentPositionProperties> | null {
  if (!isValidTrackPoint(position)) {
    return null;
  }

  return {
    type: "Feature",
    geometry: {
      type: "Point",
      coordinates: toCoordinate(position),
    },
    properties: {
      kind: "current_position",
      altitudeFeet: position.altitudeFeet ?? null,
      groundspeedKnots: position.groundspeedKnots ?? null,
      headingDegrees: position.headingDegrees ?? null,
      timestamp: position.timestamp,
    },
  };
}

export function splitLineStringForAntimeridian(
  coordinates: LongitudeLatitude[],
): LongitudeLatitude[][] {
  if (coordinates.length <= 1) {
    return coordinates.length === 0 ? [] : [[coordinates[0] as LongitudeLatitude]];
  }

  const lines: LongitudeLatitude[][] = [];
  let currentLine: LongitudeLatitude[] = [coordinates[0] as LongitudeLatitude];

  for (let index = 1; index < coordinates.length; index += 1) {
    const previous = coordinates[index - 1] as LongitudeLatitude;
    const current = coordinates[index] as LongitudeLatitude;
    const previousLongitude = previous[0];
    const currentLongitude = current[0];
    const delta = currentLongitude - previousLongitude;

    if (Math.abs(delta) <= 180) {
      currentLine.push(current);
      continue;
    }

    const adjustedCurrentLongitude = currentLongitude + (delta > 0 ? -360 : 360);
    const boundaryLongitude = delta > 0 ? -180 : 180;
    const wrappedBoundaryLongitude = delta > 0 ? 180 : -180;
    const ratio =
      (boundaryLongitude - previousLongitude) / (adjustedCurrentLongitude - previousLongitude);
    const boundaryLatitude = interpolate(previous[1], current[1], ratio);

    currentLine.push([boundaryLongitude, boundaryLatitude]);
    lines.push(currentLine);
    currentLine = [[wrappedBoundaryLongitude, boundaryLatitude], current];
  }

  lines.push(currentLine);

  return lines;
}

export function simplifyTrackPoints<TPoint extends TrackPoint>(
  points: TPoint[],
  maxPointCount: number,
): TPoint[] {
  if (points.length <= maxPointCount || maxPointCount <= 0) {
    return points.slice();
  }

  const keepIndexes = new Set<number>();
  keepIndexes.add(0);
  keepIndexes.add(points.length - 1);

  points.forEach((point, index) => {
    if (
      point.important === true ||
      point.kind === "departure" ||
      point.kind === "arrival" ||
      point.kind === "latest"
    ) {
      keepIndexes.add(index);
    }
  });

  const requiredIndexes = [...keepIndexes].sort((left, right) => left - right);
  if (requiredIndexes.length >= maxPointCount) {
    return requiredIndexes.slice(0, maxPointCount).map((index) => points[index] as TPoint);
  }

  const remainingSlots = maxPointCount - requiredIndexes.length;
  const candidateIndexes = points
    .map((_, index) => index)
    .filter((index) => !keepIndexes.has(index));

  for (let slot = 1; slot <= remainingSlots; slot += 1) {
    const candidateOffset =
      Math.floor((slot * (candidateIndexes.length + 1)) / (remainingSlots + 1)) - 1;
    const index =
      candidateIndexes[Math.max(0, Math.min(candidateIndexes.length - 1, candidateOffset))];
    if (index !== undefined) {
      keepIndexes.add(index);
    }
  }

  return [...keepIndexes]
    .sort((left, right) => left - right)
    .map((index) => points[index] as TPoint);
}

function buildTrackPointFeature(
  point: ValidTrackPoint,
  sequence: number,
): GeoJsonFeature<PointGeometry, TrackPointProperties> {
  return {
    type: "Feature",
    geometry: {
      type: "Point",
      coordinates: toCoordinate(point),
    },
    properties: {
      kind: "track_point",
      timestamp: point.timestamp,
      sequence,
      altitudeFeet: point.altitudeFeet ?? null,
      groundspeedKnots: point.groundspeedKnots ?? null,
      headingDegrees: point.headingDegrees ?? null,
    },
  };
}

function buildGreatCircleCoordinates(
  start: LongitudeLatitude,
  end: LongitudeLatitude,
  stepCount: number,
): LongitudeLatitude[] {
  const safeStepCount = Math.max(1, Math.floor(stepCount));
  const startLat = degreesToRadians(start[1]);
  const startLon = degreesToRadians(start[0]);
  const endLat = degreesToRadians(end[1]);
  const endLon = degreesToRadians(end[0]);
  const angularDistance =
    2 *
    Math.asin(
      Math.sqrt(
        Math.sin((endLat - startLat) / 2) ** 2 +
          Math.cos(startLat) * Math.cos(endLat) * Math.sin((endLon - startLon) / 2) ** 2,
      ),
    );

  if (angularDistance === 0) {
    return [start, end];
  }

  const coordinates: LongitudeLatitude[] = [];
  for (let step = 0; step <= safeStepCount; step += 1) {
    const fraction = step / safeStepCount;
    const a = Math.sin((1 - fraction) * angularDistance) / Math.sin(angularDistance);
    const b = Math.sin(fraction * angularDistance) / Math.sin(angularDistance);
    const x = a * Math.cos(startLat) * Math.cos(startLon) + b * Math.cos(endLat) * Math.cos(endLon);
    const y = a * Math.cos(startLat) * Math.sin(startLon) + b * Math.cos(endLat) * Math.sin(endLon);
    const z = a * Math.sin(startLat) + b * Math.sin(endLat);
    const latitude = radiansToDegrees(Math.atan2(z, Math.sqrt(x * x + y * y)));
    const longitude = normalizeLongitude(radiansToDegrees(Math.atan2(y, x)));

    coordinates.push([roundCoordinate(longitude), roundCoordinate(latitude)]);
  }

  return coordinates;
}

function isValidTrackPoint(point: TrackPoint | null | undefined): point is ValidTrackPoint {
  return (
    isValidGeoPoint(point) &&
    typeof point.timestamp === "string" &&
    isValidIsoLikeTimestamp(point.timestamp)
  );
}

function isValidGeoPoint(point: GeoPoint | null | undefined): point is GeoPoint & {
  latitude: number;
  longitude: number;
} {
  return (
    point !== null &&
    point !== undefined &&
    typeof point.latitude === "number" &&
    Number.isFinite(point.latitude) &&
    point.latitude >= -90 &&
    point.latitude <= 90 &&
    typeof point.longitude === "number" &&
    Number.isFinite(point.longitude) &&
    point.longitude >= -180 &&
    point.longitude <= 180
  );
}

function isValidIsoLikeTimestamp(timestamp: string): boolean {
  return /^\d{4}-\d{2}-\d{2}T/.test(timestamp) && !Number.isNaN(Date.parse(timestamp));
}

function toCoordinate(
  point: GeoPoint & { latitude: number; longitude: number },
): LongitudeLatitude {
  return [roundCoordinate(point.longitude), roundCoordinate(point.latitude)];
}

function interpolate(start: number, end: number, ratio: number): number {
  return roundCoordinate(start + (end - start) * ratio);
}

function normalizeLongitude(longitude: number): number {
  if (longitude > 180) {
    return longitude - 360;
  }
  if (longitude < -180) {
    return longitude + 360;
  }
  return longitude;
}

function roundCoordinate(value: number): number {
  return Number(value.toFixed(6));
}
