# Recovery gate for a signed Community candidate (Linux Scoped)

This is a **pre-release operations runbook**, not permission to deploy to a real node. This document does not assert that a particular RC or signed artifact currently exists. Only the owner/release integrator may authorize installation, production update, tag publication or destructive rollback.

## 1. Authenticate what would be installed

On a trusted disposable validation machine, obtain the *exact published tag* from the official GitHub Releases page. Do not select a moving `main`, guessed tag, or a commit merely because CI is green.

Before any lifecycle test, verify both artifact classes for the SAME explicit tag:

```sh
# Set only after confirming a published candidate in GitHub Releases.
: "${RELEASE_TAG:?first export the exact published, signed RC tag}"
mkdir -p "./release-assets/$RELEASE_TAG"
gh release download "$RELEASE_TAG" \
  --repo josemirmoura/mcp-vps-agent-gateway \
  --dir "./release-assets/$RELEASE_TAG" \
  --pattern 'mcp-vps-agent-source-package.tar.gz*'

python3 scripts/verify-release-source.py \
  --directory "./release-assets/$RELEASE_TAG" --tag "$RELEASE_TAG"
python3 scripts/verify-release-images.py --tag "$RELEASE_TAG"
```

**Stop** if the source signature, signed checksum, internal VERSION, any of the three image signatures, OIDC signer identity or published tag check fails. A mock `cosign`/mock `crane` CI result is **not** cryptographic acceptance. Preserve and record verified digest references; do not substitute mutable tags for the verified digests.

## 1A. Mandatory complete-publication preflight (AUD-01/AUD-02)

The updated `scripts/update.sh` accepts only an explicit published SemVer release tag, including operator-selected release candidates. Automatic selection of a local stable tag must pass exactly the same checks. A tag in Git alone, `main`, an arbitrary SHA, a draft release or an unsigned legacy release is not an update channel. The updater needs `git`, Python 3, GitHub CLI (`gh`), `cosign` and `crane` installed from trustworthy sources, and network access to GitHub/GHCR/Sigstore. An absent verifier/dependency or network timeout blocks the update without stopping containers. Never disable signature verification to bypass an outage.

Before touching Compose, an operational backup or Git checkout, the updater verifies:

1. The local tag and freshly queried remote tag both resolve to the exact candidate commit SHA.
2. GitHub Release for the tag is published, not draft, and lists all **six required assets**: source tarball, its SHA-256, their two Sigstore bundles, `RELEASE-PROVENANCE.json` and its Sigstore bundle. Files are downloaded to a private temporary directory and size-bounded.
3. Sigstore verifies the source, signed checksum and signed provenance with the expected release-workflow identity and GitHub OIDC issuer. Signed provenance includes schema, exact release tag, 40-character SHA, and the package SHA-256.
4. The full signed source TAR tree is checked against a locally reproduced `git archive` of that **exact commit**, comparing member paths, types, modes, link targets and content hashes. Equal `VERSION` alone never satisfies this check.
5. All three image signatures are independently checked by immutable GHCR digest. The remote tag is read again to detect retargeting during the preflight.

Only after this passes may the updater create an operational backup or stop a container. The GitHub Actions publication workflow likewise creates the remotely visible tag only in its final job, after the image build/signing and source/provenance signing. There is still a small publication window before the Release is complete; the updater explicitly rejects it.

### Interrupted publication and recovery

- If the publishing job fails after images were signed or a tag appeared, treat the candidate as **incomplete**. Keep installed services running and inspect the release job, published asset list and provenance at the same SHA; do not select that tag manually to bypass preflight.
- Never retarget a published release tag, substitute another package with identical `VERSION`, or invent missing Sigstore bundles. Investigate partial uploads and restore a coherent, owner-authorized publication of the *same* reviewed commit through the release workflow. If immutability cannot be established, publish a **new version/tag** after fresh review and all release gates.
- If signature services or GitHub/GHCR cannot be reached, retry verification later. There is no silent offline fallback. A failed preflight requires no runtime rollback because runtime, checkout and operational snapshot have not been mutated.
- An older installer/updater without this guard needs migration through the explicitly reviewed installation path first. Do not claim unsigned historical RCs become trusted simply because this verifier is present.

These guards are tested with simulated CLI and signatures; acceptance of a **real** release still requires separately verifying genuine source/image signatures and the owner-controlled deployment. This runbook grants no permission to publish or modify the owner's VPS.

## 2. Disposable validation, never owner production

- Install candidate on a newly provisioned Linux test host with an isolated Scoped root and independent fake test credentials.
- Record exact `git rev-parse HEAD`, `git describe --tags --exact-match`, effective Compose file list, Docker volume names, SHA-256 fingerprints of backups, and local health/audit reports. Never print tokens or `.env`.
- Exercise normal update, interrupted `stop`, backup failure, tar corruption, schema migration failure, modified policy, changed credential, Docker volume restoration failure and successful rollback, confirming `system.info`, policy and `audit-status`.
- Confirm returned state does NOT resurrect root delegation, temporary grant or pending approval: rollback must invalidate all elevated authority in an offline staging copy, append audit event and restart the old Broker only after that.
- Confirm immutable owner-created backups remain private (0700 directory, restricted files), recoverable and retained until acceptance. Do not copy snapshots to chat, public issues or Git.
- A two-volume Zitadel rollback is **not** atomic across volumes. Verify real identity login and database consistency after recovery. A synthetic Alpine volume test cannot substitute for this case.

## 3. Fail-closed recovery if a real update goes wrong

Use only the approved `scripts/update.sh` from the verified candidate/branch, with its explicitly selected release ref and the exact Compose overlays previously validated. Keep the old image/digest and signed snapshot available. A full operator approval portal overlay may not be auto-discovered by older updater versions: have Block 1/5 approve the effective Compose list before attempting a live update.

If the updater reports `ROLLBACK BLOCKED` or `ROLLBACK FAILED`:

1. **Stop automated retries.** Never run `rm -rf state`, `docker compose down -v`, `docker volume prune` or overwrite a mismatched `.env` or policy with stale copies.
2. Preserve the snapshot directory and `failed-target-state` from the attempted rollback, plus Docker volume staging/previous entries. Restrict access; they may contain administrator tokens.
3. Check which commit and Compose manifests remain active, then arrange a recovery in a *separate* test copy of the snapshots. Do not claim restored service until its Broker audit chain, scoped permissions, OAuth and operator grants are verified.
4. If policy or credentials changed since backup, reconcile those changes explicitly. No automatic reactivation of prior elevated grants or older access rules.
5. If one of the identity volumes failed to restore, consider the pair inconsistent until verified with a real Zitadel/PostgreSQL smoke. Do not delete remaining volume copies.
6. Escalate production-only intervention to the owner only with a precise SHA, failure log stripped of secrets, expected impact and rollback plan.

## 4. Gate to hand over to Block 5

Record four independent states:

- **TESTED:** isolated Python and Docker/Broker failure-path suites green on exact candidate SHA.
- **SIGNED:** real downloaded source and image artifacts authenticated against the publish workflow and tag. No mock inference.
- **ACCEPTED:** operator verifies real desktop/mobile compatible client and Scoped policy with authorized/denied requests.
- **PUBLISHED:** immutable RC and later stable release published after licensing/IP and owner decisions.

Do not equate any two of these states. Only Block 5 integrates into the release branch or `main`.
