#!/usr/bin/env python3
"""Loopback-only HTTPS fixture: production operator BFF, simulated Broker and host.

This does not connect to an external MCP client, production node or real Broker.
The browser exercises the actual portal HTML/JS, cookies, CSP and HTTP handlers.
"""
import datetime as dt
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import importlib.util
import json
import os
from pathlib import Path
import signal
import ssl
import subprocess
import tempfile
import threading
import time
from urllib.parse import parse_qs, urlsplit

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("portico_browser_operator", ROOT / "web/operator-approval/server.py")
app = importlib.util.module_from_spec(spec)
spec.loader.exec_module(app)
PASSWORD = "browser-fixture-passphrase"
READ = "apr_browser_read_1234"
READ_TWO = "apr_browser_other_1234"
CRITICAL = "apr_browser_work_1234"
SENSITIVE = "apr_browser_sensitive_1234"
NODE = "browser-fixture-node"
LOCK = threading.Lock()
ITEMS = {}
CALLS = []
HTTP_CALLS = []


def reset():
    now = dt.datetime.now(dt.timezone.utc)
    with LOCK:
        ITEMS.clear(); CALLS.clear(); HTTP_CALLS.clear()
        for request_id, kind, access, target in (
                (READ, "root", "read", "/opt/projects/browser-test"),
                (READ_TWO, "root", "read", "/opt/projects/browser-second"),
                (CRITICAL, "root", "work", "/opt/projects/browser-test"),
                (SENSITIVE, "sensitive", "read", "/opt/projects/browser-test/.env")):
            ITEMS[request_id] = {"id": request_id, "kind": kind, "access": access,
                                 "target": target, "subject": "browser-fixture-agent",
                                 "ttl_ns": 300_000_000_000, "expires": now + dt.timedelta(minutes=5),
                                 "fingerprint": hashlib.sha256(request_id.encode()).hexdigest(),
                                 "node_id": NODE, "status": "pending"}
    with app.LOCK:
        app.SESSIONS.clear(); app.NONCES.clear(); app.STEPUPS.clear(); app.LOGIN_FAILURES.clear()


def list_requests():
    with LOCK:
        return [dict(item) for item in ITEMS.values() if item["status"] == "pending"]


def broker_call(action, request_id=None, *, snapshot_hash=None, node_id=None, step_up=False):
    with LOCK:
        item = ITEMS.get(request_id)
        if not item:
            raise RuntimeError("fixture request missing")
        if action == "status":
            return {"request_id": request_id, "status": item["status"], "subject": item["subject"],
                    "resource": item["target"], "access": item["access"],
                    "delegation_ttl_seconds": 300, "approval_expires_at": item["expires"].isoformat()}
        if action not in ("approve", "deny") or item["status"] != "pending":
            raise RuntimeError("fixture refuses replay")
        if snapshot_hash != item["fingerprint"] or node_id != NODE:
            raise RuntimeError("fixture refuses changed snapshot/node")
        critical = item["kind"] != "root" or item["access"] != "read"
        if action == "approve" and critical and not step_up:
            raise RuntimeError("fixture refuses missing step-up")
        CALLS.append({"action": action, "request_id": request_id, "step_up": step_up})
        item["status"] = "approved" if action == "approve" else "denied"
        return {"request_id": request_id, "status": item["status"]}


app.OperatorIPC.request = staticmethod(lambda rid: next((item for item in list_requests() if item["id"] == rid), None))
app.OperatorIPC.call = staticmethod(broker_call)


class PortalHandler(app.Handler):
    def reply(self, status, obj, extra_headers=None):
        with LOCK:
            HTTP_CALLS.append({"method": self.command, "path": urlsplit(self.path).path, "status": status})
        super().reply(status, obj, extra_headers)


