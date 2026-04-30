#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
terraform_bin="${TERRAFORM:-terraform}"
status=0
pids=()
terraform_dirs=()

add_terraform_dir_if_config() {
  local dir="$1"

  if compgen -G "${repo_root}/${dir}/*.tf" >/dev/null; then
    terraform_dirs+=("${dir}")
  fi
}

add_terraform_dir_if_config "infra/terraform/bootstrap"

for dir in "${repo_root}"/infra/terraform/envs/*; do
  [[ -d "${dir}" ]] || continue
  add_terraform_dir_if_config "${dir#"${repo_root}/"}"
done

if [[ "${#terraform_dirs[@]}" -eq 0 ]]; then
  echo "No Terraform root directories found" >&2
  exit 1
fi

init_terraform() {
  local dir="$1"
  local init_output

  if init_output="$("${terraform_bin}" -chdir="${dir}" init -backend=false -input=false -lockfile=readonly 2>&1)"; then
    return 0
  fi

  printf '%s\n' "${init_output}" >&2
  echo "Retrying terraform init after removing ${dir}/.terraform" >&2
  rm -rf "${dir}/.terraform"
  "${terraform_bin}" -chdir="${dir}" init -backend=false -input=false -lockfile=readonly >/dev/null
}

validate_terraform() {
  local dir="$1"

  init_terraform "${dir}"
  if "${terraform_bin}" -chdir="${dir}" validate; then
    return 0
  fi

  echo "Retrying terraform validate after removing ${dir}/.terraform" >&2
  rm -rf "${dir}/.terraform"
  init_terraform "${dir}"
  "${terraform_bin}" -chdir="${dir}" validate
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
