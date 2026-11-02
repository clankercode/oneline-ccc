#!/usr/bin/env python3
"""Capture the real native renderer in a disposable PTY; no live app backend."""
import argparse
import errno
import fcntl
import hashlib
import json
import os
import pathlib
import pty
import re
import select
import shlex
import shutil
import signal
import struct
import subprocess
import sys
import termios
import time

ROOT = pathlib.Path(__file__).resolve().parents[2]
HERE = pathlib.Path(__file__).resolve().parent
CACHE = ROOT / ".cache/tui-visual"
TOOLS = CACHE / "tools"
BINARY = CACHE / "native-capture.test"
DUMMY_SECRET = "sk-capture-DUMMY-secret-never-publish"
SIZES = ["120x40", "100x30", "80x24", "60x20", "40x12"]
THEMES = ["dark", "light", "mono"]
SCREENS = ["dashboard", "models", "review", "jobs", "gateway", "service", "repair", "ssh", "utilities", "settings"]
MODALS = ["help", "palette", "discard", "quit", "ssh-confirm", "monitor-install", "monitor-uninstall"]
EDITORS = ["edit-filter", "edit-model", "edit-budget", "edit-compaction", "edit-endpoint", "edit-credential", "edit-key"]
SCENARIOS = SCREENS + MODALS + EDITORS
EXTENSIONS = {"more": {"semantic_id": "nav:More", "availability": "production nav:More required; observed at integration base cb936e6",
                       "modal_inventory_after_integration": 8, "variants": ["baseline", "last-item"]}}
VARIANTS = {"dashboard": ["no-results", "pending-focus", "filter-match", "filter-unicode"], "palette": ["no-results"], "help": ["last-page"],
            "review": ["last-page", "acknowledged"], "edit-budget": ["invalid"],
            "edit-compaction": ["invalid"], "edit-endpoint": ["invalid"], "more": ["last-item"]}
GAPS = ["loading and asynchronous progress/cancellation", "discovery loading/error/empty",
        "plan no-op/stale/conflict/failure/apply receipt journeys", "service running/stopped/unknown/conflict/unavailable",
        "SSH/child handoff and terminal return", "key save/persistent/error and draft focus-return journeys",
        "long multiline repair briefs, editor caret/paste/save/cancel permutations",
        "Windows and macOS terminal evidence", "independent native/reference visual and geometry acceptance",
        "More first/last evidence is bounded to explicitly captured sizes/themes; complete modal matrix remains open",
        "Help intermediate pages and complete compact pagination/geometry matrix remain unverified",
        "Dashboard prepared fixture setup keys are not PTY journey evidence; A3b compact draft/filter visual acceptance remains open"]


class CaptureError(RuntimeError):
    pass


def check_private(data):
    # Endpoints and credentials are injected fixtures, never host configuration.
    if any(value in data for value in [DUMMY_SECRET.encode(), b"capture.invalid", b"https://", b"http://"]):
        raise CaptureError("private fixture value or endpoint found; artifacts refused")


