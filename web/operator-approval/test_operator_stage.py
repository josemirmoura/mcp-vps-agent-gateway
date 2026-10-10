"""Fail-closed staging checks; no production network, Docker, credentials or worktree."""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


HERE = Path(__file__).resolve().parent
SCRIPT = HERE.parents[1] / "scripts" / "operator-portal-stage.sh"
BASE = "a" * 40
FETCHED = "b" * 40


class OperatorStagingPinning(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="portico-operator-stage-test-")
        self.addCleanup(self.tmp.cleanup)
        self.path = Path(self.tmp.name)
        self.checkout = self.path / "checkout"
        self.checkout.mkdir()
        (self.checkout / ".git").mkdir()
        (self.checkout / "state").mkdir()
        (self.checkout / ".env").write_text("FIXTURE_ONLY=1\n")
        (self.checkout / "compose.yaml").write_text("services: {}\n")
        self.parent = self.path / "stage-outside-checkout"
        self.log = self.path / "fake-git.log"
        bindir = self.path / "bin"
        bindir.mkdir()
        git = bindir / "git"
        git.write_text("""#!/usr/bin/env bash
printf '%s\\n' "$*" >> "$PORTICO_STAGE_TEST_LOG"
case "$1" in
 status) exit 0 ;;
 rev-parse)
  if [[ "$2" == "HEAD" ]]; then echo "$PORTICO_STAGE_TEST_BASE"
  elif [[ "$2" == "--verify" && "$3" == 'FETCH_HEAD^{commit}' ]]; then echo "$PORTICO_STAGE_TEST_FETCHED"
  else exit 80; fi ;;
 check-ref-format) exec "$PORTICO_STAGE_TEST_REAL_GIT" check-ref-format "$2" ;;
 fetch|merge-base) exit 0 ;;
 worktree) exit 77 ;; # never actually create a checkout or run the candidate's code
 *) exit 79 ;;
esac
""")
        git.chmod(0o700)
        real_git = shutil.which("git")
        if not real_git:
            self.skipTest("Git is necessary to test reference validation")
        self.env = os.environ.copy()
        self.env.update(
            PATH=str(bindir) + os.pathsep + self.env.get("PATH", ""),
            HOME=str(self.path),
            PORTICO_STAGE_PARENT=str(self.parent),
            PORTICO_STAGE_TEST_LOG=str(self.log),
            PORTICO_STAGE_TEST_BASE=BASE,
            PORTICO_STAGE_TEST_FETCHED=FETCHED,
            PORTICO_STAGE_TEST_REAL_GIT=real_git,
        )
        self.env.pop("PORTICO_STAGE_CANDIDATE_SHA", None)
        self.env.pop("PORTICO_STAGE_CANDIDATE_REF", None)

    def run_stage(self, **values):
        env = {**self.env, **values}
        return subprocess.run(["bash", str(SCRIPT)], cwd=self.checkout,
                              env=env, capture_output=True, text=True, timeout=10)

    def commands(self):
        return self.log.read_text().splitlines() if self.log.exists() else []

    def test_no_sha_must_fail_before_fetch_and_snapshot(self):
        result = self.run_stage()
        self.assertEqual(result.returncode, 2, result.stderr)
        self.assertIn("PORTICO_STAGE_CANDIDATE_SHA", result.stderr)
        self.assertFalse(any(line.startswith("fetch ") for line in self.commands()))
        self.assertFalse(self.parent.exists())

    def test_invalid_sha_or_reference_rejected_before_fetch(self):
        for values in (
            {"PORTICO_STAGE_CANDIDATE_SHA": "main"},
            {"PORTICO_STAGE_CANDIDATE_SHA": FETCHED.upper()},
            {"PORTICO_STAGE_CANDIDATE_SHA": FETCHED,
             "PORTICO_STAGE_CANDIDATE_REF": "refs/heads/../../other"},
            {"PORTICO_STAGE_CANDIDATE_SHA": FETCHED,
             "PORTICO_STAGE_CANDIDATE_REF": "-option"},
        ):
            with self.subTest(values=values):
                self.log.unlink(missing_ok=True)
                result = self.run_stage(**values)
                self.assertEqual(result.returncode, 2, result.stderr)
                self.assertFalse(any(line.startswith("fetch ") for line in self.commands()))
                self.assertFalse(self.parent.exists())

    def test_moving_branch_sha_mismatch_fails_before_checkout_or_backup(self):
        result = self.run_stage(PORTICO_STAGE_CANDIDATE_SHA=BASE)
        self.assertEqual(result.returncode, 2, result.stderr)
        self.assertIn("divergiu do SHA revisado", result.stderr)
        self.assertTrue(any(line.startswith(
            "fetch --quiet --no-tags origin refs/heads/integration/community-stable-block5-20261009"
        ) for line in self.commands()))
        self.assertFalse(any(line.startswith("worktree ") for line in self.commands()))
        self.assertFalse(self.parent.exists())

    def test_exact_match_passes_pin_and_only_then_attempts_worktree(self):
        result = self.run_stage(PORTICO_STAGE_CANDIDATE_SHA=FETCHED)
        self.assertEqual(result.returncode, 77, result.stderr)
        self.assertNotIn("divergiu", result.stderr)
        self.assertTrue(any(line.startswith("merge-base --is-ancestor") for line in self.commands()))
        self.assertTrue(any(line.startswith("worktree add --quiet --detach") and
                            line.endswith(FETCHED) for line in self.commands()))
        self.assertTrue(self.parent.is_dir())

    def test_explicit_tag_ref_is_accepted_only_at_pinned_sha(self):
        result = self.run_stage(PORTICO_STAGE_CANDIDATE_SHA=FETCHED,
                                PORTICO_STAGE_CANDIDATE_REF="refs/tags/v0.1.0-test")
        self.assertEqual(result.returncode, 77, result.stderr)
        self.assertIn("fetch --quiet --no-tags origin refs/tags/v0.1.0-test", self.commands())


if __name__ == "__main__":
    unittest.main()
