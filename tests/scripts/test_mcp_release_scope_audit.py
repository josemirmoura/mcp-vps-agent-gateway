"""Integrity and non-waiver regressions for frozen MCP release scope annotations."""
import importlib.util
from pathlib import Path
import unittest


SCRIPT = Path(__file__).resolve().parents[2] / "scripts" / "mcp_release_scope_audit.py"
spec = importlib.util.spec_from_file_location("mcp_release_scope_audit", SCRIPT)
audit = importlib.util.module_from_spec(spec)
assert spec and spec.loader
spec.loader.exec_module(audit)


def fixture(status="FAIL", scenario="completion-complete", exit_code=1):
    rows = [
        {"scenario": name,
         "classification": status if name == scenario else "PASS",
         "checks": {}, "evidence_files": []}
        for name in audit.REQUIRED_SERVER
    ]
    scores = {v: sum(r["classification"] == v for r in rows)
              for v in ("PASS", "FAIL", "SKIPPED", "NOT_TESTED")}
    return {
        "protocol_revision": audit.REVISION,
        "upstream_commit": audit.UPSTREAM,
        "role": "server",
        "required_server_scenarios": 37,
        "required_client_scenarios": 32,
        "client_role_classification": "NOT_APPLICABLE",
        "scenarios": rows,
        "score": scores,
        "runner_exit_code": exit_code,
        "release_gate_passed": scores["PASS"] == 37 and exit_code == 0,
    }


class FrozenRequirementsTest(unittest.TestCase):
    def test_completion_failure_is_not_waived(self):
        result = audit.assess(fixture())
        self.assertEqual(result["frozen_suite"]["score"]["FAIL"], 1)
        self.assertFalse(result["frozen_suite"]["release_gate_passed"])
        self.assertEqual(result["frozen_suite"]["waivers_applied"], 0)
        row = next(r for r in result["scenarios"]
                   if r["scenario"] == "completion-complete")
        self.assertEqual(row["product_feature_context"], "OPTIONAL_NOT_ADVERTISED")
        self.assertEqual(row["raw_classification"], "FAIL")
        self.assertFalse(row["accepted_as_full_suite_pass"])

    def test_missing_synthetic_multiround_remains_raw_failure(self):
        result = audit.assess(fixture(scenario="input-required-result-multi-round"))
        row = next(r for r in result["scenarios"]
                   if r["scenario"] == "input-required-result-multi-round")
        self.assertEqual(row["raw_classification"], "FAIL")
        self.assertEqual(row["product_feature_context"], "REQUIRES_TEST_ONLY_FIXTURE")
        self.assertFalse(result["frozen_suite"]["release_gate_passed"])

    def test_all_required_scenarios_and_exit_zero_needed(self):
        ok = fixture(status="PASS",exit_code=0)
        self.assertTrue(audit.assess(ok)["frozen_suite"]["release_gate_passed"])
        ok["runner_exit_code"] = 1
        with self.assertRaises(ValueError):
            audit.assess(ok)
        ok["release_gate_passed"] = False
        self.assertFalse(audit.assess(ok)["frozen_suite"]["release_gate_passed"])

    def test_wrong_upstream_not_accepted(self):
        raw = fixture()
        raw["upstream_commit"] = "unexpected"
        with self.assertRaises(ValueError):
            audit.assess(raw)

    def test_missing_or_duplicate_scenario_not_accepted(self):
        raw = fixture()
        raw["scenarios"][1] = raw["scenarios"][0]
        with self.assertRaises(ValueError):
            audit.assess(raw)

    def test_inconsistent_score_not_accepted(self):
        raw = fixture()
        raw["score"]["PASS"] = 37
        with self.assertRaises(ValueError):
            audit.assess(raw)

    def test_not_tested_is_never_accepted_as_pass(self):
        result = audit.assess(fixture(status="NOT_TESTED",
                                      scenario="input-required-result-multi-round"))
        row = next(r for r in result["scenarios"]
                   if r["scenario"] == "input-required-result-multi-round")
        self.assertEqual(row["raw_classification"], "NOT_TESTED")
        self.assertFalse(row["accepted_as_full_suite_pass"])

    def test_client_conformance_not_claimed(self):
        raw = fixture()
        raw["client_role_classification"] = "PASS"
        with self.assertRaises(ValueError):
            audit.assess(raw)


if __name__ == "__main__":
    unittest.main()
