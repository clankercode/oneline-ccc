Spec verdict: APPROVED for the scoped R1 legacy/current-schema boundary.
Quality verdict: APPROVED WITH NIT — no new blocking finding; one inherited malformed-observation gap noted below.

# A2 fixround3 independent re-review

Reviewed worktree: `/home/xertrov/src/oneline-ccc/.worktrees/native-captures`.
Exact range: `86c3486a016cb6922523c2d8698b8b879fac7453..4b096c1aaa8dc813043f84ed544a721982800f89`.
Locations below are relative to that worktree.

Read the full fix3 brief, canonical `A2-rereview2.md`, full saved diff, implementation report and all named fix3 RED/GREEN logs. Saved diff exactly equals `git diff --unified=8 86c3486 4b096c1`; SHA-256 `4335b7219ea022850a9cc884994f8a66325a62c007ed41bc5f9358cb3f6f38e5`. The commit changes five files under `tests/tui-visual/**`, with 81 insertions and four deletions. HEAD is exact and reviewed worktree status remained clean.

No source edits, commits, git mutations, agents, notifications, attention tools, operator-state access, network, builds, new captures or matrix reruns. Focused execution used file reads and in-memory mutations/adapters. The three renderer tests that create disposable artifact directories were excluded. This report was written last and is the only review-created file. The main checkout has an unrelated modification to `docs/plans/remaining-items-so-far.md`; this review did not edit it.

## Findings and original dispositions

**R1 — CLOSED.** At `tests/tui-visual/evidence.mjs:55`, absent legacy client observations now bypass only the unavailable client-state predicate. Recorded clients still undergo that predicate before the legacy return. Schema 3 rejects a missing client key even for an empty-state request and retains required before/after outcome checks.

Fresh standalone verification of the actual unchanged original probe now succeeds with:

```text
legacy-ansi-only; original production View unavailable;
semantics=legacy; distinguishing outcomes not recorded; client observations unavailable
```

The actual first original baseline row also passes the shared gate unchanged. These positives use historical files without `fixture.clients`, rather than relying solely on downgraded modern fixtures. Both legacy schemas also pass focused missing-observation controls. Recorded inconsistent clients fail in schemas 1, 2 and 3; current baseline/empty rows with deleted clients fail; missing semantic outcomes fail. `semantic_before` remains required by the unchanged explicit guard. No historical clients or semantic outcomes are reconstructed.

| Prior finding | Disposition in this round |
| --- | --- |
| I1a — opacity | Preserved. Fresh exporter tests reject valid real-RGB PNGs re-encoded with alpha 0/128, accept alpha 255, and reject transparency after removing legacy client observations. |
| I1b — actual semantic state | Preserved for the reviewed specimens and predicates. All five genuine schema-3 representatives pass; fresh refusal/value/draft corruption and post-key drift checks fail as intended. Legacy distinguishing-state proof remains explicitly unavailable. |
| I1 — View/frame consistency | Preserved. Fresh solid/header-only/wrong-target/cell-text/geometry failures still reject evidence. |
| I2 — failed-leader cleanup | Implementation unchanged. Saved fix3 process log records all ten focused tests passing, including real failed-leader descendant cleanup. No fresh process test was needed for this JavaScript boundary change. |
| I3 — immutable resume provenance | Preserved. Renderer implementation unchanged; fresh mixed-resume test refuses the changed environment with zero writes and unchanged old renderer metadata. |
| I4 — source provenance / historical honesty | Preserved. All 12 saved original-manifest/first-row artifact digests match before and after review execution. Five current specimens retain their original clean capture revision `86c3486`; their listed production/fixture/tool input hashes match bytes at that revision. |
| M1 — browser/tool provenance | Preserved. Fresh adapter test still records served bytes separately from tool HEAD and leaves the served revision unknown. |
| M2 — report ownership | Preserved. No tracked `.superpowers` files; designated report path is ignored. Changes remain confined to owned test tooling. |

Adaptive key ordering is unchanged. The saved focused process log includes forward and reverse actual PTY delivery. Fresh standalone verification of all five current specimens still checks protocol plan/delivery/announcement. A3 Help/More integration remains open as recorded in the prior review.