def capture_pty(command, env, cols, rows, timeout=10):
    """Wait for completed renderer writes, round-trip real keys, then cleanly exit.

    The OSC barrier comes *after* the renderer's output Write; no sleep-based
    readiness or text-layout guesses. Keep the active alt-screen stream separate
    from exit cleanup (replaying the latter would erase the evidence).
    """
    master, slave = pty.openpty()
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))
    original = termios.tcgetattr(slave)
    raw = bytearray()
    deadline = time.monotonic() + timeout
    proc = None
    try:
        try:
            proc = subprocess.Popen(command, stdin=slave, stdout=slave, stderr=slave,
                                    env=env, cwd=ROOT, start_new_session=True)
        except FileNotFoundError as error:
            raise CaptureError("missing capture executable; run bootstrap and build") from error
        # Keep the slave open for termios verification after the child exits.
        def pump():
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                raise CaptureError("capture readiness/exit deadline exceeded")
            if not select.select([master], [], [], min(remaining, .1))[0]:
                return
            try:
                data = os.read(master, 65536)
            except OSError as error:
                if error.errno == errno.EIO:
                    return
                raise
            start = len(raw)
            raw.extend(data)
            # Answer only well-known terminal capability queries, never network.
            # Search the boundary too, since UTF-8 and control sequences can split.
            window = bytes(raw[max(0, start - 64):])
            responses = {b"\x1b[6n": b"\x1b[1;1R", b"\x1b[c": b"\x1b[?1;2c",
                         b"\x1b]10;?\x07": b"\x1b]10;rgb:e6e6/eaea/f0f0\x07",
                         b"\x1b]11;?\x07": b"\x1b]11;rgb:1111/1818/2020\x07"}
            for query, response in responses.items():
                for match in re.finditer(re.escape(query), window):
                    if max(0, start - 64) + match.end() > start:
                        os.write(master, response)

        def ended():
            code = proc.poll()
            if code is None:
                return False
            if code:
                raise CaptureError(f"capture process exited with status {code}; no artifacts published")
            if not raw:
                raise CaptureError("empty capture")
            raise CaptureError("capture process exited before renderer readiness")

        keys = ["tab", "shift+tab"]  # compatible with the disposable protocol peer
        key_bytes = {"tab": b"\t", "shift+tab": b"\x1b[Z"}
        for epoch in range(3):
            if epoch:
                os.write(master, key_bytes[keys[epoch-1]])
            marker = f"\x1b]777;native-capture:{epoch}\x07".encode()
            while marker not in raw:
                pump()
                ended()
            if epoch == 0:
                plan = re.search(rb"\x1b]777;native-capture:keys=([^\x07]*)\x07", raw)
                if plan:
                    keys = plan[1].decode("ascii").split(",")
                    if keys not in [["tab", "shift+tab"], ["shift+tab", "tab"]]:
                        raise CaptureError("invalid capture key protocol")
        marker_end = raw.index(marker) + len(marker)
        frame = bytes(raw[:marker_end])
        if b"\x1b[?1049h" not in frame or b"oneline-ccc" not in frame:
            raise CaptureError("empty/non-native alt-screen capture")
        # F12 is a test-only wrapper exit and exercises normal Bubble Tea cleanup.
        os.write(master, b"\x1b[24~")
        while proc.poll() is None:
            pump()
        while select.select([master], [], [], 0)[0]:
            pump()
        if proc.returncode:
            raise CaptureError(f"capture process exited with status {proc.returncode}; no artifacts published")
        cleanup = bytes(raw[marker_end:])
        if b"\x1b[?1049l" not in cleanup or b"\x1b[?25h" not in cleanup:
            raise CaptureError("terminal exit cleanup missing alt-screen leave/cursor restore")
        restored = termios.tcgetattr(slave) == original
        if not restored:
            raise CaptureError("terminal attributes were not restored")
        check_private(bytes(raw))
        return {"ansi": frame, "cleanup": cleanup, "transcript": bytes(raw), "termios_restored": restored}
    finally:
        try:
            if proc is not None:
                # Ownership is of the session/process group, not just its leader.
                # A failed, already-reaped leader may still have live children.
                try:
                    os.killpg(proc.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
                finally:
                    proc.wait()
        finally:
            try:
                os.close(master)
            finally:
                os.close(slave)


def run(command, **kwargs):
    try:
        subprocess.run(command, cwd=ROOT, check=True, **kwargs)
    except FileNotFoundError as error:
        raise CaptureError(f"missing tool: {command[0]}") from error
    except subprocess.CalledProcessError as error:
        raise CaptureError(f"tool failed ({error.returncode}): {shlex.join(map(str, command))}") from error


def build():
    scratch = CACHE / "tmp"
    scratch.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ, GOMAXPROCS="2", GOTMPDIR=str(scratch), TMPDIR=str(scratch))
    run(["go", "test", "-p", "2", "-tags=visualcapture", "-c", "-o", str(BINARY), "./internal/tui"], env=env)


