#!/usr/bin/env bash
set -euo pipefail

terraform_dirs=(
  "infra/terraform/bootstrap"
  "infra/terraform/envs/dev"
  "infra/terraform/envs/stg"
  "infra/terraform/envs/prod"
)

status=0
pids=()

init_terraform() {
  local dir="$1"
  local init_output

  if init_output="$(terraform -chdir="${dir}" init -backend=false -input=false -lockfile=readonly 2>&1)"; then
    return 0
  fi

  printf '%s\n' "${init_output}" >&2
  echo "Retrying terraform init after removing ${dir}/.terraform" >&2
  rm -rf "${dir}/.terraform"
  terraform -chdir="${dir}" init -backend=false -input=false -lockfile=readonly >/dev/null
}

validate_terraform() {
  local dir="$1"

  init_terraform "${dir}"
  if terraform -chdir="${dir}" validate; then
    return 0
  fi

  echo "Retrying terraform validate after removing ${dir}/.terraform" >&2
  rm -rf "${dir}/.terraform"
  init_terraform "${dir}"
  terraform -chdir="${dir}" validate
}

for dir in "${terraform_dirs[@]}"; do
  (
    echo "==> terraform validate ${dir}"
    validate_terraform "${dir}"
  ) &
  pids+=("$!")
done

for pid in "${pids[@]}"; do
  if ! wait "${pid}"; then
    status=1
  fi
done

exit "${status}"
