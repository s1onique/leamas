#!/usr/bin/env bash
# Tiny wrapper: delegates to leamas factory digest
set -euo pipefail

# Try to find leamas binary in common locations
LEAMAS=""
if [[ -x ./bin/leamas ]]; then
  LEAMAS="./bin/leamas"
elif command -v leamas &>/dev/null; then
  LEAMAS="leamas"
fi

# If we have leamas, use it; otherwise use go run
if [[ -n "$LEAMAS" ]]; then
  exec "$LEAMAS" factory digest "$@"
fi

# Fall back to go run for development
exec go run ./cmd/leamas factory digest "$@"
