# ChatGPT-Integration

Geprüft: 2026-09-30.

## Ziel

~~~text
ChatGPT Web
 -> OAuth Discovery + Dynamic Client Registration
 -> HTTPS /mcp
 -> Gateway
 -> Broker
 -> VPS
 -> manipulationssicheres Audit
~~~

Container-Health allein schließt die Installation nicht ab.

## ChatGPT-Voraussetzung

Prüfe vor dem öffentlichen Setup, ob das konkrete Konto/Workspace Developer Mode und benutzerdefinierte MCP-App-Erstellung anbietet und welche Berechtigungen dort tatsächlich verfügbar sind. OpenAI steuert Rollout und UI.

Projekt-Referenzen:

- https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt
- https://developers.openai.com/plugins/build/app-quickstart

Read-only kann diagnostisch nützlich sein, ist aber nicht gleichbedeutend mit dem vollständigen write-fähigen Produktweg.

## Voraussetzungen

1. MCP-App-Erstellung ist verfügbar.
2. `bash scripts/verify.sh` war erfolgreich.
3. DNS zeigt auf die VPS.
4. Integrierte Auth ist bereit:

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Erwartet:

~~~text
INTEGRATED AUTH: READY
~~~

Öffentliche Grenze erneut testen:

~~~bash
bash scripts/verify-public.sh
~~~

## Verbindung

~~~bash
bash scripts/connect-chatgpt.sh
~~~

Erwarteter Flow:

1. ChatGPT liest RFC-9728 Protected-Resource-Metadata;
2. entdeckt ZITADEL;
3. registriert einen öffentlichen OAuth-Client dynamisch;
4. Operator meldet sich mit dediziertem OAuth-Konto an;
5. Authorization Code + PKCE;
6. ChatGPT entdeckt MCP-Tools;
7. `system.info` wird aufgerufen;
8. Gateway validiert das Token;
9. Broker prüft stabiles Subject und Policy;
10. Audit registriert den Aufruf.

ChatGPT erhält niemals VPS-Passwort, privaten SSH-Schlüssel, root oder uneingeschränkte Remote-Shell. Das OAuth-Passwort wird nur in der Identity-Provider-Anmeldung eingegeben.

## Abschluss-Gate

`connect-chatgpt.sh` speichert eine Audit-Basislinie. Nach dem Setup drückt der Operator Enter; Portico sucht einen neueren authentifizierten `system.info`-Aufruf.

~~~text
Tutorial angezeigt
 != Erfolg

ChatGPT-App verbunden
 + authentifiziertes system.info
 + erwartetes Subject
 + Broker allow
 + erfolgreiche Ausführung
 + passendes Audit
 = INSTALLATION COMPLETE
~~~

## Transport

~~~text
https://<domain>/mcp
~~~

MCP Streamable HTTP über HTTPS, kein eigener WebSocket-Transport.

## Sicherheitsgrenze

ChatGPT-Bestätigungen sind zusätzliche UX-Kontrollen, nicht die Autorisierung.

**Gateway authentifiziert. Broker autorisiert. Der VPS-Eigentümer wählt die Policy.**
