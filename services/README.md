# Services

This directory contains Go Lambda service implementations.

- `cmd/api`: HTTP API Lambda entrypoint
- `cmd/fetcher`: Fetch worker Lambda entrypoint
- `cmd/dispatcher`: scheduled due-flight dispatcher Lambda entrypoint
- `internal/application`: use cases and ports
- `internal/domain`: dependency-free domain models and helpers
- `internal/*integration`: adapter implementations for infrastructure boundaries
- `internal/runtimewiring`: runtime selection of AWS adapters in Lambda and memory adapters for local tests
- `internal/geojson`: dependency-free backend GeoJSON feature construction

The Lambda entrypoints wire the application layer to runtime adapters. Local runs default to in-memory adapters, while deployed Lambda runs use AWS-backed DynamoDB, S3, SQS, and Secrets Manager clients unless `AIRPATH_RUNTIME_BACKEND=memory` is set explicitly. Runtime configuration keeps real FlightAware calls disabled unless both `FLIGHTAWARE_FETCH_ENABLED=true` and `FLIGHTAWARE_REAL_CALLS_ENABLED=true` are present.

The free-allowance MVP intentionally excludes WebSocket delivery and FlightAware Alerts receiver services.
