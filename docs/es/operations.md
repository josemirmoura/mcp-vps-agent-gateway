# Operaciones

Este documento separa observabilidad operativa de auditoría de seguridad.

## Estado

~~~bash
bash scripts/diagnose.sh status
~~~

Muestra versión, estado Compose, salud de Broker/Gateway, reinicios de contenedores, snapshot de salud del Broker y estado de la cadena de auditoría. Es el primer comando para responder: **¿está sano el sistema?**

## Salud

~~~bash
bash scripts/diagnose.sh health
~~~

Devuelve el snapshot de salud del Broker.

## Logs

~~~bash
bash scripts/diagnose.sh logs 200
~~~

Muestra logs recientes con valores secretos configurados redactados. Sirven para investigar reinicios, fallos de autenticación, pérdida de conectividad Gateway/Broker, timeouts o contenedores unhealthy.

## Auditoría

~~~bash
bash scripts/diagnose.sh audit 100
~~~

Responde quién ejecutó qué herramienta/recurso/acción, si fue permitido o denegado, resultado, secuencia e integridad de hash. Logs y auditoría tienen objetivos distintos.

## Bundle de diagnóstico

~~~bash
bash scripts/diagnose.sh bundle
~~~

Incluye versión/runtime, salud, estado/tail de auditoría, logs recientes, versiones Docker/Compose y snapshot redactado de policy. Se crea con modo 0600. Revísalo antes de compartirlo aunque CI compruebe que los tokens configurados no estén presentes.

## Versión y actualizaciones

~~~bash
bash scripts/version.sh
bash scripts/version.sh --check
~~~

El primer comando muestra la versión instalada. `--check` actualiza tags cuando es posible e informa si el checkout está actual, adelantado respecto al estable, divergente o tiene una actualización estable.

## Actualizar

~~~bash
bash scripts/update.sh
~~~

Por defecto apunta al SemVer estable más reciente. El actualizador:

1. exige working tree limpio;
2. muestra versión actual/objetivo y cambios;
3. rechaza targets non-fast-forward;
4. detiene el paquete;
5. respalda `.env`, policy, estado e identidad integrada;
6. prueba la migración sobre una copia del DB de estado;
7. construye y verifica el target;
8. ejecuta verificación pública con auth integrada;
9. revierte código/estado/identidad automáticamente si falla;
10. registra el resultado en `state/update.log`.

Para un RC explícito:

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~

## Eliminación segura

~~~bash
bash scripts/remove.sh safe
~~~

Revoca delegaciones temporales, retira el runtime y rota credenciales locales activas, conservando configuración, policy, estado/auditoría e identidad integrada. Los recursos administrados permanecen intactos.

Reinstalación:

~~~bash
bash scripts/install.sh
~~~

## Purge

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Elimina únicamente artefactos propios de Portico. El checkout se conserva salvo segunda confirmación:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

La eliminación del source solo se permite si el directorio actual se verifica como checkout de este proyecto. Nunca se borran arbitrariamente proyectos delegados, apps, sitios, bases de datos, imágenes/contenedores de terceros, servicios o archivos administrados.
