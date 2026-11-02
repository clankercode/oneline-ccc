# Validation-process cleanup and terminal-state handoff

Commit: `1c5c455d327d1fad0831f18e562b1ae2e064b76b` — `Own validation subprocess trees and restore child terminal state`.
Branch/worktree: `fix/native-process-cleanup`, `.worktrees/native-process-cleanup`; base `154b35eaabd79518a7a53b781ab1e9865aabd572`.

Implementation, local RED → GREEN, scoped race checks, vet and six-platform builds are complete. No parent integration, push/merge/rebase, actual Windows/macOS runtime, real Codex or native TUI acceptance claim is made. Parent native gates remain required.

## Diagnosis and evidence

Read both supplied native Windows failure logs completely. CI 41446760137 on the same base repeatedly passes functional assertions but fails TempDir cleanup on randomly named codex-stage directories: first Standard catalog validation; second FullTier/Standard catalog and native engine apply/rerun. This establishes repeated native sharing failures, not the identity of the process holding the directory. No unchanged CI rerun was requested or performed here.

Source traced two concrete seams: codexrpc Run waited/killed only its direct process, and adapter Inspect/edit discarded deferred RemoveAll failures. Version probes also lacked descendant ownership. A deterministic synthetic RPC fixture spawns a native helper which retains the stage as its CWD and answers a liveness socket. The direct RPC process exits independently. RED: Run success, validation refusal and cancellation all return while that helper still executes. A version probe reproduces the same leak. A held CWD is a Windows stage-removal obstruction; Linux alone allows unlinking it, so the portable test additionally asserts post-return helper liveness rather than mistaking Linux deletion success for proof.

This confirms an owned-process leak capable of producing the supplied sharing symptom. It does not prove that the real Codex process holding the directory was that exact helper class rather than another native handle/transient OS actor. The native Windows rerun is the causal acceptance gate; no OS sharing retries or unexplained sleeps were added to hide it.

Separate RED fixtures establish that cleanup could silently delete a substituted staging directory and that a raw-mode child leaves terminal modes raw after ordinary return or SIGTERM. The latter directly matches the controller's terminal-state ownership ruling: restore in the owned runner before Tea captures a new baseline.

## Changes

- Extracted the existing POSIX process-group and Windows suspended-start kill-on-close Job implementation into `internal/processowner`, with its platform policy tests. The algorithms were moved and reused, not forked. CLI runProcess retains its signature, stream/environment forwarding, exit projection, context policy and two-second pipe drain.
- Shared Owner Prepare/Start/Terminate/Close owns setup and failed-start cleanup. Windows assignment/resume and cancellation-ready barrier remain intact, with no new console process-group flags. Job active-process accounting keeps its existing five-second bound. POSIX retains the actual-character-device guard, foreign group/session refusal, foreground transfer/restoration and SIGTTOU handling.
- Owner snapshots real terminal descriptors before starting the child, including separate Windows input/output console modes. Close stops owned descendants, restores foreground ownership, then restores the saved modes before returning to Tea. It restores the exact pre-child baseline rather than imposing an arbitrary global terminal mode. No competing main/signals/TUI restoration implementation was added.
- RPC Run and Version reuse that owner. Run closes stdin and allows the existing one-second graceful direct exit, terminates the owned tree when needed, completes descendant cleanup even after a clean direct exit, cancels scanner delivery, closes/drains both reader goroutines, and returns fixed redacted cleanup errors. Pipe handles are also closed on setup/start errors. Version retains its output limit, exact identity checks and bounded pipe draining while adding descendant cleanup.
- Adapter stage allocation retains an identity captured by File.Stat on an opened-and-immediately-closed directory. This avoids Windows Lstat's lazy identity lookup and does not itself keep a stage handle open during validation. Cleanup verifies the original directory identity/private policy and performs one RemoveAll. Substituted directories are preserved; cleanup errors are fixed/redacted and propagated from Inspect/edit. Inspect loses Supported on cleanup failure; edit returns no reviewed bytes when its cleanup fails. No blind deletion retry.

## Verification already run

All executions used synthetic subprocesses/private fixtures only. No operator config/keys/vault/services/inference, real native Codex, subagents or external endpoints. The ephemeral loopback listener is solely a deterministic fixture liveness barrier.

Environment:

```sh
GOMAXPROCS=2 GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local
GOTMPDIR=$REPO/.cache/go-tmp
TMPDIR=$REPO/.cache/go-tmp
GOCACHE=$REPO/.cache/go-build
```

