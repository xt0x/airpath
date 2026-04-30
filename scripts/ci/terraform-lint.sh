#!/usr/bin/env bash
set -euo pipefail

tflint_bin="${TFLINT:-tflint}"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
tflint_config="${TFLINT_CONFIG:-${repo_root}/.tflint.hcl}"
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

for dir in "${repo_root}"/infra/terraform/modules/*; do
  [[ -d "${dir}" ]] || continue
  add_terraform_dir_if_config "${dir#"${repo_root}/"}"
done

if [[ "${#terraform_dirs[@]}" -eq 0 ]]; then
  echo "No Terraform root directories found" >&2
  exit 1
fi

if ! command -v "${tflint_bin}" >/dev/null; then
  tflint_bin="$(bash "${repo_root}/scripts/ci/ensure-tflint.sh")"
fi

"${tflint_bin}" --init --config="${tflint_config}"

for dir in "${terraform_dirs[@]}"; do
  echo "==> tflint ${dir}"
  "${tflint_bin}" --config="${tflint_config}" --chdir="${dir}"
done
