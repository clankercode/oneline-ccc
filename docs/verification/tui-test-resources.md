# TUI verification CPU and resource budget

Recorded 2026-11-03. The avoidable load came from concurrent broad suites and
captures across worktrees, together with capture builds overriding an explicit
one-CPU setting. A profile of the normal TUI tests showed real rendering work:
`assertFrame` accounted for 90.64% cumulative sampled CPU, `View` 82.94%, layout
48.83% and fill 40.80%. These stacks overlap and cannot be added. The profile
does not show an idle application busy loop.

The reviewed tooling integrated at `e070d7d` defaults to `GOMAXPROCS=1` and one
Go package worker. Follow-up `743b837` sets and verifies Linux CPU affinity inside
the supervisor before command launch. The command subtree, including Node and
Chromium, inherits at most that many allowed CPUs; an already narrower inherited
mask stays narrow. Failure to establish the mask refuses command execution. An explicit valid CPU budget is honored. Heavy cooperating
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
Direct/unwrapped commands do not participate. Affinity is an inherited execution
bound, not a CPU-time quota or sandbox; a command deliberately widening its own
mask can escape it. `GOMAXPROCS` alone does not cap browser CPU. The production Go
application's Windows/macOS support is unchanged.

Fresh integrated resource controls at `743b837` pass **35 tests, zero skips,
4.278 seconds**, including child/grandchild affinity inheritance, constrained
parent masks, setup refusal and the existing subtree/slot lifetime checks. This
proves a tested peak-execution bound; no lower aggregate CPU-time or rendering
algorithm speedup is claimed. The focused resource test suite exercises explicit
two-CPU cases; run that suite directly rather than inside the one-CPU wrapper.

The reviewed browser fixes at `bcaad1d`/`7359dce` pin Playwright 1.64.0 and
Chromium 156 and bound complete Unix socket paths under the common checkout.
The real scene controller passes 16 captures, 44 Python tests and 154 Node
checks (one optional legacy-fixture skip). A 150-frame base-screen matrix
was generated and strictly verified on candidate `5564042`; its provenance is
retained unchanged. Independent pixel/source reviews found usability defects
that require fresh affected captures after fixes. Browser timing profiles,
remaining layers/states and full visual acceptance are still open.
See [capture commands and guarantees](../../tests/tui-visual/README.md), the
[remediation checkpoint](native-remediation-progress.md) and the
[remaining items](../plans/remaining-items-so-far.md).
