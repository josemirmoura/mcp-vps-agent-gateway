#!/usr/bin/env python3
"""Classify pinned official MCP server conformance evidence without masking gaps.

This is a reporting tool. Exit 0 means evidence was parsed, NOT that the
Community is compliant. Pass --enforce to require every scored server scenario
to pass before shipping a release candidate.
"""
from __future__ import annotations

import argparse
import json
import pathlib
import re
import sys
from collections import Counter

SPEC = "2026-07-28"
UPSTREAM_SHA = "c37eec888e1c6ff140af79987a40008548b7cc5f"


def yaml_items(path: pathlib.Path, section: str) -> list[str]:
    """Read simple, frozen top-level YAML lists without adding dependencies."""
    found = False
    output: list[str] = []
    for raw in path.read_text(encoding="utf-8").splitlines():
        if not raw.strip() or raw.lstrip().startswith("#"):
            continue
        if re.match(r"^[a-z_]+:\s*$", raw):
            found = raw.split(":", 1)[0] == section
            continue
        if found:
            match = re.fullmatch(r"  - ([a-z0-9_./-]+)\s*", raw)
            if match:
                output.append(match.group(1))
    if not output or len(output) != len(set(output)):
        raise ValueError(f"invalid/empty/duplicate {section} requirements")
    return output


def scenario_outcome(checks: list[dict]) -> tuple[str, dict[str, int]]:
    if not checks:
        return "NOT_TESTED", {}
    counts = Counter(str(row.get("status", "unknown")).lower() for row in checks)
    if counts["failure"] or counts["error"] or counts["unknown"]:
        # Fixture diagnostics are not silently promoted to PASS.
        if all(
            row.get("status", "").lower() in ("success", "skipped")
            or (
                row.get("status", "").lower() == "failure"
                and str(row.get("errorMessage", "")).startswith(
                    "Not testable: server does not list the diagnostic tool"
                )
            )
            for row in checks
        ):
            return "NOT_TESTED", dict(counts)
        return "FAIL", dict(counts)
    if counts["warning"] or any(k not in ("success", "skipped") for k in counts):
        return "FAIL", dict(counts)
    if counts["success"]:
        # A scenario with both success and skipped is only partially covered.
        return ("SKIPPED" if counts["skipped"] else "PASS"), dict(counts)
    return "SKIPPED", dict(counts)


def summarize(requirements: pathlib.Path, results: pathlib.Path, runner_exit: int) -> dict:
    scored = yaml_items(requirements, "server")
    # This tool audits a server, not an OAuth *client* implementation.
    clients = yaml_items(requirements, "client")
    rows: list[dict] = []
    for scenario in scored:
        report_files = sorted(results.glob(f"server-{scenario}-*/checks.json"))
        # Some suite versions write results under a grouped run directory.
        if not report_files:
            report_files = sorted(results.glob(f"**/server-{scenario}-*/checks.json"))
        checks: list[dict] = []
        errors: list[str] = []
        for file in report_files:
            try:
                data = json.loads(file.read_text(encoding="utf-8"))
                if not isinstance(data, list):
                    raise ValueError("not a list")
                checks.extend(x for x in data if isinstance(x, dict))
            except (ValueError, OSError) as exc:
                errors.append(type(exc).__name__)
        status, counts = scenario_outcome(checks)
        if errors:
            status = "FAIL"
        rows.append({
            "scenario": scenario,
            "classification": status,
            "checks": counts,
            "evidence_files": [str(p.relative_to(results)) for p in report_files],
            "parse_errors": errors,
        })
    counts = dict(Counter(item["classification"] for item in rows))
    complete = counts.get("PASS", 0) == len(scored) and runner_exit == 0
    return {
        "protocol_revision": SPEC,
        "role": "server",
        "upstream_commit": UPSTREAM_SHA,
        "required_server_scenarios": len(scored),
        "required_client_scenarios": len(clients),
        "client_role_classification": "NOT_APPLICABLE",
        "score": {status: counts.get(status, 0) for status in
                  ("PASS", "FAIL", "SKIPPED", "NOT_TESTED")},
        "runner_exit_code": runner_exit,
        "release_gate_passed": complete,
        "interpretation": ("All required server scenarios passed on this fixture."
                           if complete else
                           "INCOMPLETE: do not claim full MCP conformance or release gate."),
        "scenarios": rows,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--requirements", required=True, type=pathlib.Path)
    parser.add_argument("--results", required=True, type=pathlib.Path)
    parser.add_argument("--runner-exit", type=int, required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    parser.add_argument("--enforce", action="store_true")
    args = parser.parse_args()
    try:
        report = summarize(args.requirements, args.results, args.runner_exit)
    except (OSError, ValueError) as exc:
        print(f"ERROR: cannot classify official conformance evidence: {type(exc).__name__}: {exc}",
              file=sys.stderr)
        return 2
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print("OFFICIAL MCP REQUIREMENTS CLASSIFICATION:",
          json.dumps({
              "upstream": report["upstream_commit"],
              "revision": report["protocol_revision"],
              "required": report["required_server_scenarios"],
              "client_role": report["client_role_classification"],
              "score": report["score"],
              "runner_exit": report["runner_exit_code"],
              "release_gate_passed": report["release_gate_passed"],
          }, sort_keys=True))
    if args.enforce and not report["release_gate_passed"]:
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
