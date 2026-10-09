#!/usr/bin/env python3
"""Operator-origin authentication and one-shot decisions over restricted Unix IPC.

MCP Apps never carry credentials. An optional cross-origin iframe reuses this
service with an independently authenticated, partitioned operator session.
"""
from __future__ import annotations
import datetime as dt
import hashlib
import hmac
import http.cookies
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import importlib.util
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
SPEC = importlib.util.spec_from_file_location("operator_approvals", REPO / "scripts/operator-approvals.py")
assert SPEC and SPEC.loader
operator = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(operator)
ID_RE = re.compile(r"apr_[A-Za-z0-9_-]{8,100}\Z")
API_RE = re.compile(r"/operator(?P<embed>/embed)?/api/approvals/(?P<id>apr_[A-Za-z0-9_-]{8,100})(?P<action>/decision|/step-up)?\Z")
UI_PATH = HERE / "index.html"
COOKIE = "portico_operator"
EMBED_COOKIE = "__Secure-portico_operator_embed"
MAX_AGE = 900
LOCK = threading.Lock()
AUTH_SLOTS = threading.BoundedSemaphore(2)
SESSIONS: dict[str, tuple[float, str, str]] = {}
LOGIN_FAILURES: dict[str, list[float]] = {}
NONCES: dict[str, dict] = {}
STEPUPS: dict[str, tuple[float, str]] = {}


def identity():
    return operator.safe_text(os.environ.get("PORTICO_OPERATOR_ID", "local-owner"))


def binding():
    # Rotation, changed node/origin, and operator identity swaps invalidate every
    # existing browser session. Neither cookie nor MCP subject can change this.
    value = [identity(), *[os.environ.get(k, "") for k in (
        "PORTICO_OPERATOR_PASSWORD_SCRYPT", "PORTICO_OPERATOR_APPROVAL_TOKEN",
        "PORTICO_OPERATOR_PUBLIC_ORIGIN", "PORTICO_OPERATOR_NODE_ID")]]
    return hashlib.sha256(json.dumps(value).encode()).hexdigest()


def frame_ancestors():
    values = os.environ.get("PORTICO_OPERATOR_FRAME_ANCESTORS", "").split()
    for value in values:
        u = urlsplit(value)
        if (u.scheme != "https" or not u.hostname or u.username or u.password
                or u.path or u.query or u.fragment or "*" in value
                or any(c.isspace() or c in "'\";\\" for c in value)):
            raise ValueError("frame ancestor must be an exact HTTPS origin")
    return values


class OperatorIPC:
    @staticmethod
    def call(action, request_id=None, *, snapshot_hash=None, node_id=None, step_up=False):
        path = os.environ.get("PORTICO_OPERATOR_SOCKET", "/run/portico-operator/operator.sock")
        token = os.environ.get("PORTICO_OPERATOR_APPROVAL_TOKEN", "")
        if len(token) < 32 or not path.startswith("/"):
            raise RuntimeError("operator IPC unconfigured")
        tools = {"approvals": "admin.approval.list", "approve": "admin.approval.approve",
                 "deny": "admin.approval.deny", "status": "admin.approval.status"}
        if action not in tools:
            raise RuntimeError("unsupported operator operation")
        payload = {"id": secrets.token_urlsafe(12), "tool": tools[action],
                   "admin_token": token, "subject": identity()}
        if request_id is not None:
            if not ID_RE.fullmatch(request_id):
                raise RuntimeError("invalid request id")
            payload["args"] = {"request_id": request_id}
            if action in ("approve", "deny"):
                if not isinstance(snapshot_hash, str) or not re.fullmatch(r"[a-f0-9]{64}", snapshot_hash):
                    raise RuntimeError("snapshot binding required")
                payload["args"].update(snapshot_hash=snapshot_hash, node_id=node_id, step_up=step_up)
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as client:
            client.settimeout(5)
            client.connect(path)
            client.sendall(json.dumps(payload).encode() + b"\n")
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
        result = []
        for row in rows:
            item = operator.request_detail(row)
            if item:
                fp, node = row.get("Fingerprint"), row.get("NodeID", "")
                if not isinstance(fp, str) or not re.fullmatch(r"[a-f0-9]{64}", fp):
                    raise RuntimeError("Broker missing immutable request binding")
                if node != os.environ.get("PORTICO_OPERATOR_NODE_ID", ""):
                    raise RuntimeError("destination node mismatch")
                item.update(fingerprint=fp, node_id=node)
                result.append(item)
        return result


