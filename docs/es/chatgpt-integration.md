# Integración con ChatGPT

Comprobado: 2026-09-30.

## Objetivo

~~~text
ChatGPT Web
 -> discovery OAuth + Dynamic Client Registration
 -> HTTPS /mcp
 -> Gateway
 -> Broker
 -> VPS
 -> auditoría con detección de alteraciones
~~~

La instalación exige más que “contenedores healthy”.

## Requisito de producto ChatGPT

Portico incluye herramientas de modificación. Antes del setup público, verifica en la cuenta/workspace real que ChatGPT exponga modo desarrollador y creación de app MCP personalizada, y comprende los permisos que ese workspace concede.

La disponibilidad de funciones y nombres de UI son controlados por OpenAI y deben revisarse contra documentación oficial en cada release.

Referencias usadas por el proyecto:

- https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt
- https://developers.openai.com/plugins/build/app-quickstart

Una conexión MCP solo lectura puede servir para diagnóstico, pero no equivale a completar el camino de producto con escritura.

## Prerrequisitos

1. ChatGPT expone creación de MCP personalizada.
2. `bash scripts/verify.sh` pasó.
3. DNS apunta a la VPS.
4. Auth integrada pasó:

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Debe terminar en:

~~~text
INTEGRATED AUTH: READY
~~~

Para repetir solo la frontera pública:

~~~bash
bash scripts/verify-public.sh
~~~

## Conexión

~~~bash
bash scripts/connect-chatgpt.sh
~~~

El script vuelve a comprobar la frontera pública y muestra endpoint MCP, issuer y pasos actuales.

Flujo esperado:

1. ChatGPT lee metadata RFC 9728 del recurso protegido.
2. descubre ZITADEL;
3. registra dinámicamente un cliente OAuth público;
4. el operador inicia sesión con la cuenta OAuth dedicada;
5. completa Authorization Code + PKCE;
6. ChatGPT descubre las tools MCP;
7. se invoca `system.info`;
8. Gateway valida el access token;
9. Broker comprueba subject estable y policy;
10. auditoría registra la llamada.

Nunca se introducen en ChatGPT contraseña VPS, clave SSH privada, root o shell irrestricto. La contraseña OAuth dedicada se introduce únicamente en la pantalla del proveedor de identidad. El login puede pedir username, por eso Portico muestra username y correo por separado.

## Puerta de finalización

`scripts/connect-chatgpt.sh` fija una línea base de auditoría. El operador configura ChatGPT sin countdown oculto. Al pulsar Enter, Portico busca un `system.info` autenticado posterior a esa línea base.

~~~text
tutorial mostrado
 != éxito

app ChatGPT conectada
 + system.info autenticado
 + subject esperado
 + Broker permite
 + ejecución correcta
 + auditoría correspondiente
 = INSTALACIÓN COMPLETA
~~~

Si la llamada aún no llegó, el script sigue incompleto, explica qué revisar y permite reintentar.

## Transporte

MCP usa Streamable HTTP sobre HTTPS:

~~~text
https://<domain>/mcp
~~~

No se introduce WebSocket propio.

## Frontera de seguridad

Las confirmaciones de ChatGPT son controles UX adicionales. No sustituyen la autorización server-side.

**Gateway autentica. Broker autoriza. El propietario de la VPS elige la policy.**
