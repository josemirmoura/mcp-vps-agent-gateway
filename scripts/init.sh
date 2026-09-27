#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker is required." >&2
  exit 1
fi
docker compose version >/dev/null

if [ ! -f .env ]; then
  cp .env.example .env
  ADMIN_TOKEN="$(openssl rand -hex 32)"
  MCP_TOKEN="$(openssl rand -hex 32)"
  INSTANCE_ID="$(openssl rand -hex 16)"
  sed -i "s/CHANGE_ME_ADMIN_TOKEN/$ADMIN_TOKEN/" .env
  sed -i "s/CHANGE_ME_MCP_TOKEN/$MCP_TOKEN/" .env
  sed -i "s/CHANGE_ME_INSTANCE_ID/$INSTANCE_ID/" .env
  chmod 600 .env
  echo "Created .env with random tokens."
else
  echo ".env already exists; leaving it unchanged."
fi

mkdir -p state
chmod 700 state

if [ ! -f config/policy.yaml ]; then
  cp config/policy.example.yaml config/policy.yaml
  chmod 600 config/policy.yaml
  echo "Created config/policy.yaml from the versioned template."
else
  echo "config/policy.yaml already exists; leaving operator policy unchanged."
fi

if grep -q 'CHANGE_ME_' .env; then
  echo "ERROR: .env still contains CHANGE_ME values." >&2
  exit 1
fi

set -a
. ./.env
set +a

SCOPE_ROOT="${VPS_AGENT_SCOPE_ROOT:-}"
if [ -z "$SCOPE_ROOT" ]; then
  echo "ERROR: set VPS_AGENT_SCOPE_ROOT in .env before starting the Scoped package." >&2
  exit 1
fi
if [ "$SCOPE_ROOT" = "/" ]; then
  echo "NOTE: whole-host authority requires the explicit compose.host.yaml override."
elif [ ! -d "$SCOPE_ROOT" ]; then
  echo "NOTE: VPS_AGENT_SCOPE_ROOT does not exist yet: $SCOPE_ROOT"
  echo "Create that directory or change .env before docker compose up."
fi

docker compose config -q

cat <<EOF

Bootstrap ready.

1. Review .env and choose the physical Scoped ceiling:
     VPS_AGENT_SCOPE_ROOT=$SCOPE_ROOT
   The directory must exist before startup.
2. Edit config/policy.yaml. Filesystem, shell and Compose paths must stay inside that ceiling.
3. Start Scoped mode:
     docker compose up -d --build
   Whole-host is a separate explicit override:
     docker compose -f compose.yaml -f compose.host.yaml up -d --build
4. Verify:
     bash scripts/verify.sh

EOF
