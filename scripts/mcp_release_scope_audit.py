#!/usr/bin/env python3
"""Attach capability/scenario applicability WITHOUT editing official MCP results.

The upstream 2026-07-28 requirements are frozen and remain the only authority
for a *full-suite* conformance result. This companion report is a release
review aid, not a score reducer, waiver, or new interpretation of the spec.
"""
from __future__ import annotations

import argparse
import json
from pathlib import Path
import sys

UPSTREAM = "c37eec888e1c6ff140af79987a40008548b7cc5f"
REVISION = "2026-07-28"

# Exact scenario inventory from the frozen upstream requirements file.
REQUIRED_SERVER = (
    "server-stateless", "completion-complete", "tools-list",
    "tools-call-simple-text", "tools-call-image", "tools-call-audio",
    "tools-call-embedded-resource", "tools-call-mixed-content",
    "tools-call-error", "tools-call-with-progress",
    "server-sse-multiple-streams", "resources-list",
    "resources-read-text", "resources-read-binary",
    "resources-templates-read", "sep-2164-resource-not-found",
    "prompts-list", "prompts-get-simple", "prompts-get-with-args",
    "prompts-get-embedded-resource", "prompts-get-with-image",
    "dns-rebinding-protection", "caching",
    "input-required-result-basic-elicitation",
    "input-required-result-basic-sampling",
    "input-required-result-basic-list-roots",
    "input-required-result-request-state",
    "input-required-result-multiple-input-requests",
    "input-required-result-multi-round",
    "input-required-result-missing-input-response",
    "input-required-result-non-tool-request",
    "input-required-result-result-type",
    "input-required-result-unsupported-methods",
    "input-required-result-tampered-state",
    "input-required-result-capability-check",
    "input-required-result-ignore-extra-params",
    "input-required-result-validate-input",
)

# This categorization does NOT remove any of the 37 scored scenarios.
# It only explains how to interpret the corresponding raw results.
CONDITIONAL_UNADVERTISED = {
    "completion-complete": "completions is not advertised; Go SDK "
        "v1.8.0 infers it only when CompletionHandler is installed. "
        "The frozen suite still scores this scenario as required.",
}
REQUIRES_SYNTHETIC_FIXTURE = {
    *(name for name in REQUIRED_SERVER if name.startswith("tools-call-")),
    *(name for name in REQUIRED_SERVER if name.startswith("prompts-get-")),
    *(name for name in REQUIRED_SERVER if name.startswith("resources-read-")),
    "resources-templates-read",
    *(name for name in REQUIRED_SERVER if name.startswith("input-required-result-")
      and name not in (
          "input-required-result-unsupported-methods",
          "input-required-result-validate-input",
          "input-required-result-ignore-extra-params",
      )),
}
REFERENCE_FIXTURES = {
    "server-stateless": "Some subchecks require diagnostic test_missing_capability, "
        "test_streaming_elicitation and test_logging_tool; other subchecks "
        "do exercise the normal Gateway.",
    "caching": "The resources/read caching subcheck is skipped when the "
        "production catalog has no readable resource fixture.",
}

def assess(raw: dict) -> dict:
    if raw.get("protocol_revision") != REVISION:
        raise ValueError("unexpected protocol revision")
    if raw.get("upstream_commit") != UPSTREAM:
        raise ValueError("unexpected upstream commit")
    if raw.get("role") != "server":
        raise ValueError("not MCP server-role evidence")
    if raw.get("required_server_scenarios") != len(REQUIRED_SERVER):
        raise ValueError("frozen scenario count mismatch")
    rows = raw.get("scenarios")
    if not isinstance(rows, list) or len(rows) != len(REQUIRED_SERVER):
        raise ValueError("missing or duplicate scenario evidence")
    actual = [r.get("scenario") for r in rows if isinstance(r, dict)]
    if actual != list(REQUIRED_SERVER):
        raise ValueError("upstream scenario order/name mismatch")
    statuses = ("PASS", "FAIL", "SKIPPED", "NOT_TESTED")
    counted = {key: 0 for key in statuses}
    result = []
    for row in rows:
        scenario, status = row["scenario"], row.get("classification")
        if status not in counted:
            raise ValueError("invalid raw classification")
        counted[status] += 1
        if scenario in CONDITIONAL_UNADVERTISED:
            feature = "OPTIONAL_NOT_ADVERTISED"
            note = CONDITIONAL_UNADVERTISED[scenario]
        elif scenario in REQUIRES_SYNTHETIC_FIXTURE:
            feature = "REQUIRES_TEST_ONLY_FIXTURE"
            note = ("The official scenario invokes a synthetic test-specific tool,"
                    " prompt or resource; the production catalog is not equivalent"
                    " to the conformance everything-server.")
        elif scenario in REFERENCE_FIXTURES:
            feature = "PARTIAL_SYNTHETIC_SUBCHECKS"
            note = REFERENCE_FIXTURES[scenario]
        else:
            feature = "PRODUCT_PROTOCOL_PROBE"
            note = "Evaluate the normal Gateway's raw check evidence."
        result.append({
            "scenario": scenario,
            "required_in_frozen_suite": True,
            "raw_classification": status,
            "product_feature_context": feature,
            "interpretation_note": note,
            "accepted_as_full_suite_pass": status == "PASS",
        })

    if counted != raw.get("score"):
        raise ValueError("raw status counts inconsistent")
    if raw.get("client_role_classification") != "NOT_APPLICABLE":
        raise ValueError("client-role boundary is not explicit")
    runner_ok = raw.get("runner_exit_code") == 0
    full_pass = counted["PASS"] == len(REQUIRED_SERVER) and runner_ok
    if raw.get("release_gate_passed") is not full_pass:
        raise ValueError("raw release gate contradicts raw evidence")
    return {
        "protocol_revision": REVISION,
        "upstream_commit": UPSTREAM,
        "role": "server",
        "frozen_suite": {
            "scenario_count": len(REQUIRED_SERVER),
            "score": counted,
            "runner_exit_code": raw["runner_exit_code"],
            "release_gate_passed": full_pass,
            "waivers_applied": 0,
        },
        "product_scope_release_claim": (
            "No full MCP 2026-07-28 conformance claim unless ALL 37 frozen "
            "scenarios pass with upstream exit 0. Feature-level applicability "
            "does not rewrite the frozen requirements."
        ),
        "completion_opt_out_proof": (
            "internal/gateway/optional_completion_capability_test.go "
            "(must pass CI at this exact SHA; capability absent and method -32601)"
        ),
        "synthetic_catalog_boundary_proof": (
            "internal/gateway/no_synthetic_catalog_test.go "
            "(must pass CI at this exact SHA)"
        ),
        "scenarios": result,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--official", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    try:
        official = json.loads(args.official.read_text(encoding="utf-8"))
        report = assess(official)
    except (OSError, ValueError, TypeError, KeyError) as exc:
        print(f"FAIL CLOSED: {type(exc).__name__}: {exc}", file=sys.stderr)
        return 2
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n",
                           encoding="utf-8")
    print("FROZEN MCP SUITE (UNCHANGED):",
          json.dumps(report["frozen_suite"], sort_keys=True))
    print("COMPANION APPLICABILITY IS NOT A WAIVER OR FULL CONFORMANCE")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
