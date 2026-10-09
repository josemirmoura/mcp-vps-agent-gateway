#!/usr/bin/env python3
"""Local-only operator portal backend. Reverse-proxy with HTTPS before remote access.

Security boundary: explicitly configured operator password, CSRF-bound sessions,
loopback-only listener and existing Broker admin decision path. The service must
run in a separately hardened context; never publish port 8765 directly.
"""
from __future__ import annotations
import datetime as dt
import hashlib
import hmac
import http.cookies
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import re
import secrets
import socket
import threading
import time
from urllib.parse import urlsplit

HERE = Path(__file__).resolve().parent
REPO = HERE.parents[1]
from datetime import timezone
import unicodedata
import subprocess
import sys
import importlib.util
SPEC = importlib.util.spec_from_file_location("operator_approvals", REPO / "scripts/operator-approvals.py")
assert SPEC and SPEC.loader
operator = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(operator)

class OperatorIPC:
    @staticmethod
    def call(action, request_id=None):
        path = os.environ.get("PORTICO_OPERATOR_SOCKET", "/run/portico-operator/operator.sock")
        token = os.environ.get("PORTICO_OPERATOR_APPROVAL_TOKEN", "")
        if len(token) < 32 or not path.startswith("/"):
            raise RuntimeError("operator IPC unconfigured")
        tools = {"approvals": "admin.approval.list", "approve": "admin.approval.approve", "deny": "admin.approval.deny"}
        if action not in tools:
            raise RuntimeError("unsupported operator operation")
        payload = {"id": secrets.token_urlsafe(12), "tool": tools[action], "admin_token": token}
        if request_id is not None:
            if not ID_RE.fullmatch(request_id):
                raise RuntimeError("invalid request id")
            payload["args"] = {"request_id": request_id}
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as client:
            client.settimeout(5)
            client.connect(path)
            client.sendall(json.dumps(payload).encode() + bytes([10]))
            with client.makefile("rb") as source:
                data = source.readline(131072)
        reply = json.loads(data)
        if not isinstance(reply, dict) or not reply.get("ok"):
            raise RuntimeError("operator Broker rejected operation")
        return reply.get("result")
    @staticmethod
    def list_requests():
        rows = OperatorIPC.call("approvals") or []
        if not isinstance(rows, list):
            raise RuntimeError("invalid approval response")
        return [item for row in rows if (item := operator.request_detail(row))]

SESSIONS: dict[str, tuple[float, str]] = {}
LOCK = threading.Lock()
LOGIN_FAILURES: dict[str, list[float]] = {}
ID_RE = re.compile(r"^apr_[A-Za-z0-9_-]{8,100}$")
API_RE = re.compile(r"^/api/operator/approvals/(apr_[A-Za-z0-9_-]{8,100})(/decision)?$")
UI_PATH = REPO / "web/operator-approval/index.html"
COOKIE = "portico_operator"
MAX_AGE = 900


def verify_password(password: str) -> bool:
    """Password verifier is scrypt salt:hex digest, kept outside the repository."""
    encoded = os.environ.get("PORTICO_OPERATOR_PASSWORD_SCRYPT", "")
    try:
        salt_hex, digest_hex = encoded.split(":", 1)
        salt, expected = bytes.fromhex(salt_hex), bytes.fromhex(digest_hex)
        if len(salt) < 16 or len(expected) != 32:
            return False
        actual = hashlib.scrypt(password.encode(), salt=salt, n=2**14, r=8, p=1, dklen=32)
        return hmac.compare_digest(actual, expected)
    except (ValueError, TypeError):
        return False


def session_from_cookie(raw: str) -> tuple[str, str] | None:
    jar = http.cookies.SimpleCookie()
    try:
        jar.load(raw)
    except http.cookies.CookieError:
        return None
    if COOKIE not in jar:
        return None
    sid = jar[COOKIE].value
    with LOCK:
        state = SESSIONS.get(sid)
        if state and state[0] > time.monotonic():
            return sid, state[1]
    return None


