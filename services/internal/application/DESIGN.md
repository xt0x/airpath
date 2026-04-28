# Application Layer Design

The `application` package owns backend use cases that can run without AWS adapters. It depends on domain models and small ports for cached flight data, map artifacts, position history, fetch task enqueueing, and usage guards.

- Search is cache-first. Cache misses can enqueue a bounded summary fetch task only when the usage guard allows fetching.
- Flight detail assembles cached summary, planned route, actual track, current position, and freshness metadata.
- Map data assembles route, track, and current-position layers from cached repositories.
- Refresh requests dedupe task types and enqueue only when the budget/rate guard allows fetching.
- Refresh task idempotency is based on `flightId`, `taskType`, and a five-minute request window so repeated requests do not create extra upstream fetch work.
- Runtime fetch policy can disable route, track/final-track, and background polling tasks without changing use case code.
- When fetching is disabled but cached flight data exists, refresh returns an empty task list with stale cache metadata instead of forcing an upstream fetch error.
- Usage status returns local accounting and rate-limit state without calling AWS.
- Usage budget state is normalized around a monthly environment scope and defaults to a USD 4.00 soft stop threshold.
- Error mapping converts application and FlightAware boundary errors into the shared typed API error schema.
- F15 polling owns conservative low-frequency scheduling, due-flight dispatch, fetch task execution, flight-level leases, route/track artifact updates, position history appends, idle stop behavior, and safe diagnostic metadata for failed fetch tasks.
