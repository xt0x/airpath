#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
artifact_dir="${repo_root}/artifacts/dev"
bootstrap_path="${artifact_dir}/bootstrap"
lambda_targets=(api fetcher dispatcher)

cleanup() {
  rm -f "${bootstrap_path}"
}

trap cleanup EXIT

mkdir -p "${artifact_dir}"

for name in "${lambda_targets[@]}"; do
  cleanup
  (
    cd "${repo_root}/services"
    env -u GOROOT GOOS=linux GOARCH=amd64 go build -o "${bootstrap_path}" "./cmd/${name}"
  )
  (
    cd "${artifact_dir}"
    zip -q -j "${name}-lambda.zip" bootstrap
  )
done
