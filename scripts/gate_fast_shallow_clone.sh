#!/usr/bin/env bash
# gate_fast_shallow_clone.sh - Qualify gate-fast under depth-one clone.
#
# Set up a bare remote from the current repository and a temporary
# depth-one clone via a file:// URL, then run `make gate-fast`
# inside the clone. The script intentionally avoids:
#   - contact with GitHub,
#   - fetching missing historical objects,
#   - copying the source object directory,
#   - alternates or unshallow operations,
#   - injecting unreachable objects,
#   - branch-conditional skips.
#
# Exit codes: 0 on pass, non-zero on any failure.

set -euo pipefail

source_root="$(git rev-parse --show-toplevel)"
branch="$(git -C "$source_root" branch --show-current)"
tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

git -C "$source_root" clone --bare --quiet "$source_root" "$tmp_root/remote.git"
git clone --depth=1 --branch "$branch" --quiet \
  "file://$tmp_root/remote.git" "$tmp_root/shallow"

is_shallow="$(git -C "$tmp_root/shallow" rev-parse --is-shallow-repository)"
if [ "$is_shallow" != "true" ]; then
  echo "shallow clone did not produce a shallow repository: $is_shallow" >&2
  exit 1
fi

cd "$tmp_root/shallow"
# Bootstrap git hooks so the gate-fast hook gate does not refuse
# the fresh clone. The hooks path is local to the clone and does
# not touch the source repository.
make bootstrap
CGO_ENABLED=0 make gate-fast
