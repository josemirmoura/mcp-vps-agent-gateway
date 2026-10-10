"""Release workflow regression: no tag/image publication before security gates.

This is a static contract test, not a replacement for executing the scanners,
checking branch rules, or verifying real published Sigstore signatures.
"""
from pathlib import Path
import os
import re
import subprocess
import unittest

WORKFLOW = Path(__file__).resolve().parents[2] / ".github/workflows/release.yml"


def job(name: str, contents: str) -> str:
    lines = contents.splitlines(keepends=True)
    expected = f"  {name}:"
    start = next((i for i, line in enumerate(lines) if line.strip("\r\n") == expected), None)
    if start is None:
        raise AssertionError(f"missing required release job: {name}")
    end = next(
        (i for i in range(start + 1, len(lines))
         if re.fullmatch(r"  [a-zA-Z0-9_-]+:", lines[i].strip("\r\n"))),
        len(lines),
    )
    return "".join(lines[start:end])

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
        self.assertIn("git diff --exit-code -- go.mod go.sum", validate)
        self.assertIn("needs: [validate, security, image-security]", prepare)
        self.assertIn("git merge-base --is-ancestor", validate)
        images = job("image-security", source)
        self.assertIn("target: [gateway, broker, cloud-node]", images)
        self.assertIn("arch: [amd64, arm64]", images)
        self.assertIn("trivy-action@v0.36.0", images)
        self.assertIn("exit-code: '1'", images)
        for expected in ("govulncheck", "gosec", "gitleaks"):
            self.assertIn(expected, security)
        for forbidden in ("continue-on-error:", "|| true", "exit 0"):
            self.assertNotIn(forbidden, security)

    def test_release_outputs_cannot_skip_tag_gate(self):
        source = self.workflow
        self.assertIn("needs: prepare-tag", job("images", source))
        self.assertIn("Scan exactly the published immutable image digest", job("images", source))
        self.assertIn("image-ref: ghcr.io/", job("images", source))
        self.assertIn("@${{ steps.build.outputs.digest }}", job("images", source))
        self.assertIn("needs: images", job("github-release", source))
        self.assertIn("needs: [validate, security, image-security]", job("prepare-tag", source))

        prepare = job("prepare-tag", source)
        published = job("github-release", source)
        self.assertNotIn("git push origin", prepare)
        self.assertIn("git push origin", published)
        self.assertLess(published.index("Keyless-sign and verify source artifacts"),
                        published.index("Create tag only after all signed assets"))
        self.assertIn("dist/RELEASE-PROVENANCE.json.sigstore.json", published)

    def test_token_permissions_are_job_scoped(self):
        before_jobs, rest = self.workflow.split("\njobs:\n", 1)
        self.assertIn("permissions:\n  contents: read", before_jobs)
        self.assertNotIn("packages: write", before_jobs)
        self.assertNotIn("id-token: write", before_jobs)
        # The publication-visible tag is created in the final job, after signing.
        self.assertIn("contents: read", job("prepare-tag", self.workflow))
        self.assertIn("contents: write", job("github-release", self.workflow))
        self.assertIn("packages: write", job("images", self.workflow))
        self.assertIn("id-token: write", job("images", self.workflow))
        self.assertIn("id-token: write", job("github-release", self.workflow))

    def test_release_signature_identity_anchored(self):
        self.assertIn("--certificate-oidc-issuer", self.workflow)
        self.assertIn('--certificate-identity "$identity"', self.workflow)
        self.assertNotIn("--certificate-identity-regexp", self.workflow)
        self.assertIn("cosign sign --yes", self.workflow)
        self.assertIn("cosign verify-blob", self.workflow)
        self.assertIn("git archive --format=tar", self.workflow)

    def test_release_cannot_publish_automatically_from_tag_push(self):
        event_section = self.workflow.split("\nconcurrency:", 1)[0]
        self.assertIn("\n  workflow_dispatch:\n", event_section)
        self.assertNotIn("\n  push:\n", event_section)
        for value in ("approved_sha:", "publish_confirmation:"):
            self.assertIn(value, event_section)

    def test_owner_release_request_is_checked_before_build(self):
        validate = job("validate", self.workflow)
        before_checkout = validate.split("      - uses: actions/checkout@v7", 1)[0]
        for required in (
            'test "$GITHUB_EVENT_NAME" = \'workflow_dispatch\'',
            'test "$GITHUB_ACTOR" = \'josemirmoura\'',
            'test "$GITHUB_TRIGGERING_ACTOR" = \'josemirmoura\'',
            'test "$GITHUB_REF" = \'refs/heads/main\'',
            'test "$APPROVED_SHA" = "$GITHUB_SHA"',
            'test "$PUBLISH_CONFIRMATION" = "PUBLICAR $RELEASE_TAG"',
            '[[ "$APPROVED_SHA" =~ ^[0-9a-f]{40}$ ]]',
        ):
            self.assertIn(required, before_checkout)
        self.assertIn('main_sha="$(git rev-parse refs/remotes/origin/main)"', validate)
        self.assertIn('test "$GITHUB_SHA" = "$main_sha"', validate)

    def test_owner_release_request_negative_paths(self):
        # Execute the actual inline bash owner gate with simulated GitHub inputs.
        validate = job("validate", self.workflow)
        start_marker = "      - name: Validate explicit owner publication request\n"
        self.assertIn(start_marker, validate)
        gate = validate.split(start_marker, 1)[1].split(
            "      - uses: actions/checkout@v7", 1)[0]
        script_part = gate.split("        run: |\n", 1)[1]
        script = "\n".join(
            line[10:] if line.startswith("          ") else line
            for line in script_part.splitlines()
        )
        commit = "a" * 40
        environment = dict(os.environ,
            GITHUB_EVENT_NAME="workflow_dispatch",
            GITHUB_ACTOR="josemirmoura",
            GITHUB_TRIGGERING_ACTOR="josemirmoura",
            GITHUB_REF="refs/heads/main",
            GITHUB_SHA=commit,
            APPROVED_SHA=commit,
            RELEASE_TAG="v0.1.0-rc.7",
            PUBLISH_CONFIRMATION="PUBLICAR v0.1.0-rc.7",
        )
        def check(overrides, success):
            result = subprocess.run(
                ["bash", "-c", script], env=dict(environment, **overrides),
                capture_output=True, text=True, timeout=10, check=False)
            if success:
                self.assertEqual(result.returncode, 0, result.stderr)
            else:
                self.assertNotEqual(result.returncode, 0, overrides)
        check({}, True)
        for overrides in (
            {"GITHUB_EVENT_NAME": "push"},
            {"GITHUB_ACTOR": "unapproved"},
            {"GITHUB_TRIGGERING_ACTOR": "unapproved"},
            {"GITHUB_REF": "refs/heads/release-test"},
            {"GITHUB_SHA": "b" * 40},
            {"APPROVED_SHA": ""},
            {"APPROVED_SHA": "a" * 39},
            {"PUBLISH_CONFIRMATION": "PUBLICAR v0.1.0"},
        ):
            check(overrides, False)

    def test_concurrent_release_of_same_tag_is_serialized(self):
        self.assertIn("group: community-release-", self.workflow)
        self.assertIn("cancel-in-progress: false", self.workflow)


if __name__ == "__main__":
    unittest.main()
