# Terraform

AWS resources are managed with Terraform. Lambda and Next.js build artifacts are produced outside Terraform and are not managed as Terraform source.

## Layout

- `bootstrap`: Initial state bucket, KMS, OIDC, and Terraform execution role setup
- `envs/dev`: dev environment
- `envs/stg`: stg environment
- `envs/prod`: prod environment
- `modules`: Reusable modules

## Naming And Secrets

Terraform resource names follow the root naming policy: `airpath-${environment}` for the resource prefix and `airpath-${environment}-${table-suffix}` for DynamoDB tables. Environment directories are scoped to exactly one environment and validate their own `environment` variable.

Terraform state must contain resource metadata and secret references only. Store runtime secret values in AWS Secrets Manager and pass secret names or ARNs to services. Do not put API keys, JWT material, WebSocket tokens, FlightAware credentials, or Mapbox secret tokens in Terraform variables, outputs, local values, plan files, or committed tfvars examples.

## Local Checks

```sh
make terraform-fmt
make terraform-lint
make terraform-validate
make terraform-check
```

`terraform fmt` is the canonical Terraform formatter. TFLint is used for Terraform linting. Local runs use `tflint` from `PATH` or download the pinned version into `.cache/tflint`.
