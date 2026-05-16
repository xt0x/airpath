# Terraform

AWS resources are managed with Terraform. Lambda and Next.js build artifacts are produced outside Terraform and are not managed as Terraform source.

## Layout

- `bootstrap`: Initial state bucket, KMS, OIDC, and Terraform execution role setup
- `envs/dev`: implemented personal demo environment
- `envs/stg`: staging root with backend, provider, variable, and output configuration only
- `envs/prod`: production root with backend, provider, variable, and output configuration only
- `modules`: Reusable modules with native `terraform test` coverage under each module's `tests/` directory

Only the dev environment currently declares resources. The stg and prod roots keep only the files required to validate environment naming, partial S3 backend wiring, variables, and CI paths without implying deployable staging or production infrastructure.

Reusable modules declare their own Terraform and AWS provider requirements so they can be initialized and tested directly. Module-level tests use Terraform's mock provider to inspect interpreted plan/apply values without requiring AWS credentials or creating AWS resources.

The stg and prod roots also carry native `terraform test` coverage under `tests/` to prove their default environment output and reject cross-environment variable values.

The dev root carries native plan-based tests. IAM policy semantic tests decode Terraform-rendered policy JSON to guard service boundaries without changing the services contract: API must not read secret values, only the fetcher can read the FlightAware secret value, dispatcher has no S3 or Secrets Manager permissions, and SQS/DynamoDB permissions remain scoped to module outputs. Dev plan-value tests also assert the default and opt-in FlightAware runtime flags, secret-reference-only outputs, and the `airpath-dev-*` naming policy for planned Lambda, data, storage, and eventing outputs.

## Naming And Secrets

Terraform resource names follow the root naming policy: `airpath-${environment}` for the resource prefix and `airpath-${environment}-${table-suffix}` for DynamoDB tables. Environment directories are scoped to exactly one environment and validate their own `environment` variable.

The bootstrap root uses `bootstrap_environment` for resource names and tags. The default is `dev`, but the variable is validated against the full environment set: `local`, `dev`, `stg`, and `prod`. The generated backend keys keep the same environment-scoped form for every environment:

- `local/terraform.tfstate`
- `dev/terraform.tfstate`
- `stg/terraform.tfstate`
- `prod/terraform.tfstate`

Terraform state must contain resource metadata and secret references only. Store runtime secret values in AWS Secrets Manager and pass secret names or ARNs to services. Do not put API keys, JWT material, WebSocket tokens, FlightAware credentials, or Mapbox secret tokens in Terraform variables, outputs, local values, plan files, or committed tfvars examples.

## Bootstrap Backend

`infra/terraform/bootstrap` creates the shared Terraform backend primitives:

- S3 state bucket with versioning, public access blocking, and KMS default encryption
- S3 native lockfile access for `*.tflock`
- KMS key with rotation enabled
- GitHub Actions OIDC provider
- CI plan role scoped to explicit GitHub OIDC subject claims, read-only Terraform state object access, read/write native lockfile access, S3 bucket and configuration metadata read access without object-key enumeration, and read-only AWS APIs needed by Terraform plan refresh

Apply bootstrap from a trusted administrator session before configuring remote state for environment roots. Environment roots declare an empty S3 backend block, so after bootstrap apply use `terraform output backend_config` to populate `terraform init -backend-config` values for the selected environment key. CI should assume `github_ci_plan_role_arn` through GitHub OIDC instead of storing AWS access keys. The default OIDC subject allows only the configured repository's `main` branch; set `github_oidc_subjects` explicitly when CI must run from a pull request subject or a GitHub environment subject.

## Local Checks

```sh
make terraform-fmt
make terraform-lint
make terraform-policy
make terraform-validate
make terraform-test
make terraform-check
```

