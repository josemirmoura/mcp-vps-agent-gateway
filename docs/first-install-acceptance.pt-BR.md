# Primeiro Pórtico: roteiro de aceite para iniciantes (PT-BR)

> **Não execute na sua VPS de produção agora.** Este roteiro destina-se à homologação da versão Community candidata, **após** o Bloco 5 fornecer uma revisão imutável e aprovar um Linux de teste. Uma instalação concluída no GitHub Actions não comprova funcionamento na sua conta ChatGPT.

[English version](first-install-acceptance.md) · [Início rápido](quick-start.pt-BR.md)

## O que significam os nomes

- **VPS / Linux:** o computador no qual o Pórtico será instalado. Pode ser um servidor alugado.
- **Terminal:** a janela em que você digita comandos no Linux.
- **Docker:** programa que inicia os componentes separados do Pórtico.
- **Domínio:** endereço público de acesso, como `mcp.exemplo.com`.
- **OAuth:** página de login da conta **do serviço Pórtico**, diferente da conta SSH/Linux.
- **MCP:** canal seguro pelo qual um chat compatível solicita ferramentas ao Pórtico.
- **Broker:** componente da máquina que confere e aplica cada autorização.
- **Standard/Scoped:** o Pórtico conhece um teto físico, normalmente `/opt`, mas **nenhuma pasta de projeto é liberada automaticamente**.

## Preparação, sem instalar nada ainda

1. Confirme com o responsável da integração qual revisão exata será testada e qual Linux descartável usar. Não use sua máquina com dados importantes enquanto o candidato não estiver aceito.
2. No Linux de teste: Ubuntu 24.04, Docker Engine 24+, Compose v2, Git, OpenSSL, Python 3 e curl. O instalador verifica esses requisitos e informa links para corrigir faltas.
3. Tenha um nome de domínio próprio apontando para o computador, com HTTPS válido e portas 80/443 acessíveis. A alteração no provedor DNS exige uma ação consciente do proprietário.
4. No navegador, confira se sua conta ChatGPT oferece criar aplicativo MCP personalizado (Modo Desenvolvedor / Aplicativos ou equivalente). Os nomes de menu mudam. **Se a opção não existir, não invente uma configuração.** Registre “não disponível nesta conta”.

## 1. Começar a instalação

Abra o terminal do **Linux de teste**, não o chat, e execute:

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

**Atenção à versão:** esses comandos demonstram a navegação, mas a instalação de homologação deve apontar à revisão imutável indicada pelo Bloco 5. `main` é mutável; não é automaticamente uma release assinada. A versão estável `v0.1.0` ainda exige publicação e aceite.

O que observar:

- banner `Portico MCP` e idioma;
- lista de pré-requisitos marcada `OK` ou mensagem identificando o que falta;
- escolha de autoridade: selecione **1, Standard**, para manter teto `/opt`;
- resumo mostrando **zero raízes de projeto autorizadas**;
- confirmação explícita para iniciar. Não escolha Whole Host para este teste.

**Se algo falhar:** leia a causa na tela; corrija o pré-requisito e execute novamente o instalador. Não digite comandos de purge para resolver um erro inicial.

## 2. Conferir o funcionamento local

Depois de iniciar os contêineres, a instalação executa a verificação local. Se precisar repetir:

~~~bash
bash scripts/verify.sh
~~~

O que observar: Gateway e Broker saudáveis, política aplicada, auditoria íntegra e teste local `system.info`. Isso ainda **não** significa que o ChatGPT conectou.

Se aparecer erro de contêiner, execute diagnósticos não destrutivos:

~~~bash
bash scripts/diagnose.sh status
bash scripts/diagnose.sh health
~~~

Não copie senhas, tokens, `.env`, cookies ou chaves privadas para a conversa.

## 3. Configurar acesso público e login

Na tela do instalador, digite o **hostname** escolhido, sem prefixo `https://` nem `/mcp`. Confirme que aponta para o computador correto. O instalador prepara Traefik quando permitido, ZITADEL/OAuth e faz as verificações públicas.

Se a configuração não terminar, depois de corrigir DNS/HTTPS com segurança, rode:

~~~bash
bash scripts/verify-public.sh
~~~

