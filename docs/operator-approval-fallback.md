# Operador: aprovação segura sem janela nativa MCP

Mesmo se o cliente de IA anunciar MCP Apps ou *MCP elicitation*, o
`permissions.request_root_access` ou `permissions.request_sensitive_access`
cria um pedido **pendente**. O Gateway não libera acesso e **não** compartilha
credenciais de aprovação com o modelo. Abrir a interface ou aceitar elicitation
não autentica o proprietário nem concede acesso.

Clientes podem mostrar `request_id=apr_...`, canal escolhido e um link específico
em `operator_approval_url`. Esse resultado NÃO é uma aprovação. Na implementação
adaptativa candidata da [PR #93](https://github.com/josemirmoura/mcp-vps-agent-gateway/pull/93),
o operador dispõe das seguintes apresentações, conforme capacidades verificadas:

1. Central embutida por MCP Apps, quando handshake, enquadramento e sessão
   funcionarem; autenticação e decisão ocorrem na origem HTTPS do operador.
2. Elicitation nativa de navegação, quando anunciada. Seu aceite mantém o
   pedido pendente até a decisão autenticada na Central ou no CLI.
3. Link HTTPS específico para a Central responsiva, com login do operador.
   Uma sessão válida evita repetir senha para leitura temporária de pasta;
   arquivos protegidos e work/compose exigem verificação fresca por pedido.
4. Na VPS, abrir uma sessão SSH confiável e usar a confirmação interativa
   do Pórtico, autenticada pelo Docker/Broker.

Sem portal ou para permissões permanentes, administrativas ou de todo o limite
físico, usar o CLI confiável. A [matriz de compatibilidade](approval-compatibility.md)
separa recursos documentados, testes controlados e clientes reais ainda pendentes.

## Operação por SSH (Community)

No terminal SSH do proprietário da instalação:

```bash
# Ajuste este caminho caso tenha instalado o Pórtico em outro diretório.
cd ~/mcp-vps-agent-gateway
python3 scripts/operator-approvals.py
```

Para revisar somente os pedidos abertos, sem oferecer decisão:

```bash
python3 scripts/operator-approvals.py --list
```

Para selecionar o pedido exato mostrado na conversa:

```bash
python3 scripts/operator-approvals.py --request apr_EXEMPLO_DE_ID
```

O programa só apresenta pedidos pendentes **e não expirados** do tipo pasta
ou arquivo protegido. Exibe identidade MCP, destino, perfil, duração da
delegação e validade do pedido. Uma aprovação de todo o limite físico
recebe aviso adicional. A decisão exige um terminal interativo e a frase
exata `APROVAR apr_...` ou `NEGAR apr_...`. Enter, texto diferente, EOF,
falta de Docker/Broker e pedidos expirados cancelam sem conceder autoridade.

O programa invoca o comando administrativo `vps-agent approve/deny`
**dentro do contêiner Broker**, cujo token de administrador permanece no
ambiente desse contêiner. Não solicita nem imprime tokens, credenciais
OAuth ou dados do arquivo `.env`. O Broker verifica novamente escopo,
identidade e estado do pedido, impede decisões repetidas e registra
as decisões na trilha de auditoria.

**Atenção:** quem tiver permissão para executar comandos Docker no host
já possui uma autoridade administrativa poderosa. Não ofereça acesso
Docker, SSH ou o token de administrador ao modelo; não copie arquivos
`.env` para o chat. O fluxo local SSH é um *fallback*, não um pop-up
padrão do ChatGPT. Fluxos de elevação geral `permissions.request_elevation`
continuam separados e não aparecem no menu simplificado.

Para conferir a autorização depois:

```text
permissions.list_root_access
permissions.list_sensitive_access
```

Pedidos de aprovação expiram, por padrão, após cerca de 10 minutos; nesse
caso solicite um novo pedido. A delegação temporária tem duração própria,
diferente da validade do pedido.

## Critérios de aceitação da Central adaptativa

O portal web de aprovação do operador exige autenticação independente,
autorização do sujeito, validação de sessão/CSRF, expiração, tela legível em
mobile, confirmação explícita de perfil/escopo/TTL e auditoria; o navegador
não recebe segredo administrativo ou `approval_token` do Broker. A decisão,
concessão e auditoria são confirmadas atomicamente pelo Broker, e cada execução
posterior é reautorizada. Consultar `permissions.approval_status` após perda de
resposta, antes de tentar uma nova decisão.

A presença de catálogo, Apps ou elicitation não comprova funcionamento em um
produto específico. Validar o cliente real em staging, com operador autenticado,
aprovação, negação e fallback observados. Ver [arquitetura](adaptive-operator-approvals.md)
e [procedimento de ativação/rollback](adaptive-approvals-rollout.md).
