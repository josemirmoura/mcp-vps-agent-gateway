# Conectar o Pórtico MCP a clientes de IA

Este guia apresenta **clientes que oferecem conexão MCP remota**. A disponibilidade do recurso depende da versão e do plano de cada cliente. **A existência de suporte MCP no cliente não constitui prova de compatibilidade E2E com o Pórtico**. A verificação da Community neste ciclo é feita com ChatGPT Web, OAuth e uma chamada `system.info` registrada no Broker.

## Dados comuns

- **Nome sugerido:** Pórtico MCP
- **Descrição sugerida:** Conecta IAs a um computador Linux com acesso controlado e aprovações.
- **URL MCP:** `https://SEU-DOMINIO/mcp` (substitua pelo endereço exibido pelo instalador).
- **Transporte:** MCP Streamable HTTP sobre HTTPS.
- **Autenticação:** OAuth/OIDC. Prefira descoberta automática, quando oferecida pelo cliente.
- **Login:** usuário e senha **OAuth do operador Pórtico**, criados durante a instalação. Nunca informe a senha SSH/root ou tokens internos do Gateway.

O Pórtico não autoriza automaticamente acesso aos projetos. No perfil Standard, `/opt` é apenas o teto físico; cada raiz de projeto exige aprovação separada. O Broker aplica a política independentemente do cliente.

## ChatGPT Web

**Estado da integração:** validação real em andamento no E2E Community (#153).

1. Abra **Configurações > Aplicativos** e o cadastro de **servidor MCP personalizado** (o nome do menu pode mudar).
2. Preencha **Nome:** `Pórtico MCP` e **Descrição:** `Conecta IAs a um computador Linux com acesso controlado e aprovações.`.
3. Em **Conexão**, escolha **URL do servidor** e informe o endereço `https://SEU-DOMINIO/mcp`.
4. Em **Autenticação**, selecione **OAuth**. Conserve as opções avançadas descobertas automaticamente, salvo instrução de erro explícita do cliente.
5. Termine o cadastro, siga o redirecionamento OAuth, faça login com o usuário e senha do operador e confirme a autorização.
6. Habilite o Pórtico no chat e peça: `Call system.info on my VPS MCP and tell me the hostname.`
7. No terminal da VPS onde `bash scripts/connect-chatgpt.sh` aguarda, pressione **Enter** para verificar se a chamada autenticada foi registrada pela auditoria do Broker.

**Importante:** o instalador só imprime `INSTALLATION COMPLETE` quando a chamada real for detectada. Cadastrar o servidor e abrir a tela de login **não** bastam.

Guia anterior mais detalhado: [ChatGPT e Pórtico](chatgpt-integration.pt-BR.md).

## Claude (Web / Desktop / Cowork)

**Estado:** o fornecedor documenta conectores MCP remotos, mas **não há E2E Pórtico + Claude comprovado neste repositório**.

1. Abra **Personalizar > Conectores** e **Adicionar conector personalizado**.
2. Informe o nome `Pórtico MCP` e `https://SEU-DOMINIO/mcp`.
3. Revise a descoberta OAuth e escolha um método de autenticação compatível com as opções oferecidas pelo Claude. Se a opção de registro automático de cliente OAuth estiver disponível, teste-a conforme as instruções oficiais.
4. Faça login no Pórtico, ative o conector na conversa e teste uma chamada somente de leitura, como `system.info`.
5. Registre a evidência do resultado antes de declarar esse cliente validado.

[Documentação oficial do Claude (PT-BR)](https://support.claude.com/pt/articles/11175166-comece-com-conectores-personalizados-usando-mcp-remoto).

## Cursor

**Estado:** o Cursor anuncia MCP remoto por Streamable HTTP e OAuth, mas **E2E com Pórtico ainda não validado**.

1. Abra as configurações MCP do Cursor.
2. Cadastre um servidor remoto chamado `portico` usando a URL `https://SEU-DOMINIO/mcp`, com transporte HTTP remoto (não `stdio`).
3. Realize a autenticação OAuth caso seja solicitada e verifique se as ferramentas são descobertas.
4. Teste `system.info` e registre o resultado. Não presuma compatibilidade se o fluxo OAuth for recusado.

[Documentação oficial do Cursor sobre MCP](https://docs.cursor.com/context/model-context-protocol).

## Visual Studio Code / GitHub Copilot

**Estado:** o VS Code documenta servidores MCP HTTP com OAuth, mas **E2E com Pórtico ainda não validado**.

Pela paleta de comandos, execute **MCP: Add Server** e escolha um servidor **HTTP remoto**, ou crie a configuração de workspace `.vscode/mcp.json`:

```json
{
  "servers": {
    "portico": {
      "type": "http",
      "url": "https://SEU-DOMINIO/mcp"
    }
  }
}
```

A descoberta OAuth poderá abrir uma janela de autorização. Confirme as ferramentas e faça um teste de leitura `system.info`. **Não adicione Client ID/Secret fixos sem necessidade.**

[Guia oficial de configuração VS Code](https://code.visualstudio.com/docs/agent-customization/mcp-servers) · [Referência do arquivo MCP](https://code.visualstudio.com/docs/agents/reference/mcp-configuration).

## Hosts locais com MCP e outras IAs

O Pórtico adota arquitetura **agnóstica de modelo e fornecedor**. Um LLM/API isolado precisa de um **host de agente/cliente** que implemente o transporte, as ferramentas e a autenticação MCP compatíveis. Um runtime local de agentes, inclusive o futuro host Hermes/Nara, pode ser avaliado pela mesma URL MCP ou por adaptador local aprovado; **a integração Pórtico + Hermes/Nara ainda não possui E2E certificado**.

Para cada novo host, registrar versão, transporte, OAuth/identidade do cliente e ferramentas descobertas; executar `system.info` autenticado e leitura permitida; verificar evento correspondente no Broker e uma negativa de política; acrescentar evidências à [matriz de compatibilidade](compatibility.md). Nunca desativar TLS, OAuth ou a política local para forçar compatibilidade.

## Diagnóstico e segurança

Se ocorrer falha de conexão, confira DNS, HTTPS válido, endpoint terminando em `/mcp`, seleção de OAuth e autorização efetivamente concluída. Não enfraqueça a configuração para HTTP sem TLS ou tokens colados no chat para contornar o problema. A política de pastas, o escopo e os registros de auditoria precisam continuar ativos.

**Legenda de validação:** *suporte do cliente documentado* significa apenas que o fabricante anuncia o recurso; *Pórtico E2E verificado* exige uma conexão autenticada real e evidência de chamada auditada, não somente um teste de descoberta OAuth.

Acompanhe o [ensaio de instalação #153](https://github.com/josemirmoura/portico-community/issues/153) e as [melhorias deste guia #159](https://github.com/josemirmoura/portico-community/issues/159).
