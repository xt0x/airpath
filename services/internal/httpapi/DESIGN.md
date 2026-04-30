# HTTP API Adapter Design

The `httpapi` package maps API Gateway HTTP API v2 events to application use cases and typed JSON responses. It keeps Lambda event parsing separate from application logic so handlers can be tested without AWS.

- The adapter surface covers `GET /v1/flights/search`, `GET /v1/flights/{flightId}`, `GET /v1/flights/{flightId}/map-data`, `GET /v1/flights/{flightId}/positions`, `POST /v1/flights/{flightId}/refresh`, and `GET /v1/usage/status`.
- Route constants live in `routes.go` and mirror the shared TypeScript route templates. Tests guard route drift against the shared contract.
- Read-only routes are method-gated at the adapter boundary. Unsupported or missing methods return the typed not-found response without invoking read use cases.
- Search requires a trimmed `ident` query value with at least two characters. Date-scoped search is not implemented by the application use case, so a supplied `date` query is rejected before calling the application layer.
- Flight resource routes reject empty path IDs, decode the single flight ID path segment after route splitting, and reject malformed path escapes before calling the application layer.
- Map-data rejects unsupported `include` and `simplify` query parameters instead of ignoring them.
- Position history accepts optional RFC3339 `since` and numeric `limit` query values, normalizes `since` to the second-precision UTC storage format, rejects unsupported `quality`, rejects present-but-empty query values, defaults to 200 items, and caps `limit` at 500.
- Refresh accepts one strict JSON object with unique public refresh task types and a public client reason. The adapter rejects unknown fields, trailing JSON, summary tasks, internal worker reasons, and invalid enum values; the application layer then rejects invalid task/reason pairs before any queue write.
- Application errors are mapped through the shared typed API error response with status codes for validation, budget/fetch-disabled, rate-limit, stale-cache/not-found, and upstream failure cases.
- JSON responses always use `content-type: application/json` and browser-safe CORS headers. OPTIONS requests return an empty 204 preflight response without invoking application use cases.
