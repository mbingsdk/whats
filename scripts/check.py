"""Check source documentation, migration integrity and obvious credential patterns."""
from pathlib import Path
from migration_checksums import validate as validate_migrations
import re
import subprocess
import sys
from urllib.parse import unquote

ROOT = Path(__file__).resolve().parents[1]
EXCLUDED = {".git", ".local", "node_modules", ".next", "__pycache__", "bin"}
TEXT = {".md", ".py", ".go", ".sql", ".json", ".yaml", ".yml", ".sh", ".mjs", ".ts", ".tsx", ".css", ".example"}
errors = []
paths = [p for p in ROOT.rglob("*") if p.is_file() and not set(p.relative_to(ROOT).parts) & EXCLUDED]
# Include tracked ignored files too: an accidentally committed .env must be inspected.
if (ROOT / ".git").exists():
    tracked = subprocess.check_output(["git", "ls-files", "-z"], cwd=ROOT).decode().split("\0")
    paths = list(set(paths) | {ROOT / p for p in tracked if p and (ROOT / p).is_file()})
links = 0
for path in paths:
    relative = path.relative_to(ROOT)
    if ".local" in relative.parts or (path.name.endswith(".env") or path.name == ".env"):
        errors.append(f"Private local/config file included in source: {relative}")
    if path.suffix not in TEXT and not path.name.startswith(".env"):
        continue
    try:
        text = path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        errors.append(f"Invalid UTF-8: {path.relative_to(ROOT)}")
        continue
    relative = path.relative_to(ROOT)
    if "\ufffd" in text:
        errors.append(f"Replacement character: {relative}")
    # Values are never printed, even on failure. High-confidence patterns only;
    # this is not a claim to recognize every possible secret.
    patterns = [
        r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----",
        r"\bAKIA[0-9A-Z]{16}\b",
        r"\bgh[pousr]_[A-Za-z0-9]{30,}\b",
        r"\bEAA[A-Za-z0-9]{50,}\b",
        r"postgres(?:ql)?://[^\s:]+:(?!synthetic(?:@|$))[^\s@]+@",
    ]
    if path.name not in {"package-lock.json", "check.py", "dev.py"}:
        if any(re.search(pattern, text) for pattern in patterns):
            errors.append(f"Possible credential: {relative}")
    if path.name.startswith(".env") and path.name != ".env.example":
        errors.append(f"Environment file included in source: {relative}")
    if path.suffix != ".md":
        continue
    if len(re.findall(r"(?m)^\s*```", text)) % 2:
        errors.append(f"Unbalanced code fence: {relative}")
    for target in re.findall(r"\[[^\]]+\]\(([^)]+)\)", text):
        target = target.strip("<>")
        if re.match(r"^[a-zA-Z][a-zA-Z0-9+.-]*:", target):
            continue
        links += 1
        filename, _, fragment = unquote(target).partition("#")
        dest = (path.parent / filename).resolve() if filename else path
        if not dest.is_file():
            errors.append(f"Broken local link: {relative} -> {target}")
        elif fragment and dest.suffix == ".md":
            headings = re.findall(r"(?m)^#{1,6}\s+(.+?)\s*$", dest.read_text(encoding="utf-8"))
            slugs = [re.sub(r"\s", "-", re.sub(r"[^\w\s-]", "", heading.lower())) for heading in headings]
            if fragment not in slugs:
                errors.append(f"Broken heading: {relative} -> {target}")
errors.extend(validate_migrations(ROOT / "database/migrations"))
# Route coverage is checked independently of the OpenAPI syntax validator.
route_source = (ROOT / "backend/internal/identity/http.go").read_text(encoding="utf-8")
implemented = {(method.lower(), "/api/v1" + path) for method, path in re.findall(r'\{"(GET|POST|PUT|PATCH|DELETE)", "([^"]+)"', route_source)}
meta_source = (ROOT / "backend/internal/meta/service.go").read_text(encoding="utf-8")
implemented |= {(m.lower(), "/api/v1"+p) for m,p in re.findall(r'\{"(GET|POST|PUT|PATCH|DELETE)", "([^"]+)"', meta_source)}
implemented |= {(m.lower(),p) for m,p in re.findall(r'HandleFunc\("(GET|POST) (/api/v1/[^"]+)"', meta_source)}
implemented |= {("get", "/healthz"), ("get", "/readyz"), ("get", "/api/v1/auth/csrf")}
documented = set()
current_path = None
for line in (ROOT / "contracts/openapi.yaml").read_text(encoding="utf-8").splitlines():
    found = re.match(r"^  (/[^:]+):$", line)
    if found:
        current_path = found.group(1)
    found = re.match(r"^    (get|post|put|patch|delete):$", line)
    if found and current_path:
        documented.add((found.group(1), current_path))
if implemented != documented:
    errors.append("OpenAPI routes differ from implemented HTTP routes.")
if errors:
    print("\n".join(errors))
    sys.exit(1)
print(f"Source checks passed: {links} local links; migration checksums; UTF-8/fences; obvious-secret scan.")
