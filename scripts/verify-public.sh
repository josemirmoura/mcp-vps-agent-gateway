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
AUDIENCE_PROJECT_ID="${VPS_AGENT_INTEGRATED_AUDIENCE_PROJECT_ID:-}"
REQUIRED_SCOPES="${VPS_AGENT_REQUIRED_SCOPES:-}"

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
if [ "$AUTH_MODE" != "integrated" ]; then
  echo "Public ChatGPT verification requires the integrated self-hosted OAuth mode." >&2
  echo "Run scripts/setup-integrated-auth.sh first." >&2
  exit 1
fi
if [ -z "$ISSUER" ]; then
  echo "VPS_AGENT_OIDC_ISSUER is required in integrated mode." >&2
  exit 1
fi
while [ "$ISSUER" != "/" ] && [ "${ISSUER%/}" != "$ISSUER" ]; do
  ISSUER="${ISSUER%/}"
done
if [ -z "$AUDIENCE_PROJECT_ID" ]; then
  echo "VPS_AGENT_INTEGRATED_AUDIENCE_PROJECT_ID is required in integrated mode." >&2
  exit 1
fi
AUDIENCE_SCOPE="urn:zitadel:iam:org:project:id:$AUDIENCE_PROJECT_ID:aud"
case " $REQUIRED_SCOPES " in
  *" openid "*" $AUDIENCE_SCOPE "*) ;;
  *)
    echo "Integrated OAuth must require both openid and the dedicated resource audience scope." >&2
    exit 1
    ;;
esac

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

python3 - "$RESOURCE" "$ISSUER" "$REQUIRED_SCOPES" <<'PY'
import json, sys
resource, issuer, required_raw = sys.argv[1:]
data = json.load(open("/tmp/vps-agent-prm.json"))
assert data.get("resource") == resource, (data, resource)
servers = [x.rstrip("/") for x in data.get("authorization_servers", [])]
assert issuer.rstrip("/") in servers, (data, issuer)
assert "header" in data.get("bearer_methods_supported", []), data
required=set(required_raw.split())
advertised=set(data.get("scopes_supported", []))
assert required <= advertised, (required, advertised)
print("OAUTH PROTECTED RESOURCE + SCOPES: PASS")
PY

echo
echo "Checking integrated Authorization Server discovery..."
curl --fail --silent --show-error "$ISSUER/.well-known/openid-configuration" >/tmp/vps-agent-oidc-discovery.json
python3 - "$ISSUER" <<'PY'
import json,sys
from urllib.parse import urlparse
issuer=sys.argv[1].rstrip("/")
data=json.load(open("/tmp/vps-agent-oidc-discovery.json"))
assert data.get("issuer","").rstrip("/") == issuer, data
for field in ("authorization_endpoint", "token_endpoint"):
    value=data.get(field,"")
    parsed=urlparse(value)
    assert parsed.scheme == "https" and parsed.netloc, {"missing_or_insecure": field, "value": value}
registration=data.get("registration_endpoint","")
parsed=urlparse(registration)
assert parsed.scheme == "https" and parsed.netloc, data
methods=data.get("code_challenge_methods_supported", [])
assert "S256" in methods, data
scopes=data.get("scopes_supported", [])
assert "offline_access" in scopes, data
grants=data.get("grant_types_supported", [])
assert "authorization_code" in grants, data
assert "refresh_token" in grants, data
token_auth=data.get("token_endpoint_auth_methods_supported", [])
assert token_auth, "token_endpoint_auth_methods_supported must be published"
assert "none" in token_auth, data
print("INTEGRATED OAUTH DISCOVERY + DCR + PKCE + REFRESH: PASS")
PY

echo
echo "Checking private audience-bound token introspection..."
docker compose exec -T gateway sh -c '
  curl --fail --silent --show-error     --request POST     --url "$VPS_AGENT_INTEGRATED_INTROSPECTION_URL"     --user "$VPS_AGENT_INTEGRATED_INTROSPECTION_CLIENT_ID:$VPS_AGENT_INTEGRATED_INTROSPECTION_CLIENT_SECRET"     --header "Host: $VPS_AGENT_INTEGRATED_INTROSPECTION_HOST"     --header "X-Forwarded-Proto: https"     --header "Content-Type: application/x-www-form-urlencoded"     --data "token=deliberately-invalid-verification-probe"
' >/tmp/vps-agent-introspection-probe.json
python3 - <<'PY'
import json
x=json.load(open("/tmp/vps-agent-introspection-probe.json"))
assert x.get("active") is False, x
print("PRIVATE AUDIENCE-BOUND INTROSPECTION: PASS")
PY

echo
echo "Checking that unauthenticated MCP access fails closed with OAuth discovery challenge..."
code="$(curl --silent --show-error \
  --dump-header /tmp/vps-agent-public-unauth-headers.txt \
  --output /tmp/vps-agent-public-unauth.txt \
  --write-out '%{http_code}' \
  "$PUBLIC_URL" || true)"
if [ "$code" != "401" ]; then
  echo "Expected HTTP 401 from unauthenticated MCP request, got $code." >&2
  cat /tmp/vps-agent-public-unauth.txt >&2 || true
  exit 1
fi
python3 - "$METADATA_URL" <<'PY'
import sys
metadata_url=sys.argv[1]
raw=open("/tmp/vps-agent-public-unauth-headers.txt", errors="replace").read()
headers={}
for line in raw.splitlines():
    if ":" not in line:
        continue
    name,value=line.split(":",1)
    headers.setdefault(name.strip().lower(), []).append(value.strip())
challenges=headers.get("www-authenticate", [])
assert challenges, "401 response is missing WWW-Authenticate"
joined=", ".join(challenges)
assert "bearer" in joined.lower(), joined
needle=f'resource_metadata="{metadata_url}"'
assert needle in joined, {"expected": needle, "www_authenticate": joined}
print("UNAUTHENTICATED MCP DENIAL + OAUTH CHALLENGE: PASS")
PY

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
  echo "OAuth discovery, DCR/PKCE, private introspection and fail-closed challenge behavior are valid."
  echo "The real ChatGPT authorization flow remains the final client-side gate."
fi
