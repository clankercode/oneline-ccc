FAIL

Standards: PASS. Spec: FAIL. Five material issues remain in the frozen integration: new shutdown finding N1 (P1), carried transaction finding T1 (P2), controller-discovered and independently reproduced protocol finding T2 (P2), and new integration findings U2 and K1 (both P2). The separately assigned CLI C1–C4 remain outside closure pending their final delta. D1 is a minor documentation correction, not an additional release gate.

## Frozen scope and documentation

Reviewed baseline `5769d423bbe5c698c5ef559da5d17204e23f99ce` through candidate **`da770d96303a7c8f4b956516720a58fbf9b1e6d1`**. At capture, the working tree was clean: the controller's README/site and `docs/native-app.md` changes had just been committed as da770d9. They are included, not treated as absent. Production source was frozen with `git archive` under `.cache/native-app/final-review-frozen-jspclrqc/source`. Final byte comparison against that archive checked all 190 regular files with zero differences; see `source-integrity.json`. The initial manifest and frozen branch diff are beside it.

The explicitly supplied documentation follow-ups were captured separately in `final-review-frozen-jspclrqc/documentation-addendum/`: `docs/native-app.md`, `docs/plans/README.md`, `docs/verification/native-foundation.md`, README and both site pages. Its manifest records each SHA-256, the initial documentation HEAD da770d9, later capture HEAD `468e886d402459107180ece040e5fd8886fe3df7`, and supplied review-package digest `d8e2c63ec899dabe8712595fde9aa075a242890370aec16b37a22751b5ceecab`. These documentation copies include the uncommitted controller corrections. Production conclusions remain pinned to da770d9; later CLI/test and Darwin fixture changes are not silently incorporated.

Authority: native foundation, production plan, usability contract and native usage document; consulted compatibility/platform and stage-zero evidence, transaction/observer contracts, progress ledger and all prior subsystem review/rereview dispositions. The broad production plan is not the claimed deliverable. No subagents, source fixes, commits, pushes, attention tools, operator configuration, real vaults/services, gateways, public network or inference were used. All probes use synthetic data, scratch roots and Go overlays; only the prescribed report and private review/cache artifacts were written.

## Standards assessment — PASS

No additional actionable documented coding-standard breach found. Application authorization, native syntax, protected I/O, credentials/catalog, service lifecycle, and terminal presentation have identifiable boundaries. The repeated credential-reference denylist is deliberate, documented and checked by a cross-package contract test. No stylistic refactor is requested merely from a smell heuristic. The issues below are concrete contract/behavior failures, independently assessed from style.

## Spec findings

### N1 — P1: Root shutdown neither cancels nor drains active TUI mutations

**Location:** `internal/tui/model.go:182`; `cmd/oneline-ccc/main.go:35`; `cmd/oneline-ccc/signals.go:39` (rootBoundBackend only overrides Command).

**Trigger/effect:** SIGTERM cancels the root passed to Bubble Tea while a reviewed Apply, credential save or lifecycle mutation is running. `begin` created that operation from `context.Background()`. Root binding covers child Command only. Bubble Tea does not wait for ordinary Cmd goroutines on shutdown, and the production runUI immediately returns its result; main subsequently exits. Thus an active configuration transaction can be killed before cancellation/rollback completes, and other mutations can outlive UI shutdown. This violates the foreground lifecycle requirement to finish/cancel to a safe boundary before exit; a recoverable journal does not substitute for handling an ordinary cancellation request.

**Independent evidence:** `TestFinalReviewRootShutdownWaitsForActiveMutation` drives the actual public TUI model through selection, management, Plan, acknowledged Review and Apply. It submits that exact prepared TUI command to a real headless Bubble Tea Program with the production rootBoundBackend, holds the synthetic Apply callback at a channel barrier, cancels the root, and waits for Program.Run to return before releasing the callback. Result:

```text
operation_context_active_after_program_return=true
backend_can_mutate_after_return=true
```

The callback finishes after release, so no test goroutine remains blocked. This is an in-process runtime/control-flow reproduction, not PTY or actual transaction crash proof. The production immediate return/exit path is directly visible at the cited call sites.

**Minimum fix/regression:** Give all foreground backend work a session/root cancellation lifetime and track its completion. On shutdown, request cancellation and drain active mutation cleanup before the application returns/exits; terminal restoration and child-specific SIGINT behavior must still occur correctly. Merely cancelling a wrapper context without awaiting cleanup is insufficient. Add a barrier regression at the real application shutdown boundary: cancel with Apply in progress, prove cancellation reaches the transaction, and prove exit waits until its final safe outcome. Cover a persistent credential/service mutation too. This is separate from C1's descendant-process cleanup.

### T1 — P2: Unsupported Linux nodump metadata is accepted and removed

