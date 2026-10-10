#!/usr/bin/env python3
"""Fail closed on permissions when restoring an older Community SQLite state.

Operates ONLY on a disposable rollback staging copy before Broker starts.
Matches Store.RevokeAll for elevated grants, root delegations and pending
approvals, with an append-only, hash-chained recovery audit event in the
same SQLite transaction. A failed precheck never edits the live database.
"""
from __future__ import annotations

import argparse
import datetime
import hashlib
import json
import pathlib
import sqlite3
import time


REQUIRED_COLUMNS = {
    "grants": {"grant_id", "revoked_at"},
    "root_delegations": {"delegation_id", "revoked_at"},
    "approvals": {"request_id", "status", "decided_at"},
    "audit_events": {"seq", "event_json", "prev_hash", "hash"},
}


def invalidate_staged_authority(database: pathlib.Path) -> dict[str, int]:
    if database.is_symlink() or not database.is_file():
        raise ValueError("staged state DB missing or symlinked")
    if database.stat().st_size == 0:
        raise ValueError("staged state DB is empty")
    # Connecting in rw mode prevents sqlite3 from creating a new empty DB.
    database = database.resolve(strict=True)
    with sqlite3.connect(database.as_uri() + "?mode=rw", uri=True,
                         timeout=5, isolation_level=None) as db:
        integrity = db.execute("PRAGMA integrity_check").fetchone()
        if not integrity or integrity[0] != "ok":
            raise ValueError("staged state DB integrity check failed")
        for table, required in REQUIRED_COLUMNS.items():
            columns = {row[1] for row in db.execute(f"PRAGMA table_info({table})")}
            if not required.issubset(columns):
                raise ValueError(f"staged DB missing required columns in {table}")
        # Reject audit corruption rather than appending a valid event to
        # a broken chain. This is the same hash/GENESIS format as state.go.
        previous = "GENESIS"
        next_sequence = 1
        for sequence, body, parent, digest in db.execute(
                "SELECT seq,event_json,prev_hash,hash FROM audit_events ORDER BY seq"):
            expected = hashlib.sha256(
                f"{sequence}\n{parent}\n{body}".encode("utf-8")
            ).hexdigest()
            if sequence != next_sequence or parent != previous or digest != expected:
                raise ValueError("staged DB audit chain invalid")
            next_sequence += 1
            previous = digest

        timestamp = time.time_ns()
        event = {
            "time": datetime.datetime.now(datetime.timezone.utc).isoformat(
                timespec="microseconds").replace("+00:00", "Z"),
            "subject": "local-recovery",
            "tool": "system.rollback_revoke_all",
            "resource": "staged-snapshot",
            "decision": "revoked",
        }
        event_json = json.dumps(event, separators=(",", ":"), ensure_ascii=False)
        new_digest = hashlib.sha256(
            f"{next_sequence}\n{previous}\n{event_json}".encode("utf-8")
        ).hexdigest()
        db.execute("BEGIN IMMEDIATE")
        try:
            grants = db.execute(
                "UPDATE grants SET revoked_at=? WHERE revoked_at IS NULL",
                (timestamp,),
            ).rowcount
            delegations = db.execute(
                "UPDATE root_delegations SET revoked_at=? WHERE revoked_at IS NULL",
                (timestamp,),
            ).rowcount
            approvals = db.execute(
                "UPDATE approvals SET status='revoked', decided_at=? "
                "WHERE status='pending'", (timestamp,),
            ).rowcount
            db.execute(
                "INSERT INTO audit_events(seq,event_json,prev_hash,hash) "
                "VALUES(?,?,?,?)",
                (next_sequence, event_json, previous, new_digest),
            )
            db.execute("COMMIT")
        except Exception:
            db.execute("ROLLBACK")
            raise
        # Bundle the WAL pages into the staged DB before the same-filesystem
        # rename. Never leave uncheckpointed privileges hidden in a -wal.
        db.execute("PRAGMA wal_checkpoint(TRUNCATE)")
    return {"grants": grants, "root_delegations": delegations,
            "pending_approvals": approvals}


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--db", type=pathlib.Path, required=True)
    args = parser.parse_args()
    try:
        count = invalidate_staged_authority(args.db)
    except (ValueError, OSError, sqlite3.Error) as exc:
        parser.exit(1, f"ROLLBACK AUTHORITY PRECHECK FAILED: {exc}\n")
    print("ROLLBACK AUTHORITY INVALIDATED: "
          + ", ".join(f"{kind}={value}" for kind, value in count.items()))


if __name__ == "__main__":
    main()
