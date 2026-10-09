"""Unit and route regression smoke tests for the operator web service."""
import hashlib
import importlib.util
import os
from pathlib import Path
import secrets
import unittest
from unittest.mock import patch
from http.cookies import SimpleCookie

ROOT = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("portico_operator_web", ROOT / "server.py")
app = importlib.util.module_from_spec(spec)
spec.loader.exec_module(app)

class PortalSecurityTests(unittest.TestCase):
    def test_password_fails_closed_without_config(self):
        with patch.dict(os.environ, {}, clear=True):
            self.assertFalse(app.verify_password("anything"))
    def test_scrypt_success_and_bad_password(self):
        salt = secrets.token_bytes(16)
        digest = hashlib.scrypt(b"correct horse battery staple", salt=salt, n=2**14, r=8, p=1, dklen=32)
        with patch.dict(os.environ, {"PORTICO_OPERATOR_PASSWORD_SCRYPT": salt.hex() + ":" + digest.hex()}):
            self.assertTrue(app.verify_password("correct horse battery staple"))
            self.assertFalse(app.verify_password("wrong"))
    def test_random_cookie_no_session(self):
        self.assertIsNone(app.session_from_cookie("portico_operator=unknown"))
    def test_expired_session_rejected(self):
        with app.LOCK:
            app.SESSIONS["expired"] = (-1, "csrf")
        self.assertIsNone(app.session_from_cookie("portico_operator=expired"))
    def test_session_with_csrf(self):
        with app.LOCK:
            app.SESSIONS["valid"] = (float("inf"), "csrf")
        self.assertEqual(app.session_from_cookie("portico_operator=valid"), ("valid", "csrf"))
    def test_request_id_rejects_special_characters(self):
        self.assertIsNone(app.API_RE.fullmatch("/operator/api/approvals/apr_12345678/../../deny"))
        self.assertIsNone(app.API_RE.fullmatch("/operator/api/approvals/apr_12345678%2Fdecision"))
    def test_portal_path_exists(self):
        self.assertTrue(app.UI_PATH.is_file())

if __name__ == "__main__":
    unittest.main()
