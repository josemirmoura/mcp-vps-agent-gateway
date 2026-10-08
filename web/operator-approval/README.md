# Operador web: implementação experimental (NÃO implantada)

Branch: `feat/operator-approval-web-backend`; revisão de segurança requerida antes de deploy.

### Arquivos
- `web/operator-approval/index.html`: interface desktop/mobile e login.
- `web/operator-approval/server.py`: API HTTP Python padrão, escuta **somente 127.0.0.1:8765**; não habilitar na internet diretamente.
- `web/operator-approval/test_server.py`: testes unitários iniciais.

### Dependências
- Backend reutiliza `scripts/operator-approvals.py` e o comando administrativo existente do Broker; o usuário de execução precisaria de autoridade administrativa sobre o container Broker. **Esse é um risco material:** um comprometimento do serviço web poderia abusar dessa autoridade. Antes de habilitar publicamente, substituir acesso Docker amplo por um IPC restrito de operador, autenticação forte e isolamento com menor privilégio, e auditar toda a superfície.
- O verificador de senha é fornecido por `PORTICO_OPERATOR_PASSWORD_SCRYPT` no formato `salthex:digesthex` (scrypt n=16384 r=8 p=1 dklen=32).
- O frontend espera `PORTICO_OPERATOR_PUBLIC_ORIGIN` como origem exata HTTPS, obrigatória na verificação Origin dos POSTs.
- Para publicar, adicionar reverse proxy TLS com limites de requisição, limite de login por IP real no proxy, política explícita de headers, log de acesso reduzido e validação de Host. A verificação em processo por loopback não substitui esses controles.
- Não habilitar endpoints de aprovação em staging público sem revisão independente do threat model.

### API
- `POST /api/operator/login`: senha do operador, sessão cookie Secure HttpOnly SameSite Strict de 15 minutos.
- `GET /operator?request=apr_...`: HTML da Central (sem dados até autenticação).
- `GET /api/operator/approvals/:id`: detalhes do pedido pendente (exige sessão).
- `POST /api/operator/approvals/:id/decision`: JSON decision approve/deny; sessão + CSRF + Origin; Broker revalida e audita.
- Nenhuma API entrega token administrativo ou token de aprovação ao navegador.

### Gates bloqueadores de release
1. Introduzir canal Broker operador com privilégio mínimo, sem Docker socket no processo web.
2. Aplicar autenticação robusta, defesa contra força bruta distribuída, sessão persistente multiworker (ou processo único supervisionado), logout e proteção de proxies.
3. Validar campos/permissões, expiração, origem, CSRF, condição de corrida e autorização de identidade por testes adversariais.
4. Empacotar como serviço não-root, proxy HTTPS, health probes, instalador/update e rollback.
5. Testar em ChatGPT desktop/mobile, navegadores mobile e clientes MCP sem elicitation. MCP Apps é trabalho separado.
6. Apenas depois do gate de segurança, disponibilizar em uma versão Community e projetar para Cloud.

Nunca anunciar este protótipo como autorização web em produção.
