# Terraform Design

Terraform owns AWS resource wiring for Airpath environments. It does not build application artifacts and does not store runtime secret values.

## Responsibilities

- `bootstrap` creates the shared Terraform state backend, lockfile access, encryption key, GitHub OIDC provider, and CI plan role. The role trust policy accepts explicit GitHub OIDC subject claims instead of a repository-wide wildcard; the default subject is the configured repository's `main` branch. Native Terraform tests cover state bucket hardening, KMS-backed encryption defaults, GitHub OIDC role naming, environment-scoped state and lockfile keys, and backend configuration output.
- `envs/dev` is the only deployable environment root today. It composes the reusable modules into the personal demo runtime and carries native validation-failure tests for environment scoping, artifact retention, fetch-task DLQ thresholds, IAM policy semantics, and plan-value contracts for FlightAware runtime flags, secret-reference outputs, and dev naming.
- `envs/stg` and `envs/prod` validate provider, partial S3 backend, variable, and output shape only. They carry native Terraform tests for environment defaults and invalid environment rejection.
- `modules/api-http` owns HTTP API v2 routing to the API Lambda, stage access logging, access-log retention, and default stage throttling.
- `modules/compute-lambda` owns Lambda execution roles, basic logging, inline least-privilege policy attachment, runtime settings, artifact references, environment variables, and bounded CloudWatch log retention. Lambda functions are created after their managed log group and execution-role policy attachments to make initial applies deterministic. The module exposes planned environment variables as module output metadata for root-level Terraform plan tests; root outputs still expose only environment-appropriate deployment metadata.
- `modules/data-dynamodb` owns flight cache, lookup/idempotency, position history, and usage budget tables.
- `modules/eventing` owns the encrypted fetch task queue, encrypted DLQ, configurable queue visibility timeout, queue-consume permissions, fetcher SQS event source mapping, dispatcher schedule, and EventBridge invoke permission. It creates the fetcher event source mapping only after the consume policy is attached to satisfy AWS Lambda's SQS source validation.
- `modules/storage-s3` owns route and track GeoJSON artifact storage.
- `modules/secrets` owns Secrets Manager secret metadata only. Secret values are created or rotated outside Terraform state.
- `modules/observability` owns dashboards and CloudWatch alarms for Lambda, SQS, FlightAware, and budget signals. It receives the concrete AWS region for dashboard widgets and the Lambda timeout seconds keyed by function name so timeout alarms track each function's configured runtime window. Alarm names strip an already-present environment prefix from Lambda-specific suffixes to avoid duplicated names.

Each reusable module declares its own Terraform and AWS provider requirements and carries native Terraform tests in `tests/*.tftest.hcl`. These tests use Terraform mock providers so module contracts are checked through Terraform-interpreted plan/apply values without AWS credentials or live AWS resources. Input validation belongs at the nearest owner: reusable modules reject invalid reusable inputs such as Lambda memory and timeout ranges, log retention periods, API throttle limits, S3 retention periods, fetch-task DLQ thresholds, and fetch-task visibility timeouts; environment roots reject invalid environment-scoped values before they reach composed modules.

The staging and production roots are intentionally non-deployable placeholders. Their native Terraform tests prove the root owns exactly one environment name.

Terraform security policy scanning uses Checkov from `.checkov.yml` through `make terraform-policy`. TFLint runs against bootstrap, environment roots, and reusable modules, and CI runs the full `make terraform-check` target so native Terraform tests are enforced with fmt, validate, lint, and policy checks. `make terraform-test` discovers Terraform directories from committed `tests/*.tftest.hcl` files instead of a hand-maintained module list, so new tested roots and modules enter the native test suite automatically. The selected Checkov baseline covers S3 public-access controls, S3 encryption, SQS encryption, and public access block attachment. The bootstrap CI plan role can read Terraform state objects but can write or delete only native `.tflock` lock objects. Its general S3 plan-discovery permissions are limited to bucket and configuration metadata without object-key enumeration; object content reads stay in the dedicated Terraform state policy. GitHub OIDC subjects are explicit inputs so broad repository wildcards are not required for CI access.

## Runtime Contract

The dev root mirrors the runtime contract consumed by `services/internal/runtimewiring` and the Lambda entrypoints.

- API receives environment, personal-demo notice, FlightAware fetch gates, the FlightAware secret ARN, DynamoDB table names, GeoJSON bucket name, and fetch task queue URL.
- Fetcher receives environment, mode, FlightAware fetch gates, the FlightAware secret ARN, DynamoDB table names, GeoJSON bucket name, usage budget table name, and diagnostic queue URL.
- Dispatcher receives environment, mode, FlightAware fetch gates, fetch task queue URL, flight table name, lookup table name, usage budget table name, and the dev-only active-viewer override.

IAM policies follow those boundaries:

- API can read and update cache/usage state, read/write GeoJSON artifacts, enqueue fetch tasks, and inspect the configured secret reference without reading the secret value.
- Fetcher can consume fetch task batches, update DynamoDB lease/cache/position/usage state including transactional writes, write GeoJSON artifacts, read the configured FlightAware secret value, and send safe diagnostics to the DLQ-backed diagnostic queue.
- Dispatcher can query due polling state through the flight table GSI, update poll state, reserve or release fetch task idempotency records in the lookup table, read usage budget state, and send fetch tasks to SQS.

Native dev-root Terraform tests decode the rendered IAM policy JSON to enforce those boundaries semantically. They keep secret value access fetcher-only, prevent dispatcher S3 and Secrets Manager permissions, and verify SQS/DynamoDB resource scopes against the module outputs that compose the dev environment.

Dev-root plan-value tests inspect Terraform-interpreted module outputs rather than raw HCL strings. They prove that real FlightAware calls are disabled by default, that `allow_real_flightaware_calls = true` switches the API, fetcher, and dispatcher fetch flags plus the fetcher mode, that the FlightAware root output remains secret ARN metadata only, and that planned Lambda, data, storage, and eventing names keep the `airpath-dev-*` prefix.

## Secret Handling

Terraform variables and outputs carry only secret names or ARNs. Raw API key values must not appear in Terraform variables, outputs, locals, tfvars examples, plan files, or state produced by Terraform-managed resources. Runtime code may resolve the FlightAware key from `FLIGHTAWARE_API_KEY_SECRET_ARN` through Secrets Manager, while local development can use `FLIGHTAWARE_API_KEY` from the process environment outside Terraform.
