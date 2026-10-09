# Adaptive authorization browser harness

`adaptive_approval.mjs` runs the actual operator HTTPS server, portal assets and MCP Apps wrapper in Chromium. A loopback-only TLS fixture simulates the restricted Broker reply and a reference host bridge. It does not connect to production or prove compatibility in ChatGPT, Claude, VS Code or a physical phone.

The desktop and mobile Chromium profiles cover session reuse with explicit decisions, approve/deny final states, critical password step-up, logout, independent embedded cookies, browser same-origin isolation, host attempts to trigger approval, and portal fallback when framing is not confirmed. `web/operator-approval/test_bridge.mjs` separately tests malformed/spoofed bridge messages, origin/source binding and fallback timeouts without requiring a browser.

The embedded host deliberately grants only `allow-scripts allow-same-origin`,
without `allow-forms` or `allow-modals`. Operator login must work through an
explicit button click or Enter key and same-origin fetch. Native form submission
cannot be the trigger: the [HTML submission algorithm](https://html.spec.whatwg.org/multipage/form-control-infrastructure.html#form-submission-algorithm)
stops for the sandboxed forms flag **before dispatching the submit event**.
The desktop embedded case exercises the button; the mobile profile exercises
Enter. Neither action can approve a request without a separate owner decision.

Locator waits expire before the enclosing test timeout. On failure the suite
records only allowlisted synthetic frame status, browser errors and fixture
call summaries. It never captures input values, cookies, storage, headers or
request bodies. This preserves a useful failure location instead of a bare
30-second cancellation, without skipping or relaxing any assertion.

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
