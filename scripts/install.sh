#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# shellcheck source=scripts/lib/product.sh
source scripts/lib/product.sh

PROFILE=""
SCOPE=""
DOMAIN=""
OPERATOR_EMAIL=""
OPERATOR_USERNAME=""
RUN_AS=""
LANG_OVERRIDE=""
CREATE_SCOPE=0
LOCAL_ONLY=0
ASSUME_YES=0
ACK_WHOLE_HOST=0

usage() {
  vps_agent_block install.usage
}

# Resolve an explicit language before normal argument parsing so help/banner can use it.
if [ -n "${VPS_AGENT_LANG:-}" ]; then
  VPS_AGENT_LANG_EXPLICIT=1
fi
previous=""
for arg in "$@"; do
  if [ "$previous" = "--lang" ]; then
    LANG_OVERRIDE="$arg"
    VPS_AGENT_LANG_EXPLICIT=1
    break
  fi
  previous="$arg"
done
vps_agent_init_language "$LANG_OVERRIDE"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --profile)
      [ "$#" -ge 2 ] || { echo "$(vps_agent_text 'ERROR: --profile needs a value.' 'ERRO: --profile precisa de um valor.')" >&2; exit 2; }
      PROFILE="$2"; shift 2 ;;
    --scope)
      [ "$#" -ge 2 ] || { echo "$(vps_agent_text 'ERROR: --scope needs a value.' 'ERRO: --scope precisa de um valor.')" >&2; exit 2; }
      SCOPE="$2"; shift 2 ;;
    --create-scope) CREATE_SCOPE=1; shift ;;
    --run-as)
      [ "$#" -ge 2 ] || { echo "$(vps_agent_text 'ERROR: --run-as needs a value.' 'ERRO: --run-as precisa de um valor.')" >&2; exit 2; }
      RUN_AS="$2"; shift 2 ;;
    --lang)
      [ "$#" -ge 2 ] || { echo "$(vps_agent_text 'ERROR: --lang needs a value.' 'ERRO: --lang precisa de um valor.')" >&2; exit 2; }
      LANG_OVERRIDE="$2"
      VPS_AGENT_LANG_EXPLICIT=1
      vps_agent_init_language "$LANG_OVERRIDE"
      shift 2 ;;
    --domain)
      [ "$#" -ge 2 ] || { echo "$(vps_agent_text 'ERROR: --domain needs a value.' 'ERRO: --domain precisa de um valor.')" >&2; exit 2; }
      DOMAIN="$2"; shift 2 ;;
    --operator-email)
      [ "$#" -ge 2 ] || { echo "$(vps_agent_text 'ERROR: --operator-email needs a value.' 'ERRO: --operator-email precisa de um valor.')" >&2; exit 2; }
      OPERATOR_EMAIL="$2"; shift 2 ;;
    --operator-username)
      [ "$#" -ge 2 ] || { echo "$(vps_agent_text 'ERROR: --operator-username needs a value.' 'ERRO: --operator-username precisa de um valor.')" >&2; exit 2; }
      OPERATOR_USERNAME="$2"; shift 2 ;;
    --local-only) LOCAL_ONLY=1; shift ;;
    --yes) ASSUME_YES=1; shift ;;
    --ack-whole-host) ACK_WHOLE_HOST=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "$(vps_agent_msg error.unknown_argument "arg=$1")" >&2; usage >&2; exit 2 ;;
  esac
done

vps_agent_confirm_language
vps_agent_banner

vps_agent_step "$(vps_agent_text '1/7 Prerequisites' '1/7 Pré-requisitos')"
if ! command -v python3 >/dev/null 2>&1; then
  cat >&2 <<EOF
$(vps_agent_text   'Python 3 was not found. Portico MCP uses Python 3 for its installation preflight.'   'Python 3 não foi encontrado. O Portico MCP usa Python 3 na verificação de pré-requisitos.')

$(vps_agent_text 'Official installation page:' 'Página oficial de instalação:')
  https://www.python.org/downloads/

$(vps_agent_text 'Install Python 3, then run this installer again.' 'Instale o Python 3 e execute este instalador novamente.')
EOF
  exit 1
fi
preflight_args=()
if [ "$LOCAL_ONLY" -ne 1 ]; then
  preflight_args+=(--public)
fi
python3 scripts/preflight.py "${preflight_args[@]}"

