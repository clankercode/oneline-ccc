# A3b scoped rereview 1

## Spec verdict: APPROVED within the requested source/geometry scope

**F1 ADDRESSED. F2 ADDRESSED.** No remaining Missing, Extra or Misunderstood requirement found in these two corrections and their immediate regressions. The selected-client action target, explicit draft/applied distinction, Unicode cell budget and matching-filter guidance satisfy the bounded brief. This is not all-screen visual acceptance or P0 completion.

## Quality verdict: APPROVED within the requested scope

No new Critical, Important or Minor finding identified. No PARTIAL or OPEN source finding remains from the original two findings. Changed-state PNG acceptance and the previously disclosed integration/body dependencies remain open separately.

## Exact scope and inputs

- Worktree: `/home/xertrov/src/oneline-ccc/.worktrees/native-composition`.
- Exact comparison: `fe62ec8f05d3b9618f9b85e78dd75f2cad0ed031..63006cf824517ac1a7cfd1c0d480c519aeb06ff5`.
- HEAD verified before and after checking: `63006cf824517ac1a7cfd1c0d480c519aeb06ff5`, one corrective commit.
- Three changed tracked files only: `internal/tui/components.go`, `internal/tui/dashboard.go`, `internal/tui/composition_test.go`; 107 insertions, 4 deletions. Tracked tree remained clean. The existing untracked capture helper and `tests/tui-visual/` remain outside this diff.
- Read the original `A3b-review.md`, supplied rereview brief/diff, worktree `A3b-fix1-report.md`, original visual notes and `A3b-fix1-visual-notes.md`, relevant source/tests and focused/final logs. Review remained limited to accepted F1/F2 and regressions.

## Finding dispositions

### F1 — ADDRESSED: dirty focused Codex remains explicit while Claude is selected

**Source:** `internal/tui/dashboard.go:133-145` reserves `Draft: off/on` before sizing the client name. Lines 238–247 separately reserve `Applied: on/off` and measure the model/default suffix and applied-state tail in display cells. The tiny non-selected branch retains these two rows without adding body rows. `internal/tui/components.go:252-264` still derives primary action names from `m.client()`, the persistent selection; activation/navigation/selection code is unchanged.

**Reproduction checked:** 40x12 six-client fixture, **Space → Down → Shift+Tab**. Selected client remains Claude Code, focused client is Codex, and only Codex's draft Enabled field changes. Fresh rendered output includes:

```text
Selected: Claude Code / applied setti…
> [ ] Codex    CONFIGURED / Draft: off
  gpt-6.1-sol · Standard  Applied: on
   Configure Claude Code
  Review Claude Code draft
```

The long CJK/combining-character version retains the draft/applied labels and adjacent Standard tier while shortening the display name/model. The stored model and draft values are not rewritten by rendering. Independent probes also exercised the reverse case, applied off with staged on, including emoji in long model/name values. No further fix requested.

**Regression strength:** `TestCompositionTinyDraftSurvivesFocusSelectionSeparation` at `composition_test.go:442` uses the actual three key messages, checks exact expected Codex draft, selected/focused identities and every other client's unchanged draft, and checks visible draft/applied cues, selected-client Configure/Review labels, model/default adjacency and frame invariants. This directly covers the original missing focus/selection separation rather than merely checking draft memory. It is rendered View/state evidence, not a PTY keyboard journey or PNG inspection.

### F2 — ADDRESSED: active matching filter is visible and editable without shrinking the body

**Source:** `internal/tui/components.go:174-179` reuses the existing second hint row for `Filter: <bounded query>  / edit/clear  Esc back`. Its remaining query width accounts for the fixed ASCII prefix/suffix; `ansi.Truncate` budgets the actual query in display cells. Modal/editor-specific hints override this afterward at lines 181–207.

**Reproduction checked:** 40x12 fixture, **/ → paste Codex → Enter**, then **/ → Ctrl+U → Enter**. While matching, the frame shows the filter value, complete `/ edit/clear`, `Selected 1/1` and all three Configure/Review/Launch labels. After clearing, the filter cue disappears and `Selected 1/6` returns. A long matching CJK/combining-character query also passes; independent probes added emoji. Every checked focus position retains the active query cue and valid frame geometry. No further fix requested.