**Location:** `internal/configtxn/metadata_linux.go:10`; supported policy at `internal/configtxn/metadata_posix.go:11` and `:25`.

**Trigger/effect:** A reviewed target receives the unprivileged FS_NODUMP_FL inode flag before Apply. Linux checks xattrs but not inode flags; replacement succeeds and removes nodump, changing the operator's backup/dump exclusion. The documented policy accepts ordinary mode/owner/effective group and refuses extra native metadata. Nodump is separate from xattrs, so the refusal guarantee is not satisfied.

**Independent evidence:** Read and assessed `txn-rereview.md` T1; then separately reran this scenario against the whole-branch frozen source using `TestFinalReviewNodumpPreservation`. Real disposable Linux file, no privileges or timing race:

```text
outcome=Succeeded error=<nil> before=40 after=0
```

This independently confirms the residual finding without reopening the fixed ACL/no-op/retry defects.

**Minimum fix/regression:** Check relevant Linux user-policy inode flags through the shared preflight/commit/recovery metadata seam and refuse unsupported ones without changing content or metadata. Allow ordinary filesystem-managed flags where appropriate rather than rejecting every nonzero value. Test initial nodump, drift after review/staging, and nodump on an own output before recovery; preserve foreign state and report Conflict as appropriate.

### T2 — P2: Native busctl arrays are decoded with an invented wrapper

**Location:** `internal/monitor/systemd.go:27–34`; consumers at `:37–43` and `:75–90`; matching incorrect fixture at `internal/monitor/lifecycle_test.go:42–45`.

**Trigger/effect:** A normal systemd user manager returns UnitPath as `{"type":"as","data":["/path/a","/path/b",...]}`. busProperty requires exactly one data element and returns only that element. Normal UnitPath therefore becomes Unavailable, preventing Linux observer lifecycle use. A one-path response also fails its []string consumer. ExecStart's data is directly an array of ten-field structs; unwrapping data[0] turns the first struct into the root and fails effectiveCommand's outer-array check. The safe refusal avoids activating an unverified service, but leaves the implemented Linux backend unusable with native responses.

**Evidence:** The controller reported a successful read-only native UnitPath query with 18 directly encoded string paths, plus directly encoded ExecStart structs, without retaining raw argv or changing services. That native observation is controller evidence, not my own service query. Independently, `TestFinalReviewNativeBusctlArrayShape` supplies these protocol shapes through the existing fake runner and exercises both actual consumers. UnitPath fails with errUnavailable; ExecStart fails with errForeignOverride. No real native command executes. Existing fixture tests pass because they add the same extra wrapper expected by the implementation.

**Minimum fix/regression:** Preserve the complete `data` JSON value for the validated array signature; let each typed consumer validate array size and element shape. Correct both fixture encodings. Test normal multi-path and one-path UnitPath, one valid ExecStart struct, and malformed/wrong-signature responses, while retaining authority/drop-in/effective-argv refusal checks. Controller's already-authorized read-only native protocol summary can confirm the shape after the fix; no service mutation is required for that check.

### U2 — P2: TUI drops the returned uncertain/conflict service state on error

**Location:** `internal/tui/async.go:31–49`, contrasted with service handling at `:104–106`; producer `internal/app/actions.go:218–223` returns safe ServiceInfo alongside failure.

**Trigger/effect:** Start/install/restart returns an error plus `ServiceInfo{Status:"Unknown"}` after an uncertain native result, or Conflict/Unavailable after refusal. The generic error branch returns before consuming ServiceInfo. The Service screen continues displaying its previous `Observed: Stopped` (or Running), losing the returned uncertainty/reconciliation state precisely when it matters. The backend reservation still prevents blind retries; this finding does not claim that guard is bypassed.

**Independent evidence:** A complete synthetic TUI service action (`TestFinalReviewServiceFailureProjection`) returns each of Unknown, Conflict and Unavailable alongside the same fixed application error shape. All three retain Stopped and fail to show the returned observed state:

```text
returned=Unknown retained=Stopped visible_returned=false
returned=Conflict retained=Stopped visible_returned=false
returned=Unavailable retained=Stopped visible_returned=false
```

**Minimum fix/regression:** Consume a valid returned safe ServiceInfo even when the service operation also returns an error, retaining an error/uncertainty message and reconciliation guidance. Do not auto-resend the mutation or label it successful. Add error-plus-state journeys for the three statuses, including Unknown with the previous state Running.

### K1 — P2: Explicit persistent key entry does not select the new key

**Location:** `internal/app/actions.go:107–110`; `internal/credentials/store.go:99–102,111–125`.

