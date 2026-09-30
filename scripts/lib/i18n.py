#!/usr/bin/env python3
"""Shared locale helper for human-facing Portico output.

English is the canonical source language. pt-BR keeps the existing inline
fallback for backward compatibility; all other official locales are loaded
from versioned catalogs in locales/cli/.

Machine-readable JSON/protocol payloads must remain language-neutral unless
explicitly documented otherwise.
"""

from __future__ import annotations

import argparse
import json
import os
import pathlib
import re
import sys
from functools import lru_cache

ROOT = pathlib.Path(__file__).resolve().parents[2]
LOCALES_DIR = ROOT / "locales" / "cli"
MANIFEST = ROOT / "locales" / "manifest.json"
SUPPORTED = ("en", "pt-BR", "es", "de", "fr", "ja", "id")


def normalize_lang(raw: str | None) -> str:
    value = (raw or "").split(".", 1)[0].replace("@", "-").replace("_", "-")
    lower = value.lower()
    if lower == "pt" or lower.startswith("pt-br"):
        return "pt-BR"
    for code in ("en", "es", "de", "fr", "ja", "id"):
        if lower == code or lower.startswith(code + "-"):
            return code
    return "en"


def current_lang() -> str:
    for key in ("VPS_AGENT_LANG", "LC_ALL", "LC_MESSAGES", "LANG"):
        value = os.environ.get(key)
        if value:
            return normalize_lang(value)
    return "en"


@lru_cache(maxsize=None)
def catalog(lang: str) -> dict:
    lang = normalize_lang(lang)
    path = LOCALES_DIR / f"{lang}.json"
    if not path.exists():
        return {"strings": {}, "blocks": {}, "messages": {}}
    data = json.loads(path.read_text(encoding="utf-8"))
    for section in ("strings", "blocks", "messages"):
        data.setdefault(section, {})
    return data


@lru_cache(maxsize=1)
def manifest() -> dict:
    return json.loads(MANIFEST.read_text(encoding="utf-8"))


def is_pt_br() -> bool:
    return current_lang() == "pt-BR"


DYNAMIC_TEXT_PATTERNS = (
    (r"^ERROR: unknown argument: (?P<arg>.+)$", "error.unknown_argument"),
    (r"^Creating (?P<scope>.+) with sudo install\\.$", "install.creating_scope"),
    (r"^(?P<scope>.+) does not exist\\. Create it now\\? \\[y/N\\]: $", "install.scope_missing_prompt"),
    (r"^ERROR: scope does not exist: (?P<scope>.+)$", "error.scope_missing"),
    (r"^ERROR: invalid non-root shell user: (?P<user>.+)$", "error.invalid_shell_user"),
    (r"^Standard profile selected\\. Physical ceiling: (?P<scope>.+)\\.$", "install.standard_selected"),
    (r"^ERROR: --scope must be an absolute path: (?P<scope>.+)$", "error.scope_absolute"),
    (r"^ERROR: --scope must be canonical \\(no '\\.\\.', '\\.' or trailing slash\\): (?P<scope>.+)$", "error.scope_canonical"),
    (r"^Physical ceiling changed from (?P<old>.+) to (?P<new>.+); static roots will start empty\\.$", "init.ceiling_changed_empty"),
    (r"^Physical ceiling changed from (?P<old>.+) to (?P<new>.+)\\. Existing logical policy roots were preserved\\.$", "init.ceiling_changed_preserved"),
    (r"^ERROR: VPS_AGENT_SCOPE_ROOT may not traverse symlinks: (?P<path>.+)$", "error.scope_symlink"),
    (r"^ERROR: VPS_AGENT_SCOPE_ROOT does not exist: (?P<path>.+)$", "error.scope_root_missing"),
    (r"^ERROR: shell execution user does not exist: (?P<user>.+)$", "error.shell_user_missing"),
    (r"^ERROR: required command not found: (?P<command>.+)$", "error.required_command"),
    (r"^ERROR: DNS for (?P<domain>.+) does not resolve yet\\.$", "error.dns_unresolved"),
    (r"^ERROR: integrated auth was already initialized for (?P<domain>.+)\\.$", "error.auth_already_initialized"),
    (r"^ERROR: operator email does not look valid: (?P<email>.+)$", "error.operator_email_invalid"),
    (r"^ERROR: more than one Traefik is attached to (?P<network>.+)\\.$", "error.traefik_multiple_network"),
    (r"^ERROR: Traefik has (?P<count>\\d+) candidate Docker networks: (?P<networks>.+)$", "error.traefik_candidate_networks"),
    (r"^ERROR: Docker network not found: (?P<network>.+)$", "error.docker_network_missing"),
    (r"^ERROR: OIDC discovery did not become reachable at (?P<url>.+)$", "error.oidc_unreachable"),
    (r"^ERROR: operator creation failed \\(curl=(?P<curl_status>[^ ]+) HTTP=(?P<http_code>[^)]+)\\)\\.$", "error.operator_creation_failed"),
    (r"^Public MCP endpoint must use HTTPS: (?P<url>.+)$", "error.public_url_https"),
    (r"^Expected HTTP 401 from unauthenticated MCP request, got (?P<code>.+)\\.$", "error.expected_401"),
    (r"^Local MCP tool call skipped for auth mode (?P<mode>.+); public OAuth verification will be used\\.$", "verify.local_call_skipped"),
)


