#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
checkov_bin="${CHECKOV:-checkov}"

if command -v "${checkov_bin}" >/dev/null; then
  "${checkov_bin}" --config-file "${repo_root}/.checkov.yml"
  exit 0
fi

if command -v uvx >/dev/null; then
  checkov_version="$(<"${repo_root}/.checkov-version")"
  uvx --from "checkov==${checkov_version}" checkov --config-file "${repo_root}/.checkov.yml"
  exit 0
fi

echo "Checkov is required for Terraform policy checks. Install checkov or uvx." >&2
exit 1
