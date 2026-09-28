# MCP VPS Agent Gateway

Plano de controle MCP orientado a segurança para permitir que ChatGPT ou outro cliente MCP trabalhe numa VPS Linux com a autoridade escolhida explicitamente pelo dono da VPS.

> **Status: candidato a release no que depende de aceitação automatizada.** Os workflows em Ubuntu 24.04 validam o pacote Docker de ponta a ponta. O gate restante é o OAuth integrado e auto-hospedado mais uma chamada real auditada do ChatGPT contra a VPS.

## A ideia

Um único pacote traz a caixa de ferramentas ampla. **Quem decide a porção da VPS entregue ao MCP é o usuário.**

~~~text
um projeto:         /opt/meu-app
várias raízes:      /opt/app + /var/www/site + /srv/dados
filesystem inteiro: /
~~~

Filesystem é só uma dimensão. A policy controla separadamente shell, units systemd, recursos Docker/Compose, rede, pacotes, usuários/grupos, firewall e administração temporária.

**O LLM nunca é a fronteira de segurança. O servidor decide.**

## Runtime

~~~text
ChatGPT Web / cliente MCP
        |
        | HTTPS + MCP Streamable HTTP
        v
Gateway container
sem root, sem /host
        |
        | Unix socket protegido
        v
Broker container
fronteira privilegiada
        |
        | host montado em /host
        v
Linux / systemd / Docker / arquivos
~~~

Docker é o mecanismo de empacotamento. O Broker continua sendo um componente privilegiado e deve ser tratado como equivalente a root na porção delegada. A policy server-side é autoritativa.

## Começo rápido

Requisitos: VPS Linux, Docker Engine, plugin Docker Compose, Git, OpenSSL e Python 3.

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git &&
cd mcp-vps-agent-gateway &&
sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 /opt/vps-agent-sandbox &&
bash scripts/init.sh --scope /opt/vps-agent-sandbox &&
docker compose up -d --build &&
bash scripts/verify.sh
~~~

O Começo rápido delega propositalmente apenas `/opt/vps-agent-sandbox`; troque esse caminho pelo diretório que você quer entregar ao MCP. A cadeia com `&&` para imediatamente se alguma etapa falhar.

O bootstrap cria segredos locais aleatórios, ID estável da instância, estado e uma policy do operador **fora do Git**, a partir de config/policy.example.yaml. `--scope` grava `VPS_AGENT_SCOPE_ROOT` no `.env` e migra os caminhos da policy padrão a partir do escopo anterior. O diretório escolhido precisa existir; agora o bootstrap falha antes de qualquer build Docker quando ele não existe. A autoridade continua sendo escolhida explicitamente pelo usuário.

A verificação local prova health, negação de token inválido, chamada MCP real de system.info e integridade do audit chain. **A instalação ainda não terminou.**

### Limite do procedimento suportado

O fluxo documentado é autônomo na própria VPS do usuário. Comandos temporários usados por mantenedores durante desenvolvimento ou validação não são requisitos de instalação. O tutorial suportado nunca pede que o usuário exponha senha da VPS, chave SSH privada, acesso administrativo remoto irrestrito ou segredos não relacionados ao serviço ao ChatGPT ou a um mantenedor.

A senha do operador OAuth integrado é digitada localmente no script de configuração porque é uma credencial deste serviço. Ela não é uma credencial SSH/VPS e não é fornecida ao ChatGPT. Veja o [contrato de instalação](docs/installation-contract.md).

### Autoridade de filesystem sobre o host inteiro

Scoped é o padrão. Para expor deliberadamente o filesystem inteiro ao Broker, use o override explícito:

~~~bash
sed -i 's/^VPS_AGENT_WHOLE_HOST=.*/VPS_AGENT_WHOLE_HOST=1/' .env
docker compose -f compose.yaml -f compose.host.yaml up -d --build
~~~

`compose.host.yaml` é a chave deliberada de whole-host e define `/` como teto físico do Broker. A policy server-side continua controlando quais operações MCP são permitidas.
### Terminar a instalação: OAuth integrado + ChatGPT

O caminho público suportado é OAuth/OIDC auto-hospedado dentro deste pacote. Não é necessário contratar um provedor de identidade externo nem usar um túnel separado.

Antes do próximo comando, crie um registro DNS A/AAAA para um domínio ou subdomínio seu apontando para a VPS. Depois rode:

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

O script:

