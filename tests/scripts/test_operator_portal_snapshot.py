"""Snapshot contract: preserves configuration and SQLite state, never modifies source."""
import importlib.util
import json
import os
from pathlib import Path
import sqlite3
import tempfile
import unittest

ROOT=Path(__file__).resolve().parents[2]
SPEC=importlib.util.spec_from_file_location("portal_snapshot",ROOT/"scripts/operator-portal-snapshot.py")
MODULE=importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)

class SnapshotTests(unittest.TestCase):
    def test_consistent_backup_and_private_permissions(self):
        with tempfile.TemporaryDirectory() as td:
            root=Path(td)
            app=root/"installation"
            app.mkdir()
            (app/"config").mkdir()
            (app/"state").mkdir()
            (app/".env").write_text("A=private\n")
            (app/"compose.yaml").write_text("services: {}\n")
            db=sqlite3.connect(app/"state"/"state.db")
            db.execute("CREATE TABLE test (x TEXT)")
            db.execute("INSERT INTO test VALUES ('before')")
            db.commit()
            db.close()
            out=root/"backup"
            info=MODULE.snapshot(out,app)
            self.assertIn(".env",info["captured"])
            self.assertEqual((out/".env").read_text(),"A=private\n")
            self.assertEqual((out/".env").stat().st_mode & 0o777,0o600)
            check=sqlite3.connect(out/"state"/"state.db")
            self.assertEqual(check.execute("SELECT x FROM test").fetchone(),("before",))
            check.close()
            self.assertEqual((out/"manifest.json").stat().st_mode & 0o777,0o600)
            self.assertTrue((out/"manifest.json").is_file())
            self.assertEqual((app/".env").read_text(),"A=private\n")
    def test_reject_backup_inside_repo(self):
        with tempfile.TemporaryDirectory() as td:
            app=Path(td)
            with self.assertRaises(ValueError):
                MODULE.snapshot(app/"backups"/"new",app)

if __name__=="__main__":
    unittest.main()