**Trigger/effect:** On the supported Windows persistence path, enter key A as session-only, then explicitly enter key B with Persist checked. SaveKey calls Save only, which writes B to the vault without replacing the foreground session. Resolve continues returning A. An already selected environment value likewise overrides the explicitly entered and successfully saved B. The UI reports that the key was saved, but subsequent Launch/Discover still authenticates with the old key; no UI control explains a save-for-next-session-only interpretation or selects B. The ordinary startup session/environment/vault precedence is reasonable; it must not negate a newer explicit key entry.

**Independent evidence:** `TestFinalReviewExplicitSavedKeyBecomesSelected`, using an injected vault only, covers both previous-session and previous-environment cases. In each, saved_new=true and selected_new=false. Source tracing confirms that actual Engine.SaveKey does not make the newly saved key the foreground selection. The application fixture's Save calls SetSession, so its existing test double masks this mismatch with the real credential Manager. No native vault access was performed or claimed.

**Minimum fix/regression:** After successful explicit persistent entry, select that exact supplied key for the current foreground session as well, without modifying the parent environment. Preserve the previous active credential if persistence fails. Keep normal startup reference precedence. Test SaveKey through the application boundary with a credential implementation matching the real Save/Resolve semantics, both prior sources, and persistence failure; verify subsequent synthetic launch/discovery receives B without logging either value.

### D1 — P3, documentation only: README still says design is the next step

**Location:** `README.md:209–212` in the included controller snapshot.

A reader following Scope and research is told the next step is a design/grilling session and the reuse decision remains future, despite the new introduction, completed stage-zero trial and native implementation. Replace those final sentences with links to the implemented foundation, retained adoption evidence and remaining production plan. This is a small status correction and does not independently block the development build.

## Prior findings, rulings and declared deferrals

| Item | Whole-branch disposition |
| --- | --- |
| R1–R3 runtime fixes and reserved env references | Source shows bounded/exact version identity, pre-dispatch cancellation and valid UTF-8 IDs; existing regressions executed in the fresh full run. Closed findings are not repeated. General descendant cleanup was not established by the old runtime bounded-pipe result. |
| Adapter A1–A3 | Import clears confirmations, any custom catalog reaches native model/list validation, and marked foreign providers are refused. Pure ownership regression executes; native adapter cases skip in my run. Prior native proofs remain separately attributed. |
| A4 withdrawn | Correct ruling: TiersConfirmed means both Fast and Ultrafast. Model/default tier and availability remain separate; no Fast-only extension is required for this build. |
| AP1/AP2/AP3, failed status projection, canonical reconciliation | Reviewed integrated source retains cancellable supersession, post-wait protected-state checks, exact public Desired and private canonical values, Failed status and successful-client-only draft reconciliation. Their committed regressions execute in the full run. No reopening of these proven cases. |
| Transaction F1/F2/W1 | Full source checks complete preconditions including no-ops, immutable retry sources and unsupported ACL/xattr/owner/group refusal. Relevant Linux regressions execute. The independent transaction rereview's closures are coherent; T1 is the remaining distinct policy gap. |
| UI F3/F4 and U1 | Error Apply receipts retained; wide layout, semantic labels/colors and adjacent model/default maintained; composed tier label sanitized. Existing regressions execute. Parent's earlier visual/PTY evidence is not claimed as my own. |
| Monitor S1/F1/F2/F3 | Read the integrated authority/drop-in/effective-command/absence/runtime-enable corrections. Their committed synthetic tests execute. Curie's scoped independent rereview remains the authority for the original probes; I did not duplicate them. T2 and U2 concern different protocol/consumer seams. |
| CLI C1–C4 | Still active assigned fixes, not silently cleared by a passing ordinary suite: descendant cleanup, ANSI/path redaction ordering, exact JSON Unicode identity, first output-write failure. Final producer delta needs scoped rereview. C1 verification must also account for TUI ExecProcess bypassing runProcess; the frozen app CommandContext currently uses direct-child cancellation. N1 independently concerns non-child foreground jobs. |
| Darwin ACL fixture | Corrected after the freeze; controller reports native macOS now passes. I did not rerun it or treat a fixture-only failure as a new production bug. |
| R1–R4/R6/R7 ledger scope | Native implementation authorized; executable oneline-ccc avoids collision; pinned Go/Tea/native config API justified; adoption trial supports the remaining custom seams; existing cache workflow accepted. Authenticated bootstrap/self-update remain disabled. No new approval or workflow migration gate imposed. |
| Declined/deferred surfaces | Five other adapters/installers, utilities, Windows observation, Linux/macOS persistent vault providers, automatic repair and authenticated distribution remain explicitly unavailable/deferred. Bare Codex cannot obtain the in-memory session key. Missing broad-plan functionality is not a finding against this first native slice. Ordinary POSIX hard-link relationship preservation and universal foreign-writer CAS were never promised; do not silently strengthen those contracts. |

