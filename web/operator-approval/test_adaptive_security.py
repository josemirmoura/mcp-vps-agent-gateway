"""HTTP adversarial tests: operator sessions are independent of MCP identity."""
import datetime as dt
import http.server
import importlib.util
import json
import os
from pathlib import Path
import threading
import unittest
from unittest.mock import patch
from urllib.request import Request, urlopen
from urllib.error import HTTPError

spec = importlib.util.spec_from_file_location("adaptive_portal", Path(__file__).with_name("server.py"))
app = importlib.util.module_from_spec(spec)
spec.loader.exec_module(app)
ID = "apr_abcdefgh1234"


class OperatorSecurity(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), app.Handler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.base = f"http://127.0.0.1:{cls.server.server_port}"

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown(); cls.server.server_close(); cls.thread.join(5)

    def setUp(self):
        self.env = patch.dict(os.environ, {
            "PORTICO_OPERATOR_PUBLIC_ORIGIN": "https://operator.test",
            "PORTICO_OPERATOR_NODE_ID": "node-a", "PORTICO_OPERATOR_ID": "owner-a",
            "PORTICO_OPERATOR_FRAME_ANCESTORS": "https://host.test https://sandbox.test"})
        self.env.start(); self.addCleanup(self.env.stop)
        app.SESSIONS.clear(); app.NONCES.clear(); app.STEPUPS.clear(); app.LOGIN_FAILURES.clear()
        self.item = {"id": ID, "subject": "agent-a", "target": "/opt/project",
                     "access": "read", "ttl_ns": 300_000_000_000, "kind": "root",
                     "expires": dt.datetime.now(dt.timezone.utc) + dt.timedelta(minutes=5),
                     "fingerprint": "a" * 64, "node_id": "node-a"}

    def req(self, path, payload=None, cookie="", csrf="", origin="https://operator.test"):
        headers = {"Host": "operator.test", "Origin": origin, "Cookie": cookie,
                   "X-CSRF-Token": csrf, "Content-Type": "application/json"}
        data = json.dumps(payload).encode() if payload is not None else None
        try:
            response = urlopen(Request(self.base + path, data=data, headers=headers), timeout=5)
            body = response.read()
            return response.status, response.headers, json.loads(body) if "application/json" in response.headers.get("Content-Type", "") else body.decode()
        except HTTPError as error:
            return error.code, error.headers, json.load(error)

    def login(self, embedded=False):
        prefix = "/operator/embed/api" if embedded else "/operator/api"
        with patch.object(app, "verify_password", return_value=True):
            code, headers, _ = self.req(prefix + "/login", {"password": "synthetic"})
        self.assertEqual(code, 200)
        return headers["Set-Cookie"].split(";", 1)[0], headers["Set-Cookie"]

    def details(self, cookie, embedded=False):
        prefix = "/operator/embed/api" if embedded else "/operator/api"
        with patch.object(app.OperatorIPC, "request", return_value=self.item):
            code, _, result = self.req(prefix + "/approvals/" + ID, cookie=cookie)
        self.assertEqual(code, 200)
        return result

    def broker_row(self):
        return {
            "ID": self.item["id"], "Subject": self.item["subject"],
            "Resource": self.item["target"], "Access": self.item["access"],
            "TTL": self.item["ttl_ns"], "Kind": self.item["kind"],
            "ExpiresAt": self.item["expires"].isoformat(), "Status": "pending",
            "Fingerprint": self.item["fingerprint"], "NodeID": self.item["node_id"],
        }

    def assert_step_up_rejects_change(self, change, decision_code):
        """An in-flight password check cannot outlive its request/session binding."""
        self.item["access"] = "work"
        cookie, _ = self.login()
        details = self.details(cookie)
        nonce = details["decision_nonce"]
        path = "/operator/api/approvals/" + ID
        entered, release = threading.Event(), threading.Event()
        results, failures = [], []

        def blocked_verifier(password):
            entered.set()
            if not release.wait(5):
                raise AssertionError("test did not release password verification")
            return True

        def verify_request():
            try:
                results.append(self.req(
                    path + "/step-up", {"decision_nonce": nonce, "password": "synthetic"},
                    cookie, details["csrf_token"]))
            except Exception as error:
                failures.append(error)

        with patch.object(app, "verify_password", side_effect=blocked_verifier), \
                patch.object(app.OperatorIPC, "call") as call:
            worker = threading.Thread(target=verify_request, daemon=True)
            worker.start()
            try:
                self.assertTrue(entered.wait(5), "password verification was not reached")
                change(cookie, details)
            finally:
                release.set()
                worker.join(5)
            self.assertFalse(worker.is_alive(), "verification request did not finish")
            self.assertFalse(failures)
            self.assertEqual(len(results), 1)
            self.assertEqual(results[0][0], 409)
            self.assertNotIn(nonce, app.STEPUPS)
            self.assertEqual(self.req(
                path + "/decision", {"decision_nonce": nonce, "decision": "approve"},
                cookie, details["csrf_token"])[0], decision_code)
            call.assert_not_called()

    def test_top_level_remains_unframeable_and_embed_exact_allowlist(self):
        for path in ("/operator", "/operator/embed"):
            code, headers, _ = self.req(path)
            self.assertEqual(code, 200)
            if path.endswith("embed"):
                self.assertNotIn("X-Frame-Options", headers)
                self.assertIn("frame-ancestors https://host.test https://sandbox.test", headers["Content-Security-Policy"])
            else:
                self.assertEqual(headers["X-Frame-Options"], "DENY")
                self.assertIn("frame-ancestors 'none'", headers["Content-Security-Policy"])
        for value in ("*", "https://*.test", "http://host.test", "https://host.test/path", "https://host.test;unsafe", "https://user:pass@host.test"):
            with patch.dict(os.environ, {"PORTICO_OPERATOR_FRAME_ANCESTORS": value}):
                with self.assertRaises(ValueError): app.frame_ancestors()

    def test_embed_disabled_without_explicit_configuration(self):
        with patch.dict(os.environ, {"PORTICO_OPERATOR_FRAME_ANCESTORS": ""}):
            self.assertEqual(self.req("/operator/embed")[0], 404)
            self.assertEqual(self.req("/operator/embed/api/login", {"password": "x"})[0], 404)

    def test_partitioned_and_top_level_sessions_do_not_interchange(self):
        top, _ = self.login(); embedded, attributes = self.login(True)
        self.assertIn("HttpOnly", attributes); self.assertIn("Secure", attributes)
        self.assertIn("SameSite=None", attributes); self.assertIn("Partitioned", attributes)
        self.assertEqual(self.req("/operator/embed/api/approvals/" + ID, cookie=top)[0], 401)
        self.assertEqual(self.req("/operator/api/approvals/" + ID, cookie=embedded)[0], 401)

    def test_session_rotation_identity_swap_logout_and_password_change(self):
        cookie, _ = self.login(); details = self.details(cookie)
        for key, value in (("PORTICO_OPERATOR_ID", "other-owner"), ("PORTICO_OPERATOR_NODE_ID", "node-b"), ("PORTICO_OPERATOR_PASSWORD_SCRYPT", "rotated")):
            with patch.dict(os.environ, {key: value}):
                self.assertIsNone(app.session_from_cookie(cookie))
        code, _, _ = self.req("/operator/api/logout", {}, cookie, details["csrf_token"])
        self.assertEqual(code, 200); self.assertIsNone(app.session_from_cookie(cookie))
        self.assertEqual(self.req("/operator/api/approvals/" + ID + "/decision", {"decision": "approve", "decision_nonce": details["decision_nonce"]}, cookie, details["csrf_token"])[0], 401)

    def test_login_cannot_bind_old_password_verification_to_changed_operator_configuration(self):
        for key, value in (("PORTICO_OPERATOR_ID", "other-owner"),
                           ("PORTICO_OPERATOR_NODE_ID", "node-b"),
                           ("PORTICO_OPERATOR_PASSWORD_SCRYPT", "rotated"),
                           ("PORTICO_OPERATOR_APPROVAL_TOKEN", "rotated-scoped-credential")):
            with self.subTest(binding=key), patch.dict(os.environ):
                def rotate_during_verification(password):
                    os.environ[key] = value
                    return True
                with patch.object(app, "verify_password", side_effect=rotate_during_verification):
                    code, headers, _ = self.req("/operator/api/login", {"password": "previous-password"})
                self.assertEqual(code, 409)
                self.assertNotIn("Set-Cookie", headers)
                self.assertEqual(app.SESSIONS, {})

    def test_terminal_status_keeps_logout_available_without_issuing_a_decision_nonce(self):
        cookie, _ = self.login()
        with patch.object(app.OperatorIPC, "request", return_value=None), \
                patch.object(app.OperatorIPC, "call", return_value={"request_id": ID, "status": "approved"}):
            code, _, terminal = self.req("/operator/api/approvals/" + ID, cookie=cookie)
        self.assertEqual(code, 200)
        self.assertNotIn("decision_nonce", terminal)
        self.assertTrue(terminal["csrf_token"])
        self.assertEqual(app.NONCES, {})
        self.assertEqual(self.req("/operator/api/logout", {}, cookie, terminal["csrf_token"])[0], 200)
        self.assertIsNone(app.session_from_cookie(cookie))

    def test_one_shot_nonce_bound_to_session_request_and_broker_snapshot(self):
        cookie, _ = self.login(); details = self.details(cookie)
        other, _ = self.login(); other_details = self.details(other)
        body = {"decision": "approve", "decision_nonce": details["decision_nonce"]}
        path = "/operator/api/approvals/" + ID + "/decision"
        with patch.object(app.OperatorIPC, "call", return_value={"status": "approved"}) as call:
            self.assertEqual(self.req(path, body, other, other_details["csrf_token"])[0], 409)
            self.assertEqual(self.req(path.replace(ID, "apr_another1234"), body, cookie, details["csrf_token"])[0], 409)
            self.assertEqual(self.req(path, body, cookie, details["csrf_token"], "https://attacker.test")[0], 403)
            self.assertEqual(self.req(path, body, cookie, details["csrf_token"])[0], 200)
            self.assertEqual(self.req(path, body, cookie, details["csrf_token"])[0], 409)
            call.assert_called_once_with("approve", ID, snapshot_hash="a" * 64, node_id="node-a", step_up=False)

    def test_critical_approve_needs_request_bound_step_up_but_deny_does_not(self):
        self.item["access"] = "work"
        cookie, _ = self.login(); details = self.details(cookie)
        path = "/operator/api/approvals/" + ID
        body = {"decision_nonce": details["decision_nonce"], "decision": "approve"}
        with patch.object(app.OperatorIPC, "call", return_value={"status": "approved"}) as call:
            self.assertEqual(self.req(path + "/decision", body, cookie, details["csrf_token"])[0], 428)
            self.assertEqual(call.call_count, 0)
            with patch.object(app, "verify_password", return_value=True):
                self.assertEqual(self.req(path + "/step-up", {**body, "password": "synthetic"}, cookie, details["csrf_token"])[0], 200)
            self.assertEqual(self.req(path + "/decision", body, cookie, details["csrf_token"])[0], 200)
            call.assert_called_once_with("approve", ID, snapshot_hash="a" * 64, node_id="node-a", step_up=True)
        details = self.details(cookie)
        with patch.object(app.OperatorIPC, "call", return_value={"status": "denied"}):
            self.assertEqual(self.req(path + "/decision", {"decision_nonce": details["decision_nonce"], "decision": "deny"}, cookie, details["csrf_token"])[0], 200)

    def test_expired_nonce_and_network_loss_never_allow_blind_retry(self):
        cookie, _ = self.login(); details = self.details(cookie)
        path = "/operator/api/approvals/" + ID + "/decision"
        body = {"decision_nonce": details["decision_nonce"], "decision": "approve"}
        app.NONCES[body["decision_nonce"]]["expires"] = 0
        with patch.object(app.OperatorIPC, "call") as call:
            self.assertEqual(self.req(path, body, cookie, details["csrf_token"])[0], 409)
            call.assert_not_called()
        details = self.details(cookie); body["decision_nonce"] = details["decision_nonce"]
        with patch.object(app.OperatorIPC, "call", side_effect=OSError("synthetic loss")) as call:
            self.assertEqual(self.req(path, body, cookie, details["csrf_token"])[0], 503)
            self.assertEqual(self.req(path, body, cookie, details["csrf_token"])[0], 409)
            self.assertEqual(call.call_count, 1)

    def test_nonce_expiring_during_password_verification_cannot_authorize(self):
        def expire(cookie, details):
            with app.LOCK:
                app.NONCES[details["decision_nonce"]]["expires"] = 0

        self.assert_step_up_rejects_change(expire, 409)

    def test_logout_during_password_verification_cannot_authorize(self):
        def logout(cookie, details):
            code, _, _ = self.req(
                "/operator/api/logout", {}, cookie, details["csrf_token"])
            self.assertEqual(code, 200)
            self.assertIsNone(app.session_from_cookie(cookie))

        self.assert_step_up_rejects_change(logout, 401)

    def test_replaced_nonce_during_password_verification_has_no_fresh_proof(self):
        def replace(cookie, details):
            with app.LOCK:
                nonce = details["decision_nonce"]
                app.NONCES[nonce] = dict(app.NONCES[nonce])

        self.assert_step_up_rejects_change(replace, 428)

    def test_detail_uses_one_bound_broker_row_and_no_queue_listing(self):
        with patch.object(app.OperatorIPC, "call", return_value=self.broker_row()) as call:
            item = app.OperatorIPC.request(ID)
            call.assert_called_once_with("detail", ID)
        self.assertEqual(item["id"], ID)
        self.assertEqual(item["fingerprint"], "a" * 64)
        self.assertEqual(item["node_id"], "node-a")
        with patch.object(app.OperatorIPC, "call", return_value=None) as call:
            self.assertIsNone(app.OperatorIPC.request(ID))
            call.assert_called_once_with("detail", ID)

    def test_detail_rejects_invalid_fingerprint_node_and_swapped_request(self):
        cookie, _ = self.login()
        cases = (
            {"Fingerprint": None}, {"Fingerprint": "a" * 63},
            {"Fingerprint": "g" * 64}, {"Fingerprint": "A" * 64},
            {"NodeID": ""}, {"NodeID": "node-b"},
            {"ID": "apr_another1234"},
        )
        for changed in cases:
            with self.subTest(changed=changed):
                row = {**self.broker_row(), **changed}
                with patch.object(app.OperatorIPC, "call", return_value=row) as call:
                    code, _, response = self.req(
                        "/operator/api/approvals/" + ID, cookie=cookie)
                    self.assertEqual(code, 503)
                    self.assertEqual(response, {"error": "broker unavailable"})
                    call.assert_called_once_with("detail", ID)
                self.assertFalse(app.NONCES)

    def test_detail_and_startup_require_configured_destination_node(self):
        with patch.dict(os.environ, {"PORTICO_OPERATOR_NODE_ID": ""}), \
                patch.object(app.OperatorIPC, "call", return_value=self.broker_row()):
            with self.assertRaisesRegex(RuntimeError, "node mismatch"):
                app.OperatorIPC.request(ID)
        with patch.dict(os.environ, {
                "PORTICO_OPERATOR_NODE_ID": "",
                "PORTICO_OPERATOR_PASSWORD_SCRYPT": "synthetic-verifier",
                "PORTICO_OPERATOR_APPROVAL_TOKEN": "t" * 32,
                "PORTICO_OPERATOR_PHYSICAL_CEILING": "/opt"}), \
                patch.object(app, "ThreadingHTTPServer") as server:
            with self.assertRaisesRegex(SystemExit, "node ID"):
                app.main()
            server.assert_not_called()

    def test_repeat_low_risk_requests_reuse_authenticated_session(self):
        cookie, _ = self.login()
        for _ in range(3):
            details = self.details(cookie)
            self.assertEqual(details["operator"], "owner-a")
            self.assertFalse(details["step_up_required"])


if __name__ == "__main__":
    unittest.main()
