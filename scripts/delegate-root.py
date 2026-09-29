#!/usr/bin/env python3
"""Safely add/remove logical filesystem roots inside the configured physical ceiling.

This helper edits only the root lists in config/policy.yaml. The Broker remains
authoritative and re-validates every path at startup.
"""

from __future__ import annotations

import argparse
import os
import pathlib
import shlex
import shutil
import stat
import sys
import time
from typing import Iterable


MANAGED_LISTS = (
    ("filesystem", "read"),
    ("filesystem", "write"),
    ("shell", "cwd_roots"),
    ("compose", "inspect"),
    ("compose", "manage"),
)


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


def parse_lists(text: str) -> dict[tuple[str, ...], list[str]]:
    lists: dict[tuple[str, ...], list[str]] = {}
    stack: list[tuple[int, str]] = []
    for raw in text.splitlines():
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
        vals = inline_list(value)
        if vals or value == "[]":
            lists[path_key] = vals
        if not value:
            stack.append((indent, key))
    return lists


def find_key(lines: list[str], target: tuple[str, ...]) -> tuple[int, int]:
    stack: list[tuple[int, str]] = []
    for idx, raw in enumerate(lines):
        if not raw.strip() or raw.lstrip().startswith("#"):
            continue
        content = raw.split("#", 1)[0].rstrip()
        if not content.strip():
            continue
        indent = len(content) - len(content.lstrip(" "))
        stripped = content.strip()
        while stack and stack[-1][0] >= indent:
            stack.pop()
        if stripped.startswith("- ") or ":" not in stripped:
            continue
        key, value = stripped.split(":", 1)
        key = key.strip()
        value = value.strip()
        current = tuple(x[1] for x in stack) + (key,)
        if current == target:
            return idx, indent
        if not value:
            stack.append((indent, key))
    raise ValueError(f"policy key not found: {'.'.join(target)}")


def replace_list(text: str, target: tuple[str, ...], values: Iterable[str]) -> str:
    lines = text.splitlines()
    idx, indent = find_key(lines, target)
    end = idx + 1
    while end < len(lines):
        raw = lines[end]
        if not raw.strip() or raw.lstrip().startswith("#"):
            end += 1
            continue
        content = raw.split("#", 1)[0].rstrip()
        current_indent = len(content) - len(content.lstrip(" "))
        if current_indent <= indent:
            break
        end += 1

    values = list(dict.fromkeys(values))
    key = target[-1]
    prefix = " " * indent
    if values:
        replacement = [f"{prefix}{key}:"] + [f"{prefix}  - {value}" for value in values]
    else:
        replacement = [f"{prefix}{key}: []"]
    return "\n".join(lines[:idx] + replacement + lines[end:]) + "\n"


def canonical_path(raw: str) -> str:
    if not os.path.isabs(raw):
        raise ValueError("delegated root must be an absolute path")
    normalized = os.path.normpath(raw)
    if normalized != raw:
        raise ValueError("delegated root must be canonical (no '.', '..' or trailing slash)")
    return normalized


def within(ceiling: str, target: str) -> bool:
    try:
        return os.path.commonpath([ceiling, target]) == ceiling
    except ValueError:
        return False


def reject_symlink_ancestor(target: str) -> None:
    cur = pathlib.Path(target)
    while not cur.exists() and cur.parent != cur:
        cur = cur.parent
    if not cur.exists():
        raise ValueError(f"no existing ancestor for delegated root: {target}")
    resolved = pathlib.Path(os.path.realpath(cur))
    if resolved != cur:
        raise ValueError(f"delegated root traverses a symlinked ancestor: {cur}")


def profile_targets(access: str) -> tuple[tuple[str, ...], ...]:
    if access == "read":
        return (("filesystem", "read"),)
    if access == "work":
        return (
            ("filesystem", "read"),
            ("filesystem", "write"),
            ("shell", "cwd_roots"),
        )
    if access == "compose":
        return (
            ("filesystem", "read"),
            ("filesystem", "write"),
            ("shell", "cwd_roots"),
            ("compose", "inspect"),
            ("compose", "manage"),
        )
    raise ValueError(f"unsupported access profile: {access}")


