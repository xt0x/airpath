#!/usr/bin/env bash
set -euo pipefail

tflint_bin="${TFLINT:-tflint}"
tflint_config="${TFLINT_CONFIG:-$(pwd)/.tflint.hcl}"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

terraform_dirs=(
  "infra/terraform/bootstrap"
  "infra/terraform/envs/dev"
  "infra/terraform/envs/stg"
  "infra/terraform/envs/prod"
)

if ! command -v "${tflint_bin}" >/dev/null; then
  tflint_bin="$(bash "${repo_root}/scripts/ci/ensure-tflint.sh")"
fi

"${tflint_bin}" --init --config="${tflint_config}"

for dir in "${terraform_dirs[@]}"; do
  echo "==> tflint ${dir}"
  "${tflint_bin}" --config="${tflint_config}" --chdir="${dir}"
done
