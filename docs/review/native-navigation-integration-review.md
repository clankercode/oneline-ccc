# A3 bounded integration review

Spec verdict: **PASS for the fixture-integration and backlog-retention scope.**

Quality verdict: **PASS for this diff. No Critical, Important or Minor findings.**

**Native visual/interaction acceptance remains OPEN and rejected pending the required evidence.** These verdicts do not approve native appearance, every-screen usability, capture-tool integrity or platform acceptance.

## Scope and evidence

Reviewed the supplied `da17aa0..a64a16f` diff: `ce63417` updates two CLI fixture journeys; `a64a16f` retains acceptance/evidence obligations in two planning documents. No production code changes appear. Diff SHA-256: `d1820a115300533d613ef55c613711083a5871fa505aaa167d1006097c4bd29f`.

All six saved postimage hunks match the current four files. Read the three specified logs and traced only the relevant input/editor/navigation/backend-lifetime context. Checked the previously reported capture and masking gaps against the new backlog entries. No tests, builds, agents, git operations, network/operator-state access or notifications were performed. This report is the only write.

| Saved evidence | Observed result | What it establishes |
| --- | --- | --- |
| `A3-merged-tests.log` | RED: foreground `key` cannot reach the mutation; dirty Ctrl+C guard journey fails | The old fixture navigation is incompatible with the integrated focus model |
| `A3-integration-focused.log` | CLI package `ok`, 0.157s | Supplied focused GREEN evidence; output is package-level, without individual test names or invocation flags |
| `A3-integration-full.log` | All listed tested packages `ok`; CLI 0.461s | Supplied merged GREEN evidence; no visual, race-detector or native-platform PASS follows from this log |

The logs are saved execution evidence, not independently rerun results. Their output does not itself attest the exact command, revision or environment.

## Lifecycle and dirty-guard guarantees

- **Apply still executes the reviewed mutation.** F6 moves from Nav to Clients; Space stages management rather than merely changing focus. The palette returns the plan command, its result enters Review, and the unchanged acknowledgment path reaches Apply (`foreground_lifecycle_test.go:81–93`). Production still requires an acknowledged, unchanged exact plan before calling backend `Apply` (`internal/tui/async.go:148–159`). No direct backend shortcut was added.
- **Key reaches SaveKey, including persistence.** Tab crosses from Gateway's endpoint form to the key action; the new assertion requires the hidden editor before paste. Tab → Space → Tab → Enter selects persistence and Save (`foreground_lifecycle_test.go:95–105`; `internal/tui/editor.go:47–75,116–128`). The fixture rejects nonpersistent saves before entering `mutate` (`foreground_lifecycle_test.go:46–50`), so a passing backend-entry wait cannot be satisfied by merely returning a cursor/focus command.
- **Service retains explicit confirmation.** Its unchanged journey selects Install, opens the confirmation, moves off the safe default and confirms (`foreground_lifecycle_test.go:106–113`; `internal/tui/commands.go:107–140`). That command calls backend `Service` through the production lifetime wrapper.
- **Cleanup assertions are unchanged.** Each subtest runs the selected command through Bubble Tea and the production `runForeground` boundary. It waits for `f.entered`, which is emitted only inside the overridden backend mutation, then requires cancellation propagation, terminal-program return, continued boundary blocking until release, and the final safe marker (`foreground_lifecycle_test.go:120–173`). Admission-closure/drain coverage at lines 176–219 is untouched. The existing headless `WithoutRenderer` setting was neither introduced nor expanded; these assertions establish mutation lifetime, not rendered-terminal restoration.
- **Dirty quit is strengthened.** F6/Space now stages the actual client draft and adds an explicit Draft precondition. The original assertions still require Ctrl+C to return no quit command and show the unapplied-draft dialog, then Escape to preserve the draft (`signals_test.go:179–196`). No skip, timeout relaxation or assertion removal appears.

## P0 and backlog retention

