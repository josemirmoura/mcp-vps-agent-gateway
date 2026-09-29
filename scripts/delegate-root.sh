#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

APPLY=0
ARGS=()
for arg in "$@"; do
  case "$arg" in
    --apply) APPLY=1 ;;
    *) ARGS+=("$arg") ;;
  esac
done

if [ "${#ARGS[@]}" -eq 0 ]; then
  cat >&2 <<'EOF'
usage:
  bash scripts/delegate-root.sh add /opt/project [--access read|work|compose] [--apply]
  bash scripts/delegate-root.sh remove /opt/project [--apply]
  bash scripts/delegate-root.sh list

Access profiles:
  read      filesystem read only
  work      filesystem read/write + scoped shell cwd (default)
  compose   work + Docker Compose inspect/manage for that project directory

The path must be inside VPS_AGENT_SCOPE_ROOT. Delegating the physical ceiling
itself is refused unless --allow-ceiling is explicitly supplied.
EOF
  exit 2
fi

python3 scripts/delegate-root.py "${ARGS[@]}"

case "${ARGS[0]}" in
  add|remove) ;;
  *) exit 0 ;;
esac

if [ "$APPLY" -ne 1 ]; then
  cat <<'EOF'

Policy file updated but the running Broker has not been reloaded.
Review first:
  python3 scripts/authority-summary.py

Then apply:
  bash scripts/delegate-root.sh list
  docker compose up -d --force-recreate broker
  bash scripts/verify.sh

Or rerun the add/remove command with --apply.
EOF
  exit 0
fi

set -a
. ./.env
set +a

compose=(docker compose -f compose.yaml)
if [ "${VPS_AGENT_WHOLE_HOST:-0}" = "1" ]; then
  compose+=(-f compose.host.yaml)
fi

echo "Reloading Broker with the updated policy..."
"${compose[@]}" up -d --force-recreate broker
bash scripts/verify.sh
