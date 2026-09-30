# Autenticação

Verificado contra o modelo atual MCP/OpenAI OAuth em 2026-09-27.

## Caminho de produto suportado

O caminho público suportado para ChatGPT é **OAuth/OIDC integrado e auto-hospedado**.

O pacote executa uma instância dedicada do ZITADEL mais PostgreSQL ao lado do Gateway. O fluxo suportado é autocontido e não exige serviço de identidade terceiro ou túnel separado.

Verificação local/lab continua usando o bearer estático gerado por `scripts/init.sh`. Esse token nunca vira credencial pública do ChatGPT.

## Topologia pública

~~~text
ChatGPT
   |
   | HTTPS + OAuth 2.x / OIDC
   v
domínio público
   |-------------------------------|
   |                               |
   v                               v
Gateway                         ZITADEL
recurso protegido              authorization server
   |                               |
   v                               v
Broker                       PostgreSQL identity state
   |
   v
VPS
~~~

O mesmo hostname público pode servir os dois papéis. Prioridade do router envia `/mcp`, `/healthz` e metadata RFC 9728 ao Gateway; discovery OAuth/OIDC, login, token, user-info e DCR vão ao ZITADEL.

## Bootstrap

Execute:

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

O script gera segredos do identity stack, reutiliza Traefik quando possível, inicia a borda embutida apenas em host limpo, inicia ZITADEL/PostgreSQL, habilita Dynamic Client Registration aberto exigido por clientes MCP, cria identidade dedicada não-admin do operador, grava o subject estável em `.env` e recria Gateway/Broker em modo integrado.

A senha do operador é lida sem eco e nunca escrita em `.env` nem no marcador de estado.

O owner humano temporário de IAM do bootstrap é removido após criar o operador dedicado. O PAT da máquina de bootstrap também é removido. O PAT interno do login-client permanece no volume Docker privado porque o ZITADEL Login precisa dele.

## Metadata do recurso protegido MCP

O Gateway expõe:

~~~text
/.well-known/oauth-protected-resource
~~~

No modo integrado normalmente anuncia:

~~~text
resource: https://mcp.example.com/mcp
authorization_servers:
  - https://mcp.example.com
scopes_supported:
  - openid
bearer_methods_supported:
  - header
~~~

`scripts/verify-public.sh` valida metadata, scopes de recurso, discovery OIDC, anúncio DCR, PKCE S256, refresh token, introspecção privada, HTTPS e negação MCP sem autenticação.

## Resource binding e introspecção privada

O Authorization Server integrado usa um projeto ZITADEL dedicado como audience do recurso MCP.

~~~text
MCP VPS Agent Resource
  └── API application: MCP VPS Agent Introspector
~~~

O project ID vira scope obrigatório:

~~~text
urn:zitadel:iam:org:project:id:<resource-project-id>:aud
~~~

ZITADEL adiciona esse project ID ao audience do access token. O Gateway valida cada bearer opaco pela rede Docker privada de identidade usando introspecção RFC 7662.

Um token só é aceito se:

- `active: true`;
- `sub` presente;
- `iss` igual ao issuer público esperado;
- `exp` no futuro;
- `aud` contém o projeto de recurso MCP;
- scopes incluem todos os exigidos pela metadata protegida.

O cliente de introspecção pertence ao mesmo projeto. O ZITADEL também rejeita introspecção se o audience não contém client/project ID adequado. O Gateway repete a checagem como segunda fronteira.

O endpoint de introspecção não é exposto como superfície administrativa. O Gateway o acessa pela rede privada `integrated-auth`. O client secret fica somente em configuração local legível por root e é redigido dos bundles.

## Dynamic Client Registration

Clientes MCP se registram antes do login, então DCR não autenticado é habilitado. O endpoint dedicado recebe rate limiting no Traefik. Clientes dinâmicos vivem num projeto DCR próprio.

O operador ainda precisa se autenticar antes de o ChatGPT obter token utilizável.

## Binding do subject

O setup cria um usuário humano regular dedicado e grava seu ID estável em:

~~~dotenv
VPS_AGENT_SUBJECT=<operator-user-id>
~~~

O owner de bootstrap existe apenas no setup inicial e é removido antes da instalação seguir. Nunca é a identidade aceita pelo Broker.

## Regras fail-closed

- Gateway local inicia em loopback/static auth para aceitação determinística;
- URL MCP pública é rejeitada salvo modo integrado;
- `verify-public.sh` rejeita URL sem HTTPS;
- metadata pública deve identificar recurso e issuer esperados;
- discovery integrado deve anunciar DCR, PKCE S256 e refresh token;
- metadata deve anunciar o scope de audience dedicado;
- token aceito deve estar ativo, não expirado, issuer correto e audience correto;
- `/mcp` sem autenticação retorna HTTP 401 com challenge Bearer apontando para metadata;
- Broker revalida subject exato e policy;
- Gateway continua sem host root e sem Docker socket.

## Proxy de borda

Se exatamente um Traefik em execução for detectado, o instalador entra na rede Docker dele e reutiliza o ACME resolver. Sem Traefik e com 80/443 livres, inicia o Traefik pinado do pacote.

O instalador recusa substituir web server desconhecido ocupando 80/443.

## Bearer estático local/lab

Permanece apenas para CI local e verificação pré-pública:

~~~dotenv
VPS_AGENT_AUTH_MODE=static
~~~

Não é o caminho público do ChatGPT.
