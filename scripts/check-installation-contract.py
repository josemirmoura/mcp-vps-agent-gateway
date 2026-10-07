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
    "josemir-" + "hu" + "man-proof",
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
    if "__pycache__/" in rel or rel.endswith((".pyc", ".pyo")):
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

# Portico MCP guided-product invariants discovered through clean-install E2E.
installer_text = (ROOT / "scripts/install.sh").read_text(errors="replace")
init_text = (ROOT / "scripts/init.sh").read_text(errors="replace")
product_text = (ROOT / "scripts/lib/product.sh").read_text(errors="replace")
remove_text = (ROOT / "scripts/remove.sh").read_text(errors="replace")
quick_text = texts.get("docs/quick-start.md", "")

product_requirements = {
    "scripts/install.sh": [
        "--dynamic-baseline",
        'SCOPE="/opt"',
        "--run-as",
        "--lang",
    ],
    "scripts/init.sh": [
        "--dynamic-baseline",
        "static project roots are authorized",
        "run_as:",
    ],
    "scripts/lib/product.sh": [
        'VPS_AGENT_PRODUCT_NAME="Portico MCP"',
        "Feito por",
        "github.com/josemirmoura",
        "pt-BR",
    ],
    "scripts/remove.sh": [
        "mcp-vps-agent-gateway:local",
        "mcp-vps-agent-broker:local",
        "--remove-source",
        "/opt/vps-agent-sandbox",
    ],
    "docs/quick-start.md": [
        "Portico MCP Quick Start",
        "static project roots:        none",
        "permissions.request_root_access",
    ],
}
source_map = {
    "scripts/install.sh": installer_text,
    "scripts/init.sh": init_text,
    "scripts/lib/product.sh": product_text,
    "scripts/remove.sh": remove_text,
    "docs/quick-start.md": quick_text,
}
for rel, markers in product_requirements.items():
    source = source_map[rel]
    for required in markers:
        if required not in source:
            errors.append(f"{rel}: Portico guided-product invariant missing: {required}")

if "Refine config/policy.yaml" in installer_text or "Edit config/policy.yaml now" in installer_text:
    errors.append("scripts/install.sh: normal guided flow still requires manual policy YAML editing")
if "--scope /opt/vps-agent-sandbox" in quick_text or "install -d" in quick_text and "/opt/vps-agent-sandbox" in quick_text:
    errors.append("docs/quick-start.md: legacy sandbox must not be the normal installation path")

# Clean E2E UX/security invariants (#33-#41).
authority_text = (ROOT / "scripts/authority-summary.py").read_text(errors="replace")
preflight_text = (ROOT / "scripts/preflight.py").read_text(errors="replace")
auth_text = (ROOT / "scripts/setup-integrated-auth.sh").read_text(errors="replace")
connect_text = (ROOT / "scripts/connect-chatgpt.sh").read_text(errors="replace")
native_approval_text = (ROOT / "internal/gateway/native_approval.go").read_text(errors="replace")
authority_tools_text = (ROOT / "internal/gateway/authority_tools.go").read_text(errors="replace")
gateway_text = (ROOT / "internal/gateway/gateway.go").read_text(errors="replace")
tool_annotations_text = (ROOT / "internal/gateway/tool_annotations.go").read_text(errors="replace")
broker_text = (ROOT / "internal/broker/broker.go").read_text(errors="replace")
if (ROOT / "internal/gateway/root_approval_widget.go").exists():
    errors.append("internal/gateway/root_approval_widget.go: custom approval iframe must not return as the normal approval surface")

