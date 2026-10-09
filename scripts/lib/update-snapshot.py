#!/usr/bin/env python3
"""Validate and stage a trusted operator-state snapshot before rollback mutation.

Fail closed: no extraction of symlinks, hardlinks, special files, absolute
paths, path traversal, duplicate entries, or files outside the known layout.
"""
from __future__ import annotations

import argparse
import os
import pathlib
import tarfile

MAX_ENTRIES = 100_000
MAX_EXTRACTED_BYTES = 10 * 1024 * 1024 * 1024
REQUIRED = {".env", "config/policy.yaml", "state/state.db"}


def stage_snapshot(archive: pathlib.Path, destination: pathlib.Path) -> None:
    if destination.exists():
        raise ValueError("rollback staging directory must be new")
    destination.mkdir(mode=0o700, parents=False)
    seen: set[str] = set()
    size = 0
    with tarfile.open(archive, "r:gz") as pack:
        members = pack.getmembers()
        if len(members) > MAX_ENTRIES:
            raise ValueError("too many operator snapshot entries")
        for member in members:
            name = member.name
            normalized = name.rstrip("/")
            segments = normalized.split("/")
            if (not normalized or name.startswith("/") or
                    any(seg in ("", ".", "..") for seg in segments) or
                    chr(92) in name):
                raise ValueError("unsafe operator snapshot path")
            if (normalized not in (".env", "config", "config/policy.yaml", "state")
                    and not normalized.startswith("state/")):
                raise ValueError("unexpected operator snapshot member")
            if normalized in seen:
                raise ValueError("duplicate operator snapshot member")
            seen.add(normalized)
            if not (member.isfile() or member.isdir()):
                raise ValueError("snapshot links or special files are forbidden")
            if member.isfile():
                size += member.size
                if size > MAX_EXTRACTED_BYTES:
                    raise ValueError("operator snapshot exceeds expanded size limit")
        if not REQUIRED.issubset(seen):
            raise ValueError("snapshot missing .env, policy.yaml or state.db")
        # Extract only validated regular files/directories. The destination
        # is an operator-owned fresh directory, never the live state.
        for member in members:
            if member.isdir():
                (destination / member.name).mkdir(parents=True, exist_ok=True)
                continue
            target = destination / member.name
            target.parent.mkdir(parents=True, exist_ok=True)
            handle = pack.extractfile(member)
            if handle is None:
                raise ValueError("unreadable snapshot member")
            with target.open("xb") as out:
                while block := handle.read(1024 * 1024):
                    out.write(block)
            os.chmod(target, 0o600)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--archive", type=pathlib.Path, required=True)
    parser.add_argument("--dest", type=pathlib.Path, required=True)
    args = parser.parse_args()
    try:
        stage_snapshot(args.archive, args.dest)
    except (ValueError, OSError, tarfile.TarError) as exc:
        parser.exit(1, f"ROLLBACK PRECHECK FAILED: {exc}\n")


if __name__ == "__main__":
    main()
