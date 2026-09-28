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

Required commands:

- Docker Engine
- Docker Compose v2 plugin exposed as `docker compose`
- Git
- OpenSSL
- Python 3
- curl

The current release candidate does not claim a lower Docker Engine or Compose version floor that has not been explicitly exercised in CI. The installer checks required functionality rather than publishing an unverified minimum version.

## Public ChatGPT path

A completed public installation requires:

- a DNS A/AAAA record controlled by the operator;
- public TCP 80/443 availability through an existing compatible Traefik edge or the bundled Traefik path;
- a valid HTTPS certificate;
- a ChatGPT account/workspace whose current product surface exposes custom MCP app registration.

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

No minimum CPU/RAM claim is published yet because the integrated identity stack and workload mix materially affect consumption. Operators should use `scripts/diagnose.sh health` and host metrics during the release-candidate period; a sizing floor should be published only after measured real-VPS evidence.
