Spec verdict: PASS — accepted I1 ADDRESSED. No Missing/Extra/Misunderstood requirement found within this scoped correction.

Quality verdict: APPROVED for the bounded I1 correction. No new Critical, Important or Minor finding identified in this review.

# A1 fix-round1 independent re-review

Scope: `d7869e1b3e1181ccaae27f0f00b63e8430535b33..5ccc5fa79b78d1c5278278f0014c3d2651ba7e2e`, in `/home/xertrov/src/oneline-ccc/.worktrees/terminal-race`.

Read the scoped re-review brief, actual scoped diff, actual `third_party/terminal-input.patch`, fix report and validation ledger, all six changed/added upstream files at their relevant boundaries, application/test changes, and retained diagnostic/gate evidence. The original A1 review and its accepted I1 are the baseline. Both saved scoped and full review diffs match the exact commit range byte-for-byte with `git diff --no-ext-diff --unified=7`; the full archive diff was compared programmatically, not dumped into context.

## I1 disposition: ADDRESSED

The previously reproduced scenario was queued decoded input surviving cancellation or normal Quit because dispatch could not observe cancellation and Tea abandoned its read-loop wait after 500 ms. The correction fixes both causes and covers live-context terminal handoff:

- At `third_party/ultraviolet/terminal_reader.go:230`, ordinary, timeout and EOF dispatch select between event delivery and the input-loop context. The canceled-context branch at line 163 does not attempt the old unconditional buffered-event flush; it waits for the raw-read worker. Buffered dispatch on the EOF path uses the same cancellation-aware send.
- At `third_party/bubbletea/tty.go:78`, each input loop gets a separate child context. Release can cancel that scope while the program/root remain live. `closeInputReader` at line 101 cancels delivery, cancels the underlying reader, waits for `readLoopDone`, and only then closes and clears the reader. The completion wait has no timer fallback and does not depend on `Cancel()` returning true. `StreamEvents`' wait group completes the underlying read before the decoder closes its completion channel.
- `third_party/bubbletea/tea.go:1298` routes graceful and forced shutdown through that same join-before-close operation. Terminal release at line 1366 uses it before restoring modes or running the child. Restore creates a fresh input scope after the old reader has been joined and closed. Nil input has no reader to wait for.
- `cmd/oneline-ccc/main.go:48` now supplies the actual root context directly to Tea again. The detached context and Quit watcher are removed. Backend/child cancellation, foreground session enrollment and protected-operation draining remain unchanged; the helper retains error/cancellation preservation.

There is no remaining I1 Scenario/Effect finding to carry forward. This approval establishes the corrected cancellation/dispatch/join ordering for the supported foreground path, subject to the explicit reader contract and separately retained platform/startup gaps below.

## Provenance and maintenance checked offline

Executed, read-only and without network access:

```sh
python3 third_party/verify.py /home/xertrov/go/pkg/mod/cache/download
```

Result, exit 0:

```text
charm.land/bubbletea/v2@v2.1.1: archive SHA-256, module h1, go.mod h1 verified; 79 upstream files compared
github.com/charmbracelet/ultraviolet@v0.0.0-20261008173134-6b8d4baf91b4: archive SHA-256, module h1, go.mod h1 verified; 109 upstream files compared
Complete mirrors match the pinned upstream archives plus exactly terminal-input.patch.
```

The manifests were not accepted merely as self-consistent metadata: both module/go.mod h1 pairs also match the **prior tracked** `d7869e1:go.sum`; cached archive Origin metadata matches the recorded upstream commits. Root `go.sum` is unchanged. The saved latest-main lookup/comparison evidence supports the worker's reason for a local correction; no network lookup was repeated here.

The verifier reconstructs every archive/local file difference, including additions/deletions, and rejects unexpected symlinks or a mismatch with the committed patch. It establishes three production edits, one existing test adjustment and two added dependency regression files. Every mirrored file is tracked: 80 Tea files and 110 Ultraviolet files after adding their tests. The other archive bytes, platform reader implementations, module files and licenses are unchanged. Public API signatures and required module versions remain intact. The README documents the changed ownership obligation, offline verification, rebasing/upstream removal and separate dependency test gates. The supplied fresh-extraction/apply log additionally reports exact patched file-set and byte equality; this review's verifier independently confirms the resulting bytes.

## Regression strength and evidence

