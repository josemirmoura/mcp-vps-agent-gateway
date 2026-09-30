#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# shellcheck source=scripts/lib/product.sh
source scripts/lib/product.sh
vps_agent_init_language ""
vps_agent_banner

mode="safe"
remove_source=0
for arg in "$@"; do
  case "$arg" in
    safe) mode="safe" ;;
    --purge) mode="--purge" ;;
    --remove-source) remove_source=1 ;;
    -h|--help)
      vps_agent_block remove.usage >&2
      exit 0
      ;;
    *)
      echo "$(vps_agent_msg remove.usage_line "script=$0")" >&2
      exit 2
      ;;
  esac
done

if [ "$remove_source" -eq 1 ] && [ "$mode" != "--purge" ]; then
  echo "$(vps_agent_text 'ERROR: --remove-source requires --purge.' 'ERRO: --remove-source requer --purge.')" >&2
  exit 2
fi

repo_root="$PWD"

remove_legacy_sandbox() {
  [ -d /opt/vps-agent-sandbox ] || return 0
  if rmdir /opt/vps-agent-sandbox >/dev/null 2>&1; then
    return 0
  fi
  command -v sudo >/dev/null 2>&1 || return 0
  if [ -t 0 ]; then
    sudo rmdir /opt/vps-agent-sandbox >/dev/null 2>&1 || true
  else
    sudo -n rmdir /opt/vps-agent-sandbox >/dev/null 2>&1 || true
  fi
}

source_verified=0
if [ "$remove_source" -eq 1 ]; then
  git_top="$(git rev-parse --show-toplevel 2>/dev/null || true)"
  if [ "$git_top" != "$repo_root" ] || ! grep -q '^module github.com/josemirmoura/mcp-vps-agent-gateway$' go.mod 2>/dev/null; then
    echo "$(vps_agent_text 'ERROR: refusing source deletion because this directory cannot be verified as a Portico MCP checkout.' 'ERRO: remoção do código recusada porque este diretório não pôde ser verificado como checkout do Portico MCP.')" >&2
    exit 1
  fi
  source_verified=1
fi

if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi

if [ "$mode" = "--purge" ] && [ "${VPS_AGENT_PURGE_CONFIRM:-}" != "PURGE" ]; then
  if [ ! -t 0 ]; then
    echo "$(vps_agent_text 'Full purge requires VPS_AGENT_PURGE_CONFIRM=PURGE in non-interactive mode.' 'O purge completo requer VPS_AGENT_PURGE_CONFIRM=PURGE em modo não interativo.')" >&2
    exit 1
  fi
  printf '%s' "$(vps_agent_text 'Type PURGE to delete this Portico MCP installation configuration/state: ' 'Digite PURGE para apagar a configuração/estado desta instalação do Portico MCP: ')"
  read -r answer
  if [ "$answer" != "PURGE" ]; then
    echo "$(vps_agent_text 'Purge cancelled.' 'Purge cancelado.')"
    exit 1
  fi
fi

if [ "$remove_source" -eq 1 ] && [ "${VPS_AGENT_REMOVE_SOURCE_CONFIRM:-}" != "REMOVE_SOURCE" ]; then
  if [ ! -t 0 ]; then
    echo "$(vps_agent_text 'Source deletion requires VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE in non-interactive mode.' 'A remoção do código requer VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE em modo não interativo.')" >&2
    exit 1
  fi
  printf '%s' "$(vps_agent_text 'Type REMOVE_SOURCE to also delete this Git checkout: ' 'Digite REMOVE_SOURCE para apagar também este checkout Git: ')"
  read -r source_answer
  if [ "$source_answer" != "REMOVE_SOURCE" ]; then
    echo "$(vps_agent_text 'Source deletion cancelled.' 'Remoção do código cancelada.')"
    exit 1
  fi
fi

policy_state="$(vps_agent_msg remove.state_preserved)"
source_state="$(vps_agent_msg remove.state_preserve)"
if [ "$mode" != "safe" ]; then policy_state="$(vps_agent_msg remove.state_deleted)"; fi
if [ "$remove_source" -eq 1 ]; then source_state="$(vps_agent_msg remove.state_delete)"; fi
vps_agent_block remove.summary "mode=$mode" "policy_state=$policy_state" "source_state=$source_state"


project_ids="$(docker ps -aq --filter label=com.docker.compose.project=mcp-vps-agent 2>/dev/null || true)"
if [ ! -f .env ] && [ -z "$project_ids" ]; then
  if [ "$mode" = "--purge" ]; then
    rm -rf -- state backups
    rm -f -- config/policy.yaml
    docker image rm mcp-vps-agent-gateway:local mcp-vps-agent-broker:local >/dev/null 2>&1 || true
    remove_legacy_sandbox
  fi
  echo "$(vps_agent_text 'Runtime already absent; removal is idempotently complete.' 'O runtime já está ausente; a remoção idempotente está concluída.')"
  if [ "$remove_source" -eq 1 ] && [ "$source_verified" -eq 1 ]; then
    parent="$(dirname "$repo_root")"
    cd /
    rm -rf -- "$repo_root"
    if [ "$(basename "$parent")" = "vps-agent-lab" ]; then
      rmdir "$parent" >/dev/null 2>&1 || true
    fi
    echo "$(vps_agent_text 'Portico MCP source checkout removed.' 'Checkout do Portico MCP removido.')"
  fi
  exit 0