class Handler(BaseHTTPRequestHandler):
    server_version = "PorticoOperator/0.1"
    def headers_base(self, typ="application/json; charset=utf-8"):
        self.send_header("Content-Type", typ)
        self.send_header("Cache-Control", "no-store")
        self.send_header("X-Content-Type-Options", "nosniff")
        self.send_header("X-Frame-Options", "DENY")
        self.send_header("Referrer-Policy", "no-referrer")
        self.send_header("Content-Security-Policy", "default-src 'none'; connect-src 'self'; script-src 'self'; style-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
    def reply(self, status: int, obj: dict, extra_headers: dict | None = None):
        payload = json.dumps(obj, ensure_ascii=False).encode()
        self.send_response(status)
        self.headers_base()
        if extra_headers:
            for key, value in extra_headers.items():
                self.send_header(key, value)
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)
    def origin_ok(self):
        origin = self.headers.get("Origin")
        allowed = os.environ.get("PORTICO_OPERATOR_PUBLIC_ORIGIN", "")
        return bool(origin and allowed and origin == allowed)
    def body(self):
        length = self.headers.get("Content-Length", "")
        if not length.isdecimal() or int(length) > 2048:
            raise ValueError("invalid body length")
        obj = json.loads(self.rfile.read(int(length)))
        if not isinstance(obj, dict):
            raise ValueError("body must be object")
        return obj
    def session(self):
        return session_from_cookie(self.headers.get("Cookie", ""))
    def do_GET(self):
        path = urlsplit(self.path).path
        if path in ("/", "/operator", "/operator/"):
            page = UI_PATH.read_bytes()
            self.send_response(200)
            self.headers_base("text/html; charset=utf-8")
            self.send_header("Content-Length", str(len(page)))
            self.end_headers()
            self.wfile.write(page)
            return
        match = API_RE.fullmatch(path)
        if match and not match.group(2):
            session = self.session()
            if session is None:
                self.reply(401, {"error": "login required"})
                return
            try:
                item = next((x for x in OperatorIPC.list_requests() if x["id"] == match.group(1)), None)
                if item is None:
                    self.reply(404, {"error": "not found or expired"})
                    return
                ceiling = os.environ.get("PORTICO_OPERATOR_PHYSICAL_CEILING", "")
                if not ceiling.startswith("/"):
                    raise RuntimeError("physical ceiling unavailable")
                self.reply(200, {
                    "request_id": item["id"], "node": os.environ.get("PORTICO_OPERATOR_NODE_LABEL", "Local"),
                    "subject": item["subject"], "resource": item["target"],
                    "access": item["access"], "ttl_label": "Até revogação" if item["ttl_ns"] == 0 else f'{item["ttl_ns"] // 1_000_000_000} segundos',
                    "expires_at": item["expires"].isoformat(), "ceiling_wide": item["kind"] == "root" and bool(ceiling) and item["target"] == ceiling,
                    "status": "pending", "csrf_token": session[1]})
            except (RuntimeError, OSError, ValueError):
                self.reply(503, {"error": "broker unavailable"})
            return
        self.reply(404, {"error": "not found"})
    def do_POST(self):
        if not self.origin_ok():
            self.reply(403, {"error": "origin denied"})
            return
        path = urlsplit(self.path).path
        if path == "/api/operator/login":
            # Bound rate limiting by remote address; TLS proxy MUST preserve access controls.
            peer = self.client_address[0]
            now = time.monotonic()
            with LOCK:
                failures = [t for t in LOGIN_FAILURES.get(peer, []) if now - t < 300]
                LOGIN_FAILURES[peer] = failures
                limited = len(failures) >= 5
            if limited:
                self.reply(429, {"error": "too many attempts"})
                return
            try:
                supplied = self.body().get("password")
                if not isinstance(supplied, str) or len(supplied) > 1024 or not verify_password(supplied):
                    with LOCK:
                        LOGIN_FAILURES.setdefault(peer, []).append(time.monotonic())
                    self.reply(403, {"error": "invalid credentials"})
                    return
                with LOCK:
                    LOGIN_FAILURES.pop(peer, None)
                sid, csrf = secrets.token_urlsafe(32), secrets.token_urlsafe(32)
                with LOCK:
                    if len(SESSIONS) > 500:
                        SESSIONS.clear()
                    SESSIONS[sid] = (time.monotonic() + MAX_AGE, csrf)
                self.reply(200, {"ok": True}, {"Set-Cookie": f"{COOKIE}={sid}; HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age={MAX_AGE}"})
            except (ValueError, json.JSONDecodeError):
                self.reply(400, {"error": "invalid payload"})
            return
        match = API_RE.fullmatch(path)
        if match and match.group(2):
            session = self.session()
            if session is None:
                self.reply(401, {"error": "login required"})
                return
            if not hmac.compare_digest(self.headers.get("X-CSRF-Token", ""), session[1]):
                self.reply(403, {"error": "csrf denied"})
                return
            try:
                request = self.body()
                decision = request.get("decision")
                if decision not in ("approve", "deny"):
                    self.reply(400, {"error": "invalid decision"})
                    return
                item = next((x for x in OperatorIPC.list_requests() if x["id"] == match.group(1)), None)
                if item is None:
                    self.reply(409, {"error": "request expired or already decided"})
                    return
                result = OperatorIPC.call(decision, item["id"])
                expected = "approved" if decision == "approve" else "denied"
                if not isinstance(result, dict) or result.get("status") != expected:
                    self.reply(409, {"error": "broker refused decision"})
                    return
                self.reply(200, {"status": expected})
            except (RuntimeError, OSError, ValueError, json.JSONDecodeError):
                self.reply(503, {"error": "broker refused or unavailable"})
            return
        self.reply(404, {"error": "not found"})


def main():
    if not os.environ.get("PORTICO_OPERATOR_PASSWORD_SCRYPT") or not os.environ.get("PORTICO_OPERATOR_PUBLIC_ORIGIN") or len(os.environ.get("PORTICO_OPERATOR_APPROVAL_TOKEN", "")) < 32 or not os.environ.get("PORTICO_OPERATOR_PHYSICAL_CEILING", "").startswith("/"):
        raise SystemExit("Operator password verifier and public origin must be configured")
    # Binding externally is deliberately unsupported. Access only through a
    # separately authenticated, rate-limited TLS reverse proxy.
    address = ("0.0.0.0" if os.environ.get("PORTICO_OPERATOR_CONTAINER") == "1" else "127.0.0.1", 8765)
    print("Portico operator service started behind loopback-published proxy", flush=True)
    ThreadingHTTPServer(address, Handler).serve_forever()


if __name__ == "__main__":
    main()
