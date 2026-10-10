#!/usr/bin/env python3
"""Fail-closed preflight for a fully published, signed, commit-bound update.

Reads GitHub release metadata and downloads assets into a private TEMP directory.
Does not mutate the checkout, Docker, state, or the installed service.
"""
from __future__ import annotations

import argparse
import hashlib
import importlib.util
import io
import json
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / "scripts/verify-release-source.py"
REPOSITORY = "josemirmoura/mcp-vps-agent-gateway"
TAG_PATTERN = re.compile(r"v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?\Z")
SHA_PATTERN = re.compile(r"[0-9a-f]{40}\Z")
PACKAGE = "mcp-vps-agent-source-package.tar.gz"
PROVENANCE = "RELEASE-PROVENANCE.json"
REQUIRED = (
    PACKAGE, PACKAGE + ".sha256", PACKAGE + ".sigstore.json",
    PACKAGE + ".sha256.sigstore.json", PROVENANCE, PROVENANCE + ".sigstore.json",
)


class Blocked(RuntimeError):
    pass


def execute(argv: list[str], *, cwd: Path, timeout: int = 90) -> str:
    try:
        proc = subprocess.run(argv, cwd=cwd, capture_output=True, text=True,
                              check=False, timeout=timeout)
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise Blocked("required command unavailable or timed out: " + argv[0]) from exc
    if proc.returncode:
        # Do not echo stderr: external commands can expose credentials.
        raise Blocked("command failed: " + Path(argv[0]).name + " " + argv[1])
    return proc.stdout.strip()


def sha256(path: Path) -> str:
    with path.open("rb") as handle:
        return hashlib.file_digest(handle, "sha256").hexdigest()


def remote_commit(repo: Path, tag: str) -> str:
    output = execute(["git", "ls-remote", "--tags", "origin",
                      "refs/tags/" + tag, "refs/tags/" + tag + "^{}"], cwd=repo)
    entries = {}
    for line in output.splitlines():
        parts = line.split("\t")
        if len(parts) != 2 or not SHA_PATTERN.fullmatch(parts[0]):
            raise Blocked("malformed remote tag response")
        entries[parts[1]] = parts[0]
    direct = entries.get("refs/tags/" + tag)
    peeled = entries.get("refs/tags/" + tag + "^{}")
    if not direct or any(k not in ("refs/tags/" + tag, "refs/tags/" + tag + "^{}")
                         for k in entries):
        raise Blocked("tag missing on remote or invalid")
    return peeled or direct


def release_assets(gh: str, root: Path, tag: str, destination: Path) -> None:
    response = execute([gh, "release", "view", tag, "--repo", REPOSITORY,
                        "--json", "tagName,isDraft,assets,publishedAt"], cwd=root)
    try:
        record = json.loads(response)
        if (record["tagName"] != tag or record["isDraft"] is not False
                or not record["publishedAt"]):
            raise Blocked("release missing, draft or not published")
        names = {}
        for item in record["assets"]:
            name = item["name"]
            if name in names:
                raise Blocked("duplicate published release asset")
            names[name] = item["size"]
        if any(name not in names or not isinstance(names[name], int) or
               names[name] <= 0 for name in REQUIRED):
            raise Blocked("published release assets missing or incomplete")
    except (ValueError, KeyError, TypeError) as exc:
        raise Blocked("invalid GitHub release metadata") from exc

    execute([gh, "release", "download", tag, "--repo", REPOSITORY,
             "--dir", str(destination), "--pattern", PACKAGE + "*",
             "--pattern", PROVENANCE + "*"], cwd=root, timeout=180)
    limits = {
        PACKAGE: 300 * 1024 * 1024,
        PACKAGE + ".sha256": 512,
        PACKAGE + ".sigstore.json": 3 * 1024 * 1024,
        PACKAGE + ".sha256.sigstore.json": 3 * 1024 * 1024,
        PROVENANCE: 4096,
        PROVENANCE + ".sigstore.json": 3 * 1024 * 1024,
    }
    for name in REQUIRED:
        item = destination / name
        if item.is_symlink() or not item.is_file():
            raise Blocked("missing or symlinked downloaded asset: " + name)
        if not (0 < item.stat().st_size <= limits[name]):
            raise Blocked("invalid downloaded release asset size: " + name)
        if item.stat().st_size != names[name]:
            raise Blocked("published asset size differs from downloaded bytes: " + name)


