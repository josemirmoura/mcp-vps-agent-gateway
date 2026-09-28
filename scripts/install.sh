#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# shellcheck source=scripts/lib/product.sh
source scripts/lib/product.sh

PROFILE=""
SCOPE=""
DOMAIN=""
OPERATOR_EMAIL=""
CREATE_SCOPE=0
LOCAL_ONLY=0
ASSUME_YES=0
ACK_WHOLE_HOST=0

usage() {
  cat <<'EOF'
usage: bash scripts/install.sh [options]

Guided, transparent installation for MCP VPS Agent Gateway.

Profiles:
  project      One project/filesystem root. Safe default.
  custom       One physical ceiling with operator-edited resource policy.
  whole-host   Physical filesystem ceiling "/". Does NOT enable Full.

Options:
  --profile PROFILE         project | custom | whole-host
  --scope PATH              Existing absolute scope root for project/custom.
  --create-scope            Explicitly create a missing scope with sudo install.
  --domain HOSTNAME         Public MCP/OAuth hostname.
  --operator-email EMAIL    Dedicated OAuth operator email.
  --local-only              Stop after local runtime verification.
                            Installation is NOT complete in this mode.
  --yes                     Accept the displayed effective authority non-interactively.
  --ack-whole-host          Required with --yes for whole-host.
  -h, --help                Show this help.

The script orchestrates the existing transparent components. It does not replace
init.sh, Docker Compose, verify.sh, setup-integrated-auth.sh or connect-chatgpt.sh.
It can be rerun safely after an interrupted phase.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --profile)
      [ "$#" -ge 2 ] || { echo "ERROR: --profile needs a value." >&2; exit 2; }
      PROFILE="$2"; shift 2 ;;
    --scope)
      [ "$#" -ge 2 ] || { echo "ERROR: --scope needs a value." >&2; exit 2; }
      SCOPE="$2"; shift 2 ;;
    --create-scope) CREATE_SCOPE=1; shift ;;
    --domain)
      [ "$#" -ge 2 ] || { echo "ERROR: --domain needs a value." >&2; exit 2; }
      DOMAIN="$2"; shift 2 ;;
    --operator-email)
      [ "$#" -ge 2 ] || { echo "ERROR: --operator-email needs a value." >&2; exit 2; }
      OPERATOR_EMAIL="$2"; shift 2 ;;
    --local-only) LOCAL_ONLY=1; shift ;;
    --yes) ASSUME_YES=1; shift ;;
    --ack-whole-host) ACK_WHOLE_HOST=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "ERROR: unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

vps_agent_banner

vps_agent_step "1/7 Environment"
[ "$(uname -s)" = "Linux" ] || { echo "ERROR: Linux is required." >&2; exit 1; }
for cmd in docker git openssl python3 curl; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "ERROR: required command not found: $cmd" >&2; exit 1; }
done
docker compose version >/dev/null
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "ERROR: the supported update path requires a Git checkout." >&2
  echo "Install from a tagged Git release checkout rather than an extracted source snapshot." >&2
  exit 1
fi
vps_agent_note "Linux: $(uname -sr)"
vps_agent_note "Docker: $(docker version --format '{{.Server.Version}}' 2>/dev/null || echo unavailable)"
vps_agent_note "Compose: $(docker compose version --short 2>/dev/null || docker compose version)"

vps_agent_step "2/7 Scope"
if [ -z "$PROFILE" ]; then
  if [ ! -t 0 ]; then
    echo "ERROR: --profile is required in non-interactive mode." >&2
    exit 2
  fi
  cat <<'EOF'
Choose the physical authority profile:
  1) Project    one project root (recommended)
  2) Custom     one physical ceiling, then edit policy for selected resources
  3) Whole Host filesystem ceiling "/" (still not Full)
EOF
  printf 'Profile [1]: '
  read -r answer
  case "${answer:-1}" in
    1|project|Project) PROFILE="project" ;;
    2|custom|Custom) PROFILE="custom" ;;
    3|whole-host|"Whole Host"|"whole host") PROFILE="whole-host" ;;
    *) echo "ERROR: invalid profile." >&2; exit 2 ;;
  esac
fi

case "$PROFILE" in
  project|custom) ;;
  whole-host) SCOPE="/" ;;
  *) echo "ERROR: --profile must be project, custom or whole-host." >&2; exit 2 ;;
esac

if [ "$PROFILE" != "whole-host" ] && [ -z "$SCOPE" ]; then
  if [ ! -t 0 ]; then
    echo "ERROR: --scope is required for $PROFILE in non-interactive mode." >&2
    exit 2
  fi
  default_scope="/opt/vps-agent-sandbox"
  if [ "$PROFILE" = "custom" ]; then
    default_scope="/opt"
  fi
  printf 'Physical filesystem ceiling [%s]: ' "$default_scope"
  read -r SCOPE
  SCOPE="${SCOPE:-$default_scope}"
