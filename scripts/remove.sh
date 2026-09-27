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

compose=(docker compose)
if [ -n "${VPS_AGENT_DOMAIN:-}" ] && [ -f compose.https.yaml ]; then
  compose=(docker compose -f compose.yaml -f compose.https.yaml)
fi

echo "Stopping MCP VPS Agent containers. Managed VPS resources are never deleted by this script."
"${compose[@]}" down --remove-orphans

if [ -n "${VPS_AGENT_PUBLIC_URL:-}" ]; then
  code="$(curl --silent --show-error --max-time 5 --output /dev/null --write-out '%{http_code}' "${VPS_AGENT_PUBLIC_URL}" 2>/dev/null || true)"
  if [ -n "$code" ] && [ "$code" != "000" ]; then
    echo "WARNING: public endpoint still returned HTTP $code; check external reverse proxy/tunnel routing." >&2
  else
    echo "Public endpoint is no longer reachable from this host."
  fi
fi

if [ "$mode" = "safe" ]; then
  cat <<'EOF'

SAFE REMOVE COMPLETE
Preserved:
  .env
  config/policy.yaml
  state/
  Caddy data volumes (if used)

To reinstall, run docker compose up -d --build and verify again.
EOF
  exit 0
fi

if [ "${VPS_AGENT_PURGE_CONFIRM:-}" != "PURGE" ]; then
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

"${compose[@]}" down -v --remove-orphans || true
rm -rf -- state backups
rm -f -- .env config/policy.yaml

cat <<'EOF'

FULL PURGE COMPLETE
Deleted only MCP-owned local artifacts:
  .env
  config/policy.yaml
  state/
  backups/
  package-owned Docker volumes

Applications, sites, databases, containers, systemd services and files that the MCP was authorized to manage were not deleted.
EOF
