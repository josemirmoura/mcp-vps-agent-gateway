# Portico MCP Schnellstart

Portico MCP wird im Terminal mit transparentem Docker Compose und Repository-Skripten installiert.

## Anforderungen

- Linux-VPS; Ubuntu 24.04 LTS ist RC-validiertes Ziel;
- Docker Engine 24+ und Docker Compose v2;
- mindestens 2 GB RAM für gebündeltes ZITADEL;
- Git, OpenSSL, Python 3 und curl;
- öffentliche DNS, TCP 80/443 und gültiges HTTPS;
- ChatGPT-Konto/Workspace mit tatsächlicher Developer-Mode-/MCP-App-Funktion.

## Start

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

Offizielle Sprachen:

~~~text
en · pt-BR · es · de · fr · ja · id
~~~

Explizit:

~~~bash
bash scripts/install.sh --lang de
~~~

## Geführter Ablauf

~~~text
Voraussetzungen
 -> Scope
 -> effektive Autorität
 -> Container
 -> lokale Prüfung
 -> sicherer öffentlicher Zugriff
 -> ChatGPT verbinden
 -> echter auditierter MCP-Aufruf
 -> INSTALLATION COMPLETE
~~~

Preflight prüft Host, Docker/Compose, Tools, systemd, RAM und Edge. Ein vorhandenes Traefik wird sicher wiederverwendet, sonst kann das gebündelte Traefik 80/443 übernehmen, wenn frei.

## Empfohlen: Standard

~~~text
physische Obergrenze: /opt
statische Wurzeln:    keine
Projektberechtigung:  später explizit genehmigt
~~~

Die Obergrenze definiert nur das maximal erreichbare Gebiet und autorisiert `/opt` nicht.

Anderer Pfad:

~~~bash
bash scripts/install.sh --profile custom --scope /srv/apps
~~~

`permissions.discover_scope` zeigt nur unmittelbare Ordnernamen. Zugriff wird anschließend über native MCP-Bestätigung beantragt:

~~~text
permissions.request_root_access
 -> native MCP-Bestätigung
 -> Broker aktiviert read / work / compose
~~~

Die KI kann ihre eigene Autorität nicht genehmigen.

### Secrets

`.env` und `.env.*` bleiben in autorisierten Projekten gesperrt. Vorlagen wie `.env.example` bleiben lesbar. Ein echtes Geheimnis benötigt eine separate temporäre `permissions.request_sensitive_access`-Freigabe.

## Project

~~~bash
bash scripts/install.sh --profile project --scope /opt/my-app
~~~

## Whole Host

~~~bash
bash scripts/install.sh --profile whole-host
~~~

Setzt Obergrenze auf `/`, aktiviert aber nicht automatisch Full, Netzwerk, Pakete, Benutzer oder Firewall.

## Shell-Benutzer

Scoped Jobs laufen als echter Nicht-root-Benutzer:

~~~bash
bash scripts/install.sh --run-as deploy
~~~

## Öffentliches OAuth

Portico führt durch DNS und erstellt eine dedizierte OAuth-Identität:

~~~text
Username: vps-operator
Email:    operator@example.com
~~~

Das sind keine Linux-/SSH-/root-Credentials.

## ChatGPT-Abschluss

`scripts/connect-chatgpt.sh` zeigt das vollständige Tutorial. App/OAuth in eigenem Tempo einrichten, `system.info` senden, zum Terminal zurückkehren und Enter drücken.

## Nur lokal testen

~~~bash
bash scripts/install.sh --profile custom --scope /opt --local-only --yes
~~~

Das ist **keine** abgeschlossene öffentliche Installation.

## Entfernung

~~~bash
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Checkout ebenfalls löschen:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

## Stabile Releases

Nach `v0.1.0` Produktionsinstallationen an stabile Tags binden:

~~~bash
git clone --branch v0.1.0 https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

Der Git-Checkout ermöglicht verifizierte Fast-Forward-Updates, Backups, Migrationen und automatischen Rollback.
