# Terraform

AWS resources are managed with Terraform. Lambda and Next.js build artifacts are produced outside Terraform and are not managed as Terraform source.

## Layout

- `bootstrap`: Initial state bucket, KMS, OIDC, and Terraform execution role setup
- `envs/dev`: implemented personal demo environment
- `envs/stg`: staging root with backend, provider, variable, and output configuration only
- `envs/prod`: production root with backend, provider, variable, and output configuration only
- `modules`: Reusable modules

Only the dev environment currently declares resources. The stg and prod roots keep only the files required to validate environment naming, backend keys, variables, and CI paths without implying deployable staging or production infrastructure.

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
- CI plan role scoped to one `owner/name` repository, the Terraform state objects, and read-only AWS APIs needed by Terraform plan refresh

Apply bootstrap from a trusted administrator session before configuring remote state for environment roots. After apply, use `terraform output backend_config` to populate `terraform init -backend-config` values. CI should assume `github_ci_plan_role_arn` through GitHub OIDC instead of storing AWS access keys.

## Local Checks

```sh
make terraform-fmt
make terraform-lint
make terraform-validate
make terraform-check
```

`terraform fmt` is the canonical Terraform formatter. TFLint is used for Terraform linting. Local runs use `tflint` from `PATH` or download the pinned version into `.cache/tflint`.

## Dev Personal Demo

The `envs/dev` root deploys the personal demo environment. It is personal, non-commercial, and low-frequency by policy. Lambda artifacts are built outside Terraform with `make lambda-artifacts`, then referenced by the dev artifact path variables.

Real FlightAware calls are disabled by default. Keep `allow_real_flightaware_calls = false` for normal dev deployments. Set it to `true` only for a limited opt-in smoke test after a backend-only FlightAware API key has been stored in Secrets Manager. Terraform still stores only the secret reference, never the raw API key.
