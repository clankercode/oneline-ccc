# A3a fix-round1 independent re-review

Spec verdict: **APPROVED for the bounded F1–F4 correction scope.** F1, F2, F3 and F4 are **ADDRESSED**.

Quality verdict: **APPROVED for this fix round.** No new blocking regression identified in the reviewed changes. This is not overall native-TUI acceptance: **A3b, P0 visual/interaction acceptance, PTY journeys and platform acceptance remain OPEN.**

## Scope and verification

Reviewed the complete re-review brief, original `A3-review.md`, implementer `A3-fix1-report.md`, and exact `A3-fix1-review.diff`. Reviewed all changed production hunks and all six new tests, with directly relevant callers/restoration, asynchronous discovery, editor, layout and review-authority boundaries only. No broader quality audit was undertaken.

Worktree: `/home/xertrov/src/oneline-ccc/.worktrees/native-polish`.
FIX_BASE: `1e2fec9064880ef2aefb689846595148422a2f5a`.
HEAD: `403ea354238e677163dd2781f44ecb85a6350558`.

The saved package byte-for-byte matches `git diff --no-ext-diff --unified=5 FIX_BASE HEAD`; SHA256 `2a205fa430b9563dc11240403ac4d5c6a6f1d7edf40da6d6121b6c0cc6d435c0`. All nine hashes in `A3-fix1-source-sha256.txt` match both the current files and their committed contents. HEAD and clean worktree status were checked again before writing this report; `git diff --check FIX_BASE HEAD` succeeded.

No source/git changes, tests or broad suites, application execution, subagents, attention utilities, operator configuration, real credentials or live gateway calls were used. The only written artifact is this designated ignored report, written last. Runtime results below are independently inspected saved implementer evidence, not freshly executed reviewer results. Source references are worktree-relative at HEAD.

## F1 — ADDRESSED: visit-specific Back frames

**Source:** `internal/tui/navigation.go:16`, `navigation.go:47`, `navigation.go:57`; history type at `internal/tui/model.go:39`.

Each pushed frame owns a value copy of screen, control ID, fallback index and scroll. Back restores the popped frame directly rather than resolving its location through the mutable per-screen cache. Later `remember()` calls can therefore update section-revisit preferences without overwriting earlier visits still on the Back stack. Same-screen navigation and the empty-history parent fallback retain their prior behavior.

**Regression evidence:** `TestFix1BackRestoresEachVisitFrame` (`internal/tui/navigation_fix1_test.go:13`) reproduces the original Models → Review → Models sequence with different controls and scrolls, then two Escapes. It checks both returned screens/locations, the retained draft and zero backend action calls. The saved initial RED fails at `advanced:budget`, expected `advanced:model`; final targeted evidence records this test passing.

## F2 — ADDRESSED: stable discovered identity and safe fallback

**Source:** `internal/tui/controls.go:17`, `controls.go:62`; `internal/tui/input.go:119`; `internal/tui/model.go:171`; `internal/tui/navigation.go:138`.

Control identity is now the quoted `(Source, ID)` pair. Quoting keeps delimiter-containing, escaped and Unicode fields distinct; neither array position nor advertised capacity participates. Activation matches that identity in the current inventory and assigns the original `info.ID` directly. A stale identity changes no draft and offers rediscovery. Existing `changeProfile` handling still clears unverified attestations when the model changes.

Restoring a vanished discovered identity on Models moves to `discover`, rather than the replacement at its old index. This helper is shared by asynchronous focus restoration, Back/section locations and ordinary dialog/editor return. F6 deliberately does not use that cross-region fallback: it scans the destination region and defaults to its first control. A vanished Form anchor therefore returns to `enabled`, from which reverse F6 still reaches Nav.

**Regression evidence:** `TestFix1DiscoveredFocusTracksExactIdentity` (`navigation_fix1_test.go:64`) covers reordered results while focused and while away on Jobs, with matching model IDs from different sources and escaped/Unicode text. It checks the intended focused identity, exact activated model ID, retained scroll/draft before activation, no inferred attestations and no Plan/Apply call. `TestFix1DisappearingDiscoveryFocusIsSafe` (`:94`) covers both locations, stale activation refusal and deliberate rediscovery without staging a replacement. `TestFix1MissingDiscoveryDoesNotTrapRegionTraversal` (`:264`) checks the region-local fallback and reverse escape from it.

The saved original RED shows positional-focus failures in both reorder and disappearance cases. The additional region RED shows the intermediate erroneous `discover` fallback where `enabled` was required. All these cases are present in final targeted passing evidence. The sole existing-test edit (`internal/tui/regression_test.go:16`) replaces the obsolete positional activation route; exact model/budget/capability assertions are unchanged.

## F3 — ADDRESSED: bounded symmetric Help page state

**Source:** `internal/tui/panels.go:36`, `panels.go:54`; `internal/tui/commands.go:54`; `internal/tui/model.go:77`.

