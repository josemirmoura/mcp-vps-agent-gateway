#!/usr/bin/env python3
"""Small shared locale helper for human-facing Portico CLI output.

Machine-readable JSON and protocol payloads must not use this module.
"""

from __future__ import annotations

import os


def normalize_lang(raw: str | None) -> str:
    value = (raw or "").split(".", 1)[0].replace("@", "-")
    if value == "pt" or value.startswith("pt_BR") or value.startswith("pt-BR") or value.startswith("pt_"):
        return "pt-BR"
    if value == "en" or value.startswith("en_") or value.startswith("en-"):
        return "en"
    return "en"


def current_lang() -> str:
    for key in ("VPS_AGENT_LANG", "LC_ALL", "LC_MESSAGES", "LANG"):
        value = os.environ.get(key)
        if value:
            return normalize_lang(value)
    return "en"


def is_pt_br() -> bool:
    return current_lang() == "pt-BR"


def t(en: str, pt: str) -> str:
    return pt if is_pt_br() else en


def yes_no(value: bool) -> str:
    if is_pt_br():
        return "sim" if value else "não"
    return "yes" if value else "no"


def none_label() -> str:
    return "nenhuma" if is_pt_br() else "none"
