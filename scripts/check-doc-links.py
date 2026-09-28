#!/usr/bin/env python3
import pathlib
import re
import sys
from urllib.parse import unquote

ROOT = pathlib.Path(__file__).resolve().parents[1]
files = [ROOT / "README.md", ROOT / "README.pt-BR.md"]
files += sorted((ROOT / "docs").glob("*.md"))

pattern = re.compile(r"!?(?:\[[^\]]*\])\(([^)]+)\)")
errors = []

for path in files:
    if not path.is_file():
        errors.append(f"missing documentation file: {path.relative_to(ROOT)}")
        continue
    text = path.read_text(errors="replace")
    for raw in pattern.findall(text):
        target = raw.strip().split()[0].strip("<>")
        if not target or target.startswith(("#", "http://", "https://", "mailto:")):
            continue
        target = unquote(target.split("#", 1)[0])
        resolved = (path.parent / target).resolve()
        try:
            resolved.relative_to(ROOT.resolve())
        except ValueError:
            errors.append(f"{path.relative_to(ROOT)}: link escapes repository: {raw}")
            continue
        if not resolved.exists():
            errors.append(f"{path.relative_to(ROOT)}: broken local link: {raw}")

if errors:
    print("DOCUMENT LINK CHECK: FAIL", file=sys.stderr)
    for error in errors:
        print(f"- {error}", file=sys.stderr)
    raise SystemExit(1)

print("DOCUMENT LINK CHECK: PASS")
