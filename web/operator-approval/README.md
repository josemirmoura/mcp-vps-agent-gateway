# Pórtico: Central de Autorizações (pré-release)

**Estado: implementação em PR draft, sem deploy de produção.** As aprovações continuam exigindo confirmação do proprietário; o Broker continua autoritativo.

## Arquitetura de menor privilégio
- O Broker pode expor um **segundo socket Unix** em `/run/portico-operator/operator.sock`, montado no volume `operator-run`.
- O handler desse socket aceita **somente** `admin.approval.list`, `admin.approval.approve` e `admin.approval.deny`, autenticados por um token randômico independente com no mínimo 32 caracteres. O token do operador web jamais é o token administrativo integral do Broker.
- O contêiner `operator-portal` tem exclusivamente esse volume (sem o socket MCP geral e sem Docker socket), filesystem read-only, UID não privilegiado, capacidades Linux removidas.
- O Broker ainda verifica os pedidos, identidade MCP, estado/expiração e grava audit trail. O frontend não amplia TTL/perfil.
- O portal publica `8765` apenas no **loopback do host**, e só funciona externamente depois de configurar HTTPS em um reverse proxy confiável.

## Configuração opt-in
Antes de ativar o profile do portal, configurar em `.env`:
- `VPS_AGENT_OPERATOR_APPROVAL_TOKEN`: token randômico exclusivo, 32+ caracteres; protegido, nunca compartilhado com cliente MCP.
- `PORTICO_OPERATOR_PASSWORD_SCRYPT`: verificador `salthex:digesthex`, gerado com `hashlib.scrypt(password,salt,n=2**14,r=8,p=1,dklen=32)`. Não armazenar a senha em texto claro.
- `PORTICO_OPERATOR_PUBLIC_ORIGIN`: origem HTTPS exata, por exemplo `https://exemplo-do-operador.invalid` (placeholder, não um endereço real).
- `VPS_AGENT_SCOPE_ROOT`: raiz física, exibida pelo portal para alertas de autorização ampla.

Ativação **somente após revisão de segurança e provisionamento HTTPS**:

```bash
docker compose --profile operator-portal up -d --build
```

Por segurança, este comando **não** faz parte do instalador padrão.

## Fluxo e endpoints
- `GET /operator?request=apr_...`: página do operador.
- `POST /operator/api/login`: verifica senha, atribui cookie de sessão de 15 min (Secure, HttpOnly, SameSite Strict).
- `GET /operator/api/approvals/:id`: apresenta dados de pedido pendente, exige sessão.
- `POST /operator/api/approvals/:id/decision`: decisão approve/deny; exige sessão, cabeçalho CSRF e Origin idêntico à origem pública definida; usa somente IPC restrito ao Broker.
- Id do pedido **não é segredo nem token de concessão**. A URL nunca autoriza sem autenticação e POST explícito.
- `scripts/operator-approvals.py` por SSH permanece fallback.

## Bloqueadores de release
1. Aprovação de threat model do socket restrito e isolamento de processo, incluindo tentativa de acesso ao socket MCP normal.
2. E2E real: expiração, replay, sessão, CSRF, ID trocado, concorrência, revogação e testes móveis.
3. Configurar proxy HTTPS em domínio dedicado/controlado; CSP sem inline JS/CSS como melhoria antes da exposição à internet, validar Host e encaminhamento Origin; limitar força bruta no proxy.
4. Completar provisionamento seguro de credenciais, rotação, logs, atualização, rollback e proteção de arquivos locais.
5. Validar navegador desktop/mobile e coexistência com confirmação nativa MCP.
6. MCP Apps exige validação própria; interface lateral do ChatGPT não é prometida.

Não colocar em produção até que os gates sejam satisfeitos.
