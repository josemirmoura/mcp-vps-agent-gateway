# Your first Portico: beginner acceptance walkthrough (EN)

> **Do not run this on your production VPS now.** This guide is for accepting an exact Community candidate **after** Block 5 identifies the immutable revision and approves a disposable Linux test host. GitHub CI success does not prove that your actual ChatGPT account works.

[Português (Brasil)](first-install-acceptance.pt-BR.md) · [Quick Start](quick-start.md)

## Plain-language glossary

- **VPS / Linux:** the computer running Portico, possibly a rented server.
- **Terminal:** the window where you type Linux commands.
- **Docker:** the program that launches Portico's separate components.
- **Domain:** a public address, such as `mcp.example.com`.
- **OAuth:** a login page for a **dedicated Portico operator account**, not Linux/SSH access.
- **MCP:** the connection through which a compatible AI chat asks Portico to call tools.
- **Broker:** the local component that verifies and enforces each permission.
- **Standard/Scoped:** Portico has a physical ceiling, usually `/opt`, but **no project folder is automatically unlocked**.

## Before installing anything

1. Ask the integration owner for the exact candidate revision and disposable Linux host. Do not use a computer containing valuable data before acceptance.
2. The test Linux should have Ubuntu 24.04, Docker Engine 24+, Compose v2, Git, OpenSSL, Python 3 and curl. The installer checks these and points you to fixes.
3. Prepare a public DNS hostname that points to that machine, with valid HTTPS and reachable ports 80/443. Changing DNS at your provider is a deliberate owner operation.
4. In your browser, confirm your ChatGPT workspace really offers custom MCP app creation (Developer Mode / Apps or similar). The button names may change. **If you do not see it, do not invent a workaround.** Mark it unavailable for that account.

## 1. Begin installation

Open a terminal on the **disposable Linux host**, not in the AI chat:

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

**Revision gate:** these commands show the workflow. Acceptance must use the immutable revision supplied by Block 5. The moving `main` branch is not itself a signed release, and stable `v0.1.0` has not yet passed publication and acceptance.

What you should see:

- a `Portico MCP` banner and language selection;
- prerequisite checks marked `OK` or a clear item to fix;
- authority choice: pick **1, Standard**, which uses the `/opt` physical ceiling;
- a summary showing **zero authorized project roots**;
- explicit confirmation before starting. Do not select Whole Host for this acceptance.

**If something fails:** read and correct the prerequisite, then rerun the installer. Do not try purge as an installation repair.

## 2. Verify local health

After startup the installer runs local verification. You can repeat it with:

~~~bash
bash scripts/verify.sh
~~~

Expect the Gateway and Broker to be healthy, policy enforced, the audit chain intact and a harmless local `system.info` check. This is **not** proof of a ChatGPT connection.

For a container health failure, run safe diagnostics:

~~~bash
bash scripts/diagnose.sh status
bash scripts/diagnose.sh health
~~~

Never paste passwords, tokens, `.env`, cookies or private keys into an AI chat.

## 3. Configure public HTTPS and OAuth

At the installer prompt, enter only your **hostname**, without `https://` or `/mcp`. Confirm it points to the intended host. The installer configures Traefik when safe, sets up ZITADEL/OAuth and checks the public endpoint.

After safely correcting any DNS/HTTPS problem, rerun:

~~~bash
bash scripts/verify-public.sh
~~~

Expect `INTEGRATED AUTH: READY` from setup, followed by passing public checks.

The **dedicated OAuth operator** password is entered locally during setup and then on the service's login page. It is **not** your VPS password and must never be sent to the chat.

## 4. Connect ChatGPT

The installer invokes `scripts/connect-chatgpt.sh`, prints the exact `https://YOUR-DOMAIN/mcp` endpoint and explains the client setup:

1. Open ChatGPT **in a browser** using an account that has custom MCP apps enabled.
2. Under Settings → Apps, look for custom MCP app creation. The screen may change; if you cannot find it, record the account limitation.
3. Supply exactly the endpoint printed by Portico; select OAuth authentication when offered.
4. On the Portico login page, enter the **dedicated OAuth operator username** shown in the terminal and its service password.
5. Finish creating the app, open a new chat with the app enabled and ask: `Call system.info on my VPS MCP and tell me the hostname.`
6. Return to the Linux terminal and press **Enter** so Portico can check whether a **new** matching audited call arrived. If it did not, review OAuth, the selected app and the test message. Press Enter again when ready.

Expect `CHATGPT WEB CONNECTION VERIFIED` and **`INSTALLATION COMPLETE`** only after real execution, expected identity and audit. Connectivity does **not** grant access to projects.

## 5. Safe Scoped approval test

Use **only a disposable test folder beneath `/opt`**. Never request access to all of `/opt`, secrets or destructive commands during first-time acceptance.

1. Ask ChatGPT to request read-only access to the disposable test folder.
2. Observe the result. Without native client approval support, the request may be **pending**, with `approval_method=operator_fallback`. **Pending is not approved.**
3. The owner must use an authenticated HTTPS operator center **if actually integrated and configured** on the installed build, or an independent trusted SSH session. Never let the AI approve its own request.
4. From the operator's trusted SSH terminal, the existing fallback is:

~~~bash
python3 scripts/operator-approvals.py --list
~~~

5. Verify identity, test path, access level and expiry. **Deny** the first request, proving reads remain blocked. Make a new request, **approve** it deliberately, confirm only this folder can be read, and test expiry/revocation under the acceptance owner's guidance.
6. On desktop and mobile, inspect the full scope, duration and readable controls. A missing or illegible button is a **failure or unsupported feature**, not consent.

If a confirmation method fails, leave the request pending. ChatGPT login is not sufficient proof of authenticated Broker operator consent.

## 6. Stop without deleting the owner's data

On the **disposable Linux host**, stop the Portico runtime while preserving state:

~~~bash
bash scripts/remove.sh safe
~~~

Expect Portico components to stop without deleting user project folders, policy or audit.

**Never use purge for troubleshooting.** Purge needs separate explicit confirmation and can destroy Portico-owned state.

## Record the acceptance evidence

Only record non-sensitive facts: candidate SHA/tag, language, Linux, client/browser, whether MCP app creation was exposed, audited `system.info`, pending approval, denial, approval, expiry/revocation and desktop/mobile display behavior. Mark each **PASS, FAIL, UNSUPPORTED or NOT TESTED**.

**Do not share** screenshots exposing passwords, tokens, cookies, `.env` files or private machine data. Block 5 collects acceptance evidence and makes final integration/release decisions.
