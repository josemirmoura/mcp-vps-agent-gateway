# Múltiplas instâncias VPS no mesmo workspace do ChatGPT

Este é um cenário avançado. O onboarding normal continua sendo uma instância do Portico MCP por instalação.

## Modelo de identidade

Os nomes das tools permanecem estáveis entre instalações. Não renomeie tools por servidor.

Cada instalação recebe:

- seu próprio endpoint MCP HTTPS;
- sua própria relação cliente/recurso OAuth e credenciais;
- um `VPS_AGENT_INSTANCE_ID` estável gerado no bootstrap;
- um `VPS_AGENT_INSTANCE_NAME` visível ao operador;
- identidade da instância em `system.info`, eventos de auditoria do Broker e logs estruturados.

Exemplo:

~~~dotenv
VPS_AGENT_INSTANCE_NAME="VPS Agent | Loja"
VPS_AGENT_PUBLIC_URL=https://mcp-loja.example.com/mcp
~~~

Em outra VPS:

~~~dotenv
VPS_AGENT_INSTANCE_NAME="VPS Agent | Blog"
VPS_AGENT_PUBLIC_URL=https://mcp-blog.example.com/mcp
~~~

Registre-as como apps separados no ChatGPT com nomes visíveis correspondentes. Selecione ou @mencione o app desejado em vez de depender de desambiguação silenciosa de tools com nomes iguais.

## Verificação

Para cada app, chame `system.info` e compare `instance_id` e `instance_name` com a VPS esperada. Trate hostname apenas como informação auxiliar: provedores podem reutilizar o mesmo hostname em máquinas descartáveis diferentes.

Depois inspecione a auditoria do Broker na VPS:

~~~bash
bash scripts/diagnose.sh audit 20
~~~

O mesmo `instance_id`/nome deve aparecer nos novos eventos e logs estruturados.

## Uso na mesma conversa

Se a interface atual do ChatGPT permitir múltiplos apps MCP customizados na mesma conversa, selecione ou mencione explicitamente o app pretendido antes de cada operação. O comportamento da UI pode mudar independentemente deste servidor.

## Gate automatizado de uso simultâneo

O repositório possui um workflow dedicado com três máquinas. Ele inicia o pacote Docker real em três runners Ubuntu independentes ao mesmo tempo e reutiliza deliberadamente o mesmo subject, nomes de tools, path lógico e operation id.

O gate só passa quando:

- as três instalações têm IDs e nomes distintos;
- usam credenciais geradas independentes;
- os tempos de vida se sobrepõem;
- o mesmo operation id funciona independentemente nos três states do Broker;
- cada cadeia de auditoria registra apenas sua própria identidade;
- hostnames e heads de auditoria permanecem distintos;
- cada instância termina no estado local esperado.

Esse é o teste server-side de colisão e não exige intervenção humana.

## Superfície do produto ChatGPT

Conectar vários apps vivos ao mesmo workspace é um experimento separado de UI. Não é requisito do onboarding normal de uma única VPS nem do gate final ChatGPT → VPS. Se testado, registre cada VPS como app nomeado separadamente e selecione explicitamente o app correto.
