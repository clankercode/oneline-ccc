# Independent A10 Help source/geometry review

Spec verdict: **PASS for the bounded Help redesign.**

Quality verdict: **APPROVED for source, geometry and behavior. No Critical, Important or Minor findings.**

**This is not a PNG/visual PASS.** Parent vision review, merged-revision verification, native platform and all-screen/product acceptance remain open.

## Scope and exact identity

Reviewed `/home/xertrov/src/oneline-ccc/.worktrees/native-help-design` at **`f90fa47d7cc10676db57ccd80cf937b3cb4ed36c`**, base **`028f14f6d910fed499045b85dd83060f9b463c6f`**. Read the complete author report, supplied package, all four changed files and the directly relevant catalog/input/control, shell/layout, modal/resize, sanitation and final-page evidence boundaries.

The package's patch section exactly matches `git diff --unified=10 028f14f f90fa47`. Package SHA-256: `011d6d6cda718f1768568cc2c76bfbc5b1d0e9fec8d877f7aa37fbcc578afb9d`. All changed files and inspected shared callers match the reviewed HEAD. `git diff --check` passes. The production delta is `help.go` and the Help case of `panels.go`; the other two files are tests. No Review/receipt hunk, binding, state/editor implementation or capture-tool change appears.

No production/source/git edits, agents, network/operator-state access or attention/notification tools were used. Only focused offline tests were run, with checkout-local build/temp caches. This report is the only authored review file.

## Checked and accepted

- **Useful first page and retained bindings:** `help.go:18–59` projects existing catalog keys into movement/action, section/command, current-control, text-entry and paging groups. Every currently keyed catalog command is included. Unkeyed destinations remain reachable in the unchanged command palette and More menu; the narrow Help page is no longer occupied by bare destination names. The first-page regression requires actual visible movement/activation guidance and Close at 40×12. The wide regression requires the complete baseline guidance on one readable 120×40 page.
- **Context agrees with activation:** `helpControl` (`help.go:62–95`) follows the relevant `input.go:90–115` ordering: explicit unavailable reasons; client rows and navigation before the busy-operation guard; then checkbox/default activation. It evaluates management eligibility for the focused client rather than the previously selected client, preserves the ability to inspect read-only clients, blocks management wording while busy, distinguishes body navigation links from horizontal tabs, and retains actual idle-cancel reasons. Tests cover read-only Pi, More, a Settings navigation link, a checkbox, unavailable cancellation and busy Configure. The generic Enter/Space/local-arrow descriptions replace three context-inaccurate catalog labels without changing their keys or actual bindings.
- **Text and modal guidance is truthful:** the unchanged input/editor paths support the described typing/paste, caret, save/cancel, API-key input/options/persistence and Ctrl+C exception. `commands.go:54–63` implements the stated forward/reverse Help keys, bounded first/last movement and Enter/Escape/`?` close. Close still restores invoking focus/scroll; reading/paging Help introduces no draft/backend operation. Origin tests check restored focus, an unchanged draft and zero mutation calls.
- **Sanitation precedes styling:** dynamic labels/reasons pass through `public`; complete descriptions and keys are sanitized again before wrapping or styling (`help.go:99–117`). Group/title/Close text is fixed application text or numeric page metadata. The Help renderer preserves these already-sanitized styles instead of stripping and rebuilding them (`panels.go:48–56`). No secret/editor buffer is newly projected into Help. NO_COLOR tests reject style escapes.
- **Wrapped-row geometry remains authoritative:** `help.go:129–209` uses the same body width and actual header/footer footprint as layout, reserves wrapped page-title/Close rows, and packs complete binding rows beneath group headings. A fitting entry moves intact; oversized content continues with repeated headings and no dropped remainder. A heading is inserted only when at least one content row can follow. Optional inter-group spacing is counted before insertion. Page count remains bounded by content-row count, supporting the conservative title-digit reservation. Below supported dimensions, capacity stays positive and chrome yields to content rather than causing a zero-capacity loop.
- **Coverage was changed for the new contract, not relaxed to hide clipping:** all original per-page viewport, width, visible title/Close, reverse-boundary, resize and focus checks remain. Canonical descriptions are still checked for keyed commands except the three intentionally generalized labels; destination-only entries are checked in the palette instead. New tests independently require complete editor/paging prose, nonempty pages within capacity, no orphan headings and readable layout across all five required sizes × dark/light/mono/NO_COLOR. Existing repeated first/last/reverse/resize tests are unchanged and pass.
- **Final-page evidence remains exact:** capture/helper and verifier files are unchanged by this range. The Go predicate still rejects Help unless `page.Number == page.Total` (`visual_capture_test.go:176–184`). The artifact verifier still requires recorded page numbers to match the production frame and requires final-page equality (`tests/tui-visual/evidence.mjs:66–72`). Nothing substitutes a merely open Help modal or nonempty page for actual final-page arrival.

## Execution evidence

Fresh reviewer command, with `GOMAXPROCS=2`, `GOPROXY=off`, `GOSUMDB=off`, and checkout-local `GOCACHE`/`GOTMPDIR`:

```sh
go test -p 2 -count=1 ./internal/tui -run '^(TestHelp|TestFix1Help)'
```

**PASS, exit 0; package 0.318s.** This includes the new Help cases and retained boundary/resize tests. No broad suite was rerun.

Inspected saved `red.log`, `context-red.log` and `wide-red.log`: they demonstrate the original missing useful guidance/context and the intermediate two-page wide layout. `iteration2-focused.log` is an explicitly failed provisional attempt with split prose, not GREEN evidence. Final `iteration2-normal.log` and `iteration2-race.log` record successful TUI packages at 2.109s and 22.180s. Those full-package outputs remain historical author evidence; this reviewer did not independently rerun the full normal/race suites or artifact verifier.

## Unknowns and handoff limits

Actual native/reference PNG judgment belongs to the parent. I did not open or grade the author's 23 final native frames, crops or reference renders. Their documented comparison uses a Gateway browser parent versus a Dashboard native parent and does not establish an exact full-shell/state match. Implementer inspection and artifact integrity are not independent visual acceptance.

The current fixtures primarily capture Help invoked from its Help control; other origin/availability contexts have source/test coverage, not equivalent origin-specific native PNG proof. Broader real-input journeys, merged-revision recapture, ANSI-16/256, Windows/macOS terminals, installed-build fidelity and every other screen remain separate gates. Reading the unchanged exact final-page predicates establishes no new claim that every setup key was delivered through a PTY.

This approval applies only to the exact reviewed Help delta. Preserve it separately from the parent's concurrent Review/receipt work and complete the parent's visual disposition and integrated verification before advancing affected acceptance.

=== DONE ===
