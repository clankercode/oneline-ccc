# Independent A3b source and geometry review

## Spec verdict: changes requested

The bounded shared-shell/Dashboard implementation is substantially present, but compact state fidelity is incomplete. **Missing:** the unsaved-draft indicator for a focused, non-selected client at 40x12 (F1), and active-filter context in the tiny composition (F2). **Extra:** no unauthorized production scope found; the compact client window and full-width Review exception have recorded coordinator rulings in `A3b-progress.md` (R3 and R2). **Misunderstood:** the tiny non-selected-client branch treats applied management as sufficient summary state and drops the separate draft signal that `clientRows` already supplies.

Authority read: the supplied A3b brief/review brief, P0 in `docs/plans/remaining-items-so-far.md`, approved visual/usability contracts and Dashboard specimen source, native screen inventory, implementer report/index, and coordinator vision rounds 1–3. The P0 rejection remains authoritative. This review does not close P0 or approve the full native interface.

## Quality verdict: changes requested

**0 Critical, 1 Important, 1 Minor finding.** Fix F1 and rereview the exact resulting revision. F2 should be addressed with the compact correction. No production source was changed during this review.

### F1 — Important: tiny focused-client rendering suppresses a real pending draft

**Source:** `internal/tui/dashboard.go:229-236`; compare the draft indicator retained by `clientRows` at lines 154–156 and by the selected tiny branch at lines 242–246.

**Scenario/reproduction:** Use the six-client composition fixture at 40x12, initially selected/focused Codex. Press **Space**, **Down**, then **Shift+Tab**. Space stages a Codex management change; Down selects Claude Code; Shift+Tab focuses Codex without changing the persistent selection. This uses existing keyboard behavior, without staging anything on an unsupported client.

The independent probe confirmed `selected=client-1`, `focused=client:codex`, and Codex draft unequal to its baseline. The actual frame contains:

```text
Selected: Claude Code / applied set...
> [ ] Codex                 CONFIGURED
  gpt-6.1-sol / Standard / managed
  Configure Claude Code
  Review Claude Code draft
Unavailable: Read only: this native
```

There is **no `Draft` indicator**. The checkbox reflects the staged disablement while `managed` reflects the applied state, with their discrepancy left unexplained. The branch replaces the original model/management/draft subrows with a new two-row entry and never consults `drafts != baselines`. Scrolling does not recover that focused client's discarded draft subrow: the appended applied detail belongs to the separately selected client.

**Effect:** an ordinary traversal hides a pending management change exactly where the operator is inspecting/toggling that client. The draft remains in memory; this is a loss of visible state, not data loss or unauthorized mutation. It violates the usability contract's explicit text for unsaved changes and the compact fidelity requirement. The compact tests cover selected dirty Codex and arrow traversal, but not dirty focus separated from selection.

**Fix:** retain an explicit draft cue in the tiny non-selected entry, preserving the applied/draft distinction and selected-client action target. Budget its cells deliberately, including model/default adjacency. Add a regression using the three actual keys above; assert draft state, persistent selection, focused control and rendered `Draft` cue, with no extra staging from traversal. Exercise long Unicode values as well.

### F2 — Minor: tiny matching filters become invisible

**Source:** `internal/tui/dashboard.go:210-216`, with the unqualified matching count at line 248; tiny footer hints in `internal/tui/components.go:158-173` do not supply the omitted filter context.

**Scenario/reproduction:** At 40x12, use `/` to filter the six-client fixture to `Codex`, then save the filter. The frame shows Codex, `Selected 1/1` and its primary actions. Neither the active filter nor a filter/edit/clear hint appears. The compact branch below 13 rows omits both the filter line and `/ filter` instruction. The zero-results case does show Clear filter, so the gap concerns filters that still match clients.

**Effect:** the operator can mistake a filtered window for the complete client inventory, and local arrows appear to have no neighboring clients. The `/` editor still works and Help remains reachable; this is a context/discoverability defect, not removal of the underlying control.

**Fix:** retain a bounded active-filter cue and `/` edit/clear guidance in existing summary/footer space. Preserve the primary-action row budget. Cover a matching filter, clearing it, and a long Unicode filter at 40x12.

## Checked and fine within this modality

- Exact reviewed worktree: `/home/xertrov/src/oneline-ccc/.worktrees/native-composition`; HEAD `fe62ec8f05d3b9618f9b85e78dd75f2cad0ed031`; comparison `git diff da17aa0 fe62ec8`. Five improving commits, exactly six tracked files. The only untracked source/tool paths remain `internal/tui/visual_capture_test.go` and `tests/tui-visual/`. No navigation/input/backend production file is in this diff.
- Shared surface roles, structural active-tab brackets, one control-focus marker, selected text, default-tier adjacency and separate capability wording are implemented. NoColor bypasses the ANSI styles; ASCII separators are explicit. No animation was introduced. Source checks do not establish contrast or terminal appearance.
- Header/footer budgets use composed row counts. The normal multiline public-status path is normalized before rendering. An artificial multiline `busy` value can still produce an extra physical row, but all current production assignments use fixed single-line operation names; this is not promoted to a reachable finding.
- Compact controls retained their IDs and traversal. Default selected Codex exposes Configure/Review/Launch in the tiny viewport. All compact focus positions were inspected as rendered strings; wide 100x30 client traversal also retained visible focus and identity. Disabled actions show next-step guidance and an unavailable reason, with longer explanations continuing in the scrollable document.
- Independent contiguous-paging probes placed 90 distinct tokens in applied detail and traversed PgDn from Nav, Clients, Configure and Commands focus at 40x12. Every token was observed; no skipped token was found. This supplements the implementer's single-focus paging regression.
- **Review exception:** the body remains at full terminal width in both layout and final output, preserving the existing `panels.go` wrapping footprint. A stronger independent probe checked **every `reviewPages()` row in the actual View at `review:read` focus**, including long Unicode plan identity, revision and 400-character before/after values, at 120x40, 100x30, 80x24, 60x20 and 40x12. All rows were present; the smallest size produced 41 pages. No Review-value clipping was reproduced with those inputs. Review acknowledgment/apply semantics and resize invalidation are outside the changed files and remain covered by the package regressions. This is bounded source/geometry evidence, not proof that a user read the pages or visual approval of Review.

