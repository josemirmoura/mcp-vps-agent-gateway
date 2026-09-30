# Privacidad, datos y telemetría

## Ubicación de los datos

El paquete se autoaloja en la VPS del operador. El estado operativo, la policy, el historial de auditoría y los datos de identidad integrada permanecen allí salvo exportación deliberada.

## Audit

Los registros del Broker muestran quién invocó una tool, qué recurso/acción se solicitó, si fue permitido y el estado de secuencia/integridad resultante.

Audit es distinto de los logs normales del servicio.

## Logs

Los logs de Gateway, Broker, Docker y servicios de identidad pueden contener marcas de tiempo, identificadores, nombres de tools, errores y contexto. Los bundles redactan secretos configurados y se prueban en CI contra filtraciones, pero siguen siendo datos potencialmente sensibles.

## ChatGPT / MCP

Los datos necesarios para una solicitud MCP pueden pasar entre ChatGPT, u otro cliente MCP, y Gateway. La autoridad server-side la controla la policy. La salida del modelo o contenido remoto no constituyen autorización. No expongas secretos en archivos genéricos, shell o resultados de tools.

## Secrets

Las credenciales locales se guardan en configuración controlada por root/operador o volúmenes privados. La instalación nunca pide contraseñas VPS, claves SSH privadas, root ni secretos no relacionados. La credencial OAuth dedicada es distinta de VPS/SSH.

## Telemetry

No hay cliente propio de analítica o tracking. La telemetría de ZITADEL se deshabilita con `ZITADEL_TELEMETRY_ENABLED=false`. No se introduce tracking de marketing.

## Removal

La eliminación segura conserva configuración, estado y auditoría, retira el runtime activo y rota credenciales locales. El purge completo elimina artefactos propios de MCP y conserva aplicaciones, sitios, bases, contenedores de terceros, servicios y archivos administrados.

~~~bash
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~