vps_agent_step "$(vps_agent_text '2/7 Scope' '2/7 Escopo')"
if [ -z "$PROFILE" ]; then
  if [ ! -t 0 ]; then
    echo "$(vps_agent_text 'ERROR: --profile is required in non-interactive mode.' 'ERRO: --profile é obrigatório em modo não interativo.')" >&2
    exit 2
  fi
  vps_agent_block install.profile_menu
  printf '%s' "$(vps_agent_msg install.profile_prompt)"
  read -r answer
  case "${answer:-1}" in
    1|standard|Standard|custom|Custom) PROFILE="custom" ;;
    2|project|Project) PROFILE="project" ;;
    3|whole-host|"Whole Host"|"whole host") PROFILE="whole-host" ;;
    *) echo "$(vps_agent_text 'ERROR: invalid profile.' 'ERRO: perfil inválido.')" >&2; exit 2 ;;
  esac
fi

case "$PROFILE" in
  custom|project) ;;
  whole-host) SCOPE="/" ;;
  *) echo "$(vps_agent_text 'ERROR: --profile must be custom, project or whole-host.' 'ERRO: --profile deve ser custom, project ou whole-host.')" >&2; exit 2 ;;
esac

if [ "$PROFILE" = "custom" ] && [ -z "$SCOPE" ]; then
  if [ -t 0 ]; then
    printf '%s' "$(vps_agent_text 'Physical filesystem ceiling [/opt]: ' 'Teto físico do filesystem [/opt]: ')"
    read -r SCOPE
    SCOPE="${SCOPE:-/opt}"
  else
    SCOPE="/opt"
  fi
fi

if [ "$PROFILE" = "project" ] && [ -z "$SCOPE" ]; then
  if [ ! -t 0 ]; then
    echo "$(vps_agent_text 'ERROR: --scope is required for project profile.' 'ERRO: --scope é obrigatório para o perfil project.')" >&2
    exit 2
  fi
  printf '%s' "$(vps_agent_text 'Project root (absolute path): ' 'Raiz do projeto (caminho absoluto): ')"
  read -r SCOPE
  [ -n "$SCOPE" ] || { echo "$(vps_agent_text 'ERROR: project root cannot be empty.' 'ERRO: a raiz do projeto não pode ficar vazia.')" >&2; exit 2; }
fi

