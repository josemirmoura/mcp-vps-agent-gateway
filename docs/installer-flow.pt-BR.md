# Fluxo da primeira instalação do Portico MCP

Este é o guia brasileiro de [Portico MCP first-run flow](installer-flow.md).

## Objetivo do produto

A instalação é declarativa, orientada pelo terminal e controlada pelo operador.

O operador controla dois arquivos locais acessíveis à operação (mas não compartilhados por padrão com a IA), além de uma variável de teto físico:

~~~text
.env                    # inclui VPS_AGENT_SCOPE_ROOT
config/policy.yaml     # política lógica de capacidades e recursos
~~~

São arquivos de estado local do proprietário. `config/policy.yaml` é criado a partir do modelo versionado `config/policy.example.yaml` e propositalmente não é colocado no Git.

Depois disso, o Docker Compose inicia o pacote. Não existe outro assistente de instalação que assuma posse dessa configuração.

## Limites da instalação suportada

O procedimento é executado pelo usuário na própria VPS. Ele não exige a presença de desenvolvedores, um shell remoto controlado pelo ChatGPT ou divulgação de credenciais de administração.

A senha do operador OAuth é diferente: ela é necessária à pilha de identidade integrada, é informada localmente ao `setup-integrated-auth.sh` sem eco no terminal e não é transmitida ao chat.

O [Contrato de instalação em português](installation-contract.pt-BR.md) descreve de forma normativa o que pertence à instalação e o que é exclusivo da engenharia.

## Entrada do assistente guiado

O usuário começa com:

~~~bash
bash scripts/install.sh
~~~

O script coordena componentes transparentes, em vez de implementar um segundo instalador. Ele apresenta a autoridade efetiva antes de iniciar o runtime, só interrompe para decisões realmente necessárias e pode ser executado novamente após uma etapa interrompida.

Operadores avançados também podem executar as fases individualmente.

## Fase 1: preparação inicial

O ponto de entrada suportado é:

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

O perfil interativo recomendado é **Standard**: teto físico `/opt` e nenhuma raiz de projeto estática. O teto limita até onde uma permissão poderá existir, mas não autoriza ler ou alterar projetos. O operador pode escolher outro teto absoluto quando organiza aplicações em outro caminho.

Para reproduzir a preparação explicitamente:

~~~bash
bash scripts/init.sh --scope /opt --dynamic-baseline
~~~

Antes de criar estado local, o instalador executa `scripts/preflight.py`: sistema e dependências, Docker Engine/Compose, ferramentas do host, systemd/memória e, no caminho público, Traefik e portas 80/443. A falta de requisito obrigatório encerra essa fase **antes** de criar `.env`, política e runtime.

Na sequência, a preparação verifica o Compose, cria `.env` com segredos locais aleatórios e ID estável da instância quando necessários, copia o modelo de política, persiste o teto físico escolhido, configura o shell confinado com usuário real não-root, cria o diretório de estado local e valida a sintaxe do Compose.

## Teto físico do filesystem

O `compose.yaml` padrão monta somente `VPS_AGENT_SCOPE_ROOT` no Broker em `/host`. O Broker exige que filesystem, diretório de trabalho de shell e caminhos Compose da política permaneçam nesse teto. Tentativas de escape falham de maneira segura.

Alterar `VPS_AGENT_SCOPE_ROOT` de instalação já existente troca apenas o teto físico. As raízes lógicas já existentes em `config/policy.yaml` são preservadas por padrão. No Standard de instalação limpa, `--dynamic-baseline` evita conceder `/opt` como raiz estática. `scripts/init.sh --migrate-policy-root` é um opt-in explícito quando há migração real da raiz do projeto.

Um operador avançado pode selecionar teto de filesystem de host inteiro com `VPS_AGENT_WHOLE_HOST=1` em `.env` **e** `compose.host.yaml`. Essa configuração persiste para comandos de ciclo de vida e monta `/` no Broker, mas **não integra o caminho Scoped/Standard padrão** e não habilita Full automaticamente.

## Fase 2: escolha da autoridade MCP

No Standard, a instalação começa sem acesso autorizado a raízes de projeto.

