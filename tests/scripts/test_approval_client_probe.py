"""Adversarial transport/metadata tests; no live service or authorization."""
import contextlib
import importlib.util
import io
import json
import os
from pathlib import Path
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location("approval_probe", ROOT / "scripts/approval-client-probe.py")
PROBE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(PROBE)


class FixtureHandler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        self.server.calls.append((body, dict(self.headers)))
        method = body["method"]
        if self.server.redirect:
            self.send_response(307)
            self.send_header("Location", self.server.redirect)
            self.send_header("Content-Length", "0")
            self.end_headers()
            return
        if self.server.fail_code:
            self.send_response(self.server.fail_code)
            self.end_headers()
            self.wfile.write(b"private token secret echoed by untrusted server")
            return
        if method == "notifications/initialized":
            self.send_response(202)
            self.end_headers()
            return
        if method == "server/discover":
            result = {"supportedVersions": [PROBE.MODERN, PROBE.LEGACY],
                      "capabilities": {"extensions": {PROBE.UI_EXTENSION: {"mimeTypes": [PROBE.APP_MIME]}}}}
        elif method == "initialize":
            result = {"protocolVersion": PROBE.LEGACY, "capabilities": {"tools": {}, "resources": {}}}
        elif method == "tools/list":
            if self.server.pagination and not body["params"].get("cursor"):
                result = {"tools": [], "nextCursor": "opaque-not-printed"}
            else:
                tools = [{"name": name} for name in PROBE.TOOL_NAMES]
                if self.server.expose_confirm:
                    tools.append({"name": "permissions.confirm_root_access", "description": "SECRET"})
                if self.server.apps:
                    tools[0]["_meta"] = {"ui": {"resourceUri": self.server.resource_uri}}
                result = {"tools": tools}
                if self.server.cycle_cursor:
                    result["nextCursor"] = "opaque-not-printed"
        elif method == "resources/read":
            result = {"contents": [{"uri": PROBE.APP_URI, "mimeType": self.server.mime,
                       "text": "<!DOCTYPE html><html>credential=NEVER_EXECUTED</html>",
                       "_meta": {"ui": {"csp": {"frameDomains": ["https://operator.test"]}}}}]}
        else:
            raise AssertionError("Probe attempted an unauthorized method")
        value = {"jsonrpc": "2.0", "id": body["id"], "result": result}
        if self.server.rpc_error:
            value = {"jsonrpc": "2.0", "id": body["id"], "error": {
                "code": -32601, "message": "SECRET token=" + self.headers.get("Authorization", ""),
                "data": {"password": "secret"}}}
        if self.server.wrong_id:
            value["id"] += 1
        encoded = json.dumps(value).encode()
        self.send_response(200)
        if method == "initialize":
            self.send_header("Mcp-Session-Id", "fixture-session-never-printed")
        if self.server.sse:
            encoded = b": keepalive\n\ndata: " + encoded + b"\n\n"
            self.send_header("Content-Type", "text/event-stream")
        else:
            self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)


@contextlib.contextmanager
def fixture():
    server = ThreadingHTTPServer(("127.0.0.1", 0), FixtureHandler)
    server.calls = []
    server.redirect = ""
    server.fail_code = None
    server.rpc_error = False
    server.wrong_id = False
    server.apps = True
    server.expose_confirm = False
    server.pagination = False
    server.cycle_cursor = False
    server.sse = False
    server.resource_uri = PROBE.APP_URI
    server.mime = PROBE.APP_MIME
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        yield server, "http://127.0.0.1:%d/mcp" % server.server_port
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=2)