def verify_password(password):
    encoded = os.environ.get("PORTICO_OPERATOR_PASSWORD_SCRYPT", "")
    try:
        salt_hex, digest_hex = encoded.split(":", 1)
        salt, expected = bytes.fromhex(salt_hex), bytes.fromhex(digest_hex)
        if len(salt) < 16 or len(expected) != 32:
            return False
        with AUTH_SLOTS:
            actual = hashlib.scrypt(password.encode(), salt=salt, n=2**14, r=8, p=1, dklen=32)
        return hmac.compare_digest(actual, expected)
    except (ValueError, TypeError, AttributeError):
        return False


def session_from_cookie(raw, embedded=False):
    jar = http.cookies.SimpleCookie()
    try:
        jar.load(raw)
    except http.cookies.CookieError:
        return None
    key = EMBED_COOKIE if embedded else COOKIE
    if key not in jar:
        return None
    sid = jar[key].value
    with LOCK:
        state = SESSIONS.get(sid)
        if state and len(state) == 3 and state[0] > time.monotonic() and state[2] == binding():
            return sid, state[1]
    return None


def prune():
    now = time.monotonic()
    for sid, state in list(SESSIONS.items()):
        if state[0] <= now:
            SESSIONS.pop(sid, None)
    for nonce, value in list(NONCES.items()):
        if value["expires"] <= now:
            NONCES.pop(nonce, None)
            STEPUPS.pop(nonce, None)


def create_nonce(session, item):
    nonce = secrets.token_urlsafe(32)
    with LOCK:
        prune()
        if len(NONCES) >= 4096:
            raise RuntimeError("decision capacity exceeded")
        NONCES[nonce] = {"sid": session[0], "request_id": item["id"],
                        "fingerprint": item["fingerprint"], "node_id": item["node_id"],
                        "step_up": item["kind"] != "root" or item["access"] != "read",
                        "expires": time.monotonic() + min(60, max(0, (item["expires"] - dt.datetime.now(dt.timezone.utc)).total_seconds()))}
    return nonce


def pending_details(item):
    return {"request_id": item["id"], "node": os.environ.get("PORTICO_OPERATOR_NODE_LABEL", "Local"),
            "node_id": item["node_id"], "operator": identity(), "subject": item["subject"],
            "resource": item["target"], "access": item["access"],
            "ttl_label": f'{item["ttl_ns"] // 1_000_000_000} segundos',
            "expires_at": item["expires"].isoformat(), "ceiling_wide": False,
            "status": "pending", "step_up_required": item["kind"] != "root" or item["access"] != "read"}


