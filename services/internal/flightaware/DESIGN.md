# FlightAware Client Boundary Design

The `flightaware` package owns the backend boundary around FlightAware AeroAPI access. It defines a small Go port for the Personal MVP endpoints and keeps real network transport outside application use cases until guarded integration work is added.

- `Client` abstracts search, summary, route, position, track, schedules, and account usage.
- `FixtureClient` is deterministic test infrastructure and never performs external calls.
- `MaxPagesClient` normalizes list requests to `max_pages = 1` by default and rejects larger values before transport.
- `UsageAccountingClient` records before/after local usage estimates around each endpoint call.
- `RateLimitedClient` blocks endpoints when local stop state is active or an upstream 429 is observed, and 429 backoff also pauses low-priority route, track, and schedule calls.
- Redaction helpers sanitize sensitive headers and log text before structured logging.

This package must not hardcode API keys. Real credentials are loaded by later runtime adapters from environment variables or Secrets Manager and passed only to transport code.
