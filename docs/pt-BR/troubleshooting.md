# Solução de problemas

Comece com:

~~~bash
bash scripts/diagnose.sh status
~~~

Se não estiver claro:

~~~bash
bash scripts/diagnose.sh bundle
~~~

Revise o bundle antes de compartilhar.

## Diretório de escopo não existe

O instalador não cria diretórios arbitrários silenciosamente:

~~~bash
sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 /opt/my-app
~~~

Ou rode novamente o instalador guiado e aprove a etapa sudo visível.

## Path rejeitado por ser não canônico ou symlink

Use path absoluto canônico, sem segmentos ponto ou ancestrais symlink. Isso é hardening intencional da fronteira do filesystem.

## Broker ou Gateway nunca fica healthy

~~~bash
docker compose ps -a
docker compose logs --no-color --tail 100 broker gateway
bash scripts/diagnose.sh status
~~~

Não ignore o health gate. Corrija o serviço e repita o instalador.

## Portas 80/443 já em uso

A auth integrada reutiliza um Traefik existente quando pode identificá-lo com segurança. Se outro web server ocupa 80/443 e não há Traefik reutilizável, o setup falha em vez de substituí-lo.

## Múltiplos Traefik

~~~bash
bash scripts/setup-integrated-auth.sh --edge-network YOUR_NETWORK
~~~

Se o resolver de certificado for ambíguo:

~~~bash
--certresolver YOUR_RESOLVER
~~~

## DNS não resolve

Crie/corrija A/AAAA para o hostname MCP, aguarde propagação e rode novamente.

## Verificação OAuth pública falha

~~~bash
bash scripts/verify-public.sh
~~~

Confira DNS, certificado, hostname/issuer, metadata do recurso protegido, DCR e PKCE S256 no discovery OIDC, portas e rotas de proxy. Não enfraqueça issuer/audience para fazer o teste passar.

## Conexão com ChatGPT não chega

~~~bash
bash scripts/connect-chatgpt.sh
~~~

Confira app MCP, OAuth, seleção do Portico e peça ao ChatGPT para chamar `system.info`. Timeout não marca instalação concluída.

## update.sh diz que não há release estável

Antes do primeiro tag estável, não há target automático por design.

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~

`main` não é canal automático de produção.

## Update faz rollback

Não apague o backup. Revise saída, `backups/<timestamp>/migration-check.json`, `state/update.log` e diagnóstico. O rollback é recurso de segurança.

## Segredo aparece num diagnóstico

Não compartilhe o artefato. Preserve localmente, gire credenciais potencialmente expostas e siga `SECURITY.md`.
