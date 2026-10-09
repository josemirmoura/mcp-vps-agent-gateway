# Compatibilidade da Central de Autorizações

Revisão das fontes: **2026-10-09 UTC**. Esta é uma matriz de evidências da alteração candidata de autorizações adaptativas, não uma promessa de suporte por marca, plano ou versão móvel. Produção não foi modificada nesta validação.

## O que cada evidência demonstra

| Evidência | Demonstra | Não demonstra |
|---|---|---|
| Documentação primária do fornecedor | Recurso anunciado e condições descritas pelo fornecedor | Que a conta conectada anuncia a capacidade, aceita nosso iframe ou mantém cookies |
| Sonda de protocolo | Discover/initialize, catálogo, metadados e recurso HTML recebidos para capacidades simuladas | Renderização real, elicitation exibida ou identidade do proprietário |
| SDK oficial e harness de bridge | Negociação/continuação e mensagens exercitadas em testes controlados | Funcionamento no aplicativo de um fornecedor |
| Navegador com harness | Comportamento do HTML e isolamento nas origens locais do teste | Origem, CSP, cookies e permissões efetivas de ChatGPT/Claude/VS Code |
| Cliente real com staging | Experiência observada naquela conta, cliente, versão e endpoint | Compatibilidade de todas as contas, clientes ou máquinas |

## Matriz por cliente e superfície

| Cliente/superfície | MCP Apps documentado | Elicitation documentada | Pórtico adaptativo verificado nessa superfície | Degradação esperada até prova real |
|---|---|---|---|---|
| ChatGPT Web | Sim; `_meta.ui.resourceUri`, MIME MCP App e CSP padrão [1] | As fontes aqui consultadas não comprovam os modos anunciados pela conexão | **Pendente em staging**. A conexão MCP existente responde a inspeções de produção; isso não testa o código candidato nem uma App | HTTPS/SSH; Apps só com capacidade real, handshake e enquadramento aceitos |
| ChatGPT móvel/desktop | Nenhuma extensão automática da evidência Web | Não verificado por este trabalho | **Pendente por superfície e conta** | HTTPS responsivo/SSH |
| Claude Web, Desktop, Cowork e iOS/Android | Conectores interativos documentados nessas superfícies; sandbox/CSP descritos [2] | Não extrapolar elicitation de Claude Code para esses produtos | **Pendente**; documentação de conectores publicados não comprova nosso conector personalizado nem iframe aninhado | HTTPS/SSH; observar capacidades e CSP efetivos |
| Claude Code interativo | A documentação trata recursos `ui://` como recursos de UI e os filtra das sugestões de contexto; isso não prova renderização do Pórtico [3] | Form e URL, com `elicitation: {form: {}, url: {}}` documentado para MCP 2026-07-28 [3] | **Pendente**; nenhuma sessão Claude Code foi executada | Elicitation de navegação quando anunciada; decisão no HTTPS/SSH |
| VS Code com agente MCP | Apps inline documentado [4] | Implementação primária contém o serviço de elicitation [5]; conferir modo/versão efetivos | **Pendente**; nenhum VS Code real foi executado | Apps ou navegação compatível; HTTPS/SSH se o host restringir |
| Goose Desktop | Goose é listado entre os hosts MCP Apps no anúncio do projeto MCP [6] | Guia do fornecedor documenta form [7] | **Pendente**; nenhuma instalação real Goose foi executada | Apps se anunciado e utilizável; form de navegação/HTTPS/SSH |
| Goose CLI | Não presumir UI gráfica porque o Desktop tem Apps | Guia documenta form no terminal [7] | **Pendente** | Form de navegação/HTTPS; CLI interativa por SSH sem navegador |
| Agente local/headless ou cliente desconhecido | Só se a conexão declarar a extensão MIME correta | Só nos modos declarados; `{}` equivale a form [8] | Catálogo e transporte podem ser sondados; cliente específico não verificado | HTTPS se disponível; CLI/SSH |
| Cliente de teste Go SDK v1.8.0 | Perfis com/sem extensão exercitados em testes do Gateway | Continuação accept/decline/cancel exercitada sem produzir autorização | **Harness do repositório**, sujeito ao resultado CI do commit revisado | Mesma seleção do runtime; não equivale a fornecedor real |

