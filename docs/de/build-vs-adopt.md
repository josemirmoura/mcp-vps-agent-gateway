# Bauen oder übernehmen

Geprüft: 2026-09-26.

Vor dem Bau einer großen Komponente bestehende Lösungen evaluieren.

## Gate -1

Fragen:

1. Verbindet ein bestehendes Produkt den echten Zielclient?
2. Unterstützt es die nötigen Operationen?
3. Erzwingt es serverseitige Berechtigungen?
4. Ist Self-Hosting oder passende Kontrolle/Privatsphäre möglich?
5. Gibt es Recovery/Revocation?
6. Ist Adaptieren günstiger als ein neuer privilegierter Control Plane?

Ergebnisse:

- Adopt
- Adapt/fork
- Build

## Vergleichspunkte

### VPS Guardian MCP

VPS-fokussierter MCP-Server mit strukturierten, sicherheitsgeprüften Operationen ohne generisches Shell-Tool.

Zum Prüfdatum dokumentiert: Python-Server auf der VPS + lokaler npm-Launcher über SSH/stdIO. Stark als Benchmark für typisierte VPS-Operationen, kleine Mutationsfläche, Bestätigungen, Diagnose/Rollback und Release-Disziplin.

Er löst ein anderes Client-Transportproblem als die Zielarchitektur für Remote ChatGPT Web.

Projekt:
https://github.com/murzirius/VPS-Guardian-MCP

### Remote Desktop Commander

Gehosteter Remote-MCP-Service für Filesystem und Terminal mit Streamable HTTP, OAuth und gepaartem Device Agent.

Starker Benchmark für Remote-MCP-UX, Pairing/Revocation, OAuth, Terminal-/Filesystem-Ergonomie und Multi-Client-Support.

Sein Hosted Service ist nicht die hier beschriebene Self-Hosted-Broker-Architektur.

Projekt:
https://github.com/desktop-commander/remote-desktop-commander

## Warum diese Architektur bauen

Wenn die nötige Kombination lautet:

- selbst kontrollierte VPS-Grenze;
- ChatGPT Web als Ziel;
- serverseitige Scoped-Policy;
- typisierte Linux/Docker/systemd-Operationen;
- optional kontrollierte Shell;
- optionale temporäre Elevation;
- explizite Recovery-Semantik;
- operator-eigener privilegierter Broker.

## Regel

Keine Komponente nur bauen, weil sie im North-Star-Diagramm steht.

Bauen nur, wenn bestehende Lösungen den Bedarf nicht erfüllen und das vorherige MVP-Gate die Notwendigkeit belegt.
