"""Corrupt real evidence, updating its hashes as an export bug would do."""
import copy
import hashlib
import importlib.util
import json
import os
import pathlib
import shutil
import signal
import struct
import subprocess
import sys
import tempfile
import time
import unittest
import zlib
from unittest import mock

from test_capture import capture

spec = importlib.util.spec_from_file_location("verify", pathlib.Path(__file__).with_name("verify.py"))
verify = importlib.util.module_from_spec(spec)
spec.loader.exec_module(verify)
SCRATCH = capture.CACHE / "tmp"


def solid_png(width, height):
    def chunk(kind, data):
        return struct.pack(">I", len(data)) + kind + data + struct.pack(">I", zlib.crc32(kind + data))
    return (b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", width, height, 8, 2, 0, 0, 0))
            + chunk(b"IDAT", zlib.compress((b"\0" + bytes.fromhex("111820") * width) * height)) + chunk(b"IEND", b""))


@unittest.skipUnless(os.getenv("A2_TEST_MANIFEST"), "run run_fault_tests.py for evidence faults")
class EvidenceFailures(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(dir=SCRATCH)
        self.addCleanup(self.temp.cleanup)
        self.folder = pathlib.Path(self.temp.name)
        original = pathlib.Path(os.environ["A2_TEST_MANIFEST"]).resolve()
        self.manifest = json.loads(original.read_text())
        self.row = self.manifest["captures"][0]
        self.manifest["captures"] = [self.row]
        for record in self.row["files"].values():
            shutil.copyfile(original.parent / record["path"], self.folder / record["path"])
        # Before fix1 the real fixture lacks View metadata. Its recorded native
        # text (not a hand-written layout) seeds the comparison for RED faults.
        frame = json.loads((self.folder / self.row["files"]["frame"]["path"]).read_text())
        self.row["fixture"].setdefault("expected_view", {"content": '\n'.join(frame['text']), "alt_screen": True})

    def replace(self, kind, data):
        record = self.row["files"][kind]
        (self.folder / record["path"]).write_bytes(data)
        record.update(sha256=hashlib.sha256(data).hexdigest(), bytes=len(data))

    def assert_rejected(self, message):
        path = self.folder / "manifest.json"
        path.write_text(json.dumps(self.manifest))
        with self.assertRaisesRegex(verify.CaptureError, message):
            verify.verify(path)

    def test_valid_solid_png_rejected_even_with_matching_hash(self):
        m = self.row["terminal"]
        self.replace("png", solid_png(m["width_px"], m["height_px"]))
        self.assert_rejected("blank|glyph")

    def test_header_only_frame_rejected_against_production_reference(self):
        frame = json.loads((self.folder / self.row["files"]["frame"]["path"]).read_text())
        for y in range(1, len(frame["cells"])):
            frame["text"][y] = ""
            for cell in frame["cells"][y]:
                cell["text"], cell["width"] = "", 1
        self.replace("frame", json.dumps(frame).encode())
        self.assert_rejected("production View|expected.*frame")

    def test_wrong_target_rejected(self):
        self.row.update(scenario="jobs", state="missing", theme="light")
        self.assert_rejected("scene|target")

    def test_cell_text_disagreement_rejected(self):
        frame = json.loads((self.folder / self.row["files"]["frame"]["path"]).read_text())
        for row in frame["cells"]:
            for cell in row:
                cell["text"] = ""
        self.replace("frame", json.dumps(frame).encode())
        self.assert_rejected("cell.*text|text.*cell")


class OwnershipFailures(unittest.TestCase):
    @unittest.skipUnless(sys.platform == "linux", "real fork cleanup probe uses Linux /proc")
    def test_leader_exit_does_not_leave_live_descendant(self):
        SCRATCH.mkdir(parents=True, exist_ok=True)
        leader = []
        original_popen = subprocess.Popen
        def remember(*args, **kwargs):
            proc = original_popen(*args, **kwargs)
            leader.append(proc.pid)
            return proc
        code = """import os,signal
r,w=os.pipe()
if os.fork()==0:
    os.close(r)
    for fd in (0,1,2): os.close(fd)
    os.write(w,b'R');os.close(w)
    signal.pause()
else:
    os.close(w);os.read(r,1);os.close(r)
    os.write(1,b'leader exits before readiness')
    os._exit(7)
"""
        def live():
            found = []
            for entry in pathlib.Path('/proc').iterdir():
                if not entry.name.isdigit(): continue
                try:
                    fields = (entry / 'stat').read_text().rsplit(')', 1)[1].split()
                    if int(fields[2]) == leader[0] and fields[0] != 'Z': found.append(entry.name)
                except (FileNotFoundError, ProcessLookupError): pass
            return found
        try:
            with mock.patch.object(capture.subprocess, "Popen", side_effect=remember):
                with self.assertRaisesRegex(capture.CaptureError, "exited.*7"):
                    capture.capture_pty([sys.executable, "-c", code], {}, 80, 24, 2)
            # SIGKILL delivery can finish just after killpg returns. This is a
            # bounded process-state wait, not a capture readiness sleep.
            deadline = time.monotonic() + 1
            while live() and time.monotonic() < deadline: time.sleep(.01)
            self.assertEqual(live(), [], "live capture descendant survived leader exit")
        finally:
            if leader:
                try: os.killpg(leader[0], signal.SIGKILL)
                except ProcessLookupError: pass

    def test_untracked_build_and_fixture_inputs_are_identified(self):
        SCRATCH.mkdir(parents=True, exist_ok=True)
        with tempfile.TemporaryDirectory(dir=SCRATCH) as temp:
            root = pathlib.Path(temp)
            def git(*args):
                return subprocess.run(["git", *args], cwd=root, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            git("init", "-q")
            (root / 'go.mod').write_text('module fixture\ngo 1.26.1\n')
            (root / '.gitignore').write_text('ignored.go\n')
            git('add', 'go.mod', '.gitignore')
            git('-c','user.name=Fixture','-c','user.email=fixture@example.invalid','commit','-qm','fixture')
            helper = root / 'internal/tui/visual_capture_test.go'
            helper.parent.mkdir(parents=True)
            helper.write_text('package tui\n')
            tool = root / 'tests/tui-visual'
            tool.mkdir(parents=True)
            fixture = tool / 'seed.json'
            fixture.write_text('{"fixture":true}')
            production = root / 'internal/extra.go'
            production.write_text('package internal\n')
            ignored = root / 'internal/ignored.go'
            ignored.write_text('package internal\n')
            with mock.patch.multiple(capture, ROOT=root, HERE=tool, BINARY=root/'.cache/missing'):
                info = capture.source()
            self.assertTrue(info['dirty'], 'untracked source must not claim a clean checkout')
            inputs = {entry['path']: entry['sha256'] for entry in info['untracked_inputs']}
            for path in [helper, fixture, production, ignored]:
                self.assertEqual(inputs[str(path.relative_to(root))], hashlib.sha256(path.read_bytes()).hexdigest())
            self.assertIn('production', info)
            self.assertIn('fixture', info)
            self.assertIn('capture_tool', info)


if __name__ == '__main__': unittest.main()
