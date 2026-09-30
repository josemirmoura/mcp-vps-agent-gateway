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
  echo
  if vps_agent_is_pt_br; then
    cat <<EOF
=== Conectar o Portico MCP ao ChatGPT ===

A VPS está pronta. Agora falta conectar o Portico MCP ao ChatGPT.
Esta próxima parte acontece no ChatGPT Web.

Endereço MCP:
  $PUBLIC_URL

1. Abra o ChatGPT no navegador.
2. Ative o Modo de Desenvolvedor, caso essa opção ainda não esteja ativa.
3. Vá em Configurações > Aplicativos e escolha criar um novo app MCP.
4. Quando o ChatGPT pedir o endereço do MCP, use exatamente:

   $PUBLIC_URL

5. Escolha OAuth como autenticação.
6. Na tela de login do Portico MCP, use:

   Nome de usuário: $operator_username
EOF
    if [ -n "$operator_email" ]; then
      echo "   E-mail da conta:  $operator_email"
    fi
    cat <<'EOF'

   Use a senha de operador criada durante a instalação.
   Não use sua senha SSH/root/Linux da VPS.

7. Revise as ferramentas encontradas e conclua a criação do app.
8. Abra um novo chat usando o Portico MCP e envie exatamente:

   Call system.info on my VPS MCP and tell me the hostname.

Essa chamada é apenas um teste inofensivo de conectividade.
Ela confirma que o ChatGPT realmente conseguiu chegar até esta VPS e que a
chamada foi autenticada e registrada na auditoria do Broker.

Faça essa conexão no seu tempo.
EOF
  else
    cat <<EOF
=== Connect Portico MCP to ChatGPT ===

The VPS side is ready. The last step is connecting Portico MCP to ChatGPT.
This next part happens in ChatGPT Web.

MCP endpoint:
  $PUBLIC_URL

1. Open ChatGPT in your browser.
2. Enable Developer Mode if that option is not already enabled.
3. Open Settings > Apps and choose to create a new MCP app.
4. When ChatGPT asks for the MCP endpoint, use exactly:

   $PUBLIC_URL

5. Choose OAuth authentication.
6. On the Portico MCP login screen, use:

   Username: $operator_username
EOF
    if [ -n "$operator_email" ]; then
      echo "   Account email: $operator_email"
    fi
    cat <<'EOF'

   Use the operator password created during installation.
   Do not use your VPS SSH/root/Linux password.

7. Review the discovered tools and finish creating the app.
8. Start a new chat using Portico MCP and send exactly:

   Call system.info on my VPS MCP and tell me the hostname.

This is only a harmless connectivity test. It confirms that ChatGPT reached
this VPS through authenticated MCP and that the Broker audited the call.

Take as much time as you need to complete the connection.
EOF
  fi
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
    if vps_agent_is_pt_br; then
      printf '[Enter] verificar conexão  [T] mostrar tutorial novamente  [S] sair: '
    else
      printf '[Enter] verify connection  [T] show tutorial again  [Q] quit: '
    fi
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
        if vps_agent_is_pt_br; then
          cat <<'EOF'
Ainda não detectamos a chamada system.info do ChatGPT.

Confira se:
- o app MCP foi criado;
- o OAuth foi concluído;
- o Portico MCP está selecionado no chat;
- a mensagem de teste foi realmente enviada.

Nada foi perdido. Faça os ajustes necessários e pressione Enter para tentar novamente.
EOF
        else
          cat <<'EOF'
We have not detected the ChatGPT system.info call yet.

Check that:
- the MCP app was created;
- OAuth was completed;
- Portico MCP is selected in the chat;
- the test message was actually sent.

Nothing was lost. Fix anything needed and press Enter to try again.
EOF
        fi
        ;;
      *)
        echo "$(vps_agent_text 'Choose Enter, T or S.' 'Escolha Enter, T ou S.')"
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
