#!/usr/bin/env python3
"""Bound cooperative verification jobs without changing their evidence or inputs."""
import argparse
import contextlib
import fcntl
import os
import pathlib
import re
import subprocess
import sys


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
    The descriptor is not inherited by subprocesses; the owning wrapper waits
    for its command and closes it on normal exit, exceptions or process death.
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
        try:
            yield
        finally:
            fcntl.flock(handle, fcntl.LOCK_UN)


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
        return subprocess.run(command, cwd=root, env=env).returncode


if __name__ == "__main__":
    try:
        sys.exit(main())
    except BudgetError as error:
        print(f"verification: {error}", file=sys.stderr)
        sys.exit(1)
