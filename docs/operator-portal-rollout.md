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
3. Para a instalação atual, utilizar `https://${VPS_AGENT_DOMAIN}/operator` no domínio TLS já existente, respeitando as rotas MCP/OAuth e sem modificar o Gateway atual. Validar o DNS e o certificado real antes de ativar. O frontend usa `/operator`, `/operator/app.js`, `/operator/style.css`, `/operator/api/login` e `/operator/api/approvals/:id`. O cookie tem Path=/operator, evitando tráfego para `/mcp` e `/oauth`. O arquivo opt-in `compose.operator-portal.edge.yaml` publica somente essas rotas por Traefik na rede `${VPS_AGENT_EDGE_NETWORK}` e aplica limite à rota de login. A aplicação confirma Host e Origin públicos; a porta 8765 publicada no host permanece restrita ao loopback. Não permitir proxy público direto ao socket.
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

## Verificação do Traefik na VPS (2026-10-09)
- O operador confirmou que o contêiner Gateway e o Traefik pertencem à rede `traefik-public`; o `gateway` atualmente registra somente o router MCP para `/mcp`, `/healthz` e metadados OAuth. O proxy Zitadel tem routers próprios em `compose.integrated-auth.yaml`.
- `compose.operator-portal.edge.yaml` declara router TLS dedicado para `/operator` e subrotas, priority=1500, e router do login com prioridade=1600 e rate limit. O router MCP continua priority=1000 e as rotas de autenticação têm prioridades menores; não há takeover da raiz `/`.
- Para ativar a rota opt-in, a instância precisará usar os três arquivos `-f compose.yaml -f compose.integrated-auth.yaml -f compose.operator-portal.edge.yaml`. Não executar `docker compose up` genérico antes de conferir o modelo efetivo da instalação.
- A escolha do mesmo host demanda Host+Origin corretos, cookie restrito a `/operator`, e testes negativos para impedir encaminhamento de credenciais à conexão MCP.
