#!/usr/bin/env python3
"""Bound cooperative verification jobs without changing their evidence or inputs."""
import argparse
import contextlib
import contextvars
import fcntl
import json
import os
import pathlib
import re
import select
import subprocess
import sys

_SLOT_FD = contextvars.ContextVar("verification_slot_fd", default=None)


class BudgetError(ValueError):
    pass


def cpu_budget():
    value = os.environ.get("GOMAXPROCS", "1")
    maximum = os.cpu_count() or 1
    if not re.fullmatch(r"[1-9][0-9]*", value) or len(value) > 6 or int(value) > maximum:
        raise BudgetError(f"GOMAXPROCS must be an integer from 1 to {maximum}")
    return value


def budget_environment(scratch):
    budget = cpu_budget()
    scratch.mkdir(parents=True, exist_ok=True)
    return dict(os.environ, GOMAXPROCS=budget, GOTMPDIR=str(scratch), TMPDIR=str(scratch))


@contextlib.contextmanager
def verification_slot(root):
    """All worktrees share one persistent inode; flock sleeps in the kernel.

    Never unlink the lock: replacing its inode would let queued jobs overlap.
    Supervised commands inherit a reference in a separate Linux subreaper.
    Closing the wrapper's reference must not unlock the shared description:
    the supervisor retains it until the complete command subtree has exited.
    """
    try:
        common = subprocess.check_output(["git", "rev-parse", "--path-format=absolute", "--git-common-dir"], cwd=root, stderr=subprocess.DEVNULL).decode().strip()
    except subprocess.CalledProcessError as error:
        raise BudgetError("repository CPU slot requires a Git worktree") from error
    lock = pathlib.Path(common).resolve() / "tui-visual-verification.lock"
    with lock.open("a") as handle:
        try:
            fcntl.flock(handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            print("verification: waiting for repository CPU slot", file=sys.stderr, flush=True)
            fcntl.flock(handle, fcntl.LOCK_EX)
        print("verification: acquired repository CPU slot", file=sys.stderr, flush=True)
        token = _SLOT_FD.set(handle.fileno())
        try:
            yield
        finally:
            _SLOT_FD.reset(token)


class _SupervisedProcess:
    """Keep job/process-group identity separate from command observation.

    poll/wait/communicate still describe the supervisor and complete subtree.
    command_poll observes the actual command leader, including early failure.
    """
    def __init__(self, process, status_fd):
        self.process = process
        self.status_fd = status_fd
        self.buffer = bytearray()
        self.command_pid = None
        self.command_returncode = None
        os.set_blocking(status_fd, False)

    def __getattr__(self, name):
        return getattr(self.process, name)

    def close(self):
        if self.status_fd is not None:
            os.close(self.status_fd)
            self.status_fd = None

    def _receive(self, block=False):
        if self.status_fd is None:
            return
        if block:
            select.select([self.status_fd], [], [])
        while True:
            try:
                data = os.read(self.status_fd, 4096)
            except BlockingIOError:
                return
            if not data:
                self.close()
                if self.command_returncode is None:
                    raise BudgetError("verification supervisor ended without command exit status")
                return
            self.buffer.extend(data)
            while b"\n" in self.buffer:
                line, _, rest = self.buffer.partition(b"\n")
                self.buffer = bytearray(rest)
                event = json.loads(line)
                if event["event"] == "started":
                    self.command_pid = event["pid"]
                elif event["event"] == "exit":
                    self.command_returncode = event["returncode"]
                elif event["event"] == "exec-error":
                    raise OSError(event["errno"], event["message"], event["filename"])
                else:
                    raise BudgetError(event["message"])

    def command_poll(self):
        self._receive()
        return self.command_returncode

    def wait(self, timeout=None):
        status = self.process.wait(timeout=timeout)
        self.close()
        return status

    def communicate(self, *args, **kwargs):
        result = self.process.communicate(*args, **kwargs)
        self._receive()
        return result

    def __del__(self):
        self.close()


def popen_supervised(command, **options):
    fd = _SLOT_FD.get()
    if fd is None:
        return subprocess.Popen(command, **options)
    if not sys.platform.startswith("linux"):
        raise BudgetError("command-lifetime verification slots require Linux child-subreaper support")
    reader, writer = os.pipe()
    passed = tuple(options.get("pass_fds", ()))
    options["pass_fds"] = (*passed, fd, writer)
    options.setdefault("start_new_session", True)
    supervisor = [sys.executable, str(pathlib.Path(__file__).resolve()), "--supervise",
                  str(fd), str(writer), json.dumps(passed), *command]
    try:
        try:
            process = subprocess.Popen(supervisor, **options)
        finally:
            os.close(writer)
    except BaseException:
        os.close(reader)
        raise
    job = _SupervisedProcess(process, reader)
    try:
        while job.command_pid is None:
            job._receive(block=True)
    except BaseException as error:
        # An exec/setup refusal has no running command. On outer interruption,
        # closing this channel leaves the supervisor and its lock alive.
        job.close()
        if isinstance(error, (OSError, BudgetError)):
            process.wait()
        raise
    return job


def run_supervised(command, *, check=False, **options):
    if _SLOT_FD.get() is None:
        return subprocess.run(command, check=check, **options)
    # subprocess.run kills its child on KeyboardInterrupt. That child is our
    # supervisor: killing it would abandon the actual job and release the slot.
    # Wait without a context manager so wrapper interruption leaves it alive.
    child = popen_supervised(command, **options)
    stdout, stderr = child.communicate()
    result = subprocess.CompletedProcess(command, child.command_returncode, stdout, stderr)
    if check:
        result.check_returncode()
    return result


def _send_event(fd, event):
    if fd is None:
        return
    data = memoryview((json.dumps(event) + "\n").encode())
    try:
        while data:
            data = data[os.write(fd, data):]
    except BrokenPipeError:
        pass  # Outer wrapper died; keep supervising the owned subtree.


def _supervise(fd, command, status_fd=None, pass_fds=()):
    # This process outlives a killed wrapper. A detached descendant that closes
    # inherited FDs is still adopted here when its command parent exits.
    import ctypes
    os.fstat(fd)
    libc = ctypes.CDLL(None, use_errno=True)
    prctl = libc.prctl
    prctl.argtypes = [ctypes.c_int, ctypes.c_ulong, ctypes.c_ulong, ctypes.c_ulong, ctypes.c_ulong]
    prctl.restype = ctypes.c_int
    if prctl(36, 1, 0, 0, 0):  # PR_SET_CHILD_SUBREAPER
        raise BudgetError("cannot enable verification command child-subreaper; command refused")
    child = subprocess.Popen(command, pass_fds=pass_fds)
    _send_event(status_fd, {"event": "started", "pid": child.pid})
    status = child.wait()
    _send_event(status_fd, {"event": "exit", "returncode": status})
    while True:
        try:
            os.waitpid(-1, 0)  # blocking reap, including adopted descendants
        except ChildProcessError:
            break
    return status if status >= 0 else 128 - status


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    command = args.command
    if command[:1] == ["--"]:
        command = command[1:]
    if not command:
        parser.error("a verification command is required after --")
    root = pathlib.Path(__file__).resolve().parents[2]
    env = budget_environment(root / ".cache/tui-visual/tmp")
    # Go's package build concurrency is independent of GOMAXPROCS. Pin it last
    # in GOFLAGS; an explicit command-line -p still takes precedence.
    env["GOFLAGS"] = env.get("GOFLAGS", "") + " -p=" + env["GOMAXPROCS"]
    with verification_slot(root):
        return run_supervised(command, cwd=root, env=env).returncode


if __name__ == "__main__":
    try:
        if sys.argv[1:2] == ["--supervise"]:
            status_fd = int(sys.argv[3])
            try:
                sys.exit(_supervise(int(sys.argv[2]), sys.argv[5:], status_fd, tuple(json.loads(sys.argv[4]))))
            except OSError as error:
                _send_event(status_fd, {"event": "exec-error", "errno": error.errno,
                                       "message": error.strerror, "filename": error.filename})
                sys.exit(1)
            except BudgetError as error:
                _send_event(status_fd, {"event": "supervisor-error", "message": str(error)})
                sys.exit(1)
        sys.exit(main())
    except BudgetError as error:
        print(f"verification: {error}", file=sys.stderr)
        sys.exit(1)