## Validation and evidence assessment

Fresh normal package gate passed:

```sh
GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local \
GOTMPDIR="$PWD/.cache/a3b/tmp" TMPDIR="$PWD/.cache/a3b/tmp" \
go test -p 2 ./internal/tui -count=1
```

Result: `ok github.com/clankercode/oneline-ccc/internal/tui 2.208s`. `git diff --check da17aa0 fe62ec8` also passed. No full-repository or fresh race gate is claimed.

Focused probes used Go's `-overlay` with virtual test source and overlay JSON held in Linux memory file descriptors (`/proc/<live-python-pid>/fd/<fd>`), avoiding source/test-file writes. They used `GOMAXPROCS=2`, `-p 2`, offline/local toolchain settings and the same checkout temporary directory. The first attempt encountered unavailable Python `os.memfd_create`; libc `memfd_create` supplied the same facility. The decisive probe body was:

```go
m := compositionFixture(t)
m.width, m.height = 40, 12
before := m.baselines["codex"]
press(m, "space")
press(m, "down")
press(m, "shift+tab")
if m.selected != "client-1" || m.focused().id != "client:codex" ||
    m.drafts["codex"] == before {
    t.Fatal("unexpected setup")
}
if !strings.Contains(ansi.Strip(m.View().Content), "Draft") {
    t.Fatal("focused Codex unsaved-draft state omitted")
}
```

Observed result: **FAIL**, `focused Codex unsaved-draft state omitted`. Other probes checked complete Review rows, all compact focus states, long client names/model IDs/reasons, matching/zero-result filters and contiguous paging. Their passing assertions are not visual acceptance.

The evidence index names 21 added tests. Historical logs contain matching failure records for the claimed 19 RED tests; the two supplemental tests are correctly not claimed RED. The corrected alphanumeric paging RED is distinguished from the earlier wrapping-sensitive experiment. Recorded final normal/race JSON logs each contain 120 passing test/subtest events and no failures/skips, matching the reported 77 top-level tests. These are historical implementer runs, not independently rerun race evidence.

Test limits matter: `TestCompositionReviewAckRetainsWholePageViewport` checks start position and frame bounds rather than every page value; the independent row-presence probe supplies additional bounded evidence. `TestCompositionLongContentEveryControlAndScroll` mainly checks frame/focus invariants. `TestCompositionPrimaryActionHasSeparateTreatment` checks ANSI presence rather than a strong comparison of primary versus auxiliary style. None proves visual fidelity. The dirty-selected compact test and arrow-only window test miss F1's focus/selection separation.

Canonical manifests checked:

- `artifacts/a3b/provisional-compact-head-dashboard/manifest.json` — 15 captures; SHA-256 `6cfe37a4b2ce164cfeb111852bea7f26f7243f48f223a059f07b6dc706f9e724`.
- `artifacts/a3b/provisional-compact-head-shell/manifest.json` — 20 captures; SHA-256 `3876b342ac2f228df7ba654a6a179538af72b98cb05134863a74d374e4a109f9`.
- `artifacts/a3b/provisional-compact-head-nocolor/manifest.json` — 1 capture; SHA-256 `8e4d6f421354bdc832b941e994acc88535557b775ec1666594647670b148a224`.

All three hashes match the index; all recorded capture-file hashes across the 36 records matched the files on disk. The metadata records exact production HEAD and dirty/untracked development fixture/tool inputs separately. This file-integrity check does not independently certify capture semantics, source-input aggregation or PNG appearance.

## Explicit acceptance limits and handoffs

- **No visual PASS is issued.** This review opened source and geometry evidence, not PNGs. Coordinator vision rounds inspected provisional images/crops and reported V1–V5 improvements; those observations do not replace independent same-revision final acceptance.
- A2 transparent-PNG and semantic-variant issues remain open. The archived `more/last-item` and `help/last-page` labels are not accepted as proof of Settings or the last Help page. Reviewed-tool integration and exact integrated-source recapture remain required.
- Help readability candidate `c643366` is separately approved work; it is not integrated or reapproved by this six-file review. The known wrap-aware Help pagination gap remains a dependency here. No duplicate fix was made.
- Models, Review body redesign, Jobs and secondary bodies remain outside A3b's redesign claim. Shared shell coverage does not approve those bodies, editors or overlays. Review's full-width exception must remain until pagination, composed budgets and acknowledgment coverage are changed together.
- No live backend, operator configuration, credentials, real terminal keyboard journey, child/service lifecycle, platform trial or full P0 matrix was exercised. No agents, network calls, notifications, git/source mutations or operator-state changes were performed. The designated review report is the only authored persistent artifact; Go verification produced ordinary build/temporary artifacts.

=== DONE ===
