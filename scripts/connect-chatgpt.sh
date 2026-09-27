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

case "${VPS_AGENT_AUTH_MODE:-static}" in
  integrated|oidc) ;;
  *)
    echo "ChatGPT public connection requires integrated or external OAuth/OIDC." >&2
    echo "Run scripts/setup-integrated-auth.sh for the default self-hosted path." >&2
    exit 1
    ;;
esac

bash ./scripts/verify-public.sh

cat <<EOF

=== Connect ChatGPT Web ===

MCP endpoint:
  $PUBLIC_URL

Authentication:
  ${VPS_AGENT_AUTH_MODE:-oidc} OAuth/OIDC via ${VPS_AGENT_OIDC_ISSUER:-<issuer>}
  Protected-resource metadata: ${VPS_AGENT_RESOURCE_METADATA_URL:-<origin>/.well-known/oauth-protected-resource}

Expected authenticated subject:
  ${VPS_AGENT_SUBJECT:-operator}

Current OpenAI flow (checked 2026-09-27):
1. Use ChatGPT on the web in an eligible Business, Enterprise or Edu workspace.
2. Enable Developer Mode for your account/workspace as permitted by your role.
3. Open Settings / Workspace Settings -> Apps -> Create.
4. Enter the remote HTTPS MCP endpoint above.
5. Choose OAuth authentication. ChatGPT discovers the Authorization Server and registers its client automatically.
6. Sign in with the VPS operator account created by setup-integrated-auth.sh and approve access.
7. Click Scan Tools, review the discovered tools, then Create the draft app.
8. Start a new chat and select or @mention the draft app.
9. Ask ChatGPT exactly:

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