Movement and rendering use the same page bound. Forward input clamps the stored index at the last page, reverse input clamps at zero, and resize normalizes it immediately against the new height. There is no invisible accumulated page state to consume before reversing. Resizing to one page and shrinking again retains a valid index. Review acknowledgment revocation still runs before this additional Help normalization.

**Regression evidence:** `TestFix1HelpPagingBoundariesAndResize` (`navigation_fix1_test.go:123`) checks repeated first/last boundary input, immediate reversal, fewer pages after resize, one-page resize and subsequent shrink for Tab/Shift+Tab, Down/Up and PgDown/PgUp. It independently checks stored state, the generated page header and frame bounds. The saved original RED has stored page 16, expected 6; final targeted evidence contains all three passing subcases.

**Retained A3b warning, independently assessed:** Help still counts catalog entries rather than wrapped display lines (`panels.go:37`), while layout follows the focused Close row (`internal/tui/layout.go:25`). Wrapped entries at narrow widths can therefore scroll the page title above the viewport. The first targeted log exposes the resulting visible-title assertion failure. Changing the new test to inspect `modalRows()[0]` verifies the corrected page semantics but does not establish visible-title readability. This pre-existing viewport behavior does not reinstate the invisible-page/reversal bug, so F3 is addressed; A3b must keep Help title/current-page and wrapped-entry readability in its coverage. No visual PASS is inferred from frame geometry.

## F4 — ADDRESSED: all four credential-entry hints match behavior

**Source:** `internal/tui/view.go:62`; unchanged key handling at `internal/tui/editor.go:47`, `editor.go:56`, `editor.go:73`, and `editor.go:116`.

- Input advertises typing/paste, caret movement and Enter focusing Save; it no longer advertises a Space toggle. Literal Space remains input, with constant masking.
- Persistence advertises Space/Enter toggling. Those keys change only the persistence choice.
- Save advertises Enter saving the key. Space/paste cannot change the hidden buffer or opt-in here; save remains deliberate and session-only by default.
- Back advertises Enter dropping the unsaved key. It restores the invoker without saving.

Traversal/cancel hints remain visible at 40 columns. `TestFix1CredentialHintsFollowFocusAndKeys` (`navigation_fix1_test.go:171`) reaches each position with actual Tab input and checks visible footer strings alongside actual Space, Enter, caret, paste, masking, opt-in, Save/Back and invoking-control behavior. All four saved RED hint failures become passing final targeted subcases. This fix changes footer selection, not credential storage or editor authority.

## Regressions

No new blocking regression identified within the fix diff and its directly affected boundaries.

- Stable model IDs retain the `discovered:` region prefix, so existing Form traversal still recognizes them. Disappearance cannot redirect F6 into Actions. Inventory replacement alone stages no choice.
- Per-visit restoration changes focus/scroll only; it introduces no backend action or configuration write. Existing modal/editor restoration remains shared through stable IDs and retains scroll.
- Help bounds are independent of Review pagination and acknowledgment. Apply remains absent from the palette; matching captured desired state, exact plan ID/revision, stale-draft refusal, resize revocation and inspection invalidation remain intact.
- Editor/palette typing and paste guards are unchanged. Key hints introduce no interception. More-menu routes, unavailable-client guards, cancellation and child/terminal ownership were not rewritten.
- Saved normal and race records both include passing existing navigation, typing/More, exact-plan/mismatch/stale/resize, credential masking, service/SSH confirmation, cancellation, read-only navigation, unsupported-client and successful-Apply reconciliation safeguards. Existing assertions were preserved apart from the justified identity-route update above.

## Evidence results and remaining limits

Independently parsed the saved JSONL files and checked counts/failure/race events:

| Evidence | Top-level / total tests | Result recorded | Package elapsed |
| --- | --- | --- | --- |
| `A3-fix1-red.jsonl` | 5 / 16 failures | Original F1–F4 cases fail | 0.004s |
| `A3-fix1-targeted.jsonl` | 6 / 17 passes | No failures | 0.006s |
| `A3-fix1-green.jsonl` | 56 / 99 passes | No failures | 0.203s |
| `A3-fix1-race.jsonl` | 56 / 99 passes | No failures or data-race warnings | 2.873s |

Also read the incremental F1/F2 logs, additional region RED and first targeted Help failure. Final source hashes match the supplied tested-file manifest and commit; saved logs remain implementer-run evidence and do not independently attest their process environment. No reviewer test rerun was needed to resolve a remaining source ambiguity.

Native pixel/reference comparisons, complete readable screen/overlay coverage, real PTY key delivery/journeys and Windows/macOS evidence remain unavailable in this re-review. The Help viewport warning above belongs explicitly to A3b. Approval closes the four original findings for this bounded correction only and does not close those acceptance gates.

=== DONE ===
