#!/usr/bin/env bash
set -euo pipefail

if [[ -x ./bin/leamas ]]; then
  exec ./bin/leamas factory verify doctrine-agent-contracts "$@"
fi

exec go run ./cmd/leamas factory verify doctrine-agent-contracts "$@"