The documentation addendum correctly supplies CGO_ENABLED=0 POSIX and literal PowerShell builds, distinguishes no separate language runtime from macOS system libraries, indexes actual native progress, and separates Linux terminal evidence, native CI, fake observer tests and remaining product gates. README/site preserve the browser specimen's simulation label and label Go artifacts as development builds. The verification draft is assessed as a controller evidence account; I did not independently replay its Linux PTY/launch/SSH claims. Its final ledger/revision/gate references must be filled by the controller after the fix pass. These documents were present and reviewed.

## Fresh verification and proof limits

One full gate only, from the frozen archive:

```text
go test -race -p 2 -count=1 -json ./...
PASS: 11 tested packages, 537 test/subtest pass events.
```

All invocations used GOMAXPROCS=2, GOPROXY=off, GOSUMDB=off, GOTOOLCHAIN=local, and the parent `.cache/go-build` / `.cache/go-tmp` for GOCACHE/GOTMPDIR/TMPDIR. Native fixture/vault opt-in variables were removed. No dependency downloads or public calls were made.

There were **14 test/subtest skip events**: eight native adapter events without explicit Codex/catalog fixtures, two native application journeys, the unsupported-platform guard, two nonexistent new-file restore-stage boundaries, and the Windows observer assertion on Linux. Two additional packages report no test files. The passing full run therefore is **not fresh native adapter, native Windows vault, macOS filesystem, real service or PTY acceptance**. Full output: `final-review-frozen-jspclrqc/whole-branch-race.json`.

Targeted additional race-enabled probes used the retained `probes/overlay.json` and only these packages:

```text
go test -race -p 2 -count=1 -overlay <manifest> -run '^TestFinalReview' -v \
  ./internal/configtxn ./internal/tui ./internal/credentials
# Expected assertion failures: T1, U2, K1; result.log.

go test -race -p 2 -count=1 -overlay <manifest> \
  -run '^TestFinalReview(NativeBusctlArrayShape|RootShutdownWaitsForActiveMutation)$' -v \
  ./internal/monitor ./cmd/oneline-ccc
# Expected assertion failures: T2, N1; integration.log.
```

Probe sources, manifest and both logs are retained under `final-review-frozen-jspclrqc/probes/`. They add no production-source files. No data race was reported. These deterministic failures supplement rather than invalidate the accurate full-suite PASS.

Controller reports run **41441105936** passed native Windows/Linux and six builds, with only Darwin fixture failure, then run **41442098324** at **468e886** passed all nine jobs after the fixture correction. Those are separately attributed later-revision native/build results; they precede final CLI/T1/T2/new-issue fixes and do not certify the final candidate. The reviewed local `native-ci-second.log` contains older run failures and must not be substituted for the controller's newer result. Native service and actual platform terminal acceptance remain distinct from fake-runner and compile results, as the docs state.

## Production seams read in full

- CLI/root: all seven production `cmd/oneline-ccc` Go files and embedded-helper `sshhelper.go`; dispatch, flags, output, child execution, signal/root binding, helper bytes/environment.
- Application/domain: all production `internal/app` files (engine/state/plan/actions/attestation), `internal/core/profile.go` and `types.go`; request identity, executable/config snapshots, evidence persistence, no-op, supersession, credentials/discovery/service/repair and receipts.
- Adapter/RPC: all production Codex adapter and codexrpc files; native resolution, version identity, staging, native layer/version/validation, named-file ownership, full catalog semantics and transport cleanup.
- Protected I/O: all production configtxn files, including Linux/Darwin metadata and ACL parser, POSIX/Windows/unsupported platform/security seams, complete conditions, locks, journals, retry immutability, Apply/recovery/private/uncreated paths.
- Credentials/catalog: all production files; resolution/save precedence, minimal launch environment, fixed public metadata, endpoint/redirect/timeout/size/JSON/identity policy and independent tier confirmation.
- Observer: all production files and README; observation, protected records, reservations, lifecycle/absence/reconciliation, effective systemd properties, literal service templates, runner and Windows unavailability.
- TUI: all production files; async state, exact review identity/pages, help/palette/input/editor, drafts/quit/cancellation, rendering/privacy, wide/compact layouts, themes and terminal handoff.
- Integration/docs: module pins/workflow, native fixture helpers/journeys, baseline diff, authority/compatibility/stage-zero documents, README/site changes and captured controller documentation addendum. Existing tests were executed and selected regression/test-double implementations inspected; no claim that every test file received a full manual line review.

Proceed with one concrete fix pass for N1/T1/T2/U2/K1 and assigned CLI work, plus D1's small documentation correction, then the prescribed scoped delta rereview. No source changes were made by this reviewer.

=== DONE ===
