#!/usr/bin/env python3
"""Trusted SSH/operator approval fallback for MCP clients without elicitation.

This program never receives an MCP approval token. It calls the existing
Docker-local Broker admin commands, using the operator's Docker authority.
The model-facing MCP API cannot invoke this operator CLI.
"""
from __future__ import annotations

import argparse
import datetime as dt
import json
import pathlib
import re
import shlex
import subprocess
import sys
import unicodedata

ROOT = pathlib.Path(__file__).resolve().parent.parent
COMPOSE = ["docker", "compose", "-f", "compose.yaml"]
BROKER = [*COMPOSE, "exec", "-T", "broker", "/usr/local/bin/vps-agent"]
REQUEST_ID = re.compile(r"\Aapr_[a-zA-Z0-9_-]{8,100}\Z")


def cli_call(*args: str) -> object:
    proc = subprocess.run(
        [*BROKER, *args], cwd=ROOT, capture_output=True, text=True,
        timeout=25, check=False,
    )
    if proc.returncode != 0:
        raise RuntimeError(
            "O Broker recusou a operação. Confira se o Pórtico está ativo e "
            "se este operador tem acesso autorizado ao Docker."
        )
    try:
        result = json.loads(proc.stdout)
    except ValueError as exc:
        raise RuntimeError("Resposta inesperada do Broker") from exc
    if not isinstance(result, dict) or not result.get("ok"):
        raise RuntimeError("O Broker não confirmou a operação")
    return result.get("result")


def safe_text(value: object) -> str:
    if not isinstance(value, str) or not value or len(value) > 1024:
        raise ValueError("campo de autorização inválido")
    if any(
        not ch.isprintable()
        or unicodedata.category(ch) in {"Cf", "Zl", "Zp"}
        for ch in value
    ):
        raise ValueError("campo de autorização contém caracteres invisíveis")
    return value


def parse_expiry(value: object) -> dt.datetime:
    if not isinstance(value, str):
        raise ValueError("expiração inválida")
    try:
        stamp = dt.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError as exc:
        raise ValueError("expiração inválida") from exc
    if stamp.tzinfo is None:
        raise ValueError("expiração sem fuso")
    return stamp


def request_detail(row: dict) -> dict | None:
    """Validate and classify an unexpired root/sensitive request; fail closed."""
    if not isinstance(row, dict) or row.get("Status") != "pending":
        return None
    rid = row.get("ID")
    if not isinstance(rid, str) or not REQUEST_ID.fullmatch(rid):
        return None
    try:
        expires = parse_expiry(row.get("ExpiresAt"))
        if expires <= dt.datetime.now(dt.timezone.utc):
            return None
        subject = safe_text(row.get("Subject"))
        kind = row.get("Kind")
        ttl_ns = row.get("TTL")
        if not isinstance(ttl_ns, int) or isinstance(ttl_ns, bool) or ttl_ns < 0:
            return None
        if kind == "root":
            target = safe_text(row.get("Resource"))
            if not target.startswith("/"):
                return None
            access = row.get("Access")
            if access not in ("read", "work", "compose"):
                return None
            title = "Pasta"
        elif kind == "capability":
            caps = row.get("Capabilities")
            if not isinstance(caps, list) or len(caps) != 1 or not isinstance(caps[0], str):
                return None
            matched = re.fullmatch(r"sensitive\.(read|work):([a-fA-F0-9]+)", caps[0])
            if not matched:
                return None  # elevation requires a separate step-up path
            access = matched.group(1)
            target = safe_text(bytes.fromhex(matched.group(2)).decode("utf-8"))
            if not target.startswith("/"):
                return None
            title = "Arquivo protegido"
        else:
            return None
    except (ValueError, UnicodeDecodeError):
        return None
    return {
        "id": rid, "subject": subject, "title": title, "target": target,
        "access": access, "ttl_ns": ttl_ns, "expires": expires, "kind": kind,
    }


def list_requests() -> list[dict]:
    result = cli_call("approvals")
    if result is None:  # Go encodes a nil slice as JSON null when the queue is empty.
        return []
    if not isinstance(result, list):
        raise RuntimeError("Lista de aprovações inesperada")
    return [detail for row in result if (detail := request_detail(row))]


