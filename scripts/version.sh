#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
# shellcheck source=scripts/lib/product.sh
source scripts/lib/product.sh

check=0
if [ "${1:-}" = "--check" ]; then
  check=1
elif [ "$#" -gt 0 ]; then
  echo "usage: $0 [--check]" >&2
  exit 2
fi

current_version="$(vps_agent_version)"
printf '%s %s\n' "$VPS_AGENT_PRODUCT_NAME" "$current_version"

[ "$check" -eq 1 ] || exit 0

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "Update status: unavailable outside a Git checkout."
  exit 0
fi

if git remote get-url origin >/dev/null 2>&1; then
  if ! git fetch --tags --quiet origin; then
    echo "Update status: unable to refresh release tags; local tag information only." >&2
  fi
fi

latest="$(git tag -l 'v[0-9]*' --sort=-v:refname | grep -Ev -- '-' | head -n 1 || true)"
if [ -z "$latest" ]; then
  echo "Update status: no stable release has been published yet."
  exit 0
fi

latest_sha="$(git rev-list -n 1 "$latest")"
head_sha="$(git rev-parse HEAD)"
if [ "$head_sha" = "$latest_sha" ]; then
  echo "Update status: up to date on stable $latest."
  exit 0
fi

if git merge-base --is-ancestor "$head_sha" "$latest_sha" 2>/dev/null; then
  echo "Update available: $latest"
  echo "Run: bash scripts/update.sh"
  exit 0
fi

if git merge-base --is-ancestor "$latest_sha" "$head_sha" 2>/dev/null; then
  echo "Update status: current checkout is ahead of stable $latest."
  exit 0
fi

echo "Update status: current checkout diverges from stable $latest."