`permissions.discover_scope` pode listar somente os nomes dos diretórios imediatos abaixo do teto físico. A ferramenta não lê arquivos nem percorre recursivamente esses caminhos. Ela ajuda a identificar o projeto exato sobre o qual pedir uma autorização.

A implementação atual de `main` usa confirmação nativa quando o cliente efetivamente anuncia suporte a MCP elicitation. O fluxo esperado é:

~~~text
permissions.request_root_access
        -> solicitação pendente no Broker
        -> confirmação MCP nativa (se o cliente suportar)
        -> delegação limitada ao sujeito, se aprovada validamente
~~~

Quando o cliente não oferecer elicitation, **não presuma uma aprovação**. A solicitação permanece pendente. Portal autenticado, MCP Apps e fallback CLI/SSH integram implementações candidatas separadas e só podem ser usados quando estiverem incluídos, configurados e validados na versão instalada. Em qualquer canal, a IA não pode aprovar sua própria ampliação de acesso.

Perfis:

- `read`: leitura de arquivos;
- `work`: leitura e escrita, com shell confinado ao diretório autorizado;
- `compose`: acrescenta operações Compose que a política estática já permite.

Arquivos protegidos formam uma segunda fronteira. Por padrão, `.env` e `.env.*` permanecem bloqueados, enquanto `.env.example`, `.env.sample` e `.env.template` são modelos comuns. Ler ou modificar arquivo protegido requer uma permissão temporária separada por `permissions.request_sensitive_access`, limitada ao caminho exato, vinculada ao sujeito, auditada, com vencimento e revogável independentemente. Trabalhos de shell Scoped também mascaram caminhos protegidos, inclusive aliases de hardlinks nos projetos delegados, exceto quando há grant `work` temporário para o arquivo preciso.

Operadores avançados podem configurar raízes estáticas usando `scripts/delegate-root.sh`. O fluxo guiado não exige editar `config/policy.yaml` manualmente.

Uma delegação dinâmica pode apontar para uma subpasta ou para o teto físico configurado, este último sujeito a alerta reforçado por abranger projetos atuais e futuros. Ela nunca pode ultrapassar o teto. Mesmo a concessão do teto não libera segredos protegidos.

Ações systemd, Docker/Compose, rede, pacotes, usuários/grupos, firewall e elevação temporária têm restrições próprias na política estática.

## Fase 3: iniciar o runtime

~~~bash
docker compose up -d --build
~~~

Separação de processos:

~~~text
Gateway: não-root, sem acesso à raiz do host
Broker: privilegiado, host montado em /host, sem porta remota de controle
~~~

Docker empacota o Broker privilegiado, mas **não** é o mecanismo que decide as autorizações. A política aplicada pelo Broker é essa fronteira.

## Fase 4: verificação local

~~~bash
bash scripts/verify.sh
~~~

O verificador cobre configuração Compose, saúde do Broker e Gateway, integridade da auditoria, autenticação e uma chamada inofensiva `system.info`.

Verificação local verde significa que o runtime está pronto para a próxima fase. **A instalação ainda não terminou.**

## Fase 5: OAuth integrado e endpoint público

Um chat web remoto precisa acessar MCP por HTTPS válido. O pacote automatiza a pilha de identidade, mas não controla o provedor DNS do usuário.

O operador deve criar/apontar um hostname, por exemplo `mcp.example.com`, usando registros DNS A/AAAA e informar **somente o hostname**. O instalador confirma a resolução antes de avançar; a verificação pública valida HTTPS.

Então execute:

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

O script reutiliza um Traefik existente quando detecta exatamente um compatível. Quando nenhum está presente e as portas 80/443 estão livres, inicia o Traefik empacotado. Ele sobe ZITADEL e PostgreSQL, cria uma identidade dedicada **não administrativa**, mostra o nome de usuário e o e-mail de login, prepara a audiência OAuth específica do MCP e cliente privado de introspecção, ativa Dynamic Client Registration compatível com MCP, vincula identidades Gateway/Broker e executa a verificação pública.

Uma execução bem-sucedida encerra com:

~~~text
INTEGRATED AUTH: READY
~~~

O próprio setup valida o endpoint. Depois de mudanças DNS, proxy ou OAuth, rode:

~~~bash
bash scripts/verify-public.sh
~~~

