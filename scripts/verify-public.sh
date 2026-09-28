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
if [ "$AUTH_MODE" != "oidc" ]; then
  echo "Public ChatGPT verification requires VPS_AGENT_AUTH_MODE=oidc." >&2
  echo "Static bearer/no-auth is reserved for local/lab acceptance." >&2
  exit 1
fi
if [ -z "$ISSUER" ]; then
  echo "VPS_AGENT_OIDC_ISSUER is required in oidc mode." >&2
  exit 1
fi

# Match the Gateway's canonical issuer handling.
while [ "$ISSUER" != "/" ] && [ "${ISSUER%/}" != "$ISSUER" ]; do
  ISSUER="${ISSUER%/}"
done

ORIGIN="$(printf '%s' "$PUBLIC_URL" | sed -E 's#^(https://[^/]+).*$#\1#')"
METADATA_URL="${VPS_AGENT_RESOURCE_METADATA_URL:-$ORIGIN/.well-known/oauth-protected-resource}"
RESOURCE="${VPS_AGENT_OAUTH_RESOURCE:-$PUBLIC_URL}"
OIDC_METADATA_URL="$ISSUER/.well-known/openid-configuration"
REQUIRED_SCOPES="${VPS_AGENT_REQUIRED_SCOPES:-}"
PREDEFINED_CLIENT="${VPS_AGENT_PREDEFINED_OAUTH_CLIENT:-0}"

echo "Checking public health endpoint..."
curl --fail --silent --show-error "$ORIGIN/healthz" >/tmp/vps-agent-public-health.json
cat /tmp/vps-agent-public-health.json

echo
echo "Checking OAuth Protected Resource Metadata..."
curl --fail --silent --show-error "$METADATA_URL" >/tmp/vps-agent-prm.json
cat /tmp/vps-agent-prm.json

python3 - "$RESOURCE" "$ISSUER" "$REQUIRED_SCOPES" <<'PY'
import json
import sys

resource, issuer, required_raw = sys.argv[1:]
data = json.load(open("/tmp/vps-agent-prm.json"))

assert data.get("resource") == resource, (data, resource)
servers = data.get("authorization_servers", [])
assert issuer in servers, (data, issuer)
assert "header" in data.get("bearer_methods_supported", []), data

required = set(required_raw.replace(",", " ").split())
supported = set(data.get("scopes_supported", []))
if required:
    assert required <= supported, {
        "missing_scopes": sorted(required - supported),
        "protected_resource_metadata": data,
    }

print("OAUTH PROTECTED RESOURCE METADATA: PASS")
PY

echo
echo "Checking Authorization Server / OpenID Connect discovery..."
curl --fail --silent --show-error "$OIDC_METADATA_URL" >/tmp/vps-agent-oidc.json
cat /tmp/vps-agent-oidc.json

python3 - "$ISSUER" "$REQUIRED_SCOPES" "$PREDEFINED_CLIENT" <<'PY'
import json
import sys
from urllib.parse import urlparse

issuer, required_raw, predefined = sys.argv[1:]
data = json.load(open("/tmp/vps-agent-oidc.json"))

assert data.get("issuer") == issuer, {
    "expected_issuer": issuer,
    "advertised_issuer": data.get("issuer"),
}

for field in ("authorization_endpoint", "token_endpoint"):
    value = data.get(field)
    assert isinstance(value, str) and urlparse(value).scheme == "https" and urlparse(value).netloc, {
        "missing_or_insecure": field,
        "value": value,
    }

assert "S256" in data.get("code_challenge_methods_supported", []), {
    "missing_pkce_method": "S256",
    "code_challenge_methods_supported": data.get("code_challenge_methods_supported", []),
}

methods = set(data.get("token_endpoint_auth_methods_supported", []))
assert methods, "token_endpoint_auth_methods_supported must be published"

cimd = data.get("client_id_metadata_document_supported") is True
registration = data.get("registration_endpoint")
dcr = isinstance(registration, str) and bool(registration)

