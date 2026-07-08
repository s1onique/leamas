#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

chmod +x githooks/pre-push
git config --local core.hooksPath githooks

echo "Installed Leamas Git hooks:"
echo "  core.hooksPath=$(git config --get core.hooksPath)"
