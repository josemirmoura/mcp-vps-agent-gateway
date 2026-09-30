# FAQ

## Whole Host significa Full?

Não. Whole Host define `/` como teto físico do filesystem do Broker. Full é um bundle explícito de capacidades controlado por policy/feature gates, desabilitado por padrão.

## Full implica Internet irrestrita?

Não. Rede irrestrita é uma capacidade/aprovação separada.

## O ChatGPT recebe minha senha da VPS ou chave SSH?

Não. A instalação suportada é executada pelo operador no terminal da VPS. O ChatGPT recebe o endpoint MCP público e completa OAuth. Credenciais VPS/SSH não fazem parte do fluxo.

## Por que o Broker é privilegiado?

systemd do host, Docker e operações delegadas do filesystem exigem um componente privilegiado confiável. O Broker é local-only e autoritativo em policy; o Gateway remoto continua non-root e sem Docker socket/host root.

## Docker é a fronteira de segurança?

Não. Docker é mecanismo de packaging/lifecycle. A autorização server-side do Broker é a fronteira efetiva.

## Há telemetria?

O projeto não tem cliente próprio de analytics/tracking. A telemetria do ZITADEL embutido é explicitamente desabilitada. Veja `privacy.md`.

## Posso limitar a um projeto?

Sim. Escolha **Project**.

## Qual o padrão recomendado?

**Standard**. Usa `/opt` como teto físico, começa sem raiz de projeto autorizada e permite aprovações específicas depois pelo ChatGPT.

## Posso delegar vários diretórios?

Sim, dinamicamente abaixo do teto. Operadores avançados também podem usar raízes estáticas.

## Posso gerenciar várias VPS?

Sim. Cada instalação tem identidade, credenciais, estado e cadeia de auditoria próprios.

## Outro cliente MCP pode usar?

O core usa MCP Streamable HTTP baseado em padrões. O caminho público é validado com ChatGPT Web, mas clientes compatíveis podem usar o mesmo servidor se suportarem autenticação necessária.

## Quando a instalação termina?

Somente quando uma chamada real atravessa autenticação, Gateway, Broker, policy, execução e auditoria. Saúde dos contêineres não basta.

## Por que update.sh evita main?

`main` é desenvolvimento. Produção deve seguir tags SemVer estáveis.

## Posso remover o MCP sem apagar minhas apps?

Sim. Safe remove e purge removem artefatos do Portico e preservam os recursos que ele administrava.
