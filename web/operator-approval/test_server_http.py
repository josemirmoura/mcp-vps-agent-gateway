"""HTTP contract tests with a stubbed Broker; no real authority granted."""
import contextlib
import http.cookiejar
import http.server
import importlib.util
import json
import os
from pathlib import Path
import threading
import time
import unittest
from unittest.mock import patch
from urllib.request import Request, build_opener, HTTPCookieProcessor
from urllib.error import HTTPError

HERE=Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location("portico_http_contract",HERE/"server.py")
app=importlib.util.module_from_spec(spec)
spec.loader.exec_module(app)
ID="apr_12345678"

class HttpContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server=http.server.ThreadingHTTPServer(("127.0.0.1",0),app.Handler)
        cls.thread=threading.Thread(target=cls.server.serve_forever,daemon=True)
        cls.thread.start()
        cls.base=f"http://127.0.0.1:{cls.server.server_port}"
    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join(timeout=5)
    def setUp(self):
        app.LOGIN_FAILURES.clear()
        app.SESSIONS.clear()
        self.client=build_opener(HTTPCookieProcessor(http.cookiejar.CookieJar()))
    def req(self,path,method="GET",payload=None,headers=None):
        h={"Origin":"https://operator.example.test"} if method=="POST" else {}
        h.update(headers or {})
        data=json.dumps(payload).encode() if payload is not None else None
        if data is not None:h["Content-Type"]="application/json"
        return Request(self.base+path,method=method,data=data,headers=h)
    def login(self):
        with patch.object(app,"verify_password",return_value=True):
            return self.client.open(self.req("/api/operator/login","POST",{"password":"valid"}))
    def test_get_details_needs_login(self):
        with self.assertRaises(HTTPError) as c:
            self.client.open(self.req(f"/api/operator/approvals/{ID}"))
        self.assertEqual(c.exception.code,401)
    def test_login_requires_matching_origin(self):
        with self.assertRaises(HTTPError) as c:
            self.client.open(self.req("/api/operator/login","POST",{"password":"x"},{"Origin":"https://attacker.test"}))
        self.assertEqual(c.exception.code,403)
    def test_login_and_read_details(self):
        item={"id":ID,"subject":"subject","target":"/opt/project","access":"work",
              "ttl_ns":900000000000,"expires":app.dt.datetime.now(app.dt.timezone.utc)+app.dt.timedelta(minutes=8),"kind":"root"}
        with patch.dict(os.environ,{"PORTICO_OPERATOR_PUBLIC_ORIGIN":"https://operator.example.test",
            "PORTICO_OPERATOR_PHYSICAL_CEILING":"/opt"}),patch.object(app.OperatorIPC,"list_requests",return_value=[item]):
            with patch.object(app,"verify_password",return_value=True):
                self.client.open(self.req("/api/operator/login","POST",{"password":"valid"}))
            result=json.load(self.client.open(self.req(f"/api/operator/approvals/{ID}")))
            self.assertEqual(result["resource"],"/opt/project")
            self.assertFalse(result["ceiling_wide"])
            self.assertTrue(result["csrf_token"])
            with self.assertRaises(HTTPError) as e:
                self.client.open(self.req(f"/api/operator/approvals/{ID}/decision","POST",{"decision":"approve"}))
            self.assertEqual(e.exception.code,403)
            with patch.object(app.OperatorIPC,"call",return_value={"status":"approved"}) as call:
                response=json.load(self.client.open(self.req(f"/api/operator/approvals/{ID}/decision",
                         "POST",{"decision":"approve"},{"X-CSRF-Token":result["csrf_token"]})))
                self.assertEqual(response["status"],"approved")
                call.assert_called_once_with("approve",ID)
if __name__=="__main__":
    unittest.main()
