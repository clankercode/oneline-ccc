#!/usr/bin/env python3
"""Bound cooperative verification jobs without changing their evidence or inputs."""
import argparse
import contextlib
import contextvars
import fcntl
import os
import pathlib
import re
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


def _supervised(command, options):
    fd = _SLOT_FD.get()
    if fd is None:
        return command, options
    if not sys.platform.startswith("linux"):
        raise BudgetError("command-lifetime verification slots require Linux child-subreaper support")
    options = dict(options)
    options["pass_fds"] = (*options.get("pass_fds", ()), fd)
    options.setdefault("start_new_session", True)
    return [sys.executable, str(pathlib.Path(__file__).resolve()), "--supervise", str(fd), *command], options


def popen_supervised(command, **options):
    command, options = _supervised(command, options)
    return subprocess.Popen(command, **options)


def run_supervised(command, *, check=False, **options):
    if _SLOT_FD.get() is None:
        return subprocess.run(command, check=check, **options)
    command, options = _supervised(command, options)
    # subprocess.run kills its child on KeyboardInterrupt. That child is our
    # supervisor: killing it would abandon the actual job and release the slot.
    # Wait without a context manager so wrapper interruption leaves it alive.
    child = subprocess.Popen(command, **options)
    stdout, stderr = child.communicate()
    result = subprocess.CompletedProcess(command, child.returncode, stdout, stderr)
    if check:
        result.check_returncode()
    return result


def _supervise(fd, command):
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
    child = subprocess.Popen(command)
    status = child.wait()
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
            sys.exit(_supervise(int(sys.argv[2]), sys.argv[3:]))
        sys.exit(main())
    except BudgetError as error:
        print(f"verification: {error}", file=sys.stderr)
        sys.exit(1)
