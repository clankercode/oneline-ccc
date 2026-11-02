PASS — narrow acceptance delta

**Standards: PASS. Spec: PASS for this acceptance delta.** The terminal-state acceptance failure and validation-stage process leak are addressed. No new material regression was found. Native CI on the changed candidate now passes, including the previously failing Windows catalog/engine staging paths. The known pinned-dependency reader-close race remains an explicitly disclosed limitation; this verdict does not claim race-free rendered terminal shutdown or native Windows/macOS interactive-console acceptance.

## Exact scope and preservation

- Base: `154b35eaabd79518a7a53b781ab1e9865aabd572`.
- Frozen candidate: `94feea622814bc677cf95267096ac4872bce830e`.
- Production producer `1c5c455d327d1fad0831f18e562b1ae2e064b76b`, integrated as `bbfbe43456c93a11b377e151830b0257ad5d2076`.
- Regression producer `099cda13ca34960547ff37a52574bd7a52f6ac50`, integrated as `94feea622814bc677cf95267096ac4872bce830e`.

Frozen with `git archive` at 2026-11-02 13:17:12 UTC into `.cache/native-app/acceptance-rereview-frozen-l2mq93e9/source`. Reviewed only the 19-file acceptance delta. The supplied `acceptance-review.diff` has identical changed content/file identities to the frozen Git delta; only surrounding diff context differs. Both integrations match their producers on their changed paths. A final comparison found all 213 archived files unchanged; the scoped diff whitespace check passes.

Read `process-cleanup-brief.md`, `process-cleanup-report.md`, and the complete actual-terminal follow-up in Avicenna's `final-fix-report.md`. Read all changed production files and regressions, the moved POSIX/Windows ownership policies, and the small dependency/platform seams needed to assess their contracts. Applied the code-review skill's Standards/Spec axes locally, respecting the no-subagent instruction.

Source, index and branches were not modified. Controller site/docs edits remained outside the code freeze and were preserved. The current draft verification document was read only to assess its limitation disclosures; it was captured separately as documentation evidence. No operator configuration, credentials, services, external network, inference or real client invocation was used by this reviewer. No package-wide suite, native gate or additional probe was run: source inspection, existing deterministic RED/GREEN evidence and the completed parent gates answered the review questions.

## Standards — PASS

The extraction puts the existing group/Job algorithms in one internal owner reused by CLI, RPC and version probes. It adds the terminal snapshot at that same ownership boundary without a competing outer TUI reset. The lifecycle distinguishes Prepare/Start, caller-owned Wait, termination and final cleanup. Existing context/environment/argv/exit handling and pre-start failure cleanup are retained. The new staging object keeps identity, deletion policy and redacted cleanup projection together. No documented-standard violation or material design smell was found in this delta.

## Spec — addressed findings

| Acceptance concern | Disposition | Assessment and evidence |
| --- | --- | --- |
| Raw Codex + actual SIGTERM leaves the terminal raw (N1/C1 acceptance extension) | **ADDRESSED** | Owner snapshots terminal file descriptors before child start. Close terminates owned processes, restores foreground ownership, then restores the exact saved modes before returning to Tea. The real-renderer regression verifies raw child readiness, sends actual SIGTERM to the application, requires child reaping, and compares the entire initial/final termios. Parent's fresh isolated native Codex SIGTERM journey now passes exact original termios and child cleanup. |
| Validation RPC/version descendants survive direct process exit and can retain Windows staging CWD/handles | **ADDRESSED** | CLI, Run and Version all use the shared owner. RPC closes stdin, permits its existing one-second graceful direct exit, terminates when necessary, always closes descendant ownership even after clean exit, cancels scanner delivery, closes read pipes and joins both reader goroutines before returning. Version also closes descendant ownership after Wait and retains its five-second timeout, output cap, exact identity and 100ms inherited-pipe wait bound. Synthetic success/refusal/cancel and version helper-liveness regressions pass. Changed-candidate native Windows catalog/engine cleanup now passes in CI 41451803057. |
| Private-stage identity and truthful cleanup failure | **ADDRESSED** | Allocation captures identity through File.Stat while the directory handle is open, then immediately closes the handle. Cleanup requires the same directory and private policy before a single RemoveAll. Replacements are preserved. Cleanup failures use fixed redacted text; Inspect returns error and loses Supported, and edit discards returned reviewed bytes. The substitution regression demonstrates refusal/preservation and absence of private path text. No blind sharing retry or native assertion weakening was introduced. |
| Preservation of CLI/ownership behavior | **PRESERVED** | POSIX owned groups, actual-character-device guard, foreign-session/group refusal, foreground restoration and SIGTTOU policy were moved with their tests. Windows retains suspended start, assignment/resume cancellation barrier, kill-on-close Job policy, unchanged console process-group flags and bounded active-process drain. CLI signature/stream/explicit environment/cancellation and exit projection remain intact. Existing regressions execute in the parent's normal/race gates; all three native OS suites now pass. |
| Durable regression coverage | **ADDRESSED, with disclosed race boundary** | Normal tests run the real rendered Tea/SIGTERM regression. Lower-level terminal restoration runs normal and SIGTERM cases under normal and race instrumentation. Headless production shutdown/drain/admission remains race-tested. The inherited-pipe fixture was strengthened from waiting for a leaked helper to proving that a genuinely ready helper cannot execute after return; its retained pre-fix overlay produces the expected failure. |

