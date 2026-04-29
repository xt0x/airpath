# Services

This directory contains Go Lambda service implementations.

- `cmd/api`: HTTP API Lambda entrypoint
- `cmd/fetcher`: Fetch worker Lambda entrypoint
- `cmd/dispatcher`: scheduled due-flight dispatcher Lambda entrypoint
- `internal/application`: use cases and ports
- `internal/domain`: dependency-free domain models and helpers
- `internal/*integration`: adapter implementations for infrastructure boundaries

The Lambda entrypoints are deployable shell implementations for the personal dev demo. Runtime configuration keeps real FlightAware calls disabled unless both `FLIGHTAWARE_FETCH_ENABLED=true` and `FLIGHTAWARE_REAL_CALLS_ENABLED=true` are present.

The free-allowance MVP intentionally excludes WebSocket delivery and FlightAware Alerts receiver services.
