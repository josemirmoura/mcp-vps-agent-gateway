"""Direct Unix IPC boundary regression test with a fake Broker."""
import importlib.util
import io
import json
import os
from pathlib import Path
import socket
import tempfile
import threading
import unittest
from unittest.mock import MagicMock, patch

SPEC=importlib.util.spec_from_file_location("portico_operator_ipc",Path(__file__).with_name("server.py"))
app=importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(app)

class IPCTests(unittest.TestCase):
    def test_rejects_crossed_invalid_and_incomplete_replies(self):
        replies = [
            b'{"id":"another-call","ok":true,"result":{}}\n',
            b'{"id":"fixture-id","ok":1,"result":{}}\n',
            b'{"id":"fixture-id","ok":true,"result":{}}',
        ]
        for reply in replies:
            with self.subTest(reply=reply):
                client = MagicMock()
                client.__enter__.return_value = client
                client.makefile.return_value = io.BytesIO(reply)
                with patch.object(app.socket, 'socket', return_value=client), \
                        patch.object(app.secrets, 'token_urlsafe', return_value='fixture-id'), \
                        patch.dict(os.environ, {'PORTICO_OPERATOR_APPROVAL_TOKEN':'t'*48}):
                    with self.assertRaises(RuntimeError):
                        app.OperatorIPC.call('approvals')

    def test_newline_framed_broker_call(self):
        with tempfile.TemporaryDirectory() as tmp:
            path=str(Path(tmp)/"bridge.sock")
            listener=socket.socket(socket.AF_UNIX,socket.SOCK_STREAM)
            listener.bind(path)
            listener.listen(1)
            captured=[]
            def accept_once():
                conn,_=listener.accept()
                with conn:
                    with conn.makefile("rb") as stream:
                        frame=stream.readline(8192)
                    captured.append(frame)
                    conn.sendall(json.dumps({"id":json.loads(frame)["id"],"ok":True,"result":[]}).encode()+bytes([10]))
            thread=threading.Thread(target=accept_once,daemon=True)
            thread.start()
            with patch.dict(os.environ,{"PORTICO_OPERATOR_SOCKET":path,
                      "PORTICO_OPERATOR_APPROVAL_TOKEN":"t"*48}):
                self.assertEqual(app.OperatorIPC.call("approvals"),[])
            thread.join(timeout=3)
            listener.close()
            self.assertEqual(len(captured),1)
            self.assertTrue(captured[0].endswith(bytes([10])))
            self.assertFalse(captured[0].endswith(b"\\n"))
            self.assertEqual(json.loads(captured[0])["tool"],"admin.approval.list")
    def test_unknown_command_fails_before_socket(self):
        with self.assertRaisesRegex(RuntimeError,"unsupported"):
            with patch.dict(os.environ,{"PORTICO_OPERATOR_APPROVAL_TOKEN":"t"*48}):
                app.OperatorIPC.call("admin.revoke_all")
if __name__=="__main__":
    unittest.main()
