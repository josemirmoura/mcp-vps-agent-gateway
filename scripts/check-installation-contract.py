#!/usr/bin/env python3
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]
SUPPORTED = [
    "README.md",
    "README.pt-BR.md",
    "docs/quick-start.md",
    "docs/installer-flow.md",
    "docs/chatgpt-integration.md",
    "site/index.html",
    "site/pt-BR/index.html",
]

DEVELOPMENT_ONLY_LITERALS = [
    "mcp.josemirmoura.com.br",
    "85.209.93.51",
    "srv678294",
    "/opt/vps-agent-lab",
    "feat/integrated-oauth",
    "multi-vps",
    "public-endpoint-probe",
    "real-vps-preflight",
    "TRAEFIK DEFAULT CERT",
    "josemir-proof",
    "josemir-human-proof",
    "/opt/josemir-agradece-seu-gpt",
    "Josemir agradece seu GPT",
]

DANGEROUS_PATTERNS = [
    re.compile(r"BEGIN OPENSSH PRIVATE KEY", re.I),
    re.compile(r"\bsshpass\b", re.I),
    re.compile(r"\bssh\s+-i\s+", re.I),
    re.compile(
        r"\b(give|send|share|provide|paste)\b.{0,80}"
        r"(chatgpt|maintainer|developer).{0,80}"
        r"(vps password|ssh password|private ssh key|ssh private key|root password|private key)",
        re.I | re.S,
    ),
    re.compile(
        r"\b(d[eê]|envie|mande|compartilhe|forne[cç]a|cole)\b.{0,80}"
        r"(chatgpt|mantenedor|desenvolvedor).{0,80}"
        r"(senha da vps|senha ssh|chave ssh privada|chave privada|senha root)",
        re.I | re.S,
    ),
]

errors = []
texts = {}

for rel in SUPPORTED:
    path = ROOT / rel
    if not path.is_file():
        errors.append(f"missing supported-installation document: {rel}")
        continue
    text = path.read_text(errors="replace")
    texts[rel] = text

    for literal in DEVELOPMENT_ONLY_LITERALS:
        if literal.lower() in text.lower():
            errors.append(f"{rel}: development-only literal leaked into supported docs: {literal}")

    for pattern in DANGEROUS_PATTERNS:
        if pattern.search(text):
            errors.append(f"{rel}: prohibited credential/remote-access instruction matched: {pattern.pattern}")


# Repository-wide public sanitization guard. The literal definitions above live
# in this file by design, so this checker excludes itself from the scan.
for path in ROOT.rglob("*"):
    if not path.is_file():
        continue
    rel = path.relative_to(ROOT).as_posix()
    if rel == "scripts/check-installation-contract.py":
        continue
    if rel.startswith((".git/", "state/", "backups/", "diagnostics/", "dist/", "evidence/")):
        continue
    try:
        text = path.read_text(errors="replace")
    except OSError:
        continue
    for literal in DEVELOPMENT_ONLY_LITERALS:
        if literal.lower() in text.lower():
            errors.append(f"{rel}: development-only literal leaked into repository: {literal}")
    for pattern in DANGEROUS_PATTERNS[:3]:
        if pattern.search(text):
            errors.append(f"{rel}: sensitive credential material matched repository guard")

required_flow = {
    "docs/quick-start.md": ["scripts/install.sh", "INSTALLATION COMPLETE"],
    "README.md": ["scripts/install.sh"],
    "README.pt-BR.md": ["scripts/install.sh"],
    "docs/installer-flow.md": [
        "scripts/setup-integrated-auth.sh",
        "scripts/verify-public.sh",
        "scripts/connect-chatgpt.sh",
        "Development-only procedures",
    ],
    "docs/chatgpt-integration.md": ["scripts/connect-chatgpt.sh"],
    "site/index.html": [
        "scripts/setup-integrated-auth.sh",
        "scripts/connect-chatgpt.sh",
        'hreflang="pt-BR"',
        'href="pt-BR/"',
    ],
    "site/pt-BR/index.html": [
        "scripts/setup-integrated-auth.sh",
        "scripts/connect-chatgpt.sh",
        'lang="pt-BR"',
        'hreflang="en"',
        'href="../"',
    ],
}

for rel, markers in required_flow.items():
    text = texts.get(rel, "")
    for marker in markers:
        if marker not in text:
            errors.append(f"{rel}: required supported-flow marker missing: {marker}")

completion_files = [
    "docs/quick-start.md",
    "README.md",
    "README.pt-BR.md",
    "docs/installer-flow.md",
    "docs/chatgpt-integration.md",
    "site/index.html",
    "site/pt-BR/index.html",
]
for rel in completion_files:
    text = texts.get(rel, "")
    if "INSTALLATION COMPLETE" not in text and "INSTALAÇÃO CONCLUÍDA" not in text:
        errors.append(f"{rel}: installation completion gate is not explicit")

if errors:
    print("INSTALLATION DOCUMENT CONTRACT: FAIL", file=sys.stderr)
    for error in errors:
        print(f"- {error}", file=sys.stderr)
    raise SystemExit(1)

print("INSTALLATION DOCUMENT CONTRACT: PASS")
