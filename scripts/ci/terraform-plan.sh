#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
selected_env="${1:-all}"
plan_out_dir="${PLAN_OUT_DIR:-}"

env_dir() {
  case "$1" in
    dev | stg | prod)
      printf '%s\n' "infra/terraform/envs/$1"
      ;;
    *)
      echo "Unsupported Terraform environment: $1" >&2
      exit 1
      ;;
  esac
}

if [[ "${selected_env}" == "all" ]]; then
  terraform_envs=(dev stg prod)
else
  terraform_envs=("${selected_env}")
fi

if [[ -n "${plan_out_dir}" ]]; then
  mkdir -p "${plan_out_dir}"
fi

for terraform_env in "${terraform_envs[@]}"; do
  dir="$(env_dir "${terraform_env}")"
  echo "==> terraform plan ${dir}"
  terraform -chdir="${repo_root}/${dir}" init -input=false -lockfile=readonly >/dev/null

  plan_args=(-input=false -lock-timeout=5m)
  if [[ -n "${plan_out_dir}" ]]; then
    plan_args+=("-out=${plan_out_dir}/${terraform_env}.tfplan")
  fi

  terraform -chdir="${repo_root}/${dir}" plan "${plan_args[@]}"
done
