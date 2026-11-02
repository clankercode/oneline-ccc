# Final native foundation fix report

Completed the assigned single fix pass in `$WORKTREES/native-final-fix`, branch `feat/native-final-fix`, base `468e886d402459107180ece040e5fd8886fe3df7`.

Commit: **`419b0cf83efff128f15761daa42186972e51fea9`** — Fix native foreground shutdown, metadata and state fidelity.
Working tree clean after commit. No merge, rebase, push, subagents or independent review commissioned. Only the 17 allowed paths below changed. CLI producer-owned source/tests and `sshhelper.go` are untouched. No README/site/docs changes; D1 remains the parent's assignment.

Read the complete brief, final review and exact retained probes, native foundation plan, native usage document, protected transaction and observer module contracts. All test/build scratch and evidence are under the native-app sibling's `.cache`; all Go commands use GOMAXPROCS=2 and offline module settings. No operator configuration, real credentials/vaults, native services, public network, gateway or inference calls were used.

## Finding results and RED/GREEN evidence

Evidence directory: `$REPO/.cache/native-app/final-fix/`.

### N1 — root lifetime, cancellation, safe drain, closed enrollment

`runForeground` is now the actual production `runUI` boundary. It creates the session root, supplies it to `tui.Options.Context`, wraps every core.Backend method, and tracks enrolled awaits. When Bubble Tea returns, including error/root shutdown, it closes enrollment, cancels the session and waits for active calls to finish cleanup. The injected child runner uses the same admission/drain mechanism. No shutdown timeout abandons synchronous vault writes or transaction cleanup. `Command` preserves its context until terminal execution finishes.

RED: `red.log` reproduces the retained original probe: `operation_context_active_after_program_return=true backend_can_mutate_after_return=true`. `red-foreground.log` separately drives a real headless Bubble Tea Program through the production helper for reviewed/acknowledged Apply, explicit persistent key save, and confirmed service install. Each fails because cancellation does not reach the backend and the production boundary returns before safe cleanup. The helper was first extracted with unchanged behavior for this RED run.

GREEN: `green-foreground-runner.log` passes `TestFinalFixForegroundShutdown` for all three journeys. Tests hold cleanup at a channel barrier after Program.Run returns, observe operation cancellation, prove the production helper remains blocked, release cleanup, then prove the safe final outcome precedes helper return. The full suite also passes `TestFinalFixForegroundClosesAdmission`, which rejects all eight backend methods during/after shutdown while draining the enrolled Apply. Existing raw Ctrl+C dirty-quit and cooked-child SIGINT/SIGTERM tests pass. `retained-green.log` now records both original probe booleans false; the new production-helper tests provide the drain proof missing from that older probe.

Limit: these are actual Bubble Tea/control-flow tests with synthetic mutation callbacks, not a new native transaction power-loss or actual vault/service test.

### C1 — shared owned-child runner integration

`Options.Runner` has the unchanged producer signature `(context.Context, *exec.Cmd, io.Reader, io.Writer, io.Writer) error`. The TUI adapts it to `tea.Exec`; Tea still suspends/restores the terminal. Production's foreground helper defaults to `runProcess` and wraps its complete lifetime in cancellation/admission/drain. No process-tree or terminal-group algorithm was copied into tui.

RED: `red-runner.log`, `TestFinalFixTerminalRunner`: `terminal handoff bypassed injected owned-child runner`.

GREEN: the same actual headless Bubble Tea test passes in `green-foreground-runner.log` and the full suite. It checks the original command pointer, literal space/Unicode/metacharacter argv, explicit non-nil environment, supplied streams, live context, and preservation of synthetic child exit code 23.

Integration: read producer `7fbda89277bbef7ac0966e1a4e364267ddcf906b` with git show, then placed its ten changed source/test files in a Go overlay under this evidence directory. No owned file in this worktree was changed. `producer-integration.log` is a fresh race-enabled full CLI-package PASS against that overlay, including the producer's fixture terminal/process tests: `ok .../cmd/oneline-ccc 7.626s`. Parent reported integration as b9ca80d; this worker did not merge it. Actual production TUI PTY/platform acceptance remains with the parent.

### T1 — Linux native inode policy preservation/refusal

The shared `noExtraMetadata` handle seam now queries FS_IOC_GETFLAGS after xattrs. It permits filesystem representation bits INDEX, HUGE_FILE, EXTENT, EOFBLOCKS and INLINE_DATA. Other bits, including NODUMP, NOATIME, compression/NOCOW/project policy and unknown bits, produce Conflict. Query errors, including unsupported ioctls/filesystems, fail closed with ErrUnsafe. Existing preflight, staged replacement and recovery metadata callers therefore enforce the same policy without a new public Write/journal format.

