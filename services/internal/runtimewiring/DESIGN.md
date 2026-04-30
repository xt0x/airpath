# Runtime Wiring Design

The `runtimewiring` package owns runtime adapter selection and dependency assembly for Lambda entrypoints and command packages.

- `BackendFromEnv` selects memory mode when `AIRPATH_RUNTIME_BACKEND=memory` or when `AWS_LAMBDA_FUNCTION_NAME` is absent. Deployed Lambda execution defaults to AWS SDK-backed DynamoDB, S3, SQS, and Secrets Manager clients.
- `NewDependencies` creates the selected low-level client set. Memory clients are deterministic and used by tests and local command execution.
- Environment variable names are centralized in `env.go`, reusing runtime configuration constants where appropriate so Lambda wiring, tests, and Terraform-provided settings do not drift through repeated string literals.
- Table names, queue URLs, GeoJSON bucket name, usage budget scope, diagnostic queue URL, FlightAware credential-source presence, dispatcher activity override, and FlightAware usage estimate cost are resolved in this package. Shared environment resolution trims values and applies the same `local` environment fallback as runtime configuration.
- `NewHTTPAdapter` wires the HTTP adapter with a scoped DynamoDB repository, S3 GeoJSON repository, persistent SQS fetch queue, runtime usage guard, shared rate-limit status, and fetch policy.
- `NewPollingDispatcher` wires due-flight polling with the same persistent fetch-task idempotency, the same external-call opt-in policy used by user-triggered refresh, and an explicit runtime activity store. Polling enqueues no FlightAware work unless both runtime fetch flags are true, and flights without activity can enter the idle-stop path instead of being assumed viewer-active.
- `NewFetchProcessor` wires DynamoDB, S3 artifacts, optional diagnostics, usage budget and runtime policy checks for already queued tasks, poll schedule policy, and the FlightAware-to-application fetch adapter. It resolves the real API key and FlightAware HTTP client only when external calls are enabled; disabled runtimes use an inert client because fetch policy skips queued work before transport calls.
- The runtime FlightAware client stack is HTTP transport plus usage accounting, shared in-process rate-limit state, and max-page enforcement. Warm Lambda invocations reuse local rate-limit backoff through a package-level singleton.
- `RuntimeUsageGuard` combines runtime fetch enablement, real-call opt-in, repository budget state, and shared in-process rate-limit state. It normalizes missing budget rows, reports active backoff reset times in usage status, and ensures disabled real calls or stopped budgets disable new fetching.
- FlightAware usage accounting defaults to a non-zero estimated cost per call and can be overridden with `FLIGHTAWARE_ESTIMATED_COST_USD_PER_CALL`.
- Fetch failure diagnostics are sent only when `FETCH_TASK_DIAGNOSTIC_QUEUE_URL` is explicitly configured, preventing diagnostic payloads from falling back into the executable fetch task queue.
- `DISPATCHER_ASSUME_ACTIVE_VIEWER=true` is a runtime override for local or demo polling scenarios that need the dispatcher to treat due flights as active without a separate activity signal.
