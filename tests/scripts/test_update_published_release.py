"""Security regressions for update publication trust. Mocks are NEVER cryptographic evidence."""
from __future__ import annotations

import gzip
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts/verify-update-publication.py"
ARCHIVE = "mcp-vps-agent-source-package.tar.gz"
TAG = "v0.1.0-rc.7"


def run(*args, cwd=None, check=True, env=None):
    return subprocess.run(args, cwd=cwd, env=env, text=True, capture_output=True,
                          check=check, timeout=12)


class PublishedReleasePreflightTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name)
        self.repo = self.base / "source"
        self.repo.mkdir()
        self.remote = self.base / "remote.git"
        run("git", "init", "-q", "--bare", str(self.remote))
        run("git", "init", "-q", str(self.repo))
        run("git", "config", "user.name", "Portico Fixture", cwd=self.repo)
        run("git", "config", "user.email", "fixture@example.invalid", cwd=self.repo)
        (self.repo / "VERSION").write_text("0.1.0-rc.7\n")
        (self.repo / "README.md").write_text("initial contents\n")
        self.commit_change()
        self.sha = run("git", "rev-parse", "HEAD", cwd=self.repo).stdout.strip()
        run("git", "remote", "add", "origin", str(self.remote), cwd=self.repo)
        run("git", "tag", TAG, cwd=self.repo)
        run("git", "push", "-q", "origin", "HEAD:refs/heads/main",
            "refs/tags/" + TAG, cwd=self.repo)

        self.dist = self.base / "assets"
        self.dist.mkdir()
        self.create_assets(self.sha)
        self.meta = self.base / "release.json"
        self.write_release()
        self.bin = self.base / "bin"
        self.bin.mkdir()
        self.make_command("gh", """
import json,os,sys,shutil
from pathlib import Path
args=sys.argv[1:]
if os.environ.get("MOCK_GH_DENY") == "1": sys.exit(1)
if args[:2] == ["release", "view"]:
    print(Path(os.environ["MOCK_RELEASE_META"]).read_text())
elif args[:2] == ["release", "download"]:
    folder=Path(args[args.index("--dir")+1])
    for asset in Path(os.environ["MOCK_ASSETS"]).iterdir():
        shutil.copyfile(asset,folder/asset.name)
else:
    sys.exit(33)
""")
        self.make_command("cosign", """
import os,sys
args=sys.argv[1:]
assert args[0] in ("verify-blob","verify")
assert "--certificate-oidc-issuer" in args
assert "https://token.actions.githubusercontent.com" in args
assert "--certificate-identity-regexp" in args
if os.environ.get("MOCK_SIGSTORE_DENY") == "1": sys.exit(1)
""")
        self.make_command("crane", """
import sys
assert sys.argv[1] == "digest"
print("sha256:"+"a"*64)
""")
        self.env = dict(os.environ, PATH=str(self.bin) + os.pathsep + os.environ["PATH"],
                        MOCK_ASSETS=str(self.dist), MOCK_RELEASE_META=str(self.meta))

    def make_command(self, name, body):
        target = self.bin / name
        target.write_text("#!" + sys.executable + "\n" + body)
        target.chmod(0o700)

    def commit_change(self):
        run("git", "add", ".", cwd=self.repo)
        run("git", "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
            "commit", "-qm", "fixture", cwd=self.repo)

    def create_assets(self, commit, signed_sha=None):
        # Capture binary git archive output directly, without Unicode decoding.
        raw = subprocess.run(["git", "archive", "--format=tar",
                              "--prefix=mcp-vps-agent/", commit],
                             cwd=self.repo, capture_output=True, check=True).stdout
        (self.dist / ARCHIVE).write_bytes(gzip.compress(raw, mtime=0))
        digest = hashlib.sha256((self.dist / ARCHIVE).read_bytes()).hexdigest()
        (self.dist / (ARCHIVE + ".sha256")).write_text(f"{digest}  dist/{ARCHIVE}\n")
        (self.dist / "RELEASE-PROVENANCE.json").write_text(json.dumps({
            "schema": 1, "tag": TAG, "commit_sha": signed_sha or commit,
            "source_sha256": digest,
        }))
        for name in (ARCHIVE, ARCHIVE + ".sha256", "RELEASE-PROVENANCE.json"):
            (self.dist / (name + ".sigstore.json")).write_text('{"test":true}')

    def write_release(self, *, draft=False, omitted=None, published="2026-10-10T09:00:00Z"):
        self.meta.write_text(json.dumps({
            "tagName": TAG, "isDraft": draft, "publishedAt": published,
            "assets": [{"name": p.name, "size": p.stat().st_size}
                       for p in self.dist.iterdir() if p.name != omitted],
        }))

    def verify(self, commit=None, env=None):
        return run(sys.executable, str(SCRIPT), "--tag", TAG,
                   "--commit", commit or self.sha, "--repo", str(self.repo),
                   check=False, env=env or self.env)

    def assert_blocked(self, result, text=None):
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertIn("UPDATE BLOCKED", result.stderr)
        if text:
            self.assertIn(text, result.stderr)

    def test_valid_signed_published_release_for_exact_commit(self):
        result = self.verify()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("RELEASE PREFLIGHT VERIFIED", result.stdout)

    def test_release_missing_or_unreachable_fails_closed(self):
        self.assert_blocked(self.verify(env={**self.env, "MOCK_GH_DENY": "1"}))

    def test_draft_and_unpublished_release_blocked(self):
        self.write_release(draft=True)
        self.assert_blocked(self.verify(), "draft")
        self.write_release(published=None)
        self.assert_blocked(self.verify(), "published")

    def test_missing_publication_asset_blocks_before_download(self):
        self.write_release(omitted="RELEASE-PROVENANCE.json.sigstore.json")
        self.assert_blocked(self.verify(), "missing or incomplete")

    def test_missing_signature_blocks(self):
        (self.dist / (ARCHIVE + ".sigstore.json")).unlink()
        self.write_release()
        self.assert_blocked(self.verify(), "assets missing or incomplete")

    def test_fake_signer_denial_blocks(self):
        self.assert_blocked(self.verify(env={**self.env, "MOCK_SIGSTORE_DENY": "1"}),
                            "signature")

    def test_signed_manifest_commit_mismatch_blocks(self):
        manifest = json.loads((self.dist / "RELEASE-PROVENANCE.json").read_text())
        manifest["commit_sha"] = "b" * 40
        (self.dist / "RELEASE-PROVENANCE.json").write_text(json.dumps(manifest))
        self.write_release()
        self.assert_blocked(self.verify(), "provenance differs")

    def test_tag_retargeted_after_publication_blocks(self):
        (self.repo / "README.md").write_text("retargeted\n")
        self.commit_change()
        run("git", "tag", "-f", TAG, cwd=self.repo)
        run("git", "push", "-q", "--force", "origin", "refs/tags/" + TAG, cwd=self.repo)
        self.assert_blocked(self.verify(), "remote tag")

    def test_package_other_commit_same_version_blocks(self):
        previous = self.sha
        (self.repo / "README.md").write_text("modified content, same VERSION\n")
        self.commit_change()
        latest = run("git", "rev-parse", "HEAD", cwd=self.repo).stdout.strip()
        run("git", "tag", "-f", TAG, cwd=self.repo)
        run("git", "push", "-q", "--force", "origin",
            "HEAD:refs/heads/main", "refs/tags/" + TAG, cwd=self.repo)
        # The bundle is ostensibly signed, and correctly records the newer SHA,
        # yet the actual package was created from the older tree.
        self.create_assets(previous, signed_sha=latest)
        self.write_release()
        self.assert_blocked(self.verify(commit=latest), "source package differs")

    def test_invalid_tag_or_commit_rejected(self):
        self.assert_blocked(self.verify(commit="main"), "SemVer tags")

    def test_rejected_release_never_stops_docker_or_creates_operational_backup(self):
        # Execute the real updater in a disposable Git checkout, with a
        # populated .env and policy, while GitHub release metadata is denied.
        (self.repo / "scripts/lib").mkdir(parents=True)
        shutil.copyfile(ROOT / "scripts/update.sh", self.repo / "scripts/update.sh")
        (self.repo / "scripts/lib/product.sh").write_text(
            'vps_agent_init_language() { :; }\n'
            'vps_agent_text() { printf "%s" "$1"; }\n'
            'vps_agent_banner() { :; }\n'
            'vps_agent_version() { printf "0.1.0-rc.7"; }\n'
        )
        self.commit_change()
        installed = run("git", "rev-parse", "HEAD", cwd=self.repo).stdout.strip()
        (self.repo / "RELEASE-MARKER").write_text("candidate\n")
        self.commit_change()
        candidate = run("git", "rev-parse", "HEAD", cwd=self.repo).stdout.strip()
        run("git", "tag", "-f", TAG, cwd=self.repo)
        run("git", "push", "-q", "--force", "origin",
            "refs/tags/" + TAG, cwd=self.repo)
        run("git", "reset", "--hard", installed, cwd=self.repo)
        (self.repo / ".env").write_text("PORTICO_TEST_ONLY=1\n")
        (self.repo / "config").mkdir()
        (self.repo / "config/policy.yaml").write_text("{}\n")
        docker_log = self.base / "docker.log"
        self.make_command("docker", """
import os,sys
with open(os.environ["MOCK_DOCKER_LOG"],"a") as handle:
    handle.write(" ".join(sys.argv[1:])+"\n")
sys.exit(83)
""")
        env = dict(self.env, VPS_AGENT_UPDATE_REF=TAG, MOCK_GH_DENY="1",
                   MOCK_DOCKER_LOG=str(docker_log))
        result = run("bash", "scripts/update.sh", cwd=self.repo,
                     env=env, check=False)
        self.assert_blocked(result)
        self.assertFalse(docker_log.exists(), "Docker was touched before release verification")
        self.assertEqual(run("git", "rev-parse", "HEAD",
                             cwd=self.repo).stdout.strip(), installed)
        self.assertFalse(list((self.repo / "backups").iterdir()),
                         "operational backups exist despite failed publication")
        self.assertNotEqual(installed, candidate)

    def test_timeout_on_required_helper_is_fatal(self):
        import importlib.util
        spec = importlib.util.spec_from_file_location("preflight_timeout", SCRIPT)
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        with self.assertRaises(module.Blocked):
            module.execute([sys.executable, "-c", "import time;time.sleep(4)"],
                           cwd=self.repo, timeout=1)


if __name__ == "__main__":
    unittest.main()
