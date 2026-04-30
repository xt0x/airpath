#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
terraform_bin="${TERRAFORM:-terraform}"

terraform_test_dirs=(
  "infra/terraform/envs/dev"
  "infra/terraform/envs/stg"
  "infra/terraform/envs/prod"
  "infra/terraform/modules/api-http"
  "infra/terraform/modules/compute-lambda"
  "infra/terraform/modules/data-dynamodb"
  "infra/terraform/modules/eventing"
  "infra/terraform/modules/observability"
  "infra/terraform/modules/secrets"
  "infra/terraform/modules/storage-s3"
)

for dir in "${terraform_test_dirs[@]}"; do
  if [[ ! -d "${repo_root}/${dir}" ]]; then
    echo "Missing Terraform test directory: ${dir}" >&2
    exit 1
  fi

  echo "==> terraform test ${dir}"
  "${terraform_bin}" -chdir="${repo_root}/${dir}" init -backend=false -input=false
  "${terraform_bin}" -chdir="${repo_root}/${dir}" test
done
