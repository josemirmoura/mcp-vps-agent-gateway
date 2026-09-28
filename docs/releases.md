# Release and version policy

MCP VPS Agent Gateway follows Semantic Versioning.

## Channels

### Stable

Stable releases use tags such as `v0.1.0`, `v0.1.1` and `v0.2.0`.

A stable release is the normal update channel. `main` is not a production update channel.

### Release candidate

Release candidates use SemVer pre-release tags, for example `v0.1.0-rc.1`.

They are intended for final acceptance and may receive fixes before the corresponding stable release.

### Development

`main` is the current development branch. It may be ahead of the latest release and is not the default target for `scripts/update.sh`.

## First public release

The productization branch identifies itself as `0.1.0-rc.1`.

The stable `v0.1.0` tag is created only after the owner's final clean-install human gate succeeds.

## Publishing a release

The native GitHub Actions path can be started from **Actions -> release -> Run workflow** on `main`.

Enter the exact SemVer tag required by the repository `VERSION` file, for example:

~~~text
v0.1.0-rc.1
~~~

The workflow validates source/tests first, creates the tag only after validation, builds the multi-architecture images, and publishes the GitHub Release. Tags containing a SemVer pre-release suffix such as `-rc.1` are marked as GitHub pre-releases.

A normal external tag push remains supported and enters the same validated pipeline.

## Release artifacts

The tag workflow builds:

- Gateway and Broker container images for `linux/amd64` and `linux/arm64`;
- a Docker Compose package bundle;
- SHA-256 checksum material;
- GitHub Release notes.

Security CI separately produces vulnerability reports and CycloneDX SBOM evidence during acceptance.

The normal updateable installation remains a tagged Git checkout because `scripts/update.sh` intentionally uses Git fast-forward semantics, migration validation, backup and rollback.

## Update behavior

By default, `scripts/update.sh` selects the newest stable SemVer tag available from `origin`.

A release candidate or explicit version can be selected deliberately:

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.2 bash scripts/update.sh
~~~

The updater refuses non-fast-forward targets and retains its backup after success or rollback.
