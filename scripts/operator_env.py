"""Explicit, literal operator settings for local server commands."""
from pathlib import Path
import re

EXAMPLE = Path(__file__).resolve().parents[1] / ".env.example"


def load_operator_env(paths):
    """Later files override earlier files. Never evaluate shell syntax or print values."""
    allowed = set(re.findall(r"^([A-Z][A-Z0-9_]*)=", EXAMPLE.read_text(encoding="utf-8"), re.M))
    merged = {}
    for index, filename in enumerate(paths, 1):
        try:
            lines = Path(filename).read_text(encoding="utf-8-sig").splitlines()
        except (OSError, UnicodeError):
            raise ValueError(f"Cannot read operator env file #{index}; check path and UTF-8 encoding.") from None
        values = {}
        for number, line in enumerate(lines, 1):
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            key, separator, value = line.partition("=")
            key, value = key.strip(), value.strip()
            if not separator or key not in allowed or key in values or "\0" in line:
                raise ValueError(f"Invalid or duplicate setting in operator env file #{index}, line {number}.")
            if value.startswith(("'", '"')):
                if len(value) < 2 or value[-1] != value[0]:
                    raise ValueError(f"Unclosed quote in operator env file #{index}, line {number}.")
                value = value[1:-1]
            values[key] = value
        merged.update(values)
    return merged
