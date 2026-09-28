# Authentication

Checked against current OpenAI guidance: 2026-09-27.

## Two deliberately different modes

### Local/lab acceptance: static bearer

The Docker package keeps static bearer authentication for deterministic local tests and CI. It is useful for proving the Gateway -> Broker -> VPS path without an external identity provider.

Static bearer is not the public ChatGPT authentication path.

### Public ChatGPT: OAuth/OIDC resource server

For a private or write-capable MCP server, the Gateway acts as an OAuth resource server and validates access tokens issued by an external Authorization Server.

The package uses the official MCP Go SDK primitives and exposes:

~~~text
/.well-known/oauth-protected-resource
~~~

The metadata identifies the MCP resource, its Authorization Server, supported scopes and bearer-token transport. The SDK bearer middleware also returns a WWW-Authenticate challenge pointing back to that protected-resource metadata when an unauthenticated request is rejected.

Required public settings normally include:

~~~dotenv
VPS_AGENT_PUBLIC_URL=https://mcp.example.com/mcp
VPS_AGENT_AUTH_MODE=oidc
VPS_AGENT_OIDC_ISSUER=https://auth.example.com
VPS_AGENT_OIDC_AUDIENCE=https://mcp.example.com/mcp
VPS_AGENT_SUBJECT=<expected-token-subject>
~~~

Optional overrides:

~~~dotenv
VPS_AGENT_OAUTH_RESOURCE=https://mcp.example.com/mcp
VPS_AGENT_RESOURCE_METADATA_URL=https://mcp.example.com/.well-known/oauth-protected-resource
VPS_AGENT_REQUIRED_SCOPES=vps.read vps.write
~~~

If VPS_AGENT_OAUTH_RESOURCE is empty, the Gateway uses VPS_AGENT_PUBLIC_URL. If VPS_AGENT_OIDC_AUDIENCE is empty in oidc mode, it defaults to the resource identifier.

## Authorization Server requirements

Use an established provider where possible. The provider needs to satisfy the current MCP/OpenAI OAuth contract.

For this package the issuer must expose OpenID Connect discovery at:

~~~text
<issuer>/.well-known/openid-configuration
~~~

The runtime uses that discovery document to configure OIDC token verification. Before ChatGPT connection, scripts/verify-public.sh also checks that the Authorization Server metadata:

- advertises the exact configured issuer;
- publishes HTTPS authorization and token endpoints;
- advertises PKCE S256;
- publishes token_endpoint_auth_methods_supported;
- exposes a ChatGPT-compatible client identification path using CIMD or DCR, unless a predefined OAuth client was deliberately configured;
- advertises every configured required scope when scopes_supported is present.

CIMD is accepted when client_id_metadata_document_supported is true and the token endpoint supports a ChatGPT-compatible method such as none or private_key_jwt. DCR is accepted when an HTTPS registration_endpoint is published.

If the deployment intentionally uses a predefined ChatGPT OAuth client and the provider advertises neither CIMD nor DCR, set:

~~~dotenv
VPS_AGENT_PREDEFINED_OAUTH_CLIENT=1
~~~

That flag only tells the verifier that the external client registration step was completed deliberately. It does not create, register or store an OAuth client.

OpenAI documents authorization-code + PKCE, resource binding, discovery metadata and CIMD/DCR/predefined clients for authenticated MCP servers:

https://developers.openai.com/plugins/build/auth

## Refresh tokens

Long-lived ChatGPT connections should use refresh-token support. For OIDC providers, OpenAI currently recommends advertising and enabling offline_access so ChatGPT can request a refresh token.

scripts/verify-public.sh reports a warning when the discovery document does not advertise offline_access. This is a durability warning rather than a startup failure because provider-specific refresh mechanisms exist.

## Subject binding

The Broker independently compares every non-admin request subject with VPS_AGENT_SUBJECT. This prevents a valid token for an unexpected identity from silently using the delegated VPS authority.

Configure VPS_AGENT_SUBJECT to the exact stable sub claim issued by your Authorization Server for the intended operator identity.

## Fail-closed rules

- The standalone Gateway defaults to 127.0.0.1:8080.
- Auth mode none is accepted only on a loopback listener; an unauthenticated non-loopback bind makes the Gateway refuse startup.
- A non-empty VPS_AGENT_PUBLIC_URL with auth mode static or none makes the Gateway refuse startup.
- verify-public.sh rejects non-HTTPS public URLs.
- verify-public.sh requires valid OAuth Protected Resource Metadata.
- verify-public.sh requires Authorization Server/OIDC discovery with PKCE S256 and a usable client-identification path.
- verify-public.sh requires unauthenticated MCP access to return HTTP 401 with a WWW-Authenticate resource_metadata challenge.
- The Broker still rechecks subject and server-side policy after Gateway authentication.

## What the package does not implement

The project does not implement a home-grown Authorization Server. Authentication infrastructure is security-critical and provider-specific; the Gateway is the protected resource and token verifier.

That keeps the package small and relies on standard OAuth/OIDC components instead of inventing an identity stack.
