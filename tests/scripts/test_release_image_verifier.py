"""Read-only CLI regression tests. Fake crane/cosign never prove cryptography."""
import importlib.util
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "scripts" / "verify-release-images.py"
IMAGES = ("mcp-vps-agent-gateway", "mcp-vps-agent-broker", "portico-cloud-node")
DIGEST = "sha256:" + "a" * 64


class ImageReleaseVerificationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.log = self.root / "calls.log"
        crane = self.bin / "crane"
        crane.write_text(
            "#!/usr/bin/env python3\n"
            "import os,sys\n"
            "from pathlib import Path\n"
            "args=sys.argv[1:]\n"
            "assert args[0]=='digest' and len(args)==2\n"
            "with open(os.environ['VERIFICATION_TEST_LOG'],'a') as f: "
            "f.write('crane '+args[1]+'\\n')\n"
            "if os.environ.get('FAIL_CRANE_ON') and os.environ['FAIL_CRANE_ON'] in args[1]: sys.exit(3)\n"
            "print(os.environ.get('FAKE_DIGEST','sha256:'+'a'*64))\n"
        )
        crane.chmod(0o700)
        cosign = self.bin / "cosign"
        cosign.write_text(
            "#!/usr/bin/env python3\n"
            "import os,sys\n"
            "args=sys.argv[1:]\n"
            "assert args[0]=='verify' and len(args)==6\n"
            "assert '@sha256:' in args[1] and ':' not in args[1].split('@',1)[0]\n"
            "assert args[2]=='--certificate-identity-regexp'\n"
            "assert args[4]=='--certificate-oidc-issuer'\n"
            "assert args[5]=='https://token.actions.githubusercontent.com'\n"
            "assert 'tags/v0\\\\.1\\\\.0\\\\-rc\\\\.7' in args[3]\n"
            "assert 'heads/main' in args[3]\n"
            "with open(os.environ['VERIFICATION_TEST_LOG'],'a') as f: "
            "f.write('cosign '+args[1]+'\\n')\n"
            "if os.environ.get('FAIL_COSIGN_ON') and os.environ['FAIL_COSIGN_ON'] in args[1]: sys.exit(4)\n"
            "print('verified by fake cosign')\n"
        )
        cosign.chmod(0o700)
        self.env = dict(os.environ)
        self.env["PATH"] = str(self.bin) + os.pathsep + self.env["PATH"]
        self.env["VERIFICATION_TEST_LOG"] = str(self.log)

    def run_verify(self, tag="v0.1.0-rc.7", extra_env=None):
        env = dict(self.env, **(extra_env or {}))
        return subprocess.run(
            [sys.executable, str(SCRIPT), "--tag", tag], env=env,
            capture_output=True, text=True, timeout=15,
        )

    def test_all_three_images_resolve_and_verify_by_digest(self):
        result = self.run_verify()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("RELEASE IMAGES VERIFIED", result.stdout)
        self.assertEqual(result.stdout.count("@sha256:"), 3)
        lines = self.log.read_text().splitlines()
        self.assertEqual(len(lines), 6)
        for n, image in enumerate(IMAGES):
            self.assertIn(image+":v0.1.0-rc.7", lines[2*n])
            self.assertIn(image+"@"+DIGEST, lines[2*n+1])

    def test_forged_digest_is_rejected_before_signature_verification(self):
        result = self.run_verify(extra_env={"FAKE_DIGEST": "sha256:not-a-digest"})
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("invalid immutable digest", result.stderr)
        self.assertEqual(len(self.log.read_text().splitlines()), 1)
        self.assertNotIn("RELEASE IMAGES VERIFIED", result.stdout)

    def test_missing_image_fails_closed_with_no_overall_success(self):
        result = self.run_verify(extra_env={"FAIL_CRANE_ON": "mcp-vps-agent-broker"})
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("digest lookup", result.stderr)
        self.assertNotIn("RELEASE IMAGES VERIFIED", result.stdout)
        self.assertEqual(len(self.log.read_text().splitlines()), 3)

    def test_failed_signature_blocks_release(self):
        result = self.run_verify(extra_env={"FAIL_COSIGN_ON": "portico-cloud-node"})
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Sigstore signature", result.stderr)
        self.assertNotIn("RELEASE IMAGES VERIFIED", result.stdout)
        self.assertEqual(len(self.log.read_text().splitlines()), 6)

    def test_invalid_tag_is_rejected_without_network(self):
        result = self.run_verify(tag="v0.1.0;rm -rf /")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("release tag", result.stderr)
        self.assertFalse(self.log.exists())

    def test_identity_permits_only_requested_tag_and_main(self):
        spec = importlib.util.spec_from_file_location("portico_image_verifier", SCRIPT)
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        identity = module.identity_for_tag("v0.1.0-rc.7")
        prefix = ("https://github.com/josemirmoura/mcp-vps-agent-gateway/"
                  ".github/workflows/release.yml@refs/")
        self.assertIsNotNone(re.fullmatch(identity,prefix+"tags/v0.1.0-rc.7"))
        self.assertIsNotNone(re.fullmatch(identity,prefix+"heads/main"))
        self.assertIsNone(re.fullmatch(identity,prefix+"tags/v0.1.0-rc.8"))
        self.assertIsNone(re.fullmatch(identity,prefix+"heads/dev"))
        self.assertIsNone(re.fullmatch(identity,prefix.replace("josemirmoura","attacker")
                                       +"tags/v0.1.0-rc.7"))


if __name__ == "__main__":
    unittest.main()
