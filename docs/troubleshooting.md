# Troubleshooting

Start with:

~~~bash
bash scripts/diagnose.sh status
~~~

If the problem is not obvious, create a local diagnostic bundle:

~~~bash
bash scripts/diagnose.sh bundle
~~~

Review the bundle before sharing it.

## Scope directory does not exist

The installer refuses to create arbitrary directories silently.

Create the chosen path explicitly:

~~~bash
sudo install -d -o "$USER" -g "$(id -gn)" -m 0750 /opt/my-app
~~~

Or rerun the guided installer and approve the visible sudo creation step.

## Scope path is rejected as non-canonical or symlinked

Use an absolute canonical path with no dot segments or symlinked ancestors.

This is intentional filesystem-boundary hardening.

## Broker or Gateway never becomes healthy

Run:

~~~bash
docker compose ps -a
docker compose logs --no-color --tail 100 broker gateway
bash scripts/diagnose.sh status
~~~

Do not bypass the health gate. Fix the failing service and rerun the same installer.

## Ports 80/443 are already in use

Integrated auth reuses one existing Traefik edge when it can identify it safely.

If another web server owns 80/443 and no reusable Traefik is detected, setup fails rather than replacing the existing edge.

Resolve the edge architecture deliberately before rerunning.

## Multiple Traefik containers are detected

Select the intended public edge explicitly:

~~~bash
bash scripts/setup-integrated-auth.sh --edge-network YOUR_NETWORK
~~~

If certificate-resolver discovery is ambiguous, also pass:

~~~bash
--certresolver YOUR_RESOLVER
~~~

## DNS does not resolve

Create or correct the A/AAAA record for the intended MCP hostname and wait for propagation. Then rerun setup.

The installer validates DNS before public configuration.

## Public OAuth verification fails

Run:

~~~bash
bash scripts/verify-public.sh
~~~

Check:

- DNS points to the expected VPS;
- certificate is valid;
- public hostname matches the configured issuer;
- protected-resource metadata is reachable;
- OIDC discovery advertises DCR and PKCE S256;
- ports/proxy routes are correct.

Do not weaken issuer/audience validation to make the check pass.

## ChatGPT connection times out

The server side may be healthy while the ChatGPT-side app was not created, authenticated or selected.

Rerun:

~~~bash
bash scripts/connect-chatgpt.sh
~~~

Then ensure the target ChatGPT workspace exposes custom MCP app registration, authenticate with the dedicated OAuth operator account and ask ChatGPT to call system.info.

The timeout does not mark installation complete.

## update.sh says no stable release exists

Before the first stable tag, automatic production update has no default target by design.

For an explicitly selected release candidate:

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.3 bash scripts/update.sh
~~~

main is not an automatic update channel.

## Update rolls back

Do not delete the backup directory. Inspect:

- the updater output;
- backups/<timestamp>/migration-check.json;
- state/update.log;
- bash scripts/diagnose.sh status.

The rollback is a safety feature, not a partial success.

## Secret or sensitive data appears in a diagnostic artifact

Do not share the artifact. Preserve it locally, rotate affected credentials if exposure is possible, and follow SECURITY.md.
