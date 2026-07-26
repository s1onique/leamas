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
# Works with both attached HEAD (local development) and detached HEAD
# (GitHub pull_request workflow checkouts).
#
# Exit codes: 0 on pass, non-zero on any failure.

set -euo pipefail

source_root="$(git rev-parse --show-toplevel)"
source_head="$(git -C "$source_root" rev-parse 'HEAD^{commit}')"

tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

remote="$tmp_root/remote.git"
qualification_branch="leamas-gate-fast-subject"
qualification_ref="refs/heads/$qualification_branch"

git -C "$source_root" clone --bare --quiet \
  "$source_root" "$remote"

git --git-dir="$remote" cat-file -e \
  "$source_head^{commit}"

git --git-dir="$remote" update-ref \
  "$qualification_ref" "$source_head"

git clone \
  --depth=1 \
  --branch "$qualification_branch" \
  --quiet \
  "file://$remote" \
  "$tmp_root/shallow"

cloned_head="$(
  git -C "$tmp_root/shallow" rev-parse 'HEAD^{commit}'
)"
if [ "$cloned_head" != "$source_head" ]; then
  printf 'shallow clone HEAD mismatch: expected %s, got %s\n' \
    "$source_head" "$cloned_head" >&2
  exit 1
fi

is_shallow="$(
  git -C "$tmp_root/shallow" \
    rev-parse --is-shallow-repository
)"
if [ "$is_shallow" != "true" ]; then
  printf 'shallow clone did not produce a shallow repository: %s\n' \
    "$is_shallow" >&2
  exit 1
fi

cd "$tmp_root/shallow"
# Bootstrap git hooks so the gate-fast hook gate does not refuse
# the fresh clone. The hooks path is local to the clone and does
# not touch the source repository.
make bootstrap
CGO_ENABLED=0 make gate-fast
