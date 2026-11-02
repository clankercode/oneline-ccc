#!/usr/bin/env python3
"""Verify the complete local module mirrors against pinned public Go archives."""

import argparse
import base64
import difflib
import hashlib
import json
from pathlib import Path
import zipfile


def module_hash(files):
    lines = "".join(
        hashlib.sha256(content).hexdigest() + "  " + name + "\n"
        for name, content in sorted(files.items())
    )
    return "h1:" + base64.b64encode(hashlib.sha256(lines.encode()).digest()).decode()


def correction(root, cache):
    patches = []
    manifest = json.loads((root / "upstream.json").read_text())
    go_mod = (root.parent / "go.mod").read_text()
    for entry in manifest:
        name, version, directory = (entry[key] for key in ("module", "version", "directory"))
        assert f"\t{name} {version}" in go_mod, f"changed requirement for {name}"
        assert f"replace {name} => ./third_party/{directory}" in go_mod
        archive = cache / name / "@v" / (version + ".zip")
        assert hashlib.sha256(archive.read_bytes()).hexdigest() == entry["archive_sha256"], archive
        with zipfile.ZipFile(archive) as source:
            files = {item.filename: source.read(item) for item in source.infolist()}
        assert module_hash(files) == entry["module_sum"], f"module checksum mismatch: {name}"
        prefix = name + "@" + version + "/"
        assert all(path.startswith(prefix) for path in files)
        originals = {path[len(prefix):]: content for path, content in files.items()}
        assert module_hash({"go.mod": originals["go.mod"]}) == entry["go_mod_sum"]
        local = root / directory
        assert not any(path.is_symlink() for path in local.rglob("*")), local
        current = {path.relative_to(local).as_posix(): path.read_bytes()
                   for path in local.rglob("*") if path.is_file()}
        for path in sorted(originals.keys() | current.keys()):
            old, new = originals.get(path, b""), current.get(path, b"")
            if old == new:
                continue
            patches.extend(difflib.unified_diff(
                old.decode().splitlines(keepends=True), new.decode().splitlines(keepends=True),
                fromfile=f"a/{directory}/{path}" if path in originals else "/dev/null",
                tofile=f"b/{directory}/{path}" if path in current else "/dev/null",
            ))
        print(f"{name}@{version}: archive SHA-256, module h1, go.mod h1 verified; "
              f"{len(originals)} upstream files compared")
    return "".join(patches)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("module_download_cache", type=Path,
                        help="Go module download cache containing the two pinned .zip archives")
    parser.add_argument("--write-patch", action="store_true",
                        help="regenerate the correction for review after intentional source edits")
    args = parser.parse_args()
    root = Path(__file__).resolve().parent
    patch = correction(root, args.module_download_cache)
    destination = root / "terminal-input.patch"
    if args.write_patch:
        destination.write_text(patch)
        print("Wrote terminal-input.patch; review this diff before committing.")
    else:
        assert patch == destination.read_text(), "local sources differ from terminal-input.patch"
        print("Complete mirrors match the pinned upstream archives plus exactly terminal-input.patch.")


if __name__ == "__main__":
    main()
