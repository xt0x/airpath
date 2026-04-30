# FlightAware Client Boundary Design

The `flightaware` package owns the backend boundary around FlightAware AeroAPI access. It defines the transport-facing Go client, typed endpoint DTOs, fixture client, real HTTP transport, reusable decorators, typed errors, usage accounting, rate-limit state, and redaction helpers. Application use cases do not import this package directly.

- `Client` abstracts search, summary, route, position, track, schedules, and account usage endpoints.
- Search and summary responses retain the FlightAware fields needed to seed cached domain flights: identifiers, airports, schedule and actual times, status, progress, aircraft type, registration, and filed ETE.
- The real HTTP search call sends `ident_type=designator` because application search input is treated as a flight designator. The summary call sends `ident_type=fa_flight_id`, decodes FlightAware's `flights` wrapper, and rejects empty result sets instead of returning a zero-value summary.
- Route, position, track, schedule, and usage response types are normalized into package DTOs before adapter code translates them into application-owned DTOs. The position call reads `last_position` and rejects missing position payloads.
- Position and track DTOs retain FlightAware altitude, altitude-change, groundspeed, heading, timestamp, and update-type fields. Known altitude-change and update-type codes are normalized to application-compatible labels; unknown codes are omitted instead of exposing invalid enum values.
- `FixtureClient` is deterministic test infrastructure and never performs external calls.
- `HTTPClient` is the real AeroAPI v4 transport. It requires a provided API key, uses the `x-apikey` header, defaults to `https://aeroapi.flightaware.com/aeroapi`, uses a default timeout, maps upstream 429 responses to typed rate-limit errors, and wraps non-2xx responses as `ClientError`.
- Account usage normalization maps FlightAware account totals into local month-to-date usage. `total_discount_cost` is preferred whenever present, including valid zero; `total_cost` is the fallback; `total_pages` is retained as the result-set estimate.
- `MaxPagesClient` enforces the Personal MVP list boundary by defaulting list requests to `max_pages=1` and rejecting larger values before transport.
- `ClientDecorator` centralizes pass-through behavior so endpoint-specific wrappers override only the calls they govern.
- `UsageAccountingClient` records before/after local usage estimates around each endpoint call, does not call upstream when the before-call recorder rejects the reservation, and does not hide upstream errors.
- `RateLimitedClient` blocks calls while local stop state is active, marks upstream 429s, and extends backoff to low-priority route, track, and schedule calls. Timed stops expire through `resetAt`; manual stops are indefinite and clear any stale timed reset for the same endpoint.
- Redaction helpers sanitize sensitive headers, raw secrets, and credential-like log text before structured logging.

This package must not hardcode API keys. Runtime adapters load real credentials from environment variables or Secrets Manager and pass the raw key only to transport construction. Real AeroAPI tests are opt-in through `AIRPATH_FLIGHTAWARE_REAL_TESTS=true` and are skipped by default.
