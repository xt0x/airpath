# Airpath Operations Runbook

This runbook covers backend service operations for the Personal free-allowance MVP. Do not place raw API keys, bearer tokens, cookies, Terraform state excerpts, SQS message bodies, or unredacted incident screenshots in tickets, logs, pull requests, or runbook notes.

## Service Surface

The API Lambda exposes operational probes before it initializes storage adapters:

- `GET /v1/health` reports service identity, environment, personal-demo notice, FlightAware runtime flags, whether external FlightAware calls are allowed, and whether a FlightAware credential source is configured.
- `GET /v1/admin/fetch-control` reports fetch enablement, real-call enablement, effective external-call permission, and the operator disabled reason.

Both control-plane routes support `OPTIONS` and reject non-GET methods as typed not-found responses.

Public service routes:

- `GET /v1/flights/search?ident=...`
- `GET /v1/flights/{flightId}`
- `GET /v1/flights/{flightId}/map-data`
- `GET /v1/flights/{flightId}/positions?since=...&limit=...`
- `POST /v1/flights/{flightId}/refresh`
- `GET /v1/usage/status`

Use `/v1/admin/fetch-control` to verify configured runtime gates. Use `/v1/usage/status` to verify effective fetching state after budget, rate-limit, and runtime policy are combined.

## Runtime Fetch Gates

Real FlightAware work requires both runtime flags:

- `FLIGHTAWARE_FETCH_ENABLED=true`
- `FLIGHTAWARE_REAL_CALLS_ENABLED=true`

The same policy is checked when work is enqueued and again when the fetcher processes queued records. Disabling either flag prevents new external work and causes already queued valid fetch tasks to be skipped before FlightAware credentials are resolved or external calls are attempted.

`FLIGHTAWARE_FETCH_DISABLED_REASON` is operator-facing context only. It does not disable fetches by itself.

## Manual Stop And Resume

Use runtime configuration flags to stop or resume FlightAware fetch workflows.

1. Stop enqueueing and worker execution: set `FLIGHTAWARE_FETCH_ENABLED=false`.
2. Set `FLIGHTAWARE_FETCH_DISABLED_REASON` to a short operator reason such as `operator_budget_stop`.
3. To stop only real external calls while preserving configured fetch mode visibility, set `FLIGHTAWARE_REAL_CALLS_ENABLED=false`.
4. Verify `/v1/admin/fetch-control` fields: `fetchingEnabled`, `realCallsEnabled`, `externalCallsAllowed`, and `reason`.
5. Verify `/v1/usage/status` fields: `fetchingEnabled`, `budget.stopped`, `rateLimit.limited`, and `rateLimit.resetAt`.

To resume real fetches, set both runtime flags to `true`, clear `FLIGHTAWARE_FETCH_DISABLED_REASON`, and confirm `/v1/usage/status.fetchingEnabled=true`. Do not resume until budget and rate-limit checks are also clear.

Cached responses may still be served while fetches are stopped. Treat returned data as cache-first and potentially stale.

## Budget Exhaustion

The free-allowance guard uses local estimated spend plus FlightAware account usage reconciliation. Treat the soft-threshold alarm as an early warning and the hard-stop alarm as a fetch freeze.

1. Confirm CloudWatch dashboard values for `EstimatedSpendUSD`, `BudgetStopCount`, and `BudgetHardStopCount`.
2. Call `/v1/usage/status` and check `budget.environment`, `budget.month`, `budget.estimatedMonthToDateCost`, `budget.softStopThreshold`, `budget.stopped`, and `fetchingEnabled`.
3. Stop fetch workflows with `FLIGHTAWARE_FETCH_ENABLED=false` if the hard stop has not already disabled effective fetching.
4. Keep the app in cache-first mode and communicate that returned data may be stale.
5. Reconcile with FlightAware account usage before resuming.

Do not lower local month-to-date spend during reconciliation. The backend keeps the higher local or remote month-to-date cost so delayed FlightAware usage cannot reopen fetching incorrectly.

## 429 Rate Limiting

429 alarms mean FlightAware rate-limited at least one request or the local rate guard emitted a stop event.

1. Check CloudWatch for `RateLimitedCount`, recent fetcher errors, and queue age.
2. Call `/v1/usage/status` and inspect `rateLimit.limited` and `rateLimit.resetAt`.
3. Check fetcher diagnostic messages or DLQ messages for safe failure metadata. Do not paste full queue payloads into tickets.
4. Leave background polling stopped until the reset window has passed.
5. Prefer cached data and manual refresh only after the guard clears.
6. Do not increase polling frequency to catch up.

The fetcher wraps real FlightAware calls with usage accounting, local rate limiting, and `max_pages=1` enforcement. A 429 can also pause lower-priority route, track, and schedule calls during the backoff window.

## Queue And Worker Checks

The dispatcher Lambda reads due flights and enqueues fetch tasks. The fetcher Lambda consumes SQS fetch tasks and performs real FlightAware calls only when both runtime flags allow external calls.

Dispatcher checks:

- `DISPATCHER_MODE` is telemetry only.
- `NOOP_FETCH_ENABLED=true` makes the dispatcher return without enqueueing work.
- `FETCH_TASK_QUEUE_URL` must be non-empty in AWS runtime or the dispatcher fails fast.
- `DISPATCHER_ASSUME_ACTIVE_VIEWER=true` is a local/demo override that treats due flights as active without a separate activity signal.
- Lambda response field `EnqueueAttempted` should be true only when at least one due task was enqueued.

