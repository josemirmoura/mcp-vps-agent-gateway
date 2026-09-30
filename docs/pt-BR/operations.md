# Operações

Este documento separa observabilidade operacional de auditoria de segurança.

## Status

~~~bash
bash scripts/diagnose.sh status
~~~

Mostra versão do produto, estado dos serviços Compose, saúde do Broker/Gateway, contagem de reinícios, snapshot de saúde do Broker e estado da cadeia de auditoria.

Use primeiro para responder: **o sistema está saudável?**

## Saúde

~~~bash
bash scripts/diagnose.sh health
~~~

Retorna o snapshot de saúde do Broker.

## Logs

~~~bash
bash scripts/diagnose.sh logs 200
~~~

Mostra logs recentes com valores secretos configurados redigidos. Ajuda a investigar reinícios, falhas de autenticação, perda de conexão Gateway/Broker, timeouts e contêineres unhealthy.

## Auditoria

~~~bash
bash scripts/diagnose.sh audit 100
~~~

Responde quem invocou, qual tool/recurso/ação, allow/deny, resultado, sequência e validade da cadeia de hash. Logs operacionais e auditoria são camadas distintas.

## Bundle de diagnóstico

~~~bash
bash scripts/diagnose.sh bundle
~~~

Inclui runtime/versão, saúde, status/tail da auditoria, logs recentes, versões Docker/Compose e snapshot redigido da policy. O arquivo é criado 0600; revise antes de compartilhar.

## Versão e atualização

~~~bash
bash scripts/version.sh
bash scripts/version.sh --check
~~~

O primeiro mostra a versão instalada. `--check` atualiza tags quando possível e informa current/ahead/divergent/update available.

## Atualização

~~~bash
bash scripts/update.sh
~~~

Por padrão usa o SemVer estável mais recente. O updater:

1. exige working tree rastreada limpa;
2. mostra versão atual/alvo e resumo das mudanças;
3. recusa non-fast-forward;
4. para o pacote;
5. faz backup de `.env`, policy, state e volumes de identidade;
6. valida migração numa cópia do banco de estado;
7. constrói e verifica o alvo;
8. roda verificação pública com auth integrada;
9. faz rollback automático em falha;
10. registra em `state/update.log`.

RC explícito:

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~

## Remoção segura

~~~bash
bash scripts/remove.sh safe
~~~

Revoga grants temporários, remove o runtime e gira credenciais locais, preservando configuração, policy, estado/auditoria e identidade integrada. Recursos administrados permanecem intactos.

## Reinstalação

~~~bash
bash scripts/install.sh
~~~

O instalador reutiliza o estado/configuração preservados.

## Purge

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

O purge remove somente artefatos do Portico. O checkout fica preservado por padrão. Para apagá-lo:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

O purge não apaga arbitrariamente projetos delegados, aplicações, sites, bancos, imagens/contêineres de terceiros, serviços ou arquivos.