**N1 — Minor, inherited, nonblocking: malformed empty-client observations lack a type guard.** At `tests/tui-visual/evidence.mjs:34`, `fixtureState()` tests `clients.length` for an empty request without validating the observation's shape. A fresh in-memory clone of the Unicode specimen, relabeled to empty in both request fields and given `fixture.clients = {}`, passes with `observed-model-and-production-view`. The same predicate is byte-for-byte unchanged from the parent, so this is outside the absent-key regression fixed here. The real helper legitimately exports `null` for its Go nil slice (`internal/tui/visual_capture_test.go:494`); a follow-up should accept that representation and arrays while rejecting object/scalar shapes. No retained specimen was altered, and this does not reopen R1's historical compatibility failure.

## Evidence and fresh verification

Named logs under `.cache/tui-visual/fix3-logs/` show meaningful RED/GREEN behavior: pre-fix legacy tests have six passes/four intended failures, including rejection of the actual old probe and incorrect acceptance of a current empty row without clients. The standalone RED probe fails on the client predicate. GREEN logs record ten passing legacy tests for each actual historical input, six passing standalone specimens, 21 renderer cases, 17 selected semantic cases, ten process cases and 12 unchanged historical digests. RED intermediate source is not a separately retained commit.

Fresh commands from the worktree:

```sh
A2_TEST_MANIFEST=artifacts/tui-native-fix2/unicode/manifest.json A2_LEGACY_MANIFEST=artifacts/tui-native-probe/manifest.json node --test tests/tui-visual/legacy.test.mjs
A2_TEST_MANIFEST=artifacts/tui-native-fix2/unicode/manifest.json A2_LEGACY_MANIFEST=artifacts/tui-native-baseline/manifest.json node --test tests/tui-visual/legacy.test.mjs
PYTHONDONTWRITEBYTECODE=1 python3 tests/tui-visual/verify.py artifacts/tui-native-probe/manifest.json artifacts/tui-native-fix2/unicode/manifest.json artifacts/tui-native-fix2/help-last/manifest.json artifacts/tui-native-fix2/budget-invalid/manifest.json artifacts/tui-native-fix2/compact-invalid/manifest.json artifacts/tui-native-fix2/endpoint-invalid/manifest.json
A2_TEST_MANIFEST=artifacts/tui-native-fix1/final-dashboard/manifest.json node --test --test-name-pattern 'renderer rejects|renderer checks|real production evidence|mixed resume refuses|reference records|genuine RGB re-encoded as opaque|same-screen|Unicode name metadata|legacy rows without' tests/tui-visual/render.test.mjs
```

Results: both legacy runs pass all ten cases without skips; all six standalone specimens pass; all 18 selected renderer tests pass. Only the first row of the baseline manifest is validated by the legacy test, with the manifest reread to confirm unchanged bytes.

Also ran `semantic.test.mjs` with `A2_SCENE_MANIFESTS` pointing to the five existing fix2 specimens and pattern `unicode positive|help-last positive|budget-invalid|compact-invalid|endpoint-invalid|new schema|post-key|historical semantic`: all 17 selected cases pass. Read-only Python probes confirmed exact diff equality, all 12 historical fingerprints and every listed source input against the original capture commit. Node syntax checks, `git diff --check`, ownership/ignore checks and final worktree status checks pass. The additional in-memory empty-observation probe established N1 without writing artifacts.

## Limits

Approval covers the requested missing-observation compatibility boundary and preservation checks. Historical absent observations remain unavailable; successful verification does not establish their distinguishing semantic outcomes. No full matrix, full fault runner, broad Go suite or native A3 capture was rerun. Process cleanup was assessed through unchanged source and recorded focused evidence. This round did not reopen images; prior pixel inspection and current frame/ANSI/paint checks remain bounded evidence.

No Windows/macOS terminal acceptance, matched browser composition acceptance or product visual PASS is established. Every fresh standalone result reports `visual_acceptance: OPEN`. PNG integrity, opacity and semantic gates alone do not complete product visual acceptance.

=== DONE ===
