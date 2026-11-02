# A6 independent startup/error ownership review

**Spec verdict: PARTIAL / CHANGES REQUESTED.** The bounded terminal ownership correction is implemented and supported by real Linux PTY evidence. One cancellation admission check remains missing (I1). No unauthorized extra scope or separate misunderstood requirement found. Native platform acceptance and failed-reader-construction descriptor cleanup remain explicitly open parts of backlog20.

**Quality verdict: CHANGES REQUESTED — one medium (P2) finding.** No new terminal-restoration, input join-before-close, or original-error-identity regression was demonstrated in the exercised paths. I1 is a remaining A6 cancellation gap, not evidence that A1 joining regressed.

## Reviewed package and authority

- BASE: `6c50a7a50c5aead032b89ec451ea8b9d11bb5412`.
- HEAD: `9504322174abf2282927abfc89b1a6f98e200475`.
- Worktree: `/home/xertrov/src/oneline-ccc/.worktrees/terminal-startup`.
- Exact supplied package: main-controller `.superpowers/sdd/product-completion/A6-review.diff`, all 1,384 lines, including the maintained patch. It byte-matches `git diff --no-ext-diff --unified=8 BASE HEAD --binary`.
- Diff SHA-256: `d2bf9cfdf5f65367c7c350ccfee541a3ea1ec738cdcaf8a57abf8b54402d4ebb`.
- Read the A6 review/implementation briefs and implementer report, backlog20, product-completion campaign, third-party ownership/provenance documentation, and controller A1 review/rereview. Inspected affected Tea/platform/renderer ownership, production foreground integration, actual fixtures, fault overlays, RED/GREEN logs and validation records.

The six changed paths are the authorized new CLI startup fixture, new dependency startup fixture, Tea `tea.go`/`tty.go`, `third_party/README.md`, and `third_party/terminal-input.patch`. Root/module dependency files, manifest/verifier, Ultraviolet, existing A1 tests and production CLI helper are unchanged. The reviewed worktree remained clean at the exact HEAD. Source/git were not mutated; no agents, notifications, network or live operator configuration were used. This report is the only authored durable review artifact; bounded verification used existing worktree-local caches and anonymous overlay files.

## I1 — Medium (P2): cancellation during renderer startup still admits Model.Init

**Location:** `third_party/bubbletea/tea.go:1193–1197`, particularly the unconditional callback at line 1196. Related cancellation check: lines 1173–1175; renderer acquisition: line 1178. Ownership documentation: `third_party/README.md:41–43`.

`Run` checks cancellation before starting the renderer, then starts it, registers handlers and unlocks terminal acquisition. It calls `model.Init()` without rechecking cancellation. The next check is after that callback returns. A root context canceled during renderer startup therefore still causes a new model initialization callback to run. This contradicts the documented A6 behavior, “Startup cancellation skips further model initialization/rendering,” and leaves cancellation coverage incomplete after the earlier acquisition check.

**Deterministic reproduction:** use the existing dependency fixture renderer, overriding `start()` to cancel the external root before returning. The callback has not begun when cancellation occurs. An anonymous Go overlay appended the following test to `startup_lifetime_test.go`; no source/test file was edited:

```go
type a6ReviewCancelRenderer struct {
	startupLifecycleRenderer
	cancel context.CancelFunc
}

func (r *a6ReviewCancelRenderer) start() {
	r.startupLifecycleRenderer.start()
	r.cancel()
}

func TestA6ReviewCanceledDuringRendererStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	m := &startupLifecycleModel{init: func() Cmd {
		calls++
		return nil
	}}
	p := NewProgram(m, WithContext(ctx),
		WithInput(strings.NewReader("")), WithOutput(io.Discard),
		WithoutSignalHandler())
	r := &a6ReviewCancelRenderer{cancel: cancel}
	p.renderer = r
	_, err := p.Run()
	if !errors.Is(err, ErrProgramKilled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("error identity: %v", err)
	}
	if calls != 0 {
		t.Errorf("Model.Init ran %d time(s) after renderer startup canceled the root context; views=%d starts=%d closes=%d",
			calls, m.views, r.starts, r.closes)
	}
	p.Kill()
	p.Kill()
	p.Wait()
}
```

The existing file already imports everything this body needs. To reproduce, overlay its absolute path with the original contents plus this body, retaining the existing test helpers. The actual run used Linux anonymous `memfd_create` files exposed through the Python parent's `/proc/<pid>/fd/` paths: overlay JSON `Replace` mapped the original absolute source path to the appended anonymous source. Those paths expired when the host process ended; the complete added probe body and observed result are retained here.

Command executed from the reviewed worktree, with the bounded/offline environment below:

```sh
GOWORK="$PWD/.cache/a6/go.work" go test -race -p 2 -count=1 -timeout=20s \
  -overlay=/proc/1028039/fd/4 ./third_party/bubbletea \
  -run '^Test(A6ReviewCanceledDuringRendererStart|Startup)' -v
```

Observed Go child **exit 1**:

