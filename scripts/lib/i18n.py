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
    if lang in ("en", "pt-BR"):
        return {"strings": {}, "blocks": {}, "messages": {}}
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


def t(en: str, pt: str) -> str:
    lang = current_lang()
    if lang == "en":
        return en
    if lang == "pt-BR":
        return pt
    translated = catalog(lang)["strings"].get(en)
    if translated is not None:
        return translated
    if os.environ.get("VPS_AGENT_I18N_STRICT") == "1":
        raise KeyError(f"missing {lang} translation for: {en}")
    return en


def tr(key: str, en: str, pt: str, **values: object) -> str:
    lang = current_lang()
    if lang == "en":
        template = en
    elif lang == "pt-BR":
        template = pt
    else:
        template = catalog(lang)["messages"].get(key, en)
        if template == en and os.environ.get("VPS_AGENT_I18N_STRICT") == "1":
            raise KeyError(f"missing {lang} message translation for: {key}")
    return template.format(**values)


def block(key: str, en: str = "", pt: str = "") -> str:
    lang = current_lang()
    if lang == "en":
        value = en
    elif lang == "pt-BR":
        value = pt
    else:
        value = catalog(lang)["blocks"].get(key, en)
        if value == en and os.environ.get("VPS_AGENT_I18N_STRICT") == "1":
            raise KeyError(f"missing {lang} block translation for: {key}")
    return value


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
    p_block.add_argument("en")
    p_block.add_argument("pt")

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
        print(block(args.key, args.en, args.pt), end="")
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
