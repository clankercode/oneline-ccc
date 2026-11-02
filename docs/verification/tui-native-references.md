# Secondary TUI design references

Recorded: 2026-11-03. These are browser design targets, not screenshots of the
native app or evidence that the rejected TUI now meets the approved standard.
The [P0 requirements](../plans/remaining-items-so-far.md#p0-native-tui-visual-quality-and-interaction-remediation)
and [visual system](../design/tui-visual-system.md) remain the acceptance authority.

The [companion](https://clankercode.github.io/oneline-ccc/tui-native-reference.html)
extends the [approved primary preview](https://clankercode.github.io/oneline-ccc/tui-preview.html)
with Gateway, Service, Repair, SSH, Utilities and Settings. Its selectors expose
dark/light/monochrome appearances, wide/compact/narrow layouts and state variants.
Eight layers cover Help, Commands, More, quit, discard, SSH handoff and monitor
install/uninstall. Seven editors cover filter, model, context budget, compaction,
endpoint, credential reference and hidden API-key entry.

Use, for example, [Gateway](https://clankercode.github.io/oneline-ccc/tui-native-reference.html?screen=Gateway),
[Settings](https://clankercode.github.io/oneline-ccc/tui-native-reference.html?screen=Settings)
or [Commands](https://clankercode.github.io/oneline-ccc/tui-native-reference.html?screen=Gateway&overlay=palette).
The page uses fixture data and page memory. It accesses no gateway, credential
store, configuration, installer or service. Reload resets edits. Unavailable
actions retain reasons and do not imply an operation succeeded.

## Source and bounded verification

Reference source was independently reviewed through task revision `46b50cd`;
the integration-safe test/CI correction is task revision `f285ff8`. Source files
are [HTML](../../site/tui-native-reference.html),
[styles](../../site/tui-native-reference.css) and
[behavior](../../site/tui-native-reference.mjs). The original four-screen
preview was preserved. Task revisions identify the reviewed worktree inputs;
record the integrated revision and asset hashes when pinning a comparison.
The [reference re-review](../review/native-reference-rereview.md) and
[test/CI review](../review/native-reference-integration-review.md) retain the
independent scope, finding dispositions and evidence limits.

The focused [browser verifier](../../tests/tui-native-reference-browser.mjs)
exercises visible initial/return focus, visit anchors, browser history, appearance
retention, service ownership and unavailable confirmations, useful utility
choices, budget/compaction and credential-reference drafts, wrapped command rows
at 320 pixels, and the fixture's reload/storage/network boundary.

```sh
node tests/tui-native-reference-browser.mjs --fix1 --record=local
```

Set `PLAYWRIGHT_MODULE` to an installed Playwright module path when needed;
relative paths resolve from the test file. The local CI-equivalent run used
pinned Playwright 1.61.1 after the primary preview journeys. Browser scratch,
measurements, structured outcomes and PNGs stay under checkout-local
`.cache/artifacts/`.

**Publishing the additional Pages gate is pending.** GitHub rejected the workflow
update because the current OAuth credential lacks `workflow` scope. The reviewed
change is preserved on local branch `wip/reference-ci-wiring` at `2389c88`;
published source and documentation use a separate history without a workflow
mutation. Existing Pages checks still run; the companion gate above must also be
run locally until its CI wiring can be published with suitable authorization.

The controller's integrated focused run passed **14 journeys**, with zero page
errors and zero external requests. The earlier broad task run recorded 159
reference captures, 161 render checks and 11 journeys; it preceded the scoped
fixes and is not a rerun of every final state. Focused captures and source review
cover the changed regions; neither result certifies native terminal geometry.

## Native acceptance remains open

The native app must still be captured through actual terminal input and compared
with pinned references across every screen, layer, editor, size and appearance.
Alternate vision inspection, including crops, with independent source/geometry
review; fix and rerender until both pass the same final revision. Preserve
review authority, masked secrets, drafts and safe terminal ownership throughout.
Browser focus checks and the addition of these targets do not close those gates.
