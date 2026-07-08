#!/usr/bin/env bash
set -euo pipefail

if [[ -x ./bin/leamas ]]; then
  exec ./bin/leamas factory verify static-binary "$@"
fi

exec go run ./cmd/leamas factory verify static-binary "$@"
