# GeoJSON Backend Design

The `geojson` package owns dependency-free backend GeoJSON feature construction for persisted route and track map artifacts.

- `LineFeature` builds GeoJSON `Feature` values with `MultiLineString` geometry and `kind`/`source` properties.
- Invalid points are filtered before feature construction. Points with NaN, infinite, out-of-range longitude, or out-of-range latitude values are ignored.
- Features require at least two valid points and at least one non-degenerate segment; otherwise construction returns unavailable.
- Coordinates are rounded to six decimal places for stable persisted artifacts and deterministic tests. Valid in-range longitudes are preserved as supplied, including `+180`.
- Lines crossing the antimeridian are split into separate segments with interpolated boundary points at `+/-180` degrees. Split output omits zero-length segments when an input point is already on the boundary.
- `+180` and `-180` endpoints with different latitudes are treated as the same antimeridian boundary line. If split and boundary handling leaves no drawable segment, feature construction returns unavailable instead of emitting an empty `MultiLineString`.
- Multi-point routes carry forward the effective longitude emitted in the previous segment, so a rewritten boundary point cannot make the next leg draw across the map.
- FlightAware and S3 adapter code call this package instead of hand-building geometry maps.
- Tests compare the backend antimeridian split against the same shared fixture used by the TypeScript geo package.
