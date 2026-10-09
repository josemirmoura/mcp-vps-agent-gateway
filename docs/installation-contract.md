# Installation contract

[Português (Brasil)](installation-contract.pt-BR.md)

Status: **normative project decision**.

This document defines the boundary between temporary development/validation procedures and the supported end-user installation experience. Implementation, tests, README/tutorials and release review must follow it.

## Supported user experience

The supported installation is:

- terminal-first;
- Docker Compose first;
- based on transparent, inspectable commands and scripts;
- reproducible on the user's own VPS;
- usable without assistance from this project's developers;
- native-first: official platform mechanisms, Docker, MCP and standards-based OAuth/OIDC before custom glue.

The normal user installs and operates the package directly on the VPS. ChatGPT is connected only after the server-side installation and public authentication path are ready.

## Secrets and remote access

The supported tutorial must **never instruct the user to give ChatGPT or a project maintainer**:

- the VPS password;
- a private SSH key;
- unrestricted SSH or other remote administrative access;
- root credentials;
- cloud-provider administrative credentials;
- terminal output containing secrets;
- any secret that is not strictly required to configure this service.

Secrets that are strictly required by the service are entered locally on the VPS or through the corresponding native service UI/API. For example, the dedicated OAuth operator password is entered locally by the setup script and is not supplied to ChatGPT.

Development sessions may temporarily require an operator to run commands and return sanitized diagnostics because the development agent lacks a direct execution channel. That is a limitation of the development environment, **not** a product requirement and must not be copied into user-facing installation instructions.

## Automation responsibility

If a deterministic installation task can reasonably be automated, the package should absorb it. This includes, where applicable:

- dependency detection and actionable prerequisite errors;
- port and edge-proxy detection;
- Compose configuration validation;
- container startup and health validation;
- HTTPS setup and validation;
- OAuth/OIDC bootstrap and discovery validation;
- MCP endpoint checks;
- fail-closed authentication tests;
- diagnostic collection with secret redaction;
- clear recovery instructions when automation cannot safely choose for the user.

Do not turn one-off investigation commands used during development into mandatory tutorial steps.

## Deliberate manual actions

Manual actions are acceptable only when the platform inherently requires an operator decision or browser/UI action. The documentation must say exactly:

1. what the user must do;
2. why it cannot be safely automated;
3. how to validate the result.

Examples include selecting the VPS authority scope, creating or pointing DNS when the package does not control DNS, entering the dedicated OAuth operator password locally, and connecting the MCP app in ChatGPT Web.

## Installation completion gate

Container startup and local verification are intermediate gates, not completion.

The supported completion sequence is:

~~~text
VPS configured
 -> MCP public over valid HTTPS
 -> OAuth/OIDC authentication works
 -> ChatGPT Web connection configured
 -> real ChatGPT MCP call succeeds
 -> expected subject/policy/audit confirmed
 -> INSTALLATION COMPLETE
~~~

\`scripts/connect-chatgpt.sh\` is part of the official installation flow. The installation is not complete until the Broker observes the expected authenticated ChatGPT call and the audit chain remains valid.

## Development-only procedures

The following belong to development/acceptance work unless explicitly promoted into a safe supported feature:

- asking the project owner to run ad-hoc diagnostic commands and paste output;
- temporary GitHub Actions probes;
- project-specific self-hosted runners;
- ephemeral CI machines;
- target-VPS hostnames, IP addresses, checkout paths or branch names;
- temporary development branches;
- manual curl/openssl probes created only to investigate a failure;
- developer-only SSH access or remote-shell work.

Such procedures belong in issue/PR evidence or dedicated development notes, not in README, the installation tutorial, the public site, or the supported ChatGPT connection flow.

## Final release review

Before a release is declared ready:

1. review README.md and README.pt-BR.md;
2. review docs/installer-flow.md and docs/chatgpt-integration.md;
3. review the public landing page;
4. remove development-only hostnames, IPs, branch names, runner names and ad-hoc diagnostics;
5. verify the tutorial never asks the user to share VPS credentials or private keys;
6. verify every unavoidable manual action has a reason and a validation step;
7. run `python3 scripts/check-installation-contract.py` in CI;
8. run the full installation from a clean supported VPS state;
9. finish with a real ChatGPT Web MCP call and Broker audit evidence.

## Engineering rule

**NATIVE FIRST.** Prefer official APIs, supported configuration, Docker/Compose, MCP, OAuth/OIDC and native platform mechanisms. Custom code is used only when the native path is insufficient, and then must be minimal, centralized, documented and reversible.

[Português (Brasil)](installation-contract.pt-BR.md).
