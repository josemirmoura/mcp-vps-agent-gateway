# Portico MCP: início rápido (PT-BR)

O Portico MCP é instalado pelo terminal, com Docker Compose e scripts transparentes deste repositório.

## Pré-requisitos

Antes de executar a instalação guiada:

- VPS ou computador Linux; Ubuntu 24.04 LTS é o alvo homologado do candidato de lançamento;
- Docker Engine 24 ou superior e plugin Docker Compose v2;
- pelo menos 2 GB de RAM para a instalação do ZITADEL incluída;
- Git, OpenSSL, Python 3 e curl;
- DNS público para o domínio MCP, portas TCP 80/443 acessíveis e HTTPS com certificado válido;
- conta/workspace ChatGPT Web em que Modo Desenvolvedor e cadastro de aplicativo MCP personalizado estejam efetivamente disponíveis.

A OpenAI controla a disponibilidade por plano, conta, workspace e distribuição gradual. Verifique a interface realmente oferecida à sua conta. A documentação oficial descreve acesso integral de escrita/modificação por MCP em Business, Enterprise e Edu. Separadamente, em 06/10/2026, uma conta Plus do operador do Portico demonstrou escrita, leitura e exclusão com escopo delimitado. Isso comprova aquele ambiente específico e não garante a mesma capacidade para todas as contas Plus. Consulte a documentação oficial e teste a sua conta antes de começar.

## Instale pelo terminal

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

O instalador identifica o idioma do terminal. Inicialmente há suporte a português do Brasil e inglês. Você pode escolher explicitamente:

~~~bash
bash scripts/install.sh --lang pt-BR
# ou
bash scripts/install.sh --lang en
~~~

O banner identifica o produto e a versão, incluindo a autoria:

~~~text
Feito por Josemir Moura | github.com/josemirmoura
~~~

## Etapas da instalação guiada

~~~text
Pré-requisitos
 -> Escopo físico da máquina
 -> Resumo da autoridade efetiva
 -> Contêineres
 -> Verificação local
 -> Acesso público seguro
 -> Conexão com ChatGPT
 -> Chamada MCP real registrada na auditoria
 -> INSTALAÇÃO CONCLUÍDA / INSTALLATION COMPLETE
~~~

Antes de criar o estado do Portico, o instalador verifica Linux, Docker Engine/Compose, ferramentas do host, systemd, memória e, para acesso público, as condições de proxy de borda. Se faltar um requisito obrigatório, ele interrompe o processo e apresenta a documentação oficial para instalação.

Quando encontra um Traefik compatível existente, o Portico o reutiliza. Se não encontrar Traefik e as portas 80/443 estiverem livres, instala o Traefik incluído. Ele nunca substitui silenciosamente outro serviço que esteja ocupando essas portas.

Se uma etapa for interrompida, o instalador pode ser executado novamente. Os scripts preservam o estado local, exceto quando o operador solicita explicitamente a remoção completa.

## Modelo recomendado de autoridade: Standard

A opção interativa padrão é **Standard**.

~~~text
teto físico do filesystem: /opt
raízes estáticas autorizadas: nenhuma
acesso aos projetos: aprovação explícita posterior
~~~

O teto físico limita onde o Portico poderá receber permissão no futuro. Ele **não autoriza automaticamente** a leitura ou a escrita em /opt durante a instalação.

Depois de conectar o ChatGPT, o operador poderá aprovar uma subpasta ou o teto físico exato. A autorização para todo o teto apresenta um aviso reforçado: o perfil escolhido alcançará os caminhos atuais e futuros dentro desse teto até a revogação.

Comando não interativo equivalente:

~~~bash
bash scripts/install.sh --profile custom --scope /opt
~~~

O argumento da CLI continua sendo custom por compatibilidade; o nome mostrado ao operador na interface é Standard.

Após conectar o ChatGPT, a autorização de projeto ocorre assim:

~~~text
permissions.request_root_access
        |
        v
confirmação nativa do cliente MCP (elicitation)
        |
        v
Broker ativa read / work / compose apenas para a raiz aprovada
~~~

O modelo de IA não pode aprovar a própria ampliação de permissões. Em clientes com suporte a elicitação MCP, a confirmação ocorre na interface nativa, sem uma ferramenta de autoaprovação nem um token utilizável pelo modelo. O Broker vincula a decisão à identidade autenticada e à solicitação pendente.

## Perfil restrito a um projeto

Escolha este perfil quando o Portico não deve operar fora de uma pasta específica.

~~~bash
bash scripts/install.sh \
  --profile project \
  --scope /opt/my-app
~~~

Se a pasta não existir, a interface guiada poderá oferecer sua criação explícita. Em modo de comando:

