# Autenticación

Comprobado contra el modelo actual MCP/OpenAI OAuth el 2026-09-27.

## Camino soportado

El camino público soportado para ChatGPT es **OAuth/OIDC integrado y autoalojado**.

El paquete ejecuta ZITADEL dedicado + PostgreSQL junto al Gateway. No requiere servicio de identidad tercero ni túnel separado.

La verificación local/lab sigue usando el bearer estático generado por `scripts/init.sh`; nunca se convierte en credencial pública de ChatGPT.

## Topología

~~~text
ChatGPT
   |
   | HTTPS + OAuth 2.x / OIDC
   v
dominio público
   |-------------------------------|
   |                               |
   v                               v
Gateway                         ZITADEL
recurso protegido              authorization server
   |                               |
   v                               v
Broker                       PostgreSQL identity state
   |
   v
VPS
~~~

El mismo hostname puede servir ambos roles. `/mcp`, `/healthz` y metadata RFC 9728 van al Gateway; discovery OAuth/OIDC, login, token, user-info y DCR a ZITADEL.

## Bootstrap

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Genera secretos del identity stack, reutiliza Traefik cuando es seguro, inicia borde incluido solo en host limpio, arranca ZITADEL/PostgreSQL, habilita Dynamic Client Registration abierto para MCP, crea operador dedicado no-admin, guarda su subject estable en `.env` y recrea Gateway/Broker en modo integrado.

La contraseña se lee sin eco y nunca se escribe en `.env` ni en el marcador de estado.

El owner IAM humano temporal y el PAT de bootstrap se eliminan tras crear el operador. El PAT interno del login-client permanece en volumen privado porque ZITADEL Login lo necesita.

## Metadata del recurso protegido

~~~text
/.well-known/oauth-protected-resource
~~~

Normalmente anuncia:

~~~text
resource: https://mcp.example.com/mcp
authorization_servers:
  - https://mcp.example.com
scopes_supported:
  - openid
bearer_methods_supported:
  - header
~~~

`scripts/verify-public.sh` valida metadata, scopes, discovery, DCR, PKCE S256, refresh token, introspección privada, HTTPS y rechazo sin auth.

## Audience e introspección

Un proyecto ZITADEL dedicado representa el recurso MCP:

~~~text
MCP VPS Agent Resource
  └── API application: MCP VPS Agent Introspector
~~~

Su project ID es scope obligatorio:

~~~text
urn:zitadel:iam:org:project:id:<resource-project-id>:aud
~~~

ZITADEL lo añade al audience. Gateway valida cada token opaco por RFC 7662 en la red Docker privada.

Token aceptado solo si:

- `active: true`;
- `sub` presente;
- `iss` coincide exactamente;
- `exp` futuro;
- `aud` contiene el proyecto;
- scopes incluyen todos los requeridos.

El cliente de introspección pertenece al mismo proyecto y ZITADEL aplica además su propio audience check. Gateway repite la validación.

El endpoint de introspección no se expone como administración; Gateway lo alcanza por `integrated-auth`. El secret queda local y se redacta de bundles.

## Dynamic Client Registration

MCP registra clientes antes del login, por lo que DCR no autenticado está habilitado y rate-limited en Traefik. Los clientes dinámicos viven en un proyecto DCR dedicado. El operador sigue teniendo que autenticarse para obtener token útil.

## Subject binding

El setup crea usuario humano regular y guarda:

~~~dotenv
VPS_AGENT_SUBJECT=<operator-user-id>
~~~

El owner de bootstrap nunca es la identidad aceptada por Broker.

## Reglas fail-closed

- Gateway local usa static auth para aceptación determinista;
- URL pública exige modo integrado;
- HTTPS obligatorio;
- metadata debe identificar recurso/issuer esperados;
- discovery debe anunciar DCR, S256 y refresh token;
- metadata debe anunciar audience scope dedicado;
- token debe estar activo, vigente, issuer/audience correctos;
- `/mcp` sin auth devuelve 401 + `WWW-Authenticate: Bearer`;
- Broker revalida subject y policy;
- Gateway sigue sin host root ni Docker socket.

## Edge proxy

Un único Traefik detectable se reutiliza y su ACME resolver también. Si no hay Traefik y 80/443 están libres, se inicia el incluido. Nunca se reemplaza un web server desconocido.

## Bearer estático local/lab

~~~dotenv
VPS_AGENT_AUTH_MODE=static
~~~

Solo para CI/local; no es el camino público.
