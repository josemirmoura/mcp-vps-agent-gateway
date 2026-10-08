# Operador: aprovação segura sem janela nativa MCP

Se o cliente de IA não anunciar a capability de *MCP elicitation*, o
`permissions.request_root_access` ou `permissions.request_sensitive_access`
cria um pedido **pendente**. O Gateway não libera acesso e **não** compartilha
o token de confirmação com o modelo. Isso é esperado e deve falhar fechado.

O ChatGPT pode mostrar `approval_method=operator_fallback` e `request_id=apr_...`.
Esse resultado NÃO é uma aprovação. O operador tem duas opções:

1. Utilizar um cliente MCP que anuncie elicitation, exibindo sua própria
   confirmação nativa.
2. Na VPS, abrir uma sessão SSH confiável e usar a confirmação interativa
   do Pórtico, autenticada pelo Docker/Broker.

## Operação por SSH (Community)

No terminal SSH do proprietário da instalação:

```bash
cd /home/jmour/mcp-vps-agent-gateway
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

## Critérios de aceitação para futura UX web

O portal web de aprovação do operador deverá ter autenticação independente,
autorização do sujeito, validação de sessão/CSRF, expiração, tela legível em
mobile, confirmação explícita de perfil/escopo/TTL e auditoria; o navegador
não pode receber segredo administrativo ou `approval_token` do Broker.
A janela nativa do GPT só será considerada suportada quando o cliente
anunciar elicitation e executar uma confirmação real, não apenas após
atualizar o catálogo de ferramentas.
