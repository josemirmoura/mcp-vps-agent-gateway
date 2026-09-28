#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

usage() {
  cat <<'EOF'
usage: bash scripts/setup-integrated-auth.sh [--domain mcp.example.com] [options]

Configure the supported public ChatGPT path: self-hosted OAuth/OIDC with
ZITADEL, PostgreSQL and the existing package Gateway/Broker.

Options:
  --domain DOMAIN          Public DNS name for MCP + OAuth.
  --operator-email EMAIL   Login email for the VPS operator.
  --operator-username NAME Login username (default: vps-operator).
  --edge-network NETWORK   Existing Traefik Docker network.
  --certresolver NAME      Existing Traefik ACME resolver.
  --bundled-proxy          Force the package Traefik instead of reusing one.
  -h, --help               Show this help.

The operator password is requested without terminal echo. For CI/automation,
VPS_AGENT_OPERATOR_PASSWORD may be supplied in the environment; do not put a
real password on a shared command line.
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
    --domain)
      [ "$#" -ge 2 ] || { echo "ERROR: --domain needs a value." >&2; exit 2; }
      DOMAIN="$2"; shift 2 ;;
    --operator-email)
      [ "$#" -ge 2 ] || { echo "ERROR: --operator-email needs a value." >&2; exit 2; }
      OPERATOR_EMAIL="$2"; shift 2 ;;
    --operator-username)
      [ "$#" -ge 2 ] || { echo "ERROR: --operator-username needs a value." >&2; exit 2; }
      OPERATOR_USERNAME="$2"; shift 2 ;;
    --edge-network)
      [ "$#" -ge 2 ] || { echo "ERROR: --edge-network needs a value." >&2; exit 2; }
      EDGE_NETWORK="$2"; shift 2 ;;
    --certresolver)
      [ "$#" -ge 2 ] || { echo "ERROR: --certresolver needs a value." >&2; exit 2; }
      CERTRESOLVER="$2"; shift 2 ;;
    --bundled-proxy) BUNDLED_PROXY=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "ERROR: unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [ ! -f .env ] || [ ! -f config/policy.yaml ]; then
  echo "ERROR: run scripts/init.sh and bash scripts/verify.sh first." >&2
  exit 1
fi

if [ -z "$DOMAIN" ]; then
  if [ -t 0 ]; then
    printf 'Public MCP domain (for example mcp.example.com): '
    read -r DOMAIN
  else
    echo "ERROR: --domain is required in non-interactive mode." >&2
    exit 2
  fi
