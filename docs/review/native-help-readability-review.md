# Independent A3c Help readability review

Spec verdict: **PASS for the bounded wrapped-row readability correction.**

Quality verdict: **APPROVED for this source/geometry and behavior scope. No Critical, Important or Minor findings.**

**Help visual design, P0 interaction/visual acceptance and native all-screen/platform acceptance remain OPEN.** The flat destination catalog on the first narrow page remains an explicit product finding, not an approved Help design.

## Scope and identity

Reviewed worktree `/home/xertrov/src/oneline-ccc/.worktrees/native-help` at **`c6433668c1bea1bda621a24e6727cd5ec213db28`**, base **`497ad84d8694c5376d8bb647a3b4b2fb09708c25`**. Read the supplied brief, plan, report, exact diff, saved logs, authoritative P0 Help findings and directly relevant shared-shell/input/resize/security context.

The saved diff exactly matches `git diff --unified=8 497ad84 c643366`; SHA-256 `c7f53fe6948b7d1e2db66760e8813a2334d2b4d66c55eabd053dd2baabfe5dae`. Its four paths are two production files and two test files. All four and the inspected shared-shell/caller sources match the reviewed commit. `git diff --check 497ad84 c643366` passes.

No source/git edits, agents, network/operator-state access or notifications. Ran only the focused Help tests below, with checkout-local Go build/temp paths. This report is the only authored review file; the test command may populate those caches. No broad suite was rerun.

## Checked and accepted

- **Pagination measures the actual viewport.** `internal/tui/help.go` wraps public catalog text at `bodyWidth()`, subtracts the actual shared header/footer footprints, and reserves wrapped heading and Close rows before allocating binding rows. These match `layout.go`, `components.go:107–219` and `dashboard.go:169–183`, rather than an assumed number of entries or fixed shell height. Footer scope text occupies the same reserved row. The removed body paging hint remains available in the Help footer.
- **Descriptions remain complete.** Every catalog entry contributes its full label and existing key metadata. Entries that fit remain together; oversized entries continue line by line across pages. Page construction neither skips a remainder nor introduces an empty leading page. The heading reservation uses the total wrapped-row count as an upper bound on page-count digit width, avoiding a heading that grows beyond the reserved space as pages are added. Rendering applies the same wrapping width and retains the existing public-text sanitation.
- **Title, Close and page content fit at supported sizes.** The new coverage inspects every page at 40×12, 60×20, 80×24, 100×30 and 120×40. It requires zero body scroll, all page rows within the actual viewport, bounded row widths, visible heading/Close, and every complete catalog description across the pages. These are meaningful checks beyond merely finding a pinned title in the final frame (`help_readability_test.go:14–59`).
- **Boundary and resize behavior remains coherent.** Rendering clamps its page slice; unchanged Help input handling clamps stored state before movement; resize clamps it after updating dimensions (`panels.go:48–55`, `commands.go:54–63`, `model.go:69–79`). Repeated first/last input, immediate reverse movement, shrinking the page range, one-page expansion and subsequent shrink remain asserted for Tab/Shift+Tab, Down/Up and PgDown/PgUp. Stale negative/oversized indices cannot become an unchecked page slice. Capacity is at least one, preventing zero-capacity pagination loops; model dimension normalization and the existing capped View retain safe tiny-size behavior. Full readability below 40×12 is not claimed, and those extreme/stale-index cases were assessed by source inspection rather than a new runtime probe.
- **The existing test change removes the obsolete assumption only.** `navigation_fix1_test.go:123–178` replaces catalog-count divisors with the displayed range. It retains stored/rendered agreement, frame bounds, repeated overshoot, immediate reversal, resize clamping, the explicit one-page case and subsequent forward movement. The new independent viewport/content assertions would still reject a falsely small declared range that omits descriptions. No behavior assertion was dropped to obtain GREEN.
- **Authority and lifetime contracts are preserved.** The production delta is confined to Help pagination/rendering. Review pagination/acknowledgment and exact-plan authority, editor input/masking, draft state, navigation restoration and backend/lifetime code are untouched. Help still uses the existing modal close/restore path; the new test verifies return to the invoking control. No private model/key metadata or new backend action is introduced.

## Execution evidence

| Evidence | Observed result |
| --- | --- |
| Saved `.cache/help/red.log` | Original viewport failure at 40×12, 60×20, 80×24 and 100×30, plus resize failure; rows exceed the real viewport |
| Saved `.cache/help/green-race.log` | FAIL: all three old Help boundary subcases disagree with the changed page range; this provisional file is not GREEN evidence |
| Saved `.cache/help/green-race2.log` | Worker-reported final full TUI race gate: package `ok`, 21.552s; no diagnostics in the saved output |
| Fresh focused reviewer execution | PASS, exit 0, package 0.027s |

Fresh command, with `GOMAXPROCS=2`, checkout-local `GOCACHE` and `GOTMPDIR`:

```sh
go test -p 2 -count=1 ./internal/tui -run '^(TestHelpEveryPageIsReadable|TestHelpResizeRetainsReadablePage|TestFix1HelpPagingBoundariesAndResize)$'
```

The saved race output is historical parent evidence, not an independently rerun race suite; its package-level text does not itself attest the invocation flags or individual test names.

## Integration and acceptance limits

The base includes the unintegrated A3b shared shell. This approval depends on those reviewed shell contracts: integrate A3b first, then cherry-pick only the A3c delta and verify the merged revision's viewport/journeys. It does not approve A3b itself or a different shell footprint.

Untracked A2 tooling/helper inputs are provisional capture infrastructure, outside this reviewed production delta. The parent's 40×12/80×24 PNG observations are disclosed provisional evidence; this source/geometry review did not perform an independent vision/reference comparison. Exact integrated recaptures, real input through final page/boundary states, grouped useful shortcut presentation, alternating independent visual/source review and final installed/native-platform evidence remain required. The main P0 findings explicitly retain the destination-only first narrow page and weak Help hierarchy.

=== DONE ===