RED: `red.log`, `TestFinalFixNodumpRefusal`: initial/reviewed/staged nodump was accepted; recovery failed to preserve the foreign policy.

GREEN: `green-subsystems.log` and final suite pass all four real unprivileged nodump cases: initial refusal before state creation; drift after review before Apply with no state; drift after staging; nodump on own output before recovery. Tests assert inode/content/flag preservation and retained exact backup on recovery Conflict. `TestFinalFixInodeFlagQueryPolicy` covers permitted representation flags, rejected policy/unknown bits and ENOTTY/EOPNOTSUPP/ENOSYS/EIO refusal. Unsupported-filesystem errors are policy-unit fixtures; no separate mounted unsupported filesystem was created. `retained-green.log` records the original real probe's `outcome=Conflict ... before=40 after=40`.

### T2 — native busctl direct typed arrays

`busProperty` retains the complete validated array `data`. UnitPath consumes the direct string array; ExecStart consumes the direct array of ten-field structs. Fixture producer encoding is corrected. Signature, response limit, strict JSON, array bounds, path authority, all drop-in checks and exact effective argv remain enforced. The tuple's boolean and remaining unsigned/signed numeric fields are checked for type/null/range too.

RED: `red.log`, `TestFinalFixNativeBusctlArrays`: multiple paths, single path and one command tuple all rejected native direct-array encoding.

GREEN: `green-subsystems.log`, final suite and `retained-green.log` pass those native wire shapes. `TestFinalFixMalformedBusctlArrays` refuses absent/null/scalar/nested/wrong-signature/invalid-arity values. `TestFinalFixExecStartTypedFields` additionally rejects invalid booleans, null/wrong/overflowing timestamps, PID and signed exit fields. Existing authority/drop-in/argv/absence/runtime-enablement/revocation regressions pass in the whole suite. Read the controller's `native-systemd-protocol-check.md`; no actual systemd command or native service mutation was performed by this worker.

### U2 — failure retains returned safe service state

On an error-plus-ServiceInfo result, the TUI adopts Unknown/Conflict/Unavailable, retains the failure notice, and adds inspect/reconcile-before-retry guidance. Invalid status projections are ignored. It neither labels the action successful nor resends it.

RED: `red.log`, `TestFinalFixServiceFailureState`: all six prior Stopped/Running × returned Unknown/Conflict/Unavailable journeys retained the previous observed state.

GREEN: `green-subsystems.log`, final suite and `retained-green.log` pass those projections. Tests assert visible returned state, retained error/reconciliation notice and one mutation call; invalid returned status remains ignored.

### K1 — explicit saved key becomes current, with persistence fidelity

After successful persistence Engine.SaveKey selects the exact supplied key through the existing SetSession port. Manager.Save also publishes the successfully stored key as the current foreground selection only after its injected/native synchronous vault write succeeds. Manager tracks stored-session provenance; idempotent SetSession of that same key retains `Stored`, while a different unsaved session key reports `Session only`. Save failure leaves the previous selection unchanged. Startup session/environment/vault precedence remains unchanged, and the parent environment is untouched. No new vault abstraction or public credential API was introduced.

RED: `red.log`, `TestFinalFixPersistedSelection` (real Manager with injected fake vault) and `TestFinalFixEnginePersistedSelection` (Engine with a separate-storage credential implementation matching Manager precedence): successful persistence leaves the old session/environment key selected. Both failure-preserves-selection cases already pass.

GREEN: `green-subsystems.log` and final suite pass both prior sources and vault failure paths through Engine.SaveKey. Subsequent synthetic launch construction and injected discovery receive the selected saved key (or previous key on failure), with no credential in argv or changed parent environment. Real Manager tests with injected vault verify selection and `Stored` provenance plus a later different session-only key. Existing startup precedence and cancellation tests pass. `retained-green.log` records `saved_new=true selected_new=true status=Stored` for both original cases. No credential values are included in this report.

## Final verification

Common settings on all commands:

```sh
GOMAXPROCS=2 GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local
GOCACHE=$REPO/.cache/go-build
GOTMPDIR=$REPO/.cache/go-tmp
TMPDIR=$REPO/.cache/go-tmp
```

Commands (from the isolated worktree; the environment above is supplied to each):

