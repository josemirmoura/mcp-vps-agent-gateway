#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# shellcheck source=scripts/lib/product.sh
source scripts/lib/product.sh
vps_agent_init_language ""

if [ ! -f .env ]; then
  echo "$(vps_agent_text 'Missing .env. Run scripts/init.sh first.' 'Arquivo .env ausente. Execute scripts/init.sh primeiro.')" >&2
  exit 1
fi
set -a
. ./.env
set +a

docker compose config -q

broker_id="$(docker compose ps -a -q broker 2>/dev/null || true)"
gateway_id="$(docker compose ps -a -q gateway 2>/dev/null || true)"
if [ -z "$broker_id" ] || [ -z "$gateway_id" ]; then
  echo "$(vps_agent_text 'ERROR: Broker/Gateway containers have not both been created.' 'ERRO: os contêineres Broker/Gateway ainda não foram ambos criados.')" >&2
  echo "$(vps_agent_text 'Run docker compose up -d --build successfully before verify.sh.' 'Execute docker compose up -d --build com sucesso antes de verify.sh.')" >&2
  docker compose ps -a >&2 || true
  exit 1
fi

echo "$(vps_agent_text 'Waiting for Portico containers...' 'Aguardando os contêineres do Portico...')"
ready=0
for _ in $(seq 1 60); do
  broker="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$broker_id" 2>/dev/null || true)"
  gateway="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$gateway_id" 2>/dev/null || true)"
  if [ "$broker" = "healthy" ] && [ "$gateway" = "healthy" ]; then
    ready=1
    break
  fi
  if [ "$broker" = "unhealthy" ] || [ "$gateway" = "unhealthy" ] || [ "$broker" = "exited" ] || [ "$gateway" = "exited" ] || [ "$broker" = "dead" ] || [ "$gateway" = "dead" ]; then
    break
  fi
  sleep 1
done

if [ "$ready" != "1" ]; then
  echo "$(vps_agent_text 'ERROR: Broker/Gateway did not become healthy.' 'ERRO: Broker/Gateway não ficaram saudáveis.')" >&2
  docker compose ps -a >&2 || true
  docker compose logs --no-color --tail 80 broker gateway >&2 || true
  exit 1
fi

docker compose ps

docker compose exec -T broker /usr/local/bin/vps-agent audit-status >/tmp/vps-agent-audit.json
cat /tmp/vps-agent-audit.json

if [ "${VPS_AGENT_AUTH_MODE:-static}" = "static" ]; then
  docker compose exec -T gateway sh -c \
    'header_name="Authorization"; bad_scheme="Bearer"; bad_value="definitely-wrong"; code="$(curl -sS -o /tmp/bad-auth.out -w "%{http_code}" -H "$header_name: $bad_scheme $bad_value" http://127.0.0.1:8080/mcp)"; test "$code" = "401"'

  docker compose exec -T gateway /usr/local/bin/vps-agent-mcp-call \
    --endpoint http://127.0.0.1:8080/mcp \
    --token "$VPS_AGENT_STATIC_TOKEN" \
    --tool system.info \
    --args '{}' >/tmp/vps-agent-system-info.json
else
  echo "$(vps_agent_msg verify.local_call_skipped "mode=${VPS_AGENT_AUTH_MODE}")"
  printf '{"is_error":false,"mode":"oauth-integrated"}\n' >/tmp/vps-agent-system-info.json
fi
cat /tmp/vps-agent-system-info.json

VPS_AGENT_LANG="$VPS_AGENT_LANG" python3 - <<'PY'
import json, os
pt=os.environ.get("VPS_AGENT_LANG")=="pt-BR"
a=json.load(open("/tmp/vps-agent-audit.json"))
assert a["ok"] is True and a["result"]["valid"] is True, a
s=json.load(open("/tmp/vps-agent-system-info.json"))
assert s["is_error"] is False, s
print("VERIFICAÇÃO MCP LOCAL: OK" if pt else "LOCAL MCP VERIFICATION: PASS")
PY

echo
echo "$(vps_agent_text 'Runtime is healthy. Installation is NOT complete until ChatGPT Web is connected and verified.' 'O runtime está saudável. A instalação AINDA NÃO terminou até o ChatGPT Web ser conectado e verificado.')"
if [ "${VPS_AGENT_AUTH_MODE:-static}" = "integrated" ]; then
  echo "$(vps_agent_text 'Next: bash scripts/verify-public.sh' 'Próximo: bash scripts/verify-public.sh')"
  echo "$(vps_agent_text 'Then: bash scripts/connect-chatgpt.sh' 'Depois: bash scripts/connect-chatgpt.sh')"
else
  echo "$(vps_agent_text 'Next: bash scripts/setup-integrated-auth.sh' 'Próximo: bash scripts/setup-integrated-auth.sh')"
fi
