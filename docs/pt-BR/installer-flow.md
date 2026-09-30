# Fluxo de primeira execução do Portico MCP

## Objetivo do produto

A instalação é declarativa e orientada ao terminal. O operador controla dois arquivos locais legíveis por humano/IA e uma variável de teto físico:

~~~text
.env                   # inclui VPS_AGENT_SCOPE_ROOT
config/policy.yaml     # policy lógica de capacidades/recursos
~~~

Ambos são estado local do operador. O Docker Compose inicia o pacote. Não há um wizard ou instalador paralelo dono da configuração.

## Fronteira suportada

Este documento descreve a instalação do usuário final. Ela não exige desenvolvedor, shell remoto controlado pelo ChatGPT nem compartilhamento de credenciais da VPS.

A senha dedicada do operador OAuth é diferente: é uma credencial do serviço necessária ao stack de identidade. Ela é digitada localmente em `setup-integrated-auth.sh`, sem eco no terminal, e não é fornecida ao ChatGPT.

## Entrada guiada

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

`install.sh` é a camada de orquestração. Ele usa os mesmos componentes transparentes, mostra a autoridade efetiva antes de iniciar o runtime, pausa apenas quando há decisão inevitável do operador e pode ser executado novamente após uma interrupção.

## Fase 1 — Bootstrap

O perfil recomendado é **Standard**: teto físico `/opt` e nenhuma raiz estática de projeto. O teto é apenas o limite máximo do filesystem; ele não concede leitura/escrita. O operador pode escolher outro teto absoluto.

~~~bash
bash scripts/init.sh --scope /opt --dynamic-baseline
~~~

Antes de alterar estado, `scripts/preflight.py` verifica requisitos do host/runtime e, no caminho público, a situação do Traefik e portas 80/443. Falhas obrigatórias param antes de criar `.env`, policy ou estado.

O bootstrap cria `.env` com segredos locais aleatórios e ID estável da instância, cria `config/policy.yaml`, persiste o teto escolhido, configura o shell confinado com usuário real não-root, cria estado local e valida o Compose.

## Teto físico do filesystem

`compose.yaml` monta apenas `VPS_AGENT_SCOPE_ROOT` dentro do Broker como `/host`. O Broker valida que paths de filesystem, cwd do shell e Compose permanecem dentro desse teto. Escape de policy falha fechado.

Alterar `VPS_AGENT_SCOPE_ROOT` muda apenas o teto físico. As raízes lógicas existentes são preservadas. No Standard novo, `--dynamic-baseline` garante que escolher `/opt` não cria autorização estática para `/opt`.

Whole Host exige `VPS_AGENT_WHOLE_HOST=1` e `compose.host.yaml`, mudando o teto físico para `/`. Isso não habilita Full.

## Fase 2 — Autoridade MCP

No Standard, nenhum projeto é autorizado durante a instalação.

`permissions.discover_scope` pode listar somente os nomes das pastas imediatamente abaixo do teto. Não abre arquivos nem desce nessas pastas.

Depois da conexão:

~~~text
permissions.request_root_access
 -> solicitação pendente no Broker
 -> elicitation MCP / confirmação nativa do host
 -> delegação ativa vinculada ao subject
~~~

Perfis:

- `read`: leitura do filesystem;
- `work`: leitura/escrita + cwd de shell confinado;
- `compose`: acrescenta autoridade Compose somente para ações já liberadas pela policy estática.

Quando o cliente suporta MCP elicitation, a confirmação humana aparece na UI nativa do cliente. O antigo iframe customizado não faz parte do fluxo normal.

### Segredos dentro de projetos autorizados

Existe uma segunda fronteira. `.env` e `.env.*` permanecem bloqueados, enquanto `.env.example`, `.env.sample` e `.env.template` continuam como templates legíveis.

Um arquivo protegido exige `permissions.request_sensitive_access`: grant temporário, exact-path, subject-bound, auditável, com expiração e revogação própria. Shell Scoped mascara esses paths, inclusive aliases por hardlink encontrados nas raízes delegadas, salvo grant temporário explícito para aquele arquivo.

A aprovação dinâmica pode atingir uma subpasta ou, após aviso reforçado, o próprio teto físico. Nunca pode sair dele. Liberar o teto é amplo porque inclui pastas atuais e futuras, mas ainda não libera arquivos secretos protegidos.

systemd, Docker/Compose, rede, pacotes, usuários/grupos, firewall e elevação continuam limitados separadamente pela policy.

## Fase 3 — Inicialização

~~~bash
docker compose up -d --build
~~~

~~~text
Gateway: non-root, sem root do host
Broker: privilegiado, host montado em /host, sem porta remota de controle
~~~

Docker empacota o Broker; a fronteira de autorização é a policy.

## Fase 4 — Verificação local

~~~bash
bash scripts/verify.sh
~~~

Verifica Compose, saúde do Broker/Gateway, integridade da auditoria, autenticação e uma chamada inofensiva `system.info`.

Sucesso local significa runtime pronto. **Não significa instalação concluída.**

## Fase 5 — OAuth integrado + endpoint público

O ChatGPT precisa de MCP remoto HTTPS. DNS é uma ação manual inevitável porque o pacote não controla o provedor do usuário. O fluxo guiado explica A/AAAA e valida a resolução.

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Se houver exatamente um Traefik seguro para reutilização, ele é reutilizado. Se não houver e 80/443 estiverem livres, o Traefik embutido é iniciado. Depois entram ZITADEL + PostgreSQL, identidade dedicada não-admin, username/e-mail OAuth, audiência de recurso MCP, cliente privado de introspecção, Dynamic Client Registration compatível com MCP, binding de identidade Gateway/Broker e verificação pública.

Sucesso termina com:

~~~text
INTEGRATED AUTH: READY
~~~

Para repetir a fronteira pública:

~~~bash
bash scripts/verify-public.sh
~~~

Valida HTTPS, OAuth/OIDC discovery, metadata do recurso protegido, DCR/PKCE, introspecção privada e MCP não autenticado fail-closed. O script não substitui um serviço web desconhecido em 80/443.

## Fase 6 — Capacidade do ChatGPT

Antes de iniciar a conexão, confirme que a conta/workspace real expõe Developer Mode e criação de app MCP customizado. A disponibilidade é controlada pelo produto ChatGPT e pode mudar.

## Fase 7 — Tutorial ChatGPT Web

~~~bash
bash scripts/connect-chatgpt.sh
~~~

Criar/selecionar o app e completar OAuth acontece no ChatGPT Web. O usuário entra com a conta OAuth dedicada, nunca com credenciais VPS/SSH.

## Fase 8 — Verificação real

O script registra uma baseline de auditoria. O usuário configura o ChatGPT no próprio ritmo, retorna e pressiona Enter. O ChatGPT deve chamar `system.info`.

Sucesso exige:

~~~text
chamada MCP real do ChatGPT
+ subject autenticado esperado
+ policy allow
+ execução bem-sucedida
+ registro de auditoria do Broker
+ cadeia de auditoria válida
~~~

Somente então:

~~~text
CHATGPT WEB CONNECTION VERIFIED
INSTALLATION COMPLETE
~~~

## Atualização

~~~bash
bash scripts/update.sh
~~~

Faz backup de `.env`, policy e estado, aplica fast-forward Git, reconstrói, verifica e faz rollback automático em falha.

## Remoção

~~~bash
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Para remover também o checkout:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

O pacote nunca apaga arbitrariamente projetos delegados, apps, sites, bancos, imagens/contêineres de terceiros, serviços ou arquivos apenas porque podia administrá-los.
