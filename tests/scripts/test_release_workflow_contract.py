"""Release workflow regression: no tag/image publication before security gates.

This is a static contract test, not a replacement for executing the scanners,
checking branch rules, or verifying real published Sigstore signatures.
"""
from pathlib import Path
import re
import unittest

WORKFLOW = Path(__file__).resolve().parents[2] / ".github/workflows/release.yml"


def job(name: str, contents: str) -> str:
    match = re.search(
        rf"^  {re.escape(name)}:\\n(?:(?!^  [a-zA-Z0-9_-]+:).*(?:\\n|\\Z))*",
        contents,
        flags=re.MULTILINE,
    )
    if match is None:
        raise AssertionError(f"missing required release job: {name}")
    return match.group(0)


class ReleaseWorkflowSecurityContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.workflow = WORKFLOW.read_text(encoding="utf-8")

    def test_security_blocks_tag_creation(self):
        source = self.workflow
        validate = job("validate", source)
        security = job("security", source)
        prepare = job("prepare-tag", source)
        self.assertIn("go test -race -count=1 ./...", validate)
        self.assertIn("needs: [validate, security]", prepare)
        for expected in ("govulncheck", "gosec", "gitleaks"):
            self.assertIn(expected, security)
        for forbidden in ("continue-on-error:", "|| true", "exit 0"):
            self.assertNotIn(forbidden, security)

    def test_release_outputs_cannot_skip_tag_gate(self):
        source = self.workflow
        self.assertIn("needs: prepare-tag", job("images", source))
        self.assertIn("needs: images", job("github-release", source))
        self.assertIn("needs: [validate, security]", job("prepare-tag", source))

    def test_token_permissions_are_job_scoped(self):
        before_jobs, rest = self.workflow.split("\njobs:\n", 1)
        self.assertIn("permissions:\n  contents: read", before_jobs)
        self.assertNotIn("packages: write", before_jobs)
        self.assertNotIn("id-token: write", before_jobs)
        self.assertIn("contents: write", job("prepare-tag", self.workflow))
        self.assertIn("packages: write", job("images", self.workflow))
        self.assertIn("id-token: write", job("images", self.workflow))
        self.assertIn("id-token: write", job("github-release", self.workflow))

    def test_release_signature_identity_anchored(self):
        self.assertIn("--certificate-oidc-issuer", self.workflow)
        self.assertIn("--certificate-identity-regexp", self.workflow)
        self.assertIn("cosign sign --yes", self.workflow)
        self.assertIn("cosign verify-blob", self.workflow)
        self.assertIn("git archive --format=tar", self.workflow)

    def test_concurrent_release_of_same_tag_is_serialized(self):
        self.assertIn("group: community-release-", self.workflow)
        self.assertIn("cancel-in-progress: false", self.workflow)


if __name__ == "__main__":
    unittest.main()
