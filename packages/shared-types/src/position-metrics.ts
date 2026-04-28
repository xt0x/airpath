export interface FlightPositionMetricsInput {
  altitudeHundredsFeet: number | null | undefined;
  groundspeedKnots: number | null | undefined;
  headingDegrees: number | null | undefined;
}

export interface FlightPositionMetrics {
  altitudeHundredsFeet: number | null;
  altitudeFeet: number | null;
  groundspeedKnots: number | null;
  headingDegrees: number | null;
}

export function normalizeAltitudeFeet(
  altitudeHundredsFeet: number | null | undefined,
): number | null {
  return altitudeHundredsFeet === null || altitudeHundredsFeet === undefined
    ? null
    : altitudeHundredsFeet * 100;
}

export function normalizeGroundspeedKnots(
  groundspeedKnots: number | null | undefined,
): number | null {
  return groundspeedKnots ?? null;
}

export function normalizeHeadingDegrees(headingDegrees: number | null | undefined): number | null {
  if (headingDegrees === null || headingDegrees === undefined) {
    return null;
  }

  return headingDegrees === 360 ? 0 : headingDegrees;
}

export function normalizeFlightPositionMetrics(
  input: FlightPositionMetricsInput,
): FlightPositionMetrics {
  const altitudeHundredsFeet = input.altitudeHundredsFeet ?? null;

  return {
    altitudeHundredsFeet,
    altitudeFeet: normalizeAltitudeFeet(altitudeHundredsFeet),
    groundspeedKnots: normalizeGroundspeedKnots(input.groundspeedKnots),
    headingDegrees: normalizeHeadingDegrees(input.headingDegrees),
  };
}