`terraform fmt` is the canonical Terraform formatter. TFLint is used for Terraform linting across bootstrap, environment roots, and reusable modules. Local runs use `tflint` from `PATH` or download the pinned version into `.cache/tflint`. Checkov is used for selected security policy checks through `make terraform-policy`; the script uses `checkov` from `PATH` or the pinned `.checkov-version` through `uvx`. `make terraform-test` delegates to `scripts/ci/terraform-test.sh`, which initializes the dev root, the stg/prod placeholder roots, and each reusable module with `-backend=false`, then runs their native `.tftest.hcl` files. CI runs `make terraform-check`, so the same fmt, validate, native Terraform tests, lint, and policy checks are enforced for pull requests. Generated module-level `.terraform.lock.hcl` files are ignored; environment and bootstrap lock files remain committed.

The Checkov baseline is intentionally narrow and enforced by `.checkov.yml`: S3 public access controls, S3 encryption, SQS encryption, and S3 public access block attachment.

## Dev Personal Demo

The `envs/dev` root deploys the personal demo environment. It is personal, non-commercial, and low-frequency by policy. Lambda artifacts are built outside Terraform with `make lambda-artifacts`, then referenced by the dev artifact path variables. The default artifact paths resolve from `infra/terraform/envs/dev` to the repository-level `artifacts/dev/*.zip` files produced by that build target.

Real FlightAware calls are disabled by default. Keep `allow_real_flightaware_calls = false` for normal dev deployments. Set it to `true` only for a limited opt-in smoke test after a backend-only FlightAware API key has been stored in Secrets Manager. Terraform still stores only the secret reference, never the raw API key.

The dev root owns the deployed runtime contract for the Go services. Lambda creation is ordered after its managed log group and execution-role policy attachments so initial applies do not expose functions before their Terraform-managed IAM permissions are attached.

- API Lambda: HTTP API integration, DynamoDB cache and usage tables, S3 GeoJSON artifacts, fetch task enqueue permission, and the FlightAware secret ARN environment reference.
- Fetcher Lambda: SQS event source mapping with `ReportBatchItemFailures`, DynamoDB lease/cache/position/usage writes, S3 route and track artifact writes, Secrets Manager read access for the FlightAware API key, and diagnostic SQS send access. The event source mapping is ordered after queue-consume IAM policy attachment because AWS validates those permissions when the mapping is created.
- Dispatcher Lambda: EventBridge schedule, due-poll DynamoDB reads and updates, fetch-task idempotency reservations, fetch task SQS send access, and the runtime queue/table environment variables required by `services/internal/runtimewiring`.

Fetch task SQS queues use SQS-managed server-side encryption. The fetch task queue visibility timeout is owned by the eventing module input and the dev root derives it from the fetcher Lambda timeout plus a buffer. This changes AWS queue storage and retry timing only; Lambda event source mapping, queue URLs, and service environment variable names stay unchanged.

CloudWatch observability uses the dev root's concrete AWS region for dashboard widgets and receives each Lambda's configured timeout so timeout alarms are derived per function instead of using a shared threshold. The API and dispatcher use 10 second timeout windows; the fetcher uses a 30 second timeout window. Lambda and HTTP API log groups are Terraform-managed with bounded retention. The HTTP API stage emits JSON access logs and applies low-frequency default throttling. Lambda alarm names avoid duplicating the environment prefix when function names already include it.

The dispatcher uses `DISPATCHER_ASSUME_ACTIVE_VIEWER=true` in dev so the personal demo can enqueue due polling tasks without a separate viewer activity signal. Staging and production should revisit that input when they introduce a real activity source.

The dev root also validates environment-specific deployment inputs before planning resources. It rejects non-dev environment names, non-positive GeoJSON artifact retention, and non-positive fetch-task DLQ receive thresholds. Its native Terraform tests decode rendered IAM policy JSON and inspect planned module output values to keep API, fetcher, and dispatcher permissions, FlightAware opt-in flags, secret references, and naming aligned with their runtime responsibilities. The `compute-lambda` module exposes planned environment variables as module metadata so root tests can inspect the interpreted plan without adding root outputs or changing the services contract. Module tests cover reusable contracts where the input belongs to a module.