fi
if [[ "$DOMAIN" == *"://"* || "$DOMAIN" == */* || "$DOMAIN" == *" "* || "$DOMAIN" == .* || "$DOMAIN" == *. ]]; then
  echo "ERROR: --domain must be a hostname only, for example mcp.example.com." >&2
  exit 2
fi
if [[ ! "$DOMAIN" =~ ^[A-Za-z0-9.-]+$ ]]; then
  echo "ERROR: --domain contains unsupported characters." >&2
  exit 2
fi

for cmd in docker curl openssl python3 ss getent; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "ERROR: required command not found: $cmd" >&2; exit 1; }
done
docker compose version >/dev/null

upsert_env() {
  python3 - .env "$1" "$2" <<'PY'
import pathlib, shlex, sys
path=pathlib.Path(sys.argv[1])
key=sys.argv[2]
value=sys.argv[3]
lines=path.read_text().splitlines()
replacement=f"{key}={shlex.quote(value)}"
out=[]
found=False
for line in lines:
    if line.startswith(key+"="):
        out.append(replacement)
        found=True
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
            values=shlex.split(raw)
            print(values[0] if values else "")
        except Exception:
            print(raw.strip("'\""))
        break
PY
}

marker_value() {
  python3 - "$1" "$2" <<'PY'
import json, pathlib, sys
path=pathlib.Path(sys.argv[1])
key=sys.argv[2]
if path.is_file():
    try:
        print(json.loads(path.read_text()).get(key,""))
    except Exception:
        pass
PY
}

if ! getent ahosts "$DOMAIN" >/dev/null 2>&1; then
  echo "ERROR: DNS for $DOMAIN does not resolve yet." >&2
  echo "Create the A/AAAA record pointing to this VPS, wait for DNS propagation, and rerun." >&2
  exit 1
fi

MARKER="state/integrated-auth.json"
CURRENT_SUBJECT="$(env_value VPS_AGENT_SUBJECT || true)"
MARKER_DOMAIN="$(marker_value "$MARKER" domain || true)"
if [ -s "$MARKER" ] && [ -n "$MARKER_DOMAIN" ] && [ "$MARKER_DOMAIN" != "$DOMAIN" ]; then
  echo "ERROR: integrated auth was already initialized for $MARKER_DOMAIN." >&2
  echo "Automatic issuer/domain migration is intentionally not supported." >&2
  exit 1
fi

NEEDS_OPERATOR=1
if [ -s "$MARKER" ] && [ -n "$CURRENT_SUBJECT" ]; then
  NEEDS_OPERATOR=0
fi

if [ "$NEEDS_OPERATOR" -eq 1 ]; then
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
else
  OPERATOR_EMAIL="$(marker_value "$MARKER" email || true)"
  OPERATOR_USERNAME="$(marker_value "$MARKER" username || true)"
  echo "Existing integrated operator identity detected; bootstrap will be reused."
fi

mapfile -t TRAEFIK_IDS < <(
  docker ps --format '{{.ID}} {{.Image}}' |
    awk 'tolower($0) ~ /traefik/ {print $1}'
)

TRAEFIK_ID=""
if [ "$BUNDLED_PROXY" -eq 1 ]; then
  EDGE_NETWORK="${EDGE_NETWORK:-mcp-vps-agent-edge}"
  CERTRESOLVER="${CERTRESOLVER:-letsencrypt}"
  docker network inspect "$EDGE_NETWORK" >/dev/null 2>&1 || docker network create "$EDGE_NETWORK" >/dev/null
else
  if [ -n "$EDGE_NETWORK" ]; then
    for id in "${TRAEFIK_IDS[@]:-}"; do
      [ -n "$id" ] || continue
      if docker inspect "$id" --format '{{range $name, $cfg := .NetworkSettings.Networks}}{{$name}}{{"\n"}}{{end}}' |
          grep -Fxq "$EDGE_NETWORK"; then
        if [ -n "$TRAEFIK_ID" ]; then
          echo "ERROR: more than one Traefik is attached to $EDGE_NETWORK." >&2
          exit 1
        fi
        TRAEFIK_ID="$id"
      fi
    done
  elif [ "${#TRAEFIK_IDS[@]}" -eq 1 ]; then
    TRAEFIK_ID="${TRAEFIK_IDS[0]}"
  elif [ "${#TRAEFIK_IDS[@]}" -gt 1 ]; then
    echo "ERROR: multiple running Traefik containers found." >&2
    echo "Pass --edge-network NETWORK to select the edge explicitly." >&2
    exit 1
  fi

  if [ -z "$TRAEFIK_ID" ]; then
    if ss -ltnH | awk '{print $4}' | grep -Eq '(^|:)(80|443)$'; then
      echo "ERROR: no reusable Traefik was found, but host ports 80/443 are already in use." >&2
      echo "The installer will not replace an unknown web server." >&2
      exit 1
    fi
    echo "No existing Traefik found and ports 80/443 are free; using the bundled Traefik."
    BUNDLED_PROXY=1
    EDGE_NETWORK="${EDGE_NETWORK:-mcp-vps-agent-edge}"
    CERTRESOLVER="${CERTRESOLVER:-letsencrypt}"
    docker network inspect "$EDGE_NETWORK" >/dev/null 2>&1 || docker network create "$EDGE_NETWORK" >/dev/null
  else
    if [ -z "$EDGE_NETWORK" ]; then
      mapfile -t NETWORKS < <(
        docker inspect "$TRAEFIK_ID" --format '{{range $name, $cfg := .NetworkSettings.Networks}}{{$name}}{{"\n"}}{{end}}' |
          grep -Ev '^(bridge|host|none)$' |
          sed '/^$/d'
      )
      if [ "${#NETWORKS[@]}" -ne 1 ]; then
        echo "ERROR: Traefik has ${#NETWORKS[@]} candidate Docker networks: ${NETWORKS[*]:-none}" >&2
        echo "Pass --edge-network NETWORK." >&2
        exit 1
      fi
      EDGE_NETWORK="${NETWORKS[0]}"
    fi
    docker network inspect "$EDGE_NETWORK" >/dev/null 2>&1 || {
      echo "ERROR: Docker network not found: $EDGE_NETWORK" >&2
      exit 1
    }

    if [ -z "$CERTRESOLVER" ]; then
      CERTRESOLVER="$(
        docker inspect "$TRAEFIK_ID" --format '{{json .Config.Labels}}' |
          python3 -c '
import json,sys
labels=json.load(sys.stdin) or {}
values=sorted({v for k,v in labels.items() if k.endswith(".tls.certresolver") and v})
print(values[0] if len(values)==1 else "")
'
      )"
    fi
    if [ -z "$CERTRESOLVER" ]; then
      echo "ERROR: could not uniquely discover the Traefik ACME certificate resolver." >&2
      echo "Pass --certresolver NAME." >&2
      exit 1
    fi
  fi
fi

MASTERKEY="$(env_value ZITADEL_MASTERKEY || true)"
POSTGRES_PASSWORD="$(env_value ZITADEL_POSTGRES_PASSWORD || true)"
BOOTSTRAP_ADMIN_PASSWORD="$(env_value ZITADEL_BOOTSTRAP_ADMIN_PASSWORD || true)"
[ -n "$MASTERKEY" ] || MASTERKEY="$(openssl rand -hex 16)"
[ -n "$POSTGRES_PASSWORD" ] || POSTGRES_PASSWORD="$(openssl rand -hex 24)"
[ -n "$BOOTSTRAP_ADMIN_PASSWORD" ] || BOOTSTRAP_ADMIN_PASSWORD="T1!$(openssl rand -hex 18)"

upsert_env VPS_AGENT_DOMAIN "$DOMAIN"
upsert_env VPS_AGENT_PUBLIC_URL "https://$DOMAIN/mcp"
upsert_env VPS_AGENT_AUTH_MODE integrated
upsert_env VPS_AGENT_OIDC_ISSUER "https://$DOMAIN"
upsert_env VPS_AGENT_INTEGRATED_USERINFO_URL "http://zitadel-auth-internal:8080/oidc/v1/userinfo"
upsert_env VPS_AGENT_INTEGRATED_USERINFO_HOST "$DOMAIN"
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
if [ -n "$OPERATOR_EMAIL" ]; then
  upsert_env VPS_AGENT_LETSENCRYPT_EMAIL "$OPERATOR_EMAIL"
fi
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

"${compose[@]}" config -q
if [ "$BUNDLED_PROXY" -eq 1 ]; then
  "${compose[@]}" up -d edge-proxy
fi
"${compose[@]}" up -d zitadel-postgres zitadel-api zitadel-login

OIDC_URL="https://$DOMAIN/.well-known/openid-configuration"
echo "Waiting for public OIDC discovery and TLS..."
rm -f /tmp/vps-agent-oidc.json
for _ in $(seq 1 90); do
  if curl -fsS --max-time 5 "$OIDC_URL" >/tmp/vps-agent-oidc.json 2>/dev/null; then
    break
  fi
  sleep 2
done
if ! test -s /tmp/vps-agent-oidc.json; then
  echo "ERROR: OIDC discovery did not become reachable at $OIDC_URL" >&2
  echo "Check DNS, firewall/ports 80/443 and Traefik, then rerun." >&2
  "${compose[@]}" ps >&2 || true
  exit 1
fi
python3 - "$DOMAIN" <<'PY'
import json,sys
domain=sys.argv[1]
data=json.load(open("/tmp/vps-agent-oidc.json"))
issuer=data.get("issuer","").rstrip("/")
assert issuer == f"https://{domain}", issuer
print("OIDC DISCOVERY: PASS")
PY

if [ "$NEEDS_OPERATOR" -eq 1 ]; then
  BOOTSTRAP_PAT="$(
    docker run --rm       -v mcp-vps-agent_zitadel-bootstrap:/zitadel/bootstrap:ro       alpine:3.22       cat /zitadel/bootstrap/bootstrap-admin.pat 2>/dev/null || true
  )"
  if [ -z "$BOOTSTRAP_PAT" ]; then
    echo "ERROR: ZITADEL bootstrap PAT was not produced." >&2
    echo "If this is a partially initialized install, restore the identity backup or purge and retry." >&2
    exit 1
  fi

  ZITADEL_CID="$("${compose[@]}" ps -q zitadel-api)"
  if [ -z "$ZITADEL_CID" ]; then
    echo "ERROR: ZITADEL API container is not running." >&2
    exit 1
  fi

  curl_zitadel_internal() {
    docker run --rm -i       --network "container:$ZITADEL_CID"       curlimages/curl:8.16.0       -sS       -H "Host: $DOMAIN"       -H 'X-Forwarded-Proto: https'       "$@"
  }

  echo "Enabling MCP-compatible Dynamic Client Registration privately..."
  curl_zitadel_internal --fail     --request PUT     --url "http://127.0.0.1:8080/v2/settings/security"     --header "Authorization: Bearer $BOOTSTRAP_PAT"     --header 'Content-Type: application/json'     --data '{"dynamicClientRegistration":{"enabled":true,"allowUnauthenticated":true}}'     >/tmp/vps-agent-dcr-settings.json

  PASSWORD="${VPS_AGENT_OPERATOR_PASSWORD:-}"
  if [ -z "$PASSWORD" ]; then
    if [ ! -t 0 ]; then
      echo "ERROR: set VPS_AGENT_OPERATOR_PASSWORD for non-interactive bootstrap." >&2
      exit 1
    fi
    while :; do
      printf 'Create operator password (12+ characters): ' >&2
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
  elif [ "${#PASSWORD}" -lt 12 ]; then
    echo "ERROR: VPS_AGENT_OPERATOR_PASSWORD must contain at least 12 characters." >&2
    exit 1
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
with open(path,"w") as f:
    json.dump(payload,f)
PY
  unset PASSWORD PASSWORD2 2>/dev/null || true

  echo "Creating the dedicated non-admin VPS operator identity..."
  set +e
  HTTP_CODE="$(
    curl_zitadel_internal       --output /tmp/vps-agent-create-operator.json       --write-out '%{http_code}'       --request POST       --url "http://127.0.0.1:8080/v2/users/human"       --header "Authorization: Bearer $BOOTSTRAP_PAT"       --header 'Content-Type: application/json'       --data-binary @-       < "$REQUEST_FILE"
  )"
  CURL_STATUS=$?
  set -e
  rm -f "$REQUEST_FILE"
  trap - EXIT

  if [ "$CURL_STATUS" -ne 0 ] || { [ "$HTTP_CODE" != "200" ] && [ "$HTTP_CODE" != "201" ]; }; then
    echo "ERROR: operator creation failed (curl=$CURL_STATUS HTTP=${HTTP_CODE:-none})." >&2
    cat /tmp/vps-agent-create-operator.json >&2 2>/dev/null || true
    exit 1
  fi

  upsert_env VPS_AGENT_SUBJECT "$OPERATOR_ID"
  CURRENT_SUBJECT="$OPERATOR_ID"
  mkdir -p state
  chmod 700 state
  python3 - "$MARKER" "$DOMAIN" "$OPERATOR_ID" "$OPERATOR_USERNAME" "$OPERATOR_EMAIL" <<'PY'
import datetime,json,sys
path,domain,subject,username,email=sys.argv[1:]
payload={
  "domain":domain,
  "subject":subject,
  "username":username,
  "email":email,
  "created_at":datetime.datetime.now(datetime.timezone.utc).isoformat()
}
with open(path,"w") as f:
    json.dump(payload,f,indent=2)
    f.write("\n")
PY
  chmod 600 "$MARKER"

  # Remove the bootstrap human IAM owner before discarding the short-lived PAT.
  curl_zitadel_internal --fail --request POST --url "http://127.0.0.1:8080/v2/users" --header "Authorization: Bearer $BOOTSTRAP_PAT" --header 'Content-Type: application/json' --data '{}' >/tmp/vps-agent-users.json

  BOOTSTRAP_USER_ID="$(python3 - "$DOMAIN" <<'PY'
import json,sys
domain=sys.argv[1]
data=json.load(open("/tmp/vps-agent-users.json"))
target=f"bootstrap-admin@{domain}".lower()
matches=[]
for user in data.get("result", []):
    human=user.get("human") or {}
    email=(human.get("email") or {}).get("email","").lower()
    username=user.get("username","").lower()
    if email == target or username == target or username.startswith("bootstrap-admin@"):
        uid=user.get("userId","")
        if uid:
            matches.append(uid)
if len(matches) != 1:
    raise SystemExit(f"expected exactly one bootstrap human user, found {len(matches)}")
print(matches[0])
PY
  )"

  curl_zitadel_internal --fail --request DELETE --url "http://127.0.0.1:8080/v2/users/$BOOTSTRAP_USER_ID" --header "Authorization: Bearer $BOOTSTRAP_PAT" >/tmp/vps-agent-delete-bootstrap-user.json
  echo "Bootstrap human IAM owner removed."

  # The short-lived bootstrap machine PAT is no longer needed.
  docker run --rm -v mcp-vps-agent_zitadel-bootstrap:/zitadel/bootstrap alpine:3.22 rm -f /zitadel/bootstrap/bootstrap-admin.pat >/dev/null 2>&1 || true
fi

echo "Confirming OAuth dynamic-client discovery..."
for _ in $(seq 1 30); do
  curl -fsS "$OIDC_URL" >/tmp/vps-agent-oidc.json
  if python3 - <<'PY'
import json
x=json.load(open("/tmp/vps-agent-oidc.json"))
ok=bool(x.get("registration_endpoint")) and "S256" in x.get("code_challenge_methods_supported",[])
raise SystemExit(0 if ok else 1)
PY
  then
    break
  fi
  sleep 2
done
python3 - <<'PY'
import json
x=json.load(open("/tmp/vps-agent-oidc.json"))
endpoint=x.get("registration_endpoint","")
assert endpoint.startswith("https://"), x
assert "S256" in x.get("code_challenge_methods_supported",[]), x
print("DYNAMIC CLIENT REGISTRATION + PKCE: PASS")
PY

echo "Switching Gateway and Broker to the integrated operator subject..."
"${compose[@]}" up -d --build --force-recreate broker gateway

echo "Waiting for the public MCP protected resource..."
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

The operator password was not written to this installer state.

Next:
  bash scripts/connect-chatgpt.sh

EOF
