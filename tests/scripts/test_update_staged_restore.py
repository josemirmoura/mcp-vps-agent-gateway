"""Fault injection for two-phase Community update rollback.

Only temporary directory files and shell mocks are modified. Never call a
real Docker daemon, Git checkout, production .env or Broker in these tests.
"""
from __future__ import annotations

import importlib.util
import io
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SNAPSHOT = ROOT / "scripts/lib/update-snapshot.py"
UPDATER = ROOT / "scripts/update.sh"
VOLUME_HELPER = ROOT / "scripts/lib/restore-identity-volume.sh"
SNAPSHOT_ENTRIES = {
    ".env": b"original-secret-not-real\n",
    "config/policy.yaml": b"original-policy\n",
    "state/state.db": b"original-db\n",
}


def write_tar(path: Path, entries: list[tuple[str, bytes, bytes | None]]) -> None:
    with tarfile.open(path, mode="w:gz") as bundle:
        for name, content, special in entries:
            item = tarfile.TarInfo(name)
            if special:
                item.type = special
                bundle.addfile(item)
            else:
                item.size = len(content)
                bundle.addfile(item, io.BytesIO(content))


def snapshot_items() -> list[tuple[str, bytes, bytes | None]]:
    return [(name, content, None) for name, content in SNAPSHOT_ENTRIES.items()]


class StagedRestoreTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.archive = self.root / "operator-state.tar.gz"
        self.dest = self.root / "staged"

    def stage(self):
        return subprocess.run(
            [sys.executable, str(SNAPSHOT), "--archive", str(self.archive),
             "--dest", str(self.dest)],
            capture_output=True, text=True, timeout=10, check=False,
        )

    def test_valid_operator_snapshot_stages_all_content(self):
        write_tar(self.archive, snapshot_items())
        result = self.stage()
        self.assertEqual(result.returncode, 0, result.stderr)
        for name, expected in SNAPSHOT_ENTRIES.items():
            self.assertEqual((self.dest / name).read_bytes(), expected)
        self.assertEqual((self.dest / ".env").stat().st_mode & 0o777, 0o600)

    def test_missing_state_db_rejected(self):
        write_tar(self.archive, snapshot_items()[:-1])
        result = self.stage()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("missing", result.stderr)

    def test_symlink_rejected_without_extracting(self):
        write_tar(self.archive, snapshot_items() + [
            ("state/link", b"", tarfile.SYMTYPE),
        ])
        result = self.stage()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("links", result.stderr)
        self.assertFalse((self.dest / ".env").exists())

    def test_path_traversal_rejected_without_extracting(self):
        write_tar(self.archive, snapshot_items() + [
            ("state/../../outside", b"bad", None),
        ])
        result = self.stage()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("unsafe", result.stderr)
        self.assertFalse((self.root / "outside").exists())

    def test_duplicate_member_rejected(self):
        write_tar(self.archive, snapshot_items() + [
            (".env", b"replacement", None),
        ])
        result = self.stage()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("duplicate", result.stderr)

    def test_corrupted_snapshot_fails_closed(self):
        self.archive.write_bytes(b"invalid-gzip")
        result = self.stage()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("ROLLBACK PRECHECK FAILED", result.stderr)

    def test_excessive_declared_size_is_refused(self):
        spec = importlib.util.spec_from_file_location("update_snapshot", SNAPSHOT)
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        write_tar(self.archive, snapshot_items())
        module.MAX_EXTRACTED_BYTES = 4
        with self.assertRaisesRegex(ValueError, "expanded size"):
            module.stage_snapshot(self.archive, self.dest)


class RollbackRecoveryTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / "scripts/lib").mkdir(parents=True)
        shutil.copy2(SNAPSHOT, self.root / "scripts/lib/update-snapshot.py")
        shutil.copy2(VOLUME_HELPER, self.root / "scripts/lib/restore-identity-volume.sh")
        (self.root / "scripts").mkdir(exist_ok=True)
        (self.root / "scripts/verify.sh").write_text("#!/bin/sh\nexit 0\n")
        (self.root / "scripts/verify-public.sh").write_text("#!/bin/sh\nexit 0\n")
        (self.root / "config").mkdir()
        (self.root / "config/policy.yaml").write_text("upgraded-policy\n")
        (self.root / ".env").write_text("upgraded-secret-not-real\n")
        (self.root / "state").mkdir()
        (self.root / "state/state.db").write_text("upgraded-db\n")
        (self.root / "backups/test").mkdir(parents=True)
        self.archive = self.root / "backups/test/operator-state.tar.gz"
        write_tar(self.archive, snapshot_items())
        self.log = self.root / "calls.log"
        full = UPDATER.read_text()
        begin = full.index("rollback() {\n")
        end = full.index("\ngit merge --ff-only", begin)
        self.rollback_functions = full[begin:end]

    def execute(self, failure: str = "", identity: bool = False):
        if identity:
            (self.root / "backups/test/zitadel-postgres-volume.tar.gz").write_bytes(b"synthetic")
            (self.root / "backups/test/zitadel-bootstrap-volume.tar.gz").write_bytes(b"synthetic")
        script = (
            'set -euo pipefail\n'
            'vps_agent_text() { printf "%s" "$1"; }\n'
            'current=original-commit\n'
            'target_sha=upgraded-commit\n'
            'backup_dir=backups/test\n'
            'compose=(docker compose -f compose.yaml)\n'
            'docker() {\n'
            '  printf "docker %s\\n" "$*" >> "$TEST_LOG"\n'
            '  if [ "$FAILURE" = "stop" ] && [[ "$*" == *" stop" ]]; then return 71; fi\n'
            '  if [ "$FAILURE" = "volume" ] && [ "$1" = run ]; then return 72; fi\n'
            '  if [ "$FAILURE" = "restart" ] && [[ "$*" == *" up -d --build" ]]; then return 73; fi\n'
            '  return 0\n'
            '}\n'
            'git() {\n'
            '  printf "git %s\\n" "$*" >> "$TEST_LOG"\n'
            '  if [ "$FAILURE" = "git-reset" ]; then return 74; fi\n'
            '  return 0\n'
            '}\n'
            'mv() {\n'
            '  printf "mv %s\\n" "$*" >> "$TEST_LOG"\n'
            '  if [ "$FAILURE" = "stage-move" ] && [[ "$*" == *"/state state" ]]; then return 75; fi\n'
            '  command mv "$@"\n'
            '}\n'
            'VPS_AGENT_AUTH_MODE=none\n'
            + self.rollback_functions + '\n'
            'false\n'
        )
        env = {**os.environ, "FAILURE": failure, "TEST_LOG": str(self.log)}
        result = subprocess.run(
            ["bash", "-c", script], cwd=self.root, env=env,
            capture_output=True, text=True, timeout=15,
        )
        commands = self.log.read_text().splitlines() if self.log.exists() else []
        return result, commands

    def test_corrupt_snapshot_does_not_stop_or_delete_state(self):
        self.archive.write_bytes(b"broken snapshot")
        result, calls = self.execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("ROLLBACK BLOCKED", result.stderr)
        self.assertEqual((self.root / "state/state.db").read_text(), "upgraded-db\n")
        self.assertFalse(any(" stop" in x for x in calls))

    def test_stop_error_leaves_state_intact(self):
        result, _ = self.execute(failure="stop")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("unable to stop", result.stderr)
        self.assertEqual((self.root / "state/state.db").read_text(), "upgraded-db\n")

    def test_git_reset_error_preserves_upgraded_state(self):
        result, _ = self.execute(failure="git-reset")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("previous source revision", result.stderr)
        self.assertEqual((self.root / "state/state.db").read_text(), "upgraded-db\n")

    def test_volume_failure_preserves_upgraded_state(self):
        result, calls = self.execute(failure="volume", identity=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("identity volume restore failed", result.stderr)
        self.assertEqual((self.root / "state/state.db").read_text(), "upgraded-db\n")
        self.assertTrue(any("docker run" in x for x in calls))

    def test_staged_state_move_failure_restores_previous_directory(self):
        result, _ = self.execute(failure="stage-move")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("could not install staged original state", result.stderr)
        self.assertEqual((self.root / "state/state.db").read_text(), "upgraded-db\n")

    def test_successful_rollback_recovers_original_and_keeps_new_state(self):
        result, calls = self.execute()
        self.assertNotEqual(result.returncode, 0, "failed upgrade still returns nonzero")
        self.assertIn("ROLLBACK COMPLETE", result.stderr)
        self.assertEqual((self.root / "state/state.db").read_text(), "original-db\n")
        self.assertEqual((self.root / ".env").read_text(), "original-secret-not-real\n")
        self.assertEqual((self.root / "config/policy.yaml").read_text(), "original-policy\n")
        self.assertEqual((self.root / "backups/test/failed-target-state/state.db").read_text(), "upgraded-db\n")
        self.assertTrue(any(" up -d --build" in x for x in calls))

    def test_failed_restart_is_visible_and_original_data_remains(self):
        result, _ = self.execute(failure="restart")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("original runtime did not pass", result.stderr)
        self.assertEqual((self.root / "state/state.db").read_text(), "original-db\n")



class VolumeArchivePreflightTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.tar = self.root / "identity.tar.gz"
        self.checker = ROOT / "scripts/lib/verify-update-volume.py"

    def create_tar(self, extra=None, root=True):
        with tarfile.open(self.tar, mode="w:gz") as pack:
            if root:
                item = tarfile.TarInfo(".")
                item.type = tarfile.DIRTYPE
                pack.addfile(item)
            for name, data, typ in [("./PG_VERSION", b"16\\n", None)] + (extra or []):
                item = tarfile.TarInfo(name)
                if typ:
                    item.type = typ
                    pack.addfile(item)
                else:
                    item.size = len(data)
                    pack.addfile(item, io.BytesIO(data))

    def run_check(self):
        return subprocess.run(
            [sys.executable, str(self.checker), str(self.tar)],
            capture_output=True, text=True, timeout=12,
        )

    def test_valid_integrated_volume_archive(self):
        self.create_tar(extra=[("./postgresql.auto.conf", b"ok", None)])
        result = self.run_check()
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_absolute_path_rejected(self):
        self.create_tar(extra=[("/etc/shadow", b"oops", None)])
        result = self.run_check()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("non-relative", result.stderr)

    def test_path_traversal_rejected(self):
        self.create_tar(extra=[("./../../escape", b"oops", None)])
        result = self.run_check()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("unsafe member", result.stderr)

    def test_duplicate_member_rejected(self):
        self.create_tar(extra=[("./PG_VERSION", b"bad", None)])
        result = self.run_check()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("duplicate", result.stderr)

    def test_symbolic_link_rejected(self):
        self.create_tar(extra=[("./escape", b"", tarfile.SYMTYPE)])
        result = self.run_check()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("unsafe link", result.stderr)

    def test_missing_root_rejected(self):
        self.create_tar(root=False)
        result = self.run_check()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("no root", result.stderr)

    def test_corrupted_volume_archive_rejected(self):
        self.tar.write_bytes(b"not a valid archive")
        result = self.run_check()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("IDENTITY SNAPSHOT INVALID", result.stderr)


if __name__ == "__main__":
    unittest.main()
