#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
terraform_env="${1:?Terraform environment is required: dev, stg, or prod}"
plan_file="${2:?Terraform plan file is required}"

case "${terraform_env}" in
  dev | stg | prod)
    dir="infra/terraform/envs/${terraform_env}"
    ;;
  *)
    echo "Unsupported Terraform environment: ${terraform_env}" >&2
    exit 1
    ;;
esac

terraform -chdir="${repo_root}/${dir}" init -input=false -lockfile=readonly >/dev/null
terraform -chdir="${repo_root}/${dir}" apply -input=false "${plan_file}"