fi

compose=(docker compose -f compose.yaml)
if [ "${VPS_AGENT_WHOLE_HOST:-0}" = "1" ]; then
  compose+=(-f compose.host.yaml)
fi
if [ "${VPS_AGENT_AUTH_MODE:-}" = "integrated" ] && [ -f compose.integrated-auth.yaml ]; then
  compose+=(-f compose.integrated-auth.yaml)
  if [ "${VPS_AGENT_BUNDLED_PROXY:-0}" = "1" ] && [ -f compose.integrated-auth.proxy.yaml ]; then
    compose+=(-f compose.integrated-auth.proxy.yaml)
  fi
fi

if [ -f .env ] && [ -n "$project_ids" ]; then
  echo "$(vps_agent_text 'Revoking temporary grants before shutdown...' 'Revogando autorizações temporárias antes do desligamento...')"
  docker compose exec -T broker /usr/local/bin/vps-agent revoke-all --socket /run/vps-agent/broker.sock >/dev/null 2>&1 || true
fi

if [ -f .env ]; then
  old_admin="${VPS_AGENT_ADMIN_TOKEN:-}"
  old_static="${VPS_AGENT_STATIC_TOKEN:-}"
  new_admin="$(openssl rand -hex 32)"
  new_static="$(openssl rand -hex 32)"
  python3 - .env "$old_admin" "$new_admin" "$old_static" "$new_static" <<'PY'
import pathlib, sys
p=pathlib.Path(sys.argv[1])
text=p.read_text()
old_admin,new_admin,old_static,new_static=sys.argv[2:]
if old_admin: text=text.replace(old_admin,new_admin)
if old_static: text=text.replace(old_static,new_static)
p.write_text(text)
PY
  chmod 600 .env
  echo "$(vps_agent_text 'Local admin/static credentials rotated.' 'Credenciais locais admin/static foram rotacionadas.')"
fi

echo "$(vps_agent_text 'Stopping Portico MCP containers...' 'Parando os contêineres do Portico MCP...')"
if [ -f .env ]; then
  "${compose[@]}" down --remove-orphans
else
  [ -z "$project_ids" ] || docker rm -f $project_ids >/dev/null
fi

left="$(docker ps -aq --filter label=com.docker.compose.project=mcp-vps-agent 2>/dev/null || true)"
if [ -n "$left" ]; then
  echo "$(vps_agent_msg remove.incomplete_containers "containers=$left")" >&2
  exit 1
fi

if [ -n "${VPS_AGENT_PUBLIC_URL:-}" ]; then
  code="$(curl --silent --show-error --max-time 5 --output /dev/null --write-out '%{http_code}' "${VPS_AGENT_PUBLIC_URL}" 2>/dev/null || true)"
  case "$code" in
    000|"")
      echo "$(vps_agent_text 'Public MCP endpoint is no longer reachable from this host.' 'O endpoint MCP público não está mais acessível a partir deste host.')"
      ;;
    404|410)
      echo "$(vps_agent_msg remove.public_route_gone "code=$code")"
      ;;
    *)
      echo "$(vps_agent_msg remove.url_still_live "code=$code")" >&2
      echo "$(vps_agent_text 'Check for a stale proxy route before considering removal complete.' 'Verifique uma possível rota antiga no proxy antes de considerar a remoção concluída.')" >&2
      exit 1
      ;;
  esac
fi

if [ "$mode" = "safe" ]; then
  vps_agent_block remove.safe_complete
  exit 0
fi

if [ -f .env ]; then
  "${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
fi

docker volume rm   mcp-vps-agent_broker-run   mcp-vps-agent_zitadel-postgres-data   mcp-vps-agent_zitadel-bootstrap   mcp-vps-agent_vps-agent-letsencrypt   >/dev/null 2>&1 || true

rm -rf -- state backups
rm -f -- .env config/policy.yaml

# These are package-owned default local build outputs. Never remove arbitrary
# images configured by the operator or images merely used by managed workloads.
docker image rm   mcp-vps-agent-gateway:local   mcp-vps-agent-broker:local   >/dev/null 2>&1 || true

# Legacy pre-Portico scaffold. The helper uses rmdir, never rm -rf:
# a non-empty directory is preserved rather than risking user data.
remove_legacy_sandbox

vps_agent_block remove.purge_complete


if [ "$remove_source" -eq 1 ] && [ "$source_verified" -eq 1 ]; then
  parent="$(dirname "$repo_root")"
  echo "$(vps_agent_msg remove.deleting_source "path=$repo_root")"
  cd /
  rm -rf -- "$repo_root"
  if [ "$(basename "$parent")" = "vps-agent-lab" ]; then
    rmdir "$parent" >/dev/null 2>&1 || true
  fi
  echo "$(vps_agent_text 'Portico MCP source checkout removed.' 'Checkout do Portico MCP removido.')"
fi