### Terminal restoration ordering

`internal/processowner/owner.go:28,56,74` and `cmd/oneline-ccc/process.go:56` establish the pre-child snapshot and final restoration. The shared platform owners are at `internal/processowner/owner_posix.go:24,66` and `owner_windows.go:22,45,107`.

The pinned Tea v2.1.1 implementation executes the handoff synchronously: release terminal, call adapter Run, then RestoreTerminal. Its RestoreTerminal reinitializes input and saves the state returned by MakeRaw as the next baseline. Restoring pre-child modes in Owner.Close therefore prevents Tea from adopting the killed child's abandoned raw state. This is the correct producer boundary. No production main/signals/TUI restoration algorithm was added. The actual-renderer test is `cmd/oneline-ccc/foreground_terminal_lifecycle_linux_test.go:74`; lower-level normal/SIGTERM coverage is `process_tree_terminal_linux_test.go:56`.

### Process drain, stage identity and failure projection

Relevant seams are `internal/codexrpc/client.go:39,103`, `internal/codexrpc/program.go:100`, `internal/adapters/codex/staging.go:18,47`, `internal/adapters/codex/adapter.go:29` and `internal/adapters/codex/stage.go:93`.

The old Windows sharing symptom did not identify its original lock holder conclusively. The deterministic helper/CWD reproduction establishes a real ownership leak capable of causing it, and the changed-candidate native rerun supplies the missing acceptance proof. Linux successful unlink alone was correctly not used to certify Windows cleanup: helper liveness is checked independently of directory removal. Windows Job accounting/handle closure precedes adapter stage deletion.

The staging snapshot's Windows identity detail is substantiated by the local Go 1.26.1 implementation: File.Stat obtains file identity from the open handle; Lstat can defer identity lookup until SameFile. The retained snapshot is not re-resolved through a replaced path. Current identity/private-policy failures preserve the replacement and return reconciliation guidance. This is not a promise against arbitrary hostile same-account path races; no such stronger contract was added. A failed deletion may leave owned staging to reconcile and cannot return reviewed success bytes.

## Evidence assessed

Reviewer work consists of frozen source comparison, complete scoped code/regression inspection and assessment of the retained evidence below. Test executions are attributed to their producers or the controller, not claimed as fresh reviewer runs.

