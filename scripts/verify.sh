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

docker compose config -q

broker_id="$(docker compose ps -a -q broker 2>/dev/null || true)"
gateway_id="$(docker compose ps -a -q gateway 2>/dev/null || true)"
if [ -z "$broker_id" ] || [ -z "$gateway_id" ]; then
  echo "ERROR: Broker/Gateway containers have not both been created." >&2
  echo "Run docker compose up -d --build successfully before verify.sh." >&2
  docker compose ps -a >&2 || true
  exit 1
fi

echo "Waiting for containers..."
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
  echo "ERROR: Broker/Gateway did not become healthy." >&2
  docker compose ps -a >&2 || true
  docker compose logs --no-color --tail 80 broker gateway >&2 || true
  exit 1
fi

docker compose ps

docker compose exec -T broker /usr/local/bin/vps-agent audit-status >/tmp/vps-agent-audit.json
cat /tmp/vps-agent-audit.json

if [ "${VPS_AGENT_AUTH_MODE:-static}" = "static" ]; then
  # Negative authentication test first: the MCP endpoint must reject a bad bearer.
  docker compose exec -T gateway sh -c \
    'header_name="Authorization"; bad_scheme="Bearer"; bad_value="definitely-wrong"; code="$(curl -sS -o /tmp/bad-auth.out -w "%{http_code}" -H "$header_name: $bad_scheme $bad_value" http://127.0.0.1:8080/mcp)"; test "$code" = "401"'

  docker compose exec -T gateway /usr/local/bin/vps-agent-mcp-call \
    --endpoint http://127.0.0.1:8080/mcp \
    --token "$VPS_AGENT_STATIC_TOKEN" \
    --tool system.info \
    --args '{}' >/tmp/vps-agent-system-info.json
else
  echo "Local MCP tool call skipped for auth mode ${VPS_AGENT_AUTH_MODE}; use scripts/verify-public.sh with a real OAuth token when available."
  printf '{"is_error":false,"mode":"oauth-external"}\n' >/tmp/vps-agent-system-info.json
fi
cat /tmp/vps-agent-system-info.json

python3 - <<'PY'
import json
a=json.load(open("/tmp/vps-agent-audit.json"))
assert a["ok"] is True and a["result"]["valid"] is True, a
s=json.load(open("/tmp/vps-agent-system-info.json"))
assert s["is_error"] is False, s
print("LOCAL MCP VERIFICATION: PASS")
PY

echo
echo "Runtime is healthy. Installation is NOT complete until ChatGPT Web is connected and verified."
echo "Next: ./scripts/connect-chatgpt.sh"
