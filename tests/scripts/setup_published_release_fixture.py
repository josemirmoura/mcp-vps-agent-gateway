#!/usr/bin/env python3
"""Create isolated, FAKE signed-release CLI fixtures for Docker lifecycle CI.

The fake signatures do not prove cryptographic security. Never run this helper
to approve, publish or deploy a real artifact.
"""
from __future__ import annotations

import argparse
import gzip
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys


def write_executable(path: Path, body: str) -> None:
    path.write_text("#!" + sys.executable + "\n" + body)
    path.chmod(0o700)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, required=True)
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--commit", required=True)
    args = parser.parse_args()
    root = args.root.resolve()
    asset_dir = root / "releases" / args.tag
    asset_dir.mkdir(parents=True, exist_ok=True)
    remote_version = subprocess.run(
        ["git", "show", args.commit + ":VERSION"], cwd=args.repo,
        capture_output=True, check=True, text=True).stdout.strip()
    if args.tag != "v" + remote_version:
        raise SystemExit("fixture tag MUST match the exact commit VERSION")
    raw = subprocess.run(
        ["git", "archive", "--format=tar", "--prefix=mcp-vps-agent/", args.commit],
        cwd=args.repo, check=True, capture_output=True).stdout
    name = "mcp-vps-agent-source-package.tar.gz"
    (asset_dir / name).write_bytes(gzip.compress(raw, mtime=0))
    digest = hashlib.sha256((asset_dir / name).read_bytes()).hexdigest()
    (asset_dir / (name + ".sha256")).write_text(
        f"{digest}  dist/{name}\n")
    (asset_dir / "RELEASE-PROVENANCE.json").write_text(json.dumps({
        "schema": 1, "tag": args.tag, "commit_sha": args.commit,
        "source_sha256": digest,
    }))
    for original in (name, name + ".sha256", "RELEASE-PROVENANCE.json"):
        (asset_dir / (original + ".sigstore.json")).write_text(
            '{"INSECURE_TEST_FIXTURE_ONLY":true}')
    (root / "metadata").mkdir(exist_ok=True)
    (root / "metadata" / (args.tag + ".json")).write_text(json.dumps({
        "tagName": args.tag, "isDraft": False, "publishedAt": "2026-10-10T00:00:00Z",
        "assets": [{"name": item.name, "size": item.stat().st_size}
                   for item in asset_dir.iterdir()],
    }))
    bin_dir = root / "fakebin"
    bin_dir.mkdir(exist_ok=True)
    write_executable(bin_dir / "gh", """
import os,json,sys,shutil
from pathlib import Path
args=sys.argv[1:]
root=Path(os.environ['PORTICO_FIXTURE_ROOT'])
tag=args[2]
if args[:2]==['release','view']:
    path=root/'metadata'/(tag+'.json')
    if not path.exists():sys.exit(2)
    print(path.read_text())
elif args[:2]==['release','download']:
    dst=Path(args[args.index('--dir')+1])
    for p in (root/'releases'/tag).iterdir():
        shutil.copyfile(p,dst/p.name)
else:
    sys.exit(2)
""")
    write_executable(bin_dir / "cosign", """
import sys
args=sys.argv[1:]
assert args[0] in ('verify','verify-blob')
assert '--certificate-oidc-issuer' in args
assert 'https://token.actions.githubusercontent.com' in args
assert '--certificate-identity-regexp' in args
""")
    write_executable(bin_dir / "crane", """
import sys
assert sys.argv[1]=='digest'
print('sha256:'+'a'*64)
""")
    print("FAKE RELEASE FIXTURE READY:", args.tag, args.commit)
    print("WARNING: fixture signatures are NOT authentic release evidence")


if __name__ == "__main__":
    main()
