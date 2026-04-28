# Airpath Operations Runbook

This runbook covers the Personal free-allowance MVP operations path. Do not place raw API keys, bearer tokens, cookies, or Terraform state excerpts in incidents, tickets, logs, or screenshots.

## Manual Stop And Resume

Use the runtime configuration flag to stop or resume real FlightAware fetches.

- Stop: set `FLIGHTAWARE_FETCH_ENABLED=false` and set `FLIGHTAWARE_FETCH_DISABLED_REASON` to a short operator reason such as `operator_budget_stop`.
- Resume: set `FLIGHTAWARE_FETCH_ENABLED=true` and clear `FLIGHTAWARE_FETCH_DISABLED_REASON`.
- Verify: call `/v1/admin/fetch-control` and confirm `fetchingEnabled` and `reason`.
- During stop, cached responses may still be served, but new FlightAware fetch tasks should not be created.

## Budget Exhaustion

The free-allowance guard uses local estimated spend and account usage reconciliation. Treat the soft-threshold alarm as an early warning and the hard-stop alarm as a fetch freeze.

1. Confirm the CloudWatch dashboard values for `EstimatedSpendUSD`, `BudgetStopCount`, and `BudgetHardStopCount`.
2. Stop real fetches with `FLIGHTAWARE_FETCH_ENABLED=false` if the hard stop has not already done so.
3. Keep the app in cache-first mode and communicate that data may be stale.
4. Reconcile with FlightAware account usage before resuming.

## 429 Rate Limiting

429 alarms mean FlightAware has rate-limited at least one request or the local rate guard emitted a stop event.

1. Check the dashboard for `RateLimitedCount` and recent Lambda errors.
2. Leave background polling disabled until the reset window has passed.
3. Prefer cached data and manual refresh only after the guard clears.
4. Do not increase polling frequency to catch up.

## Key Rotation

FlightAware API keys must remain backend-only and stored in Secrets Manager or environment variables.

1. Create or update the Secrets Manager value outside Terraform state.
2. Redeploy or refresh Lambda configuration if the secret reference changes.
3. Confirm `/v1/health` reports `flightawareSecretPresent=true`.
4. Never paste raw keys into Terraform variables, logs, pull requests, or runbook notes.

## Stale Cache Behavior

When fetches are stopped, rate-limited, or unavailable, the API should return cached data when available and mark freshness as stale.

1. Confirm the frontend displays cached or stale state without implying realtime accuracy.
2. Check `FlightPositions`, route/track GeoJSON objects, and cache timestamps before declaring data unavailable.
3. If stale cache is unavailable, return a typed `stale_cache_unavailable` response instead of forcing a real fetch during a stop.
4. Resume fetches only after budget and rate-limit checks are clear.
