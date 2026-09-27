#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

usage() {
  cat <<'EOF'
usage: bash scripts/setup-integrated-auth.sh --domain mcp.example.com [options]

Sets up the self-hosted OAuth/OIDC path used by ChatGPT. ZITADEL runs inside
this package; no Auth0/Okta/Entra account and no OpenAI tunnel are required.

Options:
  --domain DOMAIN          Public DNS name for both MCP and integrated auth.
  --operator-email EMAIL   Login email for the VPS operator.
  --operator-username NAME Login username (default: vps-operator).
  --edge-network NETWORK   Existing Traefik Docker network.
  --certresolver NAME      Existing Traefik ACME resolver (default: autodetect).
  --bundled-proxy          Run the package Traefik instead of an existing one.
  -h, --help               Show this help.

The script asks for the operator password without echoing it. For automation,
set VPS_AGENT_OPERATOR_PASSWORD in the environment instead of putting a
password on the command line.
EOF
}

DOMAIN=""
OPERATOR_EMAIL=""
OPERATOR_USERNAME="vps-operator"
EDGE_NETWORK=""
CERTRESOLVER=""
BUNDLED_PROXY=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --domain) DOMAIN="${2:-}"; shift 2 ;;
    --operator-email) OPERATOR_EMAIL="${2:-}"; shift 2 ;;
    --operator-username) OPERATOR_USERNAME="${2:-}"; shift 2 ;;
    --edge-network) EDGE_NETWORK="${2:-}"; shift 2 ;;
    --certresolver) CERTRESOLVER="${2:-}"; shift 2 ;;
    --bundled-proxy) BUNDLED_PROXY=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "ERROR: unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [ ! -f .env ] || [ ! -f config/policy.yaml ]; then
  echo "ERROR: run scripts/init.sh and the local verification first." >&2
  exit 1
fi
if [ -z "$DOMAIN" ]; then
  echo "ERROR: --domain is required." >&2
  exit 2
