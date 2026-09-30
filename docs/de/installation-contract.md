# Installationsvertrag

Status: **normative Projektentscheidung**.

Dieses Dokument trennt temporäre Entwicklungs-/Validierungsverfahren von der unterstützten Installation für Endanwender. Implementierung, Tests, README, Tutorials und Release-Prüfung müssen dieser Grenze folgen.

## Unterstützte Benutzererfahrung

Die offizielle Installation ist:

- terminalorientiert;
- Docker-Compose-first;
- transparent und prüfbar;
- auf der eigenen VPS reproduzierbar;
- ohne Hilfe der Projektentwickler nutzbar;
- **native first**: offizielle Plattformmechanismen, Docker, MCP und standardbasiertes OAuth/OIDC vor eigenem Glue-Code.

Der Benutzer installiert und betreibt Portico direkt auf der VPS. ChatGPT wird erst verbunden, nachdem Server und öffentliche Authentifizierung bereit sind.

## Geheimnisse und Fernzugriff

Das Tutorial darf den Benutzer **niemals** auffordern, ChatGPT oder Maintainers Folgendes zu geben:

- VPS-Passwort;
- privaten SSH-Schlüssel;
- uneingeschränkten SSH-/Admin-Fernzugriff;
- root-Zugangsdaten;
- Cloud-Admin-Zugangsdaten;
- Terminalausgabe mit Geheimnissen;
- Geheimnisse, die der Dienst nicht zwingend benötigt.

Notwendige Geheimnisse werden lokal oder über die native UI/API des betreffenden Dienstes eingegeben. Das dedizierte OAuth-Operator-Passwort wird lokal eingegeben und nicht an ChatGPT übergeben.

Temporäre Entwicklungsdiagnosen durch einen menschlichen Operator sind eine Eigenschaft der Entwicklungsumgebung und **keine Produktanforderung**.

## Automatisierungsverantwortung

Deterministische Installationsaufgaben sollen automatisiert werden, darunter:

- Abhängigkeitsprüfung mit handlungsfähigen Fehlern;
- Port-/Edge-Proxy-Erkennung;
- Compose-Validierung;
- Start und Health-Checks;
- HTTPS-Einrichtung und -Prüfung;
- OAuth/OIDC-Bootstrap und Discovery;
- MCP-Endpunktprüfungen;
- Fail-Closed-Tests;
- Diagnosepakete mit Secret-Redaktion;
- sichere Recovery-Anweisungen.

Einmalige Debug-Befehle aus der Entwicklung gehören nicht in den Benutzerweg.

## Bewusste manuelle Schritte

Manuelle Schritte sind nur zulässig, wenn Plattform oder Benutzerentscheidung sie erfordern. Dokumentation muss erklären:

1. was zu tun ist;
2. warum es nicht sicher automatisiert werden kann;
3. wie das Ergebnis geprüft wird.

Beispiele: physische Obergrenze wählen, DNS beim externen Provider setzen, OAuth-Passwort lokal eingeben und die MCP-App in ChatGPT Web verbinden.

## Abschluss-Gate

Containerstart und lokale Verifikation sind Zwischenstufen.

~~~text
VPS konfiguriert
 -> MCP öffentlich über gültiges HTTPS
 -> OAuth/OIDC funktioniert
 -> ChatGPT Web verbunden
 -> echter ChatGPT-MCP-Aufruf erfolgreich
 -> erwartetes Subject/Policy/Audit bestätigt
 -> INSTALLATION ABGESCHLOSSEN
~~~

`scripts/connect-chatgpt.sh` ist Teil des offiziellen Installationswegs. Erst ein erwarteter authentifizierter ChatGPT-Aufruf im Broker-Audit schließt die Installation ab.

## Nur Entwicklung

Ad-hoc-Diagnosen, temporäre Actions-Probes, spezielle self-hosted Runner, ephemere CI-Maschinen, Entwicklungs-IP/Hostnames/Pfade/Branches und Entwickler-SSH gehören in Issue-/PR-Evidenz, nicht in das öffentliche Tutorial.

## Release-Prüfung

Vor einer Release:

1. README und offizielle Übersetzungen prüfen;
2. `installer-flow.md` und `chatgpt-integration.md` in allen offiziellen Sprachen prüfen;
3. Landing Page prüfen;
4. Entwicklungsartefakte entfernen;
5. sicherstellen, dass keine VPS-/SSH-Credentials verlangt werden;
6. manuelle Schritte mit Grund und Prüfung dokumentieren;
7. Installationsvertrag und i18n-Coverage in CI ausführen;
8. saubere Installation auf unterstützter VPS testen;
9. mit echtem ChatGPT-MCP-Aufruf und Broker-Audit abschließen.

## Engineering-Regel

**NATIVE FIRST.** Offizielle APIs, unterstützte Konfiguration, Docker/Compose, MCP und OAuth/OIDC bevorzugen. Eigener Code nur wenn nötig, dann minimal, zentral, dokumentiert und reversibel.
