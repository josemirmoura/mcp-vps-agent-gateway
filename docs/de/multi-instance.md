# Mehrere VPS-Instanzen in einem ChatGPT-Workspace

Dies ist ein fortgeschrittenes Szenario. Das normale Onboarding bleibt eine Portico-MCP-Instanz pro Installation.

## Identitätsmodell

Tool-Namen bleiben über Installationen hinweg stabil. Tools nicht pro Server umbenennen.

Jede Installation erhält:

- eigenen HTTPS-MCP-Endpunkt;
- eigene OAuth-Client/Resource-Beziehung und Credentials;
- stabile `VPS_AGENT_INSTANCE_ID` aus dem Bootstrap;
- sichtbaren `VPS_AGENT_INSTANCE_NAME`;
- Instanzidentität in `system.info`, Broker-Audit und strukturierten Logs.

Beispiel:

~~~dotenv
VPS_AGENT_INSTANCE_NAME="VPS Agent | Loja"
VPS_AGENT_PUBLIC_URL=https://mcp-loja.example.com/mcp
~~~

Andere VPS:

~~~dotenv
VPS_AGENT_INSTANCE_NAME="VPS Agent | Blog"
VPS_AGENT_PUBLIC_URL=https://mcp-blog.example.com/mcp
~~~

Als getrennte ChatGPT-Apps mit passenden sichtbaren Namen registrieren. Ziel-App explizit auswählen/@erwähnen statt identische Tool-Namen stillschweigend unterscheiden zu lassen.

## Verifikation

Für jede App `system.info` aufrufen und `instance_id`/`instance_name` mit der erwarteten VPS vergleichen. Hostname nur als Zusatzinformation betrachten, da Provider ihn auf getrennten Wegwerfmaschinen wiederverwenden können.

Danach Broker-Audit prüfen:

~~~bash
bash scripts/diagnose.sh audit 20
~~~

Dieselbe Instanzidentität muss in neuen Audit-Ereignissen und strukturierten Logs erscheinen.

## Nutzung in derselben Unterhaltung

Falls ChatGPT mehrere Custom-MCP-Apps in einer Unterhaltung zulässt, Ziel-App vor Operationen explizit auswählen/erwähnen. UI-Verhalten kann sich unabhängig vom Server ändern.

## Automatisiertes Gleichzeitigkeit-Gate

Ein Drei-Maschinen-Workflow startet das echte Docker-Paket gleichzeitig auf drei unabhängigen Ubuntu-Runnern und verwendet absichtlich dasselbe Subject, dieselben Tool-Namen, denselben logischen Pfad und dieselbe Operation-ID.

Pass nur wenn:

- alle drei eindeutige Instanz-IDs/Namen haben;
- Credentials unabhängig sind;
- Laufzeiten sich überlappen;
- dieselbe Operation-ID in allen drei Broker-State-Stores unabhängig funktioniert;
- jede Audit-Kette nur die eigene Identität enthält;
- Hostnames und Audit-Heads getrennt bleiben;
- jede Instanz im erwarteten lokalen Zustand endet.

Das ist der serverseitige Kollisions-Test ohne menschlichen Eingriff.

## ChatGPT-Produktoberfläche

Mehrere Live-Apps im selben Workspace sind ein separates UI-Experiment und kein Standard-Onboarding- oder Single-Instance-E2E-Kriterium. Wenn getestet, jede VPS getrennt benennen und explizit auswählen.
