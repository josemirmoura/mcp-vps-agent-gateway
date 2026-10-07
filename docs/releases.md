# Release and version policy

Portico MCP follows Semantic Versioning.

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

The current acceptance candidate identifies itself as `0.1.0-rc.6`. RC6 carries forward the RC5 authority model and adds refreshed Debian runtime packages after the blocking image-security scan detected newly available fixes. Earlier RC tags are retained as immutable history.

The stable `v0.1.0` tag is created only after the owner's final clean-install operator acceptance gate succeeds.

## Publishing a release

The preferred owner path is **GitHub Actions → release → Run workflow** on `main`.

Enter the exact SemVer tag required by `VERSION`, for example:

~~~text
v0.1.0-rc.6
~~~

The workflow validates source, tests and the installation contract first. Only after validation does a manual run create the exact tag, build/publish the multi-architecture images and create the GitHub Release. Existing external tag pushes remain supported and enter the same validated pipeline. Pre-release SemVer tags such as `-rc.6` are published as GitHub pre-releases.

## Release artifacts

The tag workflow builds:

- Gateway and Broker container images for `linux/amd64` and `linux/arm64`;
- a Docker Compose package bundle;
- SHA-256 checksum material;
- GitHub Release notes.

Security CI separately produces vulnerability reports and CycloneDX SBOM evidence during acceptance.

The release workflow also creates **keyless signed provenance attestations** using GitHub OIDC and short-lived Sigstore certificates:

- each published GHCR Gateway/Broker image is attested by immutable image digest and the attestation is pushed alongside the OCI image;
- the source package receives a signed GitHub provenance attestation;
- the portable Sigstore bundle for the source package is attached to the GitHub Release;
- no long-lived project private signing key is stored.

After a release is published, verify the source package with a current GitHub CLI:

~~~bash
gh attestation verify mcp-vps-agent-source-package.tar.gz --owner josemirmoura
~~~

For a release image, use its immutable digest rather than only a mutable tag:

~~~bash
gh attestation verify \
  oci://ghcr.io/josemirmoura/mcp-vps-agent-gateway@sha256:<digest> \
  --owner josemirmoura \
  --bundle-from-oci
~~~

The first release that exercises this new workflow must itself pass verification before the project marks the signing/attestation gate complete.

The normal updateable installation remains a tagged Git checkout because `scripts/update.sh` intentionally uses Git fast-forward semantics, migration validation, backup and rollback.

## Update behavior

By default, `scripts/update.sh` selects the newest stable SemVer tag available from `origin`.

A release candidate or explicit version can be selected deliberately:

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.6 bash scripts/update.sh
~~~

The updater refuses non-fast-forward targets and retains its backup after success or rollback.
