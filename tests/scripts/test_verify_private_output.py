#!/usr/bin/env python3
"""Hermetic tests for secure verifier temporary-file lifecycle.

No network, containers, existing .env or machine-level configuration required.
"""
import os
import pathlib
import shutil
import stat
import subprocess
import tempfile
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[2]

MOCK_CURL = r"""#!/usr/bin/env python3
import json
import os
import pathlib
import stat
import sys

args = sys.argv[1:]
url = next((x for x in args if x.startswith("https://")), "")
output = args[args.index("--output") + 1] if "--output" in args else None
headers = args[args.index("--dump-header") + 1] if "--dump-header" in args else None
base = os.environ["VPS_AGENT_VERIFY_DIR"]
assert stat.S_IMODE(os.stat(base).st_mode) == 0o700, "verifier directory must be private"
trace = os.environ.get("MOCK_TRACE")
if trace:
    with open(trace, "a", encoding="utf-8") as fh:
        fh.write(base + "\n")
if url.endswith("/healthz"):
    data = {"ok": True}
elif url.endswith("/.well-known/oauth-protected-resource"):
    data = {
        "resource": "https://portico.example.invalid/mcp",
        "authorization_servers": ["https://id.example.invalid"],
        "bearer_methods_supported": ["header"],
        "scopes_supported": ["openid", "urn:zitadel:iam:org:project:id:12345:aud"],
    }
    if os.environ.get("MOCK_BAD_METADATA") == "1":
        data["resource"] = "https://wrong.example.invalid/mcp"
elif url.endswith("/.well-known/openid-configuration"):
    data = {
        "issuer": "https://id.example.invalid",
        "authorization_endpoint": "https://id.example.invalid/authorize",
        "token_endpoint": "https://id.example.invalid/token",
        "registration_endpoint": "https://id.example.invalid/register",
        "code_challenge_methods_supported": ["S256"],
        "scopes_supported": ["offline_access"],
        "grant_types_supported": ["authorization_code", "refresh_token"],
        "token_endpoint_auth_methods_supported": ["none"],
    }
elif url == "https://portico.example.invalid/mcp":
    data = ""
    if headers:
        pathlib.Path(headers).write_text(
            'HTTP/1.1 401 Unauthorized\n'
            'WWW-Authenticate: Bearer resource_metadata='
            '"https://portico.example.invalid/.well-known/oauth-protected-resource"\n',
            encoding="utf-8",
        )
else:
    print("unexpected curl URL: " + url, file=sys.stderr)
    sys.exit(2)
payload = json.dumps(data) if not isinstance(data, str) else data
if output:
    pathlib.Path(output).write_text(payload, encoding="utf-8")
else:
    print(payload, end="")
if "--write-out" in args:
    print("401", end="")
"""

MOCK_DOCKER = r"""#!/usr/bin/env python3
import json
import sys
args = sys.argv[1:]
if args[:2] == ["compose", "config"]:
    pass
elif args[:3] == ["compose", "ps", "-a"]:
    print("mock-container-id")
elif args[:2] == ["compose", "ps"]:
    print("mock-broker healthy\nmock-gateway healthy")
elif args[:2] == ["inspect", "-f"]:
    print("healthy")
elif args[:3] == ["compose", "exec", "-T"] and "broker" in args:
    print(json.dumps({"ok": True, "result": {"valid": True}}))
elif args[:3] == ["compose", "exec", "-T"] and "gateway" in args:
    print(json.dumps({"active": False}))
else:
    print("unexpected docker command: " + repr(args), file=sys.stderr)
    sys.exit(2)
"""


class PrivateVerificationOutputTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="portico-test-verification-")
        self.addCleanup(self.tmp.cleanup)
        self.home = pathlib.Path(self.tmp.name)
        (self.home / "scripts" / "lib").mkdir(parents=True)
        (self.home / "bin").mkdir()
        (self.home / "temp").mkdir()
        for relative in (
            "scripts/verify.sh",
            "scripts/verify-public.sh",
            "scripts/lib/product.sh",
            "scripts/lib/oauth-scopes.sh",
        ):
            dst = self.home / relative
            shutil.copyfile(ROOT / relative, dst)
        (self.home / ".env").write_text(
            "VPS_AGENT_LANG=en\n"
            "VPS_AGENT_AUTH_MODE=integrated\n"
            "VPS_AGENT_PUBLIC_URL=https://portico.example.invalid/mcp\n"
            "VPS_AGENT_OIDC_ISSUER=https://id.example.invalid\n"
            "VPS_AGENT_INTEGRATED_AUDIENCE_PROJECT_ID=12345\n"
            "VPS_AGENT_REQUIRED_SCOPES='openid urn:zitadel:iam:org:project:id:12345:aud'\n",
            encoding="utf-8",
        )
        for name, content in (("curl", MOCK_CURL), ("docker", MOCK_DOCKER)):
            binary = self.home / "bin" / name
            binary.write_text(content, encoding="utf-8")
            binary.chmod(0o700)
        self.env = os.environ.copy()
        self.env.update({
            "PATH": str(self.home / "bin") + os.pathsep + self.env.get("PATH", ""),
            "TMPDIR": str(self.home / "temp"),
            "MOCK_TRACE": str(self.home / "trace"),
            "VPS_AGENT_LANG": "en",
        })

    def launch(self, script, env=None):
        return subprocess.run(
            ["bash", "scripts/" + script],
            cwd=self.home,
            env=env or self.env,
            capture_output=True,
            text=True,
            timeout=30,
            check=False,
        )

    def assert_private_temp_cleanup(self):
        paths = (self.home / "trace").read_text(encoding="utf-8").splitlines()
        self.assertTrue(paths, "test did not exercise temporary verification output")
        self.assertTrue(all(pathlib.Path(p).parent == self.home / "temp" for p in paths))
        self.assertFalse(
            any((self.home / "temp").iterdir()),
            "temporary output must be removed even after unsuccessful verification",
        )

    def test_local_verifier_passes_and_cleans_up(self):
        run = self.launch("verify.sh")
        self.assertEqual(run.returncode, 0, run.stderr + run.stdout)
        self.assertIn("LOCAL MCP VERIFICATION: PASS", run.stdout)
        self.assert_private_temp_cleanup()

    def test_public_verifier_passes_and_cleans_up(self):
        run = self.launch("verify-public.sh")
        self.assertEqual(run.returncode, 0, run.stderr + run.stdout)
        self.assertIn("UNAUTHENTICATED MCP DENIAL + OAUTH CHALLENGE: PASS", run.stdout)
        self.assert_private_temp_cleanup()

    def test_failure_cleans_up(self):
        env = dict(self.env, MOCK_BAD_METADATA="1")
        run = self.launch("verify-public.sh", env)
        self.assertNotEqual(run.returncode, 0, "bad resource metadata must be rejected")
        self.assert_private_temp_cleanup()

    def test_parallel_verifiers_never_share_output(self):
        commands = [["bash", "scripts/verify-public.sh"]] * 2
        processes = [
            subprocess.Popen(
                argv, cwd=self.home, env=self.env,
                stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
            )
            for argv in commands
        ]
        for process in processes:
            output, error = process.communicate(timeout=30)
            self.assertEqual(process.returncode, 0, error + output)
        self.assert_private_temp_cleanup()
        paths = (self.home / "trace").read_text(encoding="utf-8").splitlines()
        self.assertGreaterEqual(len(set(paths)), 2, "parallel runs reused a temporary directory")


if __name__ == "__main__":
    unittest.main()
