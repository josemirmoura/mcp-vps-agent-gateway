#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# shellcheck source=scripts/lib/product.sh
source scripts/lib/product.sh
vps_agent_init_language ""

APPLY=0
ARGS=()
for arg in "$@"; do
  case "$arg" in
    --apply) APPLY=1 ;;
    *) ARGS+=("$arg") ;;
  esac
done

if [ "${#ARGS[@]}" -eq 0 ]; then
  vps_agent_block delegate.usage >&2
  exit 2
fi

python3 scripts/delegate-root.py "${ARGS[@]}"

case "${ARGS[0]}" in
  add|remove) ;;
  *) exit 0 ;;
esac

if [ "$APPLY" -ne 1 ]; then
  vps_agent_block delegate.pending
  exit 0
fi

set -a
. ./.env
set +a

compose=(docker compose -f compose.yaml)
if [ "${VPS_AGENT_WHOLE_HOST:-0}" = "1" ]; then
  compose+=(-f compose.host.yaml)
fi

echo "$(vps_agent_text 'Reloading Broker with the updated policy...' 'Recarregando o Broker com a policy atualizada...')"
"${compose[@]}" up -d --force-recreate broker
bash scripts/verify.sh
