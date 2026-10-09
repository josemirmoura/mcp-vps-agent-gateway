"""Regression guards for the pre-backup recovery window in scripts/update.sh.

Run: python3 -m unittest tests/scripts/test_update_prebackup_recovery.py -v
All Docker/Git interactions are mock shell functions. No host operations occur.
"""

from __future__ import annotations

import os
from pathlib import Path
import subprocess
import tempfile
import unittest


UPDATER = Path(__file__).resolve().parents[2] / "scripts" / "update.sh"
START = "# At this point the live package has not been modified."
END = "\nrollback() {"


class PrebackupRecoveryTest(unittest.TestCase):
    def execute_failure(self, fail_at: str) -> tuple[subprocess.CompletedProcess[str], list[str]]:
        source = UPDATER.read_text(encoding="utf-8")
        self.assertIn(START, source)
        self.assertIn(END, source)
        section = source[source.index(START):source.index(END, source.index(START))]
        with tempfile.TemporaryDirectory(prefix="portico-update-test-") as temp:
            root = Path(temp)
            (root / "backups").mkdir()
            (root / ".env").write_text("test=1\n", encoding="utf-8")
            (root / "config").mkdir()
            (root / "config" / "policy.yaml").write_text("{}\n", encoding="utf-8")
            (root / "state").mkdir()
            (root / "state" / "state.db").write_text("original\n", encoding="utf-8")
            log = root / "commands.log"
            script = (
                'set -euo pipefail\n'
                'vps_agent_text() { printf "%s" "$1"; }\n'
                'compose=(docker compose -f compose.yaml)\n'
                'backup_dir=backups/test\n'
                'mkdir -p "$backup_dir"\n'
                'current=original-commit\n'
                'target_sha=target-commit\n'
                'docker() {\n'
                '  printf "docker %s\\n" "$*" >> "$TEST_LOG"\n'
                '  if [[ "$FAIL_AT" == "stop" && "$*" == *" stop" ]]; then return 39; fi\n'
                '  if [[ "$FAIL_AT" == "volume" && "$1" == "run" ]]; then return 47; fi\n'
                '  if [[ "$FAIL_AT" == "restart" && "$*" == *" up -d" ]]; then return 52; fi\n'
                '  return 0\n'
                '}\n'
                'tar() {\n'
                '  printf "tar %s\\n" "$*" >> "$TEST_LOG"\n'
                '  if [[ "$FAIL_AT" == "tar" || "$FAIL_AT" == "restart" ]]; then return 42; fi\n'
                '  printf "partial-synthetic-backup" > "$backup_dir/operator-state.tar.gz"\n'
                '  return 0\n'
                '}\n'
            )
            if fail_at == "volume":
                script += 'VPS_AGENT_AUTH_MODE=integrated\n'
            else:
                script += 'VPS_AGENT_AUTH_MODE=none\n'
            script += section + '\nprintf "SHOULD_NOT_REACH_HERE\\n" >> "$TEST_LOG"\n'
            env = {**os.environ, "TEST_LOG": str(log), "FAIL_AT": fail_at}
            result = subprocess.run(
                ["bash", "-c", script],
                cwd=root,
                env=env,
                text=True,
                capture_output=True,
                check=False,
                timeout=15,
            )
            self.assertEqual((root / "state" / "state.db").read_text(), "original\n")
            self.assertEqual((root / ".env").read_text(), "test=1\n")
            return result, log.read_text(encoding="utf-8").splitlines()

    def assert_restart_attempted(self, fail_at: str, expected_exit: int) -> None:
        result, commands = self.execute_failure(fail_at)
        self.assertEqual(result.returncode, expected_exit, result.stderr)
        self.assertTrue(any(cmd.endswith(" stop") for cmd in commands), commands)
        self.assertTrue(any(cmd.endswith(" up -d") for cmd in commands), commands)
        self.assertFalse(any("SHOULD_NOT_REACH_HERE" in cmd for cmd in commands))

    def test_tar_backup_failure_restarts_original(self) -> None:
        self.assert_restart_attempted("tar", 42)

    def test_identity_volume_failure_restarts_original(self) -> None:
        self.assert_restart_attempted("volume", 47)

    def test_partial_stop_failure_attempts_recovery(self) -> None:
        self.assert_restart_attempted("stop", 39)

    def test_restart_failure_is_visible_without_modifying_state(self) -> None:
        result, commands = self.execute_failure("restart")
        self.assertEqual(result.returncode, 42, result.stderr)
        self.assertTrue(any(cmd.endswith(" up -d") for cmd in commands))
        self.assertIn("Automatic restart failed", result.stderr)


if __name__ == "__main__":
    unittest.main()
