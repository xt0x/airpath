# GeoJSON Backend Design

The `geojson` package owns dependency-free backend GeoJSON feature construction for persisted route and track map artifacts.

It keeps FlightAware and S3 adapter code from hand-building geometry maps directly. Line features are emitted as `MultiLineString` geometries and split at the antimeridian so backend artifacts follow the same map-data contract as the TypeScript geo package.

The backend antimeridian split is tested against the same shared fixture used by the TypeScript geo package, including the exact interpolated boundary coordinates at +/-180 degrees.
