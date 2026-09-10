"""Integrity regressions for published migration bytes and additive errata."""
import hashlib
import json
from pathlib import Path
import tempfile
import unittest

from migration_checksums import validate


class MigrationChecksumsTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.directory = Path(self.temporary.name)
        self.name = "00002_example.sql"
        self.sql = b"-- migration\nSELECT 1;\n"
        self.old = hashlib.sha256(self.sql.replace(b"\n", b"\r\n")).hexdigest()
        self.published = hashlib.sha256(self.sql).hexdigest()
        (self.directory / self.name).write_bytes(self.sql)
        (self.directory / "manifest.sha256").write_text(
            self.old + "  " + self.name + "\n", encoding="utf-8")
        self.correction = {
            "migration": self.name, "manifest_sha256": self.old,
            "published_sha256": self.published, "source_commit": "a" * 40,
            "reason": "Fixture: CRLF manifest before LF publication.",
        }
        self.write_corrections([self.correction])

    def write_corrections(self, corrections):
        (self.directory / "checksum-corrections.json").write_text(json.dumps(corrections), encoding="utf-8")

    def test_published_lf_passes_with_historical_manifest_unchanged(self):
        self.assertEqual(validate(self.directory), [])
        self.assertIn(self.old, (self.directory / "manifest.sha256").read_text())

    def test_original_crlf_is_not_an_alternate_accepted_digest(self):
        (self.directory / self.name).write_bytes(self.sql.replace(b"\n", b"\r\n"))
        self.assertTrue(validate(self.directory))

    def test_changed_sql_fails(self):
        (self.directory / self.name).write_bytes(self.sql + b"SELECT 2;\n")
        self.assertTrue(validate(self.directory))

    def test_missing_and_extra_migrations_fail(self):
        (self.directory / self.name).unlink()
        self.assertTrue(validate(self.directory))
        (self.directory / self.name).write_bytes(self.sql)
        (self.directory / "00003_extra.sql").write_bytes(self.sql)
        self.assertTrue(validate(self.directory))

    def test_unrelated_manifest_cannot_use_correction(self):
        self.correction["manifest_sha256"] = "b" * 64
        self.write_corrections([self.correction])
        self.assertTrue(validate(self.directory))

    def test_missing_duplicate_and_invalid_corrections_fail(self):
        for corrections in ([], [self.correction, self.correction], [{"migration": self.name}]):
            with self.subTest(corrections=corrections):
                self.write_corrections(corrections)
                self.assertTrue(validate(self.directory))

    def test_uncorrected_migration_is_still_checked(self):
        self.write_corrections([])
        (self.directory / "manifest.sha256").write_text(
            self.published + "  " + self.name + "\n", encoding="utf-8")
        self.assertEqual(validate(self.directory), [])
        (self.directory / self.name).write_bytes(b"changed")
        self.assertTrue(validate(self.directory))


if __name__ == "__main__":
    unittest.main()
