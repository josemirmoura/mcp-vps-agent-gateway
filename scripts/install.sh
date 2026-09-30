#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# shellcheck source=scripts/lib/product.sh
source scripts/lib/product.sh

PROFILE=""
SCOPE=""
DOMAIN=""
OPERATOR_EMAIL=""
RUN_AS=""
LANG_OVERRIDE=""
CREATE_SCOPE=0
LOCAL_ONLY=0
ASSUME_YES=0
ACK_WHOLE_HOST=0

usage() {
  if vps_agent_is_pt_br; then
    cat <<'EOF'
uso: bash scripts/install.sh [opções]

Instalação guiada e transparente do Portico MCP.

Perfis:
  custom       Padrão recomendado: teto /opt e raízes aprovadas dinamicamente.
  project      Trava o Portico MCP em uma única raiz de projeto.
  whole-host   Teto físico "/". Não habilita Full automaticamente.

Opções:
  --profile PROFILE         custom | project | whole-host
  --scope PATH              Teto/raiz existente para custom/project.
  --create-scope            Cria explicitamente uma raiz ausente.
  --run-as USER             Usuário não-root dos jobs shell confinados.
  --lang LANG               pt-BR | en
  --domain HOSTNAME         Hostname público MCP/OAuth.
  --operator-email EMAIL    E-mail dedicado do operador OAuth.
  --local-only              Para após a verificação local.
  --yes                     Aceita a autoridade exibida sem prompt.
  --ack-whole-host          Obrigatório com --yes em whole-host.
  -h, --help                Mostra esta ajuda.
EOF
  else
    cat <<'EOF'
usage: bash scripts/install.sh [options]

Guided, transparent installation for Portico MCP.

Profiles:
  custom       Recommended default: /opt ceiling with dynamic root approval.
  project      Lock Portico MCP to one project/filesystem root.
  whole-host   Physical filesystem ceiling "/". Does not enable Full.

Options:
  --profile PROFILE         custom | project | whole-host
  --scope PATH              Existing ceiling/root for custom/project.
  --create-scope            Explicitly create a missing scope.
  --run-as USER             Non-root host user for confined shell jobs.
  --lang LANG               pt-BR | en
  --domain HOSTNAME         Public MCP/OAuth hostname.
  --operator-email EMAIL    Dedicated OAuth operator email.
  --local-only              Stop after local runtime verification.
  --yes                     Accept the displayed effective authority.
  --ack-whole-host          Required with --yes for whole-host.
  -h, --help                Show this help.
EOF
  fi
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
      [ "$#" -ge 2 ] || { echo "ERROR: --profile needs a value." >&2; exit 2; }
      PROFILE="$2"; shift 2 ;;
    --scope)
      [ "$#" -ge 2 ] || { echo "ERROR: --scope needs a value." >&2; exit 2; }
      SCOPE="$2"; shift 2 ;;
    --create-scope) CREATE_SCOPE=1; shift ;;
    --run-as)
      [ "$#" -ge 2 ] || { echo "ERROR: --run-as needs a value." >&2; exit 2; }
      RUN_AS="$2"; shift 2 ;;
    --lang)
      [ "$#" -ge 2 ] || { echo "ERROR: --lang needs a value." >&2; exit 2; }
      LANG_OVERRIDE="$2"
      VPS_AGENT_LANG_EXPLICIT=1
      vps_agent_init_language "$LANG_OVERRIDE"
      shift 2 ;;
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
    *) echo "$(vps_agent_text "ERROR: unknown argument: $1" "ERRO: argumento desconhecido: $1")" >&2; usage >&2; exit 2 ;;
  esac
done

vps_agent_confirm_language
vps_agent_banner