O que observar: `INTEGRATED AUTH: READY` após setup e nenhuma falha na checagem pública.

A senha criada para o **operador OAuth** é digitada localmente na instalação e posteriormente na tela de login do serviço. **Não é sua senha da VPS** e não deve ser enviada para o chat.

## 4. Conectar seu ChatGPT

O instalador executa `scripts/connect-chatgpt.sh`, exibe o endereço completo `https://SEU-DOMINIO/mcp` e uma sequência de passos:

1. Abra ChatGPT **no navegador**, na conta com aplicativo MCP habilitado.
2. Vá a Configurações → Aplicativos e procure criar um aplicativo MCP personalizado. A posição e o nome desse controle podem variar. Se não encontrar, anote a limitação da conta.
3. Informe **exatamente** o endereço MCP mostrado no terminal e escolha OAuth quando oferecido.
4. Na página de login do Pórtico, entre com o **nome de usuário do operador OAuth** informado no terminal e com a senha desse serviço.
5. Conclua o cadastro, abra uma nova conversa com o aplicativo habilitado e peça: `Call system.info on my VPS MCP and tell me the hostname.`
6. Volte ao terminal do Linux e pressione **Enter** para conferir se a chamada **nova** apareceu na auditoria. Caso contrário, revise conexão, OAuth, aplicativo selecionado e envio da mensagem. Pressione Enter novamente quando estiver pronto.

O que observar: `CHATGPT WEB CONNECTION VERIFIED` e **`INSTALLATION COMPLETE`**, somente após execução real, sujeito esperado e auditoria correspondente. Isso comprova conexão, **não** autorização para abrir seus projetos.

## 5. Teste seguro de acesso Scoped

Faça isso **somente em pasta de teste descartável dentro de `/opt`**. Nunca peça acesso ao teto `/opt` inteiro, a arquivos protegidos ou a comandos destrutivos durante a homologação inicial.

1. Peça ao ChatGPT para solicitar leitura de uma pasta de teste pelo Pórtico.
2. Observe o método informado. Se a interface nativa de autorização não aparecer, o pedido pode ficar **pendente**, com `approval_method=operator_fallback`. **Pendente não é aprovado.**
3. O operador deve usar a Central HTTPS **se estiver realmente integrada e configurada** para a versão, ou a sessão SSH local independente do assistente. Nunca peça à IA para aprovar o próprio pedido.
4. Na sessão SSH confiável, a alternativa já existente é:

~~~bash
python3 scripts/operator-approvals.py --list
~~~

5. Confira identidade, pasta de teste, acesso e prazo. Primeiro **negue** um pedido e verifique que a leitura continua bloqueada. Crie outro pedido, **aprove** explicitamente e confirme que só a leitura dessa pasta é permitida. Teste expiração e revogação sob orientação do responsável pelo aceite.
6. No celular e no desktop, observe se informações e botões estão completos. Uma tela ilegível ou a ausência de botão é **falha ou recurso não disponível**, nunca consentimento.

Se a solicitação de aprovação falhar, mantenha o pedido pendente e use apenas um caminho de operador autenticado e validado. Login no ChatGPT não equivale à autenticação de quem autoriza no Broker.

## 6. Como sair sem apagar dados

Para interromper o Pórtico no **Linux de teste**, mantendo configuração e auditoria:

~~~bash
bash scripts/remove.sh safe
~~~

O que observar: os componentes do Pórtico param; dados e pastas de projetos do operador não devem ser removidos.

**Nunca execute purge como solução rápida para erro de instalação.** Purge requer confirmação separada e pode destruir estado pertencente ao Pórtico.

## Registrar o resultado

Anote apenas informações não sensíveis: SHA/tag da candidata, idioma, Linux, navegador/cliente, se o aplicativo MCP apareceu, `system.info` auditado, pedido pendente, negação, aprovação, prazo, revogação e se os botões funcionam no computador e no celular. Classifique cada item em **PASSOU, FALHOU, NÃO DISPONÍVEL ou NÃO TESTADO**.

**Não envie** capturas contendo senhas, tokens, cookies, arquivos `.env` ou conteúdo privado da máquina. O Bloco 5 é responsável por reunir as evidências, não este roteiro isoladamente.