O suporte MCP Apps geral não garante que o host permita **iframe aninhado do operador**. OpenAI documenta `frameDomains` opcional e revisão adicional de iframe [1]. O Pórtico exige também `hostCapabilities.sandbox.csp.frameDomains` no handshake; origens ancestrais são configuradas somente depois de observadas. Não adotar uma origem sandbox presumida como prova de compatibilidade.

## Sinais de capacidade usados pelo runtime

| Sinal | Interpretação |
|---|---|
| `extensions["io.modelcontextprotocol/ui"].mimeTypes` contém exatamente `text/html;profile=mcp-app` | Candidato MCP Apps; ainda precisa de recurso habilitado, portal e handshake utilizável [9] |
| Extensão ausente, objeto inválido ou somente `text/html` | Sem MCP Apps |
| `elicitation: {form: {}}` ou legado `elicitation: {}` | Form suportado; serve somente para intenção/navegação, sem senha ou decisão do Broker |
| `elicitation: {url: {}}` | URL suportada; não implica suporte form |
| Capacidade não anunciada | Não enviar elicitation nem inferir pelo nome do cliente |
| `ui/initialize` com protocolo UI `2026-01-26` | Handshake MCP Apps; distinto da versão do protocolo MCP principal |
| `hostCapabilities.serverTools` como objeto válido | Permite consultar `permissions.approval_status` pelo host; ausente ou malformado mantém consulta pelo portal/SSH |
| `hostCapabilities.openLinks` como objeto válido | Permite solicitar `ui/open-link` após clique explícito; ausente ou malformado mantém URL visível |
| Handshake sem o enquadramento necessário, timeout, frame/cookie bloqueado | Manter link HTTPS e orientação SSH; não tratar como autorização |

O SDK fixado suporta MCP **2026-07-28**, com `server/discover` e capacidades em `params._meta["io.modelcontextprotocol/clientCapabilities"]` **a cada solicitação** [8,10]. Para **2025-11-25**, usa `initialize.params.capabilities` e a inicialização legado [11]. A sonda testa ambas separadamente; erro em uma revisão não é ocultado por negociação automática. O runtime deixa negociação e compatibilidade de transporte ao SDK oficial.

Nome do cliente, texto da IA, `userAgent`, aceitar uma elicitation e `visibility: ["app"]` não são provas de identidade. O proprietário autentica-se na origem HTTPS do operador, e o Broker verifica novamente sujeito, máquina, pedido, escopo, prazo e política. Nenhuma capacidade de UI concede permissão.

## Sonda sem decisões

`scripts/approval-client-probe.py` usa somente `server/discover`/`initialize`, `notifications/initialized`, `tools/list` e leitura do recurso fixo `ui://portico/authorizations-v1.html`. Não chama tools, cria pedidos, responde a elicitation, executa HTML ou visita o portal. Aceita JSON e SSE, catálogo paginado e sessões legado. Recusa redirects, URIs arbitrárias, respostas maiores que 1 MiB, cursores cíclicos e callbacks inesperados. O relatório contém somente campos selecionados, hash do HTML e códigos de erro; omite endpoint, bearer, sessão, HTML, instruções e textos de erro do servidor.

Exemplo em **staging separado**, com bearer temporário colocado no ambiente pelo operador:

```bash
python3 scripts/approval-client-probe.py \
  --url https://staging.example.org/mcp \
  --staging --token-env PORTICO_PROBE_TOKEN \
  --profile all --protocol both
```

Não passar token como argumento, não copiar credenciais da produção e não apresentar o JSON como prova de cliente real. HTTP só é permitido em loopback. `--staging` declara que o endpoint remoto é de teste; não detecta se o endereço é produção. Sem token configurado, 401/403 demonstra barreira de autenticação, não ausência de suporte MCP Apps.

O comando testa sete perfis: texto, Apps, form, form legado, URL, Apps+elicitation e MIME inválido. Como não solicita permissão, **não testa a seleção final do canal nem abre uma confirmação**. Para isso, usar os testes do Gateway e a aceitação real abaixo.

