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
      if vps_agent_is_pt_br; then
        cat <<'EOF'
uso: bash scripts/remove.sh [safe|--purge] [--remove-source]

safe             Para o Portico MCP e gira credenciais locais, preservando
                 configuração, policy, auditoria e identidade integrada.
--purge          Remove runtime, volumes, estado/configuração local,
                 imagens locais padrão e estruturas legadas vazias.
--remove-source  Apenas com --purge: remove também este checkout Git após
                 uma segunda confirmação explícita.
EOF
      else
        cat <<'EOF'
usage: bash scripts/remove.sh [safe|--purge] [--remove-source]

safe             Stop Portico MCP and rotate local credentials while preserving
                 configuration, policy, audit state and integrated identity.
--purge          Delete MCP-owned runtime, volumes, local state/configuration,
                 default local package images and empty legacy scaffolding.
--remove-source  With --purge only: also delete this Git checkout after a
                 separate explicit confirmation.
EOF
      fi
      exit 0
      ;;
    *)
      echo "$(vps_agent_text "usage: $0 [safe|--purge] [--remove-source]" "uso: $0 [safe|--purge] [--remove-source]")" >&2
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

if vps_agent_is_pt_br; then
  cat <<EOF
Modo de remoção: $mode
Runtime: Gateway + Broker + proxy do pacote (quando configurado)
Recursos gerenciados da VPS: PRESERVADOS
Policy/auditoria do operador: $([ "$mode" = "safe" ] && echo PRESERVADAS || echo APAGADAS)
Checkout do código: $([ "$remove_source" -eq 1 ] && echo APAGAR || echo PRESERVAR)
EOF
else
  cat <<EOF
Removal mode: $mode
Runtime: Gateway + Broker + package proxy (when configured)
Managed VPS resources: PRESERVED
Operator policy/audit: $([ "$mode" = "safe" ] && echo PRESERVED || echo DELETED)
Source checkout: $([ "$remove_source" -eq 1 ] && echo DELETE || echo PRESERVED)
EOF
fi

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
  echo "$(vps_agent_text "Removal incomplete: package containers still exist: $left" "Remoção incompleta: ainda existem contêineres do pacote: $left")" >&2
  exit 1
fi

if [ -n "${VPS_AGENT_PUBLIC_URL:-}" ]; then
  code="$(curl --silent --show-error --max-time 5 --output /dev/null --write-out '%{http_code}' "${VPS_AGENT_PUBLIC_URL}" 2>/dev/null || true)"
  case "$code" in
    000|"")
      echo "$(vps_agent_text 'Public MCP endpoint is no longer reachable from this host.' 'O endpoint MCP público não está mais acessível a partir deste host.')"
      ;;
    404|410)
      echo "$(vps_agent_text "Public MCP route is gone (HTTP $code from the remaining edge proxy)." "A rota MCP pública foi removida (HTTP $code no proxy de borda remanescente).")"
      ;;
    *)
      echo "Removal incomplete: the MCP URL still returned HTTP $code instead of disappearing." >&2
      echo "$(vps_agent_text 'Check for a stale proxy route before considering removal complete.' 'Verifique uma possível rota antiga no proxy antes de considerar a remoção concluída.')" >&2
      exit 1
      ;;
  esac
fi

if [ "$mode" = "safe" ]; then
  if vps_agent_is_pt_br; then
    cat <<'EOF'

REMOÇÃO SEGURA CONCLUÍDA
Removidos:
  runtime Gateway/Broker/proxy do pacote
  autorizações temporárias ativas
Invalidadas:
  credenciais bearer locais admin/static anteriores
Preservados:
  .env (com credenciais locais rotacionadas)
  config/policy.yaml
  state/ auditoria e histórico de operações
  estado da identidade integrada
  todos os recursos da VPS que o Portico podia gerenciar
EOF
  else
    cat <<'EOF'

SAFE REMOVE COMPLETE
Removed:
  Gateway/Broker/package proxy runtime
  active temporary grants
Invalidated:
  previous local admin/static bearer credentials
Preserved:
  .env (with rotated local credentials)
  config/policy.yaml
  state/ audit and operation history
  integrated identity state on safe remove
  every VPS resource Portico MCP was allowed to manage
EOF
  fi
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

if vps_agent_is_pt_br; then
  cat <<EOF

PURGE COMPLETO CONCLUÍDO
Artefatos locais do Portico removidos:
  runtime/volumes
  imagens locais padrão do Gateway/Broker (quando presentes)
  .env
  config/policy.yaml
  state/
  backups/
  /opt/vps-agent-sandbox legado vazio (quando presente)
Preservados:
  aplicações, sites, bancos, imagens/contêineres de terceiros, serviços e arquivos delegados
EOF
else
  cat <<EOF

FULL PURGE COMPLETE
Deleted MCP-owned local artifacts:
  runtime/volumes
  default local Gateway/Broker images (when present)
  .env
  config/policy.yaml
  state/
  backups/
  empty legacy /opt/vps-agent-sandbox (when present)
Preserved:
  applications, sites, databases, third-party images/containers, services and delegated files
EOF
fi

if [ "$remove_source" -eq 1 ] && [ "$source_verified" -eq 1 ]; then
  parent="$(dirname "$repo_root")"
  echo "$(vps_agent_text "Deleting verified Portico MCP source checkout: $repo_root" "Apagando o checkout verificado do Portico MCP: $repo_root")"
  cd /
  rm -rf -- "$repo_root"
  if [ "$(basename "$parent")" = "vps-agent-lab" ]; then
    rmdir "$parent" >/dev/null 2>&1 || true
  fi
  echo "$(vps_agent_text 'Portico MCP source checkout removed.' 'Checkout do Portico MCP removido.')"
fi
