# Central de Autorizações: validação e entrada em operação

Estado: **candidato em PR draft #88**. Nenhum comando abaixo foi executado na VPS do operador. O Gateway existente deve permanecer disponível.

## Baseline do operador em 2026-10-08
- Instalação Compose: `/home/jmour/mcp-vps-agent-gateway`.
- Branch instalada: `main`; commit observado: `f85c833`; checkout sem alterações locais.
- `.env`, `compose.yaml`, `state/` presentes.
- Serviços Broker e Gateway saudáveis na última inspeção do operador.
- Escopo MCP `/opt` **não inclui** o checkout em `/home`. Não tentar escrever em `/home` por meio das ferramentas MCP.
- O script antigo `scripts/operator-approvals.py` existe no repositório GitHub, mas ainda não no checkout instalado.

## Pré-validação isolada (sem implantação)
Após a aprovação dos workflows da PR, executar somente em uma sessão SSH autenticada e de confiança:

```bash
cd /home/jmour/mcp-vps-agent-gateway || exit 1
git status --short
git fetch origin feat/operator-approval-web-backend
git show FETCH_HEAD:scripts/operator-portal-stage.sh > /tmp/portico-operator-stage.sh
bash -n /tmp/portico-operator-stage.sh
# Revise o script antes de executá-lo; ele cria backup com conteúdo sensível.
bash /tmp/portico-operator-stage.sh
```

A ação cria um worktree separado e um snapshot protegido do banco SQLite, da configuração OAuth local e da política. Não altera `main`, não troca imagens e não reinicia contêineres. A evidência esperada é `STAGING READY, NO DEPLOY`. Nunca compartilhar `.env` ou o conteúdo dos backups no chat.

## Gate de segurança do portal
Antes de publicar na internet:
1. Validar o CI completo no SHA **exato** escolhido, inclusive análise de vulnerabilidades, docker/package, 3 VPS, full acceptance e testes do portal.
2. Revisar que `operator-portal` monta somente o socket `operator-run`, e que a bridge permite apenas `admin.approval.list/approve/deny`. O gateway mantém socket separado.
3. Escolher origem HTTPS do operador sob controle do proprietário. O frontend usa `/operator`, `/operator/app.js`, `/operator/style.css`, `/api/operator/login` e `/api/operator/approvals/:id`; o proxy deve encaminhar essas rotas para `127.0.0.1:8765`, com validação de Host e Origin, proteção de login e limite de requisições. Não permitir proxy público direto ao socket.
4. Validar política de cookies Secure/HttpOnly/SameSite, sessões expiradas, CSRF, conteúdo cache no-store, CSP sem inline, URL com identificador opaco e nenhuma credencial no browser.
5. Configurar credenciais exclusivas pelo utilitário `scripts/operator-portal-credentials.py` somente no terminal confiável, sem transmitir senha/token no chat.
6. Testar via HTTPS real: listar, negar, aprovar pedidos temporários específicos e comprovar trilha de auditoria e revogação; testar replay/concorrência/expiração/elevação/permanência/escopo completo e UX mobile.
7. Não divulgar como MCP App nativo até validar a capability e a renderização em cliente compatível.

## Política de rollout
- Backup fora do diretório de produção e sempre antes de atualizar a versão ativa.
- Preferir staging e validação em contêiner isolado; produção somente após aprovação explícita do release gate.
- Não alterar a configuração do Zitadel ou resetar estado OAuth desnecessariamente.
- Conservar a versão anterior e comandos verificáveis de rollback. Se for preciso restaurar SQLite, parar serviços que escrevem no banco, restaurar backup consistente e auditar a integridade antes de reiniciar; isso exige um procedimento supervisionado.
- Se qualquer teste falhar, deixar a instalação ativa intacta. Não promover PR draft nem afirmar produção concluída.

## Entrega
O resultado esperado para Community é um link HTTPS de autorização emitido pelo Gateway apenas para requisições elegíveis, exigindo login independente e decisão explícita do operador. A aprovação permanece vinculada ao sujeito MCP e à política do Broker. O fallback por SSH continua funcional quando não houver navegador ou capacidade de elicitation.
