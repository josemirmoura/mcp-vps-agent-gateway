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

echo "Waiting for containers..."
for _ in $(seq 1 60); do
  broker="$(docker inspect -f '{{.State.Health.Status}}' mcp-vps-agent-broker-1 2>/dev/null || true)"
  gateway="$(docker inspect -f '{{.State.Health.Status}}' mcp-vps-agent-gateway-1 2>/dev/null || true)"
  if [ "$broker" = "healthy" ] && [ "$gateway" = "healthy" ]; then
    break
  fi
  sleep 1
done

docker compose ps

docker compose exec -T broker /usr/local/bin/vps-agent audit-status >/tmp/vps-agent-audit.json
cat /tmp/vps-agent-audit.json

docker compose exec -T gateway /usr/local/bin/vps-agent-mcp-call \
  --endpoint http://127.0.0.1:8080/mcp \
  --token "$VPS_AGENT_STATIC_TOKEN" \
  --tool system.info \
  --args '{}' >/tmp/vps-agent-system-info.json
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
