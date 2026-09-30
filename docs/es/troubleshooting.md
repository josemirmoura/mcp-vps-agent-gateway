# Solución de problemas

Empieza con:

~~~bash
bash scripts/diagnose.sh status
~~~

Si no es evidente:

~~~bash
bash scripts/diagnose.sh bundle
~~~

Revisa el bundle antes de compartirlo.

## El directorio de scope no existe

Portico no crea silenciosamente directorios arbitrarios. Créalo explícitamente:

~~~bash
sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 /opt/my-app
~~~

O vuelve a ejecutar el instalador y aprueba el paso sudo visible.

## Path no canónico o con symlink

Usa un path absoluto canónico, sin segmentos punto ni ancestros symlink. Es hardening deliberado del límite de filesystem.

## Broker/Gateway no se vuelven healthy

~~~bash
docker compose ps -a
docker compose logs --no-color --tail 100 broker gateway
bash scripts/diagnose.sh status
~~~

No saltes el health gate. Corrige el servicio y vuelve a ejecutar el instalador.

## Puertos 80/443 ocupados

La auth integrada reutiliza un único Traefik existente cuando puede identificarlo con seguridad. Si otro servidor web ocupa 80/443 y no hay Traefik reutilizable, el setup falla en vez de reemplazarlo.

## Varios Traefik

~~~bash
bash scripts/setup-integrated-auth.sh --edge-network YOUR_NETWORK
~~~

Si el resolver ACME es ambiguo:

~~~bash
--certresolver YOUR_RESOLVER
~~~

## DNS no resuelve

Crea/corrige el registro A/AAAA del hostname MCP y espera propagación. El instalador valida DNS antes de continuar.

## Falla la verificación OAuth pública

~~~bash
bash scripts/verify-public.sh
~~~

Comprueba DNS, certificado, hostname/issuer, metadata del recurso protegido, discovery OIDC con DCR y PKCE S256, y rutas del proxy. No debilites issuer/audience para hacer pasar la prueba.

## ChatGPT no conecta

~~~bash
bash scripts/connect-chatgpt.sh
~~~

Comprueba que el workspace permita apps MCP personalizadas, completa OAuth con la cuenta dedicada y pide a ChatGPT llamar `system.info`. Un timeout no marca instalación completa.

## update.sh dice que no existe release estable

Antes del primer tag estable no hay objetivo automático por diseño. Para un RC:

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~

`main` no es canal automático de producción.

## Update hace rollback

No borres el backup. Revisa la salida, `backups/<timestamp>/migration-check.json`, `state/update.log` y `bash scripts/diagnose.sh status`. El rollback es una función de seguridad.

## Aparece un secreto en un diagnóstico

No compartas el artefacto. Consérvalo localmente, rota las credenciales potencialmente expuestas y sigue `SECURITY.md`.
