#!/usr/bin/env python3
"""Read-only MCP metadata probe for a staging or loopback Portico endpoint.

This is a protocol client, not a ChatGPT/Claude/VS Code compatibility claim.
It never calls tools, creates an approval request, submits elicitation responses,
executes returned HTML, follows redirects, or visits the operator portal.
"""
from __future__ import annotations

import argparse
import base64
import copy
import datetime as dt
import hashlib
import ipaddress
import json
import os
import re
import socket
import time
from urllib import error, parse, request

APP_URI = "ui://portico/authorizations-v1.html"
APP_MIME = "text/html;profile=mcp-app"
UI_EXTENSION = "io.modelcontextprotocol/ui"
MODERN = "2026-07-28"
LEGACY = "2025-11-25"
META_VERSION = "io.modelcontextprotocol/protocolVersion"
META_CLIENT = "io.modelcontextprotocol/clientInfo"
META_CAPS = "io.modelcontextprotocol/clientCapabilities"
MAX_RESPONSE = 1024 * 1024
MAX_PAGES = 20
CLIENT_INFO = {"name": "portico-metadata-probe", "version": "1"}
TOOL_NAMES = (
    "permissions.request_root_access",
    "permissions.request_sensitive_access",
    "permissions.approval_status",
    "permissions.cancel_approval",
)
ALLOWED_METHODS = frozenset((
    "server/discover", "initialize", "notifications/initialized",
    "tools/list", "resources/read",
))
PROFILES = {
    "text": {},
    "apps": {"extensions": {UI_EXTENSION: {"mimeTypes": [APP_MIME]}}},
    "form": {"elicitation": {"form": {}}},
    "legacy-form": {"elicitation": {}},
    "url": {"elicitation": {"url": {}}},
    "apps-and-elicitation": {
        "extensions": {UI_EXTENSION: {"mimeTypes": [APP_MIME]}},
        "elicitation": {"form": {}, "url": {}},
    },
    "wrong-mime": {"extensions": {UI_EXTENSION: {"mimeTypes": ["text/html"]}}},
}


class ProbeError(Exception):
    """Only fixed labels and numeric codes may escape to the report."""

    def __init__(self, category: str, code: int | None = None):
        super().__init__(category)
        self.category, self.code = category, code


