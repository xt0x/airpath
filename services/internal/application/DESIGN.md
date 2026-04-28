# Application Layer Design

The `application` package owns backend use cases that can run without AWS adapters. It depends on domain models and small ports for cached flight data, map artifacts, position history, fetch task enqueueing, and usage guards.

- Search is cache-first. Cache misses can enqueue a bounded summary fetch task only when the usage guard allows fetching.
- Flight detail assembles cached summary, planned route, actual track, current position, and freshness metadata.
- Map data assembles route, track, and current-position layers from cached repositories.
- Refresh requests dedupe task types and enqueue only when the budget/rate guard allows fetching.
- Usage status returns local accounting and rate-limit state without calling AWS.
- Error mapping converts application and FlightAware boundary errors into the shared typed API error schema.
