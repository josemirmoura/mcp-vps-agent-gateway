# Autorizações adaptativas por cliente MCP

Estado: candidato para revisão, derivado da PR #88. Nenhuma implantação desta alteração foi realizada.

## Autoridade e canais

O Broker mantém o pedido persistente e decide autoridade, escopo físico, sujeito, prazo, política e revogação. O cliente MCP pode solicitar, consultar e cancelar apenas seu próprio pedido. Aceitar elicitation, clicar em links ou anunciar MCP Apps não autentica o proprietário.

| Prioridade | Evidência na conexão | Experiência | Decisão |
|---|---|---|---|
| 1 | `extensions.io.modelcontextprotocol/ui.mimeTypes` contém `text/html;profile=mcp-app`, recurso habilitado e portal configurado | Recurso `ui://portico/authorizations-v1.html`; link HTTPS sempre disponível | Central na origem do operador, após autenticação independente |
| 2 | Elicitation URL ou form anunciada, sem Apps elegível | Navegação/consulta nativa; nenhuma senha pelo protocolo MCP | Mesma Central HTTPS ou SSH |
| 3 | Sem UI compatível, com portal HTTPS elegível | Link opaco específico do pedido | Central autenticada |
| 4 | Sem portal ou pedido permanente/amplo/elevado | Instrução para CLI interativa por SSH | Operador com autoridade local |

Não há detecção por marca, user-agent ou nome informado pelo cliente. Sinalização ausente ou inválida usa fallback. O wrapper só tenta enquadrar a Central se o handshake MCP Apps confirmar a versão e `hostCapabilities.sandbox.csp.frameDomains`. Timeout, enquadramento bloqueado, cookies indisponíveis ou bridge recusado mantêm o portal e SSH visíveis. Não existe tool pública de aprovação.

## Isolamento da identidade

A página MCP Apps não contém credenciais. A autenticação e a decisão ocorrem em iframe HTTPS de outra origem, com proteção de mesma origem do navegador. As origens ancestrais devem ser configuradas explicitamente; sem elas o enquadramento fica desabilitado. A Central externa conserva `X-Frame-Options: DENY` e `frame-ancestors 'none'`.

A sessão externa usa cookie Secure/HttpOnly/SameSite=Strict. A sessão embutida usa outro cookie, com Path próprio, SameSite=None e Partitioned. Duração máxima: 15 minutos. A sessão se vincula à identidade configurada do operador, origem, máquina e versão das credenciais. Trocas invalidam a sessão. Cookies não se transferem automaticamente entre clientes, máquinas ou partições. Navegadores sem suporte útil devem abrir o portal externo.

O socket restrito recebe somente a identidade configurada do serviço operador e uma credencial exclusiva. Ele permite listar, consultar, aprovar e negar pedidos temporários específicos do sujeito configurado no Broker. Não oferece Docker, administração geral, permissões permanentes ou todo o teto físico.

Cada decisão precisa de nonce de uso único, válido por até 60 segundos e vinculado à sessão, pedido, fingerprint imutável e máquina. O Broker verifica novamente a fingerprint e a política corrente. Para aprovar arquivos protegidos e perfis work/compose, a Central exige verificação fresca da senha local por pedido; isso é reautenticação, não MFA/passkey. Negar continua possível sem essa nova digitação. Uma sessão autenticada evita repetir senha para pedidos read de pasta, mantendo confirmação explícita em cada pedido.

O Broker grava decisão, grant e auditoria na mesma transação SQLite. Expiração, cancelamento, decisões concorrentes e replay não criam grants duplicados. Perda de resposta exige consulta de estado antes de repetir. `permissions.approval_status` distingue pedido pendente, decisão histórica, grant expirado e revogado. Operações subsequentes reautorizam no Broker.

## Contrato público

`permissions.request_root_access` e `permissions.request_sensitive_access` criam o pedido com ID opaco `apr_...`. A resposta textual informa `approval_contract_version=1`, `approval_channel`, `client_capabilities`, `operator_authentication_required`, `operator_approval_url` quando elegível, `operator_cli`, `status_tool`, `cancel_tool` e `retry_after_seconds=3`. Não inclui `approval_token`.

`permissions.approval_status({"request_id":"apr_..."})` consulta o estado do sujeito autenticado. `permissions.cancel_approval` encerra um pedido pendente do mesmo sujeito sem conceder autoridade. A continuação elicitation transporta somente ID/tipo/recurso e intenção; dados manipulados pelo cliente nunca produzem aprovação. O link não é segredo nem credencial.

Não há migração de esquema nesta alteração. Grants antigos permanecem aplicáveis e revogáveis pelos mecanismos existentes. Pedidos antigos de arquivo protegido sem associação determinística a grant exigem inspeção do operador quando consultados pela nova ferramenta; um estado desconhecido nunca deve ser tratado como autorização.

## Ativação e verificação

`VPS_AGENT_MCP_APPS=0` é o padrão. Para staging, configurar `VPS_AGENT_OPERATOR_PORTAL_URL=https://<host>/operator`, uma identidade consistente em `VPS_AGENT_OPERATOR_ID`/`PORTICO_OPERATOR_ID`, e `VPS_AGENT_INSTANCE_ID`/`PORTICO_OPERATOR_NODE_ID`. Habilitar Apps apenas depois de definir `PORTICO_OPERATOR_FRAME_ANCESTORS` com origens HTTPS exatas realmente observadas no cliente. Nunca usar wildcard ou presumir a origem sandbox de um fornecedor.

Ver [matriz de compatibilidade](approval-compatibility.md), [fallback SSH](operator-approval-fallback.md) e [rollback](adaptive-approvals-rollout.md). Provas de SDK e harness não equivalem a funcionamento em ChatGPT, Claude, VS Code ou aplicativos móveis.

## Referências primárias

- [MCP Apps: especificação estável 2026-01-26](https://github.com/modelcontextprotocol/ext-apps/blob/main/specification/2026-01-26/apps.mdx).
- [MCP elicitation 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25/client/elicitation).
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk), versão fixada em go.mod.
- [OpenAI: referência de Apps](https://developers.openai.com/plugins/reference).
