"""Negative tests for redacted, original-evidence MCP Gate C manifest."""
import json
from pathlib import Path
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts"))
from mcp_gate_c_diagnostic import build
from mcp_release_scope_audit import REQUIRED_SERVER, REVISION, UPSTREAM

SHA = "a" * 40


def sample(folder):
    rows = []
    for index, name in enumerate(REQUIRED_SERVER):
        directory = folder / ("server-" + name + "-fixture")
        directory.mkdir()
        checks = [{"id": "check-" + str(index), "status": "SUCCESS"}]
        (directory / "checks.json").write_text(json.dumps(checks), encoding="utf-8")
        rows.append({
            "scenario": name, "classification": "PASS",
            "checks": {"success": 1},
            "evidence_files": [directory.name + "/checks.json"],
            "parse_errors": [],
        })
    return {
        "protocol_revision": REVISION, "upstream_commit": UPSTREAM, "role": "server",
        "required_server_scenarios": len(REQUIRED_SERVER),
        "client_role_classification": "NOT_APPLICABLE",
        "score": {"PASS": len(REQUIRED_SERVER), "FAIL": 0,
                  "SKIPPED": 0, "NOT_TESTED": 0},
        "runner_exit_code": 0, "release_gate_passed": True,
        "scenarios": rows,
    }


class GateCDiagnosticTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.matrix = sample(self.root)

    def test_good_original_check_evidence_has_37_rows(self):
        report = build(self.matrix, self.root, SHA)
        self.assertEqual(len(report["scenarios"]), 37)
        self.assertEqual(report["gateway_sha"], SHA)
        self.assertEqual(report["waivers_applied"], 0)

    def test_missing_original_check_file_is_rejected(self):
        (self.root / self.matrix["scenarios"][0]["evidence_files"][0]).unlink()
        with self.assertRaises(ValueError):
            build(self.matrix, self.root, SHA)

    def test_original_status_drift_is_rejected(self):
        f = self.root / self.matrix["scenarios"][0]["evidence_files"][0]
        f.write_text('[{"id":"changed","status":"FAILURE","errorMessage":"actual failure"}]')
        with self.assertRaises(ValueError):
            build(self.matrix, self.root, SHA)

    def test_no_waiver_for_real_failure(self):
        f = self.root / self.matrix["scenarios"][1]["evidence_files"][0]
        f.write_text('[{"id":"completion-complete","status":"FAILURE","errorMessage":"Failed: method not found: \\"completion/complete\\""}]')
        row = self.matrix["scenarios"][1]
        row["classification"] = "FAIL"
        row["checks"] = {"failure": 1}
        self.matrix["score"] = {"PASS": 36, "FAIL": 1, "SKIPPED": 0, "NOT_TESTED": 0}
        self.matrix["runner_exit_code"] = 1
        self.matrix["release_gate_passed"] = False
        report = build(self.matrix, self.root, SHA)
        self.assertFalse(report["release_gate_passed"])
        self.assertEqual(report["raw_score"]["FAIL"], 1)
        check = report["scenarios"][1]["checks"][0]
        self.assertEqual(check["error_kind"], "unadvertised-completion-method")
        self.assertNotIn("errorMessage", check)

    def test_invalid_upstream_is_rejected(self):
        self.matrix["upstream_commit"] = "not-pinned"
        with self.assertRaises(ValueError):
            build(self.matrix, self.root, SHA)

    def test_missing_or_duplicate_scenario_is_rejected(self):
        self.matrix["scenarios"][1]["scenario"] = self.matrix["scenarios"][0]["scenario"]
        with self.assertRaises(ValueError):
            build(self.matrix, self.root, SHA)

    def test_path_traversal_is_rejected(self):
        self.matrix["scenarios"][0]["evidence_files"] = ["../secret.json"]
        with self.assertRaises(ValueError):
            build(self.matrix, self.root, SHA)

    def test_malformed_source_sha_is_rejected(self):
        with self.assertRaises(ValueError):
            build(self.matrix, self.root, "moving-main")

    def test_unknown_check_state_is_rejected(self):
        f = self.root / self.matrix["scenarios"][0]["evidence_files"][0]
        f.write_text('[{"id":"unexpected","status":"BLOCKED"}]')
        with self.assertRaises(ValueError):
            build(self.matrix, self.root, SHA)

    def test_unsafe_message_never_reaches_report(self):
        secret = "Bearer synthetic-secret-to-redact"
        f = self.root / self.matrix["scenarios"][1]["evidence_files"][0]
        f.write_text(json.dumps([{"id":"error","status":"WARNING",
                                  "errorMessage":secret}]))
        row = self.matrix["scenarios"][1]
        row["classification"] = "SKIPPED"
        row["checks"] = {"warning": 1}
        self.matrix["score"] = {"PASS": 36, "FAIL": 0, "SKIPPED": 1, "NOT_TESTED": 0}
        self.matrix["runner_exit_code"] = 1
        self.matrix["release_gate_passed"] = False
        self.assertNotIn(secret, json.dumps(build(self.matrix, self.root, SHA)))


if __name__ == "__main__":
    unittest.main()