- reaproveita um único Traefik já existente quando ele é a borda da VPS;
- se não houver Traefik, sobe o Traefik do pacote automaticamente quando 80/443 estiverem livres;
- sobe ZITADEL + PostgreSQL com versões fixadas;
- cria uma identidade dedicada e não administrativa para o operador da VPS;
- cria uma audiência OAuth exclusiva para o MCP e um cliente privado de introspecção;
- habilita Dynamic Client Registration (DCR) compatível com MCP e discovery de PKCE;
- configura o Gateway como protected resource OAuth;
- prende o Broker exatamente ao subject desse operador;
- valida HTTPS, discovery OAuth e negação fail-closed sem autenticação.

O script pede e-mail e senha do operador de forma interativa. A senha é enviada somente à API local de bootstrap do ZITADEL e não é armazenada pelo instalador.

Antes da etapa do ChatGPT, confirme que a conta/workspace de destino oferece o nível necessário de MCP personalizado. Conforme a documentação oficial da OpenAI verificada em 28/09/2026, MCP personalizado completo com ações de escrita/modificação está disponível em Business, Enterprise e Edu. No Pro, o MCP personalizado fica limitado a leitura/fetch e não certifica a superfície completa de escrita deste projeto.

Depois:

~~~bash
bash scripts/connect-chatgpt.sh
~~~

O script mostra o fluxo de conexão no ChatGPT e espera uma **nova chamada auditada de system.info feita pelo ChatGPT**.

~~~text
OAuth integrado pronto
 -> ChatGPT descobre recurso MCP + authorization server
 -> ChatGPT registra dinamicamente seu cliente OAuth
 -> operador faz login
 -> ChatGPT chama o MCP
 -> Gateway autentica
 -> Broker valida subject + policy
 -> operação autorizada chega à VPS
 -> audit registra
 -> INSTALAÇÃO CONCLUÍDA
~~~

Sem a chamada real, a instalação permanece incompleta.

A interface e a disponibilidade de recursos do ChatGPT mudam com o tempo e devem ser conferidas na hora do release/setup. O servidor MCP continua baseado em padrões e o caminho suportado não exige provedor de identidade externo.

## Caixa de ferramentas

A implementação atual inclui filesystem completo em escopo, shell/jobs sandboxed, systemd tipado, Docker/Compose, diagnósticos, pacotes, usuários/grupos, UFW, elevação fora do canal MCP, SQLite exclusivo do Broker, journal de operações, fencing locks e audit hash-chain.

Uma capability existir no pacote não significa que esteja habilitada. Quem decide é config/policy.yaml.

Delete recursivo, chmod/chown, pacotes, usuários, firewall, rede irrestrita e shell administrativo são escolhas explícitas. file.chmod pode aplicar 0777 se o dono da VPS habilitar essa capability.

## Segurança

- Gateway roda sem root e não recebe /host.
- No modo Scoped, o pacote monta fisicamente apenas `VPS_AGENT_SCOPE_ROOT`; acesso ao filesystem inteiro exige o override explícito `compose.host.yaml`.
- Broker não expõe API TCP; o Gateway usa Unix socket.
- Broker reautoriza cada chamada privilegiada pela policy.
- Gateway não recebe Docker socket.
- filesystem bloqueia traversal e symlink escape.
- writes usam operation journal e idempotência.
- mutações usam locks/fencing quando necessário.
- resultados de tools são dados não confiáveis.
- Full não implica rede irrestrita.
- secrets não aparecem em tools genéricas.

## Validação

O GitHub Actions valida vet, race detector, govulncheck, simulação adversarial, systemd real, Docker real, acceptance Gateway -> Broker -> Linux, acceptance do pacote Docker e testes negativos.

Veja [validação da implementação](docs/implementation-validation.md).

## Operação

~~~bash
bash scripts/diagnose.sh status
bash scripts/diagnose.sh health
bash scripts/diagnose.sh logs 200
bash scripts/diagnose.sh audit 100

bash scripts/update.sh
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

A remoção segura preserva configuração e auditoria. O purge exige confirmação explícita e remove apenas artefatos do MCP. O update faz backup da configuração/estado, usa Git fast-forward, verifica o runtime novo e restaura código/estado anterior se a verificação falhar.
## Documentação

- [Contrato de instalação](docs/installation-contract.md)
- [Modelo do produto](docs/product-model.md)
- [Fluxo Docker e primeira execução](docs/installer-flow.md)
- [Arquitetura](docs/architecture.md)
- [Autenticação](docs/authentication.md)
- [Múltiplas instâncias](docs/multi-instance.md)
- [Integração ChatGPT](docs/chatgpt-integration.md)
- [Threat model](docs/threat-model.md)
- [Hardening](docs/security-hardening-v2.md)

## Licença

Apache-2.0. Veja [LICENSE](LICENSE).
