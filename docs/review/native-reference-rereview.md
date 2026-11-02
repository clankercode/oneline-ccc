# A4 fix round 1 — scoped independent re-review

Spec verdict: **COMPLIANT within this fix round**. F1–F5 and the tracked-report warning are addressed. No remaining Missing/Extra/Misunderstood item established within the requested scope.

Quality verdict: **APPROVED for the bounded browser-reference fixes**. No actionable regression or remaining Critical/Important/Minor finding established. Native all-screen visual acceptance and full product acceptance remain open.

## Reviewed inputs and boundary

- Worktree: `/home/xertrov/src/oneline-ccc/.worktrees/native-references`.
- Exact range: `a8ea038c727b60f294a334ab5390d52514d8a7ba..46b50cd016917f2b7e45b898c749530db04c5ea1`; the worktree remained clean at the requested HEAD.
- Read the re-review brief, original F1–F5 review, complete 772-line saved fix diff, and complete ignored `A4-reference-fix1-report.md`. Checked changed source and verifier assertions against the earlier contract.
- Saved `A4-fix1-review.diff` uses seven context lines. Its SHA-256 exactly matches `git diff --unified=7 a8ea038 46b50cd --binary`: `adf8ea9a9ba46a2c0246a0491708b51bed248e34f7415f1c524945899e1e319e`. A default-context diff has different bytes; this is not a content mismatch.
- Exact changes: companion CSS, companion module, browser verifier, and removal of the old report from tracking. Approved primary preview, native source and Go dependencies have no diff against the original `0fab61e` baseline.
- Review was read-only apart from this designated ignored report, authored last. No browser journey/matrix rerun, source/git mutation, agents, notifications, network call or operator-state access was performed. Fresh checks were syntax/diff checks, evidence accounting and decoding existing PNGs.

All source references below are relative to the reviewed worktree.

## Finding-by-finding disposition

### F1 — Addressed: visible focus, return anchors, history and layer appearance

**Source:** `site/tui-native-reference.mjs:45-55`, `:270-301`, `:343-345`, `:394-400`; CSS scroll margin at `site/tui-native-reference.css:8`.

Visits now retain a selector, document scroll position and invoking-control viewport position. Restoration focuses the saved control, restores its anchor, then bounds visibility. Initial focus can scroll normally. Screen return uses its saved visit; browser Back/Forward retrieves history state rather than discarding it. Layer dismissal preserves current theme/layout. Reopening the current destination through More restores the represented view/state.

**Evidence:** Final recorded journeys `F1-initial-screen-focus` through `F1-layer-return-keeps-appearance` account for seven named PASS outcomes. Assertions check all six narrow screens, five layer/editor entry targets, control identity, full viewport bounds, scrollY/top restoration within 2px, keyboard focus styling, browser Back/Forward and theme/layout retention. They check geometry before screenshot collection.

Recorded Gateway initial focus moved from RED y=1299.1–1336.5 to GREEN y=403.1–440.5 at 390×844. Key return restores y=465.2–502.7 and scrollY=1348. Settings return restores the Background monitor row at y=365.0–479.8 and scrollY=1240. The separate appearance RED records dark incorrectly returning instead of light; final GREEN retains light/compact.

**Remaining finding:** None established in the requested F1 paths.

### F2 — Addressed: lifecycle and direct confirmation authority

**Source:** `site/tui-native-reference.mjs:77-84`, `:97-111`, `:168-182`, `:342`.

Screen and confirmation share represented backend/ownership facts. Install requires absent registration; remove requires running/stopped with explicit account ownership. Unavailable, pending and unknown states cannot reach an enabled removal action. Confirmation URLs use the same guards, keep the represented backend and ownership, and expose reasons. The disabled-control dispatch guard runs before opening layers or accepting confirmation.

**Evidence:** `F2-unavailable-removal-reason` checks disabled removal, accessible reasons, Enter refusal and same-Service More reentry for unavailable/pending/error. `F2-direct-service-confirmation-ownership` checks both install and uninstall URLs across all six Service states, backend consistency, explicit/non-invented ownership, enabled positive cases and disabled negative activation. Both final journeys PASS. Archived original RED demonstrates the enabled unavailable action and invented systemd backend; an additional RED captures More resetting unavailable to absent, corrected in the final source.

Start/Stop/Restart retain their represented-state guards. No actual service operation is performed.

**Remaining finding:** None established.

### F3 — Addressed: useful named Utilities content without invented capability

**Source:** `site/tui-native-reference.mjs:138-152`; existing identity evidence at `docs/research/tui-stack.md:67-71`.

The repeated operation-unavailability rows and implementation-review checklist are removed. A named table describes herdr, tmux, T3 Code and Paseo, with one build limitation and a concise “follow its own instructions / confirm the exact project first” next step. Candidate intended identities remain qualified as unverified. Orca says “Project not identified; unavailable” and has no guessed repository. No installed-state detection or verified external link is implied.

The unavailable Manage action references the existing limitation for its accessible reason. Repetition in the live reason announcement is contextual feedback rather than duplicate visible checklist copy.

**Evidence:** Original `red-f3` fails on the missing herdr name. Final `F3-named-utility-availability` checks all five labels, existing repository-research provenance, uncertain intended matches, Orca, removed checklist language, build limitation, next step, absent external links and keyboard activation of the unavailable reason. PASS.

**Remaining finding:** None established. This is a source/product-language judgment, not independent pixel acceptance of the replacement table.

