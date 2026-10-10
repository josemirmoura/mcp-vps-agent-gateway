#!/usr/bin/env python3
"""Authenticate a COMPLETE published Pórtico release before any updater stop.

GitHub Release metadata, remote tag and signed provenance MUST agree on the
exact commit; the source archive MUST equal a reproducible git archive of that
commit. The three image tags must resolve to cosign-verified immutable digests.
Only external tools with fixed arguments are invoked; this module never edits
the checkout, starts/stops Docker or changes operator state.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile

REPO = "josemirmoura/mcp-vps-agent-gateway"
TAG_RE = re.compile(r"v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?\Z")
SHA_RE = re.compile(r"[0-9a-f]{40}\Z")
HASH_RE = re.compile(r"[0-9a-f]{64}\Z")
ARCHIVE = "mcp-vps-agent-source-package.tar.gz"
CHECKSUM = ARCHIVE + ".sha256"
PROVENANCE = "RELEASE-PROVENANCE.json"
SIGNED = (ARCHIVE, CHECKSUM, PROVENANCE)
REQUIRED = tuple(
    item for name in SIGNED for item in (name, name + ".sigstore.json")
)
LIMITS = {ARCHIVE: 300 * 1024 * 1024, CHECKSUM: 512,
          PROVENANCE: 4096}
for name in SIGNED:
    LIMITS[name + ".sigstore.json"] = 3 * 1024 * 1024
ISSUER = "https://token.actions.githubusercontent.com"
IDENTITY = (
    r"^https://github\.com/josemirmoura/mcp-vps-agent-gateway/"
    r"\.github/workflows/release\.yml@refs/heads/main$"
)


class UnsafeRelease(RuntimeError):
    pass


def command(argv: list[str], *, timeout: int = 90) -> str:
    try:
        result = subprocess.run(argv, capture_output=True, text=True,
                                timeout=timeout, check=False)
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise UnsafeRelease("required verification tool unavailable or timed out") from exc
    if result.returncode:
        # External stderr may contain registry credentials or URLs. Never echo.
        raise UnsafeRelease("remote release or cryptographic verification failed")
    return result.stdout.strip()


def remote_tag(tag: str) -> str:
    data = command(["git", "ls-remote", "--tags", "origin",
                    f"refs/tags/{tag}", f"refs/tags/{tag}^{{}}"])
    refs = {}
    for line in data.splitlines():
        fields = line.split("\t")
        if len(fields) != 2 or not SHA_RE.fullmatch(fields[0]):
            raise UnsafeRelease("invalid remote tag response")
        refs[fields[1]] = fields[0]
    ref = f"refs/tags/{tag}"
    if ref not in refs:
        raise UnsafeRelease("release tag missing on origin")
    sha = refs.get(ref + "^{}", refs[ref])
    local = command(["git", "rev-parse", "--verify", f"{ref}^{{commit}}"])
    if sha != local:
        raise UnsafeRelease("local tag is different from the remote tag")
    return sha


def published_release(tag: str, expected_sha: str) -> None:
    raw = command(["gh", "api", f"repos/{REPO}/releases/tags/{tag}"])
    try:
        release = json.loads(raw)
        assets = release["assets"]
        published = (
            release.get("tag_name") == tag
            and release.get("draft") is False
            and bool(release.get("published_at"))
            and release.get("target_commitish") == expected_sha
            and isinstance(assets, list)
        )
        if not published:
            raise UnsafeRelease("release is draft, unpublished, or points to another commit")
        known = {}
        for item in assets:
            name = item["name"]
            if name in known:
                raise UnsafeRelease("duplicate GitHub Release asset")
            known[name] = item
        for name in REQUIRED:
            item = known.get(name)
            if (item is None or item.get("state") != "uploaded"
                    or not isinstance(item.get("size"), int)
                    or not 0 < item["size"] <= LIMITS[name]):
                raise UnsafeRelease(f"missing, incomplete or oversized signed asset: {name}")
    except (KeyError, TypeError, ValueError) as exc:
        raise UnsafeRelease("malformed GitHub Release metadata") from exc


def trusted_asset(root: Path, name: str) -> Path:
    p = root / name
    if (p.is_symlink() or not p.is_file()
            or not 0 < p.stat().st_size <= LIMITS[name]):
        raise UnsafeRelease(f"missing or invalid downloaded asset: {name}")
    return p


def signed_provenance(root: Path, tag: str, sha: str) -> str:
    provenance = trusted_asset(root, PROVENANCE)
    bundle = trusted_asset(root, PROVENANCE + ".sigstore.json")
    command(["cosign", "verify-blob", str(provenance),
             "--bundle", str(bundle),
             "--certificate-identity-regexp", IDENTITY,
             "--certificate-oidc-issuer", ISSUER], timeout=120)
    try:
        data = json.loads(provenance.read_text("utf-8"))
    except (UnicodeError, OSError, ValueError) as exc:
        raise UnsafeRelease("invalid signed provenance JSON") from exc
    if (type(data) is not dict
            or set(data) != {"schema_version", "repository", "tag",
                             "commit_sha", "source_sha256"}
            or type(data["schema_version"]) is not int
            or data["schema_version"] != 1
            or data["repository"] != REPO
            or data["tag"] != tag
            or data["commit_sha"] != sha
            or not isinstance(data["source_sha256"], str)
            or not HASH_RE.fullmatch(data["source_sha256"])):
        raise UnsafeRelease("signed provenance is missing or does not bind the exact commit")
    return data["source_sha256"]


def reproducible_archive_hash(sha: str) -> str:
    # gzip -n matches release.yml and omits timestamp/name headers. pipefail
    # ensures git archive failures cannot be silently accepted as empty gzip.
    script = 'set -euo pipefail; git archive --format=tar --prefix=mcp-vps-agent/ "$1" | gzip -n | sha256sum'
    line = command(["bash", "-c", script, "portico-verify", sha], timeout=195)
    result = line.split()
    if len(result) < 1 or not HASH_RE.fullmatch(result[0]):
        raise UnsafeRelease("failed to produce deterministic source archive digest")
    return result[0]


def verify(tag: str, expected_sha: str) -> None:
    if not TAG_RE.fullmatch(tag) or not SHA_RE.fullmatch(expected_sha):
        raise UnsafeRelease("only a published SemVer tag and exact commit SHA are allowed")
    for executable in ("gh", "cosign", "crane", "git", "gzip", "sha256sum", "bash"):
        if not shutil.which(executable):
            raise UnsafeRelease(f"required verification tool missing: {executable}")
    if remote_tag(tag) != expected_sha:
        raise UnsafeRelease("remote release tag points to a different commit")
    published_release(tag, expected_sha)
    with tempfile.TemporaryDirectory(prefix="portico-update-release-") as temp:
        root = Path(temp)
        command(["gh", "release", "download", tag, "--repo", REPO,
                 "--dir", temp, "--clobber"], timeout=240)
        for name in REQUIRED:
            trusted_asset(root, name)
        source_hash = signed_provenance(root, tag, expected_sha)
        command([sys.executable, "scripts/verify-release-source.py",
                 "--directory", temp, "--tag", tag], timeout=250)
        with trusted_asset(root, ARCHIVE).open("rb") as stream:
            actual = hashlib.file_digest(stream, "sha256").hexdigest()
        if actual != source_hash:
            raise UnsafeRelease("signed source does not match the provenance digest")
        # Stronger than matching VERSION: every tracked byte in the signed
        # archive must reproduce from exactly the remote tag's commit.
        if reproducible_archive_hash(expected_sha) != actual:
            raise UnsafeRelease("signed archive does not reproduce from remote tag commit")
        command([sys.executable, "scripts/verify-release-images.py",
                 "--tag", tag], timeout=490)
    # Reread remote release/tag after time-consuming checks, rejecting
    # TOCTOU changes before the updater creates any operational backup.
    if remote_tag(tag) != expected_sha:
        raise UnsafeRelease("release tag changed during preflight")
    published_release(tag, expected_sha)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--expected-sha", required=True)
    args = parser.parse_args()
    try:
        verify(args.tag, args.expected_sha)
    except UnsafeRelease as exc:
        print(f"UPDATE BLOCKED: unverified published release: {exc}", file=sys.stderr)
        return 1
    print(f"PUBLISHED RELEASE VERIFIED: {args.tag} @ {args.expected_sha}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
