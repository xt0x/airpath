export type CoordinatePair = [number, number];

const MIN_LONGITUDE = -180;
const MAX_LONGITUDE = 180;
const MIN_LATITUDE = -90;
const MAX_LATITUDE = 90;

export function coordinatePairFromValue(value: unknown): CoordinatePair | null {
  if (
    !Array.isArray(value) ||
    value.length < 2 ||
    typeof value[0] !== "number" ||
    typeof value[1] !== "number" ||
    !Number.isFinite(value[0]) ||
    !Number.isFinite(value[1]) ||
    value[0] < MIN_LONGITUDE ||
    value[0] > MAX_LONGITUDE ||
    value[1] < MIN_LATITUDE ||
    value[1] > MAX_LATITUDE
  ) {
    return null;
  }

  return [value[0], value[1]];
}

export function collectFiniteCoordinatePairs(value: unknown): CoordinatePair[] {
  const coordinate = coordinatePairFromValue(value);
  if (coordinate !== null) {
    return [coordinate];
  }

  if (!Array.isArray(value)) {
    return [];
  }

  return value.flatMap((entry) => collectFiniteCoordinatePairs(entry));
}
