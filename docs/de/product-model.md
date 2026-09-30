# Produktmodell: vollständige Toolbox, begrenzte Autorität

## Kernentscheidung

Eine vollständige Capability-Menge ausliefern und Autorität durch serverseitige Policy steuern.

Keine separaten limited/project/full-Binaries:

~~~text
gleiche Binaries
+ gleicher MCP-Toolkatalog
+ andere Policy
= andere effektive Autorität
~~~

Das LLM entscheidet nie den Scope.

## Operator-eigener Scope

Der Operator bestimmt genau, wie viel VPS delegiert wird. Beispiele sind nur Bequemlichkeit; autoritativ ist die explizit bestätigte Policy.

~~~text
ein Verzeichnis:
/opt/my-app

mehrere:
/opt/app-a
/var/www/site
/srv/data

gesamtes Filesystem:
/
~~~

Dasselbe gilt getrennt für systemd, Docker, Netzwerk, Pakete, Benutzer/Gruppen und Admin-Ressourcen.

## Profile

### Standard

Empfohlen für Multi-Projekt-Hosts. Physische Obergrenze standardmäßig `/opt`, keine Projektwurzel bei Installation autorisiert.

Portico darf nur unmittelbare Ordnernamen entdecken. Inhalte bleiben gesperrt bis `read`, `work` oder `compose` genehmigt wird. Secrets bleiben zusätzliche innere Grenze.

### Project

Autonom nur in einer Root wie `/opt/my-app`: Filesystem CRUD, Scoped Shell im Root, ausgewählte Docker-/systemd-Operationen und Netzwerkziele.

### Advanced/static policy

Fortgeschrittene Operatoren können mehrere Roots/Ressourcengruppen statisch definieren. Guided Standard bevorzugt Runtime Delegation statt initialer YAML-Bearbeitung.

### Whole host

Explizite Host-weite Ressourcenfreigabe, keine andere Build-Variante. Kann `/`, breites systemd, Docker, Pakete, Benutzer/Gruppen, Firewall/Netzwerk und temporäre Admin-Shell umfassen. Gefährliche Capabilities bleiben explizit/auditierbar.

## Capability-Katalog

### Filesystem

`file.list/stat/read/mkdir/write/patch/copy/move/remove/remove_recursive/hash/chmod/chown`.

Capabilities existieren auch wenn Policy sie deaktiviert. Recursive delete und breite Permission Changes bleiben getrennt.

### Commands/Jobs

`shell.exec`, `job.start/status/tail/cancel`.

Shell muss serverseitig sandboxed werden. YAML-Path-Scope allein genügt nicht. Policy wird in OS-Kontrollen übersetzt: cwd roots, ProtectSystem, ReadWritePaths/ReadOnlyPaths, ProtectHome, PrivateTmp, cgroups, MemoryMax, TasksMax, Timeout, Output Limits, Network Policy.

### systemd

`service.list/status/logs/start/stop/restart/reload/enable/disable`; Policy begrenzt kanonische Units und Actions.

### Docker/Compose

`docker.list/inspect/logs/start/stop/restart`, `compose.config/pull/up/down`. Typed operations bevorzugen; Docker socket nie im Gateway.

### Diagnostics

`system.info/health/disk/memory`, `process.list/inspect`, `network.listen/check`, `journal.read`.

### Administration

Im Produkt verfügbar, standardmäßig aus: Package-Management, Users/Groups, Firewall, weite chmod/chown und `shell.exec_admin`.

## Scope ist multidimensional

Filesystem ist nur eine Achse. Policy steuert Roots, systemd Units, Docker Stacks, Shell cwd, Network Destinations, Package Actions, Users/Groups, Firewall und temporäre Admin-Capabilities separat.

Project kann volles CRUD in `/opt/my-app` und null Autorität über nginx, Docker, apt oder Internet haben.

## chmod/destruktive Aktionen

`file.chmod` kann inklusive 0777 verfügbar sein, wenn Policy es erlaubt. Defaults aktivieren kein world-writable. `file.remove_recursive` ist separat destruktiv; Host-weites chmod/chown ist eigene Adminentscheidung.

## Konfigurations-UX

`config/policy.yaml` ist autoritativ und Human/AI-lesbar. Es definiert Ressourcen, Profil, Filesystem, Shell/Sandbox, systemd, Docker/Compose, Netzwerk, Admin, Approval/Elevation und Auth. Compose/runtime validieren. Policy kann später ohne Binary-Reinstall geändert werden.

## Packaging

~~~text
Gateway container
  non-root
  no /host
  no Docker socket
        |
        | Unix socket
        v
Broker container
  privileged host-control boundary
  host mounted at /host
        |
        v
VPS
~~~

Broker ist kein Sandbox-Container, sondern die privilegierte serverseitige Sicherheitsgrenze in Docker. Docker liefert Packaging/Lifecycle; Gateway bleibt unprivilegiert; Broker erhält Host-root nur für autorisierte Aktionen, hat keine Remote-Control-TCP-API, und Policy entscheidet Ressourcen. `/host` gibt dem LLM nicht automatisch Autorität.

Primär:

~~~bash
bash scripts/install.sh
~~~

Transparent über Bootstrap, Compose, Verification, OAuth und ChatGPT-Verbindung. Standard/Project/Whole Host werden angezeigt, effektive Autorität vor Start, Wiederaufnahme nach Unterbrechung. Einzelbefehle bleiben verfügbar. [installation-contract.md](installation-contract.md) definiert die unterstützte Grenze.

## Installation erst nach ChatGPT-Verifikation fertig

Tutorial allein reicht nicht. Nach Runtime/Policy werden Endpoint, Auth und effektiver Scope gezeigt.

Erstlauf:

1. HTTPS prüfen;
2. Auth prüfen;
3. harmlosen serverseitigen Test;
4. Autorität anzeigen;
5. aktuelles ChatGPT-Tutorial;
6. auf Verbindung warten;
7. harmlosen Tool-Aufruf direkt aus ChatGPT verlangen;
8. Subject, Policy, Ausführung und Audit prüfen;
9. erst dann Installation abschließen.

ChatGPT-Oberflächen ändern sich unabhängig; Tutorial wird versioniert/revalidiert.

~~~text
clone / bundle
 -> Autorität wählen
 -> Gateway + Broker starten
 -> Policy/Security prüfen
 -> OAuth + HTTPS
 -> ChatGPT-Tutorial
 -> ChatGPT verbinden
 -> echten E2E-Aufruf verifizieren
 -> Audit verifizieren
 -> Installation complete
~~~
