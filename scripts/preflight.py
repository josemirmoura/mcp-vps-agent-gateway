#!/usr/bin/env python3
"""Portico MCP host prerequisite preflight.

Human output follows the resolved Portico locale. --json is stable and is not
localized so CI/automation can consume it.
"""

from __future__ import annotations

import argparse
import json
import os
import pathlib
import platform
import re
import shutil
import subprocess
import sys
from dataclasses import dataclass, asdict

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent / "lib"))
from i18n import is_pt_br, t  # noqa: E402


@dataclass
class Check:
    key: str
    label: str
    status: str  # ok | warn | fail
    detail: str = ""
    link: str = ""


def cmd(*args: str) -> tuple[int, str]:
    try:
        p = subprocess.run(
            args,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            timeout=12,
            check=False,
        )
        return p.returncode, p.stdout.strip()
    except (OSError, subprocess.TimeoutExpired) as exc:
        return 1, str(exc)


def major_version(raw: str) -> int | None:
    match = re.search(r"(\d+)", raw)
    return int(match.group(1)) if match else None


def command_check(name: str, label: str, link: str = "") -> Check:
    path = shutil.which(name)
    if path:
        return Check(name, label, "ok", path)
    return Check(name, label, "fail", t("not found", "não encontrado"), link)


def listening_ports() -> set[int]:
    if not shutil.which("ss"):
        return set()
    rc, out = cmd("ss", "-ltnH")
    if rc != 0:
        return set()
    ports: set[int] = set()
    for line in out.splitlines():
        parts = line.split()
        if len(parts) < 4:
            continue
        endpoint = parts[3].strip()
        match = re.search(r":(\d+)$", endpoint)
        if match:
            ports.add(int(match.group(1)))
    return ports


