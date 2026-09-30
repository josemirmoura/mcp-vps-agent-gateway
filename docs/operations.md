# Operations

This document separates operational observability from security audit.

## Status

~~~bash
bash scripts/diagnose.sh status
~~~

Shows:

- product version;
- Compose service state;
- Broker/Gateway health;
- container restart counts;
- Broker health snapshot;
- audit-chain status.

Use this first when asking: **is the system healthy?**

## Health

~~~bash
bash scripts/diagnose.sh health
~~~

Returns the Broker health snapshot without the broader human-facing status output.

## Logs

~~~bash
bash scripts/diagnose.sh logs 200
~~~

Shows recent service logs with configured secret values redacted.

Operational logs answer questions such as:

- did a service restart?
- did authentication fail?
- did Gateway lose Broker connectivity?
- did a request time out?
- did a container become unhealthy?

## Audit

~~~bash
bash scripts/diagnose.sh audit 100
~~~

Audit answers:

- who invoked the operation?
- which tool/resource/action?
- allow or deny?
- what result?
- which sequence?
- is the hash chain valid?

Audit is not a replacement for service logs, and service logs are not the authorization record.

## Diagnostic bundle

~~~bash
bash scripts/diagnose.sh bundle
~~~

The bundle includes runtime/version information, health, audit status/tail, recent logs, Docker/Compose versions and a redacted policy snapshot.

The bundle is created mode 0600 and CI checks configured token values are absent. Treat it as sensitive operational data anyway and review before sharing.

## Version and update availability

~~~bash
bash scripts/version.sh
bash scripts/version.sh --check
~~~

The first command shows the installed product version. `--check` refreshes release tags when possible and reports whether the checkout is current, ahead of stable, divergent or has a stable update available.

## Update

~~~bash
bash scripts/update.sh
~~~

The default target is the newest stable SemVer tag fetched from origin.

The updater:

1. checks for a clean tracked working tree;
2. displays current/target version and a short change summary;
3. refuses non-fast-forward targets;
4. stops the package;
5. backs up .env, policy, state and integrated identity volumes;
6. validates migration on a copied state DB;
7. builds and verifies the target;
8. runs public verification when integrated auth is enabled;
9. rolls back code/state/identity automatically on failure;
10. appends the result to state/update.log.

To deliberately test a release candidate:

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.3 bash scripts/update.sh
~~~

## Safe remove

~~~bash
bash scripts/remove.sh safe
~~~

Safe remove revokes active delegated elevation, removes the MCP runtime and rotates local active credentials while preserving operator configuration, policy, state/audit and integrated identity state.

Resources previously managed by the MCP remain untouched.

## Reinstall after safe remove

~~~bash
bash scripts/install.sh
~~~

The guided installer reuses preserved operator state/configuration.

## Purge

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Purge removes Portico MCP-owned runtime, volumes, local policy/state, backups and the default locally built Gateway/Broker images. It also attempts to remove the legacy `/opt/vps-agent-sandbox` directory with `rmdir`; a non-empty directory is preserved.

The source checkout is preserved by default. Deleting it requires a second explicit confirmation:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

Source deletion is allowed only when the running directory is verified as this project's Git checkout.

Purge does not delete arbitrary delegated project directories, applications, sites, databases, third-party images/containers, system services or files merely because Portico MCP previously managed them.
