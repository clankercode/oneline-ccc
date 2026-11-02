# TUI verification CPU and resource budget

Recorded 2026-11-03. The avoidable load came from concurrent broad suites and
captures across worktrees, together with capture builds overriding an explicit
one-CPU setting. A profile of the normal TUI tests showed real rendering work:
`assertFrame` accounted for 90.64% cumulative sampled CPU, `View` 82.94%, layout
48.83% and fill 40.80%. These stacks overlap and cannot be added. The profile
does not show an idle application busy loop.

The reviewed tooling integrated at `e070d7d` defaults to `GOMAXPROCS=1` and one
Go package worker. An explicit valid CPU budget is honored. Heavy cooperating
jobs share one blocking slot through the Git common directory, including all
worktrees. An independent Linux child subreaper retains that slot through the
actual command subtree, even when the outer wrapper dies or a descendant
detaches. Waiting and reaping block in the kernel. A separate status channel
preserves the actual command's early failure, exec error and final outcome.

For an ordinary edit, run the relevant tests:

```sh
python3 tests/tui-visual/resource_budget.py -- go test -count=1 -run '^TestPaletteHelpAndEditorFitSmallTerminal$' ./internal/tui
```

For the integrated revision, run these serially:

```sh
python3 tests/tui-visual/resource_budget.py -- go test -count=1 ./...
python3 tests/tui-visual/resource_budget.py -- go test -race -count=1 ./...
python3 tests/tui-visual/resource_budget.py -- go vet ./...
```

Capture and fault-controller entrypoints acquire the same slot themselves;
**do not nest the wrapper around them**. Existing compiler caches are reused.
Captures still rebuild against actual source and fixture inputs; there is no
new binary cache or unverified artifact reuse. Required assertions, scene
inventory, sizes, themes and race coverage have not been removed.

Actual integrated Linux results at `e070d7d`, using the warm compiler cache:

| Gate | Result | Wall seconds | Child user + system CPU seconds |
| --- | --- | ---: | ---: |
| Full normal Go tests | PASS | 21.011 | 10.903 |
| Full race Go tests | PASS | 128.929 | 69.269 |
| `go vet` | PASS | 0.861 | 0.859 |
| Focused resource-budget controls | 29 PASS, zero skips | 3.011 | Not separately sampled |

Peak child RSS across the normal gate was 185572 KiB; the cumulative maximum
after the race gate was 563252 KiB. Wall time uses a monotonic clock; CPU/RSS
use Linux `getrusage(RUSAGE_CHILDREN)`. The full normal/race suite includes
platform-gated tests; a Linux pass does not execute Windows/macOS cases or
unconfigured live-client fixtures.

These measurements establish gate cost and the tested concurrency bound, not
an application algorithm speedup. An earlier build-only comparison honored one
CPU after the fix, versus an overridden two before it; warm CPU times were
0.237 versus 0.246 seconds, too small and noisy for a speedup claim. Race
instrumentation and exhaustive frame checks remain substantial work. Avoid
repeating full race suites and matrices for each presentation edit.

Limits: the development supervisor requires Linux and must survive. Its own
SIGKILL/OOM, or escaped descendants after group-wide supervisor cleanup, are
outside its guarantee. A stuck surviving descendant retains the slot.
Direct/unwrapped commands do not participate. `GOMAXPROCS` does not cap browser
CPU. The production Go application's Windows/macOS support is unchanged.

The browser lifecycle correction still needs independent approval and actual
browser verification; a leader exit must not release a job or permit evidence
publication while its children survive. Actual browser timing, fresh native
PNG/reference comparisons and the scene-backed fault controller remain open.
See [capture commands and guarantees](../../tests/tui-visual/README.md), the
[remediation checkpoint](native-remediation-progress.md) and the
[remaining items](../plans/remaining-items-so-far.md).
