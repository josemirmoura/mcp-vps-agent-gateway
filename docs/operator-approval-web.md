# Central de Autorizações do operador (Community)

**Estado: projeto aprovado; interface web ainda não instalada ou validada.**

## Objetivo
Oferecer, em cada instalação Community, uma interface HTTPS responsiva para o proprietário autorizar ou negar pedidos de acesso a pasta ou arquivo protegido. O Broker permanece autoritativo e grava a decisão em auditoria. O fluxo CLI atual em `scripts/operator-approvals.py` continua disponível como fallback.

## Comportamento
- Um pedido criado por `permissions.request_root_access` ou `permissions.request_sensitive_access` continua **pendente** até decisão explícita.
- O identificador `apr_...` e a URL de entrada não são credenciais nem autorizam nada por si só.
- A interface apresenta operador, cliente MCP, máquina, recurso completo, perfil, duração, validade e alerta especial para autorizações abrangentes.
- A página exige autenticação própria do operador do node. Login OAuth do cliente MCP não implica autoridade administrativa.
- Decisão aprovar/negar exige sessão válida, proteção CSRF/origin, POST, verificação de escopo e expiração; Broker verifica novamente e audita.
- Nenhum `approval_token`, token administrativo, segredo do Docker, credencial OAuth, SSH ou conteúdo `.env` vai para o navegador.
- Sem o serviço web, o fluxo CLI mantém a máquina protegida.
- Integração MCP Apps é opcional e dependerá de validação de suporte real em cada cliente, sem prometer UI lateral.

## Segurança e gates
Conferir [operator-approval-fallback](operator-approval-fallback.md) e [threat model](threat-model.md). O serviço web deve executar em processo não-root, com acesso mínimo a canal broker autenticado e nenhuma permissão Docker no navegador. Política CSP, TLS obrigatório fora de loopback, cookies HttpOnly/Secure/SameSite, anti-clickjacking, sessões curtas e idempotência são condições de release. Autorizações elevadas gerais ficam fora do MVP.

## Fases ainda a executar
1. API de pedidos e decisões com teste de identidade e sessão.
2. Painel responsivo com apresentação integral do escopo.
3. Integração Broker auditada e testes adversariais.
4. Empacotamento/instalador e testes reais em dispositivos móveis.
5. Opcional: UI MCP Apps, separadamente comprovada.

Este arquivo não altera a política vigente nem substitui o fluxo operacional existente.
