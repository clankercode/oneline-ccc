"""Run the compiled Codex adapter checks with no network or operator home.

Build with CGO_ENABLED=0 go test -c ./internal/adapters/codex, then pass the
test executable, the pinned native Codex executable, and a disposable root.
The optional fourth argument is a catalog from go run ./tests/native-catalog.
"""
from pathlib import Path
import json
import subprocess
import sys

if len(sys.argv) not in (4, 5):
    raise SystemExit("usage: native-codex-sandbox.py TEST_BINARY CODEX_BINARY SCRATCH [CATALOG]")
test_binary, codex_binary, scratch = map(lambda p: Path(p).resolve(), sys.argv[1:4])
scratch.mkdir(mode=0o700, parents=True, exist_ok=True)
for name in ("tmp", "user"):
    (scratch / name).mkdir(mode=0o700, exist_ok=True)
cmd = ["bwrap", "--die-with-parent", "--unshare-all", "--cap-drop", "ALL",
       "--ro-bind", "/usr", "/usr", "--symlink", "usr/bin", "/bin",
       "--symlink", "usr/lib", "/lib", "--symlink", "usr/lib", "/lib64"]
for path in ("/etc", "/home", "/root", "/run", "/eval"):
    cmd += ["--dir", path]
cmd += ["--ro-bind", str(test_binary), "/eval/test",
        "--ro-bind", str(codex_binary), "/eval/codex",
        "--bind", str(scratch), "/trial", "--proc", "/proc", "--dev", "/dev",
        "--chdir", "/trial", "--clearenv"]
environment = {"PATH": "/usr/bin:/bin", "HOME": "/trial/user",
               "TMPDIR": "/trial/tmp", "ONELINE_CODEX_TEST_BINARY": "/eval/codex",
               "LANG": "C.UTF-8"}
if len(sys.argv) == 5:
    catalog = Path(sys.argv[4]).resolve()
    cmd += ["--ro-bind", str(catalog), "/eval/catalog.json"]
    environment["ONELINE_CODEX_TEST_CATALOG"] = "/eval/catalog.json"
for key, value in environment.items():
    cmd += ["--setenv", key, value]
cmd += ["/eval/test", "-test.v"]
(scratch / "command.json").write_text(json.dumps(cmd, indent=2))
result = subprocess.run(cmd, capture_output=True, text=True, timeout=180)
(scratch / "result.log").write_text(result.stdout + result.stderr)
print(result.stdout + result.stderr, end="")
raise SystemExit(result.returncode)
