#!/usr/bin/env python3
"""Strict preflight for an identity-volume tar.gz created by Docker/tar.

The archive is only INSPECTED, never extracted. A valid gzip stream is not
sufficient: reject paths escaping a volume, duplicate files, links and special
file types before the updater enters the post-backup rollback window.
"""
from __future__ import annotations

import argparse
import pathlib
import tarfile

MAX_ENTRIES = 250_000
MAX_EXTRACTED_BYTES = 100 * 1024 * 1024 * 1024


def verify_volume_archive(archive: pathlib.Path) -> None:
    seen: set[str] = set()
    size = 0
    root_found = False
    with tarfile.open(archive, "r:gz") as pack:
        for index, member in enumerate(pack):
            if index >= MAX_ENTRIES:
                raise ValueError("identity volume snapshot exceeds entry limit")
            name = member.name
            if name in (".", "./"):
                if root_found or not member.isdir():
                    raise ValueError("duplicate or invalid volume root")
                root_found = True
                continue
            if not name.startswith("./"):
                raise ValueError("identity volume has non-relative member path")
            segments = name[2:].rstrip("/").split("/")
            if (any(segment in ("", ".", "..") for segment in segments)
                    or chr(92) in name):
                raise ValueError("identity volume has unsafe member path")
            normalized = "/".join(segments)
            if normalized in seen:
                raise ValueError("identity volume has duplicate member")
            seen.add(normalized)
            if not (member.isfile() or member.isdir()):
                raise ValueError("identity volume has unsafe link or special member")
            if member.isfile():
                size += member.size
                if size > MAX_EXTRACTED_BYTES:
                    raise ValueError("identity volume exceeds expanded size limit")
    if not root_found:
        raise ValueError("identity volume snapshot has no root directory")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("archive", type=pathlib.Path)
    args = parser.parse_args()
    try:
        verify_volume_archive(args.archive)
    except (OSError, ValueError, tarfile.TarError) as exc:
        parser.exit(1, f"IDENTITY SNAPSHOT INVALID: {exc}\n")


if __name__ == "__main__":
    main()
