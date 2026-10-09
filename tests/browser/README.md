# Adaptive authorization browser harness

`adaptive_approval.mjs` runs the actual operator HTTPS server, portal assets and MCP Apps wrapper in Chromium. A loopback-only TLS fixture simulates the restricted Broker reply and a reference host bridge. It does not connect to production or prove compatibility in ChatGPT, Claude, VS Code or a physical phone.

The 16 desktop/mobile Chromium cases cover session reuse with explicit decisions, approve/deny final states, critical password step-up, logout, browser same-origin isolation, host attempts to trigger approval, and portal fallback when framing is not confirmed. The fixture maps three independent HTTPS sites to loopback and adds a second AI-host site: assertions verify actual CHIPS partition keys, prevent another host from inheriting an owner session, exercise click/Enter login without `allow-forms`, and click the external fallback through host `ui/open-link` without `allow-popups`. `web/operator-approval/test_bridge.mjs` separately tests malformed/spoofed bridge messages, origin/source binding, late responses for a reused app and fallback timeouts without requiring a browser.

Run the lightweight bridge tests:

```sh
node --test web/operator-approval/test_bridge.mjs
```

Install the exact Playwright version used by CI in a temporary tools directory, then run the browser suite:

```sh
npm install --prefix /tmp/portico-browser-tools --no-audit --no-fund playwright@1.62.1
node /tmp/portico-browser-tools/node_modules/playwright/cli.js install --with-deps chromium
PORTICO_PLAYWRIGHT_MODULE=/tmp/portico-browser-tools/node_modules/playwright \
  node --test tests/browser/adaptive_approval.mjs
```

The suite requires Python 3, OpenSSL and a Chromium binary. A missing browser fails visibly; no test is silently skipped. Screenshots contain only synthetic fixture data and go to `/tmp/portico-browser-artifacts` by default. Override with `PORTICO_BROWSER_ARTIFACT_DIR`. Certificates and the synthetic password verifier stay in a temporary directory and are deleted when the fixture exits.

Real-client evidence must additionally record the actual client/account/version, capabilities from MCP initialize, the Apps handshake, observed sandbox ancestors, owner authentication, approve/deny/expiry/replay outcomes and audit evidence on an isolated staging node. A successful reference harness must not be recorded as a real-client pass.
