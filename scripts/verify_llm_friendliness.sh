#!/usr/bin/env bash
# verify_llm_friendliness.sh - Bash wrapper for Go LLM-friendliness verifier
# This is tiny glue only, per tooling-boundary doctrine.

set -euo pipefail

# Check for pre-built binary first
if [[ -x ./bin/leamas ]]; then
  exec ./bin/leamas factory verify llm-friendly "$@"
fi

# Fall back to go run
exec go run ./cmd/leamas factory verify llm-friendly "$@"
