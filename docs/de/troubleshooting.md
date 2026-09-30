# Fehlerbehebung

Zuerst:

~~~bash
bash scripts/diagnose.sh status
~~~

Bei Bedarf:

~~~bash
bash scripts/diagnose.sh bundle
~~~

Bundle vor Weitergabe prüfen.

## Scope-Verzeichnis fehlt

Portico erzeugt beliebige Verzeichnisse nicht stillschweigend:

~~~bash
sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 /opt/my-app
~~~

Oder Installer erneut starten und sichtbaren sudo-Schritt bestätigen.

## Pfad nicht kanonisch / Symlink

Absoluten kanonischen Pfad ohne Dot-Segmente oder Symlink-Vorfahren verwenden. Das ist absichtliches Boundary-Hardening.

## Broker/Gateway wird nicht healthy

~~~bash
docker compose ps -a
docker compose logs --no-color --tail 100 broker gateway
bash scripts/diagnose.sh status
~~~

Health-Gate nicht umgehen.

## Ports 80/443 belegt

Ein vorhandenes Traefik wird nur wiederverwendet, wenn es sicher identifiziert werden kann. Ein unbekannter Webserver wird nicht ersetzt.

## Mehrere Traefik

~~~bash
bash scripts/setup-integrated-auth.sh --edge-network YOUR_NETWORK
~~~

Bei mehrdeutigem ACME-Resolver zusätzlich:

~~~bash
--certresolver YOUR_RESOLVER
~~~

## DNS löst nicht auf

A/AAAA korrigieren und Propagation abwarten. Setup prüft DNS vor öffentlicher Konfiguration.

## Öffentliche OAuth-Prüfung schlägt fehl

~~~bash
bash scripts/verify-public.sh
~~~

DNS, Zertifikat, Hostname/Issuer, Protected-Resource-Metadata, OIDC DCR/PKCE und Proxy-Routen prüfen. Issuer/Audience nicht abschwächen.

## ChatGPT-Verbindung fehlt

~~~bash
bash scripts/connect-chatgpt.sh
~~~

MCP-App, OAuth und Auswahl von Portico prüfen und ChatGPT `system.info` aufrufen lassen. Timeout bedeutet nicht Installation abgeschlossen.

## Kein stabiler Release für update.sh

Vor dem ersten Stable gibt es absichtlich kein automatisches Ziel:

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~

`main` ist kein automatischer Produktionskanal.

## Update rollt zurück

Backup nicht löschen. Output, `backups/<timestamp>/migration-check.json`, `state/update.log` und Diagnose prüfen. Rollback ist Schutzfunktion.

## Geheimnis im Diagnoseartefakt

Nicht teilen. Lokal sichern, betroffene Credentials ggf. rotieren und `SECURITY.md` befolgen.
