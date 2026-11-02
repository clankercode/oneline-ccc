"""Disposable protocol peer for PTY tests. It never renders application UI."""
import os
import sys
import termios
import tty

saved = termios.tcgetattr(0)
tty.setraw(0)
try:
    def marker(epoch):
        os.write(1, f"\x1b]777;native-capture:{epoch}\x07".encode())

    def read_exact(size):
        data = b""
        while len(data) < size:
            data += os.read(0, size - len(data))
        return data

    os.write(1, b"\x1b[?1049honeline-ccc protocol test\r\n")
    # Intentionally split a multibyte glyph across writes.
    for byte in "模型".encode():
        os.write(1, bytes([byte]))
    reverse = 'reverse-keys' in sys.argv
    if reverse: os.write(1,b"\x1b]777;native-capture:keys=shift+tab,tab\x07")
    marker(0)
    assert read_exact(3 if reverse else 1) == (b"\x1b[Z" if reverse else b"\t")
    marker(1)
    assert read_exact(1 if reverse else 3) == (b"\t" if reverse else b"\x1b[Z")
    marker(2)
    assert read_exact(5) == b"\x1b[24~"
    if "missing-cleanup" not in sys.argv:
        os.write(1, b"\x1b[?1049l\x1b[?25h")
finally:
    termios.tcsetattr(0, termios.TCSANOW, saved)
