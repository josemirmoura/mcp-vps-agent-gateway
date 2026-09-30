#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# shellcheck source=scripts/lib/product.sh
source scripts/lib/product.sh
vps_agent_init_language ""

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

The dedicated OAuth operator password is requested locally without terminal
echo. It is not the VPS/SSH password and must not be supplied to ChatGPT.
For CI/automation, VPS_AGENT_OPERATOR_PASSWORD may be supplied in the
environment; do not put a real password on a shared command line.
EOF
}

DOMAIN=""
OPERATOR_EMAIL=""
OPERATOR_USERNAME="vps-operator"
OPERATOR_USERNAME_EXPLICIT=0
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
      OPERATOR_USERNAME="$2"; OPERATOR_USERNAME_EXPLICIT=1; shift 2 ;;
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
    if vps_agent_is_pt_br; then
      cat <<'EOF'

Para conectar o ChatGPT ao Portico MCP, você precisa de um endereço público
para esta VPS.

1. No painel onde você gerencia seu domínio, crie um subdomínio, por exemplo:
   mcp.seudominio.com.br

2. Crie um registro DNS A apontando esse subdomínio para o IP público desta VPS.
   Se você usa IPv6, também pode criar um registro AAAA.

3. Aguarde o DNS responder e informe abaixo apenas o nome criado, sem https://
   e sem /mcp.

Exemplo:
  mcp.seudominio.com.br

EOF
      printf 'Domínio público do Portico MCP: '
    else
      cat <<'EOF'

To connect ChatGPT to Portico MCP, this VPS needs a public hostname.

1. In the DNS panel for a domain you control, create a subdomain, for example:
   mcp.example.com

2. Create a DNS A record pointing that subdomain to this VPS public IP.
   If you use IPv6, you may also create an AAAA record.

3. Wait for DNS to resolve, then enter only the hostname below, without
   https:// and without /mcp.

Example:
  mcp.example.com

EOF
      printf 'Public Portico MCP domain: '
    fi
    read -r DOMAIN
  else
    echo "$(vps_agent_text 'ERROR: --domain is required in non-interactive mode.' 'ERRO: --domain é obrigatório em modo não interativo.')" >&2
    exit 2
  fi
