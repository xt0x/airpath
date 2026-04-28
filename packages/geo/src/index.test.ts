import { describe, expect, it } from "vitest";

import {
  buildCurrentPositionFeature,
  buildPlannedRouteFeature,
  buildTrackFeatures,
  simplifyTrackPoints,
  splitLineStringForAntimeridian,
} from "./index.js";

describe("planned route GeoJSON", () => {
  it("builds a planned route from decoded route points", () => {
    const feature = buildPlannedRouteFeature({
      routePoints: [
        { name: "KSFO", latitude: 37.6188056, longitude: -122.3754167 },
        { name: "OCEANIC", latitude: 45.1, longitude: -150.2 },
        { name: "RJTT", latitude: 35.549397, longitude: 139.779839 },
      ],
      routeText: "KSFO OCEANIC RJTT",
      origin: null,
      destination: null,
    });

    expect(feature).not.toBeNull();
    expect(feature?.geometry.type).toBe("MultiLineString");
    expect(feature?.properties).toMatchObject({
      kind: "planned_route",
      source: "flightaware_route",
      routeText: "KSFO OCEANIC RJTT",
      pointCount: 3,
    });
  });

  it("falls back to an airport great-circle route when decoded route points are unavailable", () => {
    const feature = buildPlannedRouteFeature({
      routePoints: [],
      routeText: null,
      origin: { code: "KSFO", latitude: 37.6188056, longitude: -122.3754167 },
      destination: { code: "RJTT", latitude: 35.549397, longitude: 139.779839 },
      fallbackStepCount: 8,
    });

    expect(feature).not.toBeNull();
    expect(feature?.properties).toMatchObject({
      kind: "planned_route",
      source: "airport_great_circle_fallback",
      origin: "KSFO",
      destination: "RJTT",
    });
    expect(feature?.geometry.type).toBe("MultiLineString");
    expect(feature?.geometry.coordinates.flat().length).toBeGreaterThanOrEqual(9);
  });
});

describe("track GeoJSON", () => {
  it("builds display line and point features from track points", () => {
    const features = buildTrackFeatures([
      { latitude: 37.6188056, longitude: -122.3754167, timestamp: "2026-04-25T10:27:00Z" },
      { latitude: 42.5, longitude: -140.2, timestamp: "2026-04-25T13:00:00Z" },
      { latitude: 45.123, longitude: 160.456, timestamp: "2026-04-25T15:05:00Z" },
    ]);

    expect(features.line).not.toBeNull();
    expect(features.line?.properties).toMatchObject({
      kind: "actual_track",
      source: "flightaware_track",
      lastTimestamp: "2026-04-25T15:05:00Z",
      pointCount: 3,
    });
    expect(features.points.type).toBe("FeatureCollection");
    expect(features.points.features).toHaveLength(3);
    expect(features.points.features[2]?.properties).toMatchObject({
      kind: "track_point",
      timestamp: "2026-04-25T15:05:00Z",
      sequence: 2,
    });
  });

  it("creates a current position marker only when latitude, longitude, and timestamp are valid", () => {
    expect(
      buildCurrentPositionFeature({
        latitude: 45.123,
        longitude: 160.456,
        altitudeFeet: 37000,
        groundspeedKnots: 488,
        headingDegrees: 275,
        timestamp: "2026-04-25T15:05:00Z",
      }),
    ).toMatchObject({
      type: "Feature",
      geometry: { type: "Point", coordinates: [160.456, 45.123] },
      properties: {
        kind: "current_position",
        altitudeFeet: 37000,
        groundspeedKnots: 488,
        headingDegrees: 275,
        timestamp: "2026-04-25T15:05:00Z",
      },
    });

    expect(
      buildCurrentPositionFeature({ latitude: null, longitude: 160.456, timestamp: "now" }),
    ).toBeNull();
    expect(
      buildCurrentPositionFeature({ latitude: 45.123, longitude: 181, timestamp: "now" }),
    ).toBeNull();
    expect(
      buildCurrentPositionFeature({ latitude: 45.123, longitude: 160.456, timestamp: null }),
    ).toBeNull();
  });
});

describe("antimeridian handling", () => {
  it("splits line strings crossing 180 degrees instead of drawing across the world", () => {
    const split = splitLineStringForAntimeridian([
      [170, 35],
      [-170, 40],
    ]);

    expect(split).toEqual([
      [
        [170, 35],
        [180, 37.5],
      ],
      [
        [-180, 37.5],
        [-170, 40],
      ],
    ]);
  });
});

describe("track simplification", () => {
  it("keeps departure, latest, arrival, and important points while reducing display density", () => {
    const simplified = simplifyTrackPoints(
      [
        { latitude: 37, longitude: -122, timestamp: "2026-04-25T10:00:00Z", kind: "departure" },
        { latitude: 38, longitude: -130, timestamp: "2026-04-25T11:00:00Z" },
        { latitude: 39, longitude: -140, timestamp: "2026-04-25T12:00:00Z", important: true },
        { latitude: 40, longitude: -150, timestamp: "2026-04-25T13:00:00Z" },
        { latitude: 41, longitude: -160, timestamp: "2026-04-25T14:00:00Z" },
        { latitude: 42, longitude: -170, timestamp: "2026-04-25T15:00:00Z", kind: "arrival" },
        { latitude: 43, longitude: -175, timestamp: "2026-04-25T15:05:00Z", kind: "latest" },
      ],
      4,
    );

    expect(simplified.map((point) => point.timestamp)).toEqual([
      "2026-04-25T10:00:00Z",
      "2026-04-25T12:00:00Z",
      "2026-04-25T15:00:00Z",
      "2026-04-25T15:05:00Z",
    ]);
  });
});
