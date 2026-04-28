# Services

This directory contains Go Lambda service implementations.

- `api`: HTTP API Lambda
- `fetcher`: Fetch worker Lambda
- `dispatcher`: scheduled due-flight dispatcher Lambda

The Lambda entrypoints are deployable shell implementations for the personal dev demo. Runtime configuration keeps real FlightAware calls disabled unless both `FLIGHTAWARE_FETCH_ENABLED=true` and `FLIGHTAWARE_REAL_CALLS_ENABLED=true` are present.

The free-allowance MVP intentionally excludes WebSocket delivery and FlightAware Alerts receiver services.