def traefik_containers() -> list[tuple[str, str, str]]:
    if not shutil.which("docker"):
        return []
    rc, out = cmd("docker", "ps", "--format", "{{.ID}}\t{{.Image}}\t{{.Names}}")
    if rc != 0:
        return []
    found = []
    for line in out.splitlines():
        fields = line.split("\t")
        if len(fields) >= 3 and "traefik" in (fields[1] + " " + fields[2]).lower():
            found.append((fields[0], fields[1], fields[2]))
    return found


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--public", action="store_true", help="check public edge prerequisites")
    parser.add_argument("--json", action="store_true", help="stable machine-readable output")
    args = parser.parse_args()

    checks: list[Check] = []

    linux = platform.system() == "Linux"
    checks.append(Check(
        "linux",
        t("Compatible Linux", "Linux compatível"),
        "ok" if linux else "fail",
        platform.platform(),
        "https://ubuntu.com/download/server" if not linux else "",
    ))

    machine = platform.machine().lower()
    arch_map = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64"}
    arch = arch_map.get(machine, machine)
    if arch == "amd64":
        checks.append(Check("architecture", t("Architecture", "Arquitetura"), "ok", "amd64"))
    elif arch == "arm64":
        checks.append(Check(
            "architecture",
            t("Architecture", "Arquitetura"),
            "warn",
            t("arm64 builds are available; full host-runtime acceptance is not yet equivalent to amd64",
              "há builds arm64; a validação completa em host ainda não é equivalente à amd64"),
        ))
    else:
        checks.append(Check(
            "architecture",
            t("Architecture", "Arquitetura"),
            "fail",
            arch,
            "https://github.com/josemirmoura/mcp-vps-agent-gateway/blob/main/docs/compatibility.md",
        ))

    base_commands = [
        ("git", "Git", "https://git-scm.com/download/linux"),
        ("curl", "curl", "https://curl.se/download.html"),
        ("openssl", "OpenSSL", "https://www.openssl.org/source/"),
        ("python3", "Python 3", "https://www.python.org/downloads/"),
        ("ss", "ss (iproute2)", ""),
        ("getent", "getent", ""),
    ]
    checks.extend(command_check(name, label, link) for name, label, link in base_commands)

    docker_path = shutil.which("docker")
    if not docker_path:
        checks.append(Check(
            "docker",
            "Docker Engine",
            "fail",
            t("not found", "não encontrado"),
            "https://docs.docker.com/engine/install/",
        ))
        checks.append(Check(
            "compose",
            "Docker Compose v2",
            "fail",
            t("Docker is required first", "o Docker precisa ser instalado primeiro"),
            "https://docs.docker.com/compose/install/linux/",
        ))
    else:
        rc, version = cmd("docker", "version", "--format", "{{.Server.Version}}")
        daemon_ok = rc == 0 and bool(version)
        major = major_version(version) if daemon_ok else None
        if not daemon_ok:
            checks.append(Check(
                "docker",
                "Docker Engine",
                "fail",
                t("installed, but the Docker daemon is not accessible", "instalado, mas o daemon Docker não está acessível"),
                "https://docs.docker.com/engine/install/",
            ))
        elif major is not None and major < 24:
            checks.append(Check(
                "docker",
                "Docker Engine",
                "fail",
                t(f"version {version}; Portico requires 24+", f"versão {version}; o Portico requer 24+"),
                "https://docs.docker.com/engine/install/",
            ))
        else:
            checks.append(Check("docker", "Docker Engine", "ok", version))

        rc, compose = cmd("docker", "compose", "version", "--short")
        compose_major = major_version(compose) if rc == 0 else None
        if rc != 0 or compose_major != 2:
            checks.append(Check(
                "compose",
                "Docker Compose v2",
                "fail",
                compose or t("not available", "não disponível"),
                "https://docs.docker.com/compose/install/linux/",
            ))
        else:
            checks.append(Check("compose", "Docker Compose v2", "ok", compose))

    systemd_ok = bool(shutil.which("systemctl")) and pathlib.Path("/run/systemd/system").is_dir()
    checks.append(Check(
        "systemd",
        "systemd",
        "ok" if systemd_ok else "fail",
        t("host systemd detected", "systemd do host detectado") if systemd_ok
        else t("Portico host service/job features require systemd", "os recursos de serviços/jobs do Portico requerem systemd"),
        "https://ubuntu.com/download/server" if not systemd_ok else "",
    ))

    mem_kib = 0
    try:
        for line in pathlib.Path("/proc/meminfo").read_text().splitlines():
            if line.startswith("MemTotal:"):
                mem_kib = int(line.split()[1])
                break
    except (OSError, ValueError):
        pass
    mem_gib = mem_kib / 1024 / 1024 if mem_kib else 0
    if mem_kib and mem_kib < 2 * 1024 * 1024:
        checks.append(Check(
            "memory",
            t("Total memory", "Memória total"),
            "fail",
            f"{mem_gib:.1f} GiB; " + t("2 GiB minimum for bundled OAuth", "mínimo de 2 GiB para o OAuth embutido"),
            "https://zitadel.com/docs/self-hosting/manage/requirements",
        ))
    else:
        checks.append(Check(
            "memory",
            t("Total memory", "Memória total"),
            "ok",
            f"{mem_gib:.1f} GiB" if mem_kib else t("could not determine", "não foi possível determinar"),
        ))

    try:
        free = shutil.disk_usage(pathlib.Path.cwd()).free / 1024 / 1024 / 1024
        checks.append(Check(
            "disk",
            t("Free disk at checkout", "Disco livre no checkout"),
            "ok" if free >= 1 else "warn",
            f"{free:.1f} GiB",
        ))
    except OSError:
        pass

    rc, inside = cmd("git", "rev-parse", "--is-inside-work-tree") if shutil.which("git") else (1, "")
    checks.append(Check(
        "git_checkout",
        t("Git checkout", "Checkout Git"),
        "ok" if rc == 0 and inside == "true" else "fail",
        t("repository checkout detected", "checkout do repositório detectado")
        if rc == 0 and inside == "true"
        else t("run the installer from the cloned Portico repository", "execute o instalador dentro do repositório Portico clonado"),
    ))

    if args.public and docker_path:
        traefik = traefik_containers()
        if len(traefik) == 1:
            checks.append(Check(
                "edge_proxy",
                t("Public edge proxy", "Proxy público"),
                "ok",
                t(
                    f"existing Traefik detected ({traefik[0][2]}); Portico will reuse it",
                    f"Traefik existente detectado ({traefik[0][2]}); o Portico irá reutilizá-lo",
                ),
            ))
        elif len(traefik) > 1:
            checks.append(Check(
                "edge_proxy",
                t("Public edge proxy", "Proxy público"),
                "warn",
                t(
                    "multiple Traefik containers detected; the edge network must be selected explicitly",
                    "mais de um Traefik detectado; será necessário selecionar explicitamente a rede de borda",
                ),
            ))
        else:
            ports = listening_ports()
            blocked = sorted({80, 443} & ports)
            if blocked:
                checks.append(Check(
                    "edge_proxy",
                    t("Public edge proxy", "Proxy público"),
                    "fail",
                    t(
                        f"no reusable Traefik found and host ports {','.join(map(str, blocked))} are already in use",
                        f"nenhum Traefik reutilizável encontrado e as portas {','.join(map(str, blocked))} já estão em uso",
                    ),
                ))
            else:
                checks.append(Check(
                    "edge_proxy",
                    t("Public edge proxy", "Proxy público"),
                    "ok",
                    t(
                        "no Traefik found; Portico will provision its bundled Traefik automatically",
                        "nenhum Traefik encontrado; o Portico instalará automaticamente seu Traefik embutido",
                    ),
                ))

    failures = [x for x in checks if x.status == "fail"]

    if args.json:
        print(json.dumps({
            "ok": not failures,
            "language": os.environ.get("VPS_AGENT_LANG", "en"),
            "checks": [asdict(x) for x in checks],
        }, indent=2))
        return 0 if not failures else 1

    print(t("Checking Portico MCP prerequisites...", "Verificando requisitos do Portico MCP..."))
    print()
    width = max(len(x.label) for x in checks)
    for item in checks:
        badge = {"ok": "OK", "warn": t("WARN", "ATENÇÃO"), "fail": t("MISSING", "FALTA")}[item.status]
        dots = "." * max(2, width - len(item.label) + 4)
        print(f"  {item.label} {dots} {badge}" + (f"  {item.detail}" if item.detail else ""))

    if failures:
        print()
        print(t(
            "Portico cannot start yet. Install/fix the required items below and run the installer again:",
            "O Portico ainda não pode iniciar. Corrija os itens obrigatórios abaixo e execute o instalador novamente:",
        ))
        for item in failures:
            print(f"  - {item.label}: {item.detail}")
            if item.link:
                print("    " + t("Official documentation", "Documentação oficial") + f": {item.link}")
        return 1

    print()
    print(t("Prerequisites: OK", "Pré-requisitos: OK"))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
