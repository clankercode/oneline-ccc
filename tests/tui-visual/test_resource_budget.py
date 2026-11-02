"""Budget propagation and real cross-process repository lock barriers."""
import contextlib
import json
import os
import pathlib
import runpy
import select
import shutil
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
                with mock.patch.dict(os.environ, env, clear=True), mock.patch.object(capture, "CACHE", pathlib.Path(scratch)), mock.patch.object(capture, "run") as run:
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


if __name__ == "__main__":
    unittest.main()
