"""Budget propagation and real cross-process repository lock barriers."""
import contextlib
import json
import os
import pathlib
import runpy
import select
import shutil
import signal
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

import capture

HERE = pathlib.Path(__file__).resolve().parent


class BuildBudgetTests(unittest.TestCase):
    def setUp(self):
        (capture.CACHE / "tmp").mkdir(parents=True, exist_ok=True)

    def test_build_respects_explicit_and_default_budget(self):
        for requested, expected in [(None, "1"), ("1", "1"), ("2", "2")]:
            with self.subTest(requested=requested), tempfile.TemporaryDirectory(dir=capture.ROOT / ".cache/tui-visual/tmp") as scratch:
                env = {} if requested is None else {"GOMAXPROCS": requested}
                with mock.patch.dict(os.environ, env, clear=True), mock.patch("resource_budget.os.cpu_count", return_value=2), mock.patch.object(capture, "CACHE", pathlib.Path(scratch)), mock.patch.object(capture, "run") as run:
                    capture.build()
                command = run.call_args.args[0]
                self.assertEqual(command[command.index("-p") + 1], expected)
                self.assertEqual(run.call_args.kwargs["env"]["GOMAXPROCS"], expected)

    def test_invalid_budget_fails_before_build(self):
        for requested in ["", "0", "-1", "1.5", "two", " 1", "999999999999999999999"]:
            with self.subTest(requested=requested), mock.patch.dict(os.environ, {"GOMAXPROCS": requested}), mock.patch.object(capture, "run") as run:
                with self.assertRaises(ValueError):
                    capture.build()
                run.assert_not_called()

    def test_budget_tests_also_pass_on_simulated_single_cpu_host(self):
        with mock.patch("resource_budget.os.cpu_count", return_value=1):
            self.test_build_respects_explicit_and_default_budget()

    def test_single_cpu_accepts_one_and_rejects_two(self):
        with mock.patch("resource_budget.os.cpu_count", return_value=1), mock.patch.dict(os.environ, {"GOMAXPROCS": "1"}), mock.patch.object(capture, "run") as run:
            capture.build()
            self.assertEqual(run.call_args.kwargs["env"]["GOMAXPROCS"], "1")
            with mock.patch.dict(os.environ, {"GOMAXPROCS": "2"}):
                with self.assertRaisesRegex(ValueError, "1 to 1"):
                    capture.build()


class ControllerBudgetTests(unittest.TestCase):
    def test_capture_build_and_render_entrypoints_hold_the_slot(self):
        for mode in ["build", "capture", "batch"]:
            active = []
            @contextlib.contextmanager
            def slot(root):
                self.assertEqual(root, capture.ROOT)
                active.append(True)
                try:
                    yield
                finally:
                    active.pop()
            def check(*args):
                self.assertEqual(active, [True])
            with self.subTest(mode=mode), mock.patch.object(sys, "argv", ["capture.py", mode]), mock.patch.object(capture, "verification_slot", slot), mock.patch.object(capture, "build", side_effect=check), mock.patch.object(capture, "execute_capture", side_effect=check):
                capture.main()
            self.assertEqual(active, [])

    def test_plan_does_not_wait_for_cpu_slot(self):
        with mock.patch.object(sys, "argv", ["capture.py", "batch", "--plan"]), mock.patch.object(capture, "verification_slot") as slot, mock.patch.object(capture, "execute_capture") as execute:
            capture.main()
        slot.assert_not_called()
        execute.assert_called_once()

    def test_controller_preserves_sixteen_captures_and_budgets_node_workers(self):
        import resource_budget
        with mock.patch.dict(os.environ, {"GOMAXPROCS": "1"}), mock.patch.object(subprocess, "run") as run, mock.patch.object(resource_budget, "verification_slot", return_value=contextlib.nullcontext()):
            runpy.run_path(str(HERE / "run_fault_tests.py"))
        commands = [call.args[0] for call in run.call_args_list]
        self.assertEqual(sum(str(HERE / "capture.py") in command for command in commands), 16)
        node = next(call for call in run.call_args_list if call.args[0][0] == "node")
        self.assertIn("--test-concurrency=1", node.args[0])
        self.assertEqual(node.kwargs["env"]["GOMAXPROCS"], "1")
        self.assertEqual(sum(str(command).endswith(".test.mjs") for command in node.args[0]), 5)


