# Runtime Config Design

The `runtimeconfig` package owns environment-derived service configuration that is shared by Lambda entrypoints.

FlightAware real-call access is guarded by two explicit runtime flags:

- `FLIGHTAWARE_FETCH_ENABLED` allows fetch workflows to run.
- `FLIGHTAWARE_REAL_CALLS_ENABLED` allows real FlightAware API calls.

Both flags must be set to `true` before external FlightAware calls are allowed. Missing or malformed values default to disabled so personal demo deployments stay cache-first and mock-safe unless an operator opts in.
