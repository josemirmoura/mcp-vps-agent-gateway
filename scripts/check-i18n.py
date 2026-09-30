#!/usr/bin/env python3
"""Fail closed when an official Portico locale is incomplete."""

from __future__ import annotations

import json
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]
MANIFEST = json.loads((ROOT / "locales/manifest.json").read_text(encoding="utf-8"))
OFFICIAL = [x["code"] for x in MANIFEST["official"]]
NON_EN = [x for x in OFFICIAL if x != "en"]
CATALOG_DIR = ROOT / "locales/cli"

HUMAN_SHELL = [
    "install.sh", "init.sh", "setup-integrated-auth.sh", "verify-public.sh",
    "connect-chatgpt.sh", "verify.sh", "update.sh", "remove.sh",
    "diagnose.sh", "delegate-root.sh",
]
HUMAN_PY = ["preflight.py", "authority-summary.py", "delegate-root.py"]

TEXT_RE = re.compile(r"""vps_agent_text\s+(?:"([^"]*)"|'([^']*)')""")
PY_TEXT_RE = re.compile(r"""\bt\(\s*(?:"([^"]*)"|'([^']*)')\s*,""")
BLOCK_RE = re.compile(r"""vps_agent_block\s+([A-Za-z0-9_.-]+)""")
MSG_RE = re.compile(r"""vps_agent_msg\s+([A-Za-z0-9_.-]+)""")

errors: list[str] = []

catalogs = {}
for code in OFFICIAL:
    path = CATALOG_DIR / f"{code}.json"
    if not path.exists():
        errors.append(f"missing CLI catalog: {path.relative_to(ROOT)}")
        continue
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:
        errors.append(f"invalid CLI catalog {code}: {exc}")
        continue
    if data.get("locale") != code:
        errors.append(f"{path.relative_to(ROOT)}: locale field must be {code}")
    for section in ("strings", "blocks", "messages"):
        if not isinstance(data.get(section), dict):
            errors.append(f"{path.relative_to(ROOT)}: missing dict section {section}")
    catalogs[code] = data

required_strings: set[str] = set()
required_blocks: set[str] = set()
required_messages: set[str] = set()

for name in HUMAN_SHELL:
    path = ROOT / "scripts" / name
    text = path.read_text(encoding="utf-8")
    for match in TEXT_RE.finditer(text):
        source = match.group(1) or match.group(2) or ""
        if "$" not in source:
            required_strings.add(source)
    required_blocks.update(BLOCK_RE.findall(text))
    required_messages.update(MSG_RE.findall(text))

for name in HUMAN_PY:
    path = ROOT / "scripts" / name
    text = path.read_text(encoding="utf-8")
    for match in PY_TEXT_RE.finditer(text):
        source = match.group(1) or match.group(2) or ""
        if "{" not in source:
            required_strings.add(source)

for code in NON_EN:
    data = catalogs.get(code)
    if not data:
        continue
    for value in sorted(required_strings - set(data["strings"])):
        errors.append(f"{code}: missing CLI string translation: {value}")
    for key in sorted(required_blocks - set(data["blocks"])):
        errors.append(f"{code}: missing CLI block: {key}")
    for key in sorted(required_messages - set(data["messages"])):
        errors.append(f"{code}: missing CLI message: {key}")

# English and pt-BR keyed blocks/messages are also mandatory because multiline
# flows no longer have inline fallbacks.
for code in ("en", "pt-BR"):
    data = catalogs.get(code)
    if not data:
        continue
    for key in sorted(required_blocks - set(data["blocks"])):
        errors.append(f"{code}: missing keyed block: {key}")
    for key in sorted(required_messages - set(data["messages"])):
        errors.append(f"{code}: missing keyed message: {key}")

# Every canonical documentation page is part of the official localization
# contract. Localized documents must be substantive, not link-only stubs.
canonical_docs = sorted(
    p for p in (ROOT / "docs").glob("*.md")
    if p.is_file()
)
for code in NON_EN:
    for source in canonical_docs:
        target = ROOT / "docs" / code / source.name
        if not target.exists():
            errors.append(f"{code}: missing document {target.relative_to(ROOT)}")
            continue
        localized = target.read_text(encoding="utf-8").strip()
        if "TRANSLATION_PENDING" in localized:
            errors.append(f"{code}: pending translation marker in {target.relative_to(ROOT)}")
        source_len = len(source.read_text(encoding="utf-8").strip())
        # Locales vary in character density. This catches navigation-only stubs
        # without forcing translated prose to mirror English byte-for-byte.
        if source_len >= 1000 and len(localized) < source_len * 0.38:
            errors.append(
                f"{code}: document looks incomplete: {target.relative_to(ROOT)} "
                f"({len(localized)} chars vs {source_len} canonical)"
            )

# Repository landing README is localized separately.
for code in NON_EN:
    path = ROOT / f"README.{code}.md"
    if not path.exists():
        errors.append(f"{code}: missing repository README {path.name}")

# Gateway-native instructions and elicitation must cover the same locale set.
go_locale = (ROOT / "internal/gateway/locale.go").read_text(encoding="utf-8")
for code in OFFICIAL:
    if f'"{code}"' not in go_locale:
        errors.append(f"gateway locale table missing {code}")

if errors:
    print("I18N COVERAGE: FAIL", file=sys.stderr)
    for err in errors:
        print(f"- {err}", file=sys.stderr)
    raise SystemExit(1)

print(
    "I18N COVERAGE: PASS "
    + f"locales={','.join(OFFICIAL)} "
    + f"strings={len(required_strings)} blocks={len(required_blocks)} "
    + f"messages={len(required_messages)} docs={len(canonical_docs)}"
)