```text
=== RUN   TestA6ReviewCanceledDuringRendererStart
startup_lifetime_test.go:179: Model.Init ran 1 time(s) after renderer startup canceled the root context; views=0 starts=1 closes=1
--- FAIL: TestA6ReviewCanceledDuringRendererStart (0.00s)
FAIL charm.land/bubbletea/v2 0.017s
PROBE EXIT 1
```

All five committed `TestStartup*` cases passed in the same race-enabled invocation. No race diagnostic or hang occurred. Repeated Kill/Kill/Wait completed. The orchestration process printed the failing Go exit explicitly; its own exit 0 is not a passing test result.

**Effect and severity boundary:** a newly invoked user callback can perform initialization after root cancellation has already occurred. This is a medium callback-admission/spec gap, not a reproduced terminal leak or process leak. This probe uses a fixture renderer and no PTY; it establishes callback ordering, error identity, zero views, and one renderer start/close. The production `internal/tui/model.go:66` Init currently just returns its prepared command, which the subsequent cancellation check prevents from being submitted in this scenario. No additional production I/O or child launch was demonstrated. BASE also lacks the corresponding admission check; this is an incomplete A6 claim rather than a newly established BASE-to-HEAD resource regression. It concerns cancellation before a callback starts, not the disclosed inability to stop an already-running uncooperative callback.

**Requested fix:** recheck `p.ctx.Err()` immediately before invoking Init, after releasing `terminalMu`, and return `ErrProgramKilled` through the existing deferred cleanup when cancellation is already observable. Keep user callbacks outside that mutex so synchronous `p.Kill()` from Init remains safe. Add the deterministic renderer-start cancellation regression and retain `TestStartupKillFromModelInit`. Keep the README claim scoped to observed cancellation; an ordinary check does not promise atomic exclusion against cancellation racing after the check. Refresh the exact maintained patch and rerun the focused dependency/PTTY gates plus offline provenance verification.

## Ownership and behavior checked

- **Partial setup/error/panic:** the ownership defer is installed before terminal acquisition, unlocks the startup mutex before shutdown, and covers initial-size and reader-constructor errors as well as panics. Automatically opened TTY files close after teardown; caller-owned files remain open. The original failure and simultaneous external cancellation remain discoverable.
- **Cancellation and reentrancy:** Kill cancels before waiting for `terminalMu`. Setup/release/restore/shutdown serialize resource changes. Handler registration completes before startup unlocks. User Init runs outside the mutex; the committed synchronous-Init-Kill regression passes. An Init command goes through the existing command handler after initial rendering without a newly created sender after teardown. I1 is the remaining admission gap.
- **A1 lifetime invariant:** shutdown and release still cancel delivery and the underlying reader, join decoder/raw-reader completion, then close. No timeout fallback or reduced A1 assertion was introduced. The panic barrier keeps Close and Run return blocked until input completion. Existing queued-input cancellation/Quit/release and raw-child SIGTERM tests remain rendered and race-enabled.
- **Renderer ownership:** `rendererRunning` avoids stopping an unstarted renderer; its flush goroutine is joined before close/restoration/restart. Normal quit still renders initial/final views. Failure/cancellation cases exercised by the PTY fixture do not render a later model view.
- **Restoration:** successful input/output state restoration consumes the corresponding snapshot; failed restoration retains it. Canceled programs refuse RestoreTerminal acquisition. Returned reacquisition errors roll back reader/renderer/terminal state. This is a source assessment plus the bounded tests below, not exhaustive certification of external reacquisition panics or arbitrary callback combinations.

## Fresh independent verification

Linux runtime checks used disposable fixtures, with `GOMAXPROCS=2`, `-p 2`, `GOPROXY=off`, `GOTOOLCHAIN=local`, `TMPDIR`/`GOTMPDIR` at `$PWD/.cache/a6/tmp`, and `GOCACHE` at `$PWD/.cache/a6/gocache`. Only dependency tests used the existing `.cache/a6/go.work` workspace. Broad suites/builds were not rerun.

1. **Real rendered CLI/PTTY focused race gate — exit 0**, four top-level tests and 16 subcases, 4.112 seconds:

   ```sh
   go test -race -p 2 -count=1 -timeout=45s ./cmd/oneline-ccc \
     -run '^Test(Foreground(SIGTERMRestoresTerminalAfterRawChild|QueuedInputJoins|ProgramLifetime)|StartupTerminalFailureRestoresModes)$' -v
   ```

   Includes program lifetime, queued cancellation/Quit/release with 1,024 keys, the actual raw-child SIGTERM/reaping case, and all nine new startup cases: size, size-cancel, partial-setup-panic, input-init, input-init-cancel, model-init-panic, read-error, canceled-before-start and quit. Full original termios comparisons occur before fixture teardown. The two reader-constructor faults reexec and reap the same race-instrumented test binary. Intentional recovered-panic stacks are expected passing-fixture output.