class ProbeTests(unittest.TestCase):
    def test_reject_unsafe_endpoint_and_require_remote_staging(self):
        for url in ("http://remote.test/mcp", "https://owner:password@remote.test/mcp",
                    "https://remote.test/mcp?token=secret", "https://remote.test/mcp#secret",
                    "https://remote.test/mcp\n", "file:///etc/passwd", "https://remote.test:0/mcp"):
            with self.subTest(url=url), self.assertRaises(ValueError):
                PROBE.endpoint_url(url, staging=True)
        with self.assertRaises(ValueError):
            PROBE.endpoint_url("https://remote.test/mcp")
        self.assertEqual(PROBE.endpoint_url("https://remote.test/mcp", True), "https://remote.test/mcp")
        self.assertEqual(PROBE.endpoint_url("http://[::1]:1234/mcp"), "http://[::1]:1234/mcp")

    def test_capabilities_modern_are_sent_on_every_request(self):
        with fixture() as (server, url):
            result = PROBE.probe_profile(url, "apps-and-elicitation", PROBE.MODERN)
            self.assertTrue(result["ok"], result)
            self.assertEqual(result["tool_calls"], 0)
            self.assertFalse(result["browser_render_verified"])
            self.assertFalse(result["operator_decision_verified"])
            self.assertTrue(result["server_advertises_apps_mime"])
            self.assertTrue(result["app_resource"]["frame_domains_declared"])
            self.assertEqual([b["method"] for b, _ in server.calls],
                             ["server/discover", "tools/list", "resources/read"])
            for body, headers in server.calls:
                self.assertEqual(body["params"]["_meta"][PROBE.META_CAPS],
                                 PROBE.PROFILES["apps-and-elicitation"])
                self.assertEqual(headers["Mcp-Protocol-Version"], PROBE.MODERN)

    def test_legacy_initialize_and_session_are_kept_private(self):
        with fixture() as (server, url):
            result = PROBE.probe_profile(url, "url", PROBE.LEGACY, "fixture-bearer-never-printed")
            self.assertTrue(result["ok"], result)
            self.assertEqual([b["method"] for b, _ in server.calls],
                             ["initialize", "notifications/initialized", "tools/list", "resources/read"])
            self.assertEqual(server.calls[0][0]["params"]["capabilities"], {"elicitation": {"url": {}}})
            for _, headers in server.calls[1:]:
                self.assertEqual(headers["Mcp-Session-Id"], "fixture-session-never-printed")
            report = json.dumps(result)
            self.assertNotIn("fixture-bearer", report)
            self.assertNotIn("fixture-session", report)
            self.assertNotIn("NEVER_EXECUTED", report)

    def test_sse_and_paginated_metadata(self):
        with fixture() as (server, url):
            server.sse = server.pagination = True
            result = PROBE.probe_profile(url, "apps", PROBE.MODERN)
            self.assertTrue(result["ok"], result)
            self.assertEqual(result["response_transports"], ["sse"])
            self.assertTrue(all(result["approval_tools"].values()))
            self.assertEqual([b["method"] for b, _ in server.calls].count("tools/list"), 2)

    def test_redirect_never_leaks_credentials_or_session(self):
        with fixture() as (target, target_url), fixture() as (server, url):
            server.redirect = target_url
            result = PROBE.probe_profile(url, "apps", PROBE.MODERN, "private-test-bearer")
            self.assertFalse(result["ok"])
            self.assertEqual(result["error"], {"category": "http_error", "code": 307})
            self.assertEqual(target.calls, [])
            self.assertNotIn("private-test-bearer", json.dumps(result))

    def test_rpc_error_never_prints_server_message(self):
        with fixture() as (server, url):
            server.rpc_error = True
            result = PROBE.probe_profile(url, "text", PROBE.MODERN, "private-test-bearer")
            self.assertEqual(result["error"], {"category": "rpc_error", "code": -32601})
            self.assertNotIn("SECRET", json.dumps(result))
            self.assertNotIn("private-test-bearer", json.dumps(result))

    def test_unauthorized_is_not_support(self):
        with fixture() as (server, url):
            server.fail_code = 401
            result = PROBE.probe_profile(url, "apps", PROBE.MODERN)
            self.assertFalse(result["ok"])
            self.assertEqual(result["error"], {"category": "http_error", "code": 401})

    def test_catalog_confirmation_tool_is_a_review_failure(self):
        with fixture() as (server, url):
            server.expose_confirm = True
            result = PROBE.probe_profile(url, "text", PROBE.MODERN)
            self.assertFalse(result["ok"])
            self.assertTrue(result["confirmation_tools_exposed"])
            self.assertNotIn("SECRET", json.dumps(result))

    def test_unexpected_ui_resource_is_not_fetched(self):
        with fixture() as (server, url):
            server.resource_uri = "file:///etc/private"
            result = PROBE.probe_profile(url, "apps", PROBE.MODERN)
            self.assertFalse(result["ok"])
            self.assertTrue(result["unexpected_app_uri"])
            self.assertNotIn("resources/read", [b["method"] for b, _ in server.calls])
            self.assertNotIn("file:///", json.dumps(result))

    def test_wrong_mime_is_not_app_support(self):
        with fixture() as (server, url):
            server.mime = "text/html"
            result = PROBE.probe_profile(url, "apps", PROBE.MODERN)
            self.assertFalse(result["ok"])
            self.assertEqual(result["error"], {"category": "invalid_app_resource"})

    def test_repeated_cursor_is_bounded(self):
        with fixture() as (server, url):
            server.pagination = server.cycle_cursor = True
            result = PROBE.probe_profile(url, "apps", PROBE.MODERN)
            self.assertFalse(result["ok"])
            self.assertEqual(result["error"], {"category": "invalid_tools_cursor"})

    def test_wrong_response_id_fails_closed(self):
        with fixture() as (server, url):
            server.wrong_id = True
            result = PROBE.probe_profile(url, "text", PROBE.MODERN)
            self.assertEqual(result["error"], {"category": "invalid_rpc_response"})

    def test_ambiguous_rpc_envelopes_are_rejected(self):
        for envelope in (
            {"jsonrpc": "2.0", "id": True, "result": {}},
            {"jsonrpc": "2.0", "id": 1, "result": {}, "error": {}},
            {"jsonrpc": "2.0", "id": 1, "result": {}, "method": "elicitation/create"},
        ):
            with self.subTest(envelope=envelope), self.assertRaises(PROBE.ProbeError) as error:
                PROBE.decode_rpc(json.dumps(envelope).encode(), 1)
            self.assertEqual(error.exception.category, "invalid_rpc_response")

    def test_invalid_unicode_resource_is_rejected_without_html_output(self):
        with self.assertRaises(PROBE.ProbeError) as error:
            PROBE.resource_summary([{"uri": PROBE.APP_URI, "mimeType": PROBE.APP_MIME,
                                     "text": "credential=private\ud800"}])
        self.assertEqual(error.exception.category, "invalid_app_resource")

    def test_methods_and_resources_are_hard_allowlisted(self):
        client = PROBE.ProbeClient("http://127.0.0.1/mcp", {}, PROBE.MODERN)
        for method in ("tools/call", "permissions.confirm_root_access", "elicitation/create"):
            with self.subTest(method=method), self.assertRaises(PROBE.ProbeError) as error:
                client.rpc(method, {"decision": "approve"})
            self.assertEqual(error.exception.category, "method_forbidden")
        with self.assertRaises(PROBE.ProbeError) as error:
            client.rpc("resources/read", {"uri": "file:///etc/private"})
        self.assertEqual(error.exception.category, "resource_forbidden")

    def test_token_header_injection_is_rejected(self):
        for token in ("", "bad\r\nX-Injected: yes", "a b", "secret\u202e"):
            with self.subTest(token=repr(token)), self.assertRaises(ValueError):
                PROBE.validate_token(token)

    def test_json_size_is_bounded(self):
        class Response:
            headers = {"Content-Type": "application/json"}

            def read(self, limit):
                return b" " * limit

        with self.assertRaises(PROBE.ProbeError) as error:
            PROBE.read_rpc(Response(), 1)
        self.assertEqual(error.exception.category, "response_too_large")

    def test_unexpected_sse_server_request_is_never_answered(self):
        class Response(io.BytesIO):
            headers = {"Content-Type": "text/event-stream"}

        response = Response(b'data: {"jsonrpc":"2.0","id":3,"method":"elicitation/create"}\n\n')
        with self.assertRaises(PROBE.ProbeError) as error:
            PROBE.read_rpc(response, 1)
        self.assertEqual(error.exception.category, "unexpected_server_request")

    def test_main_report_has_no_endpoint_token_or_html(self):
        with fixture() as (server, url), patch.dict(os.environ, {"PORTICO_PROBE_TOKEN": "private-test-bearer"}):
            out = io.StringIO()
            with contextlib.redirect_stdout(out):
                exit_code = PROBE.main(["--url", url, "--token-env", "PORTICO_PROBE_TOKEN",
                                        "--profile", "apps", "--protocol", PROBE.MODERN])
            self.assertEqual(exit_code, 0)
            data = json.loads(out.getvalue())
            self.assertEqual(data["scope"], "metadata_only_no_tool_calls_no_decisions")
            for private in (url, "private-test-bearer", "NEVER_EXECUTED", "fixture-session"):
                self.assertNotIn(private, out.getvalue())

    def test_main_invalid_token_never_echoes_it(self):
        out = io.StringIO()
        with patch.dict(os.environ, {"PORTICO_PROBE_TOKEN": "secret\n-injection"}), \
                contextlib.redirect_stderr(out), self.assertRaises(SystemExit):
            PROBE.main(["--url", "http://127.0.0.1/mcp", "--token-env", "PORTICO_PROBE_TOKEN"])
        self.assertNotIn("secret", out.getvalue())


if __name__ == "__main__":
    unittest.main()