class NoRedirect(request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        # Prevent disclosure of Authorization or MCP session ID to another URL.
        return None


def endpoint_url(value: str, staging: bool = False) -> str:
    try:
        u = parse.urlsplit(value)
        port = u.port
    except ValueError as exc:
        raise ValueError("Invalid MCP endpoint") from exc
    if (u.scheme not in ("http", "https") or not u.hostname or u.username is not None
            or u.password is not None or u.query or u.fragment
            or any(ord(c) <= 32 or ord(c) == 127 for c in value)):
        raise ValueError("Use a clean HTTPS MCP URL, without credentials, query or fragment")
    try:
        loopback = ipaddress.ip_address(u.hostname).is_loopback
    except ValueError:
        loopback = u.hostname == "localhost"
    if not loopback and (u.scheme != "https" or not staging):
        raise ValueError("Remote endpoints require HTTPS and --staging; use a separate staging service")
    if port is not None and not 1 <= port <= 65535:
        raise ValueError("Invalid MCP endpoint port")
    return value


def validate_token(token: str | None) -> str | None:
    if token is not None and (not token or len(token) > 8192
                              or any(ord(c) <= 32 or ord(c) >= 127 for c in token)):
        raise ValueError("Invalid bearer token; provide a nonempty token through the environment")
    return token


def decode_rpc(body: bytes, rpc_id: int) -> dict:
    try:
        value = json.loads(body)
    except (ValueError, UnicodeError) as exc:
        raise ProbeError("invalid_json") from exc
    if (not isinstance(value, dict) or value.get("jsonrpc") != "2.0"
            or type(value.get("id")) is not int or value.get("id") != rpc_id
            or "method" in value or ("error" in value and "result" in value)):
        raise ProbeError("invalid_rpc_response")
    if "error" in value:
        err = value["error"]
        code = err.get("code") if isinstance(err, dict) else None
        if not isinstance(code, int) or isinstance(code, bool):
            code = None
        # Never include arbitrary error text/data: servers may echo credentials.
        raise ProbeError("rpc_error", code)
    if not isinstance(value.get("result"), dict):
        raise ProbeError("invalid_rpc_result")
    return value["result"]


def read_rpc(response, rpc_id: int) -> tuple[dict, str]:
    mime = response.headers.get("Content-Type", "").split(";", 1)[0].strip().lower()
    if mime == "application/json":
        raw = response.read(MAX_RESPONSE + 1)
        if len(raw) > MAX_RESPONSE:
            raise ProbeError("response_too_large")
        return decode_rpc(raw, rpc_id), "json"
    if mime != "text/event-stream":
        raise ProbeError("unsupported_response_type")
    data, total = [], 0
    while True:
        line = response.readline(MAX_RESPONSE + 1)
        total += len(line)
        if total > MAX_RESPONSE:
            raise ProbeError("response_too_large")
        if not line:
            raise ProbeError("sse_response_missing")
        line = line.rstrip(b"\r\n")
        if line.startswith(b"data:"):
            data.append(line[5:].lstrip(b" "))
        if line or not data:
            continue
        raw, data = b"\n".join(data), []
        try:
            value = json.loads(raw)
        except (ValueError, UnicodeError) as exc:
            raise ProbeError("invalid_json") from exc
        if not isinstance(value, dict):
            raise ProbeError("invalid_rpc_response")
        # Ignore notifications. Never answer server requests automatically.
        if "method" in value and "id" in value:
            raise ProbeError("unexpected_server_request")
        if value.get("id") == rpc_id:
            return decode_rpc(raw, rpc_id), "sse"


class ProbeClient:
    def __init__(self, url: str, caps: dict, protocol: str, token: str | None = None,
                 timeout: float = 10, opener=None):
        self.url, self.caps, self.protocol = url, copy.deepcopy(caps), protocol
        self.token, self.timeout = validate_token(token), timeout
        self.opener = opener or request.build_opener(NoRedirect())
        self.session_id = None
        self.next_id = 0
        self.transports = set()

    def meta(self) -> dict:
        return {META_VERSION: self.protocol, META_CLIENT: CLIENT_INFO, META_CAPS: self.caps}

    def rpc(self, method: str, params: dict, *, notification: bool = False) -> dict:
        if method not in ALLOWED_METHODS:
            raise ProbeError("method_forbidden")
        if method == "resources/read" and params.get("uri") != APP_URI:
            raise ProbeError("resource_forbidden")
        params = copy.deepcopy(params)
        if self.protocol == MODERN:
            params["_meta"] = self.meta()
        self.next_id += 1
        payload = {"jsonrpc": "2.0", "method": method, "params": params}
        if not notification:
            payload["id"] = self.next_id
        headers = {"Content-Type": "application/json", "Accept": "application/json, text/event-stream",
                   "MCP-Protocol-Version": self.protocol}
        if self.token:
            headers["Authorization"] = "Bearer " + self.token
        if self.session_id:
            headers["Mcp-Session-Id"] = self.session_id
        req = request.Request(self.url, json.dumps(payload).encode(), headers, method="POST")
        try:
            with self.opener.open(req, timeout=self.timeout) as response:
                sid = response.headers.get("Mcp-Session-Id")
                if sid is not None:
                    if (len(sid) > 512 or not sid or any(ord(c) <= 32 or ord(c) >= 127 for c in sid)
                            or self.session_id is not None and sid != self.session_id):
                        raise ProbeError("invalid_session_id")
                    self.session_id = sid
                if notification:
                    if response.status not in (200, 202, 204):
                        raise ProbeError("notification_rejected", response.status)
                    return {}
                result, transport = read_rpc(response, self.next_id)
                self.transports.add(transport)
                return result
        except error.HTTPError as exc:
            raise ProbeError("http_error", exc.code) from None
        except (error.URLError, TimeoutError, socket.timeout, OSError):
            raise ProbeError("transport_error") from None

    def discover(self) -> dict:
        if self.protocol == MODERN:
            result = self.rpc("server/discover", {})
            versions = result.get("supportedVersions")
            if not isinstance(versions, list) or self.protocol not in versions:
                raise ProbeError("protocol_not_supported")
            return result
        result = self.rpc("initialize", {"protocolVersion": self.protocol,
                          "clientInfo": CLIENT_INFO, "capabilities": self.caps})
        if result.get("protocolVersion") != self.protocol:
            raise ProbeError("protocol_mismatch")
        self.rpc("notifications/initialized", {}, notification=True)
        return result

    def tools(self) -> list[dict]:
        tools, cursors, cursor = [], set(), None
        for _ in range(MAX_PAGES):
            page = self.rpc("tools/list", {} if cursor is None else {"cursor": cursor})
            batch = page.get("tools")
            if not isinstance(batch, list) or not all(isinstance(t, dict) for t in batch):
                raise ProbeError("invalid_tools_page")
            tools.extend(batch)
            if len(tools) > 4096:
                raise ProbeError("too_many_tools")
            cursor = page.get("nextCursor")
            if cursor is None or cursor == "":
                return tools
            if not isinstance(cursor, str) or len(cursor) > 8192 or cursor in cursors:
                raise ProbeError("invalid_tools_cursor")
            cursors.add(cursor)
        raise ProbeError("too_many_tools_pages")


def resource_summary(contents: list) -> dict:
    if not isinstance(contents, list) or len(contents) != 1 or not isinstance(contents[0], dict):
        raise ProbeError("invalid_app_resource")
    item = contents[0]
    if item.get("uri") != APP_URI or item.get("mimeType") != APP_MIME:
        raise ProbeError("invalid_app_resource")
    text = item.get("text")
    if isinstance(text, str):
        try:
            raw = text.encode()
        except UnicodeError as exc:
            raise ProbeError("invalid_app_resource") from exc
    elif isinstance(item.get("blob"), str):
        try:
            raw = base64.b64decode(item["blob"], validate=True)
        except (ValueError, base64.binascii.Error) as exc:
            raise ProbeError("invalid_app_resource") from exc
    else:
        raise ProbeError("invalid_app_resource")
    if len(raw) > MAX_RESPONSE:
        raise ProbeError("response_too_large")
    meta = item.get("_meta")
    ui = meta.get("ui") if isinstance(meta, dict) else None
    csp = ui.get("csp") if isinstance(ui, dict) else None
    frames = csp.get("frameDomains") if isinstance(csp, dict) else None
    return {"uri": APP_URI, "mime_type": APP_MIME, "bytes": len(raw),
            "sha256": hashlib.sha256(raw).hexdigest(),
            "configuration_placeholder_present": b"__PORTICO_CONFIG__" in raw,
            "frame_domains_declared": isinstance(frames, list) and bool(frames),
            "browser_render_verified": False}


def probe_profile(url: str, profile: str, protocol: str, token: str | None = None,
                  timeout: float = 10, opener=None) -> dict:
    started = time.monotonic()
    report = {"profile": profile, "protocol": protocol, "ok": False,
              "evidence": "simulated_client_metadata_only", "tool_calls": 0,
              "browser_render_verified": False, "operator_decision_verified": False}
    client = ProbeClient(url, PROFILES[profile], protocol, token, timeout, opener)
    try:
        discovered = client.discover()
        caps = discovered.get("capabilities")
        if not isinstance(caps, dict):
            raise ProbeError("invalid_server_capabilities")
        ext = caps.get("extensions")
        ui = ext.get(UI_EXTENSION) if isinstance(ext, dict) else None
        mimes = ui.get("mimeTypes") if isinstance(ui, dict) else None
        report["server_advertises_apps_mime"] = isinstance(mimes, list) and APP_MIME in mimes
        tools = client.tools()
        found = {t.get("name"): t for t in tools if t.get("name") in TOOL_NAMES}
        report["approval_tools"] = {name: name in found for name in TOOL_NAMES}
        # Hidden/app-only tools are not an authentication boundary; a published
        # legacy confirmation tool is nevertheless a review finding.
        report["confirmation_tools_exposed"] = any(
            t.get("name") in ("permissions.confirm_root_access", "permissions.confirm_sensitive_access",
                              "admin.approval.approve", "admin.approval.deny") for t in tools)
        app_tools = []
        unexpected_uri = False
        for name, tool in found.items():
            meta = tool.get("_meta")
            ui = meta.get("ui") if isinstance(meta, dict) else None
            uri = ui.get("resourceUri") if isinstance(ui, dict) else None
            if uri == APP_URI:
                app_tools.append(name)
            elif uri is not None:
                unexpected_uri = True
        report["app_tool_bindings"] = sorted(app_tools)
        report["unexpected_app_uri"] = unexpected_uri
        if app_tools:
            result = client.rpc("resources/read", {"uri": APP_URI})
            report["app_resource"] = resource_summary(result.get("contents"))
        report["ok"] = not report["confirmation_tools_exposed"] and not unexpected_uri
    except ProbeError as exc:
        report["error"] = {"category": exc.category}
        if exc.code is not None:
            report["error"]["code"] = exc.code
    report["response_transports"] = sorted(client.transports)
    report["elapsed_ms"] = round((time.monotonic() - started) * 1000)
    return report


def main(argv=None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--url", required=True, help="HTTPS staging /mcp endpoint or loopback URL")
    parser.add_argument("--staging", action="store_true", help="assert remote URL is a separate staging service")
    parser.add_argument("--token-env", help="environment variable containing a staging bearer token (never printed)")
    parser.add_argument("--profile", choices=("all", *PROFILES), default="all")
    parser.add_argument("--protocol", choices=("both", MODERN, LEGACY), default="both")
    parser.add_argument("--timeout", type=float, default=10, help="per-request timeout in seconds, 1..30")
    args = parser.parse_args(argv)
    try:
        url = endpoint_url(args.url, args.staging)
        if not 1 <= args.timeout <= 30:
            raise ValueError("Timeout must be between 1 and 30 seconds")
        if args.token_env and not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_]*", args.token_env):
            raise ValueError("Invalid token environment variable name")
        token = os.environ.get(args.token_env) if args.token_env else None
        if args.token_env and token is None:
            raise ValueError("Token environment variable is absent")
        validate_token(token)
    except ValueError as exc:
        # Validation messages are fixed and do not interpolate submitted secrets.
        parser.error(str(exc))
    profiles = PROFILES if args.profile == "all" else (args.profile,)
    protocols = (MODERN, LEGACY) if args.protocol == "both" else (args.protocol,)
    reports = [probe_profile(url, profile, protocol, token, args.timeout)
               for protocol in protocols for profile in profiles]
    print(json.dumps({"probe_version": 1, "observed_at_utc": dt.datetime.now(dt.timezone.utc).isoformat(),
                      "scope": "metadata_only_no_tool_calls_no_decisions",
                      "runs": reports}, indent=2))
    return 0 if all(r["ok"] for r in reports) else 1


if __name__ == "__main__":
    raise SystemExit(main())
