#!/usr/bin/env python3
"""Trace frozen MCP results to original checks without editing the upstream score.

Output is intentionally redacted: check IDs/status and SHA-256 of error text,
never the text itself. Official checks.json remain in restricted CI artifacts.
The reporter validates every matrix row against its original JSON check records.
"""
from __future__ import annotations

import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path
import re
import sys

from mcp_conformance_matrix import scenario_outcome
from mcp_release_scope_audit import assess


def error_kind(message: str) -> str:
    if not message:
        return "none"
    if "Not testable:" in message:
        return "unavailable-diagnostic"
    if re.search(r'unknown (?:tool|prompt) "test_[a-zA-Z0-9_]+"', message):
        return "unavailable-synthetic-catalog"
    if message == "Failed: Resource not found":
        return "unavailable-resource-fixture"
    if "method not found: \"completion/complete\"" in message:
        return "unadvertised-completion-method"
    if "Expected InputRequiredResult" in message:
        return "input-required-fixture-not-returned"
    if message.startswith("Server returned JSON-RPC error instead"):
        return "warning-on-missing-synthetic-input"
    return "other"


def build(matrix: dict, result_root: Path, gateway_sha: str) -> dict:
    if not re.fullmatch(r"[0-9a-f]{40}", gateway_sha):
        raise ValueError("invalid Gateway source SHA")
    validated = assess(matrix)
    examined = []
    for row in matrix["scenarios"]:
        expected = row["classification"]
        checks = []
        for path_text in row.get("evidence_files", []):
            rel = Path(path_text)
            if rel.is_absolute() or ".." in rel.parts or rel.suffix != ".json":
                raise ValueError("unsafe original evidence path")
            source = result_root / rel
            if not source.is_file():
                raise ValueError("missing original checks.json")
            values = json.loads(source.read_text(encoding="utf-8"))
            if not isinstance(values, list) or not all(isinstance(v, dict) for v in values):
                raise ValueError("invalid original check records")
            checks.extend(values)
        recomputed, counts = scenario_outcome(checks, row["scenario"])
        if recomputed != expected:
            raise ValueError("official matrix differs from original checks")
        if counts != row.get("checks", {}):
            raise ValueError("official check counts differ from original records")
        evidence = []
        for item in checks:
            status = str(item.get("status", "")).upper()
            if status not in ("SUCCESS", "FAILURE", "SKIPPED", "WARNING", "INFO"):
                raise ValueError("unexpected check status")
            message = str(item.get("errorMessage", ""))
            evidence.append({
                "id": str(item.get("id", "")),
                "status": status,
                "error_kind": error_kind(message),
                "error_sha256": hashlib.sha256(message.encode("utf-8")).hexdigest()
                    if message else None,
            })
        examined.append({
            "scenario": row["scenario"],
            "official_classification": expected,
            "feature_context": next(v["product_feature_context"]
                                    for v in validated["scenarios"]
                                    if v["scenario"] == row["scenario"]),
            "checks": evidence,
        })
    return {
        "gateway_sha": gateway_sha,
        "upstream_commit": validated["upstream_commit"],
        "protocol_revision": validated["protocol_revision"],
        "origin": "original upstream checks.json; do not replace with synthetic score",
        "raw_score": validated["frozen_suite"]["score"],
        "runner_exit_code": validated["frozen_suite"]["runner_exit_code"],
        "release_gate_passed": validated["frozen_suite"]["release_gate_passed"],
        "waivers_applied": 0,
        "scenarios": examined,
    }


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--official", type=Path, required=True)
    p.add_argument("--results", type=Path, required=True)
    p.add_argument("--gateway-sha", required=True)
    p.add_argument("--output", type=Path, required=True)
    args = p.parse_args()
    try:
        matrix = json.loads(args.official.read_text(encoding="utf-8"))
        manifest = build(matrix, args.results, args.gateway_sha)
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n",
                               encoding="utf-8")
    except (ValueError, TypeError, KeyError, OSError) as exc:
        print("GATE C EVIDENCE REJECTED:", type(exc).__name__, file=sys.stderr)
        return 2
    print("GATE C FROZEN RAW SCORE:", json.dumps(manifest["raw_score"], sort_keys=True))
    print("GATE C RELEASE GATE:", manifest["release_gate_passed"])
    print("GATE C ALL 37 REQUIRED SCENARIOS (original check IDs/status, redacted):")
    for item in manifest["scenarios"]:
        exception_ids = [c["id"] + ":" + c["status"] + ":" + c["error_kind"]
                         for c in item["checks"] if c["status"] != "SUCCESS"]
        print(item["official_classification"].ljust(11), item["scenario"],
              "checks=" + str(len(item["checks"])),
              ";".join(exception_ids) if exception_ids else "all-checks-success")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
