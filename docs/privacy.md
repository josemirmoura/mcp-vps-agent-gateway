# Privacy, data and telemetry

## Data location

The package is self-hosted on the operator's VPS.

Operational state, policy, audit history and integrated identity data remain on that VPS unless the operator deliberately exports a diagnostic bundle, backup or other artifact.

## Audit data

Broker audit records are designed to answer who invoked a tool, what resource/action was requested, whether it was allowed, and the resulting sequence/integrity state.

Audit is distinct from ordinary service logs.

## Operational logs

Gateway, Broker, Docker and identity-service logs may contain operational metadata such as timestamps, instance identifiers, tool names, errors and request context.

Diagnostic bundles run redaction over configured secrets and are tested in CI against token leakage. Operators should still treat diagnostic bundles as potentially sensitive operational data.

## ChatGPT and MCP traffic

Data required to satisfy an MCP request can pass between ChatGPT (or another MCP client) and the Gateway.

The operator controls the server-side authority through policy. The server does not treat model output or remote content as authorization.

Do not expose secrets through generic files, shell output or tool results.

## Secrets

Local service credentials are stored in root/operator-controlled configuration or private service volumes. The supported installation never asks the user to send VPS passwords, private SSH keys, root passwords or unrelated infrastructure secrets to ChatGPT or a maintainer.

The dedicated OAuth operator credential is entered into the integrated identity login/bootstrap flow and is distinct from VPS/SSH credentials.

## Telemetry

The project contains no first-party analytics or usage-tracking client.

The bundled ZITADEL configuration explicitly disables ZITADEL telemetry with `ZITADEL_TELEMETRY_ENABLED=false`.

No marketing tracking is introduced by the package.

## Removal

Safe remove preserves configuration, state and audit history while removing the active MCP runtime and rotating active local credentials.

~~~bash
bash scripts/remove.sh safe
~~~

Full purge removes MCP-owned local configuration/state and identity artifacts while preserving applications, sites, databases, third-party containers, services and files that the MCP previously administered.

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~
