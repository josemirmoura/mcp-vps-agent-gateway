# Betrieb

## Status

~~~bash
bash scripts/diagnose.sh status
~~~

Zeigt Produktversion, Compose-Zustand, Broker/Gateway-Health, Neustartzähler, Broker-Health und Audit-Kettenstatus.

## Health

~~~bash
bash scripts/diagnose.sh health
~~~

Liefert nur den Broker-Health-Snapshot.

## Logs

~~~bash
bash scripts/diagnose.sh logs 200
~~~

Aktuelle Logs mit redigierten konfigurierten Geheimnissen. Damit untersucht man Neustarts, Auth-Fehler, Gateway/Broker-Verbindungsprobleme, Timeouts und unhealthy Container.

## Audit

~~~bash
bash scripts/diagnose.sh audit 100
~~~

Audit beantwortet: wer, welches Tool/Resource/Action, allow/deny, Ergebnis, Sequenz und Hash-Kettenintegrität. Logs und Audit sind unterschiedliche Ebenen.

## Diagnose-Bundle

~~~bash
bash scripts/diagnose.sh bundle
~~~

Enthält Runtime/Version, Health, Audit-Status/-Tail, Logs, Docker/Compose-Versionen und redigierte Policy. Datei wird als 0600 erstellt. Vor Weitergabe trotzdem prüfen.

## Version

~~~bash
bash scripts/version.sh
bash scripts/version.sh --check
~~~

`--check` aktualisiert Release-Tags soweit möglich und meldet current/ahead/divergent/update available.

## Update

~~~bash
bash scripts/update.sh
~~~

Standardziel ist der neueste stabile SemVer-Tag. Der Updater prüft sauberen Working Tree, zeigt Änderungen, verweigert non-fast-forward, stoppt das Paket, sichert `.env`, Policy, State und Identity-Volumes, testet Migration auf einer DB-Kopie, baut und prüft das Ziel und rollt bei Fehler automatisch zurück.

RC explizit:

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~

## Sichere Entfernung

~~~bash
bash scripts/remove.sh safe
~~~

Temporäre Berechtigungen werden widerrufen, Runtime entfernt, lokale Credentials rotiert. Operator-Konfiguration, Policy, Audit/State und integrierte Identität bleiben erhalten.

## Purge

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Quellcode wird nur mit zweiter Bestätigung entfernt:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

Delegierte Anwendungen, Sites, Datenbanken, Fremdimages/-container, Services und Dateien werden nicht gelöscht.