def bootstrap():
    TOOLS.mkdir(parents=True, exist_ok=True)
    (CACHE / "tmp").mkdir(exist_ok=True)
    for name in ["package.json", "package-lock.json"]:
        shutil.copyfile(HERE / name, TOOLS / name)
    env = dict(os.environ, npm_config_cache=str(CACHE / "npm"), TMPDIR=str(CACHE / "tmp"),
               PLAYWRIGHT_BROWSERS_PATH=str(CACHE / "browsers"))
    run(["npm", "ci", "--prefix", str(TOOLS), "--ignore-scripts", "--no-audit", "--no-fund"], env=env)
    run(["node", str(TOOLS / "node_modules/playwright/cli.js"), "install", "chromium"], env=env)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def source():
    def git(*args):
        return subprocess.check_output(["git", *args], cwd=ROOT).decode().strip()
    excluded = [":(exclude)artifacts/**", ":(exclude).cache/**", ":(exclude).worktrees/**", ":(exclude).superpowers/**"]
    tracked = set(git("ls-files", "-z", "--", ".", *excluded).split('\0')) - {""}
    # Include ignored Go/fixture inputs too: gitignore does not prevent Go from
    # compiling a new .go file. Exclude generated roots, not untracked source.
    untracked = set(git("ls-files", "--others", "-z", "--", ".", *excluded).split('\0')) - {""}
    def role(path):
        if path.startswith("tests/tui-visual/") and not any(part in path.split('/') for part in ["__pycache__", "node_modules"]):
            return "capture_tool"
        if path.endswith('_test.go') or '/testdata/' in path:
            return "fixture"
        if path.endswith('.go') or path in ["go.mod", "go.sum", "go.work", "go.work.sum"]:
            return "production"
        return None
    inputs = []
    for name in sorted(tracked | untracked):
        kind = role(name)
        path = ROOT / name
        if kind and path.is_file():
            inputs.append({"path": name, "role": kind, "tracked": name in tracked, "sha256": sha(path.read_bytes())})
    scopes = {}
    for kind in ["production", "fixture", "capture_tool"]:
        files = [entry for entry in inputs if entry['role'] == kind]
        names = [entry['path'] for entry in files]
        revision = git('log', '-1', '--format=%H', '--', *names) if names else None
        scopes[kind] = {"revision": revision or None, "inputs": files,
                        "dirty": any(not entry['tracked'] for entry in files) or bool(git('status','--porcelain','--',*names)) if names else False,
                        "inputs_sha256": sha(json.dumps(files, sort_keys=True).encode())}
    worktree_status = git("status", "--porcelain", "--untracked-files=all", "--", ".", *excluded)
    untracked_inputs = [entry for entry in inputs if not entry['tracked']]
    return {"sha": git("rev-parse", "HEAD"), "branch": git("branch", "--show-current"),
            "dirty": worktree_status or ("untracked build/fixture inputs (including ignored)" if untracked_inputs else ""),
            "untracked_inputs": untracked_inputs, **scopes,
            "tracked_diff_sha256": sha(subprocess.check_output(["git", "diff", "HEAD"], cwd=ROOT)),
            "harness_files_sha256": {p.name: sha(p.read_bytes()) for p in sorted(HERE.iterdir()) if p.is_file()},
            "helper_sha256": sha((ROOT / "internal/tui/visual_capture_test.go").read_bytes()),
            "binary_sha256": sha(BINARY.read_bytes()) if BINARY.exists() else None}


def write_json(path, value):
    path.write_text(json.dumps(value, indent=2, ensure_ascii=False) + "\n")


