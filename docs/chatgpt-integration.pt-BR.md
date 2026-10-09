# Integração do ChatGPT

Última conferência documental: 09/10/2026. Este é o guia em português de [ChatGPT integration](chatgpt-integration.md).

## Objetivo

~~~text
ChatGPT Web
   -> descoberta OAuth + registro dinâmico de cliente
   -> HTTPS /mcp
   -> Gateway
   -> Broker
   -> VPS
   -> auditoria resistente a adulteração
~~~

O critério de instalação concluída é deliberadamente mais rigoroso do que "contêineres saudáveis".

## Compatibilidade do produto ChatGPT

O caminho de produto envolve ferramentas de leitura e de escrita/alteração. A compatibilidade depende das **capacidades efetivamente oferecidas à conta/workspace e verificadas de ponta a ponta**.

Na conferência de 06/10/2026, a documentação pública da OpenAI descrevia suporte completo a escrita/alteração MCP nos ambientes **Business e Enterprise/Edu**. Separadamente, a conta **Plus** do operador deste projeto apresentou ferramentas de escrita do Pórtico e executou uma prova real de escrita, leitura e exclusão com acesso restrito pelo Broker em 06/10/2026. Esse fato é evidência **daquele ambiente específico**, não promessa universal de compatibilidade do Plus.

Antes da configuração pública:

1. confirmar que a conta/workspace atual mostra o Developer Mode e a criação de aplicativos MCP personalizados;
2. em workspaces administrados, confirmar que a gestão habilitou os recursos necessários;
3. confirmar que o usuário consegue criar o aplicativo e informar o endpoint HTTPS `/mcp`;
4. descobrir e executar as ferramentas necessárias para provar as permissões efetivas. Não inferir suporte a escrita pelo nome do plano;
5. comparar o comportamento observado com a documentação oficial vigente e registrar diferenças como evidência do ambiente, não como promessa geral.

Referências oficiais a revisar na instalação e no release:

- https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt
- https://developers.openai.com/plugins/build/app-quickstart

Uma conexão MCP somente de leitura é útil para diagnósticos, mas não comprova a conclusão da instalação com operações de escrita.

## Pré-requisitos

1. A conta/workspace ChatGPT deve efetivamente permitir a criação de aplicativo MCP personalizado em Developer Mode / Plugins.
2. A instalação local já passou em `bash scripts/verify.sh`.
3. O hostname DNS público aponta para a VPS.
4. A autenticação integrada passou:

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Esse comando precisa terminar com `INTEGRATED AUTH: READY`.

Depois de qualquer mudança de DNS, proxy ou OAuth, execute novamente as verificações públicas:

~~~bash
bash scripts/verify-public.sh
~~~

## Conexão

Execute:

~~~bash
bash scripts/connect-chatgpt.sh
~~~

O script repete as checagens públicas, informa o endpoint MCP e o emissor OAuth exatos, apresenta as instruções disponíveis para o ChatGPT e aguarda a confirmação real. O pré-teste verifica HTTPS; metadata RFC 9728; endpoints de autorização e token em HTTPS; DCR; PKCE S256; suporte a refresh token; metadados da autenticação no endpoint de token; introspecção privada vinculada ao público do recurso; e desafio `WWW-Authenticate` quando o MCP é chamado sem autenticação.

O fluxo de padrões previsto é:

1. o ChatGPT lê a metadata RFC 9728 de recurso protegido publicada no host MCP;
2. o ChatGPT descobre o servidor de autorização ZITADEL;
3. registra dinamicamente um cliente OAuth público;
4. o operador entra na conta OAuth dedicada. Credenciais Linux/SSH/root **não** são usadas nesse login;
5. conclui Authorization Code com PKCE;
6. descobre as ferramentas MCP;
7. solicita a chamada inofensiva `system.info`;
8. o Gateway valida o token junto ao emissor integrado;
9. o Broker verifica o sujeito estável e aplica a política;
10. a cadeia de auditoria registra a invocação bem-sucedida.

A interface do ChatGPT pode mudar. Os nomes dos botões e a disponibilidade por conta/workspace devem ser confirmados na versão atual da documentação oficial.

## Fronteira das credenciais

O ChatGPT utiliza o endereço público MCP e completa o fluxo OAuth padronizado. A sessão não precisa de senha de login da VPS, chave SSH privada, credencial root, shell remoto irrestrito nem outros segredos da infraestrutura.

A senha do operador OAuth dedicado é informada **na página de login do provedor de identidade integrado**. O instalador também apresenta o **nome de usuário OAuth**, cujo padrão é `vps-operator`, pois o login pode pedir nome de usuário em vez de e-mail.

As credenciais OAuth são diferentes das credenciais Linux/SSH/root. Acesso remoto eventualmente usado por mantenedores para validar uma versão não integra a jornada suportada do usuário.

## Autorizações e interfaces disponíveis

O sucesso da conexão **não** concede permissões adicionais. O cliente pode solicitar acesso, mas somente o Broker pode efetivar uma concessão conforme a identidade autenticada do operador, a política local, o recurso e a duração.

Clientes compatíveis podem apresentar MCP Apps ou elicitation nativa. **Não há garantia de que o ChatGPT da sua conta ofereça essas interfaces ou que o botão de aprovação apareça corretamente.** Uma confirmação no chat não deve ser tratada como prova autônoma da identidade do proprietário.

Na ausência de interface embutida realmente compatível, utilize um portal HTTPS autenticado **se o runtime instalado o incluir e o disponibilizar**, ou um procedimento CLI/SSH local de operador, conforme a versão. Não considere links, IDs de solicitação nem mensagens produzidas pela IA como autorização. Não siga instruções que peçam aprovação automática pelo assistente.

O suporte dessas alternativas está em implementação e homologação nos PRs de autorizações. Para instalações baseadas na `main` ainda não integrada, confira no terminal e na lista de ferramentas o fluxo efetivamente disponível; não presuma as interfaces novas.

## Critério de conclusão

`scripts/connect-chatgpt.sh` fixa a posição inicial da auditoria antes da tentativa. Em um terminal interativo, o operador pode realizar o cadastro e o login sem contagem regressiva oculta. Ao pressionar Enter, o script verifica se existe um evento `system.info` **novo**, autenticado e associado ao sujeito esperado, posterior à posição inicial.

~~~text
tutorial exibido
 != sucesso

aplicativo ChatGPT conectado
 + system.info autenticado
 + sujeito esperado
 + política do Broker permite
 + execução real na VPS bem-sucedida
 + registro correspondente na auditoria
 = INSTALLATION COMPLETE
~~~

Se nenhum evento aparecer, o script não declara a instalação concluída: explica o que verificar e oferece outra tentativa de Enter. Modo não interativo/CI usa tempo limite definido.

## Transporte

O transporte suportado é MCP Streamable HTTP sobre HTTPS:

~~~text
https://<domain>/mcp
~~~

Não há transporte WebSocket personalizado.

## Limite de segurança

Permissões de aplicativo e confirmações no cliente de IA são controles adicionais de experiência, **não** substitutos da autorização no servidor.

**O Gateway autentica. O Broker autoriza. O operador define a política.**

[English: ChatGPT integration](chatgpt-integration.md) · [Início rápido em português](quick-start.pt-BR.md).
