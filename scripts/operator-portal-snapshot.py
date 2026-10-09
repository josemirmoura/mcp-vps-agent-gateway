#!/usr/bin/env python3
"""Consistent, local, 0700-protected predeployment snapshot. No deployment."""
from __future__ import annotations
import argparse
import datetime as dt
import json
import os
from pathlib import Path
import shutil
import sqlite3
import subprocess

FILES=("compose.yaml","compose.override.yaml","compose.integrated-auth.yaml",".env",
       "config/policy.yaml","config/policy.example.yaml")
def snapshot(destination: Path, cwd: Path) -> dict:
    destination=destination.expanduser().resolve()
    cwd=cwd.resolve()
    if destination == cwd or cwd in destination.parents:
        raise ValueError("Backup must be outside the application directory")
    if destination.exists():
        raise FileExistsError("Backup directory already exists; not overwriting")
    os.umask(0o077)
    destination.mkdir(parents=True,mode=0o700)
    try:
        (destination/"config").mkdir(mode=0o700)
        captured=[]
        for relative in FILES:
            source=cwd/relative
            if source.is_symlink():
                raise ValueError(f"Refusing linked file: {relative}")
            if source.is_file():
                target=destination/relative
                target.parent.mkdir(parents=True,exist_ok=True)
                shutil.copy2(source,target)
                target.chmod(0o600)
                captured.append(relative)
        state_dir=cwd/"state"
        if state_dir.is_dir():
            (destination/"state").mkdir(mode=0o700)
            for item in state_dir.iterdir():
                if item.is_symlink():
                    raise ValueError("State directory contains symlink; refusing snapshot")
                if item.is_file() and item.name.endswith(".db"):
                    source=sqlite3.connect(f"file:{item}?mode=ro",uri=True,timeout=10)
                    try:
                        target=sqlite3.connect(destination/"state"/item.name)
                        try:source.backup(target)
                        finally:target.close()
                    finally:source.close()
                    captured.append("state/"+item.name+" (SQLite consistent backup)")
                elif item.is_file() and not item.name.endswith((".db-wal",".db-shm")):
                    shutil.copy2(item,destination/"state"/item.name)
                    (destination/"state"/item.name).chmod(0o600)
                    captured.append("state/"+item.name)
                elif item.is_dir():
                    raise ValueError("Nested state directory requires explicit backup procedure")
        rev=subprocess.run(["git","rev-parse","HEAD"],cwd=cwd,capture_output=True,text=True,check=False)
        info={"created_at":dt.datetime.now(dt.timezone.utc).isoformat(),"commit":rev.stdout.strip() if rev.returncode==0 else None,
              "application_dir":str(cwd),"captured":captured}
        (destination/"manifest.json").write_text(json.dumps(info,indent=2)+"\n")
        (destination/"manifest.json").chmod(0o600)
        return info
    except BaseException:
        # Never silently proceed after an incomplete backup.
        shutil.rmtree(destination)
        raise

def main():
    parser=argparse.ArgumentParser()
    parser.add_argument("--output",required=True,help="new absolute backup directory OUTSIDE the installation")
    args=parser.parse_args()
    info=snapshot(Path(args.output),Path.cwd())
    print("BACKUP READY; no application changes")
    print("Location:",args.output)
    print("Commit:",info["commit"] or "unavailable")
    print("Files:",len(info["captured"]))

if __name__=="__main__":
    main()