def physical_ceiling() -> str:
    # Read only this non-secret value, never print or export the .env contents.
    try:
        for line in (ROOT / ".env").read_text().splitlines():
            if not line.startswith("VPS_AGENT_SCOPE_ROOT="):
                continue
            tokens = shlex.split(line.split("=", 1)[1])
            return tokens[0] if len(tokens) == 1 else ""
    except (OSError, ValueError):
        pass
    return ""


def present(item: dict) -> None:
    expires = item["expires"].astimezone(dt.timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    ttl = "Permanente, até revogação" if item["ttl_ns"] == 0 else f'{item["ttl_ns"] // 1_000_000_000} segundos'
    print(f'\nPEDIDO: {item["id"]}')
    print(f'Operador/identidade MCP: {item["subject"]}')
    print(f'{item["title"]}: {item["target"]}')
    print(f'Perfil: {item["access"]}')
    print(f'Duração da permissão: {ttl}')
    print(f'Pedido expira: {expires}')
    ceiling = physical_ceiling()
    if item["kind"] == "root" and ceiling and item["target"] == ceiling:
        print(f'ATENÇÃO: ESCOPO AMPLO! Abrange todas as pastas atuais e futuras em {ceiling}.')


def decide(item: dict) -> bool:
    if not sys.stdin.isatty():
        print("A decisão exige um terminal SSH interativo.", file=sys.stderr)
        return False
    present(item)
    print("\nEscolha uma das opções digitando a frase EXATA:")
    yes = f'APROVAR {item["id"]}'
    no = f'NEGAR {item["id"]}'
    print(yes)
    print(no)
    print("Qualquer outra entrada cancela sem conceder acesso.")
    try:
        typed = input("Sua decisão: ").strip()
    except (EOFError, KeyboardInterrupt):
        print("\nCancelado.")
        return False
    if typed not in (yes, no):
        print("Cancelado. Nenhum acesso concedido.")
        return False

    # Re-fetch before changing state; never act on an expired or replaced request.
    refreshed = next((x for x in list_requests() if x["id"] == item["id"]), None)
    if refreshed != item:
        print("Pedido alterado ou expirado. Decisão cancelada.")
        return False
    action = "approve" if typed == yes else "deny"
    response = cli_call(action, "--request", item["id"])
    if not isinstance(response, dict) or response.get("status") != ("approved" if action == "approve" else "denied"):
        print("O Broker não confirmou a decisão. Verifique a auditoria.")
        return False
    print("Acesso autorizado e auditado." if action == "approve" else "Pedido negado e auditado.")
    return True


def main() -> int:
    parser = argparse.ArgumentParser(description="Aprovação segura do Pórtico via terminal SSH.")
    parser.add_argument("--request", help="ID apr_... específico recebido no ChatGPT")
    parser.add_argument("--list", action="store_true", help="Somente listar pedidos pendentes")
    args = parser.parse_args()
    if args.request and not REQUEST_ID.fullmatch(args.request):
        parser.error("ID de pedido inválido")
    try:
        items = list_requests()
        print("PÓRTICO MCP | Aprovação do operador")
        print("Canal: Broker local autenticado, não a conversa da IA.")
        if not items:
            print("Nenhum pedido pendente de pasta/arquivo protegido.")
            return 0
        if args.list:
            for item in items:
                present(item)
            return 0
        if args.request:
            selected = next((x for x in items if x["id"] == args.request), None)
            if selected is None:
                print("Pedido não encontrado ou expirado. Nada foi autorizado.")
                return 1
        else:
            for n, item in enumerate(items, 1):
                print(f'{n}. {item["id"]} | {item["access"]} | {item["target"]}')
            if not sys.stdin.isatty():
                print("Use um terminal interativo para escolher.")
                return 1
            try:
                choice = input("Número do pedido (Enter cancela): ").strip()
            except (EOFError, KeyboardInterrupt):
                print("\nCancelado.")
                return 1
            if not choice.isdecimal() or not 1 <= int(choice) <= len(items):
                print("Cancelado.")
                return 1
            selected = items[int(choice) - 1]
        return 0 if decide(selected) else 1
    except (RuntimeError, subprocess.TimeoutExpired, OSError) as exc:
        print(f"Falha segura: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
