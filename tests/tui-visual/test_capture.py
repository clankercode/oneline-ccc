"""Failure tests use real disposable processes and PTYs, never the app backend."""
import importlib.util
import pathlib
import sys
import unittest

spec = importlib.util.spec_from_file_location("capture", pathlib.Path(__file__).with_name("capture.py"))
capture = importlib.util.module_from_spec(spec)
spec.loader.exec_module(capture)


class CaptureFailures(unittest.TestCase):
    def test_real_key_roundtrip_retains_split_utf8_and_separate_cleanup(self):
        peer = pathlib.Path(__file__).with_name("protocol_peer.py")
        result = capture.capture_pty([sys.executable, str(peer)], {}, 80, 24, 2)
        self.assertIn("模型".encode(), result["ansi"])
        self.assertNotIn(b"\x1b[?1049l", result["ansi"])
        self.assertIn(b"\x1b[?1049l", result["cleanup"])
        self.assertTrue(result["termios_restored"])

    def test_actual_reverse_pair_is_delivered_for_clamped_last_boundary(self):
        peer = pathlib.Path(__file__).with_name("protocol_peer.py")
        try:
            result = capture.capture_pty([sys.executable,str(peer),"reverse-keys"],{},80,24,2)
        except capture.CaptureError as error:
            self.fail(f"actual reverse key pair was not delivered: {error}")
        self.assertTrue(result["termios_restored"])
        self.assertIn(b"native-capture:keys=shift+tab,tab",result["ansi"])

    def test_successful_process_without_terminal_cleanup_is_rejected(self):
        peer = pathlib.Path(__file__).with_name("protocol_peer.py")
        with self.assertRaisesRegex(capture.CaptureError, "exit cleanup"):
            capture.capture_pty([sys.executable, str(peer), "missing-cleanup"], {}, 80, 24, 2)

    def test_missing_capture_tool(self):
        with self.assertRaisesRegex(capture.CaptureError, "missing capture executable"):
            capture.capture_pty(["/does-not-exist/native-capture"], {}, 80, 24, 1)

    def test_process_failure_is_not_a_frame(self):
        with self.assertRaisesRegex(capture.CaptureError, "exited.*7"):
            capture.capture_pty([sys.executable, "-c", "raise SystemExit(7)"], {}, 80, 24, 1)

    def test_empty_capture_rejected(self):
        with self.assertRaisesRegex(capture.CaptureError, "empty capture"):
            capture.capture_pty([sys.executable, "-c", "pass"], {}, 80, 24, 1)

    def test_no_readiness_marker_rejected(self):
        with self.assertRaisesRegex(capture.CaptureError, "readiness"):
            capture.capture_pty([sys.executable, "-c", "print('not a terminal frame')"], {}, 80, 24, 1)

    def test_unresponsive_capture_is_bounded(self):
        with self.assertRaisesRegex(capture.CaptureError, "deadline"):
            capture.capture_pty([sys.executable, "-c", "import signal; signal.pause()"], {}, 80, 24, .1)

    def test_endpoint_and_dummy_secret_fail_closed(self):
        for value in [capture.DUMMY_SECRET, "https://capture.invalid/v1", "capture.invalid"]:
            with self.subTest(value=value):
                with self.assertRaisesRegex(capture.CaptureError, "private fixture"):
                    capture.check_private(value.encode())


if __name__ == "__main__":
    unittest.main()
