# Gate final de aceitação do operador

Este gate é intencionalmente operado pelo proprietário. É a última etapa antes de congelar `v0.1.0`.

Não execute casualmente numa instalação MCP importante. Prefira VPS limpa suportada ou ambiente escolhido explicitamente para lifecycle destrutivo.

## Condições iniciais

- usar o tag RC em teste;
- Linux classe Ubuntu 24.04 suportado;
- Docker Engine 24+ + Compose v2;
- pelo menos 2 GB RAM para ZITADEL embutido;
- controle do DNS;
- TCP 80/443 público pelo edge suportado;
- conta/workspace ChatGPT com Developer Mode/custom MCP realmente disponível;
- para writes, confirmar que a superfície atual do ChatGPT expõe essas ações.

Registre tag/commit, OS/arquitetura, perfil, hostname público e timestamps. Não registre senhas, tokens, chaves ou conteúdo de `.env`.

## 1. Instalação limpa

~~~bash
git clone --branch v0.1.0-rc.5 https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

Confirme banner/versão, requisitos, clareza de Standard/Project/Whole Host, autoridade mostrada antes do start, Standard com `/opt` como teto sem root autorizado, possibilidade de teto absoluto diferente, explicação de que teto não é grant, ausência de elevação silenciosa e ausência de pedido de credencial VPS/SSH.

## 2. Segurança pública e OAuth

Confirme DNS, HTTPS, protected-resource metadata, OIDC discovery, DCR + PKCE, login dedicado, subject estável e MCP público não autenticado fail-closed.

## 3. Conexão ChatGPT e E2E

No ChatGPT Web: usar Developer Mode/custom app conforme controles atuais, criar app com endpoint `/mcp`, completar OAuth, chamar `system.info`, voltar ao terminal e Enter para verificar.

O terminal não pode imprimir `INSTALLATION COMPLETE` antes de observar a chamada autenticada atravessando Gateway → Broker → policy → execução → audit.

## 4. UX nativa, discovery e operações seguras

Antes do grant:

- chamar `permissions.discover_scope`;
- confirmar apenas nomes imediatos abaixo do teto;
- confirmar zero conteúdo de arquivo.

Pedir acesso a projeto descartável:

- UI deve ser **elicitation/confirmação nativa MCP**, não iframe antigo;
- mostrar path, perfil, duração e teto;
- pedido do teto exato deve exibir warning mais forte sobre descendentes atuais/futuros;
- aprovar raiz estreita e confirmar somente `read`, `work` ou `compose`;
- revogar e confirmar falha imediata.

Dentro do scope, testar read/write descartável, service/Docker permitido, shell/job bounded se habilitado e denial fora do scope.

### Fronteira de segredo

- criar `.env.example` e confirmar leitura;
- criar `.env` e confirmar denial de read/hash e shell mesmo com pai autorizado;
- pedir `permissions.request_sensitive_access` para aquele path exato;
- confirmar UI nativa identifica arquivo, perfil e expiração;
- aprovar, executar somente o teste, revogar;
- confirmar inacessível imediatamente;
- confirmar segredo ausente de audit/log.

Não amplie policy só para fazer teste passar.

## 5. Auditoria/observabilidade

~~~bash
bash scripts/diagnose.sh status
bash scripts/diagnose.sh health
bash scripts/diagnose.sh logs 100
bash scripts/diagnose.sh audit 100
bash scripts/diagnose.sh bundle
~~~

Confirme health/logs úteis, audit independente com subject/tool/resource/decision/result/sequence, cadeia válida, bundle com permissões restritas e sem secrets configurados.

## 6. Update/rollback

~~~bash
bash scripts/version.sh --check
VPS_AGENT_UPDATE_REF=<next-rc-tag> bash scripts/update.sh
~~~

Confirme versões e resumo, backup, persistência de policy/state/identity, validação de migração, verificação após update e rollback automático em teste controlado de falha.

## 7. Safe remove/reinstall

~~~bash
bash scripts/remove.sh safe
~~~

Confirme runtime removido, grants revogados, credenciais locais rotacionadas, `.env`/policy/state/audit/identity preservados e workloads/recursos de terceiros intactos.

Reinstale com `bash scripts/install.sh` e confirme retomada.

## 8. Purge

Depois de capturar evidências:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Confirme remoção apenas de config/state/identity do MCP e preservação dos recursos administrados.

## Critério de aprovação

Só passa se todas as etapas aplicáveis funcionarem sem enfraquecer o modelo de segurança.

Falha: registrar sem segredos, corrigir produto, rerodar testes automatizados, repetir seção necessária e só então promover `v0.1.0`.

Maturidade Full/R5 e confiabilidade R4 longa continuam decisões separadas.