if dcr:
    parsed = urlparse(registration)
    assert parsed.scheme == "https" and parsed.netloc, {
        "registration_endpoint_must_use_https": registration,
    }

if not cimd and not dcr and predefined != "1":
    raise AssertionError(
        "Authorization Server advertises neither CIMD nor DCR. "
        "If a predefined ChatGPT OAuth client is deliberately configured, "
        "set VPS_AGENT_PREDEFINED_OAUTH_CLIENT=1."
    )

if cimd:
    supported_by_chatgpt_cimd = {"none", "private_key_jwt"}
    assert methods & supported_by_chatgpt_cimd, {
        "cimd_requires_compatible_token_endpoint_auth_method": sorted(methods),
        "chatgpt_supported": sorted(supported_by_chatgpt_cimd),
    }
else:
    common_methods = {"none", "private_key_jwt", "client_secret_post", "client_secret_basic"}
    assert methods & common_methods, {
        "no_known_chatgpt_compatible_token_endpoint_auth_method": sorted(methods),
    }

required = set(required_raw.replace(",", " ").split())
advertised_scopes = set(data.get("scopes_supported", []))
if required and advertised_scopes:
    assert required <= advertised_scopes, {
        "required_scopes_not_advertised_by_authorization_server": sorted(required - advertised_scopes),
    }

if "offline_access" not in advertised_scopes:
    print(
        "WARNING: discovery metadata does not advertise offline_access; "
        "long-lived ChatGPT connectivity may require reauthentication."
    )

if data.get("authorization_response_iss_parameter_supported") is True:
    print("OAUTH CALLBACK ISSUER IDENTIFICATION: PASS (stable callback eligible)")
else:
    print("OAUTH CALLBACK ISSUER IDENTIFICATION: not advertised; callback-specific redirect may be used")

mode = "CIMD" if cimd else ("DCR" if dcr else "predefined client")
print(f"AUTHORIZATION SERVER DISCOVERY: PASS ({mode})")
PY

echo
echo "Checking that unauthenticated MCP access fails closed with OAuth discovery challenge..."
code="$(curl --silent --show-error   --dump-header /tmp/vps-agent-public-unauth-headers.txt   --output /tmp/vps-agent-public-unauth.txt   --write-out '%{http_code}'   "$PUBLIC_URL" || true)"
if [ "$code" != "401" ]; then
  echo "Expected HTTP 401 from unauthenticated MCP request, got $code." >&2
  cat /tmp/vps-agent-public-unauth.txt >&2 || true
  exit 1
fi

python3 - "$METADATA_URL" <<'PY'
import sys

metadata_url = sys.argv[1]
raw = open("/tmp/vps-agent-public-unauth-headers.txt", errors="replace").read()
headers = {}
for line in raw.splitlines():
    if ":" not in line:
        continue
    name, value = line.split(":", 1)
    headers.setdefault(name.strip().lower(), []).append(value.strip())

challenges = headers.get("www-authenticate", [])
assert challenges, "401 response is missing WWW-Authenticate"
joined = ", ".join(challenges)
assert "bearer" in joined.lower(), joined
needle = f'resource_metadata="{metadata_url}"'
assert needle in joined, {
    "expected": needle,
    "www_authenticate": joined,
}
print("UNAUTHENTICATED MCP DENIAL + OAUTH CHALLENGE: PASS")
PY

if [ -n "${VPS_AGENT_TEST_ACCESS_TOKEN:-}" ]; then
  echo
  echo "Testing a real OAuth access token against system.info..."
  docker compose exec -T gateway /usr/local/bin/vps-agent-mcp-call     --endpoint "$PUBLIC_URL"     --token "$VPS_AGENT_TEST_ACCESS_TOKEN"     --tool system.info     --args '{}' >/tmp/vps-agent-public-system-info.json
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
  echo "OAuth discovery, PKCE/client-registration metadata and fail-closed behavior are valid."
  echo "The real ChatGPT authorization flow remains the final client-side gate."
fi
