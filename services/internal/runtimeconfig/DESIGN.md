# Runtime Config Design

The `runtimeconfig` package owns environment-derived service configuration shared by Lambda entrypoints and runtime wiring.

- `LoadFlightAwareRuntimeConfig` requires an explicit environment lookup function so tests and commands can provide deterministic configuration.
- `AIRPATH_ENVIRONMENT` identifies the deployment environment used by usage-budget scope and response metadata. Missing or blank values default to `local`.
- `AIRPATH_PERSONAL_DEMO_NOTICE` can override the default personal non-commercial demo notice returned by control-plane routes.
- `FLIGHTAWARE_FETCH_ENABLED` allows fetch workflows to enqueue and run.
- `FLIGHTAWARE_REAL_CALLS_ENABLED` allows real FlightAware API calls.
- `FLIGHTAWARE_FETCH_DISABLED_REASON` carries operator-facing context when fetches are disabled.
- `ExternalCallsAllowed` returns true only when both fetch and real-call flags are explicitly true.
- `BoolEnv` treats only trimmed, case-insensitive `true` as enabled. Missing, malformed, or nil lookup values default to disabled so personal demo deployments stay cache-first and mock-safe unless an operator opts in.