if [[ "$SCOPE" != /* ]]; then
  echo "$(vps_agent_text 'ERROR: scope must be an absolute path.' 'ERRO: o escopo deve ser um caminho absoluto.')" >&2
  exit 2
fi

if [ "$SCOPE" != "/" ] && [ ! -d "$SCOPE" ]; then
  if [ "$CREATE_SCOPE" -eq 1 ]; then
    echo "$(vps_agent_msg install.creating_scope "scope=$SCOPE")"
    sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 "$SCOPE"
  elif [ -t 0 ]; then
    printf '%s' "$(vps_agent_msg install.scope_missing_prompt "scope=$SCOPE")"
    read -r create_answer
    case "$create_answer" in
      y|Y|yes|YES|s|S|sim|SIM)
        sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 "$SCOPE"
        ;;
      *)
        echo "$(vps_agent_text 'Installation stopped before startup.' 'Instalação interrompida antes da inicialização.')" >&2
        exit 1
        ;;
    esac
  else
    echo "$(vps_agent_msg error.scope_missing "scope=$SCOPE")" >&2
    exit 1
  fi
fi

if [ -z "$RUN_AS" ]; then
  RUN_AS="${SUDO_USER:-$(id -un)}"
fi
if [ "$RUN_AS" = "root" ]; then
  if [ -t 0 ]; then
    printf '%s' "$(vps_agent_text 'Non-root host user for confined shell jobs: ' 'Usuário não-root para jobs shell confinados: ')"
    read -r RUN_AS
  else
    echo "$(vps_agent_text 'ERROR: pass --run-as USER when installing as root.' 'ERRO: use --run-as USER ao instalar como root.')" >&2
    exit 2
  fi
fi
if [ -z "$RUN_AS" ] || [ "$RUN_AS" = "root" ] || ! id -u "$RUN_AS" >/dev/null 2>&1; then
  echo "$(vps_agent_msg error.invalid_shell_user "user=$RUN_AS")" >&2
  exit 2
fi

init_args=(--scope "$SCOPE" --run-as "$RUN_AS" --lang "$VPS_AGENT_LANG")
if [ "$PROFILE" = "custom" ]; then
  init_args+=(--dynamic-baseline)
fi
VPS_AGENT_LANG_EXPLICIT=1 bash scripts/init.sh "${init_args[@]}"

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
  echo "$(vps_agent_msg install.standard_selected "scope=$SCOPE")"
  echo "$(vps_agent_text     'No project root is authorized yet. Portico may discover immediate folder names under the ceiling, but contents stay locked until explicit approval.'     'Nenhuma raiz de projeto está autorizada ainda. O Portico pode descobrir os nomes das pastas imediatamente abaixo do teto, mas o conteúdo fica bloqueado até aprovação explícita.')"
  echo "$(vps_agent_text     'Protected files such as .env remain locked even inside an authorized project and require a separate temporary approval.'     'Arquivos protegidos como .env continuam trancados mesmo dentro de um projeto autorizado e exigem uma autorização temporária separada.')"
fi

vps_agent_step "$(vps_agent_text '3/7 Effective authority' '3/7 Autoridade efetiva')"
python3 scripts/authority-summary.py

if [ "$PROFILE" = "whole-host" ]; then
  echo
  echo "$(vps_agent_text     'WHOLE HOST exposes / as the physical filesystem ceiling. It does not enable Full or unrestricted network.'     'WHOLE HOST expõe / como teto físico do filesystem. Isso não habilita Full nem rede irrestrita.')"
  if [ "$ASSUME_YES" -eq 1 ] && [ "$ACK_WHOLE_HOST" -ne 1 ]; then
    echo "$(vps_agent_text 'ERROR: non-interactive Whole Host requires --ack-whole-host.' 'ERRO: Whole Host não interativo requer --ack-whole-host.')" >&2
    exit 2
  fi
fi

if [ "$ASSUME_YES" -ne 1 ]; then
  if [ ! -t 0 ]; then
    echo "$(vps_agent_text 'ERROR: --yes is required in non-interactive mode after authority review.' 'ERRO: --yes é obrigatório em modo não interativo após revisar a autoridade.')" >&2
    exit 2
  fi
  printf '%s' "$(vps_agent_text     'Type CONTINUE to start the runtime with the authority shown above: '     'Digite CONTINUAR para iniciar com a autoridade exibida acima: ')"
  read -r confirm
  if vps_agent_is_pt_br; then
    [ "$confirm" = "CONTINUAR" ] || { echo "$(vps_agent_text 'Installation stopped.' 'Instalação interrompida.')"; exit 0; }
  else
    [ "$confirm" = "CONTINUE" ] || { echo "$(vps_agent_text 'Installation stopped.' 'Instalação interrompida.')"; exit 0; }
  fi
fi

vps_agent_step "$(vps_agent_text '4/7 Containers' '4/7 Contêineres')"
compose=(docker compose -f compose.yaml)
if [ "$PROFILE" = "whole-host" ]; then
  compose+=(-f compose.host.yaml)
fi
printf '%s' "$(vps_agent_text 'Running:' 'Executando:')"
printf ' %q' "${compose[@]}"
printf ' up -d --build\n'
"${compose[@]}" up -d --build

vps_agent_step "$(vps_agent_text '5/7 Local verification' '5/7 Verificação local')"
bash scripts/verify.sh

if [ "$LOCAL_ONLY" -eq 1 ]; then
  echo
  echo "$(vps_agent_text 'LOCAL CHECKPOINT COMPLETE' 'CHECKPOINT LOCAL CONCLUÍDO')"
  echo "$(vps_agent_text     'Installation is NOT complete because public OAuth + ChatGPT E2E was skipped.'     'A instalação NÃO está concluída porque OAuth público + E2E com ChatGPT foram ignorados.')"
  exit 0
fi

vps_agent_step "$(vps_agent_text '6/7 Secure public access' '6/7 Acesso público seguro')"
auth_args=()
[ -n "$DOMAIN" ] && auth_args+=(--domain "$DOMAIN")
[ -n "$OPERATOR_EMAIL" ] && auth_args+=(--operator-email "$OPERATOR_EMAIL")
[ -n "$OPERATOR_USERNAME" ] && auth_args+=(--operator-username "$OPERATOR_USERNAME")
printf '%s' "$(vps_agent_text 'Running: bash scripts/setup-integrated-auth.sh' 'Executando: bash scripts/setup-integrated-auth.sh')"
printf ' %q' "${auth_args[@]}"
printf '\n'
bash scripts/setup-integrated-auth.sh "${auth_args[@]}"

vps_agent_step "$(vps_agent_text '7/7 Connect ChatGPT and verify E2E' '7/7 Conectar ao ChatGPT e verificar E2E')"
vps_agent_block install.chatgpt_intro
bash scripts/connect-chatgpt.sh
