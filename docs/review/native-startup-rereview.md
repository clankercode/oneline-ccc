# A6 fix1 scoped independent re-review

**Spec verdict: PASS for the assigned I1 fix.** Missing: none in this scope. Extra: none. Misunderstood: none. The original A6/backlog20 limits remain open.

**Quality verdict: PASS for this delta.** I1 is resolved; no new actionable findings. This approves the observable-cancellation correction, not full native-platform or backlog20 completion.

## Exact reviewed scope

- BASE: `9504322174abf2282927abfc89b1a6f98e200475`.
- HEAD: `e2d209fc995bb0aa340f06a6b62c38f07c2d3439`.
- Worktree: `/home/xertrov/src/oneline-ccc/.worktrees/terminal-startup`.
- Read the complete controller `A6-rereview1-brief.md`, `A6-rereview1.diff`, original `A6-review.md`, and worktree `A6-fix1-report.md`.
- Supplied diff byte-matches `git diff --no-ext-diff --unified=8 BASE HEAD --binary`: 339 lines; SHA-256 `c3ec8512e5b7855926a0cb076e3eb328ec99be12684a29f7052191f2d1063f0c`.
- Exactly four tracked paths changed: `third_party/bubbletea/tea.go`, `third_party/bubbletea/startup_lifetime_test.go`, `third_party/README.md`, and `third_party/terminal-input.patch`. Reviewed all hunks, including duplicated maintained-patch changes.

No source/git mutations, agents, notifications, operator-state access, network or broad gate reruns. The designated ignored report is the only authored review file; fresh Go verification used the existing worktree-local scratch/cache. Worktree was clean at the pinned HEAD.

## Finding disposition

**I1 — Medium/P2, cancellation during renderer startup admits Model.Init: RESOLVED.**

At `third_party/bubbletea/tea.go:1193`, startup releases `terminalMu`. The new guard at lines 1197–1199 checks `p.ctx.Err()` immediately before `model.Init()` at line 1200 and returns `ErrProgramKilled` through the existing deferred cleanup. It observes cancellation delivered during renderer startup before invoking a new callback. The post-Init check remains. No callback was moved under the mutex, and handler registration still precedes the startup unlock.

`third_party/bubbletea/startup_lifetime_test.go:178` permanently retains the review's deterministic renderer-start cancellation reproduction. Its renderer cancels the actual external root before returning from `start()`. Assertions require zero Init calls, `ErrProgramKilled` and `context.Canceled` identities, zero views, exactly one renderer start/close, and completion of repeated Kill/Kill/Wait. These assertions test the reported behavior without timing sleeps or weakening existing fixtures.

`third_party/README.md:41–45` now states that checks skip work when cancellation is already observable, explicitly retaining callbacks outside the lock and acknowledging cancellation racing with callback entry. This accurately scopes an ordinary check; it does not promise atomic admission exclusion or termination of an already-running callback.

The production delta is only this guard/comment. Teardown, restoration, error joining and A1 input cancellation/join/close ordering are unchanged. Fresh reentrant Init-Kill and input-completion barriers pass, supporting that the correction did not reintroduce the earlier callback-under-lock deadlock or close-before-join behavior.

## Fresh independent verification

Linux amd64, existing local Go 1.26.1; all Go execution used `GOMAXPROCS=2`, `-p 2`, `GOPROXY=off`, `GOTOOLCHAIN=local`, and worktree-local `.cache/a6/tmp` and `.cache/a6/gocache`.

```sh
env GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local \
  TMPDIR="$PWD/.cache/a6/tmp" GOTMPDIR="$PWD/.cache/a6/tmp" \
  GOCACHE="$PWD/.cache/a6/gocache" GOWORK="$PWD/.cache/a6/go.work" \
  go test -race -p 2 -count=1 -timeout=20s ./third_party/bubbletea \
  -run '^Test(A6ReviewCanceledDuringRendererStart|Startup|InputReaderClose)' -v
```

**Exit 0**, package duration 1.019 seconds: seven top-level tests and three input-lifetime subcases pass; no skips, failures, race diagnostics or hangs. Includes the new regression, synchronous Kill from Init, setup-vs-Kill, unstarted renderer, pre-canceled acquisition, panic input join, and quit/kill/release reader-completion barriers. Expected recovered-panic stacks occur in passing preservation fixtures.

```sh
python3 third_party/verify.py /home/xertrov/go/pkg/mod/cache/download
```

**Exit 0.** Tea v2.1.1 and pinned Ultraviolet v0.0.0-20261008173134-6b8d4baf91b4 pass archive SHA-256, module h1 and go.mod h1 verification (79 and 109 upstream files respectively). Complete mirrors equal the pinned archives plus exactly `terminal-input.patch`. The verifier and manifest were not changed to accept the fix.

Non-patch `git diff --check BASE HEAD -- ':!third_party/terminal-input.patch'` passes. Frozen-scope comparison is unchanged; the four-path diff contains no module, manifest, verifier, Ultraviolet, tty, CLI fixture/helper, or A1 input-fixture edits. Generic whitespace warnings on unified-patch context remain distinct from the successful exact patch reconstruction.

## Supplied RED/GREEN and terminal evidence checked

Read `.cache/a6-fix1/red-dependency-race.log`: the new regression fails before the guard with one Init call, zero views, and one renderer start/close, while the five prior startup tests pass. This agrees with the original review's independently reproduced failure. No new RED rerun was needed for this scoped correction.

Independently parsed the actual fix1 JSON logs and checked their validation ledger, rather than relying on report counts:

| Log under worktree `.cache/a6-fix1/` | Checked result |
| --- | --- |
| `dependency-startup-race.log` | Recorded exit 0; seven top-level and three subcase passes |
| `rendered-pty-race.log` | Recorded exit 0; four top-level and 16 subcase passes |
| `partial-setup-overlay-race.log` | Recorded exit 0; one top-level and two subcase passes |

All three contain zero skips/failures/race diagnostics. The latter two are supplied worker executions, not newly rerun review gates. They preserve the actual rendered raw-child SIGTERM/reaping, queued-input cancellation/Quit/release, nine startup PTY cases, and returned partial-setup-error/cancellation probes from the original A6 review. Existing fixture/overlay code is unchanged.

Also inspected `fresh-apply.log`: the maintained patch applies to freshly extracted pinned archives and the complete resulting file sets and bytes match both mirrors (81 Tea files, 110 Ultraviolet files). Fresh independent offline verification above separately confirms exact archive-plus-patch provenance.

## Continuing evidence limits

- The deterministic renderer probe establishes callback ordering and cleanup assertions using a fixture renderer; it is not itself real-terminal failure injection. The supplied PTY gates and original independent A6 checks remain the terminal evidence.
- The ordinary check does not atomically exclude cancellation racing after it. Already-running arbitrary callbacks and uncooperative external I/O remain outside the bounded proof. Custom noncancelable readers still require their owner to unblock Read; no timeout fallback was introduced.
- The inherited cancelreader EpollCtl-constructor private wake-pipe descriptor leak remains unresolved. Fixture reaping/process exit or eventual finalization does not prove descriptor closure before Run returns. This remains a separate backlog20 obligation.
- Native macOS/Windows startup/error, partial console restoration and foreground ownership runtime proof remains open. No new native or cross-compilation claim is made, and compilation cannot close that acceptance.
- Broader unchanged suites, platform builds and fault overlays were not rerun. This report supersedes only I1's disposition and the scoped spec/quality verdict; it does not erase the original review's remaining limits.

=== DONE ===
