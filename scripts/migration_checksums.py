"""Validate immutable migration bytes, including reviewed publication errata."""
import hashlib
import json
import re
from pathlib import Path


def validate(directory: Path) -> list[str]:
    try:
        entries = [line.split("  ", 1) for line in
                   (directory / "manifest.sha256").read_text(encoding="utf-8").splitlines() if line]
        expected = {name: digest for digest, name in entries}
        if len(expected) != len(entries):
            raise ValueError("Duplicate migration manifest entry")
        corrections = json.loads((directory / "checksum-corrections.json").read_text(encoding="utf-8"))
        seen = set()
        for correction in corrections:
            name = correction["migration"]
            old = correction["manifest_sha256"]
            published = correction["published_sha256"]
            if name in seen or expected.get(name) != old:
                raise ValueError("Checksum correction does not match a unique manifest entry")
            if not re.fullmatch(r"[0-9a-f]{64}", published) or published == old:
                raise ValueError("Invalid published migration digest")
            if not re.fullmatch(r"[0-9a-f]{40}", correction["source_commit"]) or not correction["reason"].strip():
                raise ValueError("Checksum correction needs publication provenance")
            expected[name] = published
            seen.add(name)
        actual = {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in directory.glob("*.sql")}
        if actual != expected:
            return ["Migration checksum mismatch: " + ", ".join(
                sorted(name for name in actual.keys() | expected.keys() if actual.get(name) != expected.get(name))
            ) + "; preserve published SQL and checksum history."]
    except (OSError, ValueError, KeyError, TypeError) as exc:
        return ["Invalid migration checksum metadata: " + str(exc)]
    return []