class Handler(BaseHTTPRequestHandler):
    server_version = "PorticoOperator/0.2"

    def log_message(self, *_):
        # Avoid logging credentials, query IDs, cookies and decision nonces.
        pass

    def headers_base(self, typ="application/json; charset=utf-8", embedded=False):
        self.send_header("Content-Type", typ)
        self.send_header("Cache-Control", "no-store")
        self.send_header("X-Content-Type-Options", "nosniff")
        self.send_header("Referrer-Policy", "no-referrer")
        ancestors = " ".join(frame_ancestors()) if embedded else "'none'"
        if not embedded:
            self.send_header("X-Frame-Options", "DENY")
        self.send_header("Content-Security-Policy", "default-src 'none'; connect-src 'self'; script-src 'self'; style-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors " + (ancestors or "'none'"))

    def reply(self, status, obj, extra_headers=None):
        payload = json.dumps(obj, ensure_ascii=False).encode()
        self.send_response(status)
        self.headers_base()
        for key, value in (extra_headers or {}).items():
            self.send_header(key, value)
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def host_ok(self):
        try:
            u = urlsplit(os.environ.get("PORTICO_OPERATOR_PUBLIC_ORIGIN", ""))
            return bool(u.scheme == "https" and u.netloc and self.headers.get("Host", "").lower() == u.netloc.lower())
        except ValueError:
            return False

    def origin_ok(self):
        origin = self.headers.get("Origin")
        return bool(origin and origin == os.environ.get("PORTICO_OPERATOR_PUBLIC_ORIGIN", ""))

    def body(self):
        if self.headers.get("Content-Type", "").split(";", 1)[0].strip() != "application/json":
            raise ValueError("JSON content type required")
        length = self.headers.get("Content-Length", "")
        if not length.isdecimal() or int(length) > 2048:
            raise ValueError("invalid body length")
        def unique(pairs):
            value = {}
            for key, val in pairs:
                if key in value:
                    raise ValueError("duplicate JSON field")
                value[key] = val
            return value
        obj = json.loads(self.rfile.read(int(length)), object_pairs_hook=unique)
        if not isinstance(obj, dict):
            raise ValueError("body must be object")
        return obj

    def session(self, embedded=False):
        return session_from_cookie(self.headers.get("Cookie", ""), embedded)

    def auth_attempt(self, supplied):
        peer, now = self.client_address[0], time.monotonic()
        with LOCK:
            failures = [t for t in LOGIN_FAILURES.get(peer, []) if now - t < 300]
            if len(failures) >= 5:
                return 429
            # Reserve before scrypt, including concurrent requests.
            LOGIN_FAILURES[peer] = [*failures, now]
        if not isinstance(supplied, str) or len(supplied) > 1024 or not verify_password(supplied):
            return 403
        with LOCK:
            LOGIN_FAILURES[peer] = [t for t in LOGIN_FAILURES.get(peer, []) if t != now]
        return 200

    def do_GET(self):
        if not self.host_ok():
            self.reply(421, {"error": "host denied"}); return
        path = urlsplit(self.path).path
        if path in ("/operator/app.js", "/operator/style.css"):
            filename = "app.js" if path.endswith(".js") else "style.css"
            asset = (HERE / filename).read_bytes()
            self.send_response(200)
            self.headers_base("text/javascript; charset=utf-8" if filename.endswith(".js") else "text/css; charset=utf-8")
            self.send_header("Content-Length", str(len(asset))); self.end_headers(); self.wfile.write(asset); return
        if path in ("/", "/operator", "/operator/", "/operator/embed"):
            embedded = path == "/operator/embed"
            if embedded and not frame_ancestors():
                self.reply(404, {"error": "embedded channel disabled"}); return
            page = UI_PATH.read_bytes()
            self.send_response(200); self.headers_base("text/html; charset=utf-8", embedded)
            self.send_header("Content-Length", str(len(page))); self.end_headers(); self.wfile.write(page); return
        match = API_RE.fullmatch(path)
        if not match or match["action"]:
            self.reply(404, {"error": "not found"}); return
        embedded = bool(match["embed"])
        if embedded and not frame_ancestors():
            self.reply(404, {"error": "embedded channel disabled"}); return
        session = self.session(embedded)
        if session is None:
            self.reply(401, {"error": "login required"}); return
        try:
            item = next((x for x in OperatorIPC.list_requests() if x["id"] == match["id"]), None)
            if item:
                data = pending_details(item)
                data.update(csrf_token=session[1], decision_nonce=create_nonce(session, item))
            else:
                data = OperatorIPC.call("status", match["id"])
                if not isinstance(data, dict) or data.get("status") not in ("approved", "denied", "expired", "cancelled", "revoked"):
                    self.reply(404, {"error": "request unavailable"}); return
                data.update(node=os.environ.get("PORTICO_OPERATOR_NODE_LABEL", "Local"), operator=identity(),
                            ttl_label=f'{data.get("delegation_ttl_seconds", 0)} segundos', expires_at=data.get("approval_expires_at"), ceiling_wide=False)
            self.reply(200, data)
        except (RuntimeError, OSError, ValueError):
            self.reply(503, {"error": "broker unavailable"})

    def do_POST(self):
        if not self.host_ok():
            self.reply(421, {"error": "host denied"}); return
        if not self.origin_ok():
            self.reply(403, {"error": "origin denied"}); return
        path = urlsplit(self.path).path
        embedded = path.startswith("/operator/embed/")
        if embedded and not frame_ancestors():
            self.reply(404, {"error": "embedded channel disabled"}); return
        prefix = "/operator/embed/api" if embedded else "/operator/api"
        if path == prefix + "/login":
            try:
                code = self.auth_attempt(self.body().get("password"))
                if code != 200:
                    self.reply(code, {"error": "invalid credentials or rate limit"}); return
                old = self.session(embedded)
                sid, csrf = secrets.token_urlsafe(32), secrets.token_urlsafe(32)
                with LOCK:
                    prune()
                    if len(SESSIONS) >= 500:
                        self.reply(429, {"error": "session capacity exceeded"}); return
                    if old: SESSIONS.pop(old[0], None)
                    SESSIONS[sid] = (time.monotonic() + MAX_AGE, csrf, binding())
                key = EMBED_COOKIE if embedded else COOKIE
                cookie_path = "/operator/embed" if embedded else "/operator"
                cookie = f"{key}={sid}; HttpOnly; Secure; Path={cookie_path}; Max-Age={MAX_AGE}; " + ("SameSite=None; Partitioned" if embedded else "SameSite=Strict")
                self.reply(200, {"ok": True, "operator": identity()}, {"Set-Cookie": cookie})
            except (ValueError, json.JSONDecodeError):
                self.reply(400, {"error": "invalid payload"})
            return
        match = API_RE.fullmatch(path)
        if path != prefix + "/logout" and (not match or not match["action"]):
            self.reply(404, {"error": "not found"}); return
        session = self.session(embedded)
        if not session:
            self.reply(401, {"error": "login required"}); return
        if not hmac.compare_digest(self.headers.get("X-CSRF-Token", ""), session[1]):
            self.reply(403, {"error": "csrf denied"}); return
        if path == prefix + "/logout":
            with LOCK:
                SESSIONS.pop(session[0], None)
            self.reply(200, {"status": "signed_out"}); return
        try:
            request = self.body()
            nonce = request.get("decision_nonce")
            with LOCK:
                prune()
                record = NONCES.get(nonce) if isinstance(nonce, str) else None
                valid = record and record["sid"] == session[0] and record["request_id"] == match["id"]
            if not valid:
                self.reply(409, {"error": "decision expired, already used or session mismatch"}); return
            if match["action"] == "/step-up":
                code = self.auth_attempt(request.get("password"))
                if code != 200:
                    self.reply(code, {"error": "verification failed or rate limit"}); return
                with LOCK:
                    STEPUPS[nonce] = (time.monotonic() + 60, session[0])
                self.reply(200, {"ok": True}); return
            decision = request.get("decision")
            if decision not in ("approve", "deny"):
                self.reply(400, {"error": "invalid decision"}); return
            with LOCK:
                proof = STEPUPS.get(nonce)
                stepped = bool(proof and proof[0] > time.monotonic() and proof[1] == session[0])
                if decision == "approve" and record["step_up"] and not stepped:
                    self.reply(428, {"error": "step_up_required"}); return
                if NONCES.get(nonce) is not record or not self.session_valid(session):
                    self.reply(409, {"error": "decision already used or session expired"}); return
                NONCES.pop(nonce, None); STEPUPS.pop(nonce, None)
            result = OperatorIPC.call(decision, record["request_id"], snapshot_hash=record["fingerprint"], node_id=record["node_id"], step_up=stepped)
            expected = "approved" if decision == "approve" else "denied"
            if not isinstance(result, dict) or result.get("status") != expected:
                self.reply(409, {"error": "broker refused decision"}); return
            self.reply(200, {"request_id": record["request_id"], "status": expected})
        except (RuntimeError, OSError, ValueError, json.JSONDecodeError):
            self.reply(503, {"error": "broker refused or unavailable; refresh before retrying"})

    @staticmethod
    def session_valid(session):
        state = SESSIONS.get(session[0])
        return bool(state and state[0] > time.monotonic() and state[2] == binding())


def main():
    origin = urlsplit(os.environ.get("PORTICO_OPERATOR_PUBLIC_ORIGIN", ""))
    if (not os.environ.get("PORTICO_OPERATOR_PASSWORD_SCRYPT") or origin.scheme != "https"
            or not origin.hostname or origin.path or origin.query or origin.fragment or origin.username
            or len(os.environ.get("PORTICO_OPERATOR_APPROVAL_TOKEN", "")) < 32
            or not os.environ.get("PORTICO_OPERATOR_PHYSICAL_CEILING", "").startswith("/")):
        raise SystemExit("Operator verifier, HTTPS origin and scoped IPC credentials must be configured")
    identity(); frame_ancestors()
    address = ("0.0.0.0" if os.environ.get("PORTICO_OPERATOR_CONTAINER") == "1" else "127.0.0.1", 8765)
    print("Portico operator service ready behind loopback-published TLS proxy", flush=True)
    ThreadingHTTPServer(address, Handler).serve_forever()


if __name__ == "__main__":
    main()