- **The final assertions still reject the old implementation.** Inspected `.cache/go-tmp/A1-fix1-prior.overlay.json` and compared all four mapped source contents: the helper equals `d7869e1`, and the Tea/Ultraviolet production files equal their original pinned module-cache sources. No regression test is replaced by that overlay. `A1-fix1-red-prior-overlay.log` shows the final cancellation, Quit and release cases failing after approximately 503–505 ms with the original blocked input stack. Release also fails inside its execution boundary. The retained teardown descriptor race is consistent with deliberately restoring the unsafe lifetime; it is not presented as corrected-code GREEN evidence.
- **Stack checks cover active work before teardown.** `foreground_terminal_lifecycle_linux_test.go:151` matches active Tea read loops, decoders and underlying read calls while excluding only creation metadata and a completed read's `WaitGroup.Done` epilogue. The earlier provisional failure log shows precisely that epilogue false positive. The refinement does not filter the original blocking stack, as the verified RED overlay demonstrates. Checks occur inside the live-context release boundary and after `Run` returns, before closing the PTY or exiting the process. A baseline excludes pre-existing input work; no sleep, polling retry or elapsed-time success threshold hides a surviving reader.
- **Join ordering is deterministic.** `third_party/bubbletea/input_lifetime_test.go:19` withholds a completion barrier after observing reader cancellation. `testing/synctest` advances fake time beyond the old fallback and checks that both reader Close and boundary return remain blocked. Releasing the barrier must allow completion and Close. Quit, Kill and release all fail against the original implementation in `A1-fix1-red-dependencies.log`, and pass in the focused/final GREEN logs. This checks ordering rather than treating a short wall-clock duration as proof.
- **Blocked dispatch and incomplete EOF input are covered.** The Ultraviolet test leaves event delivery unreceived, waits until the source reaches EOF, then cancels. Both the ordinary-key and incomplete-escape cases fail on the original code and pass after correction. Its one-second timeout is a failure bound, not a substitute for the completion assertion. Code inspection confirms all dispatch sites use the cancellable path.
- **Rendered acceptance is preserved.** The original `TestForegroundSIGTERMRestoresTerminalAfterRawChild` body is byte-for-byte unchanged from `d7869e1`; the Linux file remains race-enabled. Its real child/raw-mode behavior, actual SIGTERM, reaping check and complete termios comparison remain. The UI and raw-child fixtures still reexec the same test binary with explicit fixture environments. The new queued-input tests use a real PTY, retain the renderer and compare modes before/after; release additionally checks canonical modes and input completion before resume while the root is live. They use no operator configuration or gateway.
- **The existing renderer race test was adapted without dropping its behavior.** `third_party/bubbletea/cursed_renderer_test.go:78` closes its owned non-cancelable pipe writer before Quit. Its 100 mouse-event deliveries, renderer setup and timeout remain. The initial full dependency failure, focused corrected test and final passing dependency suites are retained. This is the documented reader ownership contract, not restoration of the abandoned-read fallback.

The historical logs and `A1-fix1-validation.json` agree on the following results. Counts were independently parsed from the saved output, including deeper nested subtests:

| Gate | Reviewed result |
| --- | --- |
| Repeated rendered race gate | 30 queued cancellation, 30 Quit and 30 release cases; 30 original rendered raw-child SIGTERM passes; four lifetime subcases each pass 30 times; no failures/skips/race diagnostics; 32.840 s |
| Final focused gate | All named rendered/lifetime cases pass; original SIGTERM fixture is present; 2.071 s |
| Application/process-owner ordinary and race suites | Each has 54 top-level passes, 36 subtest passes, no skips; JSON exit 0 |
| Final dependency ordinary and race suites | Each has six successful packages, 223 top-level passes, 3,680 nested subtest passes and one unchanged non-lifetime skip; JSON exit 0 |
| Application and dependency vet | Empty diagnostic logs; exit 0 recorded in JSON |
| Static builds | Six Linux/Darwin/Windows amd64/arm64 commands report exit 0 and matching ELF/Mach-O/PE formats, with cgo disabled |

The dependency logs' recovered-panic messages come from their passing intentional panic tests. Their single skip is the unchanged styled-wide-cell regression documented by the implementer. No terminal-lifetime regression skips. Provisional files named GREEN that actually failed are disclosed and distinguished from final passing gates; the validation ledger also retains the initial pipe-owner failure.

## Limits and retained gaps

- Native macOS/Windows queued-input/console restoration and real startup failures after terminal mode changes remain **item20** work. The nil-model startup test does not prove those paths; this fix and this approval do not claim otherwise. They do not reopen accepted I1.
- A custom `io.Reader` whose blocking Read cannot be canceled must be unblocked by its owner before Quit/Kill/release can complete. This is an intentional, documented compatibility change required for a real join. The application foreground path uses native cancelable terminal readers. Arbitrary non-cancelable readers, blocked model initialization and all possible external I/O/panic interactions are not certified to terminate here.
- The saved test/build/vet results are historical worker evidence, checked against source, logs and the ledger. This reviewer reran only the requested offline provenance verifier and read-only comparisons; no broad test/build gates or additional runtime probes were needed. The nested dependency suites require the documented workspace invocation; root `go test ./...` alone does not include their tests.
- The stale race-exclusion statement in the coordinator-owned native verification documentation still needs integration cleanup. Whole-product completion and native visual acceptance are outside this review.

No source/git edits, new agents, notifications, attention utilities, operator-state access or network calls were performed. The sole new review artifact is this main-workspace ignored report. Final artifact checks confirm the task worktree remains clean at the exact reviewed HEAD.

=== DONE ===