`docs/plans/remaining-items-so-far.md:8–20,76–267` retains the user's detailed rejection: unlike the approved preview, ugly/unimpressive/cumbersome, inconsistent controls and awkward Down traversal. It preserves the rich native visual target, coherent local controls/focus regions, discoverable destinations, ordinary terminal journeys and draft/safety requirements.

The acceptance contract still covers all ten screens, seven baseline modals, seven editors and the added More modal; missing mockups remain work. It retains five terminal sizes, dark/light/monochrome, color/ASCII/Unicode cases and real Windows/macOS evidence. Actual native PTY output must produce faithful terminal PNGs; HTML recreations cannot stand in for native evidence. Independent implementers/reviewers, resumable fan-outs, alternating vision → geometry/source → vision → geometry/source, individual readable images/crops, re-rendered dependants and same-final-revision visual/geometry/interaction PASS gates remain mandatory.

All 35 local links in the remaining-items document and all three campaign links resolve. Supporting design, flow, visual-system, inventory, preview-source, verification and lifecycle documents remain linked. Original backlog IDs and open checkboxes are preserved; campaign milestone A now includes item 21.

The additions accurately distinguish the first four reviewed source fixes from pending rendered acceptance (`remaining-items-so-far.md:269–300`). Item 19 retains the populated-key-editor and intermediate-focus evidence gap, including renderer coalescing (`:412–422`). Item 21 retains blank/header-only/wrong-target images, scenario/cell/text consistency, cleanup after capture-leader exit, immutable resume provenance, untracked build inputs, served-reference provenance, executed negative regressions and preservation of historical evidence (`:434–445`). Neither addition claims those defects are fixed.

## Acceptance still open

Complete native/mockup comparisons, repeated independent visual/geometry rounds, real keyboard journeys, trustworthy capture evidence, the matching installed build and native platform evidence remain required. The earlier Help warning about wrapped content scrolling the title/current-page above the viewport also remains a concrete overlay-readability case (`A3-rereview1.md:47`); corrected paging semantics alone do not close it. Items 1 and 19–21 retain their outstanding terminal/evidence obligations.

No visual PASS is inferred from fixture GREEN, source fixes or artifact verification. The bounded diff is acceptable while the broader P0 remains open.

=== DONE ===

## Addendum — native navigation documentation

Reviewed HEAD: **`24b71d277a82005bb2fdf1d3567a462d04c58a15`** — `docs: explain native focus regions and direct section navigation`.

Spec verdict: **PASS for the documentation diff.** Quality verdict: **PASS; no Critical, Important or Minor findings.**

Verified `A3-navigation-doc-review.diff` exactly matches this commit's `docs/native-app.md` change. The current document and navigation/input/catalog/control sources match that HEAD. Review used read-only source/git inspection; no tests, agents or notifications.

- First-use F6 → client-list Up/Down → Space → Enter matches region movement, client selection, management staging and entry into Models (`docs/native-app.md:29–30`; `internal/tui/navigation.go:110–154`; `internal/tui/input.go:39–40,59–80,85–88,108–111`).
- The shortcut table agrees with forward/reverse region traversal and saved-anchor fallback, full Tab traversal, horizontal navigation-bar movement, direct bracket section switching, local arrows, activation/toggling, Back and guarded exit. Editor/overlay behavior remains contextual rather than implying that ordinary typed characters navigate (`docs/native-app.md:40–55`; `internal/tui/navigation.go:42–74,93–154`; `internal/tui/input.go:11–83,256–264`; `internal/tui/commands.go:13–25,36–143`).
- More's six destinations match the section catalog and menu dispatch; visible Help/Commands provide the documented traversal fallback (`docs/native-app.md:52–55`; `internal/tui/navigation.go:5`; `internal/tui/commands.go:43–53`; `internal/tui/controls.go:125–126`).
- The explicit acceptance paragraph retains the rejected implementation's **open** visual/interaction status and requires per-screen native/mockup comparisons and independent review (`docs/native-app.md:59–63`). It makes no claim that documented shortcuts certify visual fidelity or smooth complete journeys. All prior P0 and lifecycle acceptance limits remain unchanged.

=== DONE ===
