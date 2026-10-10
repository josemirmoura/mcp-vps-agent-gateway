"""Fault-inject the production updater locking and backup allocation prelude.

Only temp files and a copied product-language stub are used. This suite never
calls Docker, Git or the real VPS, and never reads operator credentials.
"""
from __future__ import annotations

import os
from pathlib import Path
import subprocess
import tempfile
import time
import unittest

ROOT = Path(__file__).resolve().parents[2]
UPDATER = ROOT / "scripts/update.sh"


class UpdateSerializationTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name) / "checkout"
        (self.root / "scripts/lib").mkdir(parents=True)
        (self.root / "scripts/lib/product.sh").write_text(
            "vps_agent_init_language() { :; }\n"
        )
        self.script_path = self.root / "scripts/update.sh"
        self.script_path.write_text("#!/bin/bash\n")
        full = UPDATER.read_text()
        self.prelude = full[:full.index("if [ ! -f .env ]")]
        self.allocator = full[
            full.index('stamp="$(date -u +%Y%m%dT%H%M%SZ)"'):
            full.index("compose=(docker compose -f compose.yaml)")
        ]

    def run_shell(self, command, timeout=6):
        return subprocess.run(
            ["bash", "-c", command, str(self.script_path)],
            cwd=self.root, text=True, capture_output=True, timeout=timeout,
            check=False,
        )

    def test_second_concurrent_update_fails_before_mutation(self):
        first = subprocess.Popen(
            ["bash", "-c", self.prelude + "\nprintf 'LOCKED\\n'\nexec sleep 15\n",
             str(self.script_path)],
            cwd=self.root, text=True, stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        try:
            self.assertEqual(first.stdout.readline().strip(), "LOCKED")
            second = self.run_shell(self.prelude + "\nprintf 'SHOULD_NOT_REACH\\n'")
            self.assertNotEqual(second.returncode, 0)
            self.assertIn("another update is already running", second.stderr)
            self.assertNotIn("SHOULD_NOT_REACH", second.stdout)
        finally:
            first.terminate()
            first.communicate(timeout=5)
        retry = self.run_shell(self.prelude + "\nprintf 'NEXT_UPDATE_OK\\n'")
        self.assertEqual(retry.returncode, 0, retry.stderr)
        self.assertIn("NEXT_UPDATE_OK", retry.stdout)

    def test_symlinked_backups_dir_is_rejected(self):
        outside = Path(self.tmp.name) / "outside"
        outside.mkdir()
        (self.root / "backups").symlink_to(outside, target_is_directory=True)
        result = self.run_shell(self.prelude + "\nprintf 'SHOULD_NOT_REACH\\n'")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("not a symlink", result.stderr)
        self.assertFalse(any(outside.iterdir()))
        
    def test_backup_paths_are_unique_even_same_second(self):
        # Runs the ACTUAL allocation fragment under one fixed clock value.
        script = (
            self.prelude
            + "\ndate() { printf '20261010T003900Z\\n'; }\n"
            + self.allocator
            + "\nprintf 'BACKUP=%s\\n' \"$backup_dir\"\n"
        )
        first = self.run_shell(script)
        second = self.run_shell(script)
        self.assertEqual(first.returncode, 0, first.stderr)
        self.assertEqual(second.returncode, 0, second.stderr)
        one = first.stdout.strip().split("BACKUP=")[-1]
        two = second.stdout.strip().split("BACKUP=")[-1]
        self.assertNotEqual(one, two)
        for name in (one, two):
            p = self.root / name
            self.assertTrue(p.is_dir())
            self.assertEqual(p.stat().st_mode & 0o777, 0o700)
        self.assertEqual(len(list((self.root/"backups").glob("20261010T003900Z.*"))), 2)

    def test_lock_releases_after_failure(self):
        failed = self.run_shell(self.prelude + "\nexit 74\n")
        self.assertEqual(failed.returncode, 74)
        next_run = self.run_shell(self.prelude + "\nprintf 'LOCK_RECOVERED\\n'\n")
        self.assertEqual(next_run.returncode, 0, next_run.stderr)
        self.assertIn("LOCK_RECOVERED", next_run.stdout)


if __name__ == "__main__":
    unittest.main()
