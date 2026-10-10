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

## Publishing a release (explicit owner-only gate)

**A Git tag push alone never publishes a release.** Publishing is a separate,
owner-initiated action and remains blocked until the documented legal, security,
conformance and client acceptance gates are actually passed.

The repository owner initiates publication in GitHub Actions → **release** →
**Run workflow**, selecting the reviewed **main** branch.

For a click-by-click explanation in Portuguese for a beginner, see
[autorização de publicação PT-BR](release-owner-authorization.pt-BR.md). The form requires:

1. **tag**: the exact SemVer tag matching `VERSION`, e.g. `v0.1.0-rc.7`;
2. **approved_sha**: the full 40-character main commit SHA already reviewed,
   with complete CI and release authorization;
3. **publish_confirmation**: type `PUBLICAR v0.1.0-rc.7` with the **actual tag**.

Only the repository owner's GitHub account may initiate **or rerun** this
publication workflow. A trigger from any other actor or branch is rejected.
The selected commit must equal both `approved_sha` and the **current tip of
main**, preventing the publication of stale or substituted builds. The tag is
only created after source, security and multi-architecture image preflight
checks pass.

The confirmation is **not** evidence that license/IP rights, MCP requirements,
physical client acceptance or signatures have passed. The owner must review
those separate objective records before invoking the workflow; see
[release integration gates](release-integration-gates.md) and
[operator acceptance](operator-acceptance.md).

A publish action creates GHCR image tags, keyless signatures and a GitHub
Release. It is **not** an installation/deployment authorization. Do not test
this workflow by publishing an unapproved version.

## Release artifacts

The publication workflow emits:

- Gateway, Broker and public connector images in GHCR for
  `linux/amd64` and `linux/arm64`, each signed at its immutable image digest;
- BuildKit SBOM/provenance attestations associated with the pushed images;
- a source `tar.gz` bundle containing `VERSION`, its SHA-256 checksum
  and separate Sigstore bundles for **both** files;
- `RELEASE-INTEGRITY.txt` and GitHub Release notes.

**No separate Docker Compose package download** is created by the current
release workflow. Compose definitions are included in the signed source tree.
The normal updateable installation remains a tagged Git checkout because
`scripts/update.sh` uses Git fast-forward checks, migration validation,
backups and rollback.

## Isolated keyless-signing proof (no release)

The `release-signing-proof` workflow is a one-off, non-release validation that runs when its definition reaches `main`, or when explicitly dispatched from `main`. It signs a **generated synthetic text file only**, using a short-lived GitHub Actions OIDC certificate and Sigstore/cosign. The job verifies the exact repository/workflow identity and issuer, and confirms that an altered file fails signature verification. A short-retention artifact contains only the synthetic text, SHA-256 and public signing bundle. This proof does **not** create a Git tag, GitHub Release, GHCR image, billing event, production deployment or secret.

The published `release.yml` remains the authoritative, separate path for release images, source packages and their signatures. A green signing-proof run establishes that GitHub keyless blob signing works in the rehearsal environment, **not** that a released package or image has been signed or that the project is approved for stable publication. Final owner clean-install, Community license/IP approval and actual immutable release signature verification are still mandatory.

## Cryptographic release verification

Use the two independent, fail-closed helpers against the **exact published
version**. Download all source assets from the matching GitHub Release into one
local directory, then run:

~~~bash
python3 scripts/verify-release-source.py --directory /path/to/assets --tag v0.1.0-rc.7
python3 scripts/verify-release-images.py --tag v0.1.0-rc.7
~~~

The first helper checks the source archive, signed checksum, both Sigstore
bundles, expected GitHub Actions OIDC release identity and packaged `VERSION`.
The second resolves and pins three GHCR image digests, then checks cosign
signatures of each against the specific release identity and issuer. These
helpers require the appropriate `cosign` and `crane` installations; absence
of either required tool must fail closed.

**A successful synthetic signing proof or mocked-verifier test does not
authenticate a real release.** Record verified SHA, tag, source digest, image
digests and signing identities against the authorized release.

Historical RC artifacts published before the signed workflow retain their
original unsigned state. Documentation cannot retroactively sign them.

## Update behavior

By default, `scripts/update.sh` selects the newest stable SemVer tag available from `origin`.

A release candidate or explicit version can be selected deliberately:

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.7 bash scripts/update.sh
~~~

The updater refuses non-fast-forward targets and retains its backup after success or rollback.