```sh
go test -race -p 2 -count=1 -json ./...
go test -race -p 2 -count=1 -overlay ../native-app/.cache/native-app/final-fix/retained-overlay.json -run '^TestFinalReview' -v ./internal/configtxn ./internal/monitor ./internal/tui ./internal/credentials ./cmd/oneline-ccc
go test -race -p 2 -count=1 -overlay ../native-app/.cache/native-app/final-fix/producer-overlay.json ./cmd/oneline-ccc
go build -trimpath -o ../native-app/.cache/native-app/final-fix/oneline-ccc ./cmd/oneline-ccc
git diff --check
```

All exit 0. Changed Go files pass gofmt; a path allowlist check passes; committed worktree is clean. The final full run has **11 tested packages, 604 passing test/subtest events, 14 skip events, no failures or race reports**. Full output is `final-fix/whole-suite-race.json`; package output:

```text
?   	github.com/clankercode/oneline-ccc	[no test files]
ok  	github.com/clankercode/oneline-ccc/internal/adapters/codex	1.011s
ok  	github.com/clankercode/oneline-ccc/internal/app	3.328s
ok  	github.com/clankercode/oneline-ccc/cmd/oneline-ccc	5.543s
ok  	github.com/clankercode/oneline-ccc/internal/catalog	2.002s
ok  	github.com/clankercode/oneline-ccc/internal/configtxn	6.800s
ok  	github.com/clankercode/oneline-ccc/internal/core	1.025s
ok  	github.com/clankercode/oneline-ccc/internal/credentials	2.025s
ok  	github.com/clankercode/oneline-ccc/internal/codexrpc	17.685s
ok  	github.com/clankercode/oneline-ccc/internal/monitor	7.800s
?   	github.com/clankercode/oneline-ccc/tests/native-catalog	[no test files]
ok  	github.com/clankercode/oneline-ccc/internal/tui	2.726s
ok  	github.com/clankercode/oneline-ccc/tests/native-journey	1.012s
```

Skips: eight native adapter events without explicit native Codex/catalog fixtures; two native application journeys without those fixtures; the unsupported-platform guard on supported Linux; two nonexistent new-file restore staging boundaries; and the Windows observer assertion on Linux. Two packages have no test files. These skips do not establish native adapter, Windows vault, macOS filesystem, native service or production TUI PTY acceptance. Parent owns native API/PTY/platform CI integration; none is claimed from cross-platform compilation or this worker's synthetic tests.

RED command output is retained in `final-fix/red.log`, `red-foreground.log` and `red-runner.log`; targeted GREEN output is in `green-subsystems.log`, `green-foreground-runner.log`, `retained-green.log` and `producer-integration.log`. `whole-suite-summary.txt` retains the exact final skip IDs. Supplemental tuple/range and metadata-error policy cases are in the final whole-suite output.

## Producer/consumer obligations and parent handoff

- Retain `runForeground` around the production terminal Program.Run on every return path. Context cancellation alone is insufficient; the helper's closed gate plus drain is required before main/exit. Backend methods must return only once their protected cleanup has reached its final safe boundary. No timeout may abandon an enrolled native write/rollback.
- `Options.Context` supplies all TUI action lifetimes; nil preserves the prior standalone-model fallback. Direct tui.New embedders needing production lifecycle guarantees must also provide an owning drain boundary. `Options.Runner` is optional for standalone consumers, whose fallback is command.Run; production always injects the owned-child runner. Keep literal argv, explicit Env and child errors intact. Tea owns terminal suspension/restoration; runProcess owns native process trees and foreground handoff. Integrate the already supplied CLI producer before claiming descendant cleanup from this branch.
- Keep all newly added Backend methods enrolled if the port grows. Command construction enrollment ends on construction return; actual child execution is separately enrolled via the runner. Operations scheduled after shutdown closure must not reach a backend or runner.
- Credential Save success followed by SetSession of the same supplied key is an idempotent selection operation. Alternative credential stores should preserve persistence provenance on that same-key selection, as Manager now does. Failed persistence must not select the attempted key. Startup precedence must remain unchanged.
- busctl fixtures/producers must encode array properties directly under `data`, with no invented property wrapper. Each consumer must retain its typed bounds/authority validation.
- Linux metadata query failure is a supported fail-closed outcome; do not relax it to zero flags. Policy bits produce Conflict even during initial preflight, before state creation.
- Keep returned safe error-plus-service-state projections and reconciliation notices; no automatic lifecycle resend.
- Parent handles D1/docs, real native API/PTY/services/platform CI, integration and the prescribed independent final delta review. No integration conflict is expected: all edits are disjoint from CLI producer paths. No remaining assigned finding is intentionally deferred.