fi

if [[ "$SCOPE" != /* ]]; then
  echo "ERROR: scope must be an absolute path." >&2
  exit 2
fi

if [ "$SCOPE" != "/" ] && [ ! -d "$SCOPE" ]; then
  if [ "$CREATE_SCOPE" -eq 1 ]; then
    echo "Creating $SCOPE with an explicit sudo command:"
    echo "  sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 "$SCOPE""
    sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 "$SCOPE"
  elif [ -t 0 ]; then
    printf '%s does not exist. Create it now with sudo install? [y/N]: ' "$SCOPE"
    read -r create_answer
    case "$create_answer" in
      y|Y|yes|YES)
        echo "Running: sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 "$SCOPE""
        sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 "$SCOPE"
        ;;
      *)
        echo "Nothing started. Create the directory or rerun with --create-scope." >&2
        exit 1
        ;;
    esac
  else
    echo "ERROR: scope does not exist: $SCOPE" >&2
    echo "Create it explicitly or rerun with --create-scope." >&2
    exit 1
  fi
fi

bash scripts/init.sh --scope "$SCOPE"

python3 - .env "$PROFILE" <<'PY'
import pathlib, sys
path=pathlib.Path(sys.argv[1])
profile=sys.argv[2]
key="VPS_AGENT_WHOLE_HOST"
value="1" if profile=="whole-host" else "0"
lines=path.read_text().splitlines()
out=[]
seen=False
for line in lines:
    if line.startswith(key+"="):
        out.append(f"{key}={value}")
        seen=True
    else:
        out.append(line)
if not seen:
    out.append(f"{key}={value}")
path.write_text("\n".join(out)+"\n")
PY
chmod 600 .env

if [ "$PROFILE" = "custom" ]; then
  echo
  echo "Custom profile selected."
  echo "The physical ceiling is $SCOPE. Refine config/policy.yaml to the exact"
  echo "filesystem roots, services, Docker resources and capabilities you want."
  if [ -t 0 ]; then
    printf 'Edit config/policy.yaml now, then press Enter to continue: '
    read -r _
  else
    echo "Non-interactive Custom assumes config/policy.yaml was prepared before this run."
  fi
fi

vps_agent_step "3/7 Effective authority"
python3 scripts/authority-summary.py

if [ "$PROFILE" = "whole-host" ]; then
  echo
  echo "WHOLE HOST exposes / as the Broker's physical filesystem ceiling."
  echo "It does not enable Full or unrestricted network, but it is still broad authority."
  if [ "$ASSUME_YES" -eq 1 ] && [ "$ACK_WHOLE_HOST" -ne 1 ]; then
    echo "ERROR: non-interactive Whole Host requires --ack-whole-host." >&2
    exit 2
  fi
fi

if [ "$ASSUME_YES" -ne 1 ]; then
  if [ ! -t 0 ]; then
    echo "ERROR: --yes is required in non-interactive mode after authority review." >&2
    exit 2
  fi
  printf 'Type CONTINUE to start the runtime with the authority shown above: '
  read -r confirm
  [ "$confirm" = "CONTINUE" ] || { echo "Installation stopped before startup."; exit 0; }
fi

vps_agent_step "4/7 Containers"
compose=(docker compose -f compose.yaml)
if [ "$PROFILE" = "whole-host" ]; then
  compose+=(-f compose.host.yaml)
fi
printf 'Running:'
printf ' %q' "${compose[@]}"
printf ' up -d --build\n'
"${compose[@]}" up -d --build

vps_agent_step "5/7 Local verification"
bash scripts/verify.sh

if [ "$LOCAL_ONLY" -eq 1 ]; then
  echo
  echo "LOCAL CHECKPOINT COMPLETE"
  echo "Installation is NOT complete because public OAuth + ChatGPT E2E was skipped."
  exit 0
fi

vps_agent_step "6/7 Secure public access"
auth_args=()
[ -n "$DOMAIN" ] && auth_args+=(--domain "$DOMAIN")
[ -n "$OPERATOR_EMAIL" ] && auth_args+=(--operator-email "$OPERATOR_EMAIL")
printf 'Running: bash scripts/setup-integrated-auth.sh'
printf ' %q' "${auth_args[@]}"
printf '\n'
bash scripts/setup-integrated-auth.sh "${auth_args[@]}"

vps_agent_step "7/7 Connect ChatGPT and verify E2E"
cat <<'EOF'
The next script shows the current ChatGPT connection steps and waits for a new
authenticated system.info call. Installation completes only when the Broker
observes that real call and the audit chain remains valid.
EOF
bash scripts/connect-chatgpt.sh
