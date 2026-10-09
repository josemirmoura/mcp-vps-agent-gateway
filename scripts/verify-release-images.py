#!/usr/bin/env python3
"""Verify ALL three Portico Community release image signatures by immutable digest.

Read-only: resolves each GHCR release tag once, pins the returned digest, and
uses cosign's certificate identity and issuer verification. Does not deploy or
pull an image. Requires a trusted local crane and cosign installation.
"""
from __future__ import annotations

import argparse
import re
import shutil
import subprocess
import sys

IMAGES = ("mcp-vps-agent-gateway", "mcp-vps-agent-broker", "portico-cloud-node")
REPOSITORY = "ghcr.io/josemirmoura"
IDENTITY_PREFIX = (
    r"^https://github\.com/josemirmoura/mcp-vps-agent-gateway/"
    r"\.github/workflows/release\.yml@refs/"
)
ISSUER = "https://token.actions.githubusercontent.com"
TAG_RE = re.compile(r"v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?\Z")
DIGEST_RE = re.compile(r"sha256:[0-9a-f]{64}\Z")


class ImageVerificationFailure(RuntimeError):
    """A required image could not be authenticated."""


def identity_for_tag(tag: str) -> str:
    # GitHub Actions workflow_dispatch signs with refs/heads/main.
    # A tag-triggered release may sign with refs/tags/<exact release tag>.
    return IDENTITY_PREFIX + r"(heads/main|tags/" + re.escape(tag) + r")$"


def run(command: list[str], label: str, timeout: int) -> str:
    try:
        finished = subprocess.run(
            command, capture_output=True, text=True, timeout=timeout,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise ImageVerificationFailure(f"{label}: unavailable or timed out") from exc
    if finished.returncode:
        # Avoid printing proxy credentials, auth URLs or registry tokens from
        # subprocess stderr. The operator can diagnose the tools separately.
        raise ImageVerificationFailure(f"{label}: external verification failed")
    return finished.stdout.strip()


def verify_release_images(tag: str) -> list[str]:
    if not TAG_RE.fullmatch(tag):
        raise ImageVerificationFailure("release tag must use explicit SemVer format")
    crane = shutil.which("crane")
    cosign = shutil.which("cosign")
    if not crane or not cosign:
        raise ImageVerificationFailure("both crane and cosign are required")
    identity = identity_for_tag(tag)
    refs = []
    for image in IMAGES:
        tagged = f"{REPOSITORY}/{image}:{tag}"
        digest = run([crane, "digest", tagged], f"digest lookup: {image}", 90)
        if not DIGEST_RE.fullmatch(digest):
            raise ImageVerificationFailure(f"invalid immutable digest for {image}")
        pinned = f"{REPOSITORY}/{image}@{digest}"
        run([
            cosign, "verify", pinned,
            "--certificate-identity-regexp", identity,
            "--certificate-oidc-issuer", ISSUER,
        ], f"Sigstore signature: {image}", 120)
        refs.append(pinned)
    return refs


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", required=True,
                        help="published release tag, e.g. v0.1.0-rc.7")
    args = parser.parse_args()
    try:
        refs = verify_release_images(args.tag)
    except ImageVerificationFailure as exc:
        print(f"RELEASE IMAGE VERIFICATION FAILED: {exc}", file=sys.stderr)
        return 1
    print(f"RELEASE IMAGES VERIFIED: {args.tag}")
    for reference in refs:
        print(reference)
    print("These digests, not the mutable tags, identify the verified images.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
