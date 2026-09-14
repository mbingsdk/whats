"""Regressions for explicit operator input and local credential boundaries."""
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import dev
from operator_env import load_operator_env


class OperatorEnvTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.path = Path(self.temp.name) / "operator.env"

    def write(self, text):
        self.path.write_bytes(text.encode("utf-8"))
        return str(self.path)

    def test_bom_quotes_windows_paths_and_literal_values(self):
        file = self.write("\ufeff# config\nPUBLIC_ORIGIN=http://localhost:3000\n"
                          "BOOTSTRAP_PASSWORD_FILE='C:\\private folder\\password'\n"
                          'SMTP_USERNAME="$(literal)#value=kept"\n')
        values = load_operator_env([file])
        self.assertEqual(values["BOOTSTRAP_PASSWORD_FILE"], "C:\\private folder\\password")
        self.assertEqual(values["SMTP_USERNAME"], "$(literal)#value=kept")
        self.assertEqual(load_operator_env([]), {})

    def test_malformed_input_never_exposes_value(self):
        for text in ("PATH=private-value", "BOOTSTRAP_EMAIL=private-value\nBOOTSTRAP_EMAIL=again",
                     'SMTP_USERNAME="private-value', "private-value", "SMTP_USERNAME=private-value\0"):
            with self.subTest(text=text):
                with self.assertRaises(ValueError) as result:
                    load_operator_env([self.write(text)])
                self.assertNotIn("private-value", str(result.exception))

    def test_missing_file_and_ordered_overrides(self):
        with self.assertRaisesRegex(ValueError, "Cannot read operator env file #1"):
            load_operator_env([str(self.path)])
        first = self.write("PUBLIC_ORIGIN=http://localhost:3000\n")
        second = Path(self.temp.name) / "second.env"
        second.write_text("PUBLIC_ORIGIN=http://localhost:3001\n", encoding="utf-8")
        self.assertEqual(load_operator_env([first, second])["PUBLIC_ORIGIN"], "http://localhost:3001")

    @patch.object(dev, "prepare_identity")
    @patch.object(dev, "prepare")
    def test_local_credentials_and_bootstrap_scope(self, _prepare, _identity):
        supplied = {"IDENTITY_DATABASE_URL_FILE": "untrusted", "IDENTITY_ROOT_KEY_FILE": "untrusted",
                    "DATABASE_URL_FILE": "untrusted", "MIGRATION_DATABASE_URL_FILE": "untrusted",
                    "BOOTSTRAP_EMAIL": "owner@example.invalid", "PUBLIC_ORIGIN": "http://localhost:3000"}
        for purpose in ("bootstrap", "runtime"):
            with self.subTest(purpose=purpose):
                env = dev.backend_env(purpose, supplied)
                self.assertEqual(env["IDENTITY_DATABASE_URL_FILE"], str(dev.LOCAL / "identity-url"))
                self.assertEqual(env["IDENTITY_ROOT_KEY_FILE"], str(dev.LOCAL / "identity-root-key"))
                self.assertEqual(env["DATABASE_URL_FILE"], str(dev.LOCAL / "database-url"))
                self.assertNotIn("MIGRATION_DATABASE_URL_FILE", env)
                self.assertEqual("BOOTSTRAP_EMAIL" in env, purpose == "bootstrap")
                self.assertEqual(env["PUBLIC_ORIGIN"], supplied["PUBLIC_ORIGIN"])

    def test_frontend_and_ci_cannot_load_operator_file(self):
        for action in ("frontend", "check", "test", "e2e", "up", "migrate"):
            with self.subTest(action=action), patch("sys.argv", ["dev.py", action, "--env-file", "missing"]), \
                    patch.object(dev, "load_operator_env") as load, patch("sys.stderr"):
                with self.assertRaises(SystemExit) as result:
                    dev.main()
                self.assertEqual(result.exception.code, 2)
                load.assert_not_called()


if __name__ == "__main__":
    unittest.main()
