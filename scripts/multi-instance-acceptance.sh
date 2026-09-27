#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

INSTANCE_KEY="${1:?usage: $0 <instance-key> <instance-name>}"
INSTANCE_NAME="${2:?usage: $0 <instance-key> <instance-name>}"
EVIDENCE_DIR="${EVIDENCE_DIR:-multi-instance-evidence}"
SCOPE_ROOT="/opt/vps-agent-multi"
SUBJECT="multi-operator"
OPERATION_ID="shared-collision-operation-001"
START_EPOCH="$(date +%s)"

mkdir -p "$EVIDENCE_DIR"
cleanup() {
  docker compose down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

sudo install -d -o "$USER" -g "$USER" -m 0750 "$SCOPE_ROOT"

bash scripts/init.sh >"$EVIDENCE_DIR/bootstrap-${INSTANCE_KEY}.txt"

python3 - "$INSTANCE_KEY" "$INSTANCE_NAME" "$SCOPE_ROOT" <<'PY'
import pathlib, sys
key, name, scope = sys.argv[1:]
p = pathlib.Path(".env")
lines = p.read_text().splitlines()
updates = {
    "VPS_AGENT_INSTANCE_NAME": name,
    "VPS_AGENT_SCOPE_ROOT": scope,
    "VPS_AGENT_SUBJECT": "multi-operator",
    "VPS_AGENT_AUTH_MODE": "static",
    "VPS_AGENT_BIND_ADDRESS": "127.0.0.1",
    "VPS_AGENT_PORT": "8080",
}
out=[]
seen=set()
for line in lines:
    if "=" in line and not line.lstrip().startswith("#"):
        k=line.split("=",1)[0]
        if k in updates:
            v=updates[k]
            if any(ch in v for ch in " |"):
                v='"'+v.replace('"','\\"')+'"'
            line=f"{k}={v}"
            seen.add(k)
    out.append(line)
for k,v in updates.items():
    if k not in seen:
        if any(ch in v for ch in " |"):
            v='"'+v.replace('"','\\"')+'"'
        out.append(f"{k}={v}")
p.write_text("\n".join(out)+"\n")

policy=pathlib.Path("config/policy.yaml")
text=policy.read_text()
text=text.replace("/opt/my-app", scope)
policy.write_text(text)
PY

set -a
. ./.env
set +a

test -n "$VPS_AGENT_INSTANCE_ID"
test "$VPS_AGENT_INSTANCE_NAME" = "$INSTANCE_NAME"
test "$VPS_AGENT_SUBJECT" = "$SUBJECT"

printf 'before\n' > "$SCOPE_ROOT/shared.txt"

docker compose up -d --build
bash scripts/verify.sh >"$EVIDENCE_DIR/verify-${INSTANCE_KEY}.txt"

docker compose exec -T gateway /usr/local/bin/vps-agent-mcp-call \
  --endpoint http://127.0.0.1:8080/mcp \
  --token "$VPS_AGENT_STATIC_TOKEN" \
  --tool system.info \
  --args '{}' >"$EVIDENCE_DIR/system-info-${INSTANCE_KEY}.json"

PATCH_ARGS="$(python3 - "$SCOPE_ROOT/shared.txt" "$OPERATION_ID" <<'PY'
import json, sys
print(json.dumps({
    "path": sys.argv[1],
    "old_text": "before",
    "new_text": "after",
    "operation_id": sys.argv[2],
}))
PY
)"

docker compose exec -T gateway /usr/local/bin/vps-agent-mcp-call \
  --endpoint http://127.0.0.1:8080/mcp \
  --token "$VPS_AGENT_STATIC_TOKEN" \
  --tool file.patch \
  --args "$PATCH_ARGS" >"$EVIDENCE_DIR/patch-first-${INSTANCE_KEY}.json"

# Same subject + same tool + same operation id is intentionally replayed on
# every VPS. Within one VPS this must be idempotent; across VPSs the three
# independent state stores must not collide.
docker compose exec -T gateway /usr/local/bin/vps-agent-mcp-call \
  --endpoint http://127.0.0.1:8080/mcp \
  --token "$VPS_AGENT_STATIC_TOKEN" \
  --tool file.patch \
  --args "$PATCH_ARGS" >"$EVIDENCE_DIR/patch-replay-${INSTANCE_KEY}.json"

test "$(cat "$SCOPE_ROOT/shared.txt")" = "after"

docker compose exec -T broker /usr/local/bin/vps-agent audit-status \
  >"$EVIDENCE_DIR/audit-status-${INSTANCE_KEY}.json"
docker compose exec -T broker /usr/local/bin/vps-agent audit-tail --limit 100 \
  >"$EVIDENCE_DIR/audit-tail-${INSTANCE_KEY}.json"

END_EPOCH="$(date +%s)"
HOSTNAME="$(hostname)"
STATIC_FP="$(printf '%s' "$VPS_AGENT_STATIC_TOKEN" | sha256sum | awk '{print $1}')"
ADMIN_FP="$(printf '%s' "$VPS_AGENT_ADMIN_TOKEN" | sha256sum | awk '{print $1}')"

python3 - "$INSTANCE_KEY" "$INSTANCE_NAME" "$VPS_AGENT_INSTANCE_ID" "$SUBJECT" "$OPERATION_ID" "$HOSTNAME" "$START_EPOCH" "$END_EPOCH" "$STATIC_FP" "$ADMIN_FP" <<'PY'
import json, pathlib, sys
(key, name, instance_id, subject, operation_id, hostname,
 start_epoch, end_epoch, static_fp, admin_fp) = sys.argv[1:]
root=pathlib.Path("multi-instance-evidence")

def load(name):
    return json.loads((root/name).read_text())

system_info=load(f"system-info-{key}.json")
first=load(f"patch-first-{key}.json")
replay=load(f"patch-replay-{key}.json")
audit_status=load(f"audit-status-{key}.json")
audit_tail=load(f"audit-tail-{key}.json")

sc=system_info.get("structured_content") or {}
assert system_info.get("is_error") is False, system_info
assert sc.get("instance_id") == instance_id, (sc, instance_id)
assert sc.get("instance_name") == name, (sc, name)
assert first.get("is_error") is False, first
assert replay.get("is_error") is False, replay
assert audit_status.get("ok") is True, audit_status
assert audit_status["result"]["valid"] is True, audit_status

events=(audit_tail.get("result") or {}).get("events") or []
matching=[
    e for e in events
    if (e.get("event") or {}).get("subject")==subject
    and (e.get("event") or {}).get("tool")=="file.patch"
]
assert matching, events
for e in matching:
    ev=e["event"]
    assert ev.get("instance_id")==instance_id, ev
    assert ev.get("instance_name")==name, ev

evidence={
    "instance_key": key,
    "instance_name": name,
    "instance_id": instance_id,
    "hostname": hostname,
    "subject": subject,
    "operation_id": operation_id,
    "logical_path": "/opt/vps-agent-multi/shared.txt",
    "static_token_sha256": static_fp,
    "admin_token_sha256": admin_fp,
    "start_epoch": int(start_epoch),
    "end_epoch": int(end_epoch),
    "duration_seconds": int(end_epoch)-int(start_epoch),
    "system_info": sc,
    "audit": audit_status["result"],
    "patch_audit_events": len(matching),
    "final_content": "after",
}
(root/f"evidence-{key}.json").write_text(json.dumps(evidence, indent=2, sort_keys=True)+"\n")
print(json.dumps(evidence, indent=2, sort_keys=True))
PY

echo "MULTI-INSTANCE VPS $INSTANCE_KEY: PASS"