class HarnessHandler(BaseHTTPRequestHandler):
    def log_message(self, *_):
        pass

    def send_bytes(self, data, content_type="text/html; charset=utf-8"):
        self.send_response(200)
        self.send_header("Content-Type", content_type)
        self.send_header("Cache-Control", "no-store")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers(); self.wfile.write(data)

    def do_GET(self):
        parsed = urlsplit(self.path)
        if parsed.path == "/__fixture/reset":
            reset(); self.send_bytes(b'{"reset":true}', "application/json"); return
        if parsed.path == "/__fixture/state":
            with LOCK:
                state = {"decisions": list(CALLS), "http": list(HTTP_CALLS),
                         "statuses": {key: value["status"] for key, value in ITEMS.items()}}
            self.send_bytes(json.dumps(state).encode(), "application/json"); return
        if parsed.path == "/app":
            config = {"operatorOrigin": self.server.operator_origin, "embedded": True}
            html = (ROOT / "internal/gateway/approval_app.html").read_text().replace("__PORTICO_CONFIG__", json.dumps(config))
            self.send_bytes(html.encode()); return
        if parsed.path != "/host":
            self.send_error(404); return
        params = parse_qs(parsed.query)
        request_id = params.get("request", [READ])[0]
        if request_id not in ITEMS:
            self.send_error(400); return
        config = {"requestID": request_id, "operatorOrigin": self.server.operator_origin,
                  "appOrigin": self.server.app_origin, "frame": params.get("frame", ["allowed"])[0]}
        html = """<!doctype html><html lang='en'><meta charset='utf-8'><title>Reference MCP Apps host harness</title>
<style>body{margin:0;font-family:system-ui}iframe{width:100%;height:1100px;border:0}</style>
<h1>Reference host harness</h1><iframe id='app' sandbox='allow-scripts allow-same-origin' title='MCP Apps test resource'></iframe>
<script>
const config=__CONFIG__;
window.bridgeMessages=[];
const frame=document.getElementById('app');
frame.src=config.appOrigin+'/app';
function send(value){frame.contentWindow.postMessage(value,config.appOrigin)}
window.sendHostMessage=send;
window.addEventListener('message', async event=>{
 if(event.source!==frame.contentWindow||event.origin!==config.appOrigin)return;
 const value=event.data;window.bridgeMessages.push(value);
 if(value?.method==='ui/initialize'){
  send({jsonrpc:'2.0',id:value.id,result:{protocolVersion:'2026-01-26',hostCapabilities:{openLinks:{},sandbox:{csp:{frameDomains:config.frame==='allowed'?[config.operatorOrigin]:[]}}}}});
 }
 if(value?.method==='ui/notifications/initialized')send({jsonrpc:'2.0',method:'ui/notifications/tool-result',params:{structuredContent:{request_id:config.requestID,operator_approval_url:config.operatorOrigin+'/operator?request='+config.requestID}}});
 if(value?.method==='tools/call'){
  if(value.params?.name!=='permissions.approval_status'){send({jsonrpc:'2.0',id:value.id,error:{code:-32601,message:'No approval tool'}});return}
  const state=await (await fetch('/__fixture/state')).json();
  send({jsonrpc:'2.0',id:value.id,result:{structuredContent:{request_id:config.requestID,status:state.statuses[config.requestID]}}});
 }
 if(value?.method==='ui/open-link'){
  const expected=config.operatorOrigin+'/operator?request='+config.requestID;
  if(value.params?.url!==expected){send({jsonrpc:'2.0',id:value.id,error:{code:-32602,message:'Unexpected portal URL'}});return}
  window.open(expected,'_blank','noopener,noreferrer');
  send({jsonrpc:'2.0',id:value.id,result:{}});
 }
});
</script></html>"""
        self.send_bytes(html.replace("__CONFIG__", json.dumps(config)).encode())


def https_server(handler, tls):
    server = ThreadingHTTPServer(("127.0.0.1", 0), handler)
    server.socket = tls.wrap_socket(server.socket, server_side=True)
    return server


def main():
    # Certificates and synthetic verifier stay outside the repository.
    with tempfile.TemporaryDirectory(prefix="portico-browser-") as temp:
        cert, key = Path(temp) / "cert.pem", Path(temp) / "key.pem"
        subprocess.run(["openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes",
                        "-keyout", str(key), "-out", str(cert), "-days", "1",
                        "-subj", "/CN=localhost", "-addext", "subjectAltName=DNS:localhost,IP:127.0.0.1,DNS:operator.portico.test,DNS:host.aiclient.test,DNS:host.secondclient.test,DNS:app.mcpapps.test"],
                       check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        tls = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        tls.load_cert_chain(cert, key)
        portal = https_server(PortalHandler, tls)
        host = https_server(HarnessHandler, tls)
        resource = https_server(HarnessHandler, tls)
        operator_origin = f"https://operator.portico.test:{portal.server_port}"
        host_origin = f"https://host.aiclient.test:{host.server_port}"
        other_host_origin = f"https://host.secondclient.test:{host.server_port}"
        app_origin = f"https://app.mcpapps.test:{resource.server_port}"
        control_origin = f"https://127.0.0.1:{host.server_port}"
        salt = b"portico-test-salt-only-1234"
        digest = hashlib.scrypt(PASSWORD.encode(), salt=salt, n=2**14, r=8, p=1, dklen=32)
        os.environ.update({"PORTICO_OPERATOR_PUBLIC_ORIGIN": operator_origin,
                           "PORTICO_OPERATOR_PASSWORD_SCRYPT": salt.hex() + ":" + digest.hex(),
                           "PORTICO_OPERATOR_APPROVAL_TOKEN": "browser-fixture-scoped-token-synthetic",
                           "PORTICO_OPERATOR_ID": "browser-fixture-owner", "PORTICO_OPERATOR_NODE_ID": NODE,
                           "PORTICO_OPERATOR_NODE_LABEL": "Browser fixture node",
                           "PORTICO_OPERATOR_FRAME_ANCESTORS": host_origin + " " + other_host_origin + " " + app_origin})
        for server in (host, resource):
            server.operator_origin, server.app_origin = operator_origin, app_origin
        reset()
        for server in (portal, host, resource):
            threading.Thread(target=server.serve_forever, daemon=True).start()
        print(json.dumps({"operator": operator_origin, "host": host_origin, "otherHost": other_host_origin, "app": app_origin, "control": control_origin,
                          "read": READ, "other": READ_TWO, "critical": CRITICAL, "sensitive": SENSITIVE,
                          "password": PASSWORD}), flush=True)
        done = threading.Event()
        signal.signal(signal.SIGTERM, lambda *_: done.set())
        signal.signal(signal.SIGINT, lambda *_: done.set())
        while not done.wait(0.2):
            pass
        for server in (portal, host, resource):
            server.shutdown(); server.server_close()


if __name__ == "__main__":
    main()