2. **Returned partial-setup-error overlay under race — exit 0**, both partial-setup-error and partial-setup-error-cancel, 1.015 seconds:

   ```sh
   go test -race -p 2 -count=1 -timeout=20s \
     -overlay="$PWD/.cache/a6/partial-setup-green.overlay.json" \
     ./cmd/oneline-ccc -run '^TestStartupTerminalFailureRestoresModes$' -v
   ```

   Inspected the existing overlay/generator: it returns an error after actual MakeRaw at the observed descriptor barrier, rather than replacing restoration with a fake assertion.

3. **Dependency startup race gate plus independent cancellation probe:** the five committed startup cases pass; I1 fails as recorded above.

4. **Offline exact provenance — exit 0:**

   ```sh
   python3 third_party/verify.py /home/xertrov/go/pkg/mod/cache/download
   ```

   Both pinned archives pass archive SHA-256, module h1 and go.mod h1 checks: Tea v2.1.1 (79 upstream files) and Ultraviolet v0.0.0-20261008173134-6b8d4baf91b4 (109 upstream files). Complete mirrors match those archives plus exactly `terminal-input.patch`. The verifier, manifest and frozen module files were unchanged. The supplied fresh-apply log also records fuzz-0 patch application and exact reconstructed-tree comparisons.

5. `git diff --check BASE HEAD -- ':!third_party/terminal-input.patch'` exits 0. Generic whitespace checking of the maintained unified patch reports context whitespace, as disclosed by the worker; exact patch reconstruction, rather than suppressing unexplained source edits, is the provenance check used here.

## Supplied historical evidence inspected, not independently rerun

Parsed the actual JSON test logs as well as `.cache/a6/validation.json` and `validation-counts.json`; did not accept the worker's summary alone.

| Supplied gate | Recorded result checked |
| --- | --- |
| `focused-race-repeat10.log` | Exit 0; 40 top-level passes, 160 subcase passes, one package; no skips/race diagnostics |
| `cli-owner-normal.log` / `cli-owner-race.log` | Each exit 0; 60 top-level passes, 68 subcase passes, two packages; no race diagnostics |
| `dependencies-normal.log` / `dependencies-race.log` | Each exit 0; 230 top-level passes, 3,680 subcase passes, six packages; one unchanged subcase skip; no race diagnostics |
| CLI/owner and dependency vet logs | Empty successful logs; exit 0 recorded in the validation ledger |
| Six static builds | Recorded exit 0 for Linux/Darwin/Windows × amd64/arm64 with CGO disabled; `build-formats.log` records corresponding ELF/Mach-O/PE formats |

The dependency skip is `TestStyledWideCellPlaceholderDetection/styled_wide_cell:_checks_disagree_(regression)`. It is not new A6 coverage.

Inspected `red-pty.log` and `red-final-pty-overlay.log`: the original and final fixture both reject BASE in four actual raw-terminal cases with changed termios. Verified `.cache/a6/base-tea.go` exactly matches BASE Tea source. `red-partial-setup-error.log` rejects both returned-error cases; corresponding GREEN race evidence passes. `red-reentrant-init-kill.log` exposes the discarded lock-across-Init design's three-second deadlock, and final dependency tests demonstrate its correction. Additional supplied unstarted-renderer/setup-Kill/pre-canceled RED evidence supports those branches. These historical observations remain separate from the fresh commands above.

## Evidence limits and remaining obligations

1. **Failed constructor descriptors remain open work.** Pinned cancelreader v0.2.2's Linux EpollCtl failure closes its epoll descriptor but leaves two constructor-private wake-pipe files for finalization/process exit, returning no reader Tea can close. The subprocess fixtures prove terminal/input cleanup before exit and successful fixture reaping; they do not prove those private descriptors close before Run returns. This is an inherited, disclosed backlog20 obligation, not a newly introduced A6 defect and not waived by this review.
2. **Native macOS/Windows runtime remains unproved.** Six compilation/format results do not establish native startup failure, partial console-mode setup, interruption, foreground ownership or restoration. Linux partial-error injection cannot close Win32/macOS acceptance. Backlog20 remains open.
3. **Custom reader contract remains necessary.** A custom noncancelable blocking Read must be unblocked by its owner. No proof is claimed for permanently blocked I/O, already-running arbitrary user callbacks, externally invalidated terminal descriptors, or every possible panic/reentrancy combination. No abandoned-reader timeout workaround was added.
4. **Child evidence is scoped.** New startup faults happen before child launch. The unchanged real raw-child and process-owner tests provide separate foreground/child evidence; they do not turn each startup failure into a child-handoff trial.
5. Fresh focused test/probe output was inspected directly in the review session; the supplied worker logs remain under the worktree cache. The I1 anonymous overlay itself expired; its complete added Go body and failing output are recorded above. Preserve the worktree artifacts before removal if durable gate-log retention is required.

Accept the demonstrated Linux ownership improvement only with these boundaries, and resolve I1 before approving the A6 cancellation claim. No full-backlog or native-platform completion is certified.

=== DONE ===