class RepositorySlotTests(unittest.TestCase):
    def setUp(self):
        helper = HERE / "resource_budget.py"
        self.assertTrue(helper.is_file(), "shared blocking repository budget helper is missing")
        (capture.CACHE / "tmp").mkdir(parents=True, exist_ok=True)
        self.scratch = tempfile.TemporaryDirectory(dir=capture.ROOT / ".cache/tui-visual/tmp")
        self.addCleanup(self.scratch.cleanup)
        self.root = pathlib.Path(self.scratch.name)
        self.repo = self.root / "repo"
        subprocess.run(["git", "init", "-q", str(self.repo)], check=True)
        subprocess.run(["git", "-C", str(self.repo), "-c", "user.name=Budget Test", "-c", "user.email=budget@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "fixture"], check=True)
        self.worktree = self.root / "worktree"
        subprocess.run(["git", "-C", str(self.repo), "worktree", "add", "-q", "-b", "fixture", str(self.worktree)], check=True)

    def start(self, root, fail=False):
        code = """
import pathlib, sys
from resource_budget import verification_slot
try:
    with verification_slot(pathlib.Path(sys.argv[1])):
        print('entered', flush=True)
        sys.stdin.readline()
        if sys.argv[2] == 'True':
            raise RuntimeError('injected failure')
except RuntimeError:
    raise SystemExit(7)
"""
        proc = subprocess.Popen([sys.executable, "-c", code, str(root), str(fail)], cwd=HERE, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        def cleanup():
            if proc.poll() is None:
                proc.kill()
            proc.communicate(timeout=5)
        self.addCleanup(cleanup)
        return proc

    def line(self, stream):
        self.assertTrue(select.select([stream], [], [], 5)[0], "process barrier timed out")
        return stream.readline().strip()

    def test_worktrees_share_slot_and_failure_releases_waiter(self):
        owner = self.start(self.repo, fail=True)
        self.assertEqual(self.line(owner.stdout), "entered")
        waiter = self.start(self.worktree)
        self.assertIn("waiting", self.line(waiter.stderr))
        self.assertFalse(select.select([waiter.stdout], [], [], 0)[0], "worktree entered while owner held slot")
        owner.stdin.write("release\n"); owner.stdin.flush()
        self.assertEqual(owner.wait(timeout=5), 7)
        self.assertEqual(self.line(waiter.stdout), "entered")
        waiter.stdin.write("release\n"); waiter.stdin.flush()
        self.assertEqual(waiter.wait(timeout=5), 0)

    def test_distinct_repositories_do_not_block_each_other(self):
        other = self.root / "other"
        subprocess.run(["git", "init", "-q", str(other)], check=True)
        owner = self.start(self.repo)
        self.assertEqual(self.line(owner.stdout), "entered")
        independent = self.start(other)
        self.assertEqual(self.line(independent.stdout), "entered")

    def test_process_death_releases_slot(self):
        owner = self.start(self.repo)
        self.assertEqual(self.line(owner.stdout), "entered")
        waiter = self.start(self.worktree)
        self.assertIn("waiting", self.line(waiter.stderr))
        owner.kill(); owner.wait(timeout=5)
        self.assertEqual(self.line(waiter.stdout), "entered")

    def test_command_wrapper_preserves_cache_budget_and_failure_status(self):
        helper = self.repo / "tests/tui-visual/resource_budget.py"
        helper.parent.mkdir(parents=True)
        shutil.copyfile(HERE / "resource_budget.py", helper)
        cache = str(self.root / "existing-go-cache")
        env = dict(os.environ, GOMAXPROCS="1", GOCACHE=cache, GOFLAGS="-trimpath")
        code = "import json,os; print(json.dumps({key:os.environ[key] for key in ['GOMAXPROCS','GOCACHE','GOFLAGS','TMPDIR','GOTMPDIR']})); raise SystemExit(7)"
        result = subprocess.run([sys.executable, str(helper), "--", sys.executable, "-c", code], env=env, capture_output=True, text=True, timeout=5)
        self.assertEqual(result.returncode, 7)
        actual = json.loads(result.stdout)
        self.assertEqual(actual["GOMAXPROCS"], "1")
        self.assertEqual(actual["GOCACHE"], cache)
        self.assertEqual(actual["GOFLAGS"], "-trimpath -p=1")
        self.assertEqual(actual["TMPDIR"], str(self.repo / ".cache/tui-visual/tmp"))
        self.assertEqual(actual["GOTMPDIR"], actual["TMPDIR"])
        follower = self.start(self.repo)
        self.assertEqual(self.line(follower.stdout), "entered")

    def test_supervised_protocol_pty_preserves_cleanup(self):
        from resource_budget import verification_slot
        # Protocol peer only: no native binary, renderer, browser or Go build.
        with verification_slot(self.repo):
            result = capture.capture_pty([sys.executable, str(HERE / "protocol_peer.py")], {}, 80, 24, 2)
        self.assertIn("模型".encode(), result["ansi"])
        self.assertIn(b"\x1b[?1049l", result["cleanup"])
        self.assertTrue(result["termios_restored"])

    def test_capture_command_failure_releases_supervised_slot(self):
        from resource_budget import verification_slot
        with self.assertRaisesRegex(capture.CaptureError, r"tool failed \(7\)"):
            with verification_slot(self.repo):
                capture.run([sys.executable, "-c", "raise SystemExit(7)"])
        follower = self.start(self.worktree)
        self.assertEqual(self.line(follower.stdout), "entered")

    def test_unsupported_platform_refuses_heavy_command(self):
        import resource_budget
        with resource_budget.verification_slot(self.repo):
            with mock.patch.object(sys, "platform", "darwin"), mock.patch.object(subprocess, "Popen") as spawn:
                with self.assertRaisesRegex(ValueError, "require Linux"):
                    resource_budget.run_supervised([sys.executable, "-c", "pass"])
                spawn.assert_not_called()

    def test_subreaper_setup_failure_refuses_heavy_command(self):
        import resource_budget
        with tempfile.TemporaryFile(dir=self.root) as lock, mock.patch("ctypes.CDLL") as libc, mock.patch.object(subprocess, "Popen") as spawn:
            libc.return_value.prctl.return_value = -1
            with self.assertRaisesRegex(ValueError, "command refused"):
                resource_budget._supervise(lock.fileno(), [sys.executable, "-c", "pass"])
            spawn.assert_not_called()

    def test_wrapper_death_keeps_live_command_inside_slot(self):
        self.check_live_command_after_owner_exit(kill_owner=True, descendant=False)

    def test_wrapper_interrupt_keeps_live_command_inside_slot(self):
        self.check_live_command_after_owner_exit(kill_owner=True, descendant=False, interrupt_owner=True)

    def test_wrapper_death_keeps_detached_descendant_inside_slot(self):
        self.check_live_command_after_owner_exit(kill_owner=True, descendant=True)

    def test_command_exit_keeps_detached_descendant_inside_slot(self):
        self.check_live_command_after_owner_exit(kill_owner=False, descendant=True)

    def check_live_command_after_owner_exit(self, *, kill_owner, descendant, interrupt_owner=False):
        helper = self.repo / "tests/tui-visual/resource_budget.py"
        helper.parent.mkdir(parents=True)
        shutil.copyfile(HERE / "resource_budget.py", helper)
        child = "import os,sys; print(os.getpid(), flush=True); sys.stdin.readline(); print('child-finished', flush=True)"
        if descendant:
            # The command leader exits; the detached grandchild closes inherited
            # non-stdio descriptors. Passing the flock FD alone cannot fix this.
            child = "import subprocess,sys; subprocess.Popen([sys.executable, '-c', " + repr(child) + "], start_new_session=True, close_fds=True)"
        owner = subprocess.Popen([sys.executable, str(helper), "--", sys.executable, "-c", child], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        def cleanup():
            try:
                owner.stdin.write("release\n"); owner.stdin.flush()
            except BrokenPipeError:
                pass
            owner.communicate(timeout=5)
        self.addCleanup(cleanup)
        child_pid = int(self.line(owner.stdout))
        waiter = self.start(self.worktree)
        self.assertIn("waiting", self.line(waiter.stderr))
        if kill_owner:
            if interrupt_owner:
                owner.send_signal(signal.SIGINT)
            else:
                owner.kill()
            self.assertNotEqual(owner.wait(timeout=5), 0)
        os.kill(child_pid, 0)
        self.assertFalse(select.select([waiter.stdout], [], [], .15)[0], "slot escaped while the owned command or descendant was alive")
        owner.stdin.write("release\n"); owner.stdin.flush()
        self.assertEqual(self.line(owner.stdout), "child-finished")
        self.assertEqual(self.line(waiter.stdout), "entered")
        if not kill_owner:
            self.assertEqual(owner.wait(timeout=5), 0)


if __name__ == "__main__":
    unittest.main()
