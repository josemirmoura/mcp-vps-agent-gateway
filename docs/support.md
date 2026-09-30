# Support policy

Portico MCP is an open-source project.

## Supported release line

During the initial productization period:

- the newest stable `0.x` release is the supported public line;
- the current release candidate is supported for acceptance/testing;
- `main` is development and is not a stable support channel.

Security fixes may require upgrading to the newest patch release.

## Getting help

Use a GitHub issue for reproducible bugs, installation failures and documentation problems that do not contain secrets.

Before opening an issue, collect a sanitized diagnostic bundle when practical:

~~~bash
bash scripts/diagnose.sh bundle
~~~

Never attach `.env`, raw credentials, private SSH keys or unredacted secret material.

## Security issues

Do not publish exploit details in an ordinary issue. Follow `SECURITY.md`.

## Service level

There is no guaranteed response-time or uptime SLA for the open-source project.

Support commitments for a future commercial distribution, if any, must be documented separately and must not be inferred from this repository.
