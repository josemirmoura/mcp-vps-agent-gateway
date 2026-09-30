# Kompatibilität

## Für den Release Candidate validiert

### Betriebssystem

- Ubuntu 24.04 LTS auf sauberen GitHub-Runnern;
- Linux mit systemd für Host-Service-/Job-Funktionen.

### Architektur

- `linux/amd64`: vollständige Runtime-Abnahme in sauberer CI;
- `linux/arm64`: Binaries/Images werden gebaut, aber gleichwertige native Host-Abnahme steht noch aus.

### Runtime

- Docker Engine 24+;
- Docker Compose v2 als `docker compose`;
- mindestens 2 GB RAM für den gebündelten ZITADEL-Pfad;
- Git, OpenSSL, Python 3 und curl.

2 GB ist ein funktionales Minimum der Identity-Abhängigkeit, keine Produktions-Sizing-Empfehlung.

Referenzen:
- https://zitadel.com/docs/self-hosting/deploy/compose
- https://zitadel.com/docs/self-hosting/manage/requirements

## Öffentlicher ChatGPT-Weg

Erfordert DNS A/AAAA, öffentliches TCP 80/443 über kompatibles vorhandenes oder gebündeltes Traefik, gültiges HTTPS und ein ChatGPT-Konto/Workspace, das Developer Mode und benutzerdefinierte MCP-App-Registrierung tatsächlich anbietet.

OpenAI steuert Plan-/Workspace-Verfügbarkeit und Rollout. Portico kann ChatGPT-seitige Berechtigungen nicht erhöhen.

Projekt-Referenz:
https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt

## Build-Abdeckung

- `linux/amd64`
- `linux/arm64`

## Derzeit nicht als unterstützt zugesagt

- Nicht-Linux;
- Docker Desktop als Produktions-VPS;
- Linux ohne systemd für Host-Service/Jobs;
- Kubernetes;
- Podman Compose;
- native Windows/macOS;
- Full/R5 Produktion.

Andere Linux-Distributionen können funktionieren, sind aber bis zu expliziter Evidenz ungetestet.

## Ressourcen

RAM-Minimum ist nur die Dependency-Untergrenze. Passwort-Hashing und reale MCP-/Docker-Last können deutlich mehr CPU/RAM benötigen. Produktionsgröße anhand gemessener Workloads und `scripts/diagnose.sh health` bestimmen.
