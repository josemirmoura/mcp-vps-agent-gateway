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
        self.env=patch.dict(os.environ,{"PORTICO_OPERATOR_PUBLIC_ORIGIN":"https://operator.example.test"})
        self.env.start()
        self.addCleanup(self.env.stop)
        app.LOGIN_FAILURES.clear()
        app.SESSIONS.clear()
        self.client=build_opener(HTTPCookieProcessor(http.cookiejar.CookieJar()))
    def req(self,path,method="GET",payload=None,headers=None):
        h={"Host":"operator.example.test"}
        if method=="POST": h["Origin"]="https://operator.example.test"
        h.update(headers or {})
        data=json.dumps(payload).encode() if payload is not None else None
        if data is not None:h["Content-Type"]="application/json"
        return Request(self.base+path,method=method,data=data,headers=h)
    def login(self):
        with patch.object(app,"verify_password",return_value=True):
            return self.client.open(self.req("/operator/api/login","POST",{"password":"valid"}))
    def test_portal_static_assets_and_strict_csp(self):
        html=self.client.open(self.req("/operator")).read().decode()
        self.assertIn('/operator/app.js',html)
        self.assertIn('/operator/style.css',html)
        self.assertNotIn('<script>',html)
        for path,typ in [("/operator/app.js","text/javascript"),("/operator/style.css","text/css")]:
            resp=self.client.open(self.req(path))
            self.assertIn(typ,resp.headers["Content-Type"])
            csp=resp.headers["Content-Security-Policy"]
            self.assertIn("script-src 'self'",csp)
            self.assertIn("style-src 'self'",csp)
            self.assertNotIn("unsafe-inline",csp)
            self.assertGreater(len(resp.read()),50)
    def test_reject_unexpected_host_and_legacy_api(self):
        with self.assertRaises(HTTPError) as rejected:
            self.client.open(self.req("/operator",headers={"Host":"attacker.example.test"}))
        self.assertEqual(rejected.exception.code,421)
        with self.assertRaises(HTTPError) as legacy:
            self.client.open(self.req("/api/operator/login","POST",{"password":"whatever"}))
        self.assertEqual(legacy.exception.code,404)
    def test_get_details_needs_login(self):
        with self.assertRaises(HTTPError) as c:
            self.client.open(self.req(f"/operator/api/approvals/{ID}"))
        self.assertEqual(c.exception.code,401)
    def test_login_requires_matching_origin(self):
        with self.assertRaises(HTTPError) as c:
            self.client.open(self.req("/operator/api/login","POST",{"password":"x"},{"Origin":"https://attacker.test"}))
        self.assertEqual(c.exception.code,403)
    def test_login_and_read_details(self):
        item={"id":ID,"subject":"subject","target":"/opt/project","access":"work",
              "ttl_ns":900000000000,"expires":app.dt.datetime.now(app.dt.timezone.utc)+app.dt.timedelta(minutes=8),"kind":"root", "fingerprint":"a"*64, "node_id":""}
        with patch.dict(os.environ,{"PORTICO_OPERATOR_PUBLIC_ORIGIN":"https://operator.example.test",
            "PORTICO_OPERATOR_PHYSICAL_CEILING":"/opt"}),patch.object(app.OperatorIPC,"request",return_value=item):
            with patch.object(app,"verify_password",return_value=True):
                login=self.client.open(self.req("/operator/api/login","POST",{"password":"valid"}))
                self.assertIn("Path=/operator",login.headers["Set-Cookie"])
                cookie=login.headers["Set-Cookie"].split(";",1)[0]
            result=json.load(self.client.open(self.req(f"/operator/api/approvals/{ID}",headers={"Cookie":cookie})))
            self.assertEqual(result["resource"],"/opt/project")
            self.assertFalse(result["ceiling_wide"])
            self.assertTrue(result["csrf_token"])
            with self.assertRaises(HTTPError) as e:
                self.client.open(self.req(f"/operator/api/approvals/{ID}/decision","POST",{"decision":"approve"},{"Cookie":cookie}))
            self.assertEqual(e.exception.code,403)
            with patch.object(app.OperatorIPC,"call",return_value={"request_id":ID,"status":"denied"}) as call:
                response=json.load(self.client.open(self.req(f"/operator/api/approvals/{ID}/decision",
                         "POST",{"decision":"deny", "decision_nonce":result["decision_nonce"]},{"X-CSRF-Token":result["csrf_token"],"Cookie":cookie})))
                self.assertEqual(response["status"],"denied")
                call.assert_called_once_with("deny",ID,snapshot_hash="a"*64,node_id="",step_up=False)
if __name__=="__main__":
    unittest.main()
