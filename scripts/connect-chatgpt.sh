#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# shellcheck source=scripts/lib/product.sh
source scripts/lib/product.sh
vps_agent_init_language ""

if [ ! -f .env ]; then
  echo "$(vps_agent_text 'Missing .env. Run scripts/init.sh first.' 'Arquivo .env ausente. Execute scripts/init.sh primeiro.')" >&2
  exit 1
fi
set -a
. ./.env
set +a

PUBLIC_URL="${VPS_AGENT_PUBLIC_URL:-}"
SUBJECT="${VPS_AGENT_SUBJECT:-operator}"
MARKER="state/integrated-auth.json"

if [ -z "$PUBLIC_URL" ]; then
  echo "$(vps_agent_text 'VPS_AGENT_PUBLIC_URL is not configured.' 'VPS_AGENT_PUBLIC_URL não está configurada.')" >&2
  exit 1
fi

if [ "${VPS_AGENT_AUTH_MODE:-static}" != "integrated" ]; then
  echo "$(vps_agent_text     'ChatGPT public connection requires the integrated OAuth path.'     'A conexão pública com o ChatGPT requer o fluxo OAuth integrado.')" >&2
  exit 1
fi

operator_username="vps-operator"
operator_email=""
if [ -s "$MARKER" ]; then
  readarray -t identity < <(python3 - "$MARKER" <<'PY'
import json,sys
try:
    data=json.load(open(sys.argv[1]))
except Exception:
    data={}
print(data.get("username","vps-operator"))
print(data.get("email",""))
PY
  )
  operator_username="${identity[0]:-vps-operator}"
  operator_email="${identity[1]:-}"
fi

bash ./scripts/verify-public.sh

wait_tool=(docker compose exec -T broker /usr/local/bin/vps-agent wait-tool --subject "$SUBJECT" --tool system.info)
BASELINE="$("${wait_tool[@]}" --baseline-only)"
[[ "$BASELINE" =~ ^[0-9]+$ ]] || {
  echo "$(vps_agent_text 'ERROR: could not establish the audit verification baseline.' 'ERRO: não foi possível estabelecer a linha de base da auditoria.')" >&2
  exit 1
}

show_tutorial() {
  local email_line=""
  if [ -n "$operator_email" ]; then
    email_line="$(vps_agent_msg connect.email_line "email=$operator_email")"$'\n'
  fi
  echo
  vps_agent_msg connect.tutorial     "public_url=$PUBLIC_URL"     "username=$operator_username"     "email_line=$email_line"
  printf '\n'
}
verify_now() {
  "${wait_tool[@]}" --after-seq "$BASELINE" --timeout 1s --poll 200ms --quiet
}

show_tutorial

if [ ! -t 0 ]; then
  echo "$(vps_agent_text     'Waiting for the real audited ChatGPT call in non-interactive mode...'     'Aguardando a chamada real e auditada do ChatGPT em modo não interativo...')"
  "${wait_tool[@]}" --after-seq "$BASELINE" --timeout "${VPS_AGENT_CHATGPT_VERIFY_TIMEOUT:-10m}"
else
  while :; do
    echo
    printf '%s' "$(vps_agent_msg connect.prompt)"
    read -r answer

    case "$answer" in
      t|T)
        show_tutorial
        continue
        ;;
      s|S|q|Q)
        echo "$(vps_agent_text           'Leaving without completing installation. Run scripts/connect-chatgpt.sh again when ready.'           'Saindo sem concluir a instalação. Execute scripts/connect-chatgpt.sh novamente quando estiver pronto.')"
        exit 2
        ;;
      "")
        if verify_now; then
          break
        fi
        echo
        vps_agent_block connect.not_detected
        ;;
      *)
        echo "$(vps_agent_msg connect.invalid_choice)"
        ;;
    esac
  done
fi

mkdir -p state
printf '{"verified_at":"%s","endpoint":"%s","subject":"%s","after_audit_seq":%s}\n' \
  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$PUBLIC_URL" "$SUBJECT" "$BASELINE" \
  > state/chatgpt-verified.json
chmod 600 state/chatgpt-verified.json

echo
echo "$(vps_agent_text 'CHATGPT WEB CONNECTION VERIFIED' 'CONEXÃO COM O CHATGPT WEB VERIFICADA')"
echo "$(vps_agent_text 'INSTALLATION COMPLETE' 'INSTALAÇÃO CONCLUÍDA')"
