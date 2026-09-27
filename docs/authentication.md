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

The metadata identifies the MCP resource, its Authorization Server, supported scopes and bearer-token transport.

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

Use an established provider where possible. The provider needs to satisfy the current MCP/OpenAI OAuth contract, including discovery metadata, authorization-code flow with PKCE, resource binding/audience semantics, and a ChatGPT-compatible client registration mechanism such as CIMD or DCR when required.

Refresh-token support matters for long-lived ChatGPT connections. Follow the provider and current OpenAI instructions for offline_access or equivalent refresh-token configuration.

OpenAI reference:
https://developers.openai.com/plugins/build/auth

## Subject binding

The Broker independently compares every non-admin request subject with VPS_AGENT_SUBJECT. This prevents a valid token for an unexpected identity from silently using the delegated VPS authority.

Configure VPS_AGENT_SUBJECT to the exact stable sub claim issued by your Authorization Server for the intended operator identity.

## Fail-closed rules

- A non-empty VPS_AGENT_PUBLIC_URL with auth mode static or none makes the Gateway refuse startup.
- verify-public.sh rejects non-HTTPS public URLs.
- verify-public.sh requires OAuth Protected Resource Metadata.
- verify-public.sh requires unauthenticated MCP access to return HTTP 401.
- The Broker still rechecks subject and server-side policy after Gateway authentication.

## What the package does not implement

The project does not implement a home-grown Authorization Server. Authentication infrastructure is security-critical and provider-specific; the Gateway is the protected resource and token verifier.

That keeps the package small and relies on standard OAuth/OIDC components instead of inventing an identity stack.
