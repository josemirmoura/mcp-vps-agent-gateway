"""Negative and positive tests for the pinned MCP conformance evidence matrix."""
from __future__ import annotations

import importlib.util
import json
from pathlib import Path
import tempfile
import unittest


TOOL = Path(__file__).resolve().parents[2] / "scripts" / "mcp_conformance_matrix.py"
spec = importlib.util.spec_from_file_location("portico_mcp_conformance_matrix", TOOL)
matrix = importlib.util.module_from_spec(spec)
assert spec and spec.loader
spec.loader.exec_module(matrix)


class ConformanceMatrixTest(unittest.TestCase):
    def test_reports_real_missing_scenarios_without_false_pass(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            req = root / "requirements.yaml"
            req.write_text(
                "server:\n  - server-stateless\n  - resources-read-binary\n"
                "client:\n  - auth/basic-cimd\nnot_scored:\n"
                "  - scenario: tasks-lifecycle\n    leg: server\n", encoding="utf-8",
            )
            result = root / "results" / "server-server-stateless-20261009"
            result.mkdir(parents=True)
            (result / "checks.json").write_text(
                json.dumps([{"status": "success", "id": "wire-schema-valid"}]),
                encoding="utf-8",
            )
            report = matrix.summarize(req, root / "results", 1)
            self.assertEqual(report["score"]["PASS"], 1)
            self.assertEqual(report["score"]["NOT_TESTED"], 1)
            self.assertFalse(report["release_gate_passed"])
            self.assertEqual(report["client_role_classification"], "NOT_APPLICABLE")
            self.assertEqual(report["required_client_scenarios"], 1)

    def test_fixture_absence_never_becomes_pass(self):
        outcome, _ = matrix.scenario_outcome([
            {"status": "success", "id": "good"},
            {"status": "failure",
             "errorMessage": "Not testable: server does not list the diagnostic tool test_x"},
        ])
        self.assertEqual(outcome, "NOT_TESTED")

    def test_unexpected_failure_is_failure(self):
        outcome, _ = matrix.scenario_outcome([
            {"status": "failure", "errorMessage": "MCP response violates schema"}
        ])
        self.assertEqual(outcome, "FAIL")

    def test_all_success_is_pass(self):
        self.assertEqual(matrix.scenario_outcome([
            {"status": "success"}, {"status": "success"}
        ])[0], "PASS")

    def test_skipped_is_not_pass(self):
        self.assertEqual(matrix.scenario_outcome([
            {"status": "success"}, {"status": "skipped"}
        ])[0], "SKIPPED")

    def test_no_checks_is_not_tested(self):
        self.assertEqual(matrix.scenario_outcome([])[0], "NOT_TESTED")

    def test_synthetic_test_tool_absence_is_not_tested(self):
        self.assertEqual(matrix.scenario_outcome([
            {"status": "SUCCESS"},
            {"status": "FAILURE",
             "errorMessage": 'JSON-RPC error: unknown tool "test_image_content"'},
        ], "tools-call-image")[0], "NOT_TESTED")

    def test_missing_reserved_prompt_is_not_tested(self):
        self.assertEqual(matrix.scenario_outcome([
            {"status": "FAILURE",
             "errorMessage": 'Failed: unknown prompt "test_simple_prompt"'},
        ], "prompts-get-simple")[0], "NOT_TESTED")

    def test_resources_fixture_missing_is_not_tested(self):
        self.assertEqual(matrix.scenario_outcome([
            {"status": "FAILURE", "errorMessage": "Failed: Resource not found"},
        ], "resources-read-text")[0], "NOT_TESTED")

    def test_real_method_missing_remains_failure(self):
        self.assertEqual(matrix.scenario_outcome([
            {"status": "FAILURE",
             "errorMessage": 'Failed: method not found: "completion/complete"'},
        ], "completion-complete")[0], "FAIL")

    def test_unexpected_tool_failure_is_not_excused(self):
        self.assertEqual(matrix.scenario_outcome([
            {"status": "FAILURE",
             "errorMessage": 'JSON-RPC error: unknown tool "file.read"'},
        ], "tools-call-simple-text")[0], "FAIL")

    def test_sse_info_does_not_create_false_error(self):
        self.assertEqual(matrix.scenario_outcome([
            {"status": "SUCCESS"},
            {"status": "INFO", "errorMessage": "SSE streams optional"},
        ], "server-sse-multiple-streams")[0], "PASS")

    def test_warning_does_not_count_as_pass(self):
        self.assertEqual(matrix.scenario_outcome([
            {"status": "SUCCESS"}, {"status": "WARNING"}
        ])[0], "SKIPPED")

    def test_mixed_real_failure_and_missing_fixture_remains_fail(self):
        self.assertEqual(matrix.scenario_outcome([
            {"status": "FAILURE",
             "errorMessage": 'unknown tool "test_image_content"'},
            {"status": "FAILURE", "errorMessage": "Invalid protocol version"},
        ], "tools-call-image")[0], "FAIL")

    def test_invalid_or_duplicate_requirement_set_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            req = Path(tmp) / "requirements.yaml"
            req.write_text("server:\n  - same\n  - same\nclient:\n  - other\n")
            with self.assertRaises(ValueError):
                matrix.yaml_items(req, "server")

    def test_all_passed_and_runner_zero_gate(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            req = root / "requirements.yaml"
            req.write_text("server:\n  - server-stateless\nclient:\n  - tools_call\n")
            folder = root / "results" / "server-server-stateless-20261009"
            folder.mkdir(parents=True)
            (folder / "checks.json").write_text('[{"status":"success"}]')
            result = matrix.summarize(req, root / "results", 0)
            self.assertTrue(result["release_gate_passed"])
            self.assertFalse(matrix.summarize(req, root / "results", 1)["release_gate_passed"])


if __name__ == "__main__":
    unittest.main()
