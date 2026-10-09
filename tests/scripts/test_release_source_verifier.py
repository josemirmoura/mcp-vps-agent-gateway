"""Unprivileged regression coverage for the source-release verification gate.

A fake cosign executable proves the verifier invokes both signature checks
with fixed issuer/identity and refuses negative cases; it does NOT simulate
cryptographic trust. Real Sigstore acceptance remains a separate release gate.
"""
import hashlib
import io
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts" / "verify-release-source.py"
ARCHIVE = "mcp-vps-agent-source-package.tar.gz"
SHA = ARCHIVE + ".sha256"


class ReleaseSourceVerifierTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name)
        self.dist = self.base / "release"
        self.dist.mkdir()
        self.bin = self.base / "bin"
        self.bin.mkdir()
        self.log = self.base / "cosign.log"
        fake = self.bin / "cosign"
        fake.write_text(
            "#!/usr/bin/env python3\n"
            "import os,sys\n"
            "from pathlib import Path\n"
            "args=sys.argv[1:]\n"
            "assert args[0]=='verify-blob'\n"
            "assert '--bundle' in args\n"
            "assert '--certificate-identity-regexp' in args\n"
            "assert '--certificate-oidc-issuer' in args\n"
            "assert 'https://token.actions.githubusercontent.com' in args\n"
            "assert 'github' in args[args.index('--certificate-identity-regexp')+1]\n"
            "with open(os.environ['FAKE_COSIGN_LOG'],'a') as f: f.write(args[1]+'\\n')\n"
            "if os.environ.get('FAKE_COSIGN_DENY')=='1': sys.exit(1)\n"
        )
        fake.chmod(0o700)
        self.env = dict(os.environ, PATH=str(self.bin) + os.pathsep + os.environ["PATH"],
                        FAKE_COSIGN_LOG=str(self.log))
        self.source_version = "0.1.0-rc.7"
        self.archive = self.dist / ARCHIVE
        self.create_package(self.source_version)

    def create_package(self, version):
        with tarfile.open(self.archive, "w:gz") as archive:
            payload = (version+"\n").encode()
            item = tarfile.TarInfo("mcp-vps-agent/VERSION")
            item.size = len(payload)
            archive.addfile(item, io.BytesIO(payload))
        digest = hashlib.sha256(self.archive.read_bytes()).hexdigest()
        (self.dist / SHA).write_text(f"{digest}  dist/{ARCHIVE}\n")
        for suffix in [ARCHIVE, SHA]:
            (self.dist / (suffix + ".sigstore.json")).write_text('{"fixture":"not a real signature"}')

    def execute(self, tag="v0.1.0-rc.7", env=None):
        return subprocess.run(
            [sys.executable, str(SCRIPT), "--directory", str(self.dist), "--tag", tag],
            env=env or self.env, capture_output=True, text=True, timeout=12,
        )

    def test_valid_contract_calls_both_signature_verifications(self):
        result = self.execute()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("RELEASE SOURCE VERIFIED", result.stdout)
        checked = self.log.read_text().splitlines()
        self.assertEqual(len(checked), 2)
        self.assertTrue(checked[0].endswith(SHA))
        self.assertTrue(checked[1].endswith(ARCHIVE))

    def test_mutated_archive_rejected_after_verifying_checksum_signature(self):
        self.archive.write_bytes(self.archive.read_bytes()+b"changed")
        result = self.execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("SHA-256", result.stderr)
        self.assertEqual(len(self.log.read_text().splitlines()), 1)

    def test_wrong_expected_tag_is_rejected(self):
        result = self.execute(tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("VERSION", result.stderr)

    def test_missing_signature_bundle_fails_closed(self):
        (self.dist / (ARCHIVE+".sigstore.json")).unlink()
        result = self.execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("missing", result.stderr)
        self.assertFalse(self.log.exists())

    def test_checksum_rejects_unexpected_file_and_multiple_entries(self):
        digest = hashlib.sha256(self.archive.read_bytes()).hexdigest()
        (self.dist / SHA).write_text(f"{digest}  /etc/passwd\n{digest}  dist/{ARCHIVE}\n")
        result = self.execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("exactly one", result.stderr)

    def test_symlinked_asset_fails_closed(self):
        checksum = self.dist / SHA
        target = self.base / "evil.sha256"
        target.write_bytes(checksum.read_bytes())
        checksum.unlink()
        checksum.symlink_to(target)
        result = self.execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("symlinked", result.stderr)
        self.assertFalse(self.log.exists())

    def test_sigstore_failure_blocks_even_matching_checksum(self):
        env = dict(self.env, FAKE_COSIGN_DENY="1")
        result = self.execute(env=env)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Sigstore rejected", result.stderr)

    def test_duplicate_version_entry_is_rejected(self):
        with tarfile.open(self.archive, "w:gz") as pack:
            for version in ["0.1.0-rc.7", "0.1.0-evil"]:
                payload = (version + "\n").encode()
                item = tarfile.TarInfo("mcp-vps-agent/VERSION")
                item.size = len(payload)
                pack.addfile(item, io.BytesIO(payload))
        self.resign_fixture_checksum()
        result = self.execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("duplicate VERSION", result.stderr)

    def test_archive_with_escaping_member_is_rejected(self):
        with tarfile.open(self.archive, "w:gz") as pack:
            payload = b"0.1.0-rc.7\n"
            item = tarfile.TarInfo("mcp-vps-agent/VERSION")
            item.size = len(payload)
            pack.addfile(item, io.BytesIO(payload))
            dangerous = tarfile.TarInfo("mcp-vps-agent/../../override")
            dangerous.size = 1
            pack.addfile(dangerous, io.BytesIO(b"x"))
        self.resign_fixture_checksum()
        result = self.execute()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("unexpected member path", result.stderr)

    def resign_fixture_checksum(self):
        digest = hashlib.sha256(self.archive.read_bytes()).hexdigest()
        (self.dist / SHA).write_text(f"{digest}  dist/{ARCHIVE}\n")

    def test_untrusted_tag_format_fails_before_cosign(self):
        result = self.execute(tag="v0.1.0;echo injected")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("expected tag", result.stderr)
        self.assertFalse(self.log.exists())


if __name__ == "__main__":
    unittest.main()
