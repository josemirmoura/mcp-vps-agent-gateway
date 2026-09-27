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

cmd="${1:-status}"
case "$cmd" in
  status)
    docker compose ps
    echo
    for service in broker gateway; do
      cid="$(docker compose ps -q "$service" 2>/dev/null || true)"
      if [ -n "$cid" ]; then
        printf '%s health: ' "$service"
        docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$cid"
      fi
    done
    echo
    docker compose exec -T broker /usr/local/bin/vps-agent audit-status
    ;;
  logs)
    tail="${2:-200}"
    tmp="$(mktemp)"
    trap 'rm -f "$tmp"' EXIT
    docker compose logs --no-color --tail "$tail" >"$tmp"
    python3 - "$tmp" "${VPS_AGENT_ADMIN_TOKEN:-}" "${VPS_AGENT_STATIC_TOKEN:-}" <<'PY'
import pathlib, sys
p=pathlib.Path(sys.argv[1])
text=p.read_text(errors="replace")
for secret in sys.argv[2:]:
    if secret:
        text=text.replace(secret, "[REDACTED]")
print(text, end="")
PY
    ;;
  audit)
    limit="${2:-100}"
    docker compose exec -T broker /usr/local/bin/vps-agent audit-status
    docker compose exec -T broker /usr/local/bin/vps-agent audit-tail --limit "$limit"
    ;;
  *)
    echo "usage: $0 [status|logs [lines]|audit [limit]]" >&2
    exit 2
    ;;
esac