ux_requirements = {
    "scripts/authority-summary.py": [
        "Resumo da autoridade efetiva",
        "Raízes de leitura do filesystem",
        "none_label",
    ],
    "scripts/preflight.py": [
        "Verificando requisitos do Portico MCP",
        "https://docs.docker.com/engine/install/",
        "https://docs.docker.com/compose/install/linux/",
        "Portico will provision its bundled Traefik automatically",
        "Portico irá reutilizá-lo",
    ],
    "scripts/install.sh": [
        "Escolha como o Portico MCP poderá acessar sua VPS",
        "nomes das pastas imediatamente abaixo do teto",
        "conteúdo fica bloqueado até aprovação explícita",
        "Arquivos protegidos como .env continuam trancados",
        "python3 scripts/preflight.py",
    ],
    "scripts/setup-integrated-auth.sh": [
        "Para conectar o ChatGPT ao Portico MCP",
        "Domínio público do Portico MCP",
        "Nome de usuário do login OAuth",
        "Traefik existente detectado e será reutilizado",
    ],
    "scripts/connect-chatgpt.sh": [
        "A VPS está pronta. Agora falta conectar o Portico MCP ao ChatGPT.",
        "[Enter] verificar conexão",
        "--baseline-only",
        "--after-seq",
        "Ainda não detectamos a chamada system.info",
    ],
    "internal/gateway/native_approval.go": [
        "mcp.InputRequestMap",
        "mcp.ElicitParams",
        "RequestState",
        "ClientCapabilities",
        "decline",
        "cancel",
        "ATENÇÃO",
        "Autorizar Pórtico?",
        "Autorizar arquivo protegido?",
        "Perfil:",
        "Duração:",
        "Somente este arquivo será liberado.",
    ],
    "internal/gateway/gateway.go": [
        '"permissions.request_root_access"',
        'nativeApprovalDecision(req, "root")',
        'nativeApprovalResult("root"',
    ],
    "internal/gateway/authority_tools.go": [
        '"permissions.request_sensitive_access"',
        'nativeApprovalDecision(req, "sensitive")',
        'nativeApprovalResult("sensitive"',
        'annotatedTool(',
    ],
    "internal/gateway/tool_annotations.go": [
        "func toolSafetyFor",
        "func annotatedTool",
        "ReadOnlyHint",
        "DestructiveHint",
        "IdempotentHint",
        "OpenWorldHint",
    ],
    "internal/broker/broker.go": [
        '"ceiling_wide": root == physical',
        '"physical_ceiling": physical',
        '"permissions.discover_scope"',
        '"permissions.request_sensitive_access"',
    ],
}
ux_sources = {
    "scripts/authority-summary.py": authority_text,
    "scripts/preflight.py": preflight_text,
    "scripts/install.sh": installer_text,
    "scripts/setup-integrated-auth.sh": auth_text,
    "scripts/connect-chatgpt.sh": connect_text,
    "internal/gateway/native_approval.go": native_approval_text,
    "internal/gateway/gateway.go": gateway_text,
    "internal/gateway/authority_tools.go": authority_tools_text,
    "internal/gateway/tool_annotations.go": tool_annotations_text,
    "internal/broker/broker.go": broker_text,
}
for rel, markers in ux_requirements.items():
    source = ux_sources[rel]
    for required in markers:
        if required not in source:
            errors.append(f"{rel}: clean-E2E invariant missing: {required}")

if "dynamic delegation of the entire physical ceiling is not allowed" in broker_text:
    errors.append("internal/broker/broker.go: physical ceiling is still hard-blocked despite explicit operator approval")
if "rootApprovalWidgetURI" in gateway_text or "openai/outputTemplate" in gateway_text or "openai/outputTemplate" in authority_tools_text:
    errors.append("internal/gateway: custom MCP Apps approval UI metadata returned instead of native elicitation")
if 'annotatedTool("permissions.confirm_root_access"' in gateway_text or 'annotatedTool("permissions.confirm_sensitive_access"' in authority_tools_text:
    errors.append("internal/gateway: model-visible approval confirmation tool returned")
if "mcp.AddTool(server, &mcp.Tool{" in gateway_text or "mcp.AddTool(server, &mcp.Tool{" in authority_tools_text:
    errors.append("internal/gateway: public tools must use centralized annotatedTool classification")
if 'wait-tool \\\n  --subject' in connect_text and '--after-seq "$BASELINE"' not in connect_text:
    errors.append("scripts/connect-chatgpt.sh: ChatGPT verification lost its fixed audit baseline")
if "VPS_AGENT_CHATGPT_VERIFY_TIMEOUT:-10m" in connect_text and 'if [ ! -t 0 ]' not in connect_text:
    errors.append("scripts/connect-chatgpt.sh: hidden interactive verification timeout returned")

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
