"""Find the npm-installed native fixture without invoking its launcher."""
from pathlib import Path
import os
import sys

root = Path(sys.argv[1]).resolve()
name = "codex.exe" if os.name == "nt" else "codex"
matches = [p for p in root.rglob(name) if p.parent.name in ("bin", "codex")
           and p.is_file() and "vendor" in p.parts]
if len(matches) != 1:
    raise SystemExit("Expected exactly one platform-native Codex fixture")
path = str(matches[0])
with open(os.environ["GITHUB_ENV"], "a", encoding="utf-8") as env:
    env.write("ONELINE_CODEX_TEST_BINARY=" + path + "\n")
print("Selected the isolated native Codex fixture")
