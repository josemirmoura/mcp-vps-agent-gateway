#!/usr/bin/env python3
"""Human-readable policy summary.

This is presentation only. The Broker's policy parser remains authoritative.
"""

from __future__ import annotations

import argparse
import pathlib
import shlex


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
            result[key] = value.strip("'\"")
    return result


def inline_list(value: str) -> list[str]:
    value = value.strip()
    if not (value.startswith("[") and value.endswith("]")):
        return []
    inner = value[1:-1].strip()
    if not inner:
        return []
    return [x.strip().strip("'\"") for x in inner.split(",") if x.strip()]


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
            lists.setdefault(key, []).append(stripped[2:].strip().strip("'\""))
            continue

        if ":" not in stripped:
            continue
        key, value = stripped.split(":", 1)
        key = key.strip()
        value = value.strip()
        path_key = tuple(x[1] for x in stack) + (key,)

        if value:
            scalars[path_key] = value.strip("'\"")
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


def show_list(label: str, entries: list[str]):
    print(f"{label}:")
    if entries:
        for entry in entries:
            print(f"  - {entry}")
    else:
        print("  - none")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--env", default=".env")
    parser.add_argument("--policy", default="config/policy.yaml")
    args = parser.parse_args()

    env_path = pathlib.Path(args.env)
    policy_path = pathlib.Path(args.policy)
    if not policy_path.is_file():
        raise SystemExit(f"policy not found: {policy_path}")

    env = env_values(env_path)
    scalars, lists = parse_policy(policy_path)

    whole = env.get("VPS_AGENT_WHOLE_HOST", "0") == "1"
    physical = env.get("VPS_AGENT_SCOPE_ROOT", "(not set)")
    mode = scalar(scalars, "mode")
    full = scalar(scalars, "features", "full_mode_enabled")
    network = scalar(scalars, "network", "mode")
    shell_enabled = scalar(scalars, "shell", "enabled")

    print("Effective authority summary")
    print("---------------------------")
    print(f"Physical filesystem ceiling: {physical}")
    print(f"Whole-host mount override:   {'yes' if whole else 'no'}")
    print(f"Policy mode:                 {mode}")
    print(f"Full feature enabled:        {full}")
    print(f"Network mode:                {network}")
    print(f"Scoped shell enabled:        {shell_enabled}")
    print()
    show_list("Filesystem read roots", values(lists, "filesystem", "read"))
    show_list("Filesystem write roots", values(lists, "filesystem", "write"))
    show_list("Shell cwd roots", values(lists, "shell", "cwd_roots"))
    print()

    for section in ("services", "docker", "compose", "packages", "users", "groups", "firewall"):
        manage = values(lists, section, "manage")
        actions = values(lists, section, "actions")
        if manage or actions:
            print(f"{section}:")
            print("  manage: " + (", ".join(manage) if manage else "none"))
            print("  actions: " + (", ".join(actions) if actions else "none"))

    print()
    print("Filesystem ceiling and capabilities are separate dimensions.")
    print("Whole Host does not imply Full. Full does not imply unrestricted network.")
    print("This summary is informational; Broker authorization remains authoritative.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
