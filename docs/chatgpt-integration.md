# ChatGPT integration

Checked: 2026-09-28.

## Target

~~~text
ChatGPT Web
   -> OAuth discovery + dynamic client registration
   -> HTTPS /mcp
   -> Gateway
   -> Broker
   -> VPS
   -> tamper-evident audit
~~~

The installation completion rule is deliberately stricter than "containers are healthy."

## ChatGPT product prerequisite

ChatGPT product surfaces and plan rollouts change independently from this project. Do **not** hard-block installation based only on a subscription name.

The supported rule is feature detection:

1. open the target ChatGPT account/workspace;
2. confirm Developer Mode / Plugins exposes the option to create a custom MCP app;
3. if the option exists, continue with the connection test;
4. if it does not, recheck current official OpenAI documentation for that account/workspace.

As of 2026-09-28, OpenAI documentation is in transition: one Help Center article describes full MCP write/modify support as a Business/Enterprise/Edu rollout, while newer Plugins documentation describes the developer-mode MCP registration flow more generally. The actual feature surface exposed to the account is therefore the installation-time authority for availability.

## Prerequisites

1. The target ChatGPT account/workspace actually exposes custom MCP app creation in Developer Mode / Plugins.
2. Local installation has already passed `bash scripts/verify.sh`.
3. A DNS hostname points to the VPS.
4. Integrated auth has passed:

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

That command must finish with `INTEGRATED AUTH: READY`.

To rerun only the public boundary checks after any DNS, proxy or OAuth change:

~~~bash
bash scripts/verify-public.sh
~~~

## Connection

Run:

~~~bash
bash scripts/connect-chatgpt.sh
~~~

The script first reruns the public checks, prints the exact MCP endpoint and OAuth issuer, then shows the current ChatGPT connection steps. The public preflight validates HTTPS, RFC 9728 metadata, HTTPS authorization/token endpoints, DCR, PKCE S256, refresh-token support, token-endpoint authentication metadata, private audience-bound introspection, and the unauthenticated `WWW-Authenticate` discovery challenge.

The intended standards flow is:

1. ChatGPT reads RFC 9728 protected-resource metadata from the MCP host.
2. ChatGPT discovers the ZITADEL authorization server.
3. ChatGPT dynamically registers an OAuth public client.
4. The operator signs in with the dedicated OAuth operator account. VPS passwords and SSH keys are never entered into ChatGPT.
5. Authorization Code + PKCE completes.
6. ChatGPT discovers the MCP tools.
7. The operator invokes `system.info`.
8. The Gateway validates the access token against the integrated issuer.
9. The Broker matches the stable subject and applies policy.
10. The audit chain records the successful tool invocation.

The ChatGPT product UI is version-sensitive, so button names and workspace eligibility must be rechecked against current official OpenAI documentation at release/setup time.

## Credential boundary

ChatGPT receives the public MCP endpoint and completes the standards-based OAuth flow. It does not need the VPS login password, a private SSH key, root credentials, unrestricted remote shell access, or unrelated infrastructure secrets.

The dedicated OAuth operator password is entered only into the integrated identity provider login flow. Development-only remote access used by maintainers to validate a build is not part of this supported connection procedure.

## Completion gate

`scripts/connect-chatgpt.sh` waits on the Broker for a new authenticated `system.info` event from the expected subject.

~~~text
tutorial shown
 != success

ChatGPT app connected
 + authenticated system.info
 + expected subject
 + Broker policy allow
 + successful VPS execution
 + matching audit record
 = INSTALLATION COMPLETE
~~~

If that event does not arrive before the timeout, installation is still incomplete.

## Transport

The MCP transport is Streamable HTTP over HTTPS at:

~~~text
https://<domain>/mcp
~~~

No custom WebSocket transport is introduced.

## Security boundary

ChatGPT confirmations and app permissions are additional UX controls. They never replace server-side authorization.

The Gateway authenticates. The Broker authorizes. The VPS owner chooses the policy.