Os **20 testes automatizados da sonda passaram localmente** nesta revisão: endpoints/credenciais, metadados por solicitação, sessão legado, JSON/SSE, paginação, redirects, autorização ausente, resposta com ID errado ou ambígua, limites de tamanho, URIs arbitrárias, Unicode inválido e rejeição de callbacks. A execução completa dos sete perfis nas duas revisões produziu **14 verificações de metadados válidas e zero chamadas a tools**, em 49 requisições HTTP. Esses testes usam um servidor TCP de fixture, não um cliente de fornecedor nem um Broker real.

## Aceitação real ainda exigida

Para cada conta/superfície, registrar data UTC, versão do cliente quando disponível, commit do staging, revisão MCP, perfil anunciado e resultado observado. Não armazenar tokens, senhas, cookies, fingerprints de credenciais ou payloads brutos. Usar pedidos temporários em diretório de staging sem dados sensíveis.

1. Confirmar ausência de grant antes do pedido e registrar canal retornado; aceitar uma elicitation isolada deve manter pedido pendente.
2. Quando houver Apps, observar o handshake, CSP e cadeia de origens; conferir se o iframe do operador realmente abre. Se não abrir, comprovar que o link HTTPS continua utilizável.
3. Autenticar o operador na Central, conferir máquina/sujeito/escopo/prazo e tomar uma decisão explícita. Repetir um pedido read deve aproveitar a sessão; work/compose/arquivo protegido deve exigir verificação fresca.
4. Negar outro pedido, tentar decisão duplicada, deixar um expirar e consultar o resultado. Nenhum desses casos deve gerar grant. Cancelar pelo mesmo sujeito e consultar com outro sujeito deve respeitar o isolamento.
5. Confirmar que um grant no nó de staging não autoriza outro nó. Encerrar sessão, verificar revogação e testar portal fora do chat e CLI interativa.
6. Repetir em tela móvel quando a superfície existir. Guardar somente evidências sanitizadas e marcar separadamente renderização, sessão, decisão do Broker e fallback.

Até existir essa evidência, ChatGPT/Claude/VS Code/Goose permanecem **documentados pelo fornecedor e não validados para esta Central candidata**. A inspeção real `system.info`/saúde do endpoint atual confirma conectividade do MCP já instalado; não confirma implantação ou renderização das novas modalidades.

## Fontes primárias

Consultadas em 2026-10-09 UTC; disponibilidade de produto pode mudar. Esta matriz usa o conteúdo das páginas, não alegações de terceiros.

1. [OpenAI: referência MCP Apps, metadados/CSP e iframe](https://developers.openai.com/plugins/reference); [UI para ChatGPT](https://developers.openai.com/plugins/build/chatgpt-ui).
2. [Claude: use interactive connectors](https://support.claude.com/en/articles/13454812-use-interactive-connectors-in-claude), página datada de 2026-08-11 na consulta.
3. [Claude Code: elicitation e recursos MCP](https://code.claude.com/docs/en/mcp).
4. [VS Code: MCP Apps](https://code.visualstudio.com/blogs/2026/01/26/mcp-apps-support), anúncio de 2026-01-26; [capacidades MCP](https://code.visualstudio.com/docs/agent-customization/mcp-servers).
5. [VS Code: serviço de elicitation no código primário](https://github.com/microsoft/vscode/blob/main/src/vs/workbench/contrib/mcp/browser/mcpElicitationService.ts), branch móvel `main`; não fixa a versão instalada pelo usuário.
6. [Projeto MCP: anúncio MCP Apps](https://blog.modelcontextprotocol.io/posts/2026-01-26-mcp-apps/), 2026-01-26.
7. [Goose: MCP elicitation](https://goose-docs.ai/docs/guides/mcp-elicitation/).
8. [MCP 2026-07-28: elicitation e capacidades por solicitação](https://modelcontextprotocol.io/specification/2026-07-28/client/elicitation).
9. [MCP Apps: especificação UI 2026-01-26](https://github.com/modelcontextprotocol/ext-apps/blob/main/specification/2026-01-26/apps.mdx).
10. [Go SDK v1.8.0: protocolo e chaves `_meta`](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/protocol.go); [revisões suportadas](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/shared.go).
11. [MCP 2025-11-25: elicitation legado](https://modelcontextprotocol.io/specification/2025-11-25/client/elicitation).
