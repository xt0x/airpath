# Airpath Services

This directory contains the Go backend services for Airpath. The backend is organized around Clean Architecture boundaries: Lambda commands are thin entrypoints, `internal/application` owns use cases and ports, `internal/domain` owns dependency-free models and helpers, and adapter packages translate AWS, FlightAware, HTTP, and runtime concerns into those ports.

## Entry Points

- `cmd/api` handles API Gateway HTTP API v2 requests. It serves `/v1/health` and `/v1/admin/fetch-control` as static, read-only, CORS-aware control-plane routes before initializing storage adapters, then delegates application routes to `internal/httpapi`.
- `cmd/fetcher` handles SQS fetch-task batches. It decodes each message into an application fetch task, uses a record-specific worker ID for flight leases, and returns Lambda partial batch failures only for records that failed parsing or processing.
- `cmd/dispatcher` handles scheduled polling invocations. It reads due poll schedules from storage, enqueues fetch tasks through the same idempotent queue path used by user refresh, and fails fast in AWS runtime when `FETCH_TASK_QUEUE_URL` is missing.

## Package Responsibilities

- `internal/application` owns cache-first flight search/detail/map/position/refresh/usage use cases, fetch-task dispatch, fetch processing, polling schedules, usage budget decisions, typed API errors, diagnostics, and application-owned DTOs. It depends only on domain models and small ports.
- `internal/domain` owns dependency-free flight, position, event, ID, polling, time, duration, nullable-value, and position-metric types/helpers. It has no AWS, HTTP, FlightAware transport, or presentation dependencies.
- `internal/httpapi` maps API Gateway events onto application inputs for flight search, detail, map data, position history, refresh, and usage status. It owns route constants, method gates, query/body validation, typed JSON error responses, and CORS headers.
- `internal/awsintegration` implements DynamoDB, S3, SQS, Secrets Manager, and FlightAware boundary adapters behind small local interfaces. DynamoDB stores cache rows, lookup rows, position history, polling state, fetch-task idempotency reservations, leases, and usage budgets; S3 stores route and track GeoJSON artifacts; SQS stores fetch tasks and optional diagnostics.
- `internal/flightaware` owns the AeroAPI client boundary: request/response DTOs, fixture client, HTTP transport, max-page guard, usage accounting decorator, rate-limit decorator, typed errors, and redaction helpers.
- `internal/geojson` builds dependency-free GeoJSON line features for persisted route and track artifacts, including coordinate validation, rounding, and antimeridian splitting.
- `internal/display` contains backend presentation helpers for nullable domain values. Domain code keeps stable missing-value reason codes; this package converts them to display-ready labels and strings.
- `internal/runtimeconfig` loads operator-facing runtime fetch configuration from environment variables.
- `internal/runtimewiring` selects memory or AWS adapters, resolves environment-derived table/bucket/queue/credential settings, shares in-process FlightAware rate-limit state across warm Lambda invocations, and assembles API, fetcher, and dispatcher dependencies.
- `internal/cmdsupport` is the small command-support wrapper for clock, background context, and runtime backend selection so Lambda command tests can replace process concerns cleanly.

## Fetch And Polling Behavior

- Real FlightAware work requires both `FLIGHTAWARE_FETCH_ENABLED=true` and `FLIGHTAWARE_REAL_CALLS_ENABLED=true`. The same policy is checked before enqueueing work and again by fetch workers before leases or external calls.
- API search is cache-first. A cache miss can enqueue a bounded summary fetch task when usage budget and runtime fetch policy allow it.
- User refresh accepts only public refresh reasons and non-summary task types for cached flights that already have a `faFlightId`.
- Poll dispatch advances only task types that were successfully enqueued or already reserved. Policy-blocked or failed task types keep their due timestamps for retry.
- Fetch workers acquire flight-level leases before external calls and use lease-guarded writes for fetched metadata, position history, and artifacts. Track writes are atomic for transaction-sized histories and staged/hidden until committed for larger histories.
- Usage accounting reserves estimated cost before upstream calls, stops calls at the soft threshold, and reconciles FlightAware account usage with conditional max updates so newer local reservations are not overwritten.

## Runtime Environment

- `AIRPATH_RUNTIME_BACKEND=memory` forces memory adapters. Outside Lambda, memory mode is the default; inside Lambda, AWS adapters are the default.
- `AIRPATH_ENVIRONMENT` scopes usage budgets and response metadata; blank values default to `local`.
- `FLIGHTS_TABLE_NAME`, `FLIGHT_LOOKUP_TABLE_NAME`, `FLIGHT_POSITIONS_TABLE_NAME`, and `USAGE_BUDGET_TABLE_NAME` configure DynamoDB tables.
- `GEOJSON_BUCKET_NAME` configures the S3 artifact bucket.
- `FETCH_TASK_QUEUE_URL` configures executable fetch-task delivery. `FETCH_TASK_DIAGNOSTIC_QUEUE_URL` is optional and is used only for safe diagnostics.
- `DISPATCHER_ASSUME_ACTIVE_VIEWER=true` is a local/demo override for polling activity.
- `FLIGHTAWARE_API_KEY` or `FLIGHTAWARE_API_KEY_SECRET_ARN` supplies FlightAware credentials. API keys must come from environment variables or Secrets Manager and must never be hardcoded.
- `FLIGHTAWARE_ESTIMATED_COST_USD_PER_CALL` can override the non-zero default local usage estimate.

## Verification

Run service checks from this directory:

```sh
go test ./...
go vet ./...
```

Real FlightAware integration tests are opt-in only with `AIRPATH_FLIGHTAWARE_REAL_TESTS=true` plus the required test identifiers and `FLIGHTAWARE_API_KEY`.
