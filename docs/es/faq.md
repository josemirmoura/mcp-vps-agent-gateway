# Preguntas frecuentes

## ¿Whole Host significa Full?

No. Whole Host coloca el techo físico del Broker en `/`. Full es un conjunto explícito de capacidades gobernado por policy/feature gates, deshabilitado por defecto y todavía no presentado como producción.

## ¿Full implica Internet irrestricto?

No. La red irrestricta es una decisión de capacidad/aprobación separada.

## ¿ChatGPT recibe mi contraseña VPS o clave SSH?

No. La instalación soportada la ejecuta el operador en la VPS. ChatGPT recibe el endpoint MCP público y completa OAuth. Credenciales VPS/SSH no forman parte del flujo.

## ¿Por qué el Broker es privilegiado?

systemd del host, Docker y filesystem delegado requieren un componente privilegiado confiable. El Broker es local-only y autoritativo en policy; el Gateway remoto sigue no-root y no recibe Docker socket ni host root.

## ¿Docker es la frontera de seguridad?

No. Docker empaqueta y gestiona lifecycle. La autorización server-side del Broker es la frontera efectiva.

## ¿Hay telemetría?

El proyecto no incluye cliente propio de analytics/tracking y desactiva explícitamente la telemetría de ZITADEL incluida. Consulta `privacy.md`.

## ¿Puedo limitarlo a un proyecto?

Sí. Usa **Project**.

## ¿Cuál es el perfil recomendado?

**Standard**: `/opt` como techo físico, ninguna raíz autorizada al inicio y aprobaciones posteriores desde ChatGPT.

## ¿Puedo delegar varios directorios?

Sí, bajo el techo físico. También existen raíces estáticas para operadores avanzados.

## ¿Varias VPS?

Sí. Cada instalación tiene identidad, credenciales, estado y cadena de auditoría independientes.

## ¿Otro cliente MCP puede usar Portico?

El core usa MCP Streamable HTTP estándar. El camino público soportado está validado con ChatGPT Web; otros clientes compatibles pueden funcionar si soportan la autenticación requerida.

## ¿Cuándo termina la instalación?

Solo cuando una llamada real atraviesa autenticación, Gateway, Broker, policy, ejecución y auditoría. Health de contenedores no basta.

## ¿Por qué update.sh evita main?

`main` es desarrollo. Producción sigue tags SemVer estables.

## ¿Puedo borrar Portico sin borrar mis apps?

Sí. Safe remove y purge eliminan artefactos propios de Portico y preservan los recursos que administraba.
