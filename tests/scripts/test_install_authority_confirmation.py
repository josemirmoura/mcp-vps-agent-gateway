"""Real Bash/PTY regression for the installation authority confirmation gate.

Runs the exact source section in a disposable mini-script using a pseudo-terminal,
so the installer sees an interactive stdin. It never invokes Docker, init,
OAuth, writes credentials, or touches an operator checkout.
"""
from __future__ import annotations

import os
from pathlib import Path
import pty
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
INSTALLER = ROOT / "scripts" / "install.sh"
START = '\nif [ "$ASSUME_YES" -ne 1 ]; then\n'
END = '\nvps_agent_step "$(vps_agent_text \'4/7 Containers\' \'4/7 Contêineres\')"'

class AuthorityConfirmationTest(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory(prefix="portico-authority-confirm-")
        self.addCleanup(temp.cleanup)
        self.tmp = Path(temp.name)
        source = INSTALLER.read_text(encoding="utf-8")
        self.assertEqual(source.count(START), 1)
        self.assertEqual(source.count(END), 1)
        self.gate = source[source.index(START):source.index(END)]
        self.assertIn('read -r confirm', self.gate)
        self.assertIn('exit 2', self.gate)

    def exercise(self, lang: str, answer: str):
        script = self.tmp / "authority-gate.sh"
        script.write_text(
            "#!/usr/bin/env bash\nset -euo pipefail\n"
            "ASSUME_YES=0\nVPS_AGENT_LANG=" + lang + "\n"
            "vps_agent_is_pt_br() { [ \"$VPS_AGENT_LANG\" = pt-BR ]; }\n"
            "vps_agent_text() { if vps_agent_is_pt_br; then printf '%s' \"$2\"; else printf '%s' \"$1\"; fi; }\n"
            + self.gate +
            "\nprintf 'NEXT_STAGE_REACHED\\n'\n",
            encoding="utf-8",
        )
        master, slave = pty.openpty()
        try:
            process = subprocess.Popen(
                ["bash", str(script)], stdin=slave, stdout=subprocess.PIPE,
                stderr=subprocess.PIPE, text=True, cwd=self.tmp,
            )
            os.close(slave)
            slave = -1
            os.write(master, (answer + "\n").encode())
            stdout, stderr = process.communicate(timeout=8)
            return process.returncode, stdout + stderr
        finally:
            os.close(master)
            if slave != -1:
                os.close(slave)

    def test_portuguese_decline_is_failure_with_resume(self):
        code, output = self.exercise("pt-BR", "NÃO")
        self.assertEqual(code, 2, output)
        self.assertIn("Instalação cancelada", output)
        self.assertIn("retomar", output)
        self.assertNotIn("NEXT_STAGE_REACHED", output)

    def test_english_decline_is_failure_with_resume(self):
        code, output = self.exercise("en", "CANCEL")
        self.assertEqual(code, 2, output)
        self.assertIn("Installation cancelled", output)
        self.assertIn("resume", output)
        self.assertNotIn("NEXT_STAGE_REACHED", output)

    def test_blank_confirmation_does_not_start(self):
        for lang in ("pt-BR", "en"):
            with self.subTest(lang=lang):
                code, output = self.exercise(lang, "")
                self.assertEqual(code, 2, output)
                self.assertNotIn("NEXT_STAGE_REACHED", output)

    def test_explicit_confirmation_advances_in_both_languages(self):
        for lang, answer in (("pt-BR", "CONTINUAR"), ("en", "CONTINUE")):
            with self.subTest(lang=lang):
                code, output = self.exercise(lang, answer)
                self.assertEqual(code, 0, output)
                self.assertIn("NEXT_STAGE_REACHED", output)


if __name__ == "__main__":
    unittest.main()
