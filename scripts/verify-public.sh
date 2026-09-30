#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# shellcheck source=scripts/lib/product.sh
source scripts/lib/product.sh
vps_agent_init_language ""

# shellcheck source=scripts/lib/oauth-scopes.sh
. ./scripts/lib/oauth-scopes.sh

if [ ! -f .env ]; then
  echo "$(vps_agent_text 'Missing .env. Run scripts/init.sh first.' 'Arquivo .env ausente. Execute scripts/init.sh primeiro.')" >&2
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
  echo "$(vps_agent_text 'VPS_AGENT_PUBLIC_URL is required.' 'VPS_AGENT_PUBLIC_URL é obrigatória.')" >&2
  exit 1
fi
case "$PUBLIC_URL" in
  https://*) ;;
  *)
    echo "$(vps_agent_msg error.public_url_https "url=$PUBLIC_URL")" >&2
    exit 1
    ;;
esac
if [ "$AUTH_MODE" != "integrated" ]; then
  echo "$(vps_agent_text 'Public ChatGPT verification requires integrated OAuth mode.' 'A verificação pública com o ChatGPT requer o modo OAuth integrado.')" >&2
  echo "$(vps_agent_text 'Run scripts/setup-integrated-auth.sh first.' 'Execute scripts/setup-integrated-auth.sh primeiro.')" >&2
  exit 1
fi
if [ -z "$ISSUER" ]; then
  echo "$(vps_agent_text 'VPS_AGENT_OIDC_ISSUER is required in integrated mode.' 'VPS_AGENT_OIDC_ISSUER é obrigatória no modo integrado.')" >&2
  exit 1
fi
while [ "$ISSUER" != "/" ] && [ "${ISSUER%/}" != "$ISSUER" ]; do
  ISSUER="${ISSUER%/}"
done
if [ -z "$AUDIENCE_PROJECT_ID" ]; then
  echo "$(vps_agent_text 'VPS_AGENT_INTEGRATED_AUDIENCE_PROJECT_ID is required in integrated mode.' 'VPS_AGENT_INTEGRATED_AUDIENCE_PROJECT_ID é obrigatória no modo integrado.')" >&2
  exit 1
fi
AUDIENCE_SCOPE="urn:zitadel:iam:org:project:id:$AUDIENCE_PROJECT_ID:aud"
if ! vps_agent_scope_list_contains "$REQUIRED_SCOPES" openid || ! vps_agent_scope_list_contains "$REQUIRED_SCOPES" "$AUDIENCE_SCOPE"; then
  echo "$(vps_agent_text 'Integrated OAuth must require openid and the dedicated resource audience scope.' 'O OAuth integrado deve exigir openid e o escopo de audiência dedicado do recurso.')" >&2
  echo "$(vps_agent_text 'Configured scopes:' 'Escopos configurados:') ${REQUIRED_SCOPES:-<vazio>}" >&2
  exit 1
fi

ORIGIN="$(printf '%s' "$PUBLIC_URL" | sed -E 's#^(https://[^/]+).*$#\1#')"
METADATA_URL="${VPS_AGENT_RESOURCE_METADATA_URL:-$ORIGIN/.well-known/oauth-protected-resource}"
RESOURCE="${VPS_AGENT_OAUTH_RESOURCE:-$PUBLIC_URL}"

echo "$(vps_agent_text 'Checking public health endpoint...' 'Verificando o endpoint público de saúde...')"
curl --fail --silent --show-error "$ORIGIN/healthz" >/tmp/vps-agent-public-health.json
cat /tmp/vps-agent-public-health.json

echo
echo "$(vps_agent_text 'Checking OAuth Protected Resource Metadata...' 'Verificando os metadados OAuth do recurso protegido...')"
curl --fail --silent --show-error "$METADATA_URL" >/tmp/vps-agent-prm.json
cat /tmp/vps-agent-prm.json

VPS_AGENT_LANG="$VPS_AGENT_LANG" python3 - "$RESOURCE" "$ISSUER" "$REQUIRED_SCOPES" <<'PY'
import json, os, sys
resource, issuer, required_raw = sys.argv[1:]
data = json.load(open("/tmp/vps-agent-prm.json"))
assert data.get("resource") == resource, (data, resource)
servers = [x.rstrip("/") for x in data.get("authorization_servers", [])]
assert issuer.rstrip("/") in servers, (data, issuer)
assert "header" in data.get("bearer_methods_supported", []), data
required=set(required_raw.split())
advertised=set(data.get("scopes_supported", []))
assert required <= advertised, (required, advertised)
print("RECURSO OAUTH PROTEGIDO + ESCOPOS: OK" if os.environ.get("VPS_AGENT_LANG")=="pt-BR" else "OAUTH PROTECTED RESOURCE + SCOPES: PASS")
PY

