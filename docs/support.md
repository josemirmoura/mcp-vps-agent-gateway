# Community support policy

This repository contains the public Portico Community runtime.

## Supported release line

During initial productization:

- the newest stable 0.x release will be the supported public line;
- the current release candidate is supported for acceptance/testing;
- main is development and is not a stable support channel.

Security fixes may require upgrading to the newest patch release.

## Getting help

Use a GitHub issue for reproducible Community/runtime bugs, installation failures and public documentation problems that do not contain secrets.

When practical:

```bash
bash scripts/diagnose.sh bundle
```

Never attach .env files, raw credentials, private SSH keys or unredacted secret material.

## Security issues

Do not publish exploit details in an ordinary issue. Follow SECURITY.md.

## Service level

The public Community repository has no guaranteed response-time or uptime SLA.

Any commercial support commitments belong to the corresponding commercial service terms and are not implied by this repository.