def dynamic_translation(lang: str, en: str) -> str | None:
    messages = catalog(lang)["messages"]
    for pattern, key in DYNAMIC_TEXT_PATTERNS:
        match = re.match(pattern, en)
        if not match:
            continue
        template = messages.get(key)
        if template is None:
            return None
        return template.format(**match.groupdict())
    return None


def t(en: str, pt: str) -> str:
    lang = current_lang()
    if lang == "en":
        return en
    if lang == "pt-BR":
        return pt
    translated = catalog(lang)["strings"].get(en)
    if translated is not None:
        return translated
    translated = dynamic_translation(lang, en)
    if translated is not None:
        return translated
    if os.environ.get("VPS_AGENT_I18N_STRICT") == "1":
        raise KeyError(f"missing {lang} translation for: {en}")
    return en


def tr(key: str, en: str = "", pt: str = "", **values: object) -> str:
    lang = current_lang()
    template = catalog(lang)["messages"].get(key)
    if template is None:
        if lang == "pt-BR" and pt:
            template = pt
        elif en:
            template = en
        else:
            template = catalog("en")["messages"].get(key)
    if template is None:
        if os.environ.get("VPS_AGENT_I18N_STRICT") == "1":
            raise KeyError(f"missing {lang} message translation for: {key}")
        template = key
    return template.format(**values)


def block(key: str, **values: object) -> str:
    lang = current_lang()
    value = catalog(lang)["blocks"].get(key)
    if value is None:
        if os.environ.get("VPS_AGENT_I18N_STRICT") == "1":
            raise KeyError(f"missing {lang} block translation for: {key}")
        value = catalog("en")["blocks"].get(key, key)
    return value.format(**values)


def yes_no(value: bool) -> str:
    return t("yes", "sim") if value else t("no", "não")


def none_label() -> str:
    return t("none", "nenhuma")


def language_label(lang: str | None = None) -> str:
    code = normalize_lang(lang or current_lang())
    for item in manifest()["official"]:
        if item["code"] == code:
            return item["native_name"]
    return "English"


def parse_values(items: list[str]) -> dict[str, str]:
    out: dict[str, str] = {}
    for item in items:
        if "=" not in item:
            raise SystemExit(f"invalid template value: {item!r}; expected name=value")
        key, value = item.split("=", 1)
        out[key] = value
    return out


def main() -> int:
    parser = argparse.ArgumentParser()
    sub = parser.add_subparsers(dest="command", required=True)

    p_text = sub.add_parser("text")
    p_text.add_argument("en")
    p_text.add_argument("pt")

    p_block = sub.add_parser("block")
    p_block.add_argument("key")
    p_block.add_argument("values", nargs="*")

    p_msg = sub.add_parser("message")
    p_msg.add_argument("key")
    p_msg.add_argument("en")
    p_msg.add_argument("pt")
    p_msg.add_argument("values", nargs="*")

    sub.add_parser("label")
    sub.add_parser("locales")

    args = parser.parse_args()
    if args.command == "text":
        print(t(args.en, args.pt), end="")
    elif args.command == "block":
        print(block(args.key, **parse_values(args.values)), end="")
    elif args.command == "message":
        print(tr(args.key, args.en, args.pt, **parse_values(args.values)), end="")
    elif args.command == "label":
        print(language_label(), end="")
    elif args.command == "locales":
        for item in manifest()["official"]:
            print(f"{item['code']}\t{item['native_name']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
