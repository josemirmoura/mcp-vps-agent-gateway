"""Fail-closed tests for the guided installer's pre-mutation phase.

Uses the exact installer source until init_args, then exits before init,
Docker, OAuth, and any live host changes. Fake preflight, UID and directory
creation commands run exclusively inside a disposable TemporaryDirectory.
"""
from __future__ import annotations

import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
INSTALLER = ROOT / "scripts" / "install.sh"
PRODUCT = ROOT / "scripts" / "lib" / "product.sh"
BOUNDARY = 'init_args=(--scope "$SCOPE" --run-as "$RUN_AS" --lang "$VPS_AGENT_LANG")'


class FirstRunPreMutationTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory(prefix="portico-first-run-")
        self.addCleanup(temp.cleanup)
        self.workdir = Path(temp.name)
        scripts = self.workdir / "scripts"
        (scripts / "lib").mkdir(parents=True)
        source = INSTALLER.read_text(encoding="utf-8")
        self.assertEqual(source.count(BOUNDARY), 1, "Installer pre-mutation boundary changed")
        # Copy, never source or execute a live repo's installer.
        (scripts / "install.sh").write_text(
            source.split(BOUNDARY, 1)[0]
            + '\nprintf "REACHED_INIT_ARGS\\n"\n',
            encoding="utf-8",
        )
        shutil.copy2(PRODUCT, scripts / "lib" / "product.sh")
        (self.workdir / "VERSION").write_text("0.1.0-rc.7\n", encoding="utf-8")
        self.commands = self.workdir / "commands.log"

        bin_dir = self.workdir / "fake-bin"
        bin_dir.mkdir()
        self._fake(bin_dir / "python3", '#!/bin/sh\nexit 0\n')
        self._fake(
            bin_dir / "id",
            """#!/bin/sh
case "$1" in
  -un) echo "${FAKE_INVOKER:-root}" ;;
  -u)
    if [ "$#" -eq 1 ]; then
      echo "${FAKE_CURRENT_UID:-0}"
    elif [ "$2" = aliasroot ]; then
      echo 0
    elif [ "$2" = invaliduser ]; then
      exit 1
    else
      echo 1001
    fi ;;
  -gn) echo appgroup ;;
  *) exit 2 ;;
esac
""",
        )
        for name in ("sudo", "install"):
            self._fake(
                bin_dir / name,
                '#!/bin/sh\nprintf "%s %s\\n" "' + name + '" "$*" >> "$TEST_INSTALL_LOG"\n',
            )
        self.env = {
            **os.environ,
            "PATH": str(bin_dir) + os.pathsep + os.environ.get("PATH", "/usr/bin:/bin"),
            "TEST_INSTALL_LOG": str(self.commands),
            "FAKE_CURRENT_UID": "0",
            "FAKE_INVOKER": "root",
            "VPS_AGENT_LANG": "pt-BR",
            "VPS_AGENT_LANG_EXPLICIT": "1",
        }
        # Reproduce minimal root shell environment; no USER dependency.
        self.env.pop("USER", None)
        self.scope = str(self.workdir / "new-project")

    def _fake(self, path: Path, contents: str):
        path.write_text(contents, encoding="utf-8")
        path.chmod(0o755)

    def run_installer(self, *args: str, current_uid: str = "0"):
        env = {**self.env, "FAKE_CURRENT_UID": current_uid}
        return subprocess.run(
            ["bash", "scripts/install.sh", *args, "--local-only"],
            cwd=self.workdir, env=env, text=True, input="", capture_output=True,
            timeout=15, check=False,
        )

    def assert_no_scope_creation(self):
        self.assertFalse(self.commands.exists(), "Host directory creation invoked")
        self.assertFalse((self.workdir / ".env").exists(), "Operator state created")
        self.assertFalse((self.workdir / "config" / "policy.yaml").exists())

    def test_missing_yes_fails_before_create_scope(self):
        proc = self.run_installer("--profile", "project", "--scope", self.scope,
                                  "--create-scope", "--run-as", "deploy")
        self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
        self.assertIn("--yes", proc.stderr)
        self.assertNotIn("REACHED_INIT_ARGS", proc.stdout)
        self.assert_no_scope_creation()

    def test_missing_whole_host_ack_fails_before_state(self):
        proc = self.run_installer("--profile", "whole-host", "--yes", "--run-as", "deploy")
        self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
        self.assertIn("--ack-whole-host", proc.stderr)
        self.assert_no_scope_creation()

    def test_custom_scope_root_requires_explicit_whole_host(self):
        proc = self.run_installer("--profile", "custom", "--scope", "/",
                                  "--run-as", "deploy", "--yes")
        self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
        self.assertIn("Whole Host", proc.stderr)
        self.assert_no_scope_creation()

    def test_whole_host_rejects_ignored_custom_scope(self):
        proc = self.run_installer("--profile", "whole-host", "--scope", self.scope,
                                  "--yes", "--ack-whole-host", "--run-as", "deploy")
        self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
        self.assertIn("--scope", proc.stderr)
        self.assert_no_scope_creation()

    def test_rejects_unsupported_language(self):
        proc = self.run_installer("--profile", "custom", "--scope", "/opt",
                                  "--yes", "--run-as", "deploy", "--lang", "fr")
        self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
        self.assertIn("--lang", proc.stderr)
        self.assert_no_scope_creation()

    def test_rejects_empty_explicit_scope(self):
        proc = self.run_installer("--profile", "project", "--scope", "",
                                  "--yes", "--run-as", "deploy")
        self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
        self.assertIn("--scope", proc.stderr)
        self.assert_no_scope_creation()

    def test_rejects_uid_zero_alias(self):
        proc = self.run_installer("--profile", "project", "--scope", self.scope,
                                  "--yes", "--create-scope", "--run-as", "aliasroot")
        self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
        self.assertIn("aliasroot", proc.stderr)
        self.assert_no_scope_creation()

    def test_rejects_unknown_shell_user(self):
        proc = self.run_installer("--profile", "project", "--scope", self.scope,
                                  "--yes", "--create-scope", "--run-as", "invaliduser")
        self.assertEqual(proc.returncode, 2, proc.stdout + proc.stderr)
        self.assertIn("invaliduser", proc.stderr)
        self.assert_no_scope_creation()

    def test_root_creates_missing_scope_for_nonroot_owner_without_sudo(self):
        proc = self.run_installer("--profile", "project", "--scope", self.scope,
                                  "--yes", "--create-scope", "--run-as", "deploy")
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertIn("REACHED_INIT_ARGS", proc.stdout)
        self.assertEqual(
            self.commands.read_text().strip(),
            f"install -d -o deploy -g appgroup -m 0750 -- {self.scope}",
        )
        self.assertFalse((self.workdir / ".env").exists())

    def test_nonroot_create_scope_uses_sudo_for_requested_owner(self):
        proc = self.run_installer("--profile", "project", "--scope", self.scope,
                                  "--yes", "--create-scope", "--run-as", "deploy",
                                  current_uid="1002")
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertIn("REACHED_INIT_ARGS", proc.stdout)
        self.assertEqual(
            self.commands.read_text().strip(),
            f"sudo install -d -o deploy -g appgroup -m 0750 -- {self.scope}",
        )


if __name__ == "__main__":
    unittest.main()
