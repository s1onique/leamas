#!/usr/bin/env bash
set -euo pipefail

if [[ -x ./bin/leamas ]]; then
  exec ./bin/leamas factory gate "$@"
fi

exec go run ./cmd/leamas factory gate "$@"
