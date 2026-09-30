# Compatibility

This document distinguishes validated support from build coverage and untested environments.

## Validated for the release candidate

### Operating system

- Ubuntu 24.04 LTS on GitHub-hosted clean runners
- Linux host with systemd for host-service and durable-job functionality

### Architecture

- `linux/amd64`: runtime acceptance exercised in clean CI
- `linux/arm64`: binaries and container images are built; equivalent full host-runtime acceptance has not yet been executed on native arm64 hardware

### Runtime

Required runtime floor for the bundled OAuth path:

- Docker Engine 24 or newer;
- Docker Compose v2 plugin exposed as `docker compose`;
- at least 2 GB RAM for the machine hosting the bundled ZITADEL Compose path;
- Git;
- OpenSSL;
- Python 3;
- curl.

The Docker/RAM floor follows the current official ZITADEL Docker Compose prerequisites because integrated OAuth is part of the supported public installation. It is a functional minimum, not a production-sizing recommendation.

References:

- https://zitadel.com/docs/self-hosting/deploy/compose
- https://zitadel.com/docs/self-hosting/manage/requirements

## Public ChatGPT path

A completed public installation requires:

- a DNS A/AAAA record controlled by the operator;
- public TCP 80/443 availability through an existing compatible Traefik edge or the bundled Traefik path;
- a valid HTTPS certificate;
- a ChatGPT Web account/workspace whose current product surface actually exposes Developer Mode and custom MCP app registration.

OpenAI controls plan availability, workspace controls and rollout. The project therefore does not treat a subscription label alone as a compatibility guarantee. Current official OpenAI documentation lists full MCP support, including write/modify actions, for Business, Enterprise and Edu; Pro users can connect custom MCPs with read/fetch permissions in developer mode. Any account whose ChatGPT surface exposes narrower capabilities remains limited to those capabilities; Portico cannot elevate ChatGPT-side permissions.

Current official reference:
- https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt

The package uses one public hostname for MCP protected-resource metadata and the integrated OAuth/OIDC issuer.

## Known to build

Release automation builds:

- `linux/amd64`
- `linux/arm64`

## Not currently claimed as supported

No compatibility promise is currently made for:

- non-Linux hosts;
- Docker Desktop as a production VPS target;
- Linux distributions without systemd for host-service/job features;
- Kubernetes deployment;
- Podman Compose;
- native Windows or macOS hosts;
- Full/R5 production use.

Other Linux distributions may work but are untested until explicit acceptance evidence is added.

## Resource sizing

The 2 GB RAM figure above is only the dependency floor for the bundled identity stack. ZITADEL notes that password hashing can create CPU spikes and publishes substantially higher production/HA guidance.

MCP workload size, Docker workloads already present on the VPS and audit/log retention materially affect real requirements. Operators should use `scripts/diagnose.sh health` and host metrics; production sizing should be based on measured workload evidence rather than the minimum installation floor.
