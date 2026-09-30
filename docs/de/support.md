# Support-Richtlinie

Portico MCP ist ein Open-Source-Projekt.

## Unterstützte Release-Linie

Während der initialen Produktisierung:

- die neueste stabile `0.x` Release ist die unterstützte öffentliche Linie;
- der aktuelle Release Candidate wird für Abnahme/Tests unterstützt;
- `main` ist Entwicklung und kein stabiler Supportkanal.

Sicherheitskorrekturen können ein Upgrade auf den neuesten Patch erfordern.

## Hilfe erhalten

GitHub Issues für reproduzierbare Bugs, Installationsfehler und Dokumentationsprobleme ohne Geheimnisse verwenden.

Vor dem Öffnen einer Issue nach Möglichkeit ein bereinigtes Diagnose-Bundle erzeugen:

~~~bash
bash scripts/diagnose.sh bundle
~~~

Nie `.env`, rohe Credentials, private SSH-Schlüssel oder unredigiertes Secret-Material anhängen.

## Sicherheitsprobleme

Exploit-Details nicht in einer normalen Issue veröffentlichen. `SECURITY.md` folgen.

## Service-Level

Für das Open-Source-Projekt gibt es keine garantierte Antwortzeit oder Uptime-SLA.

Supportzusagen einer möglichen zukünftigen kommerziellen Distribution müssen separat dokumentiert werden und dürfen nicht aus diesem Repository abgeleitet werden.
