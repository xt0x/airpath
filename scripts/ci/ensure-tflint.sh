#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
tflint_version="$(<"${repo_root}/.tflint-version")"
cache_dir="${TFLINT_CACHE_DIR:-${repo_root}/.cache/tflint}"

case "$(uname -s)" in
  Darwin)
    os="darwin"
    ;;
  Linux)
    os="linux"
    ;;
  *)
    echo "Unsupported OS for automatic TFLint install: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  arm64 | aarch64)
    arch="arm64"
    ;;
  x86_64 | amd64)
    arch="amd64"
    ;;
  *)
    echo "Unsupported architecture for automatic TFLint install: $(uname -m)" >&2
    exit 1
    ;;
esac

install_dir="${cache_dir}/${tflint_version}/${os}_${arch}"
tflint_bin="${install_dir}/tflint"

if [[ -x "${tflint_bin}" ]]; then
  printf '%s\n' "${tflint_bin}"
  exit 0
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

asset_name="tflint_${os}_${arch}.zip"
download_url="https://github.com/terraform-linters/tflint/releases/download/${tflint_version}/${asset_name}"

mkdir -p "${install_dir}"
echo "Installing TFLint ${tflint_version} for ${os}_${arch}..." >&2
curl -fsSL "${download_url}" -o "${tmp_dir}/${asset_name}"
unzip -q "${tmp_dir}/${asset_name}" -d "${install_dir}"
chmod +x "${tflint_bin}"

printf '%s\n' "${tflint_bin}"
