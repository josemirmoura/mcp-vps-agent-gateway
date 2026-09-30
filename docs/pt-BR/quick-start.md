# Início rápido do Portico MCP

O Portico MCP é instalado pelo terminal usando Docker Compose e scripts transparentes do repositório.

## Requisitos

- VPS Linux; Ubuntu 24.04 LTS é o alvo validado do RC;
- Docker Engine 24+ e Docker Compose v2;
- pelo menos 2 GB RAM para a rota ZITADEL embutida;
- Git, OpenSSL, Python 3 e curl;
- DNS público, TCP 80/443 disponível e HTTPS válido;
- conta/workspace ChatGPT que realmente exponha Developer Mode e criação de app MCP customizado.

## Comece aqui

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

Idiomas oficiais:

~~~text
en · pt-BR · es · de · fr · ja · id
~~~

Seleção explícita:

~~~bash
bash scripts/install.sh --lang pt-BR
~~~

## Fluxo guiado

~~~text
Pré-requisitos
 -> Escopo
 -> Autoridade efetiva
 -> Contêineres
 -> Verificação local
 -> Acesso público seguro
 -> Conectar ChatGPT
 -> chamada MCP real e auditada
 -> INSTALAÇÃO CONCLUÍDA
~~~

O preflight verifica Linux/runtime, Docker/Compose, ferramentas do host, systemd, memória e proxy de borda. Requisitos obrigatórios faltantes interrompem o fluxo com referência oficial.

Um Traefik existente é reutilizado quando identificado com segurança. Se não houver e 80/443 estiverem livres, o Traefik embutido é provisionado.

## Modelo recomendado: Standard

~~~text
teto físico do filesystem: /opt
raízes estáticas:           nenhuma
autoridade de projeto:      concedida depois por aprovação explícita
~~~

O teto define até onde o Portico pode chegar. **Não autoriza /opt durante a instalação.**

Outro teto:

~~~bash
bash scripts/install.sh --profile custom --scope /srv/apps
~~~

Depois da conexão, `permissions.discover_scope` pode mostrar somente os nomes das pastas imediatamente abaixo do teto.

Para entrar num projeto:

~~~text
permissions.request_root_access
 -> confirmação nativa MCP
 -> Broker ativa read / work / compose
~~~

A IA não pode aprovar sua própria expansão de autoridade.

### Arquivos protegidos

Mesmo dentro de projeto autorizado, `.env` e `.env.*` continuam bloqueados. Templates como `.env.example` continuam legíveis. Acesso real a segredo exige aprovação temporária separada via `permissions.request_sensitive_access`.

## Project

~~~bash
bash scripts/install.sh --profile project --scope /opt/my-app
~~~

## Whole Host

~~~bash
bash scripts/install.sh --profile whole-host
~~~

Muda o teto físico para `/`, sem habilitar Full, rede irrestrita, pacotes, usuários ou firewall automaticamente.

## Usuário do shell

Jobs confinados usam usuário real não-root:

~~~bash
bash scripts/install.sh --run-as deploy
~~~

## Hostname público e operador OAuth

O instalador guia a criação do DNS e mostra:

~~~text
Username: vps-operator
Email:    operator@example.com
~~~

A tela OAuth usa o username. Essa identidade é separada das credenciais Linux/SSH/root.

## Conclusão no ChatGPT

`scripts/connect-chatgpt.sh` mostra o tutorial completo. Configure app/OAuth no seu tempo, envie o teste `system.info`, volte ao terminal e pressione Enter. Se a chamada ainda não aparecer, ajuste e tente novamente.

## Validação somente local

~~~bash
bash scripts/install.sh --profile custom --scope /opt --local-only --yes
~~~

Isso **não** é instalação completa porque pula OAuth público e ChatGPT.

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

## Releases estáveis

Após `v0.1.0`, instalações de produção devem usar checkout de tag estável:

~~~bash
git clone --branch v0.1.0 https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

O checkout Git permite atualização fast-forward verificada, migração de estado, backup e rollback automático.
