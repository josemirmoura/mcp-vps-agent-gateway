#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

if [ ! -f .env ]; then
  echo "Missing .env. Run ./scripts/init.sh first." >&2
  exit 1
fi
set -a
. ./.env
set +a

PUBLIC_URL="${VPS_AGENT_PUBLIC_URL:-}"
if [ -z "$PUBLIC_URL" ]; then
  echo "Set VPS_AGENT_PUBLIC_URL in .env, for example:" >&2
  echo "  VPS_AGENT_PUBLIC_URL=https://mcp.example.com/mcp" >&2
  exit 1
fi

cat <<EOF

=== Connect ChatGPT Web ===

MCP endpoint:
  $PUBLIC_URL

Authentication mode:
  ${VPS_AGENT_AUTH_MODE:-static}

Expected subject:
  ${VPS_AGENT_SUBJECT:-operator}

1. Open ChatGPT Web.
2. Use the currently supported Apps/Plugins/Developer MCP connection flow for your plan/workspace.
3. Add the MCP endpoint above and configure the authentication method from .env.
4. Enable/select this MCP in a fresh chat.
5. Ask ChatGPT exactly:

   Call system.info on my VPS MCP and tell me the hostname.

The tutorial is now shown, but installation is NOT complete.
Waiting for the real audited ChatGPT call...

EOF

docker compose exec -T broker /usr/local/bin/vps-agent wait-tool \
  --subject "${VPS_AGENT_SUBJECT:-operator}" \
  --tool system.info \
  --timeout "${VPS_AGENT_CHATGPT_VERIFY_TIMEOUT:-10m}"

mkdir -p state
printf '{"verified_at":"%s","endpoint":"%s","subject":"%s"}\n' \
  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$PUBLIC_URL" "${VPS_AGENT_SUBJECT:-operator}" \
  > state/chatgpt-verified.json
chmod 600 state/chatgpt-verified.json

echo
echo "CHATGPT WEB CONNECTION VERIFIED"
echo "INSTALLATION COMPLETE"
