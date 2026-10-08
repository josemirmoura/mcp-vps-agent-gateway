#!/usr/bin/env bash
# Non-mutating preflight for operator portal rollout on existing installations.
set -euo pipefail
cd "$(dirname "$0")/.."
printf 'Portico operator portal | preflight only\n'
printf 'Repository: %s\n' "$PWD"
test -f compose.yaml || { echo 'FAIL: missing compose.yaml'; exit 1; }
test -f web/operator-approval/server.py || { echo 'FAIL: missing portal backend'; exit 1; }
test -f web/operator-approval/index.html || { echo 'FAIL: missing portal UI'; exit 1; }
test -f scripts/operator-approvals.py || { echo 'FAIL: missing operator fallback'; exit 1; }
command -v docker >/dev/null || { echo 'FAIL: Docker missing'; exit 1; }
docker compose config --quiet || { echo 'FAIL: existing Compose config invalid'; exit 1; }
docker compose --profile operator-portal config --quiet || { echo 'FAIL: optional portal config invalid'; exit 1; }
python3 -m py_compile web/operator-approval/server.py scripts/operator-approvals.py
printf 'Git revision: '; git rev-parse --short HEAD
printf 'Services:\n'; docker compose ps --format 'table {{.Name}}\t{{.Status}}'
printf 'PASS: non-mutating preflight. NO deployment performed.\n'
