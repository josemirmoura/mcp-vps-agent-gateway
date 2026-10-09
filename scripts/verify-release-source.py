#!/usr/bin/env python3
"""Fail-closed verification of published Portico source release assets.

This verifies the two Sigstore bundles, signed checksum, source bytes, and
VERSION inside the archive. It does not verify container image signatures,
Git tag immutability, or approve any deployment.
"""
from __future__ import annotations

import argparse
import hashlib
import re
import shutil
import subprocess
import sys
import tarfile
from pathlib import Path

ARCHIVE = "mcp-vps-agent-source-package.tar.gz"
CHECKSUM = ARCHIVE + ".sha256"
SUFFIX = ".sigstore.json"
ISSUER = "https://token.actions.githubusercontent.com"
IDENTITY = (
    r"^https://github\.com/josemirmoura/mcp-vps-agent-gateway/"
    r"\.github/workflows/release\.yml@refs/(heads/main|tags/v.*)$"
)
TAG_RE = re.compile(r"v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?\Z")
MAX_ARCHIVE_BYTES = 300 * 1024 * 1024
MAX_CHECKSUM_BYTES = 512
MAX_BUNDLE_BYTES = 3 * 1024 * 1024


class VerificationFailure(RuntimeError):
    """Release could not be authenticated to the configured trust root."""


def trusted_file(root: Path, name: str, maximum: int) -> Path:
    path = root / name
    if path.is_symlink() or not path.is_file():
        raise VerificationFailure(f"missing or symlinked release asset: {name}")
    if path.stat().st_size > maximum or path.stat().st_size == 0:
        raise VerificationFailure(f"invalid release asset size: {name}")
    return path


def signed_checksum(filename: Path) -> str:
    try:
        data = filename.read_bytes()
        line = data.decode("ascii").strip("\n")
    except (UnicodeError, OSError) as exc:
        raise VerificationFailure("cannot read ASCII checksum") from exc
    match = re.fullmatch(
        r"([0-9a-fA-F]{64})  (?:dist/)?" + re.escape(ARCHIVE),
        line,
    )
    if not match:
        raise VerificationFailure("checksum must contain exactly one expected archive name")
    return match.group(1).lower()


def verify_sigstore(cosign: str, artifact: Path, bundle: Path) -> None:
    command = [
        cosign, "verify-blob", str(artifact), "--bundle", str(bundle),
        "--certificate-identity-regexp", IDENTITY,
        "--certificate-oidc-issuer", ISSUER,
    ]
    try:
        completed = subprocess.run(command, capture_output=True, text=True,
                                   timeout=120, check=False)
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise VerificationFailure(f"Sigstore verification failed to execute: {artifact.name}") from exc
    if completed.returncode:
        raise VerificationFailure(f"Sigstore rejected artifact: {artifact.name}")


def archived_version(path: Path) -> str:
    try:
        # Streaming scan reads no extracted filesystem paths. Archive contents
        # are inspected only AFTER both signatures and SHA-256 have passed.
        with tarfile.open(path, mode="r|gz") as stream:
            for entry in stream:
                if entry.name == "mcp-vps-agent/VERSION":
                    if not entry.isfile() or entry.size < 1 or entry.size > 128:
                        raise VerificationFailure("invalid VERSION entry in source archive")
                    handle = stream.extractfile(entry)
                    if handle is None:
                        raise VerificationFailure("unreadable VERSION entry")
                    raw = handle.read(129).decode("ascii")
                    return raw.strip()
    except (tarfile.TarError, UnicodeError, OSError) as exc:
        raise VerificationFailure("invalid signed source archive") from exc
    raise VerificationFailure("source archive is missing mcp-vps-agent/VERSION")


def verify(directory: Path, expected_tag: str) -> None:
    if not TAG_RE.fullmatch(expected_tag):
        raise VerificationFailure("expected tag must be explicit vMAJOR.MINOR.PATCH[-prerelease]")
    if not directory.is_dir():
        raise VerificationFailure("release asset directory does not exist")
    directory = directory.resolve(strict=True)
    archive = trusted_file(directory, ARCHIVE, MAX_ARCHIVE_BYTES)
    checksum = trusted_file(directory, CHECKSUM, MAX_CHECKSUM_BYTES)
    archive_bundle = trusted_file(directory, ARCHIVE + SUFFIX, MAX_BUNDLE_BYTES)
    checksum_bundle = trusted_file(directory, CHECKSUM + SUFFIX, MAX_BUNDLE_BYTES)

    cosign = shutil.which("cosign")
    if not cosign:
        raise VerificationFailure("cosign is required; no unsigned fallback is allowed")
    # Verify the checksum material is signed before trusting its digest.
    verify_sigstore(cosign, checksum, checksum_bundle)
    expected_digest = signed_checksum(checksum)
    digest = hashlib.file_digest(archive.open("rb"), "sha256").hexdigest()
    if digest != expected_digest:
        raise VerificationFailure("source archive differs from the signed SHA-256 checksum")
    verify_sigstore(cosign, archive, archive_bundle)
    if archived_version(archive) != expected_tag.removeprefix("v"):
        raise VerificationFailure("signed source VERSION does not match the requested release tag")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--directory", required=True, type=Path,
                        help="directory containing four release source/signature assets")
    parser.add_argument("--tag", required=True,
                        help="expected GitHub release tag, e.g. v0.1.0-rc.7")
    args = parser.parse_args()
    try:
        verify(args.directory, args.tag)
    except VerificationFailure as exc:
        print(f"RELEASE SOURCE VERIFICATION FAILED: {exc}", file=sys.stderr)
        return 1
    print(f"RELEASE SOURCE VERIFIED: {args.tag} (two Sigstore bundles, SHA-256, packaged VERSION)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
