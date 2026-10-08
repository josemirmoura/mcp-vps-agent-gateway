#!/usr/bin/env python3
"""Provision operator portal credentials interactively without printing secrets.

Never invoke from ChatGPT or CI with a real password. Run in trusted SSH.
Requires --apply and an existing .env in the current Community installation.
"""
from __future__ import annotations
import argparse
import getpass
import fcntl
import re
import hashlib
import os
from pathlib import Path
import secrets
import stat
from urllib.parse import urlsplit

KEYS = ("VPS_AGENT_OPERATOR_APPROVAL_TOKEN", "PORTICO_OPERATOR_PASSWORD_SCRYPT",
        "PORTICO_OPERATOR_PUBLIC_ORIGIN", "VPS_AGENT_OPERATOR_PORTAL_URL")

def validate_origin(value: str) -> str:
    u = urlsplit(value)
    if (u.scheme != "https" or not u.hostname or u.username or u.password or
            u.path not in ("", "/") or u.query or u.fragment or not u.netloc):
        raise ValueError("Origem deve ser somente https://hostname[:porta], sem caminho ou credenciais")
    if not re.fullmatch(r"[A-Za-z0-9.-]+", u.hostname) or u.hostname.startswith("-"):
        raise ValueError("Nome DNS inválido")
    try:
        _ = u.port
    except ValueError as exc:
        raise ValueError("Porta HTTPS inválida") from exc
    return f"https://{u.netloc}"

def prepare(text: str, values: dict[str,str]) -> str:
    lines=text.splitlines()
    if any(any(line.startswith(key+"=") for key in KEYS) for line in lines):
        raise ValueError("Configuração da Central já existe: rotação exige procedimento separado")
    if lines and lines[-1].strip():
        lines.append("")
    lines.append("# Pórtico Operator Portal: credenciais locais, nunca enviar ao chat")
    lines.extend(key+"="+values[key] for key in KEYS)
    return "\n".join(lines)+"\n"

def main() -> int:
    p=argparse.ArgumentParser()
    p.add_argument("--origin", required=True, help="HTTPS público reservado à Central, sem caminho")
    p.add_argument("--apply", action="store_true", help="gravar no .env após confirmação explícita")
    args=p.parse_args()
    origin=validate_origin(args.origin)
    if not args.apply:
        print(f"DRY RUN: origem aceita {origin}; sem mudanças.")
        return 0
    env=Path(".env")
    if not env.is_file() or env.is_symlink():
        raise SystemExit("Arquivo .env local ausente ou link simbólico: operação cancelada")
    mode=env.stat().st_mode
    if mode & 0o077:
        raise SystemExit("Proteja .env (modo 0600) antes de provisionar.")
    pwd=getpass.getpass("Crie a senha exclusiva da Central (mínimo 16 caracteres): ")
    confirm=getpass.getpass("Repita a senha: ")
    if len(pwd)<16 or pwd!=confirm:
        raise SystemExit("Senha curta ou confirmação diferente. Nada gravado.")
    salt=secrets.token_bytes(24)
    digest=hashlib.scrypt(pwd.encode("utf-8"),salt=salt,n=2**14,r=8,p=1,dklen=32)
    values={
       "VPS_AGENT_OPERATOR_APPROVAL_TOKEN": secrets.token_urlsafe(48),
       "PORTICO_OPERATOR_PASSWORD_SCRYPT": salt.hex()+":"+digest.hex(),
       "PORTICO_OPERATOR_PUBLIC_ORIGIN": origin,
       "VPS_AGENT_OPERATOR_PORTAL_URL": origin+"/operator",
    }
    baseline=env.read_text()
    new=prepare(baseline,values)
    suffix=new[len(baseline):]
    # Lock, recheck and append while holding an exclusive lock. No temp file
    # or duplicate plaintext credentials should be written to disk.
    fd=os.open(env,os.O_RDWR|os.O_APPEND|os.O_NOFOLLOW)
    try:
        fcntl.flock(fd,fcntl.LOCK_EX)
        with os.fdopen(fd,"r+",encoding="utf-8") as out:
            if out.read()!=baseline:
                raise SystemExit("O arquivo .env mudou; tente novamente.")
            out.write(suffix)
            out.flush()
            os.fsync(out.fileno())
    finally:
        # fdopen closes fd on success/error; flock released automatically.
        pass
    print("Credenciais da Central configuradas no .env local (sem iniciar contêineres).")
    return 0

if __name__=="__main__":
    raise SystemExit(main())
