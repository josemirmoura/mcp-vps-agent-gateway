# Autentikasi

Diperiksa terhadap model MCP/OpenAI OAuth saat ini pada 2026-09-27.

## Jalur produk yang didukung

Jalur publik ChatGPT yang didukung adalah **integrated self-hosted OAuth/OIDC**.

Paket menjalankan ZITADEL dedicated + PostgreSQL di samping Gateway. Tidak membutuhkan identity service pihak ketiga atau tunnel terpisah.

Verifikasi local/lab tetap memakai static bearer dari `scripts/init.sh`; token itu tidak menjadi kredensial publik ChatGPT.

## Topologi publik

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

Hostname sama dapat melayani kedua role. `/mcp`, `/healthz`, dan metadata RFC 9728 ke Gateway; discovery/login/token/user-info/DCR ke ZITADEL.

## Bootstrap

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Script membuat identity secrets, reuse Traefik bila aman, start bundled edge hanya pada host bersih, start ZITADEL/PostgreSQL, enable open DCR yang dibutuhkan MCP, membuat dedicated non-admin operator, menyimpan stable subject di `.env`, lalu recreate Gateway/Broker dalam integrated mode.

Password operator dibaca tanpa echo dan tidak ditulis ke `.env` atau marker state.

Temporary human IAM bootstrap owner dan machine PAT dihapus setelah operator dibuat. Internal login-client PAT tetap di private Docker volume karena dibutuhkan ZITADEL Login.

## Protected-resource metadata

~~~text
/.well-known/oauth-protected-resource
~~~

Biasanya:

~~~text
resource: https://mcp.example.com/mcp
authorization_servers:
  - https://mcp.example.com
scopes_supported:
  - openid
bearer_methods_supported:
  - header
~~~

`scripts/verify-public.sh` memeriksa metadata, resource scopes, OIDC discovery, DCR, PKCE S256, refresh token, private introspection, HTTPS, dan unauthenticated denial.

## Resource binding / introspection

Dedicated ZITADEL project menjadi MCP resource audience:

~~~text
MCP VPS Agent Resource
  └── API application: MCP VPS Agent Introspector
~~~

Project ID menjadi required scope:

~~~text
urn:zitadel:iam:org:project:id:<resource-project-id>:aud
~~~

Gateway memvalidasi opaque bearer melalui RFC 7662 di private Docker identity network.

Token hanya diterima jika `active: true`, ada `sub`, `iss` tepat, `exp` belum lewat, `aud` berisi project resource, dan semua required scope ada.

Introspection client berada di project sama; ZITADEL juga memeriksa audience dan Gateway mengulanginya sebagai boundary kedua.

Endpoint introspection tidak diekspos sebagai management surface. Client secret hanya dalam konfigurasi lokal root-readable dan di-redact dari diagnostic bundle.

## Dynamic Client Registration

MCP client register sebelum login, sehingga unauthenticated DCR enabled dan rate-limited di Traefik. Dynamic client berada di dedicated DCR project. Operator tetap harus authenticate sebelum token usable diterbitkan.

## Subject binding

~~~dotenv
VPS_AGENT_SUBJECT=<operator-user-id>
~~~

Bootstrap owner dihapus dan tidak pernah menjadi identity yang diterima Broker.

## Fail-closed

Public URL membutuhkan integrated mode dan HTTPS; metadata harus cocok; discovery wajib DCR/S256/refresh; dedicated audience scope wajib; token aktif, belum expired, issuer/audience benar; unauthenticated `/mcp` mengembalikan 401 + Bearer challenge; Broker memeriksa subject/policy lagi; Gateway tetap tanpa host root atau Docker socket.

## Edge proxy

Satu Traefik yang terdeteksi dengan aman digunakan ulang. Jika tidak ada dan 80/443 bebas, bundled Traefik dimulai. Unknown web server tidak diganti.

## Static bearer local/lab

~~~dotenv
VPS_AGENT_AUTH_MODE=static
~~~

Hanya CI/local, bukan jalur publik.
