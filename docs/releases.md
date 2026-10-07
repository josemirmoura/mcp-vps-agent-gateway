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

The current acceptance candidate identifies itself as `0.1.0-rc.7`. RC7 carries forward the validated RC6 runtime line, adds compact native approval UX for narrow/mobile ChatGPT surfaces, standardizes operator/user terminology, and is the first candidate intended to exercise the keyless Sigstore release-signing path end to end. Earlier RC tags are retained as immutable history.

The stable `v0.1.0` tag is created only after the owner's final clean-install operator acceptance gate succeeds.

## Publishing a release

The preferred owner path is **GitHub Actions → release → Run workflow** on `main`.

Enter the exact SemVer tag required by `VERSION`, for example:

~~~text
v0.1.0-rc.7
~~~

The workflow validates source, tests and the installation contract first. Only after validation does a manual run create the exact tag, build/publish the multi-architecture images and create the GitHub Release. Existing external tag pushes remain supported and enter the same validated pipeline. Pre-release SemVer tags such as `-rc.7` are published as GitHub pre-releases.

## Release artifacts

The tag workflow builds:

- Gateway and Broker container images for `linux/amd64` and `linux/arm64`;
- a Docker Compose package bundle;
- SHA-256 checksum material;
- GitHub Release notes.

Security CI separately produces vulnerability reports and CycloneDX SBOM evidence during acceptance.

The normal updateable installation remains a tagged Git checkout because `scripts/update.sh` intentionally uses Git fast-forward semantics, migration validation, backup and rollback.

## Cryptographic release verification

The current release workflow is configured to keyless-sign new release images and source artifacts with Sigstore/cosign using GitHub Actions OIDC.

For source artifacts, each GitHub Release includes a Sigstore bundle beside the artifact. Verify with the expected workflow identity:

~~~bash
cosign verify-blob mcp-vps-agent-source-package.tar.gz \
  --bundle mcp-vps-agent-source-package.tar.gz.sigstore.json \
  --certificate-identity-regexp '^https://github\.com/josemirmoura/mcp-vps-agent-gateway/\.github/workflows/release\.yml@refs/(heads/main|tags/v.*)$' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com'
~~~

Release images are signed by immutable digest. Example:

~~~bash
cosign verify ghcr.io/josemirmoura/mcp-vps-agent-gateway@sha256:<digest> \
  --certificate-identity-regexp '^https://github\.com/josemirmoura/mcp-vps-agent-gateway/\.github/workflows/release\.yml@refs/(heads/main|tags/v.*)$' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com'
~~~

This signing model avoids a long-lived project private key in GitHub Secrets. Trust is anchored in Sigstore plus the repository/workflow identity.

Already-published release candidates created before this control remain historical unsigned artifacts. Do not infer a signature retroactively from documentation.

## Update behavior

By default, `scripts/update.sh` selects the newest stable SemVer tag available from `origin`.

A release candidate or explicit version can be selected deliberately:

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.7 bash scripts/update.sh
~~~

The updater refuses non-fast-forward targets and retains its backup after success or rollback.
