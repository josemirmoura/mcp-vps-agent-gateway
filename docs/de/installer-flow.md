# Erster Portico-MCP-Durchlauf

## Produktziel

Die Installation ist deklarativ und terminalorientiert. Operatorzustand liegt in:

~~~text
.env                   # enthält VPS_AGENT_SCOPE_ROOT
config/policy.yaml     # logische Capability-/Resource-Policy
~~~

Docker Compose startet das Paket. Es gibt keinen zweiten versteckten Installer.

## Unterstützte Grenze

Der Benutzer führt die Befehle direkt auf der VPS aus. Es werden weder Entwicklerzugriff noch von ChatGPT gesteuerte Remote-Shell noch VPS-Passwort/SSH-Schlüssel benötigt.

Das OAuth-Operator-Passwort ist eine notwendige Dienst-Credential und wird lokal, ohne Echo, in `setup-integrated-auth.sh` eingegeben.

## Geführter Einstieg

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

`install.sh` orchestriert die transparenten Komponenten, zeigt die effektive Autorität vor dem Start und kann nach Unterbrechung erneut ausgeführt werden.

## Phase 1 — Bootstrap

Empfohlen ist **Standard**: physische Obergrenze `/opt`, keine statischen Projektwurzeln. Die Obergrenze ist nur das maximale Dateisystem-Limit, keine Lese-/Schreibberechtigung. Ein anderer absoluter Pfad kann gewählt werden.

~~~bash
bash scripts/init.sh --scope /opt --dynamic-baseline
~~~

Vor Änderungen prüft `scripts/preflight.py` Host, Docker/Compose, Tools, systemd, RAM und die Edge-Situation. Fehlende Pflichtvoraussetzungen stoppen vor der Erzeugung von Zustand.

Bootstrap erzeugt lokale Zufallsgeheimnisse, stabile Instanz-ID, `config/policy.yaml`, die gewählte Obergrenze, einen echten Nicht-root-Benutzer für die eingeschränkte Shell, State-Verzeichnis und Compose-Validierung.

## Physische Dateisystem-Obergrenze

`compose.yaml` bind-mountet nur `VPS_AGENT_SCOPE_ROOT` als `/host`. Broker erzwingt, dass Dateisystem-, Shell-cwd- und Compose-Pfade darin bleiben. Escapes schlagen fail-closed fehl.

Standard mit `--dynamic-baseline` autorisiert `/opt` nicht automatisch. Whole Host benötigt `VPS_AGENT_WHOLE_HOST=1` plus `compose.host.yaml` und setzt die physische Obergrenze auf `/`; Full wird dadurch nicht aktiviert.

## Phase 2 — MCP-Autorität

Standard startet ohne Projektberechtigung. `permissions.discover_scope` darf nur unmittelbare Verzeichnisnamen unterhalb der Obergrenze anzeigen, keine Dateiinhalte.

Danach:

~~~text
permissions.request_root_access
 -> ausstehende Broker-Anfrage
 -> native MCP Elicitation / Client-Bestätigung
 -> subject-gebundene Delegation
~~~

- `read`: Dateisystem lesen;
- `work`: lesen/schreiben + eingeschränktes Shell-cwd;
- `compose`: zusätzlich von der statischen Policy erlaubte Compose-Aktionen.

### Geschützte Dateien

`.env` und `.env.*` bleiben auch in autorisierten Projekten gesperrt. `.env.example`, `.env.sample` und `.env.template` bleiben normale Vorlagen.

`permissions.request_sensitive_access` gewährt eine separate, temporäre, exakte, subject-gebundene, auditierbare und widerrufbare Ausnahme. Scoped-Shell-Jobs maskieren geschützte Pfade und erkannte Hardlink-Aliase, solange keine passende temporäre Freigabe aktiv ist.

Eine Delegation kann nach verstärkter Warnung auch die physische Obergrenze selbst betreffen. Sie kann diese Grenze niemals überschreiten und entsperrt geschützte Geheimnisse nicht.

systemd, Docker/Compose, Netzwerk, Pakete, Benutzer/Gruppen, Firewall und Elevation bleiben separat begrenzt.

## Phase 3 — Start

~~~bash
docker compose up -d --build
~~~

~~~text
Gateway: non-root, kein Host-root
Broker: privilegiert, Host unter /host, kein Remote-Control-Port
~~~

Docker ist Verpackung, nicht die Autorisierungsgrenze. Das ist die Broker-Policy.

## Phase 4 — Lokale Prüfung

~~~bash
bash scripts/verify.sh
~~~

Prüft Compose, Broker/Gateway-Health, Audit-Integrität, Authentifizierung und einen harmlosen `system.info`-Aufruf. Erfolg bedeutet Runtime bereit, **nicht** Installation abgeschlossen.

## Phase 5 — Integriertes OAuth + öffentlicher Endpunkt

Der Operator konfiguriert einen DNS-Hostname. Portico erklärt A/AAAA und prüft Auflösung.

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Genau ein sicher erkennbares Traefik wird wiederverwendet. Andernfalls startet das mitgelieferte Traefik, wenn 80/443 frei sind. ZITADEL + PostgreSQL starten, ein dedizierter Nicht-admin-Operator wird erstellt, Username/E-Mail werden angezeigt, Resource-Audience und privater Introspection-Client erstellt, DCR aktiviert, Gateway/Broker an das Subject gebunden und öffentliche Prüfung ausgeführt.

Erfolg:

~~~text
INTEGRATED AUTH: READY
~~~

Erneut prüfbar:

~~~bash
bash scripts/verify-public.sh
~~~

Geprüft werden HTTPS, OAuth/OIDC Discovery, Protected-Resource-Metadata, DCR/PKCE, private Introspection und fail-closed unauthenticated MCP.

## Phase 6 — ChatGPT-Fähigkeit

Prüfe, ob das tatsächliche ChatGPT-Konto/Workspace Developer Mode und benutzerdefinierte MCP-Apps anbietet. OpenAI steuert Rollout und Verfügbarkeit.

## Phase 7 — ChatGPT-Web-Tutorial

~~~bash
bash scripts/connect-chatgpt.sh
~~~

App-Erstellung und OAuth-Login erfolgen in ChatGPT Web mit der dedizierten OAuth-Identität, niemals mit VPS-/SSH-Credentials.

## Phase 8 — Echter Verbindungstest

Das Skript speichert eine Audit-Basislinie. Der Benutzer richtet ChatGPT in eigenem Tempo ein und drückt danach Enter. ChatGPT soll `system.info` aufrufen.

Erfolg erfordert:

~~~text
echter ChatGPT-MCP-Aufruf
+ erwartetes authentifiziertes Subject
+ Policy allow
+ erfolgreiche Ausführung
+ Broker-Audit
+ gültige Audit-Kette
~~~

Erst dann:

~~~text
CHATGPT WEB CONNECTION VERIFIED
INSTALLATION COMPLETE
~~~

## Update und Entfernung

~~~bash
bash scripts/update.sh
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Quellcode zusätzlich löschen:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

Delegierte Projekte und Fremdressourcen werden nicht nur deshalb gelöscht, weil Portico sie verwalten durfte.