| Evidence | Result and attribution |
| --- | --- |
| `process-cleanup-red.log` | Producer expected RED: RPC helper survives success/refusal/cancel; version helper survives; substituted stage deleted; normal/SIGTERM child leaves raw state. |
| `process-cleanup-green.log` | Producer focused GREEN for shared policy, helper cleanup, substitution preservation and terminal restoration. |
| `process-cleanup-version-red.log` | Strengthened inherited-pipe test fails specifically on post-return helper liveness with old program.go. |
| `process-cleanup-race.log`, `process-cleanup-vet.log`, `process-cleanup-evidence/builds.log` | Producer affected-package race/vet and six target compilations pass. Compilation is separate from runtime acceptance. |
| `final-fix/terminal-order-green.log`, `terminal-order-repeat-green.log` | Regression producer real-renderer test and lower-level normal/SIGTERM tests pass; real-renderer regression also passes ten repetitions. Snapshot/overlay evidence, preceding final integration. |
| `final-fix/terminal-order-headless-race.log`, `pty-shutdown-green.log` | Headless production drain/admission passes under race; producer's isolated actual Codex SIGTERM overlay journey passes. |
| `final-integrated-normal.json` at `94feea6` | Parent gate, parsed by reviewer: **643 passing test/subtest events, 4 test skip events, 12 tested packages, no failures**. Includes real-renderer SIGTERM, stage substitution, RPC/version helper cleanup and the native engine journey. |
| `final-integrated-race.json` at `94feea6` | Parent gate, parsed by reviewer: **641 passing test/subtest events, 4 test skip events, 12 tested packages, no failures**. Headless shutdown and process ownership execute; rendered test and its helper are excluded at compilation by `!race`. |
| `final-pty-journey.log`, `final-pty-shutdown.log` | Read parent logs: fresh native PTY setup/help/palette/masked key/review/apply/receipt/resize PASS; actual SIGTERM during native Codex handoff PASS, child reaped and terminal restored. Controller separately confirms launch/Ctrl+C/dashboard/quit behavior and exact original termios. |
| Parent vet/static build/nine preview states/diff check | Controller reports PASS for the same candidate. Not rerun here. |
| Native CI **41451803057** at **94feea6** | Controller-supplied final result: **COMPLETED SUCCESS, all nine jobs** — real Windows/macOS/Linux native fixtures and vet, POSIX race and six OS/architecture cgo-disabled builds. Controller has `gh run view` evidence. This reviewer did not fetch CI. It is the changed candidate, not a third unchanged retry. Prior Windows catalog/engine staging cleanup failures no longer occur. |

The four test skips in each integrated gate are the unsupported-platform test, two inapplicable rollback-boundary subcases, and Windows-only unavailable-observer behavior on Linux. Two additional package skip events mean no test files; they are not extra skipped test cases. The rendered test's race exclusion is not represented by a skip event. Avicenna's earlier broad overlay commands exited 1 because the virtual package directory was absent; those exits are not counted as successful whole commands. The parent's ordinary integrated gates supersede that integration limitation.

## Known limitation and remaining native proof

`foreground_terminal_lifecycle_linux_test.go:1` explicitly uses `linux && !race`. The retained `final-fix/terminal-order-reader-race.log` reports subprocess exit 66 and a real reader-close race between cancelreader's signal-pipe Close and its concurrent Fd lookup during Tea killed shutdown. Local dependency source confirms Tea skips waiting for its input loop on that kill path. This is an unresolved dependency concurrency issue, not proof that only the test harness is faulty, and the cleanup change does not fix it.

For this declared development scope, the normal rendered SIGTERM/actual Codex acceptance gates and the separate race-tested ownership/headless gates accurately establish their respective contracts. They do not establish race-free rendered shutdown. The current controller documentation draft explicitly discloses the pinned Tea/cancelreader issue and that a race-suite pass does not execute this rendered regression. That disclosure is consistent with the evidence and must survive final documentation closure.

All supplied changed-candidate platform CI gates are now green. Actual Windows/macOS interactive-console terminal behavior remains outside the Linux PTY proof and noninteractive CI fixtures; the verification draft retains that limit. Deliberately detached POSIX groups/sessions remain outside the existing inherited-group policy. No new Windows/macOS interactive-console or wider production claim follows from this review.

Review artifacts are under `.cache/native-app/acceptance-rereview-frozen-l2mq93e9/`: exact source, `snapshot.json`, `delta.diff`, supplied brief/reports/package, `package-and-producers.json`, `integrity.json`, and `parent-evidence/` containing immutable reviewed gate logs, parsed summaries, documentation snapshot and SHA-256 manifest. No source/index/branch mutation was made.

Standards: 0 new findings, PASS. Spec: 0 new material findings in this delta, PASS; the disclosed dependency race and native interactive-console limits remain open as stated above.

=== DONE ===
