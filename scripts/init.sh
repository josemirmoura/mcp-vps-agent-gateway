#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# shellcheck source=scripts/lib/product.sh
source scripts/lib/product.sh

usage() {
  vps_agent_block init.usage
}

SCOPE_OVERRIDE=""
MIGRATE_POLICY_ROOT=0
DYNAMIC_BASELINE=0
RUN_AS_OVERRIDE=""
LANG_OVERRIDE=""
POLICY_CREATED=0
POLICY_PRISTINE_TEMPLATE=0

# Resolve --lang first so --help can be localized regardless of argument order.
previous=""
for arg in "$@"; do
  if [ "$previous" = "--lang" ]; then
    LANG_OVERRIDE="$arg"
    break
  fi
  previous="$arg"
done
if [ -n "$LANG_OVERRIDE" ]; then
  VPS_AGENT_LANG_EXPLICIT=1
fi
vps_agent_init_language "$LANG_OVERRIDE"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --scope)
      if [ "$#" -lt 2 ] || [ -z "$2" ]; then
        echo "$(vps_agent_text 'ERROR: --scope requires an absolute path.' 'ERRO: --scope requer um caminho absoluto.')" >&2
        exit 2
      fi
      SCOPE_OVERRIDE="$2"
      shift 2
      ;;
    --dynamic-baseline)
      DYNAMIC_BASELINE=1
      shift
      ;;
    --run-as)
      if [ "$#" -lt 2 ] || [ -z "$2" ]; then
        echo "$(vps_agent_text 'ERROR: --run-as requires a username.' 'ERRO: --run-as requer um nome de usuário.')" >&2
        exit 2
      fi
      RUN_AS_OVERRIDE="$2"
      shift 2
      ;;
    --migrate-policy-root)
      MIGRATE_POLICY_ROOT=1
      shift
      ;;
    --lang)
      [ "$#" -ge 2 ] || { echo "$(vps_agent_text 'ERROR: --lang needs a value.' 'ERRO: --lang precisa de um valor.')" >&2; exit 2; }
      LANG_OVERRIDE="$2"
      VPS_AGENT_LANG_EXPLICIT=1
      vps_agent_init_language "$LANG_OVERRIDE"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "$(vps_agent_msg error.unknown_argument "arg=$1")" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if ! command -v docker >/dev/null 2>&1; then
  echo "$(vps_agent_text 'Docker is required.' 'Docker é obrigatório.')" >&2
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
  echo "$(vps_agent_text 'Created .env with random tokens.' 'Criado .env com tokens aleatórios.')"
else
  echo "$(vps_agent_text '.env already exists; preserving existing secrets and identity.' '.env já existe; preservando segredos e identidade existentes.')"
fi

mkdir -p state
chmod 700 state

# Persist the resolved human-facing locale so commands executed later keep the
# same language even when the terminal/SSH locale differs.
python3 - ".env" "$VPS_AGENT_LANG" <<'PY'
import pathlib
import shlex
import sys

path = pathlib.Path(sys.argv[1])
value = sys.argv[2]
key = "VPS_AGENT_LANG"
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

if [ ! -f config/policy.yaml ]; then
  cp config/policy.example.yaml config/policy.yaml
  chmod 600 config/policy.yaml
  POLICY_CREATED=1
  echo "$(vps_agent_text 'Created config/policy.yaml from the versioned template.' 'Criado config/policy.yaml a partir do modelo versionado.')"
else
  echo "$(vps_agent_text 'config/policy.yaml already exists; preserving operator policy.' 'config/policy.yaml já existe; preservando a policy do operador.')"
fi

if cmp -s config/policy.yaml config/policy.example.yaml; then
  POLICY_PRISTINE_TEMPLATE=1
fi

if grep -q 'CHANGE_ME_' .env; then
  echo "$(vps_agent_text 'ERROR: .env still contains CHANGE_ME values.' 'ERRO: .env ainda contém valores CHANGE_ME.')" >&2
  exit 1
fi

set -a
. ./.env
set +a
OLD_SCOPE_ROOT="${VPS_AGENT_SCOPE_ROOT:-}"

