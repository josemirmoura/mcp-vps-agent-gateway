# FAQ

## Bedeutet Whole Host Full?

Nein. Whole Host setzt die physische Dateisystem-Obergrenze des Brokers auf `/`. Full ist ein separates Capability-Bundle über Policy/Feature-Gates und standardmäßig deaktiviert.

## Bedeutet Full uneingeschränktes Internet?

Nein. Netzwerk ist separat.

## Bekommt ChatGPT mein VPS-Passwort oder meinen SSH-Schlüssel?

Nein. Installation läuft lokal auf der VPS; ChatGPT erhält den öffentlichen MCP-Endpunkt und OAuth.

## Warum ist der Broker privilegiert?

Host-systemd, Docker und delegierte Host-Dateisystemoperationen benötigen einen vertrauenswürdigen privilegierten Teil. Broker ist lokal und policy-autoritativ; Gateway bleibt non-root ohne Docker-Socket oder Host-root.

## Ist Docker die Sicherheitsgrenze?

Nein. Docker ist Packaging/Lifecycle. Server-seitige Broker-Autorisierung ist die effektive Grenze.

## Telemetrie?

Kein eigener Analytics-/Tracking-Client. ZITADEL-Telemetrie ist deaktiviert. Siehe `privacy.md`.

## Nur ein Projekt?

Ja, Profil **Project**.

## Empfohlener Standard?

**Standard**: `/opt` als physische Obergrenze, zunächst keine Projektwurzel, spätere explizite Freigaben.

## Mehrere Verzeichnisse?

Ja, dynamisch unterhalb der Obergrenze oder statisch für fortgeschrittene Operatoren.

## Mehrere VPS?

Ja. Jede Installation hat eigene Identität, Credentials, State und Audit-Kette.

## Andere MCP-Clients?

Core ist standardbasiertes MCP Streamable HTTP. Der öffentliche Produktweg ist mit ChatGPT Web validiert; kompatible Clients können funktionieren, wenn sie den Auth-Flow unterstützen.

## Wann ist Installation abgeschlossen?

Erst nach einem echten Client-Aufruf durch Authentifizierung, Gateway, Broker, Policy, Ausführung und Audit.

## Warum update.sh nicht main?

`main` ist Entwicklung. Produktion folgt stabilen SemVer-Tags.

## Portico löschen ohne Apps zu löschen?

Ja. Safe remove und purge entfernen Portico-eigene Artefakte und lassen verwaltete Ressourcen stehen.