The hint replacement intentionally uses the existing secondary hint row; it does not add a row. Current-region/F6 and Help/paging hints remain separate. Independent combined-state probes verified the available body-row count is unchanged when filtering a dirty, non-selected focused client, and that clearing preserves the draft, selected client and focused control.

**Regression strength:** `TestCompositionTinyMatchingFilterContextAndClear` at `composition_test.go:486` opens the real editor, pastes/saves, reopens, clears with Ctrl+U and saves. It checks matching and restored counts, filter value/cue, all primary-action labels, all drafts unchanged and frame invariants. This resolves the earlier gap in matching-filter coverage; zero-results evidence alone is not being substituted.

## Verification and regression evidence

Fresh focused gate passed, **9 top-level tests / 13 including subtests**:

```sh
GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local \
GOTMPDIR="$PWD/.cache/a3b/tmp" TMPDIR="$PWD/.cache/a3b/tmp" \
go test -p 2 ./internal/tui \
  -run 'TestComposition(Tiny|ReviewAck|ManualPaging|CompactDefault|LongContent|ClientFocus|NarrowNavigation|EditorFooter)' \
  -v -count=1
```

Result: `ok github.com/clankercode/oneline-ccc/internal/tui 0.971s`. This includes the prior ASCII client-focus regression, compact primary-action visibility, long content, navigation/footer, key-editor footer, Review acknowledgment viewport and contiguous manual paging checks.

Independent `TestA3bScopedRereview` probes also passed in `0.111s`. Test source and Go overlay JSON were held in Linux memory file descriptors, so no source/test file was written. The invocation used `go test -p 2 -overlay /proc/<live-python-pid>/fd/<overlay-fd> ./internal/tui -run '^TestA3bScopedRereview$' -v -count=1` with the same offline/local, GOMAXPROCS and checkout temporary-directory settings. Coverage:

- 24 combined cases: ASCII on/off × NoColor on/off × dark/light/mono × applied management on/off. Each uses the three-key journey with long CJK, combining-character and emoji values, checks draft/applied cues and selected-client action labels, then edits/clears a matching filter while checking unchanged body budget, draft, selection and focus.
- Four additional matching-filter cases: ordinary/long Unicode query × ASCII on/off. All focus positions retain the query/edit-clear cue and frame bounds; returning to selected Codex retains all three primary actions and exactly 12 rendered rows.

Fresh `gofmt -l` on the three changed files and `git diff --check fe62ec8 63006cf` produced no output.

Historical `.cache/a3b/logs/fix1-red.log` contains the expected missing Draft/Applied and Filter/edit-clear failures. `fix1-green.log` and fresh execution agree on the corrected behavior. The initial `fix1-normal.jsonl` and `fix1-race.jsonl` preserve the one-cell ASCII model truncation failure, rather than hiding it. Final `fix1-normal-final.jsonl` and `fix1-race-final.jsonl` each contain **79 top-level / 126 total passing tests, zero failures/skips**. Those final normal/race runs are implementer evidence examined here; no fresh full package, race or repository-wide gate is claimed by this scoped rereview.

## Explicit limits and remaining evidence

Source/geometry approval applies only to the exact revision and scope above. No PNG was opened in this rereview, and passing tests do not establish visual fidelity.

The fix1 manifest identifies exactly one **Dashboard/parity/no-results, 40x12** capture (`artifacts/a3b/fix1-no-results/manifest.json`, SHA-256 `fc52e60f4a866078c0cb8e7e45f8fb71be538851f4a9e7bed6d0dafa0a5c6c8b`). The report/visual notes correctly describe it as no-results evidence. It does not establish matching-filter or dirty-focus PNG acceptance. Independently developed A2b semantic variants and their post-protocol state checks were not reviewed or presumed accepted here. The old 36-frame index remains historical evidence at `fe62ec8`.

Review's full-width authority footprint, pagination/acknowledgment implementation and contiguous-paging source are untouched by this fix. The focused Review/paging regressions passed; this rereview does not replace the earlier stronger all-row Review probe or approve a redesigned Review body. Help redesign, other screen bodies, A2 capture-tool acceptance, exact integrated recapture and broader P0/native/platform acceptance remain open under their existing owners. Help candidate `c643366` is not assumed integrated or duplicated.

No agents, notifications, pushes, network calls, real backend calls, operator configuration or credentials were used. No source/git/operator-state mutation was made. This designated report was written last; verification produced ordinary Go build/temporary artifacts.

=== DONE ===