Changed paths:

- `cmd/oneline-ccc/foreground_lifecycle_test.go`
- `cmd/oneline-ccc/main.go`
- `cmd/oneline-ccc/signals.go`
- `internal/app/actions.go`
- `internal/app/persisted_selection_test.go`
- `internal/configtxn/inode_flags_linux_test.go`
- `internal/configtxn/metadata_linux.go`
- `internal/credentials/persisted_selection_test.go`
- `internal/credentials/store.go`
- `internal/monitor/lifecycle_test.go`
- `internal/monitor/native_wire_test.go`
- `internal/monitor/systemd.go`
- `internal/tui/async.go`
- `internal/tui/model.go`
- `internal/tui/runner.go`
- `internal/tui/runner_test.go`
- `internal/tui/service_failure_test.go`

## N1 actual-terminal acceptance follow-up — 2026-11-03

Follow-up commit: **`099cda13ca34960547ff37a52574bd7a52f6ac50`** — Test SIGTERM terminal restoration through real TUI handoff. This commit changes **only** `cmd/oneline-ccc/foreground_terminal_lifecycle_linux_test.go`. No production main/signals/TUI file was changed on resumption, and no process/RPC/shared-owner file was edited. Worktree remains clean; no push or merge. Parent already integrated the original pass as 154b35e; integrate this follow-up commit separately.

Controller scope/ruling: this is the original N1 terminal acceptance failure, with the production correction owned by Linnaeus in `native-process-cleanup`. The central owned runner must restore pre-child terminal modes and foreground ownership before returning to Tea. This worker owns the durable actual-TUI/raw-child regression and read-only verification. There is **no competing outer terminal reset or copied process-ownership algorithm**.

### Root cause established from actual Tea code

Read Bubble Tea v2.1.1's exec.go, eventLoop, Program.Run/shutdown and tty initialization/restoration in full. `eventLoop` invokes `p.exec` synchronously; `p.exec` calls the runner, then `RestoreTerminal` before returning to the event loop. Thus the ordinary Program.Run path does not overtake that restoration when runner enrollment ends.

The actual problem is saved-state replacement: `RestoreTerminal` calls `initTerminal`, which calls `MakeRaw` and assigns its returned state to `previousTtyInputState` again. Native Codex forcibly killed while raw cannot clean up. Tea therefore captures the abandoned raw modes as its next restore baseline; final shutdown restores raw modes. The central runner's pre-child snapshot/restoration corrects this baseline before Tea reacquires input.

### Exact RED and durable regression

- Replayed the parent's original `.cache/native-app/pty-final-shutdown.py` with the parent's built artifact and existing disposable fixture. It verified the interactive Codex screen/PID and failed `terminal modes not restored after SIGTERM`. Evidence: `final-fix/pty-shutdown-red.log`. Trial snapshots show canonical flags `4500/5/8a3b` becoming raw `4000/4/a30` (input/output/local flags).
- Added `TestForegroundSIGTERMRestoresTerminalAfterRawChild`. It allocates its own controlling PTY, enables the real Tea renderer, drives the public TUI launch command through the production `runForeground` helper and default runProcess, and starts an explicit-environment synthetic child which sets raw mode and reports its PID. The test verifies raw mode, sends actual SIGTERM to the application process, waits for application cancellation/cleanup and child reaping, then compares the entire native termios snapshot to the original. No operator terminal or configuration is used. The fixture is deliberately not `WithoutRenderer`, because that would skip the native mode handling under test.
- RED on this branch: `terminal-order-red.log`, expected `iflag 500 -> 0, oflag 5 -> 4, lflag 8a3b -> a30` mismatch after verified child reaping.
- RED with the existing CLI producer overlaid: `terminal-order-producer-red.log`, same exact assertion failure. The smaller synthetic PTY begins with different input flags than the parent's UTF-8 shell PTY; both assert exact restoration of their own initial modes.

### GREEN against the central runner snapshot

Read the new `cmd/oneline-ccc/process.go` plus `internal/processowner` implementation from the sibling without editing them. Its Owner snapshots terminal stdin/stdout/stderr before child start. `Close` first terminates owned processes/restores foreground ownership, then restores saved modes. runProcess calls Close before returning to `tea.Exec`. This matches the controller's ownership and ordering ruling.