fi
if [[ "$DOMAIN" == *"://"* || "$DOMAIN" == */* || "$DOMAIN" == *" "* ]]; then
  echo "ERROR: --domain must be a hostname only, for example mcp.example.com." >&2
  exit 2
fi
for cmd in docker curl openssl python3; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "ERROR: $cmd is required." >&2; exit 1; }
done
docker compose version >/dev/null

if [ -z "$OPERATOR_EMAIL" ]; then
  if [ -t 0 ]; then
    printf 'Operator login email: '
    read -r OPERATOR_EMAIL
  else
    echo "ERROR: --operator-email is required in non-interactive mode." >&2
    exit 2
  fi
fi
if [[ "$OPERATOR_EMAIL" != *@*.* ]]; then
  echo "ERROR: operator email does not look valid: $OPERATOR_EMAIL" >&2
  exit 2
fi
if [[ ! "$OPERATOR_USERNAME" =~ ^[A-Za-z0-9._-]+$ ]]; then
  echo "ERROR: operator username may contain only letters, numbers, dot, underscore and hyphen." >&2
  exit 2
fi

detect_traefik() {
  local ids id count=0 chosen=""
  ids="$(docker ps -q)"
  [ -n "$ids" ] || return 1
  for id in $ids; do
    image="$(docker inspect -f '{{.Config.Image}}' "$id" 2>/dev/null || true)"
    case "${image,,}" in
      *traefik*) chosen="$id"; count=$((count+1)) ;;
    esac
  done
  if [ "$count" -eq 1 ]; then
    printf '%s' "$chosen"
    return 0
  fi
  if [ "$count" -gt 1 ]; then
    echo "ERROR: multiple running Traefik containers found; pass --edge-network explicitly." >&2
  fi
  return 1
}

TRAEFIK_ID=""
if [ "$BUNDLED_PROXY" -eq 1 ]; then
  EDGE_NETWORK="${EDGE_NETWORK:-mcp-vps-agent-edge}"
  CERTRESOLVER="${CERTRESOLVER:-letsencrypt}"
  docker network inspect "$EDGE_NETWORK" >/dev/null 2>&1 || docker network create "$EDGE_NETWORK" >/dev/null
else
  TRAEFIK_ID="$(detect_traefik || true)"
  if [ -z "$TRAEFIK_ID" ]; then
    if ! ss -ltnH 2>/dev/null | awk '{print $4}' | grep -Eq '(^|:)(80|443)
    mapfile -t NETWORKS < <(docker inspect "$TRAEFIK_ID" --format '{{range $name, $cfg := .NetworkSettings.Networks}}{{$name}}{{"\n"}}{{end}}' | grep -Ev '^(bridge|host|none)$' | sed '/^$/d')
    if [ "${#NETWORKS[@]}" -ne 1 ]; then
      echo "ERROR: Traefik uses ${#NETWORKS[@]} candidate networks: ${NETWORKS[*]:-none}" >&2
      echo "Pass --edge-network NETWORK." >&2
      exit 1
    fi
    EDGE_NETWORK="${NETWORKS[0]}"
  fi
  if [ "$BUNDLED_PROXY" -eq 0 ]; then
    docker network inspect "$EDGE_NETWORK" >/dev/null 2>&1 || { echo "ERROR: Docker network not found: $EDGE_NETWORK" >&2; exit 1; }
  fi

  if [ "$BUNDLED_PROXY" -eq 0 ] && [ -z "$CERTRESOLVER" ]; then
    CERTRESOLVER="$(docker inspect "$TRAEFIK_ID" --format '{{json .Config.Labels}}' | python3 -c '
import json,sys
labels=json.load(sys.stdin) or {}
vals=sorted({v for k,v in labels.items() if k.endswith(".tls.certresolver") and v})
print(vals[0] if len(vals)==1 else "")
')"
    CERTRESOLVER="${CERTRESOLVER:-letsencrypt}"
  fi
fi

upsert_env() {
  python3 - .env "$1" "$2" <<'PY'
import pathlib, shlex, sys
path=pathlib.Path(sys.argv[1]); key=sys.argv[2]; value=sys.argv[3]
lines=path.read_text().splitlines()
replacement=f"{key}={shlex.quote(value)}"
out=[]; found=False
for line in lines:
    if line.startswith(key+"="):
        out.append(replacement); found=True
    else:
        out.append(line)
if not found:
    out.append(replacement)
path.write_text("\n".join(out)+"\n")
PY
}

env_value() {
  python3 - .env "$1" <<'PY'
import pathlib, shlex, sys
key=sys.argv[2]
for line in pathlib.Path(sys.argv[1]).read_text().splitlines():
    if line.startswith(key+"="):
        raw=line.split("=",1)[1]
        try:
            print(shlex.split(raw)[0] if raw else "")
        except Exception:
            print(raw.strip("'\""))
        break
PY
}

random_hex() { openssl rand -hex "$1"; }

MASTERKEY="$(env_value ZITADEL_MASTERKEY || true)"
POSTGRES_PASSWORD="$(env_value ZITADEL_POSTGRES_PASSWORD || true)"
BOOTSTRAP_ADMIN_PASSWORD="$(env_value ZITADEL_BOOTSTRAP_ADMIN_PASSWORD || true)"
[ -n "$MASTERKEY" ] || MASTERKEY="$(random_hex 16)"
[ -n "$POSTGRES_PASSWORD" ] || POSTGRES_PASSWORD="$(random_hex 24)"
[ -n "$BOOTSTRAP_ADMIN_PASSWORD" ] || BOOTSTRAP_ADMIN_PASSWORD="$(random_hex 24)"

upsert_env VPS_AGENT_DOMAIN "$DOMAIN"
upsert_env VPS_AGENT_PUBLIC_URL "https://$DOMAIN/mcp"
upsert_env VPS_AGENT_AUTH_MODE integrated
upsert_env VPS_AGENT_OIDC_ISSUER "https://$DOMAIN"
upsert_env VPS_AGENT_OIDC_AUDIENCE ""
upsert_env VPS_AGENT_OAUTH_RESOURCE "https://$DOMAIN/mcp"
upsert_env VPS_AGENT_RESOURCE_METADATA_URL "https://$DOMAIN/.well-known/oauth-protected-resource"
upsert_env VPS_AGENT_REQUIRED_SCOPES openid
upsert_env VPS_AGENT_EDGE_NETWORK "$EDGE_NETWORK"
upsert_env VPS_AGENT_TRAEFIK_CERTRESOLVER "$CERTRESOLVER"
upsert_env VPS_AGENT_BUNDLED_PROXY "$BUNDLED_PROXY"
upsert_env ZITADEL_VERSION v4.19.1
upsert_env ZITADEL_MASTERKEY "$MASTERKEY"
upsert_env ZITADEL_POSTGRES_DB zitadel
upsert_env ZITADEL_POSTGRES_USER zitadel
upsert_env ZITADEL_POSTGRES_PASSWORD "$POSTGRES_PASSWORD"
upsert_env ZITADEL_BOOTSTRAP_ADMIN_PASSWORD "$BOOTSTRAP_ADMIN_PASSWORD"
upsert_env ZITADEL_BOOTSTRAP_PAT_EXPIRATION "$(date -u -d '+2 hours' +%Y-%m-%dT%H:%M:%SZ)"
upsert_env ZITADEL_LOGIN_PAT_EXPIRATION 2036-01-01T00:00:00Z
upsert_env VPS_AGENT_LETSENCRYPT_EMAIL "$OPERATOR_EMAIL"
chmod 600 .env

compose=(docker compose -f compose.yaml -f compose.integrated-auth.yaml)
if [ "$BUNDLED_PROXY" -eq 1 ]; then
  compose+=(-f compose.integrated-auth.proxy.yaml)
fi

echo
echo "Integrated auth plan:"
echo "  MCP + issuer: https://$DOMAIN"
echo "  edge network: $EDGE_NETWORK"
echo "  cert resolver: $CERTRESOLVER"
echo "  proxy: $([ "$BUNDLED_PROXY" -eq 1 ] && echo bundled || echo existing)"
echo

if [ "$BUNDLED_PROXY" -eq 1 ]; then
  "${compose[@]}" up -d edge-proxy
fi
"${compose[@]}" up -d zitadel-postgres zitadel-api zitadel-login

echo "Waiting for public OIDC discovery and TLS..."
OIDC_URL="https://$DOMAIN/.well-known/openid-configuration"
for _ in $(seq 1 90); do
  if curl -fsS --max-time 5 "$OIDC_URL" >/tmp/vps-agent-oidc.json 2>/dev/null; then
    break
  fi
  sleep 2
done
if ! test -s /tmp/vps-agent-oidc.json; then
  echo "ERROR: OIDC discovery did not become reachable at $OIDC_URL" >&2
  echo "Check DNS, ports 80/443 and Traefik, then rerun this script." >&2
  "${compose[@]}" ps >&2 || true
  exit 1
fi
python3 - "$DOMAIN" <<'PY'
import json,sys
domain=sys.argv[1]
x=json.load(open("/tmp/vps-agent-oidc.json"))
assert x.get("issuer","").rstrip("/") == f"https://{domain}", x.get("issuer")
print("OIDC DISCOVERY: PASS")
PY

BOOTSTRAP_PAT="$(docker run --rm -v mcp-vps-agent_zitadel-bootstrap:/zitadel/bootstrap:ro alpine:3.22 cat /zitadel/bootstrap/bootstrap-admin.pat 2>/dev/null || true)"
if [ -z "$BOOTSTRAP_PAT" ]; then
  echo "ERROR: ZITADEL bootstrap PAT was not produced." >&2
  exit 1
fi

echo "Enabling MCP-compatible Dynamic Client Registration..."
curl -fsS --request PUT   --url "https://$DOMAIN/v2/settings/security"   --header "Authorization: Bearer $BOOTSTRAP_PAT"   --header 'Content-Type: application/json'   --data '{"dynamicClientRegistration":{"enabled":true,"allowUnauthenticated":true}}'   >/tmp/vps-agent-dcr-settings.json

CURRENT_SUBJECT="$(env_value VPS_AGENT_SUBJECT || true)"
MARKER="state/integrated-auth.json"
if [ -s "$MARKER" ] && [ -n "$CURRENT_SUBJECT" ]; then
  echo "Existing integrated operator identity found; preserving subject $CURRENT_SUBJECT."
  OPERATOR_ID="$CURRENT_SUBJECT"
else
  PASSWORD="${VPS_AGENT_OPERATOR_PASSWORD:-}"
  if [ -z "$PASSWORD" ]; then
    if [ ! -t 0 ]; then
      echo "ERROR: set VPS_AGENT_OPERATOR_PASSWORD for non-interactive bootstrap." >&2
      exit 1
    fi
    while :; do
      printf 'Create operator password: ' >&2
      read -r -s PASSWORD
      echo >&2
      printf 'Repeat operator password: ' >&2
      read -r -s PASSWORD2
      echo >&2
      if [ "$PASSWORD" != "$PASSWORD2" ]; then
        echo "Passwords do not match." >&2
        continue
      fi
      if [ "${#PASSWORD}" -lt 12 ]; then
        echo "Use at least 12 characters." >&2
        continue
      fi
      break
    done
  fi

  OPERATOR_ID="$(python3 -c 'import uuid; print(uuid.uuid4())')"
  REQUEST_FILE="$(mktemp)"
  chmod 600 "$REQUEST_FILE"
  trap 'rm -f "$REQUEST_FILE"' EXIT
  python3 - "$REQUEST_FILE" "$OPERATOR_ID" "$OPERATOR_USERNAME" "$OPERATOR_EMAIL" "$PASSWORD" <<'PY'
import json,sys
path,user_id,username,email,password=sys.argv[1:]
payload={
  "userId": user_id,
  "username": username,
  "profile": {
    "givenName": "VPS",
    "familyName": "Operator",
    "displayName": "VPS Operator",
    "preferredLanguage": "en"
  },
  "email": {"email": email, "isVerified": True},
  "password": {"password": password, "changeRequired": False}
}
open(path,"w").write(json.dumps(payload))
PY
  unset PASSWORD PASSWORD2 2>/dev/null || true

  echo "Creating the dedicated non-admin VPS operator identity..."
  HTTP_CODE="$(curl -sS --output /tmp/vps-agent-create-operator.json --write-out '%{http_code}'     --request POST     --url "https://$DOMAIN/v2/users/human"     --header "Authorization: Bearer $BOOTSTRAP_PAT"     --header 'Content-Type: application/json'     --data-binary "@$REQUEST_FILE")"
  if [ "$HTTP_CODE" != "200" ] && [ "$HTTP_CODE" != "201" ]; then
    echo "ERROR: operator creation returned HTTP $HTTP_CODE" >&2
    cat /tmp/vps-agent-create-operator.json >&2
    exit 1
  fi
  rm -f "$REQUEST_FILE"
  trap - EXIT
  upsert_env VPS_AGENT_SUBJECT "$OPERATOR_ID"
  mkdir -p state
  chmod 700 state
  python3 - "$MARKER" "$DOMAIN" "$OPERATOR_ID" "$OPERATOR_USERNAME" "$OPERATOR_EMAIL" <<'PY'
import json,sys,datetime
path,domain,sub,username,email=sys.argv[1:]
json.dump({
  "domain":domain,
  "subject":sub,
  "username":username,
  "email":email,
  "created_at":datetime.datetime.now(datetime.timezone.utc).isoformat()
},open(path,"w"),indent=2)
open(path,"a").write("\n")
PY
  chmod 600 "$MARKER"
fi

echo "Confirming DCR advertisement..."
curl -fsS "$OIDC_URL" >/tmp/vps-agent-oidc.json
python3 - <<'PY'
import json
x=json.load(open("/tmp/vps-agent-oidc.json"))
endpoint=x.get("registration_endpoint","")
assert endpoint, x
assert endpoint.startswith("https://"), endpoint
print("DYNAMIC CLIENT REGISTRATION: PASS")
PY

# The bootstrap machine PAT is no longer needed after DCR and operator creation.
docker run --rm -v mcp-vps-agent_zitadel-bootstrap:/zitadel/bootstrap alpine:3.22 rm -f /zitadel/bootstrap/bootstrap-admin.pat >/dev/null 2>&1 || true

echo "Switching Gateway and Broker to the integrated identity..."
"${compose[@]}" up -d --build --force-recreate broker gateway

echo "Waiting for the public MCP resource..."
for _ in $(seq 1 60); do
  code="$(curl -sS --max-time 5 -o /tmp/vps-agent-public-unauth.txt -w '%{http_code}' "https://$DOMAIN/mcp" 2>/dev/null || true)"
  [ "$code" = "401" ] && break
  sleep 2
done

bash scripts/verify-public.sh

cat <<EOF

INTEGRATED AUTH: READY

MCP endpoint:
  https://$DOMAIN/mcp

OAuth/OIDC issuer:
  https://$DOMAIN

Operator login:
  $OPERATOR_EMAIL

No third-party identity provider or OpenAI tunnel is required.
The password you entered is not stored by this installer.

Next:
  bash scripts/connect-chatgpt.sh

EOF
; then
      echo "No existing Traefik found and ports 80/443 are free; using the bundled Traefik."
      BUNDLED_PROXY=1
      EDGE_NETWORK="${EDGE_NETWORK:-mcp-vps-agent-edge}"
      CERTRESOLVER="${CERTRESOLVER:-letsencrypt}"
      docker network inspect "$EDGE_NETWORK" >/dev/null 2>&1 || docker network create "$EDGE_NETWORK" >/dev/null
    else
      cat >&2 <<'EOF'
ERROR: no reusable Traefik was found, but ports 80/443 are already in use.
The automatic integrated-auth path will not replace an existing web server.
Use an existing Traefik with --edge-network, or free 80/443 and rerun.
EOF
      exit 1
    fi
  fi

  if [ "$BUNDLED_PROXY" -eq 0 ] && [ -z "$EDGE_NETWORK" ]; then
    mapfile -t NETWORKS < <(docker inspect "$TRAEFIK_ID" --format '{{range $name, $cfg := .NetworkSettings.Networks}}{{$name}}{{"\n"}}{{end}}' | grep -Ev '^(bridge|host|none)$' | sed '/^$/d')
    if [ "${#NETWORKS[@]}" -ne 1 ]; then
      echo "ERROR: Traefik uses ${#NETWORKS[@]} candidate networks: ${NETWORKS[*]:-none}" >&2
      echo "Pass --edge-network NETWORK." >&2
      exit 1
    fi
    EDGE_NETWORK="${NETWORKS[0]}"
  fi
  docker network inspect "$EDGE_NETWORK" >/dev/null 2>&1 || { echo "ERROR: Docker network not found: $EDGE_NETWORK" >&2; exit 1; }

  if [ -z "$CERTRESOLVER" ]; then
    CERTRESOLVER="$(docker inspect "$TRAEFIK_ID" --format '{{json .Config.Labels}}' | python3 -c '
import json,sys
labels=json.load(sys.stdin) or {}
vals=sorted({v for k,v in labels.items() if k.endswith(".tls.certresolver") and v})
print(vals[0] if len(vals)==1 else "")
')"
    CERTRESOLVER="${CERTRESOLVER:-letsencrypt}"
  fi
fi

upsert_env() {
  python3 - .env "$1" "$2" <<'PY'
import pathlib, shlex, sys
path=pathlib.Path(sys.argv[1]); key=sys.argv[2]; value=sys.argv[3]
lines=path.read_text().splitlines()
replacement=f"{key}={shlex.quote(value)}"
out=[]; found=False
for line in lines:
    if line.startswith(key+"="):
        out.append(replacement); found=True
    else:
        out.append(line)
if not found:
    out.append(replacement)
path.write_text("\n".join(out)+"\n")
PY
}

env_value() {
  python3 - .env "$1" <<'PY'
import pathlib, shlex, sys
key=sys.argv[2]
for line in pathlib.Path(sys.argv[1]).read_text().splitlines():
    if line.startswith(key+"="):
        raw=line.split("=",1)[1]
        try:
            print(shlex.split(raw)[0] if raw else "")
        except Exception:
            print(raw.strip("'\""))
        break
PY
}

random_hex() { openssl rand -hex "$1"; }

MASTERKEY="$(env_value ZITADEL_MASTERKEY || true)"
POSTGRES_PASSWORD="$(env_value ZITADEL_POSTGRES_PASSWORD || true)"
BOOTSTRAP_ADMIN_PASSWORD="$(env_value ZITADEL_BOOTSTRAP_ADMIN_PASSWORD || true)"
[ -n "$MASTERKEY" ] || MASTERKEY="$(random_hex 16)"
[ -n "$POSTGRES_PASSWORD" ] || POSTGRES_PASSWORD="$(random_hex 24)"
[ -n "$BOOTSTRAP_ADMIN_PASSWORD" ] || BOOTSTRAP_ADMIN_PASSWORD="$(random_hex 24)"

upsert_env VPS_AGENT_DOMAIN "$DOMAIN"
upsert_env VPS_AGENT_PUBLIC_URL "https://$DOMAIN/mcp"
upsert_env VPS_AGENT_AUTH_MODE integrated
upsert_env VPS_AGENT_OIDC_ISSUER "https://$DOMAIN"
upsert_env VPS_AGENT_OIDC_AUDIENCE ""
upsert_env VPS_AGENT_OAUTH_RESOURCE "https://$DOMAIN/mcp"
upsert_env VPS_AGENT_RESOURCE_METADATA_URL "https://$DOMAIN/.well-known/oauth-protected-resource"
upsert_env VPS_AGENT_REQUIRED_SCOPES openid
upsert_env VPS_AGENT_EDGE_NETWORK "$EDGE_NETWORK"
upsert_env VPS_AGENT_TRAEFIK_CERTRESOLVER "$CERTRESOLVER"
upsert_env VPS_AGENT_BUNDLED_PROXY "$BUNDLED_PROXY"
upsert_env ZITADEL_VERSION v4.19.1
upsert_env ZITADEL_MASTERKEY "$MASTERKEY"
upsert_env ZITADEL_POSTGRES_DB zitadel
upsert_env ZITADEL_POSTGRES_USER zitadel
upsert_env ZITADEL_POSTGRES_PASSWORD "$POSTGRES_PASSWORD"
upsert_env ZITADEL_BOOTSTRAP_ADMIN_PASSWORD "$BOOTSTRAP_ADMIN_PASSWORD"
upsert_env ZITADEL_BOOTSTRAP_PAT_EXPIRATION "$(date -u -d '+2 hours' +%Y-%m-%dT%H:%M:%SZ)"
upsert_env ZITADEL_LOGIN_PAT_EXPIRATION 2036-01-01T00:00:00Z
upsert_env VPS_AGENT_LETSENCRYPT_EMAIL "$OPERATOR_EMAIL"
chmod 600 .env

compose=(docker compose -f compose.yaml -f compose.integrated-auth.yaml)
if [ "$BUNDLED_PROXY" -eq 1 ]; then
  compose+=(-f compose.integrated-auth.proxy.yaml)
fi

echo
echo "Integrated auth plan:"
echo "  MCP + issuer: https://$DOMAIN"
echo "  edge network: $EDGE_NETWORK"
echo "  cert resolver: $CERTRESOLVER"
echo "  proxy: $([ "$BUNDLED_PROXY" -eq 1 ] && echo bundled || echo existing)"
echo

if [ "$BUNDLED_PROXY" -eq 1 ]; then
  "${compose[@]}" up -d edge-proxy
fi
"${compose[@]}" up -d zitadel-postgres zitadel-api zitadel-login

echo "Waiting for public OIDC discovery and TLS..."
OIDC_URL="https://$DOMAIN/.well-known/openid-configuration"
for _ in $(seq 1 90); do
  if curl -fsS --max-time 5 "$OIDC_URL" >/tmp/vps-agent-oidc.json 2>/dev/null; then
    break
  fi
  sleep 2
done
if ! test -s /tmp/vps-agent-oidc.json; then
  echo "ERROR: OIDC discovery did not become reachable at $OIDC_URL" >&2
  echo "Check DNS, ports 80/443 and Traefik, then rerun this script." >&2
  "${compose[@]}" ps >&2 || true
  exit 1
fi
python3 - "$DOMAIN" <<'PY'
import json,sys
domain=sys.argv[1]
x=json.load(open("/tmp/vps-agent-oidc.json"))
assert x.get("issuer","").rstrip("/") == f"https://{domain}", x.get("issuer")
print("OIDC DISCOVERY: PASS")
PY

BOOTSTRAP_PAT="$("${compose[@]}" exec -T zitadel-api cat /zitadel/bootstrap/bootstrap-admin.pat 2>/dev/null || true)"
if [ -z "$BOOTSTRAP_PAT" ]; then
  echo "ERROR: ZITADEL bootstrap PAT was not produced." >&2
  exit 1
fi

echo "Enabling MCP-compatible Dynamic Client Registration..."
curl -fsS --request PUT   --url "https://$DOMAIN/v2/settings/security"   --header "Authorization: Bearer $BOOTSTRAP_PAT"   --header 'Content-Type: application/json'   --data '{"dynamicClientRegistration":{"enabled":true,"allowUnauthenticated":true}}'   >/tmp/vps-agent-dcr-settings.json

CURRENT_SUBJECT="$(env_value VPS_AGENT_SUBJECT || true)"
MARKER="state/integrated-auth.json"
if [ -s "$MARKER" ] && [ -n "$CURRENT_SUBJECT" ]; then
  echo "Existing integrated operator identity found; preserving subject $CURRENT_SUBJECT."
  OPERATOR_ID="$CURRENT_SUBJECT"
else
  PASSWORD="${VPS_AGENT_OPERATOR_PASSWORD:-}"
  if [ -z "$PASSWORD" ]; then
    if [ ! -t 0 ]; then
      echo "ERROR: set VPS_AGENT_OPERATOR_PASSWORD for non-interactive bootstrap." >&2
      exit 1
    fi
    while :; do
      printf 'Create operator password: ' >&2
      read -r -s PASSWORD
      echo >&2
      printf 'Repeat operator password: ' >&2
      read -r -s PASSWORD2
      echo >&2
      if [ "$PASSWORD" != "$PASSWORD2" ]; then
        echo "Passwords do not match." >&2
        continue
      fi
      if [ "${#PASSWORD}" -lt 12 ]; then
        echo "Use at least 12 characters." >&2
        continue
      fi
      break
    done
  fi

  OPERATOR_ID="$(python3 -c 'import uuid; print(uuid.uuid4())')"
  REQUEST_FILE="$(mktemp)"
  chmod 600 "$REQUEST_FILE"
  trap 'rm -f "$REQUEST_FILE"' EXIT
  python3 - "$REQUEST_FILE" "$OPERATOR_ID" "$OPERATOR_USERNAME" "$OPERATOR_EMAIL" "$PASSWORD" <<'PY'
import json,sys
path,user_id,username,email,password=sys.argv[1:]
payload={
  "userId": user_id,
  "username": username,
  "profile": {
    "givenName": "VPS",
    "familyName": "Operator",
    "displayName": "VPS Operator",
    "preferredLanguage": "en"
  },
  "email": {"email": email, "isVerified": True},
  "password": {"password": password, "changeRequired": False}
}
open(path,"w").write(json.dumps(payload))
PY
  unset PASSWORD PASSWORD2 2>/dev/null || true

  echo "Creating the dedicated non-admin VPS operator identity..."
  HTTP_CODE="$(curl -sS --output /tmp/vps-agent-create-operator.json --write-out '%{http_code}'     --request POST     --url "https://$DOMAIN/v2/users/human"     --header "Authorization: Bearer $BOOTSTRAP_PAT"     --header 'Content-Type: application/json'     --data-binary "@$REQUEST_FILE")"
  if [ "$HTTP_CODE" != "200" ] && [ "$HTTP_CODE" != "201" ]; then
    echo "ERROR: operator creation returned HTTP $HTTP_CODE" >&2
    cat /tmp/vps-agent-create-operator.json >&2
    exit 1
  fi
  rm -f "$REQUEST_FILE"
  trap - EXIT
  upsert_env VPS_AGENT_SUBJECT "$OPERATOR_ID"
  mkdir -p state
  chmod 700 state
  python3 - "$MARKER" "$DOMAIN" "$OPERATOR_ID" "$OPERATOR_USERNAME" "$OPERATOR_EMAIL" <<'PY'
import json,sys,datetime
path,domain,sub,username,email=sys.argv[1:]
json.dump({
  "domain":domain,
  "subject":sub,
  "username":username,
  "email":email,
  "created_at":datetime.datetime.now(datetime.timezone.utc).isoformat()
},open(path,"w"),indent=2)
open(path,"a").write("\n")
PY
  chmod 600 "$MARKER"
fi

echo "Confirming DCR advertisement..."
curl -fsS "$OIDC_URL" >/tmp/vps-agent-oidc.json
python3 - <<'PY'
import json
x=json.load(open("/tmp/vps-agent-oidc.json"))
endpoint=x.get("registration_endpoint","")
assert endpoint, x
assert endpoint.startswith("https://"), endpoint
print("DYNAMIC CLIENT REGISTRATION: PASS")
PY

# The bootstrap machine PAT is no longer needed after DCR and operator creation.
"${compose[@]}" exec -T zitadel-api rm -f /zitadel/bootstrap/bootstrap-admin.pat >/dev/null 2>&1 || true

echo "Switching Gateway and Broker to the integrated identity..."
"${compose[@]}" up -d --build --force-recreate broker gateway

echo "Waiting for the public MCP resource..."
for _ in $(seq 1 60); do
  code="$(curl -sS --max-time 5 -o /tmp/vps-agent-public-unauth.txt -w '%{http_code}' "https://$DOMAIN/mcp" 2>/dev/null || true)"
  [ "$code" = "401" ] && break
  sleep 2
done

bash scripts/verify-public.sh

cat <<EOF

INTEGRATED AUTH: READY

MCP endpoint:
  https://$DOMAIN/mcp

OAuth/OIDC issuer:
  https://$DOMAIN

Operator login:
  $OPERATOR_EMAIL

No third-party identity provider or OpenAI tunnel is required.
The password you entered is not stored by this installer.

Next:
  bash scripts/connect-chatgpt.sh

EOF
