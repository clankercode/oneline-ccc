PASS — scoped final delta rereview

Standards: PASS. Spec: PASS for the requested fixes. N1, T1, T2, U2, K1, D1, C1–C4 and the Darwin pipe correction are **ADDRESSED** in the frozen candidate. No new material regression was found in this scope. Final native candidate CI and actual application PTY acceptance remain separate parent-owned gates.

## Frozen scope

- FIX_BASE: `da770d96303a7c8f4b956516720a58fbf9b1e6d1`.
- Candidate: `154b35eaabd79518a7a53b781ab1e9865aabd572`.
- CLI integration: `b9ca80da34225fbdeb2601060b934f16845e0fd1`.
- Darwin pipe producer/integration: `76c17c554a782ce2a86dd56c231b3f4a50b10852` / `84b5a9dff05775d1381b48c9f2c5e41fc3ecb399`.
- Finalfix producer/integration: `419b0cf83efff128f15761daa42186972e51fea9` / `154b35eaabd79518a7a53b781ab1e9865aabd572`. All 17 producer paths match the integrated source; no difference on those paths.
- Documentation delta includes controller commit `67f1df7`, following the documentation already committed at `da770d9`.

Reviewed the 33-file delta `da770d9..154b35e` only, plus the existing interfaces needed to assess these fixes. Source was frozen with `git archive` at 2026-11-02 12:38:20 UTC into `final-rereview-frozen-d21ccg03/source`. A final byte comparison of all 206 archived files found no changes; the live worktree also remained clean at the candidate SHA. Snapshot, delta and integrity manifest are retained beside that source.

Read `final-fix-report.md` fully, its brief, the original five whole-branch probes, and Linnaeus's `cli-rereview.md` including the narrow pipe addendum and retained independent probes. Read the changed production seams in full: command entry/lifetime/signals, output/process/platform ownership, TUI model/async/runner, Linux metadata policy, systemd wire decoding, app credential selection and credential manager. Assessed their changed regressions, the Darwin ACL fixture correction and the four documentation files. Inspected Bubble Tea v2.1.1's terminal execution/suspension implementation to check the adapter boundary.

This is the prescribed single delta review, not another whole-branch review. No original Curie monitor authority probes were repeated. No source changes, subagents, operator configuration/credential reads, real vault/service/client calls, public network or inference were used. Synthetic child probes use test executables, explicit environments, private temporary fixtures and ephemeral loopback listeners. Only the prescribed report and private cache evidence were written.

## Standards — PASS

The production TUI uses the CLI's existing process runner through one narrow adapter, retaining a single platform ownership algorithm. Root cancellation, admission closure and safe draining are explicit at the production exit boundary. The mutex protects both WaitGroup enrollment and shutdown closure. Native metadata uncertainty fails closed; typed protocol evidence remains bounded and validated. Credential publication follows successful persistence; output projection copies backend slices and preserves its separate identity/presentation contracts. No new standards violations were found. `git diff da770d9 154b35e --check` passes.

## Spec — dispositions