echo
echo "$(vps_agent_text 'Checking integrated Authorization Server discovery...' 'Verificando a descoberta do servidor de autorização integrado...')"
curl --fail --silent --show-error "$ISSUER/.well-known/openid-configuration" >/tmp/vps-agent-oidc-discovery.json
VPS_AGENT_LANG="$VPS_AGENT_LANG" python3 - "$ISSUER" <<'PY'
import json, os, sys
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
print("DESCOBERTA OAUTH + DCR + PKCE + REFRESH: OK" if os.environ.get("VPS_AGENT_LANG")=="pt-BR" else "INTEGRATED OAUTH DISCOVERY + DCR + PKCE + REFRESH: PASS")
PY

echo
echo "$(vps_agent_text 'Checking private audience-bound token introspection...' 'Verificando a introspecção privada de token vinculada à audiência...')"
docker compose exec -T gateway sh -c '
  curl --fail --silent --show-error     --request POST     --url "$VPS_AGENT_INTEGRATED_INTROSPECTION_URL"     --user "$VPS_AGENT_INTEGRATED_INTROSPECTION_CLIENT_ID:$VPS_AGENT_INTEGRATED_INTROSPECTION_CLIENT_SECRET"     --header "Host: $VPS_AGENT_INTEGRATED_INTROSPECTION_HOST"     --header "X-Forwarded-Proto: https"     --header "Content-Type: application/x-www-form-urlencoded"     --data "token=deliberately-invalid-verification-probe"
' >/tmp/vps-agent-introspection-probe.json
VPS_AGENT_LANG="$VPS_AGENT_LANG" python3 - <<'PY'
import json, os
x=json.load(open("/tmp/vps-agent-introspection-probe.json"))
assert x.get("active") is False, x
print("INTROSPECÇÃO PRIVADA VINCULADA À AUDIÊNCIA: OK" if os.environ.get("VPS_AGENT_LANG")=="pt-BR" else "PRIVATE AUDIENCE-BOUND INTROSPECTION: PASS")
PY

echo
echo "$(vps_agent_text 'Checking that unauthenticated MCP access fails closed with OAuth discovery challenge...' 'Verificando se o acesso MCP sem autenticação é negado com o desafio OAuth correto...')"
code="$(curl --silent --show-error \
  --dump-header /tmp/vps-agent-public-unauth-headers.txt \
  --output /tmp/vps-agent-public-unauth.txt \
  --write-out '%{http_code}' \
  "$PUBLIC_URL" || true)"
if [ "$code" != "401" ]; then
  echo "$(vps_agent_msg error.expected_401 "code=$code")" >&2
  cat /tmp/vps-agent-public-unauth.txt >&2 || true
  exit 1
fi
VPS_AGENT_LANG="$VPS_AGENT_LANG" python3 - "$METADATA_URL" <<'PY'
import os, sys
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
print("NEGAÇÃO MCP SEM AUTENTICAÇÃO + DESAFIO OAUTH: OK" if os.environ.get("VPS_AGENT_LANG")=="pt-BR" else "UNAUTHENTICATED MCP DENIAL + OAUTH CHALLENGE: PASS")
PY

if [ -n "${VPS_AGENT_TEST_ACCESS_TOKEN:-}" ]; then
  echo
  echo "$(vps_agent_text 'Testing a real OAuth access token against system.info...' 'Testando um token OAuth real contra system.info...')"
  docker compose exec -T gateway /usr/local/bin/vps-agent-mcp-call \
    --endpoint "$PUBLIC_URL" \
    --token "$VPS_AGENT_TEST_ACCESS_TOKEN" \
    --tool system.info \
    --args '{}' >/tmp/vps-agent-public-system-info.json
  cat /tmp/vps-agent-public-system-info.json
  VPS_AGENT_LANG="$VPS_AGENT_LANG" python3 - <<'PY'
import json, os
x=json.load(open("/tmp/vps-agent-public-system-info.json"))
assert x["is_error"] is False, x
print("CHAMADA MCP OAUTH PÚBLICA: OK" if os.environ.get("VPS_AGENT_LANG")=="pt-BR" else "PUBLIC OAUTH MCP CALL: PASS")
PY
else
  echo
  echo "$(vps_agent_text 'No test access token is set; the token-authenticated public call was not executed.' 'Nenhum token de teste foi definido; a chamada pública autenticada por token não foi executada.')"
  echo "$(vps_agent_text 'OAuth discovery, DCR/PKCE, private introspection and fail-closed challenge behavior are valid.' 'Descoberta OAuth, DCR/PKCE, introspecção privada e negação segura estão válidos.')"
  echo "$(vps_agent_text 'The real ChatGPT authorization flow remains the final client-side gate.' 'O fluxo real de autorização pelo ChatGPT continua sendo a etapa final do lado do cliente.')"
fi
