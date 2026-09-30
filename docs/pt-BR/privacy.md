# Privacidade, dados e telemetria

## Localização dos dados

O pacote é auto-hospedado na VPS do operador. Estado operacional, policy, histórico de auditoria e dados de identidade integrada permanecem nessa VPS, salvo exportação deliberada pelo operador.

## Audit

Os registros de auditoria do Broker registram quem invocou uma tool, qual recurso/ação foi solicitado, se foi permitido e o estado de sequência/integridade resultante.

Audit é diferente de logs comuns de serviço.

## Logs

Logs do Gateway, Broker, Docker e serviços de identidade podem conter timestamps, identificadores de instância, nomes de tools, erros e contexto de requisição. Bundles de diagnóstico redigem segredos configurados e são testados na CI contra vazamento de tokens, mas ainda devem ser tratados como dados potencialmente sensíveis.

## ChatGPT / MCP

Dados necessários para atender uma requisição MCP podem passar entre ChatGPT, ou outro cliente MCP, e o Gateway. O operador controla a autoridade server-side pela policy. Saída do modelo e conteúdo remoto não são autorização. Não exponha segredos em arquivos genéricos, shell ou resultados de tools.

## Secrets

Credenciais locais ficam em configuração controlada por root/operador ou volumes privados. A instalação suportada nunca pede senhas VPS, chaves SSH privadas, senhas root ou segredos de infraestrutura não relacionados. A credencial OAuth dedicada é separada das credenciais VPS/SSH.

## Telemetry

O projeto não contém cliente próprio de analytics ou rastreamento. A telemetria do ZITADEL embutido é desabilitada com `ZITADEL_TELEMETRY_ENABLED=false`. O pacote não introduz tracking de marketing.

## Removal

A remoção segura preserva configuração, estado e histórico de auditoria, remove o runtime MCP ativo e gira credenciais locais. O purge completo remove artefatos locais pertencentes ao MCP e preserva aplicações, sites, bancos, contêineres de terceiros, serviços e arquivos administrados.

~~~bash
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~