~~~bash
bash scripts/install.sh \
  --profile project \
  --scope /opt/my-app \
  --create-scope
~~~

## Whole Host

O perfil Whole Host altera o **teto físico do filesystem** do Broker para /.

Ele não habilita automaticamente o modo Full, a rede irrestrita, a administração de pacotes, usuários ou firewall.

~~~bash
bash scripts/install.sh --profile whole-host
~~~

O instalador exige confirmação explícita antes de iniciar esse perfil.

## Usuário de execução do shell

Os trabalhos de shell confinados usam uma conta real não-root do host. O instalador detecta automaticamente o usuário não-root que iniciou a instalação.

Para escolher outra conta não-root já existente:

~~~bash
bash scripts/install.sh --run-as deploy
~~~

As tarefas normais de shell com escopo não são configuradas para rodar como root.

## Autorizações dinâmicas de pastas

No modo Standard, nenhuma raiz de projeto é autorizada durante a instalação.

Após conectar o ChatGPT, estão disponíveis:

~~~text
permissions.request_root_access
permissions.list_root_access
permissions.revoke_root_access
~~~

Perfis de acesso:

- **read:** leitura de arquivos;
- **work:** leitura e escrita, além de shell confinado à raiz autorizada;
- **compose:** tudo de work, mais operações Compose já permitidas pela política estática de ações.

As autorizações podem ser permanentes ou temporárias e entram em vigor sem reiniciar o Broker.

Para usos avançados, raízes estáticas também podem ser gerenciadas pelo terminal com scripts/delegate-root.sh, mas isso não faz parte do fluxo guiado normal.

## Domínio público e operador OAuth

Para conectar o ChatGPT pela internet, o instalador explica como criar um nome DNS como mcp.example.com, apontar o registro A/AAAA para a VPS e informar apenas o domínio, sem https:// nem /mcp. O DNS é verificado antes de avançar.

A integração OAuth cria uma conta exclusiva de operador Portico. O instalador informa:

~~~text
Nome de usuário: vps-operator
E-mail:         operator@example.com
~~~

A tela OAuth utiliza o **nome de usuário**. Essa identidade é separada das credenciais Linux/SSH/root da VPS.

## Concluir a conexão com ChatGPT

Após verificar o OAuth público, scripts/connect-chatgpt.sh apresenta um tutorial de conexão, o endpoint MCP correto e o usuário de login.

Na instalação interativa, não há contagem regressiva oculta. Cadastre o aplicativo MCP no ChatGPT, conclua o login OAuth, execute a chamada inofensiva system.info e volte ao terminal. Pressione Enter para confirmar. Se a auditoria ainda não tiver detectado a chamada, o Portico indica o que verificar e permite tentar novamente.

A instalação só termina quando a chamada autenticada passa por Gateway, Broker, política e auditoria. Contêineres saudáveis, isoladamente, não concluem o processo.

## Validação apenas local

Para validar o pacote sem OAuth público nem ChatGPT:

~~~bash
bash scripts/install.sh \
  --profile custom \
  --scope /opt \
  --local-only \
  --yes
~~~

Nesse modo, nenhuma raiz estática de projeto é autorizada e a instalação para intencionalmente antes do acesso público. **Não considere esse teste uma instalação completa.**

## Remoção segura e limpeza completa

Remoção segura: preserva a configuração local do operador e o histórico de auditoria.

~~~bash
bash scripts/remove.sh safe
~~~

Limpeza completa: remove runtime, volumes e estado/configuração pertencentes ao Portico MCP, além das imagens Gateway/Broker compiladas localmente por padrão.

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
bash scripts/remove.sh --purge
~~~

Para excluir também o diretório de código clonado, há uma segunda confirmação explícita:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

A limpeza não elimina pastas de projetos delegados, imagens de terceiros, aplicações, bancos de dados ou serviços não pertencentes ao Portico. A pasta antiga /opt/vps-agent-sandbox só é removida caso esteja vazia.

## Seleção da versão e segurança da instalação

A release estável `v0.1.0` **ainda não foi publicada**. Não tente clonar uma tag estável inexistente nem trate a branch `main` mutável como release assinada.

Para inspeção e validação de pré-lançamento, use o clone indicado no começo deste guia. Depois da publicação e verificação de uma release imutável, selecione a tag efetivamente publicada e verifique os checksums, digests e assinaturas correspondentes. O checkout Git permite atualização controlada com checagem de fast-forward, backup e rollback, sujeitos ao aceite real do lifecycle.

Até lá, este é um candidato pré-estável. A conclusão exige instalação limpa em Linux, chamada MCP autenticada de cliente compatível e auditoria do Broker, sem prometer aprovação visual mobile já aceita.

[English Quick Start](quick-start.md).