def parse_list(value, allowed):
    result = allowed if value == "all" else value.split(",")
    if not result or any(item not in allowed for item in result):
        raise CaptureError(f"expected all or comma-separated values from {allowed}")
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=["bootstrap", "build", "list", "capture", "batch"])
    parser.add_argument("--scenario", choices=SCENARIOS + list(EXTENSIONS), default="dashboard")
    parser.add_argument("--batch", choices=["screens", "all", "integrated"], default="screens", dest="batch_set")
    parser.add_argument("--sizes", default="120x40", help="comma-separated cell sizes, or all")
    parser.add_argument("--themes", default="dark", help="dark,light,mono, or all")
    parser.add_argument("--state", choices=["baseline", "empty", "missing", "unsupported", "drift", "unicode", "parity"], default="baseline")
    parser.add_argument("--variant", default="baseline")
    parser.add_argument("--color", choices=["truecolor", "256", "16", "none"], default="truecolor")
    parser.add_argument("--ascii", action="store_true")
    parser.add_argument("--plan", action="store_true", help="write pending coverage manifest without capturing")
    parser.add_argument("--output", default="artifacts/tui-native-baseline")
    args = parser.parse_args()
    if args.mode == "bootstrap": bootstrap(); return
    if args.mode == "build": build(); return
    if args.mode == "list":
        print(json.dumps({"screens": SCREENS, "modals": MODALS, "editors": EDITORS, "variants": VARIANTS, "integration_extensions": EXTENSIONS, "uncovered_states": GAPS}, indent=2)); return
    sizes = parse_list(args.sizes, SIZES)
    themes = parse_list(args.themes, THEMES)
    scenarios = [args.scenario] if args.mode == "capture" else (SCREENS if args.batch_set == "screens" else SCENARIOS + (list(EXTENSIONS) if args.batch_set == "integrated" else []))
    for scenario in scenarios:
        if args.variant not in ["baseline", *VARIANTS.get(scenario, [])]:
            raise CaptureError(f"unsupported variant {args.variant} for {scenario}")
    output = (ROOT / args.output).resolve()
    # Generated artifacts must stay in this worktree's ignored output roots.
    if not any(output.is_relative_to(ROOT / name) for name in ["artifacts", ".cache"]):
        raise CaptureError("output must be under this worktree's artifacts/ or .cache/")
    output.mkdir(parents=True, exist_ok=True)
    manifest_path = output / "manifest.json"
    if manifest_path.exists():
        raise CaptureError("output already contains a manifest; choose a new output directory")
    if not args.plan:
        if not (TOOLS / "node_modules/playwright/index.mjs").exists():
            raise CaptureError("missing pinned browser tools; run capture.py bootstrap")
        # Always rebuild to prevent stale-binary evidence on a changed UI branch.
        build()
    manifest = {"schema": 1, "kind": "native-pty", "command": shlex.join(["python3", "tests/tui-visual/capture.py", *sys.argv[1:]]),
                "source": source(), "coverage": {"inventory": {"screens": 10, "modals": 8, "editors": 7, "legacy_batch_modals": 7},
                    "required_sizes": SIZES, "required_themes": THEMES, "uncovered_states": GAPS, "integration_extensions": EXTENSIONS,
                    "visual_acceptance": "OPEN; generated frames are not visual PASS"}, "captures": []}
    for scenario in scenarios:
        for size in sizes:
            cols, rows = map(int, size.split("x"))
            for theme in themes:
                name = f"{scenario}-{args.state}-{args.variant}-{size}-{theme}-{args.color}" + ("-ascii" if args.ascii else "")
                entry = {"id": name, "validation_schema": 3, "scenario": scenario, "state": args.state, "variant": args.variant,
                         "cols": cols, "rows": rows, "theme": theme, "color_profile": args.color, "ascii": args.ascii,
                         "status": "pending", "visual_review": "not-inspected", "reference": None}
                manifest["captures"].append(entry)
    write_json(manifest_path, manifest)
    if args.plan:
        print(f"Planned {len(manifest['captures'])} captures: {manifest_path}"); return
    scratch = CACHE / "tmp"
    for entry in manifest["captures"]:
        name = entry["id"]
        # A small allowlist excludes operator credentials/config from the child.
        env = {key: os.environ[key] for key in ["PATH", "LANG", "LC_ALL"] if key in os.environ}
        env.update({"TERM": "xterm-256color", "COLORTERM": "truecolor", "GOMAXPROCS": "2", "TMPDIR": str(scratch),
                    "ONELINE_VISUAL_CAPTURE": "1", "ONELINE_CAPTURE_COLS": str(entry["cols"]), "ONELINE_CAPTURE_ROWS": str(entry["rows"]),
                    "ONELINE_CAPTURE_SCENARIO": entry["scenario"], "ONELINE_CAPTURE_THEME": entry["theme"], "ONELINE_CAPTURE_STATE": entry["state"],
                    "ONELINE_CAPTURE_VARIANT": entry["variant"], "ONELINE_CAPTURE_COLOR": args.color, "ONELINE_CAPTURE_ASCII": "1" if args.ascii else "0",
                    "ONELINE_CAPTURE_DUMMY_KEY": DUMMY_SECRET, "ONELINE_CAPTURE_META": str(output / f".{name}.fixture.json")})
        if args.color == "none": env["NO_COLOR"] = "1"
        command = [str(BINARY), "-test.run=^TestVisualCapture$", "-test.timeout=15s"]
        metadata_file = pathlib.Path(env["ONELINE_CAPTURE_META"])
        try:
            result = capture_pty(command, env, entry["cols"], entry["rows"])
            metadata_data = metadata_file.read_bytes()
            check_private(metadata_data)
            entry["fixture"] = json.loads(metadata_data)
            entry["capture_command"] = [".cache/tui-visual/native-capture.test", "-test.run=^TestVisualCapture$", "-test.timeout=15s"]
            entry["files"] = {}
            for kind, suffix in [("ansi", "ansi"), ("cleanup", "cleanup.ansi"), ("transcript", "transcript.ansi")]:
                path = output / f"{name}.{suffix}"
                path.write_bytes(result[kind])
                entry["files"][kind] = {"path": path.name, "sha256": sha(result[kind]), "bytes": len(result[kind])}
            entry["termios_restored"] = result["termios_restored"]
            entry["status"] = "captured-ansi"
        except (CaptureError, OSError) as error:
            entry["status"] = "failed"
            entry["failure"] = str(error)
            print(f"FAIL {name}: {error}", file=sys.stderr)
        finally:
            metadata_file.unlink(missing_ok=True)
            write_json(manifest_path, manifest)
    env = dict(os.environ, TMPDIR=str(scratch), PLAYWRIGHT_BROWSERS_PATH=str(CACHE / "browsers"))
    run(["node", str(HERE / "render.mjs"), "native", str(manifest_path)], env=env)
    final = json.loads(manifest_path.read_text())
    counts = {status: sum(c["status"] == status for c in final["captures"]) for status in ["rendered", "failed", "pending"]}
    print(json.dumps({"manifest": str(manifest_path), "counts": counts}))
    if counts["failed"] or counts["pending"]:
        raise CaptureError("capture batch incomplete; inspect manifest failures")


if __name__ == "__main__":
    try:
        main()
    except CaptureError as error:
        print(f"capture: {error}", file=sys.stderr)
        sys.exit(1)
