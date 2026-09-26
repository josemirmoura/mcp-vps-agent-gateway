# Contributing

Contributions are welcome.

Please keep changes aligned with the core design:

- security boundaries live server-side
- least privilege by default
- native Linux/platform capabilities before custom machinery
- simple architecture before distributed infrastructure
- client-independent core
- tests for security-sensitive behavior

For significant architecture changes, open an issue first and describe:

1. the problem
2. the proposed change
3. threat-model impact
4. operational cost
5. migration and rollback path

## Pull requests

A useful PR should normally include:

- focused change
- tests where applicable
- documentation updates
- security implications
- rollback considerations for operational changes
