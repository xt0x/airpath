# FlightAware Client Boundary Design

The `flightaware` package owns the backend boundary around FlightAware AeroAPI access. It defines a small Go port for the Personal MVP endpoints and keeps real network transport outside application use cases until guarded integration work is added.

- `Client` abstracts search, summary, route, position, track, schedules, and account usage.
- `FixtureClient` is deterministic test infrastructure and never performs external calls.
- `UsageAccountingClient` records before/after local usage estimates around each endpoint call.
- `RateLimitedClient` blocks endpoints when local stop state is active or an upstream 429 is observed.
- Redaction helpers sanitize sensitive headers and log text before structured logging.

This package must not hardcode API keys. Real credentials are loaded by later runtime adapters from environment variables or Secrets Manager and passed only to transport code.