Fetcher checks:

- `FETCHER_MODE` is telemetry only.
- Lambda response fields include `recordsReceived`, `flightawareSecretReady`, `flightawareFetchEnabled`, `flightawareRealCallsEnabled`, `externalFetchAllowed`, `externalFetchAttempted`, and optional `batchItemFailures`.
- Valid records skipped by budget or runtime policy should not appear in `batchItemFailures`.
- Malformed SQS message bodies and record-level processing errors should appear in `batchItemFailures`.
- Keep the Lambda SQS event source mapping configured with `ReportBatchItemFailures` so successful or policy-skipped records are deleted and only failed records are retried.

Queue and diagnostic behavior:

- Fetch task delivery uses process-local and optional DynamoDB-backed idempotency reservations.
- If SQS send fails before delivery, local and persistent reservations are released so a retry can enqueue the task.
- Persistent reservations expire through DynamoDB TTL so dispatch can recover from lost, DLQ'd, or terminally skipped tasks.
- `FETCH_TASK_DIAGNOSTIC_QUEUE_URL` is optional and must point to the diagnostic or DLQ destination, not the executable fetch task queue.
- Diagnostic messages contain safe failure metadata; still treat full queue payloads as operational data and avoid pasting them into tickets.

## Stale Cache Behavior

When fetches are stopped, rate-limited, over budget, or unavailable, the API should return cached data when available and mark freshness accordingly.

1. Confirm the frontend displays cached or stale state without implying realtime accuracy.
2. Check cached flight records, position history, route/track GeoJSON objects, and cache timestamps before declaring data unavailable.
3. If stale cache is unavailable, expect a typed `stale_cache_unavailable` response instead of forcing a real fetch during a stop.
4. Resume fetches only after runtime flags, budget state, and rate-limit state are clear.

Route and track artifacts are content-addressed for fetched data. A stale worker should not overwrite the artifact key currently referenced by cached flight state.

## Key Rotation

FlightAware API keys must remain backend-only and must be stored in Secrets Manager or injected through environment variables outside repository content.

1. Create or update the Secrets Manager value outside Terraform state.
2. If the secret reference changes, update `FLIGHTAWARE_API_KEY_SECRET_ARN` in Lambda configuration and redeploy or refresh the affected functions.
3. Confirm `/v1/health` reports `flightawareSecretPresent=true` for the configured `FLIGHTAWARE_API_KEY` or `FLIGHTAWARE_API_KEY_SECRET_ARN` source.
4. Run opt-in real-call verification only from an approved environment with `AIRPATH_FLIGHTAWARE_REAL_TESTS=true`, `FLIGHTAWARE_API_KEY`, and non-production-safe test identifiers.
5. Never paste raw keys into Terraform variables, logs, pull requests, SQS messages, or runbook notes.

For local verification, `FLIGHTAWARE_API_KEY` may be used as an environment variable. It must not be committed or hardcoded.

## Runtime Backend Selection

`AIRPATH_RUNTIME_BACKEND=memory` forces deterministic in-memory adapters for local execution and tests. Outside Lambda, memory mode is the default. Inside Lambda, AWS SDK-backed DynamoDB, S3, SQS, and Secrets Manager clients are the default unless memory mode is explicitly set.

Use memory mode only for local command execution and tests. Deployed environments should use AWS-backed adapters so cache, queue, artifact, lease, and usage-budget state is shared across invocations.

## Required Runtime Configuration

Infrastructure should provide these variables for deployed services:

- `AIRPATH_ENVIRONMENT`
- `AIRPATH_PERSONAL_DEMO_NOTICE`
- `FLIGHTAWARE_FETCH_ENABLED`
- `FLIGHTAWARE_REAL_CALLS_ENABLED`
- `FLIGHTAWARE_FETCH_DISABLED_REASON`
- `FLIGHTAWARE_API_KEY_SECRET_ARN` or another approved credential source
- `FLIGHTS_TABLE_NAME`
- `FLIGHT_LOOKUP_TABLE_NAME`
- `FLIGHT_POSITIONS_TABLE_NAME`
- `USAGE_BUDGET_TABLE_NAME`
- `GEOJSON_BUCKET_NAME`
- `FETCH_TASK_QUEUE_URL`

Optional runtime settings:

- `AIRPATH_RUNTIME_BACKEND=memory` forces memory adapters.
- `FLIGHTAWARE_API_KEY` can supply a local or externally injected API key source.
- `FETCH_TASK_DIAGNOSTIC_QUEUE_URL` enables safe fetch failure diagnostics.
- `DISPATCHER_ASSUME_ACTIVE_VIEWER=true` enables active-viewer polling behavior for local/demo dispatch.
- `FLIGHTAWARE_ESTIMATED_COST_USD_PER_CALL` overrides the non-zero default usage estimate applied before real upstream calls.

## Verification

Run service checks from `services/` after operational code changes:

```sh
go test ./...
go vet ./...
```

For runbook-only edits, at minimum run:

```sh
git diff --check -- services/RUNBOOK.md
```
