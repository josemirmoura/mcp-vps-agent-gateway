#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

scope="$tmp/scope"
mkdir -p "$scope"
cat >"$tmp/.env" <<EOF
VPS_AGENT_SCOPE_ROOT=$scope
VPS_AGENT_WHOLE_HOST=0
EOF

cat >"$tmp/policy.yaml" <<'EOF'
version: 1
mode: scoped
filesystem:
  read: []
  write: []
  actions: [list, stat, read, mkdir, write]
shell:
  enabled: true
  cwd_roots: []
  max_runtime_seconds: 60
  max_output_bytes: 1048576
services:
  inspect: []
  manage: []
  actions: []
docker:
  inspect: []
  manage: []
  actions: []
compose:
  inspect: []
  manage: []
  actions: [inspect, validate, up, down]
network:
  mode: blocked
  destinations: []
privilege:
  admin: broker-only
replay:
  require_idempotency_for_safe_writes: true
  blind_retry_non_replay_safe: false
EOF

root="$scope/project-a"

python3 scripts/delegate-root.py add "$root" \
  --env "$tmp/.env" --policy "$tmp/policy.yaml" --no-backup

[ "$(grep -Fxc "    - $root" "$tmp/policy.yaml")" -eq 3 ]

python3 scripts/delegate-root.py add "$root" --access compose \
  --env "$tmp/.env" --policy "$tmp/policy.yaml" --no-backup

[ "$(grep -Fxc "    - $root" "$tmp/policy.yaml")" -eq 5 ]

list_output="$(python3 scripts/delegate-root.py list --env "$tmp/.env" --policy "$tmp/policy.yaml")"
grep -F "$root: read, write, shell, compose" <<<"$list_output" >/dev/null

if python3 scripts/delegate-root.py add "$tmp/outside" \
  --env "$tmp/.env" --policy "$tmp/policy.yaml" --no-backup >/dev/null 2>&1; then
  echo "outside root was unexpectedly accepted" >&2
  exit 1
fi

if python3 scripts/delegate-root.py add "$scope" \
  --env "$tmp/.env" --policy "$tmp/policy.yaml" --no-backup >/dev/null 2>&1; then
  echo "physical ceiling was unexpectedly delegated without --allow-ceiling" >&2
  exit 1
fi

python3 scripts/delegate-root.py remove "$root" \
  --env "$tmp/.env" --policy "$tmp/policy.yaml" --no-backup

if grep -F "$root" "$tmp/policy.yaml" >/dev/null; then
  echo "delegated root remained after revoke" >&2
  exit 1
fi

echo "delegated root policy tests: PASS"
