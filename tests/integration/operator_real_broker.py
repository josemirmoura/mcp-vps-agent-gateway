#!/usr/bin/env python3
"""Disposable HTTP BFF -> restricted IPC -> actual Broker/SQLite fixture.

Invoked by Go integration test. No mocks of decisions, policy or IPC.
"""
import hashlib
import http.server
import importlib.util
import json
import os
from pathlib import Path
import threading
from urllib.error import HTTPError
from urllib.request import Request, urlopen

root = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("real_operator_bff", root / "web/operator-approval/server.py")
app = importlib.util.module_from_spec(spec)
spec.loader.exec_module(app)
password = "fixture-password-never-production"
salt = bytes.fromhex("12" * 16)
digest = hashlib.scrypt(password.encode(), salt=salt, n=2**14, r=8, p=1, dklen=32)
os.environ["PORTICO_OPERATOR_PASSWORD_SCRYPT"] = salt.hex() + ":" + digest.hex()
ids = json.loads(os.environ["PORTICO_FIXTURE_IDS"])
server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), app.Handler)
thread = threading.Thread(target=server.serve_forever, daemon=True)
thread.start()
base = f"http://127.0.0.1:{server.server_port}"
cookie = ""


def req(path, body=None, csrf=""):
    headers = {"Host": "operator.fixture.test", "Origin": "https://operator.fixture.test",
               "Cookie": cookie, "Content-Type": "application/json", "X-CSRF-Token": csrf}
    try:
        response = urlopen(Request(base + path, headers=headers,
                                   data=json.dumps(body).encode() if body is not None else None), timeout=5)
        return response.status, response.headers, json.load(response)
    except HTTPError as error:
        return error.code, error.headers, json.load(error)


try:
    code, headers, _ = req("/operator/api/login", {"password": password})
    assert code == 200, code
    cookie = headers["Set-Cookie"].split(";", 1)[0]
    for name in ("read", "denied", "work", "sensitive"):
        path = "/operator/api/approvals/" + ids[name]
        code, _, data = req(path)
        assert code == 200 and data["status"] == "pending", (name, code, data)
        assert data["node_id"] == "fixture-node" and data["operator"] == "fixture-owner"
        proof = {"decision_nonce": data["decision_nonce"]}
        csrf = data["csrf_token"]
        if name in ("work", "sensitive"):
            assert data["step_up_required"] is True
            assert req(path + "/decision", {**proof, "decision": "approve"}, csrf)[0] == 428
            assert req(path + "/step-up", {**proof, "password": "wrong"}, csrf)[0] == 403
            assert req(path + "/step-up", {**proof, "password": password}, csrf)[0] == 200
        decision = "deny" if name == "denied" else "approve"
        code, _, result = req(path + "/decision", {**proof, "decision": decision}, csrf)
        expected = "denied" if name == "denied" else "approved"
        assert code == 200 and result["status"] == expected, (name, code, result)
        assert req(path + "/decision", {**proof, "decision": decision}, csrf)[0] == 409
        code, _, terminal = req(path)
        assert code == 200 and terminal["status"] == expected, (name, code, terminal)
        assert "decision_nonce" not in terminal
    print("REAL_BROKER_BFF_OK: read approval, denial, work/protected step-up, session reuse, replay rejection, final status")
finally:
    server.shutdown()
    server.server_close()
    thread.join(5)