if [ -n "$SCOPE_OVERRIDE" ]; then
  if [[ "$SCOPE_OVERRIDE" != /* ]]; then
    echo "$(vps_agent_msg error.scope_absolute "scope=$SCOPE_OVERRIDE")" >&2
    exit 1
  fi

  NORMALIZED_OVERRIDE="$(python3 - "$SCOPE_OVERRIDE" <<'PY'
import os
import sys
print(os.path.normpath(sys.argv[1]))
PY
)"
  if [ "$NORMALIZED_OVERRIDE" != "$SCOPE_OVERRIDE" ]; then
    echo "$(vps_agent_msg error.scope_canonical "scope=$SCOPE_OVERRIDE")" >&2
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
    if [ "$DYNAMIC_BASELINE" -eq 1 ] && { [ "$POLICY_CREATED" -eq 1 ] || [ "$POLICY_PRISTINE_TEMPLATE" -eq 1 ]; }; then
      echo "$(vps_agent_msg init.ceiling_changed_empty "old=$OLD_SCOPE_ROOT" "new=$SCOPE_OVERRIDE")"
    elif [ "$POLICY_CREATED" -eq 1 ] || [ "$POLICY_PRISTINE_TEMPLATE" -eq 1 ] || [ "$MIGRATE_POLICY_ROOT" -eq 1 ]; then
      python3 - "config/policy.yaml" "$OLD_SCOPE_ROOT" "$SCOPE_OVERRIDE" <<'PY'
import os
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
old = sys.argv[2]
new = sys.argv[3]
text = path.read_text()
if old != "/" and old in text:
    text = text.replace(old, new)
    path.write_text(text)
    lang=os.environ.get("VPS_AGENT_LANG","en")
    print((f"Caminhos da policy migrados de {old} para {new}." if lang=="pt-BR" else f"Migrated policy paths from {old} to {new}."))
else:
    lang=os.environ.get("VPS_AGENT_LANG","en")
    print("A policy não continha a raiz Scoped anterior; mantendo os caminhos inalterados." if lang=="pt-BR" else "Policy did not contain the previous scoped root; leaving policy paths unchanged.")
PY
    else
      echo "$(vps_agent_msg init.ceiling_changed_preserved "old=$OLD_SCOPE_ROOT" "new=$SCOPE_OVERRIDE")"
    fi
  fi

  VPS_AGENT_SCOPE_ROOT="$SCOPE_OVERRIDE"
fi

SCOPE_ROOT="${VPS_AGENT_SCOPE_ROOT:-}"
if [ -z "$SCOPE_ROOT" ]; then
  echo "$(vps_agent_text 'ERROR: set VPS_AGENT_SCOPE_ROOT in .env or pass --scope /absolute/path.' 'ERRO: defina VPS_AGENT_SCOPE_ROOT no .env ou use --scope /caminho/absoluto.')" >&2
  exit 1
fi
if [[ "$SCOPE_ROOT" != /* ]]; then
  echo "$(vps_agent_text 'ERROR: VPS_AGENT_SCOPE_ROOT must be an absolute path.' 'ERRO: VPS_AGENT_SCOPE_ROOT deve ser um caminho absoluto.')" >&2
  exit 1
fi
NORMALIZED_SCOPE_ROOT="$(python3 - "$SCOPE_ROOT" <<'PY'
import os
import sys
print(os.path.normpath(sys.argv[1]))
PY
)"
if [ "$NORMALIZED_SCOPE_ROOT" != "$SCOPE_ROOT" ]; then
  echo "$(vps_agent_text 'ERROR: VPS_AGENT_SCOPE_ROOT must be canonical (no dots or trailing slash).' 'ERRO: VPS_AGENT_SCOPE_ROOT deve ser canônico (sem pontos ou barra final).')" >&2
  exit 1
fi

if [ "$SCOPE_ROOT" = "/" ]; then
  echo "$(vps_agent_text 'NOTE: whole-host authority requires the explicit compose.host.yaml override.' 'NOTA: autoridade de host inteiro requer o override explícito compose.host.yaml.')"
else
  CHECK_PATH="$SCOPE_ROOT"
  while [ "$CHECK_PATH" != "/" ]; do
    if [ -L "$CHECK_PATH" ]; then
      echo "$(vps_agent_msg error.scope_symlink "path=$CHECK_PATH")" >&2
      exit 1
    fi
    CHECK_PATH="$(dirname "$CHECK_PATH")"
  done

  if [ ! -d "$SCOPE_ROOT" ]; then
    cat >&2 <<EOF
$(vps_agent_msg error.scope_root_missing "path=$SCOPE_ROOT")

$(vps_agent_text "Create it first, or choose an existing directory." "Crie o diretório primeiro ou escolha um diretório existente.")
EOF
    exit 1
  fi
fi

if [ "$DYNAMIC_BASELINE" -eq 1 ] && { [ "$POLICY_CREATED" -eq 1 ] || [ "$POLICY_PRISTINE_TEMPLATE" -eq 1 ]; }; then
  python3 - "config/policy.yaml" <<'PY'
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
lines = path.read_text().splitlines()
targets = {
    ("filesystem", "read"),
    ("filesystem", "write"),
    ("shell", "cwd_roots"),
}
out = []
section = None
i = 0
while i < len(lines):
    line = lines[i]
    if line and not line.startswith(" ") and line.endswith(":"):
        section = line[:-1]
    stripped = line.strip()
    if line.startswith("  ") and not line.startswith("    ") and stripped.endswith(":"):
        key = stripped[:-1]
        if (section, key) in targets:
            out.append(f"  {key}: []")
            i += 1
            while i < len(lines) and lines[i].startswith("    - "):
                i += 1
            continue
    out.append(line)
    i += 1
path.write_text("\n".join(out) + "\n")
PY
  echo "$(vps_agent_text     'Dynamic baseline enabled: no static project roots are authorized.'     'Baseline dinâmica ativada: nenhuma raiz de projeto está autorizada estaticamente.')"
fi

if [ -z "$RUN_AS_OVERRIDE" ]; then
  candidate="${SUDO_USER:-$(id -un)}"
  if [ "$candidate" != "root" ]; then
    RUN_AS_OVERRIDE="$candidate"
  fi
fi

if [ -n "$RUN_AS_OVERRIDE" ]; then
  if [ "$RUN_AS_OVERRIDE" = "root" ]; then
    echo "$(vps_agent_text 'ERROR: normal scoped shell jobs may not run as root.' 'ERRO: jobs shell normais em modo Scoped não podem executar como root.')" >&2
    exit 1
  fi
  if ! id -u "$RUN_AS_OVERRIDE" >/dev/null 2>&1; then
    echo "$(vps_agent_msg error.shell_user_missing "user=$RUN_AS_OVERRIDE")" >&2
    exit 1
  fi
  python3 - "config/policy.yaml" "$RUN_AS_OVERRIDE" <<'PY'
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
user = sys.argv[2]
lines = path.read_text().splitlines()
out = []
section = None
changed = False
for line in lines:
    if line and not line.startswith(" ") and line.endswith(":"):
        section = line[:-1]
    if section == "shell" and line.startswith("  run_as:"):
        out.append(f"  run_as: {user}")
        changed = True
    else:
        out.append(line)
if not changed:
    raise SystemExit("shell.run_as not found in policy")
path.write_text("\n".join(out) + "\n")
PY
fi

docker compose config -q

cat <<EOF

$(vps_agent_text 'Bootstrap ready.' 'Bootstrap pronto.')

$(vps_agent_text 'Physical filesystem ceiling:' 'Teto físico do filesystem:')
  $SCOPE_ROOT

$(vps_agent_text 'Confined shell user:' 'Usuário do shell confinado:')
  ${RUN_AS_OVERRIDE:-vps-agent-exec}

$(vps_agent_text 'Start Scoped mode:' 'Iniciar modo Scoped:')
  docker compose up -d --build

$(vps_agent_text 'Verify:' 'Verificar:')
  bash scripts/verify.sh

EOF
