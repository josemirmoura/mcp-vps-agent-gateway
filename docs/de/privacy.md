# Datenschutz, Daten und Telemetrie

## Speicherort der Daten

Das Paket wird selbst auf der VPS des Operators gehostet. Betriebszustand, Policy, Audit-Historie und integrierte Identitätsdaten bleiben dort, außer der Operator exportiert sie bewusst.

## Audit

Broker-Audit-Einträge dokumentieren, wer ein Tool aufgerufen hat, welche Ressource/Aktion angefordert wurde, ob sie erlaubt war und welchen Sequenz-/Integritätszustand das ergab.

Audit ist von normalen Service-Logs getrennt.

## Logs

Gateway-, Broker-, Docker- und Identity-Logs können Zeitstempel, Instanzkennungen, Tool-Namen, Fehler und Request-Kontext enthalten. Diagnose-Bundles redigieren konfigurierte Geheimnisse und werden in CI auf Token-Leaks getestet, bleiben aber potenziell sensibel.

## ChatGPT / MCP

Für MCP-Anfragen benötigte Daten können zwischen ChatGPT oder einem anderen MCP-Client und Gateway fließen. Die serverseitige Autorität kontrolliert die Policy. Modellausgabe oder Remote-Inhalt gelten nicht als Autorisierung. Geheimnisse nicht über Dateien, Shell oder Tool-Ergebnisse offenlegen.

## Secrets

Lokale Credentials liegen in root-/operator-kontrollierter Konfiguration oder privaten Volumes. Die Installation fordert nie VPS-Passwörter, private SSH-Schlüssel, root-Passwörter oder fremde Geheimnisse. Die OAuth-Credential ist von VPS/SSH getrennt.

## Telemetry

Kein eigener Analytics-/Tracking-Client. ZITADEL-Telemetrie ist mit `ZITADEL_TELEMETRY_ENABLED=false` deaktiviert. Kein Marketing-Tracking.

## Removal

Safe remove erhält Konfiguration, State und Audit, entfernt Runtime und rotiert lokale Credentials. Full purge entfernt MCP-eigene Artefakte und erhält Anwendungen, Sites, Datenbanken, Drittanbieter-Container, Services und verwaltete Dateien.

~~~bash
bash scripts/remove.sh safe
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~
