#!/usr/bin/env bash
#
# Verify agent context files exist and contain required content.
# Wrapper for the Go-based agent context verifier.

set -euo pipefail

if [[ -x ./bin/leamas ]]; then
  exec ./bin/leamas factory verify agent-context "$@"
fi

exec go run ./cmd/leamas factory verify agent-context "$@"
