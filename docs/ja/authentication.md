# Authentication

2026-09-27 時点の MCP/OpenAI OAuth model に照らして確認。

## Supported product path

Public ChatGPT path は **integrated self-hosted OAuth/OIDC**。

Package は Gateway と並んで dedicated ZITADEL + PostgreSQL を実行します。Third-party identity service や separate tunnel は不要です。

Local/lab verification は `scripts/init.sh` の static bearer を使いますが、public ChatGPT credential にはなりません。

## Public topology

~~~text
ChatGPT
   |
   | HTTPS + OAuth 2.x / OIDC
   v
public domain
   |-------------------------------|
   |                               |
   v                               v
Gateway                         ZITADEL
protected resource             authorization server
   |                               |
   v                               v
Broker                       PostgreSQL identity state
   |
   v
VPS
~~~

同じ hostname で両方を提供できます。`/mcp`, `/healthz`, RFC 9728 metadata は Gateway、OAuth/OIDC discovery/login/token/user-info/DCR は ZITADEL。

## Bootstrap

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Identity secrets を生成し、安全なら existing Traefik を再利用、clean host のみ bundled edge を起動、ZITADEL/PostgreSQL を開始、MCP に必要な open DCR を有効化、dedicated non-admin operator を作成し stable subject を `.env` に記録、Gateway/Broker を integrated mode で再作成します。

Operator password は echo なしで読み、`.env` や state marker に書きません。

Temporary bootstrap human IAM owner と machine PAT は operator 作成後に削除。Internal login-client PAT は ZITADEL Login に必要なので private Docker volume に残ります。

## Protected-resource metadata

~~~text
/.well-known/oauth-protected-resource
~~~

通常:

~~~text
resource: https://mcp.example.com/mcp
authorization_servers:
  - https://mcp.example.com
scopes_supported:
  - openid
bearer_methods_supported:
  - header
~~~

`scripts/verify-public.sh` は metadata、required scopes、OIDC discovery、DCR、PKCE S256、refresh-token、private introspection、HTTPS、unauthenticated denial を検証。

## Resource binding / introspection

Dedicated ZITADEL project を MCP resource audience とします。

~~~text
MCP VPS Agent Resource
  └── API application: MCP VPS Agent Introspector
~~~

Project ID は required scope:

~~~text
urn:zitadel:iam:org:project:id:<resource-project-id>:aud
~~~

Gateway は private Docker identity network から RFC 7662 introspection を行います。

Token acceptance 条件: `active: true`, `sub` present, exact `iss`, future `exp`, correct project in `aud`, required scopes present。

Introspection client は同じ project に所属し、ZITADEL 自身も audience を検証。Gateway が第二境界として再検証します。

Introspection endpoint は management surface として公開しません。Client secret は local root-readable configuration のみで diagnostic bundle から redact。

## Dynamic Client Registration

MCP client は login 前に register するため unauthenticated DCR を enabled にし、Traefik で rate-limit。Dynamic clients は dedicated DCR project。Operator authentication は token 発行前に必須。

## Subject binding

~~~dotenv
VPS_AGENT_SUBJECT=<operator-user-id>
~~~

Bootstrap owner は削除され Broker accepted identity ではありません。

## Fail-closed

Public URL requires integrated mode/HTTPS; metadata must match expected resource/issuer; discovery must advertise DCR/S256/refresh; dedicated audience scope required; accepted token must be active/unexpired/correct issuer/audience; unauthenticated `/mcp` is 401 + Bearer challenge; Broker rechecks exact subject/policy; Gateway still has no host root/Docker socket.

## Edge proxy

Exactly one detected Traefik is reused. If none and ports 80/443 free, bundled pinned Traefik starts. Unknown web server is never replaced.

## Local/lab static bearer

~~~dotenv
VPS_AGENT_AUTH_MODE=static
~~~

CI/local only, not public ChatGPT path.