| ID | Disposition | Fresh candidate evidence and assessment |
| --- | --- | --- |
| N1 | **ADDRESSED** | Production `runForeground` owns cancellation and draining, wraps every Backend method and separately enrolls the complete child runner lifetime. Adapted original Apply barrier runs actual headless Tea through this production boundary: Tea returns after root cancellation, Apply's context is canceled, and `runForeground` stays blocked until the backend cleanup barrier is released and its safe outcome recorded. Committed regressions additionally cover persistent-key and confirmed-service work and refuse all eight Backend methods during/after closure. |
| T1 | **ADDRESSED** | Original disposable-file nodump probe now returns `Conflict` and retains flags `0x40 -> 0x40`. Linux metadata checking queries inode flags, allows filesystem representation bits only, refuses user-policy/unknown bits, and refuses unavailable flag queries. Committed regressions cover initial refusal, reviewed drift, staged drift and recovery refusal while retaining inode/flags/bytes/backup. |
| T2 | **ADDRESSED** | Original direct-array `UnitPath` and `ExecStart` probes both return nil error. `busProperty` preserves the entire typed data array; consumers decode direct string arrays and direct ten-field structs. Committed regressions pass for single/multiple paths and exact commands and reject invented wrappers, wrong signatures, malformed/nested records, nulls and invalid numeric field types/ranges. No actual manager/service was queried for this review. |
| U2 | **ADDRESSED** | Original failure projection retains and displays each returned `Unknown`, `Conflict` and `Unavailable`. Committed tests cover prior `Running` and `Stopped`, retain the failure and reconciliation guidance, make exactly one lifecycle call and ignore invented statuses. |
| K1 | **ADDRESSED** | Original saved-key probe selects the newly explicit saved key over both previous session and environment values, reporting `Stored`. App SaveKey selects the key after successful Save even for a store with separate storage/selection. Manager publishes after successful vault write and preserves stored provenance on idempotent SetSession. Committed tests verify failure preservation, startup precedence, unchanged environment and delivery of the selected synthetic key to actual constructed child environments and injected discovery. No real vault was exercised. |
| D1 | **ADDRESSED** | README now describes the implemented native foundation and links to the adoption trial and remaining production plan. Plans index records current integration status. Native usage includes cgo-disabled Linux/macOS commands, literal PowerShell build/run commands, no separate language-runtime claim and the macOS system-library qualification. Verification text distinguishes Linux PTY, native CI, fake observer evidence and remaining product gates. All 21 relative links in the four scoped docs resolve. |
| C1 | **ADDRESSED** | Retained independent descendant probes pass afresh for both plain exec.Command and CommandContext: cancellation returns only after the direct child is reaped, and a descendant cannot execute/write after return. New combined probe launches through actual headless Tea, `runForeground` and its default runProcess, proving the TUI reaches the same owned-process cleanup path. POSIX group ownership and Windows suspended-start/Job assignment/drain source remain integrated. Native interactive acceptance is separately pending below. |
| C2 | **ADDRESSED** | Independent plain/JSON status and doctor probes with raw, ANSI-containing and Unicode-format-containing private paths pass. Diagnostics and their private patterns are normalized consistently before replacement; backend snapshots remain unchanged. |
| C3 | **ADDRESSED** | Independent JSON Unicode model probe retains the exact ZWJ-containing model identity. Terminal-only normalization is separate from JSON identity projection; exact known private substrings remain redacted. |
| C4 | **ADDRESSED** | Independent first/middle/last output failure and short-write probes pass for status and doctor: the first error is returned, the underlying writer receives no later call, and CLI status is nonzero. |
| Darwin pipe correction | **ADDRESSED** | Candidate character-device guard prevents pipe async-I/O ownership from being mistaken for terminal foreground ownership. Fresh independent modeled ioctl probe passes for zero and caller-group async owners; committed held-open real-pipe regression also passes. Child READY assertions remain intact. Source is consistent with the Apple XNU pipe_ioctl evidence cited in cli-report.md. Local modeling is not native Darwin execution. |

### Production shutdown and TUI handoff details

Relevant frozen source: `cmd/oneline-ccc/main.go:35,47`, `cmd/oneline-ccc/signals.go:42,77,96,105`, `internal/tui/model.go:180`, `internal/tui/async.go:11` and `internal/tui/runner.go:18`.

N1's assertion belongs at `runForeground`, which encloses Tea and precedes main's exit. The adapted probe deliberately permits `tea.Program.Run` to return first; it requires cancellation and safe backend completion before the production helper returns. Demanding that the standalone Tea library itself drain backend work would test the wrong boundary. The committed admission test also proves shutdown cannot race new WaitGroup enrollment or reopen admission.

Tea's Exec releases terminal handling before calling the adapter and restores it afterward. The adapter retains the backend command, literal argv, explicit environment, assigned streams, operation context and child error. Tea supplies its original file-backed input; its cancellation reader is separate, so the production runner retains the terminal FD needed for foreground ownership. The shared runner restores the parent foreground group before Tea resumes terminal handling. The committed adapter regression preserves child exit code 23. The new combined probe uses the production default runner, not an injected imitation. It proves ownership/reaping/draining in a headless Tea session; terminal modes and real interactive handoff remain parent PTY evidence.

Other correction anchors: `internal/configtxn/metadata_linux.go:11,20`; `internal/monitor/systemd.go:20,37,69`; `internal/tui/async.go:45`; `internal/app/actions.go:102`; `internal/credentials/store.go:78,90,114`; `cmd/oneline-ccc/process.go:54`; `cmd/oneline-ccc/process_tree_posix.go:24,66`; `cmd/oneline-ccc/process_tree_windows.go:22,45,107`; `cmd/oneline-ccc/output.go:32,100,135`; `README.md:209`; `docs/native-app.md:10,22`.

