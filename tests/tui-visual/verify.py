#!/usr/bin/env python3
"""Check generated artifacts, hashes and terminal geometry; never visual PASS."""
import collections
import hashlib
import json
import pathlib
import struct
import subprocess
import sys

from capture import CaptureError, check_private, HERE


def verify(path):
    path = pathlib.Path(path).resolve()
    manifest = json.loads(path.read_text())
    if manifest["kind"] != "native-pty":
        raise CaptureError("not a native PTY manifest")
    counts = collections.Counter(row["status"] for row in manifest["captures"])
    for row in manifest["captures"]:
        if row["status"] != "rendered":
            raise CaptureError(f"unrendered row: {row['id']} ({row['status']})")
        files = {}
        for kind in ["ansi", "cleanup", "transcript", "frame", "png"]:
            record = row["files"][kind]
            artifact = (path.parent / record["path"]).resolve()
            if not artifact.is_relative_to(path.parent):
                raise CaptureError("artifact outside manifest directory")
            data = artifact.read_bytes()
            if hashlib.sha256(data).hexdigest() != record["sha256"] or len(data) != record["bytes"]:
                raise CaptureError(f"artifact hash/size mismatch: {artifact}")
            files[kind] = data
        for kind in ["ansi", "cleanup", "transcript", "frame"]:
            check_private(files[kind])
        if files["transcript"] != files["ansi"] + files["cleanup"]:
            raise CaptureError("transcript does not preserve frame and cleanup")
        if b"\x1b[?1049h" not in files["ansi"] or b"\x1b[?1049l" not in files["cleanup"]:
            raise CaptureError("alternate-screen lifecycle missing")
        keys = row["fixture"]["pty_keys"]
        if row.get("validation_schema", 1) >= 3:
            plan = row["fixture"].get("protocol_keys")
            if keys != plan or keys not in [["tab", "shift+tab"], ["shift+tab", "tab"]]:
                raise CaptureError("PTY key plan/delivery mismatch")
            marker = f"\x1b]777;native-capture:keys={','.join(keys)}\x07".encode()
            if marker not in files["ansi"]:
                raise CaptureError("PTY key plan announcement missing")
        allowed_keys = [["tab", "shift+tab"]]
        if row.get("validation_schema", 1) >= 3:
            allowed_keys.append(["shift+tab", "tab"])
        if not row["termios_restored"] or keys not in allowed_keys:
            raise CaptureError("PTY input/cleanup proof missing")
        frame = json.loads(files["frame"])
        cols, rows = row["cols"], row["rows"]
        if frame["cols"] != cols or frame["rows"] != rows or len(frame["cells"]) != rows or any(len(line) != cols for line in frame["cells"]):
            raise CaptureError("cell frame does not match requested geometry")
        if not any("oneline-ccc" in line for line in frame["text"]):
            raise CaptureError("empty/non-native cell frame")
        if not files["png"].startswith(b"\x89PNG\r\n\x1a\n"):
            raise CaptureError("not a PNG")
        width, height = struct.unpack(">II", files["png"][16:24])
        metrics = row["terminal"]
        if (width, height) != (metrics["width_px"], metrics["height_px"]) or width != cols * metrics["cell_width_px"] or height != rows * metrics["cell_height_px"]:
            raise CaptureError("PNG geometry does not match measured terminal cells")
    checked = subprocess.run(["node", str(HERE / "evidence.mjs"), str(path)], capture_output=True, text=True)
    if checked.returncode:
        raise CaptureError(checked.stderr.strip() or "terminal evidence validation failed")
    return {"manifest": str(path), "counts": dict(counts), "content_checks": json.loads(checked.stdout),
            "artifact_checks": "hashes, geometry, PTY lifecycle, fixed-fixture privacy, ANSI/cell/text/scene consistency and PNG glyph sanity", "visual_acceptance": "OPEN"}


if __name__ == "__main__":
    try:
        for manifest in sys.argv[1:]:
            print(json.dumps(verify(manifest)))
        if len(sys.argv) == 1:
            raise CaptureError("usage: verify.py MANIFEST [MANIFEST...]")
    except (CaptureError, OSError, KeyError, ValueError) as error:
        print(f"verify: {error}", file=sys.stderr)
        sys.exit(1)