1. Baseline `go test -p 2 ./internal/codexrpc ./internal/adapters/codex ./cmd/oneline-ccc`: PASS before source changes.
2. New regressions against unchanged base: expected FAIL for RPC success/refusal/cancel descendants, Version descendants, substituted-stage cleanup, and terminal-state restoration after normal return/SIGTERM. `process-cleanup-red.log` retains output.
3. Focused GREEN, including moved POSIX policy/pipe tests: PASS. `process-cleanup-green.log`.
4. The first complete race run exposed an obsolete synthetic-test teardown: TestVersionBoundsInheritedPipes expected the leaked helper to finish naturally and write a marker, incompatible with terminating it. Replaced that teardown with deterministic active-helper readiness, the original cancellation/output bounds, post-return no-liveness assertion and immediate stage removal. No native tests were changed, skipped or weakened. An overlay of the original program.go then makes this stronger test fail specifically because the helper remains active (`process-cleanup-version-red.log`, 0.105s).
5. Final complete affected-package race run, same source as committed:

```sh
go test -race -p 2 ./internal/processowner ./internal/codexrpc \
  ./internal/adapters/codex ./cmd/oneline-ccc -count=1 -timeout=90s
```

All PASS: processowner 1.011s; codexrpc 20.525s; adapter 2.039s; CLI 10.647s. This includes prior C1–C4, native synthetic SIGINT/SIGTERM/exit/stream/environment regressions, existing Linux PTY foreground restoration, and the new normal/SIGTERM raw-child state restoration. Log: `process-cleanup-race.log`.

6. `go vet -p 2` on those four packages: PASS, `process-cleanup-vet.log`.
7. With CGO_ENABLED=0, all four affected package test binaries compile and the CLI builds for **Linux/Windows/Darwin × amd64/arm64**. Each command used `-p 2`; all 12 steps exit zero. `process-cleanup-evidence/builds.log` records the six targets. This is compilation, not runtime proof.
8. Final scoped/staged diff checks passed; no source edits occurred after the successful final checks. Tests/builds were not repeated simply for this handoff.

Native adapter/catalog and journey fixtures were preserved. They require parent-supplied isolated native inputs and are not counted as locally executed native evidence. Version identity/output-limit/cancellation protections remain covered; the stronger inherited-pipe test is not a skip.

## Paths / ownership

Only authorized CLI process files, codexrpc, adapter and the extracted shared ownership module changed:

```text
M	cmd/oneline-ccc/process.go
D	cmd/oneline-ccc/process_tree_other.go
A	cmd/oneline-ccc/process_tree_terminal_linux_test.go
M	internal/adapters/codex/adapter.go
A	internal/adapters/codex/cleanup_test.go
M	internal/adapters/codex/stage.go
A	internal/adapters/codex/staging.go
A	internal/codexrpc/cleanup_test.go
M	internal/codexrpc/client.go
M	internal/codexrpc/program.go
M	internal/codexrpc/program_test.go
A	internal/processowner/owner.go
A	internal/processowner/owner_other.go
R087	cmd/oneline-ccc/process_tree_posix.go	internal/processowner/owner_posix.go
R083	cmd/oneline-ccc/process_tree_posix_test.go	internal/processowner/owner_posix_test.go
A	internal/processowner/owner_test.go
R090	cmd/oneline-ccc/process_tree_windows.go	internal/processowner/owner_windows.go
R082	cmd/oneline-ccc/process_tree_windows_test.go	internal/processowner/owner_windows_test.go
```

Avicenna's main.go/signals.go/TUI and new lifecycle regression files are untouched. Public app/backend and adapter configuration schemas are unchanged; no go.mod/go.sum changes. The stage allocator is private to the adapter.

## Integration obligations and remaining gates

- Parent should integrate this one commit. There are no unstaged implementation changes to carry separately. Shared ownership now lives in internal/processowner; platform policy tests moved there. CLI runProcess remains the integration API, so Avicenna's main/lifecycle work need not fork it or add a second terminal restoration algorithm.
- Avicenna owns the actual TUI/PTY journey: raw child + SIGTERM, canonical terminal restoration before Tea resume/exit, and no child execution after shutdown. The new low-level synthetic PTY tests prove runner behavior but do not replace that production journey. Any old outer restore logic must not overwrite the runner's restored Tea baseline.
- Parent must run the new commit on native Windows, including nested hosted-runner Job compatibility, adapter Standard/full-tier catalog and native engine review/apply/rerun cleanup. The new synthetic helper/CWD and stage-substitution tests must execute too. Run the changed candidate; do not retry the old one or weaken TempDir cleanup. If native sharing persists, fixed cleanup errors now surface at the adapter boundary rather than being silently ignored.
- Native macOS should execute the moved pipe/group policy and existing direct signal tests and the integrated runner. Crosscompilation is not native proof. Windows console mode restoration is implemented through the existing cross-platform terminal library but was not exercised on a native console here.
- POSIX ownership covers inherited process groups, not deliberately detached sessions/groups; orphan reaping belongs to the OS. Windows uses the existing exact Job policy. Cleanup does not claim protection against arbitrary hostile same-account directory races, and a deletion failure can leave part of an owned staging tree for reconciliation. Kernel process-wait behavior is subject to the OS; added application timers/Job polling are bounded, with no unbounded cleanup retry loop.

No local execution blockers remain. Remaining blockers to native acceptance are parent integration/native CI and actual TUI/PTY verification. The isolated branch/worktree is preserved for review.

=== DONE ===