## Exact verification

Fresh reviewer execution used Go **1.26.1 linux/amd64** on Linux x86_64. For every Go invocation:

```text
GOMAXPROCS=2 GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local
GOCACHE=$REPO/.cache/go-build
GOTMPDIR=$REPO/.cache/go-tmp
TMPDIR=$REPO/.cache/go-tmp
```

Inherited `ONELINE_*` opt-ins were removed. Tests explicitly set only synthetic fixture inputs. Native Codex/vault integration tests requiring those opt-ins were outside the selected tests; none is silently counted as native execution.

Executed from the frozen `source/` directory, where `$REVIEW` below denotes `$REPO/.cache/native-app/final-rereview-frozen-d21ccg03`:

```sh
go test -race -p 2 -overlay "$REVIEW/probes/overlay.json"   ./cmd/oneline-ccc ./internal/configtxn ./internal/monitor ./internal/tui ./internal/credentials   -run '^(TestFinalRereview|TestFinalReview|TestIndependentCLI).*' -count=1 -timeout=60s -v

go test -race -p 2   ./cmd/oneline-ccc ./internal/app ./internal/configtxn ./internal/credentials ./internal/monitor ./internal/tui   -run '^(TestFinalFix|TestCLIReview|TestTerminalIdentityNormalization|TestPOSIX|TestRunProcess|TestCancelledLaunch|TestNativeTUI|TestNativeDirectCLI|TestRawTUI|TestTUISignalSelection|TestDirectLaunch|TestDirectChild|TestMonitorExactCommand).*'   -count=1 -timeout=90s -json

go test -race -p 2 -overlay "$REVIEW/probes/pipe-overlay.json" ./cmd/oneline-ccc   -run '^TestIndependentDarwinPipeIoctlModel$' -count=1 -timeout=30s -v
```

| Fresh check | Result |
| --- | --- |
| Original/adapted whole-branch probes plus retained independent CLI probes | PASS; 11 top-level test events including the inert process-fixture entry, 20 total test/subtest pass events, five packages; no failure or race report |
| Focused committed fix and adjacent CLI regressions | PASS; 34 top-level / 89 total test/subtest pass events, six packages, no skips or race report; package elapsed times CLI 5.590s, app 1.107s, configtxn 1.153s, credentials 1.011s, monitor 1.112s, TUI 1.054s |
| Independent Darwin pipe ioctl model on candidate-derived overlay | PASS; 1.012s; source substitution replaces only the ioctl call for deterministic modeling |
| Archive integrity / documentation links / diff whitespace | PASS; 206 files unchanged, 21 relative links resolve, no whitespace errors |

Evidence is retained in `final-rereview-frozen-d21ccg03/`: `snapshot.json`, `delta.diff`, `producer-equality.json`, `integrity-and-docs.json`, `environment.json`, `regression-command.json`, `regressions.jsonl`, `regression-summary.json`, and `probes/` containing sources, SHA-256 manifest, overlays, exact command manifests, `probes.log` and `pipe-model.log`. T1/T2/U2/K1 and independent CLI probe sources are reused unchanged. N1 is adapted as described above; the combined production TUI runner test is new.

## Parent evidence and remaining acceptance boundary

The controller reports candidate `154b35e` passed its actual native-fixture suite, race checks, vet, static builds and nine preview cases. These are parent-provided results, not this reviewer's executions or a final counted transcript. The controller also reports native CI **41445534390 at 84b5a9d** passed all nine jobs (three native OS suites plus six binary builds). That run predates the finalfix and is not final-candidate CI proof. Candidate `154b35e` native CI was pushed and remains parent-owned; no result was supplied at this report's cutoff.

For actual application PTY checks, the controller reports setup/apply/receipt, normal quit and terminal-mode restoration pass, and real Codex launch/Ctrl+C returns to the dashboard. Its stricter shutdown harness encountered an external controlling-PTY PID-namespace mapping problem (`tcsetpgrp(0)` / ESRCH); the controller is moving controlling-PTY allocation inside the namespace with util-linux script without relaxing product checks. This review does not classify that harness observation as a product regression and does not use earlier harness passes as native C1 acceptance. Final strict PTY results remain pending with the controller.

No new material regressions. All requested fix IDs are addressed by scoped source assessment and fresh synthetic evidence; the parent retains final platform/interactive acceptance and ledger updates.

=== DONE ===
