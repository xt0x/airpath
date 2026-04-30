# Terraform Design

Terraform owns AWS resource wiring for Airpath environments. It does not build application artifacts and does not store runtime secret values.

## Responsibilities

- `bootstrap` creates the shared Terraform state backend, lockfile access, encryption key, GitHub OIDC provider, and CI plan role.
- `envs/dev` is the only deployable environment root today. It composes the reusable modules into the personal demo runtime and carries native validation-failure tests for environment scoping, artifact retention, fetch-task DLQ thresholds, IAM policy semantics, and plan-value contracts for FlightAware runtime flags, secret-reference outputs, and dev naming.
- `envs/stg` and `envs/prod` validate provider, backend, variable, and output shape only. They carry native Terraform tests for environment defaults and invalid environment rejection, plus Vitest contract tests that keep resources, data sources, and modules absent until a shared environment module or explicit root resource graph is added.
- `modules/api-http` owns HTTP API v2 routing to the API Lambda.
- `modules/compute-lambda` owns Lambda execution roles, basic logging, inline least-privilege policy attachment, runtime settings, artifact references, and environment variables. It exposes planned environment variables as module output metadata for root-level Terraform plan tests; root outputs still expose only environment-appropriate deployment metadata.
- `modules/data-dynamodb` owns flight cache, lookup/idempotency, position history, and usage budget tables.
- `modules/eventing` owns the encrypted fetch task queue, encrypted DLQ, fetcher SQS event source mapping, dispatcher schedule, EventBridge invoke permission, and queue-consume permissions.
- `modules/storage-s3` owns route and track GeoJSON artifact storage.
- `modules/secrets` owns Secrets Manager secret metadata only. Secret values are created or rotated outside Terraform state.
- `modules/observability` owns dashboards and CloudWatch alarms for Lambda, SQS, FlightAware, and budget signals.

Each reusable module declares its own Terraform and AWS provider requirements and carries native Terraform tests in `tests/*.tftest.hcl`. These tests use Terraform mock providers so module contracts are checked through Terraform-interpreted plan/apply values without AWS credentials or live AWS resources. Input validation belongs at the nearest owner: reusable modules reject invalid reusable inputs such as Lambda memory and timeout ranges, S3 retention periods, and fetch-task DLQ thresholds; environment roots reject invalid environment-scoped values before they reach composed modules.

The staging and production roots are intentionally non-deployable placeholders. Their tests are contract guards: they prove the root owns exactly one environment name, retain backend wiring as a bootstrap placeholder, expose only the `environment` output, and fail if deployable Terraform blocks are added without updating the environment design.

The dev root includes a sandbox apply/destroy integration test for manual, nightly, or release-before-deploy validation. It is explicitly opt-in because it creates live AWS resources. The test uses temporary local Terraform state, applies the dev root in a sandbox AWS account, verifies only safe output metadata for Lambda, HTTP API, SQS, DynamoDB, S3, CloudWatch, and Secrets Manager references, checks Terraform state for the expected live resource types, and then destroys the same state. It does not introduce a service-side contract change: runtime code still receives the existing environment variables and secret references, and raw secret values remain outside Terraform.

Terraform security policy scanning is split between Checkov and repository-specific Vitest contract tests. Checkov runs from `.checkov.yml` through `make terraform-policy` and CI, enforcing the selected baseline for S3 public-access controls, S3 encryption, SQS encryption, and public access block attachment. `infra/terraform/security-policy/` owns the repository-specific scanner contract tests that keep scanner wiring in the Terraform check path, prevent Terraform-managed Secrets Manager secret values, and limit wildcard IAM resources to the bootstrap key administration policy and the read-only plan discovery policy.

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
