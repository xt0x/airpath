# Airpath

Airpath is a 2D flight route display system built around FlightAware AeroAPI.

## Repository Layout

- `frontend`: Next.js / TypeScript web app
- `services`: Go Lambda services
- `packages/shared-types`: TypeScript API contracts, shared UI types, and fixtures
- `packages/geo`: GeoJSON and flight track processing
- `packages/map-rendering`: Shared Mapbox / deck.gl rendering logic
- `infra/terraform`: Terraform-managed AWS resources
- `docs`: ignored local specifications, plans, ADRs, and operations notes

The root `package.json`, `pnpm-lock.yaml`, `pnpm-workspace.yaml`, `.prettierrc.json`, and `tsconfig.base.json` configure the pnpm workspace and shared development commands. They are not application source. Next.js-specific configuration, dependencies, and environment examples live under `frontend`. App Router source files live under `frontend/src/app`. Files under `docs/` are ignored local planning artifacts unless the ignore policy is explicitly changed.

## Commands

```sh
pnpm install
make lint
make test
make build
make lambda-artifacts
make terraform-check
make terraform-validate
make ci
```

Terraform is expected to run with `1.10.5`. CI installs the pinned version through `hashicorp/setup-terraform`.
Terraform linting uses TFLint. CI installs it through `terraform-linters/setup-tflint`; local runs use `tflint` from `PATH` or download the pinned version into `.cache/tflint`.

## CI

Pull requests and pushes to `main` run static checks only. `.github/workflows/ci.yml` is the shared entrypoint, and it calls reusable workflow files split by responsibility:

- `.github/workflows/ci-ts.yml`: pnpm install, lint, typecheck, tests, workspace build, and Next.js build
- `.github/workflows/ci-go.yml`: Go formatting, vet, and tests
- `.github/workflows/ci-terraform.yml`: Terraform fmt, validate with `-backend=false`, and TFLint
- `.github/workflows/ci-workflows.yml`: GitHub Actions workflow lint

CI does not authenticate to AWS, run Terraform plan, run Terraform apply, or deploy application artifacts.

## Environment And Naming Policy

Airpath uses these environment names only:

- `local`: developer machines and local-only services. Local work must not depend on shared AWS resources by default.
- `dev`: shared development AWS environment for integration work.
- `stg`: staging AWS environment for production-like verification.
- `prod`: production AWS environment.

AWS resource names use the prefix `airpath-${environment}`. For example, dev resources start with `airpath-dev`, and prod resources start with `airpath-prod`. Resource-specific suffixes use lowercase kebab case, such as `airpath-dev-api` or `airpath-prod-fetcher-dlq`.

Terraform-managed AWS resources must include these tags when the provider supports tags:

- `Project = airpath`
- `Environment = local | dev | stg | prod`
- `ManagedBy = terraform`

DynamoDB table names use `airpath-${environment}-${table-suffix}`. Table suffixes are lowercase kebab case versions of the logical table names from the specification:

- `flights`
- `flight-lookup`
- `flight-positions`
- `flight-events`
- `user-watches`
- `user-watch-limits`
- `websocket-connections`
- `flight-subscriptions`
- `fetch-tasks`

Terraform state keys are environment-scoped. Use `${environment}/terraform.tfstate` and `${environment}/terraform.tfstate.tflock` unless a later bootstrap design explicitly narrows the path further.

## Secret Handling Policy

API keys, JWT signing material, WebSocket tokens, FlightAware credentials, and Mapbox secret tokens must be read from environment variables or from AWS Secrets Manager at runtime. They must not be hardcoded in source files, Terraform files, fixtures, tests, documentation examples, or committed `.env` files.

Only browser-safe values may use `NEXT_PUBLIC_` variables. Secret Mapbox tokens, FlightAware API keys, JWT material, and WebSocket signing or bearer tokens must never be exposed through `NEXT_PUBLIC_` variables or bundled into frontend code.

Terraform must store only secret references, such as Secrets Manager ARNs or parameter names, not raw secret values. Do not place secrets in Terraform variables, resource arguments, outputs, local values, plan files, or state. When a provider requires a sensitive value, stop and document the exception before implementing it.

Application logs, CI logs, Terraform logs, and error responses must redact sensitive request data. Do not log `Authorization`, `Cookie`, API key headers, JWTs, WebSocket tokens, FlightAware credentials, Mapbox secret tokens, or raw request bodies that may contain those values.

Local `.env` files, Terraform variable files, private keys, certificates, state files, and plan files are ignored by Git. Commit only `.env.example` files with non-secret defaults and placeholder names.

## Personal Demo Environment

The `dev` deployment is a personal, non-commercial, low-frequency demo environment. It is intended for cache-first development and limited verification, not commercial tracking or high-volume polling.

Build deployable Go Lambda artifacts before applying the dev Terraform root:

```sh
make lambda-artifacts
cd infra/terraform/envs/dev
terraform init
terraform apply
```

Real FlightAware calls are disabled by default. A dev deployment must set both `FLIGHTAWARE_FETCH_ENABLED=true` and `FLIGHTAWARE_REAL_CALLS_ENABLED=true` before any real FlightAware call is allowed. The Terraform variable `allow_real_flightaware_calls` sets both flags for a limited opt-in deployment and defaults to `false`.

F17 acceptance should cover fixture tests, local integration tests, and limited real-call smoke tests:

```sh
pnpm test
cd services && env -u GOROOT go test ./...
cd services && AIRPATH_FLIGHTAWARE_REAL_TESTS=true FLIGHTAWARE_API_KEY="$FLIGHTAWARE_API_KEY" FLIGHTAWARE_TEST_IDENT=ANA110 env -u GOROOT go test ./internal/awsintegration -run OptIn
```

Run the limited real-call smoke tests only from a personal account, with a backend-only `FLIGHTAWARE_API_KEY` environment variable, and stop after the smoke run completes.
