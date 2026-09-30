#!/usr/bin/env python3
"""Human-readable Portico authority summary.

This is presentation only. The Broker's policy parser remains authoritative.
Machine-readable policy/protocol values are not localized.
"""

from __future__ import annotations

import argparse
import pathlib
import shlex
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent / "lib"))
from i18n import is_pt_br, none_label, t, yes_no  # noqa: E402


def env_values(path: pathlib.Path) -> dict[str, str]:
    result: dict[str, str] = {}
    if not path.is_file():
        return result
    for raw in path.read_text(errors="replace").splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        try:
            parts = shlex.split(value)
            result[key] = parts[0] if parts else ""
        except ValueError:
            result[key] = value.strip("'"")
    return result


def inline_list(value: str) -> list[str]:
    value = value.strip()
    if not (value.startswith("[") and value.endswith("]")):
        return []
    inner = value[1:-1].strip()
    if not inner:
        return []
    return [x.strip().strip("'"") for x in inner.split(",") if x.strip()]


def parse_policy(path: pathlib.Path):
    scalars: dict[tuple[str, ...], str] = {}
    lists: dict[tuple[str, ...], list[str]] = {}
    stack: list[tuple[int, str]] = []

    for raw in path.read_text(errors="replace").splitlines():
        if not raw.strip() or raw.lstrip().startswith("#"):
            continue
        content = raw.split("#", 1)[0].rstrip()
        if not content.strip():
            continue
        indent = len(content) - len(content.lstrip(" "))
        stripped = content.strip()

        while stack and stack[-1][0] >= indent:
            stack.pop()

        if stripped.startswith("- "):
            key = tuple(x[1] for x in stack)
            lists.setdefault(key, []).append(stripped[2:].strip().strip("'""))
            continue

        if ":" not in stripped:
            continue
        key, value = stripped.split(":", 1)
        key = key.strip()
        value = value.strip()
        path_key = tuple(x[1] for x in stack) + (key,)

        if value:
            scalars[path_key] = value.strip("'"")
            values = inline_list(value)
            if values or value == "[]":
                lists[path_key] = values
        else:
            stack.append((indent, key))

    return scalars, lists


def scalar(scalars, *path, default="-"):
    return scalars.get(tuple(path), default)


def values(lists, *path):
    return lists.get(tuple(path), [])


def human_bool(raw: str) -> str:
    low = raw.strip().lower()
    if low == "true":
        return yes_no(True)
    if low == "false":
        return yes_no(False)
    return raw


def human_network(raw: str) -> str:
    if not is_pt_br():
        return raw
    return {
        "blocked": "bloqueada",
        "allowlist": "lista permitida",
        "unrestricted": "irrestrita",
    }.get(raw, raw)


def show_list(label: str, entries: list[str]):
    print(f"{label}:")
    if entries:
        for entry in entries:
            print(f"  - {entry}")
    else:
        print(f"  - {none_label()}")


def main() -> int:
    parser = argparse.ArgumentParser(add_help=True)
    parser.add_argument("--env", default=".env")
    parser.add_argument("--policy", default="config/policy.yaml")
    args = parser.parse_args()

    env_path = pathlib.Path(args.env)
    policy_path = pathlib.Path(args.policy)
    if not policy_path.is_file():
        raise SystemExit(t(f"policy not found: {policy_path}", f"policy não encontrada: {policy_path}"))

    env = env_values(env_path)
    scalars, lists = parse_policy(policy_path)

    whole = env.get("VPS_AGENT_WHOLE_HOST", "0") == "1"
    physical = env.get("VPS_AGENT_SCOPE_ROOT", t("(not set)", "(não definido)"))
    mode = scalar(scalars, "mode")
    full = human_bool(scalar(scalars, "features", "full_mode_enabled"))
    network = human_network(scalar(scalars, "network", "mode"))
    shell_enabled = human_bool(scalar(scalars, "shell", "enabled"))

    print(t("Effective authority summary", "Resumo da autoridade efetiva"))
    print("---------------------------")
    print(f"{t('Physical filesystem ceiling', 'Teto físico do filesystem')}: {physical}")
    print(f"{t('Whole-host mount override', 'Montagem de host inteiro')}: {yes_no(whole)}")
    print(f"{t('Policy mode', 'Modo da policy')}: {mode}")
    print(f"{t('Full feature enabled', 'Recurso Full habilitado')}: {full}")
    print(f"{t('Network mode', 'Modo de rede')}: {network}")
    print(f"{t('Scoped shell enabled', 'Shell confinado habilitado')}: {shell_enabled}")
    print()
    show_list(t("Filesystem read roots", "Raízes de leitura do filesystem"), values(lists, "filesystem", "read"))
    show_list(t("Filesystem write roots", "Raízes de escrita do filesystem"), values(lists, "filesystem", "write"))
    show_list(t("Shell cwd roots", "Raízes de trabalho do shell"), values(lists, "shell", "cwd_roots"))
    print()

    section_labels = {
        "services": t("services", "serviços"),
        "docker": "docker",
        "compose": "compose",
        "packages": t("packages", "pacotes"),
        "users": t("users", "usuários"),
        "groups": t("groups", "grupos"),
        "firewall": "firewall",
    }
    for section in ("services", "docker", "compose", "packages", "users", "groups", "firewall"):
        manage = values(lists, section, "manage")
        actions = values(lists, section, "actions")
        if manage or actions:
            print(f"{section_labels[section]}:")
            print("  " + t("manage", "gerenciar") + ": " + (", ".join(manage) if manage else none_label()))
            print("  " + t("actions", "ações") + ": " + (", ".join(actions) if actions else none_label()))

    print()
    print(t(
        "Filesystem ceiling and capabilities are separate dimensions.",
        "O teto do filesystem e as capacidades são dimensões separadas.",
    ))
    print(t(
        "Whole Host does not imply Full. Full does not imply unrestricted network.",
        "Whole Host não implica Full. Full não implica rede irrestrita.",
    ))
    print(t(
        "This summary is informational; Broker authorization remains authoritative.",
        "Este resumo é informativo; a autorização do Broker continua sendo a autoridade final.",
    ))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