def members(path: Path, mode: str) -> dict[str, tuple]:
    """Compare actual Git trees, not just VERSION or gzip timestamps."""
    result = {}
    total = 0
    try:
        with tarfile.open(path, mode=mode) as stream:
            for item in stream:
                name = item.name
                if (name in result or len(result) >= 100_000
                        or (name != "mcp-vps-agent" and
                            (not name.startswith("mcp-vps-agent/") or
                             any(part in ("", ".", "..") for part in name.split("/"))))):
                    raise Blocked("invalid, duplicate or escaping source member")
                if item.isfile():
                    total += item.size
                    if total > 1024 * 1024 * 1024:
                        raise Blocked("source members exceed uncompressed size limit")
                    data = stream.extractfile(item)
                    if data is None:
                        raise Blocked("source member cannot be read")
                    digest = hashlib.sha256()
                    while chunk := data.read(1024 * 1024):
                        digest.update(chunk)
                    kind = ("file", item.size, digest.hexdigest())
                elif item.isdir():
                    kind = ("dir",)
                elif item.issym():
                    kind = ("symlink", item.linkname)
                elif item.islnk():
                    kind = ("hardlink", item.linkname)
                else:
                    raise Blocked("unsupported source member type")
                result[name] = (kind, item.mode & 0o777)
            # git archive <commit> writes the *commit ID*, not only its tree,
            # into the global PAX comment. Comparing it rejects signed
            # same-tree packages generated from a different commit.
            provenance = stream.pax_headers.get("comment", "")
            if not SHA_PATTERN.fullmatch(provenance):
                raise Blocked("source tar lacks Git commit provenance")
            result["\\0git-archive-commit"] = (provenance,)
    except (tarfile.TarError, OSError) as exc:
        raise Blocked("source tar failed integrity inspection") from exc
    return result


def verify(tag: str, commit: str, repo: Path, offline_assets: Path | None = None) -> None:
    if not TAG_PATTERN.fullmatch(tag) or not SHA_PATTERN.fullmatch(commit):
        raise Blocked("only explicit SemVer tags and exact 40-character commit SHAs are accepted")
    if not (repo / ".git").exists():
        raise Blocked("update checkout is not a Git repository")
    remote = remote_commit(repo, tag)
    local = execute(["git", "rev-parse", "refs/tags/" + tag + "^{commit}"], cwd=repo)
    if remote != commit or local != commit:
        raise Blocked("published remote tag, local tag and target commit differ")

    spec = importlib.util.spec_from_file_location("portico_source_release", SOURCE)
    if spec is None or spec.loader is None:
        raise Blocked("source verification library unavailable")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)

    gh = shutil.which("gh")
    cosign = shutil.which("cosign")
    crane = shutil.which("crane")
    if not all((gh, cosign, crane)):
        raise Blocked("updates require gh, cosign and crane; no unsigned fallback")

    with tempfile.TemporaryDirectory(prefix="portico-update-preflight-") as tmp:
        directory = Path(tmp)
        if offline_assets is None:
            release_assets(gh, repo, tag, directory)
        else:
            # Intended solely for isolated test fixtures. Not exposed in update.sh.
            directory = offline_assets
        try:
            module.verify(directory, tag)
            module.verify_sigstore(
                cosign, directory / PROVENANCE, directory / (PROVENANCE + ".sigstore.json"),
                module.identity_for_tag(tag))
        except module.VerificationFailure as exc:
            raise Blocked("source signature or signed checksums rejected") from exc
        try:
            record = json.loads((directory / PROVENANCE).read_bytes())
            if set(record) != {"schema", "tag", "commit_sha", "source_sha256"}:
                raise Blocked("invalid provenance field set")
            if (type(record["schema"]) is not int or record["schema"] != 1
                    or record["tag"] != tag or record["commit_sha"] != commit
                    or record["source_sha256"] != sha256(directory / PACKAGE)):
                raise Blocked("signed provenance differs from tag, commit or package")
        except (OSError, ValueError, TypeError) as exc:
            raise Blocked("invalid signed provenance JSON") from exc

        with tempfile.TemporaryFile() as canonical:
            try:
                subprocess.run(
                    ["git", "archive", "--format=tar", "--prefix=mcp-vps-agent/", commit],
                    cwd=repo, stdout=canonical, stderr=subprocess.DEVNULL,
                    check=True, timeout=120)
            except (OSError, subprocess.CalledProcessError,
                    subprocess.TimeoutExpired) as exc:
                raise Blocked("cannot reproduce source tree at signed commit") from exc
            canonical.seek(0)
            # Python tarfile accepts a seekable file. Compare normalized tree contents,
            # file modes and link targets rather than gzip encoding or tar timestamps.
            with tempfile.NamedTemporaryFile() as canonical_file:
                shutil.copyfileobj(canonical, canonical_file)
                canonical_file.flush()
                expected = members(Path(canonical_file.name), "r:")
            if members(directory / PACKAGE, "r:gz") != expected:
                raise Blocked("signed source package differs from exact remote tag commit")
        execute([sys.executable, str(ROOT / "scripts/verify-release-images.py"),
                 "--tag", tag], cwd=repo, timeout=500)
        # Check tag once more to reject publication moved during verification.
        if remote_commit(repo, tag) != commit:
            raise Blocked("remote tag changed during release preflight")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--commit", required=True)
    parser.add_argument("--repo", type=Path, default=Path.cwd())
    args = parser.parse_args()
    try:
        verify(args.tag, args.commit, args.repo.resolve())
    except Blocked as exc:
        print("UPDATE BLOCKED: " + str(exc), file=sys.stderr)
        return 1
    print("RELEASE PREFLIGHT VERIFIED: signed source, exact commit, publication and images")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