Esse comando verifica certificado e rota HTTPS, descoberta OAuth/OIDC, metadata de recurso protegido, DCR/PKCE, introspecção privada e rejeição segura de chamadas MCP não autenticadas.

O script não substitui silenciosamente outro serviço que já ocupa 80/443.

## Fase 6: conferir capacidade MCP no ChatGPT

Antes de conectar, abra a conta/workspace ChatGPT de destino e verifique os recursos de Developer Mode / Plugins para criação de aplicativo MCP personalizado.

Continue somente quando a interface real permitir a ação. Não rejeite ou aceite uma conta apenas pelo nome do plano, pois a disponibilidade do produto pode variar.

Quando a criação de MCP personalizado não aparecer, o servidor Linux pode estar pronto, mas **não** é possível concluir o aceite do ChatGPT nessa conta naquele momento.

## Fase 7: apresentar o tutorial atual do ChatGPT Web

~~~bash
bash scripts/connect-chatgpt.sh
~~~

A interface do ChatGPT é uma etapa manual indispensável. O usuário precisa cadastrar/selecionar o aplicativo e fazer login OAuth pelo navegador. O script mostra endpoint, modo de autenticação, sujeito esperado e os passos de conexão. É usado o login OAuth dedicado, nunca as credenciais VPS/SSH.

Nesse instante:

~~~text
Tutorial apresentado.
Instalação NÃO concluída.
~~~

## Fase 8: verificar conexão real e auditoria

Antes de instruir o cadastro, o script fixa uma posição inicial auditada. No terminal interativo, o usuário termina o cadastro em seu próprio ritmo e volta para pressionar Enter quando desejar verificar.

No chat, peça uma chamada `system.info`. Cada Enter procura uma chamada correspondente posterior à posição inicial. Se ela não estiver presente, o Pórtico explica o que verificar e permite repetir a checagem sem reiniciar a instalação.

Condição necessária:

~~~text
chamada MCP real do ChatGPT
+ sujeito autenticado esperado
+ decisão de permitir pela política
+ execução bem-sucedida na VPS
+ registro correspondente do Broker
+ cadeia de auditoria válida
~~~

Somente então:

~~~text
CHATGPT WEB CONNECTION VERIFIED
INSTALLATION COMPLETE
~~~

Se a chamada não ocorreu ou foi negada, a instalação continua incompleta.

## Procedimentos exclusivos de desenvolvimento

Probes temporários, runners de CI, diagnósticos curl/openssl improvisados, branches de desenvolvimento e comandos executados pontualmente para um agente sem acesso à VPS pertencem à engenharia/aceite, não às etapas finais de instalação.

A revisão de release executa `python3 scripts/check-installation-contract.py` para encontrar textos de desenvolvimento e instruções indevidas de compartilhamento de credenciais nos documentos suportados.

## Atualização

~~~bash
bash scripts/update.sh
~~~

O atualizador foi projetado para preservar `.env`, política e estado do Broker, conferir avanço Git fast-forward e fazer backup, reconstrução, verificação e recuperação quando a mudança falhar. **A capacidade real de rollback precisa ser verificada na versão instalada**; testes adicionais de falha de backup, volume e migração fazem parte do gate de lifecycle ainda em validação. Não presuma recuperação garantida em todo defeito.

## Remoção

~~~bash
bash scripts/remove.sh safe
~~~

A remoção segura interrompe o pacote preservando estado de configuração e auditoria. O purge é outra ação, explicitamente confirmada:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Ele também remove as imagens Gateway/Broker padrão compiladas localmente e remove o diretório legado `/opt/vps-agent-sandbox` apenas quando estiver vazio.

Para excluir também o checkout Git, é necessária confirmação adicional:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

O pacote nunca exclui diretórios de projetos delegados, imagens de terceiros, aplicações, serviços, contêineres, bancos de dados ou arquivos arbitrários só porque teve autorização para gerenciá-los.

## Próximos passos

Após o aceite do ChatGPT, confira operações permitidas/negadas e a auditoria, sem solicitar acesso mais amplo que o necessário. Veja também o [Início rápido PT-BR](quick-start.pt-BR.md) e a [Integração ChatGPT PT-BR](chatgpt-integration.pt-BR.md). Publicação de stable, aceite mobile e licença final são gates separados.

[English: installer flow](installer-flow.md).
