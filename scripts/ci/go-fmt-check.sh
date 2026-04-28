#!/usr/bin/env bash
set -euo pipefail

unformatted_files="$(env -u GOROOT gofmt -l services)"

if [[ -n "${unformatted_files}" ]]; then
  printf '%s\n' "${unformatted_files}"
  exit 1
fi
