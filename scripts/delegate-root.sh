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
  if vps_agent_is_pt_br; then
    cat >&2 <<'EOF'
uso:
  bash scripts/delegate-root.sh add /opt/projeto [--access read|work|compose] [--apply]
  bash scripts/delegate-root.sh remove /opt/projeto [--apply]
  bash scripts/delegate-root.sh list

Perfis de acesso:
  read      somente leitura do filesystem
  work      leitura/escrita + cwd de shell confinado (padrão)
  compose   work + inspeção/gerência Docker Compose nessa pasta

O caminho deve estar dentro de VPS_AGENT_SCOPE_ROOT. A delegação do próprio
teto físico requer --allow-ceiling quando feita manualmente pelo terminal.
EOF
  else
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
itself requires --allow-ceiling when done manually from the terminal.
EOF
  fi
  exit 2
fi

python3 scripts/delegate-root.py "${ARGS[@]}"

case "${ARGS[0]}" in
  add|remove) ;;
  *) exit 0 ;;
esac

if [ "$APPLY" -ne 1 ]; then
  if vps_agent_is_pt_br; then
    cat <<'EOF'

A policy foi atualizada, mas o Broker em execução ainda não foi recarregado.
Revise primeiro:
  python3 scripts/authority-summary.py

Depois aplique:
  bash scripts/delegate-root.sh list
  docker compose up -d --force-recreate broker
  bash scripts/verify.sh

Ou repita o comando add/remove usando --apply.
EOF
  else
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
  fi
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
