#!/usr/bin/env python3
"""Linux-only capture child supervisor; completion means the subtree was reaped."""
import ctypes
import json
import os
import pathlib
import selectors
import signal
import subprocess
import sys

LIBC = ctypes.CDLL(None, use_errno=True)


def checked(call, *args):
    result = call(*args)
    if result < 0:
        number = ctypes.get_errno()
        raise OSError(number, os.strerror(number))
    return result


def event(name, **fields):
    try:
        print(json.dumps(dict(event=name, **fields)), flush=True)
    except BrokenPipeError:
        pass  # A dead renderer must not abandon its browser descendants.


def signal_children(number):
    # Only our direct children, including subreaper adoptions. No process-name
    # lookup or global scan. This single-threaded supervisor does not reap while
    # opening pidfds: even an exited child retains its PID until waitpid below.
    children = pathlib.Path(f"/proc/self/task/{os.getpid()}/children").read_text().split()
    for value in children:
        try:
            descriptor = checked(LIBC.pidfd_open, int(value), 0)
        except ProcessLookupError:
            continue
        try:
            checked(LIBC.pidfd_send_signal, descriptor, number, None, 0)
        except ProcessLookupError:
            pass
        finally:
            os.close(descriptor)


def run(command):
    if not sys.platform.startswith("linux") or os.getpgrp() != os.getpid():
        raise RuntimeError("browser job requires an owned Linux session leader")
    LIBC.prctl.argtypes = [ctypes.c_int, ctypes.c_ulong, ctypes.c_ulong, ctypes.c_ulong, ctypes.c_ulong]
    checked(LIBC.prctl, 36, 1, 0, 0, 0)  # PR_SET_CHILD_SUBREAPER
    if not hasattr(LIBC, "pidfd_open") or not hasattr(LIBC, "pidfd_send_signal"):
        raise RuntimeError("browser job requires Linux pidfd signaling")
    # This host's Python omits os.pidfd_open/signal.pidfd_send_signal; use the
    # same Linux operations through libc with explicit, checked signatures.
    LIBC.pidfd_open.argtypes = [ctypes.c_int, ctypes.c_uint]
    LIBC.pidfd_open.restype = ctypes.c_int
    LIBC.pidfd_send_signal.argtypes = [ctypes.c_int, ctypes.c_int, ctypes.c_void_p, ctypes.c_uint]
    LIBC.pidfd_send_signal.restype = ctypes.c_int
    reader, writer = os.pipe2(os.O_NONBLOCK | os.O_CLOEXEC)
    # SIGCHLD and control-pipe readiness drive all checks. No timer/proc polling.
    signal.signal(signal.SIGCHLD, lambda *_: None)
    # Caught handlers reset on exec; browser children retain normal dispositions.
    signal.signal(signal.SIGTERM, lambda *_: None)
    signal.signal(signal.SIGINT, lambda *_: None)
    signal.set_wakeup_fd(writer)
    selector = selectors.DefaultSelector()
    selector.register(reader, selectors.EVENT_READ, "signal")
    selector.register(sys.stdin.fileno(), selectors.EVENT_READ, "control")
    os.set_blocking(sys.stdin.fileno(), False)
    child = subprocess.Popen(command, stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL)
    event("started", pid=child.pid)
    mode, buffered = None, b""
    try:
        while True:
            while True:
                try:
                    pid, status = os.waitpid(-1, os.WNOHANG)
                except ChildProcessError:
                    event("drained")  # ECHILD, not merely command-leader exit.
                    return 0
                if not pid:
                    break
                code = os.waitstatus_to_exitcode(status)
                event("reaped", pid=pid, returncode=code)
                if pid == child.pid:
                    child.returncode = code
                    event("leader-exit", returncode=code)
            if mode:
                # Killing a parent can adopt more children. The next SIGCHLD
                # wakes us to reap and signal those exact new children too,
                # including descendants which created their own sessions.
                signal_children(signal.SIGKILL if mode == "kill" else signal.SIGTERM)
            for key, _ in selector.select():
                if key.data == "signal":
                    received = os.read(reader, 65536)
                    if signal.SIGINT in received:
                        mode = "kill"
                else:
                    data = os.read(sys.stdin.fileno(), 4096)
                    if not data:
                        selector.unregister(sys.stdin.fileno())
                        mode = "kill"  # Renderer died or closed its control FD.
                    else:
                        buffered += data
                        while b"\n" in buffered:
                            line, buffered = buffered.split(b"\n", 1)
                            if line not in (b"terminate", b"kill"):
                                raise RuntimeError("invalid browser job control")
                            if mode != "kill":
                                mode = line.decode()
                            if mode == "terminate":
                                # This live supervisor is the retained group
                                # identity anchor and ignores its own SIGTERM.
                                # No group signal is ever sent after it exits.
                                os.killpg(os.getpgrp(), signal.SIGTERM)
    finally:
        selector.close()
        signal.set_wakeup_fd(-1)
        os.close(reader)
        os.close(writer)


def main():
    try:
        return run(json.loads(sys.argv[1]))
    except Exception as error:
        event("error", message=str(error))
        # Fail closed, keeping the same subreaper alive to dispose/reap its own
        # children. PID identities remain retained until each waitpid.
        while True:
            signal_children(signal.SIGKILL)
            try:
                os.waitpid(-1, 0)
            except ChildProcessError:
                return 1


if __name__ == "__main__":
    sys.exit(main())
