#!/usr/bin/env python3
"""Offline regression tests for the trusted operator approval CLI."""
from __future__ import annotations

import contextlib
import datetime as dt
import importlib.util
import io
import pathlib
import sys
import types
import unittest
from unittest import mock

SCRIPT = pathlib.Path(__file__).resolve().parents[2] / "scripts" / "operator-approvals.py"
spec = importlib.util.spec_from_file_location("portico_operator_approvals", SCRIPT)
app = importlib.util.module_from_spec(spec)
spec.loader.exec_module(app)

FUTURE = "2099-10-08T22:56:00Z"
REQUEST = "apr_d3480a11b68886f8c8164d9f362f63d3"


def root(**kwargs):
    values = {
        "ID": REQUEST, "Subject": "alice", "TTL": 120 * 1_000_000_000,
        "Kind": "root", "Resource": "/opt/project", "Access": "read",
        "Status": "pending", "ExpiresAt": FUTURE, "Capabilities": [],
    }
    values.update(kwargs)
    return values


class ApprovalTests(unittest.TestCase):
    def test_root_and_permanent_expiry_are_renderable(self):
        item = app.request_detail(root(TTL=0))
        self.assertEqual(item["target"], "/opt/project")
        self.assertEqual(item["access"], "read")
        self.assertEqual(item["ttl_ns"], 0)
        with mock.patch.object(app, "physical_ceiling", return_value="/opt/project"):
            output = io.StringIO()
            with contextlib.redirect_stdout(output):
                app.present(item)
            self.assertIn("ESCOPO AMPLO", output.getvalue())
            self.assertIn("Permanente", output.getvalue())

    def test_sensitive_capability_decodes_without_printing_secret(self):
        cap = "sensitive.work:" + "/opt/project/.env".encode().hex()
        item = app.request_detail(root(Kind="capability", Resource="", Access="",
                                       Capabilities=[cap]))
        self.assertEqual((item["title"], item["target"], item["access"]),
                         ("Arquivo protegido", "/opt/project/.env", "work"))

    def test_denies_malicious_or_incomplete_requests(self):
        for candidate in (
            root(Status="approved"), root(ExpiresAt="2000-01-01T00:00:00Z"),
            root(ExpiresAt="no-time"), root(ExpiresAt="2026-01-01T00:00:00"),
            root(Resource="/opt/new\nFake approved"), root(Resource="/opt/\u202ehidden"),
            root(Subject="evil\noperator"), root(Access="admin"),
            root(TTL=-1), root(TTL=True), root(ID="apr_bad;touch /tmp/x"),
            root(Kind="capability", Capabilities=["shell.admin"]),
            root(Kind="capability", Capabilities=["sensitive.read:xx"]),
            root(Kind="capability", Capabilities=["sensitive.read:2f746d7000"]),
        ):
            with self.subTest(candidate=candidate):
                self.assertIsNone(app.request_detail(candidate))

    def test_without_tty_approval_is_impossible(self):
        item = app.request_detail(root())
        fake_sys = types.SimpleNamespace(stdin=types.SimpleNamespace(isatty=lambda: False),
                                         stderr=sys.stderr)
        with mock.patch.object(app, "sys", fake_sys):
            with mock.patch.object(app, "cli_call") as call:
                self.assertFalse(app.decide(item))
                call.assert_not_called()

    def test_exact_phrase_required(self):
        item = app.request_detail(root())
        fake_sys = types.SimpleNamespace(stdin=types.SimpleNamespace(isatty=lambda: True),
                                         stderr=sys.stderr)
        with mock.patch.object(app, "sys", fake_sys), mock.patch("builtins.input", return_value="sim"):
            with mock.patch.object(app, "cli_call") as call:
                self.assertFalse(app.decide(item))
                call.assert_not_called()

    def test_stale_request_cannot_be_approved(self):
        item = app.request_detail(root())
        fake_sys = types.SimpleNamespace(stdin=types.SimpleNamespace(isatty=lambda: True),
                                         stderr=sys.stderr)
        with mock.patch.object(app, "sys", fake_sys), mock.patch("builtins.input", return_value=f"APROVAR {REQUEST}"):
            with mock.patch.object(app, "list_requests", return_value=[]):
                with mock.patch.object(app, "cli_call") as call:
                    self.assertFalse(app.decide(item))
                    call.assert_not_called()

    def test_approved_root_only_after_confirmation_and_refresh(self):
        item = app.request_detail(root())
        fake_sys = types.SimpleNamespace(stdin=types.SimpleNamespace(isatty=lambda: True),
                                         stderr=sys.stderr)
        with mock.patch.object(app, "sys", fake_sys), mock.patch("builtins.input", return_value=f"APROVAR {REQUEST}"):
            with mock.patch.object(app, "list_requests", return_value=[item]):
                with mock.patch.object(app, "cli_call", return_value={"status": "approved"}) as call:
                    self.assertTrue(app.decide(item))
                    call.assert_called_once_with("approve", "--request", REQUEST)

    def test_denial_requires_explicit_phrase(self):
        item = app.request_detail(root())
        fake_sys = types.SimpleNamespace(stdin=types.SimpleNamespace(isatty=lambda: True),
                                         stderr=sys.stderr)
        with mock.patch.object(app, "sys", fake_sys), mock.patch("builtins.input", return_value=f"NEGAR {REQUEST}"):
            with mock.patch.object(app, "list_requests", return_value=[item]):
                with mock.patch.object(app, "cli_call", return_value={"status": "denied"}) as call:
                    self.assertTrue(app.decide(item))
                    call.assert_called_once_with("deny", "--request", REQUEST)

    def test_broker_errors_fail_closed(self):
        mock_result = types.SimpleNamespace(returncode=1, stdout="", stderr="secret")
        with mock.patch.object(app.subprocess, "run", return_value=mock_result):
            with self.assertRaisesRegex(RuntimeError, "O Broker recusou"):
                app.cli_call("approvals")


if __name__ == "__main__":
    unittest.main()
