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
  bundle)
    mkdir -p diagnostics
    stamp="$(date -u +%Y%m%dT%H%M%SZ)"
    output="${2:-diagnostics/vps-agent-diagnostic-$stamp.tar.gz}"
    tmp="$(mktemp -d)"
    trap 'rm -rf "$tmp"' EXIT

    docker compose ps >"$tmp/compose-ps.txt" 2>&1 || true
    docker version >"$tmp/docker-version.txt" 2>&1 || true
    docker compose version >"$tmp/compose-version.txt" 2>&1 || true
    uname -a >"$tmp/uname.txt" 2>&1 || true
    git rev-parse HEAD >"$tmp/git-head.txt" 2>&1 || true

    {
      printf 'instance_id=%s\n' "${VPS_AGENT_INSTANCE_ID:-}"
      printf 'instance_name=%s\n' "${VPS_AGENT_INSTANCE_NAME:-}"
      printf 'auth_mode=%s\n' "${VPS_AGENT_AUTH_MODE:-}"
      printf 'public_url=%s\n' "${VPS_AGENT_PUBLIC_URL:-}"
      printf 'oidc_issuer=%s\n' "${VPS_AGENT_OIDC_ISSUER:-}"
    } >"$tmp/runtime.txt"

    docker compose exec -T broker /usr/local/bin/vps-agent audit-status >"$tmp/audit-status.json" 2>&1 || true
    docker compose exec -T broker /usr/local/bin/vps-agent audit-tail --limit 200 >"$tmp/audit-tail.json" 2>&1 || true
    docker compose logs --no-color --tail 500 >"$tmp/logs.raw" 2>&1 || true

    python3 - "$tmp" "${VPS_AGENT_ADMIN_TOKEN:-}" "${VPS_AGENT_STATIC_TOKEN:-}" <<'PY'
import pathlib, re, sys
root=pathlib.Path(sys.argv[1])
secrets=[x for x in sys.argv[2:] if x]
for name in ("logs.raw","audit-tail.json","audit-status.json"):
    p=root/name
    if not p.exists():
        continue
    text=p.read_text(errors="replace")
    for secret in secrets:
        text=text.replace(secret,"[REDACTED]")
    text=re.sub(r'(?i)(authorization|token|secret|password)(["=: ]+)[^\s",}]+', r'\1\2[REDACTED]', text)
    (root/name.replace(".raw",".txt")).write_text(text)

policy=pathlib.Path("config/policy.yaml")
if policy.exists():
    text=policy.read_text(errors="replace")
    text=re.sub(r'(?im)^(\s*[^#\n]*(?:token|secret|password)[^:]*:\s*).+
esac
, r'\1[REDACTED]', text)
    (root/"policy.redacted.yaml").write_text(text)
PY
    rm -f "$tmp/logs.raw"
    tar -czf "$output" -C "$tmp" .
    chmod 600 "$output"
    echo "$output"
    ;;
  *)
    echo "usage: $0 [status|logs [lines]|audit [limit]|bundle [output.tar.gz]]" >&2
    exit 2
    ;;
esac