### F4 — Addressed: retained budget/compact/credential consistency

**Source:** `site/tui-native-reference.mjs:72`, `:175`, `:303-324`.

Gateway reads the retained credential reference. Successful budget save derives compact with the native integer formula; explicit compact edits remain retained. Validation returns before draft writes, and cancellation restores the earlier view without changing retained values. Discard reference copy reads the retained context budget.

**Evidence:** Original `red-f4` records both 272000 instead of 900 and the stale credential label. Final budget journey verifies 1000→900, 19→17, explicit compact 15, invalid compact 0 preserving 15, cancelled budget 2000 preserving compact, and retained budget in discard copy. Credential journey verifies save to OTHER_DUMMY_KEY, editor reopen and cancellation preserving that value. Both PASS.

**Evidence boundary:** The recorded refusal case is compact 0; it is not a separate browser refusal run for every numeric/credential input. Source shows the common numeric guard and credential guard return before the single draft-write boundary. No invalid-budget or invalid-credential preservation claim beyond that source inspection is made.

**Remaining finding:** None established.

### F5 — Addressed: wrapping palette hit areas and selected visibility

**Source:** `site/tui-native-reference.css:84-87`; selected-result scrolling at `site/tui-native-reference.mjs:223-228`.

Rows use `minmax(63px, max-content)` in the bounded 252px scrolling window. Content can increase row height without entering an adjacent hit area.

**Evidence:** Original `red-f5` fails the wrapped row containment assertion at 320×844. Final `F5-wrapped-palette-row-geometry` checks the filtered credential row, all unfiltered rows, hint/row containment, adjacent-row separation, final selected-result containment, visible input focus and horizontal page extent. PASS. The recorded credential row now has client/scroll height 84px, with its hint inside the row. Nonselected partial rows at the scrolling-window edge do not invalidate selected-row containment.

**Remaining finding:** None established.

### Tracked-report warning — Addressed

**Evidence:** The exact commit deletes `.superpowers/sdd/product-completion/A4-reference-report.md` from tracking. Fresh `git ls-files .superpowers` returns no entries. Both the original report and fix-round report match ignored paths. A fresh byte comparison proves the original local report equals `git show a8ea038:.superpowers/sdd/product-completion/A4-reference-report.md`.

The report was removed from the index without deleting its local evidence. The new handoff stays ignored. This satisfies the earlier ignored-workspace alternative without adding an out-of-scope portable documentation file.

`red-scope/results.json` records the original tracking failure, and final `F0-source-and-handoff-boundary` PASS checks untracked reports and unchanged preview/native source. The separate `red-scope-capture-error.log` is a harness screenshot timeout caused by missing fixture setup; it is correctly disclosed and is not treated as behavioral RED.

## Evidence accounting and checks

- Read actual RED result JSON/log records for F1–F5, tracking scope, appearance return and Service reentry, plus per-finding GREEN records and final `fix1/green.log` / `fix1/green/results.json`.
- Final results contain **14 distinct named PASS outcomes, 0 failed**, integer timings, queries/viewports and capture paths. Counts are derived from recorded outcomes, unlike the earlier literal “11 journeys” counter. Page/console errors and external-request arrays are empty in the supplied final records.
- The final outcome set is seven F1 journeys, two F2, one F3, two F4, one F5 and one source/handoff journey. Negative service states/direct URLs, keyboard refusal, invalid compact, cancellation, identity uncertainty and narrow row geometry are substantive consumer checks.
- `fix1/captures/manifest.json` contains **15 distinct targeted design states**, with empty errors/remote arrays. Every expected viewport/console pair exists.
- Fresh ImageMagick full decoding with `-regard-warnings` succeeded for **69 PNGs**: 28 final-journey images plus 30 targeted viewport/console images, eight crops and three enlarged crops. This reconciles the worker's 66 images plus three additional zooms. No new captures were generated. An initial Pillow import was unavailable; installed ImageMagick supplied the completed decode check without installing anything.
- Fresh `node --check site/tui-native-reference.mjs`, `node --check tests/tui-native-reference-browser.mjs` and `git diff --check a8ea038 46b50cd` each returned 0.
- Fresh source-scope comparison against `0fab61e` confirms approved preview/native source/Go dependencies are unchanged. The reviewed worktree remained clean at `46b50cd`. The main checkout's separate modification to `docs/plans/remaining-items-so-far.md` was observed and left untouched.

Artifact paths above are under `.cache/artifacts/native-references/fix1/` in the reviewed worktree unless prefixed otherwise. The full worker report remains local at `.superpowers/sdd/product-completion/A4-reference-fix1-report.md`.

## Regressions and evidence limits

No actionable regression was established in the exact fix diff and requested paths. The appearance-return and same-Service reentry regressions found during implementation have explicit RED records, final assertions and corresponding source corrections.

This re-review inspected supplied execution records and current source; it did **not** execute those 14 journeys afresh, rerun the original 159-capture/161-check matrix, or independently grade the new image pixels. Decoding verifies readable image artifacts, not visual hierarchy. The 15 targeted states and focused journeys are bounded evidence and do not refresh the original matrix.

Native terminal-cell geometry, pinned footer, ANSI/NO_COLOR behavior, Unicode widths, screen-reader operation, PTY cancellation/child ownership/restoration and real service/credential/SSH/export behavior remain unverified here. Nothing in the fix report or this approval establishes native all-screen acceptance, integrated product gates or full product completion.

=== DONE ===