vps_agent_step "$(vps_agent_text '1/7 Environment' '1/7 Ambiente')"
[ "$(uname -s)" = "Linux" ] || { echo "$(vps_agent_text 'ERROR: Linux is required.' 'ERRO: Linux é obrigatório.')" >&2; exit 1; }
for cmd in docker git openssl python3 curl; do
  command -v "$cmd" >/dev/null 2>&1 || {
    echo "$(vps_agent_text "ERROR: required command not found: $cmd" "ERRO: comando obrigatório não encontrado: $cmd")" >&2
    exit 1
  }
done
docker compose version >/dev/null
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "$(vps_agent_text     'ERROR: the supported update path requires a Git checkout.'     'ERRO: o fluxo de atualização suportado requer um checkout Git.')" >&2
  exit 1
fi
vps_agent_note "Linux: $(uname -sr)"
vps_agent_note "Docker: $(docker version --format '{{.Server.Version}}' 2>/dev/null || echo unavailable)"
vps_agent_note "Compose: $(docker compose version --short 2>/dev/null || docker compose version)"

vps_agent_step "$(vps_agent_text '2/7 Scope' '2/7 Escopo')"
if [ -z "$PROFILE" ]; then
  if [ ! -t 0 ]; then
    echo "$(vps_agent_text 'ERROR: --profile is required in non-interactive mode.' 'ERRO: --profile é obrigatório em modo não interativo.')" >&2
    exit 2
  fi
  if vps_agent_is_pt_br; then
    cat <<'EOF'
Escolha o modelo de autoridade física:
  1) Standard    teto /opt; projetos são autorizados depois (recomendado)
  2) Project     uma única raiz de projeto
  3) Whole Host  teto "/" para o filesystem (ainda não é Full)
EOF
    printf 'Perfil [1]: '
  else
    cat <<'EOF'
Choose the physical authority model:
  1) Standard    /opt ceiling; approve project roots later (recommended)
  2) Project     one project/filesystem root
  3) Whole Host  filesystem ceiling "/" (still not Full)
EOF
    printf 'Profile [1]: '
  fi
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
    echo "$(vps_agent_text "Creating $SCOPE with sudo install." "Criando $SCOPE com sudo install.")"
    sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 "$SCOPE"
  elif [ -t 0 ]; then
    printf '%s' "$(vps_agent_text "$SCOPE does not exist. Create it now? [y/N]: " "$SCOPE não existe. Criar agora? [s/N]: ")"
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
    echo "$(vps_agent_text "ERROR: scope does not exist: $SCOPE" "ERRO: o escopo não existe: $SCOPE")" >&2
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
  echo "$(vps_agent_text "ERROR: invalid non-root shell user: $RUN_AS" "ERRO: usuário não-root inválido para shell: $RUN_AS")" >&2
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
  echo "$(vps_agent_text     "Standard profile selected. Physical ceiling: $SCOPE."     "Perfil Standard selecionado. Teto físico: $SCOPE.")"
  echo "$(vps_agent_text     'No project root is authorized yet. Portico MCP will request each project root later through explicit approval.'     'Nenhuma raiz de projeto está autorizada ainda. O Portico MCP solicitará cada projeto depois por aprovação explícita.')"
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
    [ "$confirm" = "CONTINUE" ] || { echo "Installation stopped."; exit 0; }
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
printf 'Running: bash scripts/setup-integrated-auth.sh'
printf ' %q' "${auth_args[@]}"
printf '\n'
bash scripts/setup-integrated-auth.sh "${auth_args[@]}"

vps_agent_step "$(vps_agent_text '7/7 Connect ChatGPT and verify E2E' '7/7 Conectar ao ChatGPT e verificar E2E')"
if vps_agent_is_pt_br; then
  cat <<'EOF'
O próximo script mostra as etapas atuais de conexão no ChatGPT e aguarda uma
chamada system.info autenticada. A instalação só termina quando o Broker
observar essa chamada real e a cadeia de auditoria continuar válida.
EOF
else
  cat <<'EOF'
The next script shows the current ChatGPT connection steps and waits for a new
authenticated system.info call. Installation completes only when the Broker
observes that real call and the audit chain remains valid.
EOF
fi
bash scripts/connect-chatgpt.sh
