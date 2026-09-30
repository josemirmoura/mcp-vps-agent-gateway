# Integração com ChatGPT

Verificado: 2026-09-30.

## Alvo

~~~text
ChatGPT Web
 -> OAuth discovery + Dynamic Client Registration
 -> HTTPS /mcp
 -> Gateway
 -> Broker
 -> VPS
 -> auditoria resistente a adulteração
~~~

A regra de conclusão é mais rígida do que “contêineres estão saudáveis”.

## Pré-requisito do produto ChatGPT

Antes da configuração pública, confirme na conta/workspace real que Developer Mode e criação de app MCP customizado estão disponíveis e quais permissões são efetivamente oferecidas.

A disponibilidade e os nomes da UI são controlados pela OpenAI e podem mudar.

Referências usadas pelo projeto:

- https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt
- https://developers.openai.com/plugins/build/app-quickstart

Uma conexão MCP somente leitura pode ser útil para diagnóstico, mas não equivale à conclusão do caminho completo com escrita.

## Pré-requisitos

1. A conta/workspace expõe criação de MCP customizado.
2. `bash scripts/verify.sh` passou.
3. Um hostname DNS aponta para a VPS.
4. A auth integrada passou:

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Deve terminar em:

~~~text
INTEGRATED AUTH: READY
~~~

Para repetir apenas a fronteira pública:

~~~bash
bash scripts/verify-public.sh
~~~

## Conexão

~~~bash
bash scripts/connect-chatgpt.sh
~~~

Fluxo esperado:

1. ChatGPT lê metadata RFC 9728 do recurso protegido;
2. descobre o ZITADEL;
3. registra dinamicamente um cliente OAuth público;
4. operador entra com a conta OAuth dedicada;
5. Authorization Code + PKCE completa;
6. ChatGPT descobre as tools MCP;
7. operador invoca `system.info`;
8. Gateway valida o access token;
9. Broker combina subject estável e policy;
10. cadeia de auditoria registra a chamada.

O ChatGPT não recebe senha VPS, chave SSH privada, root, shell remoto irrestrito ou segredos de infraestrutura. A senha dedicada OAuth é digitada apenas no login do provedor de identidade. O setup mostra username e e-mail porque a tela de login pode pedir o username.

## Gate de conclusão

`scripts/connect-chatgpt.sh` registra uma baseline fixa da auditoria. O operador conclui a configuração do ChatGPT sem countdown oculto. Ao pressionar Enter, o script procura um novo evento autenticado `system.info` daquele subject.

~~~text
tutorial mostrado
 != sucesso

app do ChatGPT conectada
 + system.info autenticado
 + subject esperado
 + Broker policy allow
 + execução bem-sucedida
 + registro de auditoria correspondente
 = INSTALAÇÃO CONCLUÍDA
~~~

Se o evento ainda não chegou, a instalação permanece incompleta e o script permite tentar novamente.

## Transporte

MCP usa Streamable HTTP sobre HTTPS em:

~~~text
https://<domain>/mcp
~~~

Nenhum transporte WebSocket customizado é introduzido.

## Fronteira de segurança

Confirmações do ChatGPT são controles adicionais de UX. Nunca substituem a autorização server-side.

**Gateway autentica. Broker autoriza. O proprietário da VPS escolhe a policy.**
