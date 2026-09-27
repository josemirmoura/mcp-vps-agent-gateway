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
AUTH_MODE="${VPS_AGENT_AUTH_MODE:-static}"
ISSUER="${VPS_AGENT_OIDC_ISSUER:-}"

if [ -z "$PUBLIC_URL" ]; then
  echo "VPS_AGENT_PUBLIC_URL is required." >&2
  exit 1
fi
case "$PUBLIC_URL" in
  https://*) ;;
  *)
    echo "Public MCP endpoint must use HTTPS: $PUBLIC_URL" >&2
    exit 1
    ;;
esac
if [ "$AUTH_MODE" != "oidc" ] && [ "$AUTH_MODE" != "integrated" ]; then
  echo "Public ChatGPT verification requires integrated or oidc authentication." >&2
  echo "Static bearer/no-auth is reserved for local/lab acceptance." >&2
  exit 1
fi
if [ -z "$ISSUER" ]; then
  echo "VPS_AGENT_OIDC_ISSUER is required in oidc mode." >&2
  exit 1
fi

ORIGIN="$(printf '%s' "$PUBLIC_URL" | sed -E 's#^(https://[^/]+).*$#\1#')"
METADATA_URL="${VPS_AGENT_RESOURCE_METADATA_URL:-$ORIGIN/.well-known/oauth-protected-resource}"
RESOURCE="${VPS_AGENT_OAUTH_RESOURCE:-$PUBLIC_URL}"

echo "Checking public health endpoint..."
curl --fail --silent --show-error "$ORIGIN/healthz" >/tmp/vps-agent-public-health.json
cat /tmp/vps-agent-public-health.json

echo
echo "Checking OAuth Protected Resource Metadata..."
curl --fail --silent --show-error "$METADATA_URL" >/tmp/vps-agent-prm.json
cat /tmp/vps-agent-prm.json

python3 - "$RESOURCE" "$ISSUER" <<'PY'
import json, sys
resource, issuer = sys.argv[1:]
data = json.load(open("/tmp/vps-agent-prm.json"))
assert data.get("resource") == resource, (data, resource)
servers = [x.rstrip("/") for x in data.get("authorization_servers", [])]
assert issuer.rstrip("/") in servers, (data, issuer)
assert "header" in data.get("bearer_methods_supported", []), data
print("OAUTH DISCOVERY: PASS")
PY

if [ "$AUTH_MODE" = "integrated" ]; then
  echo
  echo "Checking integrated Authorization Server discovery..."
  curl --fail --silent --show-error "$ISSUER/.well-known/openid-configuration" >/tmp/vps-agent-oidc-discovery.json
  python3 - "$ISSUER" <<'PY'
import json,sys
issuer=sys.argv[1].rstrip("/")
data=json.load(open("/tmp/vps-agent-oidc-discovery.json"))
assert data.get("issuer","").rstrip("/") == issuer, data
registration=data.get("registration_endpoint","")
assert registration.startswith("https://"), data
methods=data.get("code_challenge_methods_supported", [])
assert "S256" in methods, data
print("INTEGRATED OAUTH DISCOVERY + DCR + PKCE: PASS")
PY
fi

echo
echo "Checking that unauthenticated MCP access fails closed..."
code="$(curl --silent --show-error --output /tmp/vps-agent-public-unauth.txt --write-out '%{http_code}' "$PUBLIC_URL" || true)"
if [ "$code" != "401" ]; then
  echo "Expected HTTP 401 from unauthenticated MCP request, got $code." >&2
  cat /tmp/vps-agent-public-unauth.txt >&2 || true
  exit 1
fi
echo "UNAUTHENTICATED MCP DENIAL: PASS"

if [ -n "${VPS_AGENT_TEST_ACCESS_TOKEN:-}" ]; then
  echo
  echo "Testing a real OAuth access token against system.info..."
  docker compose exec -T gateway /usr/local/bin/vps-agent-mcp-call \
    --endpoint "$PUBLIC_URL" \
    --token "$VPS_AGENT_TEST_ACCESS_TOKEN" \
    --tool system.info \
    --args '{}' >/tmp/vps-agent-public-system-info.json
  cat /tmp/vps-agent-public-system-info.json
  python3 - <<'PY'
import json
x=json.load(open("/tmp/vps-agent-public-system-info.json"))
assert x["is_error"] is False, x
print("PUBLIC OAUTH MCP CALL: PASS")
PY
else
  echo
  echo "No VPS_AGENT_TEST_ACCESS_TOKEN set; token-authenticated public call not executed."
  echo "OAuth discovery and fail-closed behavior are valid. The ChatGPT authorization flow remains the final client-side gate."
fi