The producer was still uncommitted at capture (sibling HEAD 154b35e). A private immutable snapshot of **nine files**, with per-file SHA-256, is retained in `final-fix/terminal-owner-snapshot/` and `terminal-owner-manifest.json`; `terminal-owner-overlay.json` composes that source with the previous CLI producer overlay. Fresh byte comparison confirmed zero drift before this report. It is snapshot verification, not a claim about a later producer commit.

- `terminal-order-green.log`: PASS for the durable actual Bubble Tea SIGTERM test, plus producer `TestOwnedRunnerRestoresPreChildTerminalState` normal and SIGTERM cases.
- `terminal-order-repeat-green.log`: durable actual-TUI regression PASS with `-count=10`.
- Built private `final-fix/oneline-ccc-terminal-order` using the snapshot overlay and ran `final-fix/pty-terminal-order.py`. This reuses the parent's namespace-isolated actual Codex fixture/harness with separate binary, socket, transcript and terminal snapshot names. `pty-shutdown-green.log`: **PASS actual TUI SIGTERM during native Codex handoff: child reaped and terminal restored**. The trial's `tty.shutdown.finalfix.before` and `tty.shutdown.finalfix.after` are byte-identical. The parent's artifact, harness sources and original failure evidence are preserved.
- `terminal-order-headless-race.log`: all existing cancellation/drain/closed-enrollment regressions PASS under `-race`.
- Normal and race broad overlay runs tested every existing package successfully: normal **623 pass events/14 skips**, race **621 pass events/14 skips**, no test assertion failures. Their command exits were 1 only because Go tries to chdir to the new virtual overlay package `internal/processowner`, whose physical directory deliberately does not exist in this disjoint worktree. Output retained as `terminal-order-whole-normal.json` and `terminal-order-whole-race.json`; do not describe those command exits as whole-command PASS. Compiled that package's test executable separately under `.cache` using `go test -c -vet=off` and executed it from this real worktree: its two tests PASS both normally and with race instrumentation (`terminal-owner-normal.log`, `terminal-owner-race.log`). Thus all package test assertions represented by the snapshot were executed, but parent must rerun the ordinary complete gate after integrating the actual package directory.
- `go vet ./cmd/oneline-ccc` on the actual local source, gofmt and diff whitespace checks PASS. Overlay commands use `-vet=off` because vet likewise cannot chdir into the nonexistent virtual package; this does not establish fresh integrated shared-package vet proof. Native fixture skips are the same 14 described in the original report; no opt-in native API fixture was silently claimed from these broad runs.

### Exact parent test and integration obligations

Use the original report's GOMAXPROCS=2, offline Go modules and `.cache` environment. After integrating this regression and Linnaeus's shared-runner correction, run on the actual source tree:

```sh
go test -p 2 -count=1 -run '^TestForegroundSIGTERMRestoresTerminalAfterRawChild$' -v ./cmd/oneline-ccc
python3 .cache/native-app/pty-final-shutdown.py
```

Rebuild the parent's `.cache/native-app/oneline-ccc` before the Python command. The local snapshot verification command was:

```sh
go test -vet=off -p 2 -count=1 -overlay ../native-app/.cache/native-app/final-fix/terminal-owner-overlay.json -run '^Test(ForegroundSIGTERMRestoresTerminalAfterRawChild|OwnedRunnerRestoresPreChildTerminalState)$' -v ./cmd/oneline-ccc
```

The new actual-terminal regression is tagged `linux && !race`. An initial forced race-enabled run of this exact renderer/terminal fixture reproducibly exposed a **separate pinned-dependency reader-close race**: Tea's killed shutdown closes cancelreader's signal pipe (`tea.go:1304`, `cancelreader_linux.go:118`) while its read goroutine reads that file descriptor (`cancelreader_linux.go:149`). The application subprocess exits 66 before the mode assertion; full evidence is `terminal-order-reader-race.log`, reproduced by private `terminal-race-probe-overlay.json`. No production code was changed or extra acceptance closure claimed for that dependency race. The ordinary real-PTY gate remains mandatory, and the headless lifecycle suite remains race-enabled. Parent should retain this distinction rather than interpret a race-suite run as execution of the real-terminal test.

The shared runner is a required integration dependency: this regression intentionally remains RED with the old runner. No local outer reset is needed or supplied. Parent retains native Windows/RPC-stage cleanup and final integrated native/PTY/vet gates; this worker makes no new Windows or RPC-cleanup acceptance claim.

=== DONE ===