def show(lists: dict[tuple[str, ...], list[str]]) -> None:
    roots: list[str] = []
    for key in MANAGED_LISTS:
        for item in lists.get(key, []):
            if item not in roots:
                roots.append(item)
    if not roots:
        print("No delegated roots.")
        return

    print("Delegated roots")
    print("---------------")
    for root in roots:
        flags = []
        if root in lists.get(("filesystem", "read"), []):
            flags.append("read")
        if root in lists.get(("filesystem", "write"), []):
            flags.append("write")
        if root in lists.get(("shell", "cwd_roots"), []):
            flags.append("shell")
        if root in lists.get(("compose", "inspect"), []) or root in lists.get(("compose", "manage"), []):
            flags.append("compose")
        print(f"{root}: {', '.join(flags)}")


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Delegate or revoke logical roots without widening the whole physical ceiling."
    )
    parser.add_argument("action", choices=("add", "remove", "list"))
    parser.add_argument("root", nargs="?")
    parser.add_argument("--access", choices=("read", "work", "compose"), default="work")
    parser.add_argument("--env", default=".env")
    parser.add_argument("--policy", default="config/policy.yaml")
    parser.add_argument(
        "--allow-ceiling",
        action="store_true",
        help="allow delegating the entire physical ceiling itself (broad authority)",
    )
    parser.add_argument("--no-backup", action="store_true")
    args = parser.parse_args()

    env_path = pathlib.Path(args.env)
    policy_path = pathlib.Path(args.policy)
    if not policy_path.is_file():
        raise SystemExit(f"policy not found: {policy_path}")

    text = policy_path.read_text(errors="strict")
    lists = parse_lists(text)

    if args.action == "list":
        show(lists)
        return 0

    if not args.root:
        raise SystemExit("root is required for add/remove")

    try:
        root = canonical_path(args.root)
    except ValueError as exc:
        raise SystemExit(str(exc)) from exc

    env = env_values(env_path)
    ceiling_raw = env.get("VPS_AGENT_SCOPE_ROOT", "")
    if not ceiling_raw:
        raise SystemExit("VPS_AGENT_SCOPE_ROOT is not set in the env file")
    try:
        ceiling = canonical_path(ceiling_raw)
    except ValueError as exc:
        raise SystemExit(f"invalid VPS_AGENT_SCOPE_ROOT: {exc}") from exc

    if not within(ceiling, root):
        raise SystemExit(f"delegated root {root} is outside physical ceiling {ceiling}")
    if root == ceiling and not args.allow_ceiling:
        raise SystemExit(
            f"refusing to delegate the entire physical ceiling {ceiling}; "
            "use --allow-ceiling only if that broad authority is intentional"
        )
    try:
        reject_symlink_ancestor(root)
    except ValueError as exc:
        raise SystemExit(str(exc)) from exc

    updated = text
    current = parse_lists(updated)

    if args.action == "add":
        targets = set(profile_targets(args.access))
        for key in MANAGED_LISTS:
            values = list(current.get(key, []))
            if key in targets and root not in values:
                values.append(root)
                updated = replace_list(updated, key, values)
                current = parse_lists(updated)
    else:
        for key in MANAGED_LISTS:
            values = [x for x in current.get(key, []) if x != root]
            if values != current.get(key, []):
                updated = replace_list(updated, key, values)
                current = parse_lists(updated)

    if updated == text:
        print("Policy already has the requested state.")
        show(parse_lists(updated))
        return 0

    if not args.no_backup:
        backup_dir = policy_path.parent.parent / "backups"
        backup_dir.mkdir(mode=0o700, parents=True, exist_ok=True)
        stamp = time.strftime("%Y%m%d-%H%M%S", time.gmtime())
        backup = backup_dir / f"policy-before-root-change-{stamp}.yaml"
        shutil.copy2(policy_path, backup)
        os.chmod(backup, stat.S_IRUSR | stat.S_IWUSR)
        print(f"Backup: {backup}")

    tmp = policy_path.with_name(policy_path.name + ".tmp")
    tmp.write_text(updated)
    os.chmod(tmp, stat.S_IRUSR | stat.S_IWUSR)
    os.replace(tmp, policy_path)

    verb = "Delegated" if args.action == "add" else "Revoked"
    print(f"{verb}: {root}")
    show(parse_lists(updated))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