fi
if [[ "$DOMAIN" == *"://"* || "$DOMAIN" == */* || "$DOMAIN" == *" "* || "$DOMAIN" == .* || "$DOMAIN" == *. ]]; then
  echo "$(vps_agent_text 'ERROR: --domain must be a hostname only, for example mcp.example.com.' 'ERRO: --domain deve conter apenas o hostname, por exemplo mcp.exemplo.com.')" >&2
  exit 2
fi
if [[ ! "$DOMAIN" =~ ^[A-Za-z0-9.-]+$ ]]; then
  echo "$(vps_agent_text 'ERROR: --domain contains unsupported characters.' 'ERRO: --domain contém caracteres não suportados.')" >&2
  exit 2
fi

for cmd in docker curl openssl python3 ss getent; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "$(vps_agent_text "ERROR: required command not found: $cmd" "ERRO: comando obrigatório não encontrado: $cmd")" >&2; exit 1; }
done
docker compose version >/dev/null

upsert_env() {
  local key="$1"
  local value="$2"
  python3 - .env "$key" 3<<<"$value" <<'PY'
import os, pathlib, shlex, sys
path=pathlib.Path(sys.argv[1])
key=sys.argv[2]
value=os.fdopen(3).read()
if value.endswith("\n"):
    value=value[:-1]
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
  echo "$(vps_agent_text "ERROR: DNS for $DOMAIN does not resolve yet." "ERRO: o DNS de $DOMAIN ainda não está respondendo.")" >&2
  echo "$(vps_agent_text     'Create the A/AAAA record pointing to this VPS, wait for DNS propagation, and rerun.'     'Crie o registro A/AAAA apontando para esta VPS, aguarde a propagação do DNS e tente novamente.')" >&2
  exit 1
fi
DNS_ADDRS="$(getent ahosts "$DOMAIN" 2>/dev/null | awk '{print $1}' | sort -u | paste -sd, -)"
echo "$(vps_agent_text 'DNS resolved:' 'DNS resolvido:') $DOMAIN -> ${DNS_ADDRS:-?}"

MARKER="state/integrated-auth.json"
CURRENT_SUBJECT="$(env_value VPS_AGENT_SUBJECT || true)"
MARKER_DOMAIN="$(marker_value "$MARKER" domain || true)"
if [ -s "$MARKER" ] && [ -n "$MARKER_DOMAIN" ] && [ "$MARKER_DOMAIN" != "$DOMAIN" ]; then
  echo "$(vps_agent_text "ERROR: integrated auth was already initialized for $MARKER_DOMAIN." "ERRO: a autenticação integrada já foi inicializada para $MARKER_DOMAIN.")" >&2
  echo "$(vps_agent_text 'Automatic issuer/domain migration is intentionally not supported.' 'A migração automática de issuer/domínio não é suportada intencionalmente.')" >&2
  exit 1
fi

NEEDS_OPERATOR=1
if [ -s "$MARKER" ] && [ -n "$CURRENT_SUBJECT" ]; then
  NEEDS_OPERATOR=0
fi

if [ "$NEEDS_OPERATOR" -eq 1 ]; then
  if [ "$OPERATOR_USERNAME_EXPLICIT" -ne 1 ] && [ -t 0 ]; then
    printf '%s' "$(vps_agent_text 'OAuth login username [vps-operator]: ' 'Nome de usuário do login OAuth [vps-operator]: ')"
    read -r operator_username_answer
    OPERATOR_USERNAME="${operator_username_answer:-vps-operator}"
  fi
  if [ -z "$OPERATOR_EMAIL" ]; then
    if [ -t 0 ]; then
      printf '%s' "$(vps_agent_text 'Operator login email: ' 'E-mail da conta de operador: ')"
      read -r OPERATOR_EMAIL
    else
      echo "$(vps_agent_text 'ERROR: --operator-email is required in non-interactive mode.' 'ERRO: --operator-email é obrigatório em modo não interativo.')" >&2
      exit 2
    fi
  fi
  if [[ "$OPERATOR_EMAIL" != *@*.* ]]; then
    echo "$(vps_agent_text "ERROR: operator email does not look valid: $OPERATOR_EMAIL" "ERRO: o e-mail do operador parece inválido: $OPERATOR_EMAIL")" >&2
    exit 2
  fi
  if [[ ! "$OPERATOR_USERNAME" =~ ^[A-Za-z0-9._-]+$ ]]; then
    echo "$(vps_agent_text       'ERROR: operator username may contain only letters, numbers, dot, underscore and hyphen.'       'ERRO: o nome de usuário pode conter apenas letras, números, ponto, sublinhado e hífen.')" >&2
    exit 2
  fi
  echo
  echo "$(vps_agent_text 'Portico MCP operator account' 'Conta de operador do Portico MCP')"
  echo "  $(vps_agent_text 'Email' 'E-mail'): $OPERATOR_EMAIL"
  echo "  $(vps_agent_text 'Username' 'Nome de usuário'): $OPERATOR_USERNAME"
  echo "  $(vps_agent_text     'This username will be used on the OAuth login screen. It is not your Linux/SSH user.'     'Esse nome de usuário será usado na tela de login OAuth. Ele não é seu usuário Linux/SSH.')"
else
  OPERATOR_EMAIL="$(marker_value "$MARKER" email || true)"
  OPERATOR_USERNAME="$(marker_value "$MARKER" username || true)"
  echo "$(vps_agent_text     'Existing integrated operator identity detected; bootstrap will be reused.'     'Identidade de operador existente detectada; o bootstrap será reutilizado.')"
  echo "  $(vps_agent_text 'Username' 'Nome de usuário'): $OPERATOR_USERNAME"
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
    echo "$(vps_agent_text 'ERROR: multiple running Traefik containers found.' 'ERRO: mais de um contêiner Traefik em execução foi encontrado.')" >&2
    echo "$(vps_agent_text 'Pass --edge-network NETWORK to select the edge explicitly.' 'Informe --edge-network REDE para selecionar explicitamente a borda.')" >&2
    exit 1
  fi

  if [ -z "$TRAEFIK_ID" ]; then
    if ss -ltnH | awk '{print $4}' | grep -Eq '(^|:)(80|443)$'; then
      echo "$(vps_agent_text 'ERROR: no reusable Traefik was found, but host ports 80/443 are already in use.' 'ERRO: nenhum Traefik reutilizável foi encontrado, mas as portas 80/443 já estão em uso.')" >&2
      echo "$(vps_agent_text 'The installer will not replace an unknown web server.' 'O instalador não substituirá um servidor web desconhecido.')" >&2
      exit 1
    fi
    echo "$(vps_agent_text       'No existing Traefik was found and ports 80/443 are free. Portico will provision its bundled Traefik automatically.'       'Nenhum Traefik existente foi encontrado e as portas 80/443 estão livres. O Portico instalará automaticamente seu Traefik embutido.')"
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
      echo "$(vps_agent_text "ERROR: Docker network not found: $EDGE_NETWORK" "ERRO: rede Docker não encontrada: $EDGE_NETWORK")" >&2
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
      echo "$(vps_agent_text         'ERROR: could not uniquely discover the Traefik ACME certificate resolver.'         'ERRO: não foi possível descobrir de forma única o resolvedor ACME do Traefik.')" >&2
      echo "$(vps_agent_text 'Pass --certresolver NAME.' 'Informe --certresolver NOME.')" >&2
      exit 1
    fi
    echo "$(vps_agent_text       "Existing Traefik detected and will be reused. No new proxy will be installed."       "Traefik existente detectado e será reutilizado. Nenhum novo proxy será instalado.")"
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
upsert_env VPS_AGENT_OAUTH_RESOURCE "https://$DOMAIN/mcp"
upsert_env VPS_AGENT_RESOURCE_METADATA_URL "https://$DOMAIN/.well-known/oauth-protected-resource"
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
echo "$(vps_agent_text 'Integrated authentication plan:' 'Plano de autenticação integrada:')"
echo "  MCP + issuer: https://$DOMAIN"
echo "  $(vps_agent_text 'edge network' 'rede de borda'): $EDGE_NETWORK"
echo "  $(vps_agent_text 'certificate resolver' 'resolvedor de certificado'): $CERTRESOLVER"
if [ "$BUNDLED_PROXY" -eq 1 ]; then
  echo "  proxy: $(vps_agent_text 'bundled Traefik (Portico-managed)' 'Traefik embutido (gerenciado pelo Portico)')"
else
  echo "  proxy: $(vps_agent_text 'existing Traefik (reused)' 'Traefik existente (reutilizado)')"
fi
echo

"${compose[@]}" config -q
if [ "$BUNDLED_PROXY" -eq 1 ]; then
  "${compose[@]}" up -d edge-proxy
fi
"${compose[@]}" up -d zitadel-postgres zitadel-api zitadel-login

OIDC_URL="https://$DOMAIN/.well-known/openid-configuration"
echo "$(vps_agent_text 'Waiting for public OIDC discovery and TLS...' 'Aguardando descoberta OIDC pública e TLS...')"
rm -f /tmp/vps-agent-oidc.json
for _ in $(seq 1 90); do
  if curl -fsS --max-time 5 "$OIDC_URL" >/tmp/vps-agent-oidc.json 2>/dev/null; then
    break
  fi
  sleep 2
done
if ! test -s /tmp/vps-agent-oidc.json; then
  echo "$(vps_agent_text "ERROR: OIDC discovery did not become reachable at $OIDC_URL" "ERRO: a descoberta OIDC não ficou acessível em $OIDC_URL")" >&2
  echo "$(vps_agent_text 'Check DNS, firewall/ports 80/443 and Traefik, then rerun.' 'Confira DNS, firewall/portas 80/443 e Traefik e tente novamente.')" >&2
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

DCR_READY="$(python3 - <<'PY'
import json
x=json.load(open("/tmp/vps-agent-oidc.json"))
print("1" if x.get("registration_endpoint") else "0")
PY
)"

RESOURCE_PROJECT_ID="$(env_value VPS_AGENT_INTEGRATED_AUDIENCE_PROJECT_ID || true)"
INTROSPECTION_APP_ID="$(env_value VPS_AGENT_INTEGRATED_INTROSPECTION_APP_ID || true)"
INTROSPECTION_CLIENT_ID="$(env_value VPS_AGENT_INTEGRATED_INTROSPECTION_CLIENT_ID || true)"
INTROSPECTION_CLIENT_SECRET="$(env_value VPS_AGENT_INTEGRATED_INTROSPECTION_CLIENT_SECRET || true)"

NEEDS_RESOURCE=0
if [ -z "$RESOURCE_PROJECT_ID" ] || [ -z "$INTROSPECTION_CLIENT_ID" ] || [ -z "$INTROSPECTION_CLIENT_SECRET" ]; then
  NEEDS_RESOURCE=1
fi

NEEDS_BOOTSTRAP=0
if [ "$NEEDS_OPERATOR" -eq 1 ] || [ "$NEEDS_RESOURCE" -eq 1 ] || [ "$DCR_READY" != "1" ]; then
  NEEDS_BOOTSTRAP=1
fi

BOOTSTRAP_PAT=""
ZITADEL_CID=""
if [ "$NEEDS_BOOTSTRAP" -eq 1 ]; then
  # ZITADEL writes this short-lived IAM-owner PAT into a private package volume.
  # Read it through an ephemeral container because the production ZITADEL image
  # intentionally contains almost no shell utilities.
  BOOTSTRAP_PAT="$(
    docker run --rm \
      -v mcp-vps-agent_zitadel-bootstrap:/zitadel/bootstrap:ro \
      alpine:3.22 \
      cat /zitadel/bootstrap/bootstrap-admin.pat 2>/dev/null || true
  )"
  if [ -z "$BOOTSTRAP_PAT" ]; then
    echo "$(vps_agent_text 'ERROR: integrated OAuth bootstrap is incomplete but its short-lived admin PAT is unavailable.' 'ERRO: o bootstrap OAuth integrado está incompleto, mas o PAT administrativo temporário não está disponível.')" >&2
    echo "$(vps_agent_text 'Restore the identity backup or purge this incomplete identity stack and rerun setup.' 'Restaure o backup de identidade ou remova esta stack incompleta e execute a configuração novamente.')" >&2
    exit 1
  fi

  ZITADEL_CID="$("${compose[@]}" ps -q zitadel-api)"
  if [ -z "$ZITADEL_CID" ]; then
    echo "$(vps_agent_text 'ERROR: ZITADEL API container is not running.' 'ERRO: o contêiner da API ZITADEL não está em execução.')" >&2
    exit 1
  fi

  curl_zitadel_internal() {
    docker run --rm -i \
      --network "container:$ZITADEL_CID" \
      curlimages/curl:8.16.0 \
      -sS \
      -H "Host: $DOMAIN" \
      -H 'X-Forwarded-Proto: https' \
      "$@"
  }

  # Tighten bootstrap-volume permissions before any credential is consumed.
  docker run --rm \
    -v mcp-vps-agent_zitadel-bootstrap:/zitadel/bootstrap \
    alpine:3.22 \
    sh -c 'chmod 600 /zitadel/bootstrap/*.pat 2>/dev/null || true'

  if [ "$DCR_READY" != "1" ]; then
    echo "$(vps_agent_text 'Enabling MCP-compatible Dynamic Client Registration privately...' 'Habilitando o registro dinâmico de cliente compatível com MCP de forma privada...')"
    curl_zitadel_internal --fail \
      --request PUT \
      --url "http://127.0.0.1:8080/v2/settings/security" \
      --header "Authorization: Bearer $BOOTSTRAP_PAT" \
      --header 'Content-Type: application/json' \
      --data '{"dynamicClientRegistration":{"enabled":true,"allowUnauthenticated":true}}' \
      >/tmp/vps-agent-dcr-settings.json
  fi
fi

if [ "$NEEDS_OPERATOR" -eq 1 ]; then
  PASSWORD="${VPS_AGENT_OPERATOR_PASSWORD:-}"
  if [ -z "$PASSWORD" ]; then
    if [ ! -t 0 ]; then
      echo "$(vps_agent_text 'ERROR: set VPS_AGENT_OPERATOR_PASSWORD for non-interactive bootstrap.' 'ERRO: defina VPS_AGENT_OPERATOR_PASSWORD para o bootstrap não interativo.')" >&2
      exit 1
    fi
    while :; do
      printf '%s' "$(vps_agent_text 'Create operator password (12+ characters): ' 'Crie a senha do operador (12+ caracteres): ')" >&2
      read -r -s PASSWORD
      echo >&2
      printf '%s' "$(vps_agent_text 'Repeat operator password: ' 'Repita a senha do operador: ')" >&2
      read -r -s PASSWORD2
      echo >&2
      if [ "$PASSWORD" != "$PASSWORD2" ]; then
        echo "$(vps_agent_text 'Passwords do not match.' 'As senhas não coincidem.')" >&2
        continue
      fi
      if [ "${#PASSWORD}" -lt 12 ]; then
        echo "$(vps_agent_text 'Use at least 12 characters.' 'Use pelo menos 12 caracteres.')" >&2
        continue
      fi
      break
    done
  elif [ "${#PASSWORD}" -lt 12 ]; then
    echo "$(vps_agent_text 'ERROR: VPS_AGENT_OPERATOR_PASSWORD must contain at least 12 characters.' 'ERRO: VPS_AGENT_OPERATOR_PASSWORD deve conter pelo menos 12 caracteres.')" >&2
    exit 1
  fi

  OPERATOR_ID="$(python3 -c 'import uuid; print(uuid.uuid4())')"
  REQUEST_FILE="$(mktemp)"
  chmod 600 "$REQUEST_FILE"
  trap 'rm -f "$REQUEST_FILE"' EXIT

  # Password travels on a private inherited file descriptor, never argv.
  python3 - "$REQUEST_FILE" "$OPERATOR_ID" "$OPERATOR_USERNAME" "$OPERATOR_EMAIL" "$VPS_AGENT_LANG" 3<<<"$PASSWORD" <<'PY'
import json,os,sys
path,user_id,username,email,lang=sys.argv[1:]
password=os.fdopen(3).read()
if password.endswith("\n"):
    password=password[:-1]
payload={
  "userId": user_id,
  "username": username,
  "profile": {
    "givenName": "VPS",
    "familyName": "Operator",
    "displayName": "Portico MCP Operator",
    "preferredLanguage": "pt" if lang == "pt-BR" else "en"
  },
  "email": {"email": email, "isVerified": True},
  "password": {"password": password, "changeRequired": False}
}
with open(path,"w") as f:
    json.dump(payload,f)
PY
  unset PASSWORD PASSWORD2 2>/dev/null || true

  echo "$(vps_agent_text 'Creating the dedicated non-admin Portico operator identity...' 'Criando a identidade dedicada e não administrativa do operador Portico...')"
  set +e
  HTTP_CODE="$(
    curl_zitadel_internal \
      --output /tmp/vps-agent-create-operator.json \
      --write-out '%{http_code}' \
      --request POST \
      --url "http://127.0.0.1:8080/v2/users/human" \
      --header "Authorization: Bearer $BOOTSTRAP_PAT" \
      --header 'Content-Type: application/json' \
      --data-binary @- \
      < "$REQUEST_FILE"
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
  "bootstrap_complete":False,
  "created_at":datetime.datetime.now(datetime.timezone.utc).isoformat()
}
with open(path,"w") as f:
    json.dump(payload,f,indent=2)
    f.write("\n")
PY
  chmod 600 "$MARKER"
fi

if [ "$NEEDS_RESOURCE" -eq 1 ]; then
  if [ -z "$RESOURCE_PROJECT_ID" ]; then
    echo "$(vps_agent_text 'Creating the dedicated OAuth resource/audience project...' 'Criando o projeto OAuth dedicado de recurso/audiência...')"
    RESOURCE_RESPONSE="$(mktemp)"
    chmod 600 "$RESOURCE_RESPONSE"
    curl_zitadel_internal --fail \
      --request POST \
      --url "http://127.0.0.1:8080/management/v1/projects" \
      --header "Authorization: Bearer $BOOTSTRAP_PAT" \
      --header 'Content-Type: application/json' \
      --data '{"name":"Portico MCP Resource"}' \
      >"$RESOURCE_RESPONSE"
    RESOURCE_PROJECT_ID="$(python3 - "$RESOURCE_RESPONSE" <<'PY'
import json,sys
x=json.load(open(sys.argv[1]))
value=x.get("id","")
assert value, x
print(value)
PY
    )"
    rm -f "$RESOURCE_RESPONSE"
    upsert_env VPS_AGENT_INTEGRATED_AUDIENCE_PROJECT_ID "$RESOURCE_PROJECT_ID"
  fi

  if [ -z "$INTROSPECTION_CLIENT_ID" ] || [ -z "$INTROSPECTION_CLIENT_SECRET" ]; then
    echo "$(vps_agent_text 'Creating the private token-introspection API client...' 'Criando o cliente privado de API para introspecção de token...')"
    APP_RESPONSE="$(mktemp)"
    chmod 600 "$APP_RESPONSE"
    curl_zitadel_internal --fail \
      --request POST \
      --url "http://127.0.0.1:8080/management/v1/projects/$RESOURCE_PROJECT_ID/apps/api" \
      --header "Authorization: Bearer $BOOTSTRAP_PAT" \
      --header 'Content-Type: application/json' \
      --data '{"name":"Portico MCP Introspector","authMethodType":"API_AUTH_METHOD_TYPE_BASIC"}' \
      >"$APP_RESPONSE"

    readarray -t APP_VALUES < <(
      python3 - "$APP_RESPONSE" <<'PY'
import json,sys
x=json.load(open(sys.argv[1]))
values=[x.get("appId",""), x.get("clientId",""), x.get("clientSecret","")]
assert all(values), x
for value in values:
    if "\n" in value or "\r" in value:
        raise SystemExit("newline in generated OAuth credential")
    print(value)
PY
    )
    [ "${#APP_VALUES[@]}" -eq 3 ] || { echo "ERROR: incomplete introspection client response." >&2; exit 1; }
    INTROSPECTION_APP_ID="${APP_VALUES[0]}"
    INTROSPECTION_CLIENT_ID="${APP_VALUES[1]}"
    INTROSPECTION_CLIENT_SECRET="${APP_VALUES[2]}"
    rm -f "$APP_RESPONSE"

    upsert_env VPS_AGENT_INTEGRATED_INTROSPECTION_APP_ID "$INTROSPECTION_APP_ID"
    upsert_env VPS_AGENT_INTEGRATED_INTROSPECTION_CLIENT_ID "$INTROSPECTION_CLIENT_ID"
    upsert_env VPS_AGENT_INTEGRATED_INTROSPECTION_CLIENT_SECRET "$INTROSPECTION_CLIENT_SECRET"
  fi
fi

AUDIENCE_SCOPE="urn:zitadel:iam:org:project:id:$RESOURCE_PROJECT_ID:aud"
upsert_env VPS_AGENT_INTEGRATED_INTROSPECTION_URL "http://zitadel-auth-internal:8080/oauth/v2/introspect"
upsert_env VPS_AGENT_INTEGRATED_INTROSPECTION_HOST "$DOMAIN"
upsert_env VPS_AGENT_INTEGRATED_AUDIENCE_PROJECT_ID "$RESOURCE_PROJECT_ID"
upsert_env VPS_AGENT_REQUIRED_SCOPES "openid $AUDIENCE_SCOPE"
chmod 600 .env

if [ "$NEEDS_BOOTSTRAP" -eq 1 ]; then
  echo "$(vps_agent_text 'Verifying the private introspection client...' 'Verificando o cliente privado de introspecção...')"
  curl_zitadel_internal --fail \
    --request POST \
    --url "http://127.0.0.1:8080/oauth/v2/introspect" \
    --user "$INTROSPECTION_CLIENT_ID:$INTROSPECTION_CLIENT_SECRET" \
    --header 'Content-Type: application/x-www-form-urlencoded' \
    --data 'token=deliberately-invalid-bootstrap-probe' \
    >/tmp/vps-agent-introspection-probe.json
  python3 - <<'PY'
import json
x=json.load(open("/tmp/vps-agent-introspection-probe.json"))
assert x.get("active") is False, x
print("PRIVATE TOKEN INTROSPECTION CLIENT: PASS")
PY

  # Re-read bootstrap identities while the PAT still works. Remove the human
  # IAM owner first, then the machine IAM owner that issued this PAT. Deleting
  # the machine last revokes the bootstrap credential at the authority itself.
  curl_zitadel_internal --fail \
    --request POST \
    --url "http://127.0.0.1:8080/v2/users" \
    --header "Authorization: Bearer $BOOTSTRAP_PAT" \
    --header 'Content-Type: application/json' \
    --data '{}' \
    >/tmp/vps-agent-users.json

  readarray -t BOOTSTRAP_IDS < <(python3 - "$DOMAIN" <<'PY'
import json,sys
domain=sys.argv[1].lower()
data=json.load(open("/tmp/vps-agent-users.json"))
human=[]
machine=[]
for user in data.get("result", []):
    uid=user.get("userId","")
    username=user.get("username","").lower()
    h=user.get("human")
    m=user.get("machine")
    if h is not None:
        email=(h.get("email") or {}).get("email","").lower()
        if email == f"bootstrap-admin@{domain}" or username.startswith("bootstrap-admin@"):
            human.append(uid)
    if m is not None and (username.startswith("vps-agent-bootstrap") or (m.get("name","").lower() == "vps agent bootstrap")):
        machine.append(uid)
if len(human) > 1 or len(machine) > 1:
    raise SystemExit("ambiguous bootstrap identity set")
print(human[0] if human else "")
print(machine[0] if machine else "")
PY
  )
  BOOTSTRAP_HUMAN_ID="${BOOTSTRAP_IDS[0]:-}"
  BOOTSTRAP_MACHINE_ID="${BOOTSTRAP_IDS[1]:-}"

  if [ -n "$BOOTSTRAP_HUMAN_ID" ]; then
    curl_zitadel_internal --fail \
      --request DELETE \
      --url "http://127.0.0.1:8080/v2/users/$BOOTSTRAP_HUMAN_ID" \
      --header "Authorization: Bearer $BOOTSTRAP_PAT" \
      >/tmp/vps-agent-delete-bootstrap-human.json
    echo "$(vps_agent_text 'Bootstrap human IAM owner removed.' 'Proprietário IAM humano de bootstrap removido.')"
  fi

  if [ -n "$BOOTSTRAP_MACHINE_ID" ]; then
    curl_zitadel_internal --fail \
      --request DELETE \
      --url "http://127.0.0.1:8080/v2/users/$BOOTSTRAP_MACHINE_ID" \
      --header "Authorization: Bearer $BOOTSTRAP_PAT" \
      >/tmp/vps-agent-delete-bootstrap-machine.json
    echo "$(vps_agent_text 'Bootstrap machine IAM owner removed and its PAT revoked.' 'Proprietário IAM de máquina do bootstrap removido e seu PAT revogado.')"
  fi

  docker run --rm \
    -v mcp-vps-agent_zitadel-bootstrap:/zitadel/bootstrap \
    alpine:3.22 \
    rm -f /zitadel/bootstrap/bootstrap-admin.pat \
    >/dev/null 2>&1 || true

  python3 - "$MARKER" "$RESOURCE_PROJECT_ID" "$INTROSPECTION_APP_ID" <<'PY'
import datetime,json,pathlib,sys
path=pathlib.Path(sys.argv[1])
project_id=sys.argv[2]
app_id=sys.argv[3]
data=json.loads(path.read_text())
data["resource_project_id"]=project_id
data["introspection_app_id"]=app_id
data["bootstrap_complete"]=True
data["bootstrap_completed_at"]=datetime.datetime.now(datetime.timezone.utc).isoformat()
path.write_text(json.dumps(data,indent=2)+"\n")
PY
  chmod 600 "$MARKER"
fi

echo "$(vps_agent_text 'Confirming OAuth dynamic-client discovery...' 'Confirmando descoberta dinâmica de cliente OAuth...')"
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

echo "$(vps_agent_text 'Switching Gateway and Broker to the integrated operator subject...' 'Alterando Gateway e Broker para o subject do operador integrado...')"
"${compose[@]}" up -d --build --force-recreate broker gateway

echo "$(vps_agent_text 'Waiting for the public MCP protected resource...' 'Aguardando o recurso MCP público protegido...')"
for _ in $(seq 1 60); do
  code="$(curl -sS --max-time 5 -o /tmp/vps-agent-public-unauth.txt -w '%{http_code}' "https://$DOMAIN/mcp" 2>/dev/null || true)"
  [ "$code" = "401" ] && break
  sleep 2
done

bash scripts/verify-public.sh

echo
echo "$(vps_agent_text 'INTEGRATED AUTH: READY' 'AUTENTICAÇÃO INTEGRADA: PRONTA')"
echo
echo "$(vps_agent_text 'MCP endpoint:' 'Endpoint MCP:')"
echo "  https://$DOMAIN/mcp"
echo
echo "OAuth/OIDC issuer:"
echo "  https://$DOMAIN"
echo
echo "$(vps_agent_text 'Portico operator login:' 'Login do operador Portico:')"
echo "  $(vps_agent_text 'Username' 'Nome de usuário'): $OPERATOR_USERNAME"
echo "  $(vps_agent_text 'Email' 'E-mail'): $OPERATOR_EMAIL"
echo
echo "$(vps_agent_text   'The operator password was not written to installer state and must never be replaced with VPS/SSH credentials.'   'A senha do operador não foi gravada no estado do instalador e nunca deve ser substituída por credenciais VPS/SSH.')"
echo
echo "$(vps_agent_text 'Next:' 'Próximo:')"
echo "  bash scripts/connect-chatgpt.sh"
