#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

usage() {
  cat <<'EOF'
usage: bash scripts/init.sh [--scope /absolute/existing/path] [--migrate-policy-root]

Options:
  --scope PATH   Set the physical Scoped filesystem ceiling in .env.
                 Existing operator policy roots are preserved by default.
  --migrate-policy-root
                 Replace references to the previous scope in policy.yaml.
                 Intended for an explicit project-root move, not for widening
                 the physical ceiling around existing delegated roots.
  -h, --help     Show this help.
EOF
}

SCOPE_OVERRIDE=""
MIGRATE_POLICY_ROOT=0
POLICY_CREATED=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --scope)
      if [ "$#" -lt 2 ] || [ -z "$2" ]; then
        echo "ERROR: --scope requires an absolute path." >&2
        exit 2
      fi
      SCOPE_OVERRIDE="$2"
      shift 2
      ;;
    --migrate-policy-root)
      MIGRATE_POLICY_ROOT=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "ERROR: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker is required." >&2
  exit 1
fi
docker compose version >/dev/null

if [ ! -f .env ]; then
  cp .env.example .env
  ADMIN_TOKEN="$(openssl rand -hex 32)"
  MCP_TOKEN="$(openssl rand -hex 32)"
  INSTANCE_ID="$(openssl rand -hex 16)"
  sed -i "s/CHANGE_ME_ADMIN_TOKEN/$ADMIN_TOKEN/" .env
  sed -i "s/CHANGE_ME_MCP_TOKEN/$MCP_TOKEN/" .env
  sed -i "s/CHANGE_ME_INSTANCE_ID/$INSTANCE_ID/" .env
  chmod 600 .env
  echo "Created .env with random tokens."
else
  echo ".env already exists; preserving existing secrets and identity."
fi

mkdir -p state
chmod 700 state

if [ ! -f config/policy.yaml ]; then
  cp config/policy.example.yaml config/policy.yaml
  chmod 600 config/policy.yaml
  POLICY_CREATED=1
  echo "Created config/policy.yaml from the versioned template."
else
  echo "config/policy.yaml already exists; preserving operator policy."
fi

if grep -q 'CHANGE_ME_' .env; then
  echo "ERROR: .env still contains CHANGE_ME values." >&2
  exit 1
fi

set -a
. ./.env
set +a
OLD_SCOPE_ROOT="${VPS_AGENT_SCOPE_ROOT:-}"

if [ -n "$SCOPE_OVERRIDE" ]; then
  if [[ "$SCOPE_OVERRIDE" != /* ]]; then
    echo "ERROR: --scope must be an absolute path: $SCOPE_OVERRIDE" >&2
    exit 1
  fi

  NORMALIZED_OVERRIDE="$(python3 - "$SCOPE_OVERRIDE" <<'PY'
import os
import sys
print(os.path.normpath(sys.argv[1]))
PY
)"
  if [ "$NORMALIZED_OVERRIDE" != "$SCOPE_OVERRIDE" ]; then
    echo "ERROR: --scope must be canonical (no '..', '.' or trailing slash): $SCOPE_OVERRIDE" >&2
    exit 1
  fi

  python3 - ".env" "$SCOPE_OVERRIDE" <<'PY'
import pathlib
import shlex
import sys

path = pathlib.Path(sys.argv[1])
value = sys.argv[2]
key = "VPS_AGENT_SCOPE_ROOT"
lines = path.read_text().splitlines()
replacement = f"{key}={shlex.quote(value)}"
out = []
replaced = False
for line in lines:
    if line.startswith(key + "="):
        out.append(replacement)
        replaced = True
    else:
        out.append(line)
if not replaced:
    out.append(replacement)
path.write_text("\n".join(out) + "\n")
PY

  if [ -n "$OLD_SCOPE_ROOT" ] && [ "$OLD_SCOPE_ROOT" != "$SCOPE_OVERRIDE" ]; then
    if [ "$POLICY_CREATED" -eq 1 ] || [ "$MIGRATE_POLICY_ROOT" -eq 1 ]; then
      python3 - "config/policy.yaml" "$OLD_SCOPE_ROOT" "$SCOPE_OVERRIDE" <<'PY'
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
old = sys.argv[2]
new = sys.argv[3]
text = path.read_text()
if old != "/" and old in text:
    text = text.replace(old, new)
    path.write_text(text)
    print(f"Migrated policy paths from {old} to {new}.")
else:
    print("Policy did not contain the previous scoped root; leaving policy paths unchanged.")
PY
    else
      echo "Physical ceiling changed from $OLD_SCOPE_ROOT to $SCOPE_OVERRIDE."
      echo "Existing logical policy roots were preserved."
      echo "Use scripts/delegate-root.sh to add or revoke delegated roots."
    fi
  fi

  VPS_AGENT_SCOPE_ROOT="$SCOPE_OVERRIDE"
fi

SCOPE_ROOT="${VPS_AGENT_SCOPE_ROOT:-}"
if [ -z "$SCOPE_ROOT" ]; then
  echo "ERROR: set VPS_AGENT_SCOPE_ROOT in .env or pass --scope /absolute/path." >&2
  exit 1
fi
if [[ "$SCOPE_ROOT" != /* ]]; then
  echo "ERROR: VPS_AGENT_SCOPE_ROOT must be an absolute path." >&2
  exit 1
fi
NORMALIZED_SCOPE_ROOT="$(python3 - "$SCOPE_ROOT" <<'PY'
import os
import sys
print(os.path.normpath(sys.argv[1]))
PY
)"
if [ "$NORMALIZED_SCOPE_ROOT" != "$SCOPE_ROOT" ]; then
  echo "ERROR: VPS_AGENT_SCOPE_ROOT must be canonical (no '..', '.' or trailing slash): $SCOPE_ROOT" >&2
  exit 1
fi

if [ "$SCOPE_ROOT" = "/" ]; then
  echo "NOTE: whole-host authority requires the explicit compose.host.yaml override."
else
  CHECK_PATH="$SCOPE_ROOT"
  while [ "$CHECK_PATH" != "/" ]; do
    if [ -L "$CHECK_PATH" ]; then
      echo "ERROR: VPS_AGENT_SCOPE_ROOT may not traverse symlinks: $CHECK_PATH" >&2
      exit 1
    fi
    CHECK_PATH="$(dirname "$CHECK_PATH")"
  done

  if [ ! -d "$SCOPE_ROOT" ]; then
    cat >&2 <<EOF
ERROR: VPS_AGENT_SCOPE_ROOT does not exist: $SCOPE_ROOT

Create it first, for example:
  sudo install -d -o "\$USER" -g "\$(id -gn)" -m 0750 "$SCOPE_ROOT"

Or choose an existing directory:
  bash scripts/init.sh --scope /absolute/existing/path

Nothing has been started.
EOF
    exit 1
  fi
fi

docker compose config -q

cat <<EOF

Bootstrap ready.

Scoped filesystem ceiling:
  $SCOPE_ROOT

Optional: review .env and config/policy.yaml before startup.

Start Scoped mode:
  docker compose up -d --build

Whole-host is a separate explicit override:
  docker compose -f compose.yaml -f compose.host.yaml up -d --build

Verify:
  bash scripts/verify.sh

EOF
