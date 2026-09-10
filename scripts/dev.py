"""Small local runner. Infrastructure uses Docker; Go/Next run on the host."""
from pathlib import Path
import argparse
import base64
import os
import secrets
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
LOCAL = ROOT / ".local"

def run(args, cwd=ROOT, env=None):
    subprocess.run(args, cwd=cwd, env=env, check=True)

def prepare():
    LOCAL.mkdir(exist_ok=True)
    marker = LOCAL / "postgres.env"
    if marker.exists():
        for name in ("database-url", "migration-url"):
            if not (LOCAL / name).is_file():
                raise SystemExit("Incomplete local credentials; recover the original files before continuing.")
        return
    passwords = [secrets.token_hex(24) for _ in range(3)]
    marker.write_text(
        "POSTGRES_USER=postgres\nPOSTGRES_DB=waba\n"
        f"POSTGRES_PASSWORD={passwords[0]}\nMIGRATION_PASSWORD={passwords[1]}\nRUNTIME_PASSWORD={passwords[2]}\n",
        encoding="utf-8",
    )
    for filename, role, password in (("database-url", "waba_runtime", passwords[2]), ("migration-url", "waba_migrator", passwords[1])):
        path = LOCAL / filename
        path.write_text(f"postgres://{role}:{password}@127.0.0.1:55432/waba?sslmode=disable", encoding="utf-8")
        path.chmod(0o600)
    marker.chmod(0o600)

def prepare_identity():
    marker = LOCAL / "postgres.env"
    settings = dict(line.split("=", 1) for line in marker.read_text().splitlines())
    new_identity = "IDENTITY_PASSWORD" not in settings
    if new_identity:
        password = secrets.token_hex(24)
        with marker.open("a", encoding="utf-8") as f:
            f.write("IDENTITY_PASSWORD=" + password + "\n")
        path = LOCAL / "identity-url"
        path.write_text(f"postgres://waba_identity:{password}@127.0.0.1:55432/waba?sslmode=disable", encoding="utf-8")
        path.chmod(0o600)
    if not (LOCAL / "identity-url").exists():
        raise SystemExit("Identity URL file missing; recover protected local configuration.")
    root = LOCAL / "identity-root-key"
    if not root.exists():
        if not new_identity:
            raise SystemExit("Identity root key missing; restore the original protected key. Never regenerate it for existing data.")
        root.write_text(base64.b64encode(secrets.token_bytes(32)).decode(), encoding="utf-8")
        root.chmod(0o600)

def backend_env(purpose="runtime"):
    prepare()
    prepare_identity()
    env = os.environ.copy()
    for key in list(env):
        if key.startswith("TEST_") or key.startswith("MIGRATION_DATABASE_URL") or key == "DATABASE_URL" or (key.startswith("BOOTSTRAP_") and purpose != "bootstrap"):
            env.pop(key)
    env.update(APP_ENV="development", DATABASE_URL_FILE=str(LOCAL / "database-url"), GOTOOLCHAIN="go1.27.1")
    env["IDENTITY_DATABASE_URL_FILE"] = str(LOCAL / "identity-url")
    env["IDENTITY_ROOT_KEY_FILE"] = str(LOCAL / "identity-root-key")
    if purpose == "migration":
        env["MIGRATION_DATABASE_URL_FILE"] = str(LOCAL / "migration-url")
    elif purpose == "test":
        settings = dict(line.split("=", 1) for line in (LOCAL / "postgres.env").read_text(encoding="utf-8").splitlines())
        env.update(TEST_ADMIN_DATABASE_URL=f"postgres://postgres:{settings['POSTGRES_PASSWORD']}@127.0.0.1:55432/waba?sslmode=disable",
                   TEST_MIGRATION_DATABASE_URL=(LOCAL / "migration-url").read_text(encoding="utf-8"),
                   TEST_RUNTIME_DATABASE_URL=(LOCAL / "database-url").read_text(encoding="utf-8"),
                   TEST_IDENTITY_DATABASE_URL=(LOCAL / "identity-url").read_text(encoding="utf-8"))
    return env

def npm(args):
    exe = "npm.cmd" if os.name == "nt" else "npm"
    if shutil.which("fnm"):
        run(["fnm", "exec", "--using=24.21.0", exe, *args])
    else:
        version = subprocess.check_output(["node", "--version"], text=True).strip()
        if version != "v24.21.0":
            raise SystemExit("Install Node 24.21.0 (.node-version).")
        run([exe, *args])

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=["up", "down", "migrate", "backend", "frontend", "test", "check", "e2e", "bootstrap", "mailworker"])
    action = parser.parse_args().action
    if action == "up":
        prepare()
        prepare_identity()
        run(["docker", "compose", "up", "-d", "--wait", "postgres"])
        run(["docker", "compose", "exec", "-T", "postgres", "sh", "/docker-entrypoint-initdb.d/20-identity.sh"])
    elif action == "down":
        run(["docker", "compose", "stop", "postgres"])  # Never deletes volumes.
    elif action == "frontend":
        npm(["run", "dev", "--workspace", "frontend"])
    else:
        env = backend_env("test" if action in {"test", "check", "e2e"} else "migration" if action == "migrate" else "bootstrap" if action == "bootstrap" else "runtime")
        if action == "migrate":
            run(["go", "run", "./cmd/migrate"], ROOT / "backend", env)
        elif action == "backend":
            run(["go", "run", "./cmd/api"], ROOT / "backend", env)
        elif action in {"bootstrap","mailworker"}:
            run(["go","run","./cmd/"+action], ROOT / "backend", env)
        elif action == "e2e":
            run(["go", "test", *(["-race"] if env.get("WABA_TEST_RACE") == "1" else []), "-tags=integration,e2e", "-count=1", "-timeout=5m", "-v", "-run", "TestBrowserIdentityE2E", "./internal/identity"], ROOT / "backend", env)
        elif action == "test":
            run(["go", "test", *(["-race"] if env.get("WABA_TEST_RACE") == "1" else []), "./..."], ROOT / "backend", env)
            run(["go", "test", *(["-race"] if env.get("WABA_TEST_RACE") == "1" else []), "-tags=integration", "-count=1", "-v", "./internal/..."], ROOT / "backend", env)
        elif action == "check":
            run([sys.executable, "scripts/check.py"])
            formatting = subprocess.check_output(["gofmt", "-l", "backend"], cwd=ROOT, text=True)
            if formatting.strip():
                raise SystemExit("Run gofmt on: " + formatting)
            run(["go", "vet", "./..."], ROOT / "backend", env)
            run(["go", "test", *(["-race"] if env.get("WABA_TEST_RACE") == "1" else []), "./..."], ROOT / "backend", env)
            run(["go", "test", *(["-race"] if env.get("WABA_TEST_RACE") == "1" else []), "-tags=integration", "-count=1", "-v", "./internal/..."], ROOT / "backend", env)
            run(["go", "build", "./..."], ROOT / "backend", env)
            npm(["ci"])
            for script in ("lint", "typecheck", "test", "contracts", "build"):
                npm(["run", script])

if __name__ == "__main__":
    main()
