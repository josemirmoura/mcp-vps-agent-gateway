#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

mode="${1:-safe}"
if [ "$mode" != "safe" ] && [ "$mode" != "--purge" ]; then
  echo "usage: $0 [safe|--purge]" >&2
  exit 2
fi

if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi

if [ "$mode" = "--purge" ] && [ "${VPS_AGENT_PURGE_CONFIRM:-}" != "PURGE" ]; then
  if [ ! -t 0 ]; then
    echo "Full purge requires VPS_AGENT_PURGE_CONFIRM=PURGE in non-interactive mode." >&2
    exit 1
  fi
  printf 'Type PURGE to delete this MCP installation configuration/state: '
  read -r answer
  if [ "$answer" != "PURGE" ]; then
    echo "Purge cancelled."
    exit 1
  fi
fi

cat <<EOF
Removal mode: $mode
Runtime: Gateway + Broker + package proxy (when configured)
Managed VPS resources: PRESERVED
Operator policy/audit: $([ "$mode" = "safe" ] && echo PRESERVED || echo DELETED)
EOF

project_ids="$(docker ps -aq --filter label=com.docker.compose.project=mcp-vps-agent 2>/dev/null || true)"
if [ ! -f .env ] && [ -z "$project_ids" ]; then
  if [ "$mode" = "--purge" ]; then
    rm -rf -- state backups
    rm -f -- config/policy.yaml
  fi
  echo "Runtime already absent; removal is idempotently complete."
  exit 0
fi

compose=(docker compose -f compose.yaml)
if [ "${VPS_AGENT_WHOLE_HOST:-0}" = "1" ]; then
  compose+=(-f compose.host.yaml)
fi
if [ -n "${VPS_AGENT_DOMAIN:-}" ] && [ -f compose.https.yaml ]; then
  compose+=(-f compose.https.yaml)
fi

if [ -f .env ] && [ -n "$project_ids" ]; then
  echo "Revoking temporary grants before shutdown..."
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
  echo "Local admin/static credentials rotated."
fi

echo "Stopping MCP VPS Agent containers..."
if [ -f .env ]; then
  "${compose[@]}" down --remove-orphans
else
  [ -z "$project_ids" ] || docker rm -f $project_ids >/dev/null
fi

left="$(docker ps -aq --filter label=com.docker.compose.project=mcp-vps-agent 2>/dev/null || true)"
if [ -n "$left" ]; then
  echo "Removal incomplete: package containers still exist: $left" >&2
  exit 1
fi

if [ -n "${VPS_AGENT_PUBLIC_URL:-}" ]; then
  code="$(curl --silent --show-error --max-time 5 --output /dev/null --write-out '%{http_code}' "${VPS_AGENT_PUBLIC_URL}" 2>/dev/null || true)"
  if [ -n "$code" ] && [ "$code" != "000" ]; then
    echo "Removal incomplete: public endpoint still returned HTTP $code." >&2
    echo "Disable the external reverse proxy/tunnel/DNS route and run this command again." >&2
    exit 1
  fi
  echo "Public endpoint is no longer reachable from this host."
fi

if [ "$mode" = "safe" ]; then
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
  package Caddy data volumes
  every VPS resource the MCP was allowed to manage
EOF
  exit 0
fi

if [ -f .env ]; then
  "${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
fi
docker volume rm mcp-vps-agent_broker-run mcp-vps-agent_caddy-data mcp-vps-agent_caddy-config >/dev/null 2>&1 || true
rm -rf -- state backups
rm -f -- .env config/policy.yaml

cat <<'EOF'

FULL PURGE COMPLETE
Deleted only MCP-owned local artifacts:
  runtime/volumes
  .env
  config/policy.yaml
  state/
  backups/
Preserved:
  applications, sites, databases, containers, services and files the MCP previously administered
EOF
