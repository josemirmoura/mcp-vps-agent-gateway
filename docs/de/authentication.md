# Authentifizierung

Geprüft gegen das aktuelle MCP/OpenAI-OAuth-Modell am 2026-09-27.

## Unterstützter Produktpfad

Der öffentliche ChatGPT-Pfad ist **integriertes selbst gehostetes OAuth/OIDC**.

Das Paket betreibt eine dedizierte ZITADEL-Instanz plus PostgreSQL neben dem Gateway. Kein Drittanbieter-Identity-Service oder separater Tunnel ist erforderlich.

Lokale/Lab-Tests nutzen weiterhin den statischen Bearer aus `scripts/init.sh`; er wird niemals öffentliche ChatGPT-Credential.

## Öffentliche Topologie

~~~text
ChatGPT
   |
   | HTTPS + OAuth 2.x / OIDC
   v
öffentliche Domain
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

Derselbe Hostname kann beide Rollen bedienen. `/mcp`, `/healthz` und RFC-9728-Metadaten gehen zum Gateway; Discovery, Login, Token, User-Info und DCR zu ZITADEL.

## Bootstrap

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Das Skript erzeugt Identity-Secrets, verwendet vorhandenes Traefik wenn sicher, startet gebündelten Edge nur auf sauberem Host, startet ZITADEL/PostgreSQL, aktiviert offenes DCR für MCP, erstellt dedizierten Nicht-admin-Operator, speichert sein stabiles Subject in `.env` und startet Gateway/Broker im integrierten Modus neu.

Operator-Passwort wird ohne Echo gelesen und nie in `.env` oder Marker geschrieben.

Temporärer IAM-Bootstrap-Owner und Maschinen-PAT werden nach Operator-Erstellung entfernt. Der interne Login-Client-PAT bleibt im privaten Docker-Volume, weil ZITADEL Login ihn benötigt.

## Protected-Resource-Metadaten

~~~text
/.well-known/oauth-protected-resource
~~~

Normal:

~~~text
resource: https://mcp.example.com/mcp
authorization_servers:
  - https://mcp.example.com
scopes_supported:
  - openid
bearer_methods_supported:
  - header
~~~

`scripts/verify-public.sh` prüft Metadaten, Resource Scopes, OIDC Discovery, DCR, PKCE S256, Refresh Token, private Introspection, HTTPS und unauthenticated denial.

## Resource Binding und Introspection

Ein dediziertes ZITADEL-Projekt ist die MCP Resource Audience:

~~~text
MCP VPS Agent Resource
  └── API application: MCP VPS Agent Introspector
~~~

Project ID als Pflicht-Scope:

~~~text
urn:zitadel:iam:org:project:id:<resource-project-id>:aud
~~~

Gateway validiert opaque bearer tokens per RFC 7662 im privaten Identity-Docker-Netz.

Akzeptiert nur wenn `active: true`, `sub` vorhanden, `iss` exakt, `exp` gültig, `aud` enthält Resource-Projekt und alle Pflicht-Scopes vorhanden sind.

Der Introspection-Client gehört zum selben Projekt. ZITADEL prüft ebenfalls Audience; Gateway wiederholt den Check als zweite Grenze.

Introspection ist keine öffentliche Managementfläche. Secret bleibt lokal root-readable und wird aus Diagnose-Bundles redigiert.

## Dynamic Client Registration

MCP-Clients registrieren vor Login, daher ist unauthenticated DCR aktiviert und am Traefik rate-limited. Dynamische Clients liegen in einem eigenen DCR-Projekt. Nutzbare Tokens gibt es erst nach Operator-Login.

## Subject Binding

~~~dotenv
VPS_AGENT_SUBJECT=<operator-user-id>
~~~

Bootstrap-Owner wird entfernt und ist nie die Broker-Identität.

## Fail-closed

- lokales Gateway mit static auth nur für deterministische Acceptance;
- öffentliche MCP-URL nur in integrated mode;
- HTTPS Pflicht;
- Resource/Issuer-Metadaten müssen stimmen;
- Discovery muss DCR, S256, Refresh Token anzeigen;
- Audience-Scope Pflicht;
- Token aktiv, nicht abgelaufen, richtiger Issuer/Audience;
- unauthenticated `/mcp` liefert 401 + Bearer challenge;
- Broker prüft exact Subject + Policy erneut;
- Gateway ohne Host-root und Docker socket.

## Edge Proxy

Genau ein erkennbares Traefik wird wiederverwendet. Ohne Traefik und mit freien 80/443 startet das gebündelte Traefik. Ein unbekannter Webserver wird nie ersetzt.

## Static bearer local/lab

~~~dotenv
VPS_AGENT_AUTH_MODE=static
~~~

Nur CI/local, nicht öffentlicher ChatGPT-Pfad.
