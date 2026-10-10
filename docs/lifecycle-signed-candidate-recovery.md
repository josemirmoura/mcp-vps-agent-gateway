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
