"""Fail-closed tests for restoring privilege state from an older snapshot.

All SQLite databases are synthetic and disposable; no production data, Docker
sockets, secrets, or networks are used in this test suite.
"""
from __future__ import annotations

import hashlib
import importlib.util
import io
import json
from pathlib import Path
import sqlite3
import subprocess
import sys
import tempfile
import unittest


SCRIPT = Path(__file__).resolve().parents[2] / "scripts/lib/invalidate-update-authority.py"
SQL = """
CREATE TABLE grants (grant_id TEXT PRIMARY KEY, revoked_at INTEGER);
CREATE TABLE root_delegations (delegation_id TEXT PRIMARY KEY, revoked_at INTEGER);
CREATE TABLE approvals (
  request_id TEXT PRIMARY KEY, status TEXT NOT NULL, decided_at INTEGER
);
CREATE TABLE audit_events (
  seq INTEGER PRIMARY KEY, event_json TEXT NOT NULL,
  prev_hash TEXT NOT NULL, hash TEXT NOT NULL
);
"""


class RollbackAuthorityFenceTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.dbpath = Path(self.temporary.name) / "state.db"
        with sqlite3.connect(self.dbpath) as con:
            con.executescript(SQL)
            con.execute("INSERT INTO grants VALUES ('grant-1', NULL)")
            con.execute("INSERT INTO grants VALUES ('grant-revoked', 10)")
            con.execute("INSERT INTO root_delegations VALUES ('root-1', NULL)")
            con.execute("INSERT INTO approvals VALUES ('apr-pending', 'pending', NULL)")
            con.execute("INSERT INTO approvals VALUES ('apr-denied', 'denied', 12)")
            body = '{"time":"2026-01-01T00:00:00Z","subject":"operator","tool":"test","decision":"allow"}'
            digest = hashlib.sha256(f"1\nGENESIS\n{body}".encode()).hexdigest()
            con.execute("INSERT INTO audit_events VALUES (1, ?, 'GENESIS', ?)",
                        (body, digest))

    def execute(self, path=None):
        return subprocess.run(
            [sys.executable, str(SCRIPT), "--db", str(path or self.dbpath)],
            text=True, capture_output=True, timeout=12,
        )

    def inspect(self):
        with sqlite3.connect(self.dbpath) as con:
            grants = con.execute("SELECT grant_id, revoked_at FROM grants ORDER BY grant_id").fetchall()
            roots = con.execute("SELECT delegation_id, revoked_at FROM root_delegations").fetchall()
            approvals = con.execute(
                "SELECT request_id, status, decided_at FROM approvals ORDER BY request_id"
            ).fetchall()
            history = con.execute(
                "SELECT seq, event_json, prev_hash, hash FROM audit_events ORDER BY seq"
            ).fetchall()
        return grants, roots, approvals, history

    def test_revoke_all_elevation_before_broker_restart_with_audited_event(self):
        result = self.execute()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("grants=1", result.stdout)
        grants, roots, approvals, history = self.inspect()
        self.assertTrue(all(revoked is not None for _, revoked in grants))
        self.assertIsNotNone(roots[0][1])
        self.assertEqual(approvals[0][0:2], ("apr-denied", "denied"))
        self.assertEqual(approvals[1][0:2], ("apr-pending", "revoked"))
        self.assertIsNotNone(approvals[1][2])
        self.assertEqual(len(history), 2)
        previous = "GENESIS"
        for expected_seq, (seq, body, prev, digest) in enumerate(history, start=1):
            self.assertEqual(seq, expected_seq)
            self.assertEqual(prev, previous)
            self.assertEqual(
                digest, hashlib.sha256(f"{seq}\n{prev}\n{body}".encode()).hexdigest()
            )
            previous = digest
        last = json.loads(history[-1][1])
        self.assertEqual(last["tool"], "system.rollback_revoke_all")

    def test_broken_audit_chain_fails_before_modifying_any_grant(self):
        with sqlite3.connect(self.dbpath) as con:
            con.execute("UPDATE audit_events SET hash='tampered' WHERE seq=1")
        result = self.execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("audit chain invalid", result.stderr)
        grants, roots, approvals, history = self.inspect()
        self.assertIsNone(grants[0][1])
        self.assertIsNone(roots[0][1])
        self.assertEqual(approvals[1][1], "pending")
        self.assertEqual(len(history), 1)

    def test_unknown_schema_does_not_silently_initialize_new_state(self):
        with sqlite3.connect(self.dbpath) as con:
            con.execute("DROP TABLE root_delegations")
        result = self.execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("missing required columns", result.stderr)
        self.assertIsNone(self.inspect_partial_grant_revocation())

    def inspect_partial_grant_revocation(self):
        with sqlite3.connect(self.dbpath) as con:
            row = con.execute("SELECT revoked_at FROM grants WHERE grant_id='grant-1'").fetchone()
        return row[0]

    def test_missing_file_does_not_create_database(self):
        path = self.dbpath.parent / "not-existing.db"
        result = self.execute(path=path)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("missing", result.stderr)
        self.assertFalse(path.exists())

    def test_second_execution_never_resurrects_approved_privilege(self):
        first = self.execute()
        self.assertEqual(first.returncode, 0, first.stderr)
        second = self.execute()
        self.assertEqual(second.returncode, 0, second.stderr)
        self.assertIn("grants=0", second.stdout)
        grants, roots, approvals, history = self.inspect()
        self.assertTrue(all(row[1] is not None for row in grants))
        self.assertEqual(len(history), 3)


if __name__ == "__main__":
    unittest.main()
