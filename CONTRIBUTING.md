# Contributing

Contributions are welcome.

## Engineering principles

Keep changes aligned with the core design:

- security boundaries live server-side;
- least privilege and Scoped authority by default;
- native Linux/platform capabilities before custom machinery;
- simple architecture before distributed infrastructure;
- client-independent MCP core;
- tests for security-sensitive behavior;
- transparent Docker Compose + scripts for installation.

For significant architecture or authority changes, open an issue first and describe:

1. the concrete problem;
2. the proposed change;
3. threat-model impact;
4. operational cost;
5. migration and rollback path.

## Local validation

At minimum for code changes:

~~~bash
go test ./...
bash -n scripts/*.sh
bash -n scripts/lib/*.sh
python3 scripts/check-installation-contract.py
docker compose config -q
~~~

CI adds race detection, vet, vulnerability/security scanning, architecture simulation and disposable host acceptance.

## Pull requests

A useful PR normally includes:

- one focused change;
- tests where applicable;
- negative tests for authority/security changes;
- synchronized documentation;
- explicit security implications;
- migration and rollback considerations for operational changes.

Do not include .env contents, credentials, diagnostic bundles with unreviewed sensitive data, private hostnames/IPs or development-only VPS access details.
