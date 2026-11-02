Spec verdict: CHANGES REQUESTED — I1a/I1b addressed, but the fix regresses required legacy verification.
Quality verdict: CHANGES REQUESTED — one Important compatibility regression; prior process/provenance fixes remain intact.

# A2 fixround2 independent re-review

Worktree: `/home/xertrov/src/oneline-ccc/.worktrees/native-captures`.
Exact range: `04228641919569ef5e23afdde65ee35c1230172e..86c3486a016cb6922523c2d8698b8b879fac7453`.
Locations below are relative to that worktree.

Read the fix2 brief, previous re-review, full 1,277-line saved diff, implementation report, named RED/GREEN evidence and five final manifests. Saved diff exactly equals `git diff --unified=7 0422864 86c3486`; SHA-256 `6219b522245dce297717533435ddf8fce7812f7026735601ce9d8eadbc7f64c0`. HEAD was exact and git status remained clean. Production renderer/CLI/site/dependencies are unchanged; the guarded helper is the only changed production-tree file.

No source edits, git mutations, agents, attention utilities, operator configuration/keys, network, builds, new captures or matrix reruns. This report was written last. Review execution note: an intended Node name filter did not exclude three standalone-verifier cases; they created and removed disposable directories under checkout-local `.cache/tui-visual/tmp`. Existing source/git state and saved evidence were not changed. The report is the only retained new artifact.

## Original finding dispositions

| Finding | Disposition | Evidence |
| --- | --- | --- |
| I1a — opacity | **Addressed** | Shared gate rejects non-255 RGBA alpha and PNG transparency chunks. Fresh actual exporter tests reject genuine RGB re-encoded with alpha 0/128 before PNG/frame publication; opaque RGB/RGBA controls pass. Fresh standalone tests update PNG hash/length and reject on opacity, while alpha 255 passes. |
| I1b — genuine semantic state | **Addressed for the reviewed baseline** | Helper checks actual Unicode fixture content, production Help page marker, submitted invalid value/refusal and unchanged draft. Schema 3 exports before/after observations; independent checks reject relabeled client states, wrong page/refusal/value/draft evidence and observation drift. Focused model, shared-gate and representative checks pass. See the separate legacy regression below. |
| I1 — earlier blank/frame/scene gates | **Retained for current evidence** | Existing solid RGB, header-only, wrong-target, inconsistent-cell/text and wrong-geometry tests pass. Actual View/ANSI/cell/text consistency and pre-publication checks remain. Legacy compatibility is currently incomplete because of R1. |
| I2 — failed-leader cleanup | **Preserved** | Cleanup implementation unchanged. Fresh real fork/leader-exit test passes, together with missing-process/readiness/timeout/cleanup and both wire-order controls. |
| I3 — immutable resume provenance | **Preserved** | `render.mjs` unchanged. Fresh mixed-resume test still refuses a changed environment without writes/relabeling existing renderer metadata. New evidence-code hashes match the clean representative manifests. |
| I4 — source/fixture provenance | **Preserved** | Source enumeration/hashing unchanged. All five manifests name exact clean HEAD and their listed source/helper/binary/harness/renderer fingerprints match current bytes. Historical annotations remain additive; the original probe's unannotated digest still matches its stored pre-annotation digest. |
| M1 — browser/tool provenance | **Preserved** | Browser renderer unchanged; fresh reference test still hashes served bytes separately from tool HEAD and records unknown served checkout revision. No live reference server or network was used. |
| M2 — report ownership | **Preserved** | No `.superpowers` files tracked; the operational report remains ignored. The commit contains only the owned helper and `tests/tui-visual/**`. |

**Adaptive key-order regression:** addressed at the reviewed baseline's model/driver boundaries. `visualProtocol` tries both orders on private production-model copies, refuses command-producing/non-restoring pairs, and checks production View plus draft restoration. The chosen plan is emitted before initial readiness; the Python driver sends the announced real bytes; the helper checks decoded order and actual initial/final View equality; standalone verification checks plan/delivery/announcement. Fresh actual PTY peer tests prove forward and reverse byte delivery, and fresh production-model first/last palette tests prove boundary restoration. Integrated A3 Help/More behavior remains explicitly open; this is not a claim of an A3 native reverse-order capture.

## R1 — Important: new semantic check rejects historical captures before the legacy fallback

**Locations:** `tests/tui-visual/evidence.mjs:34`, `tests/tui-visual/evidence.mjs:56`, `tests/tui-visual/evidence.mjs:58`; documented contract at `tests/tui-visual/README.md:181`.

**Scenario:** run the current standalone verifier on the unchanged original probe:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 tests/tui-visual/verify.py artifacts/tui-native-probe/manifest.json
```

**Observed:** exit 1, `verify: Fixture client state baseline not reached`. The original probe predates serialized `fixture.clients`. Its existing byte/hash/PTY/geometry checks complete before the new shared semantic check rejects it. Independently calling actual `validateEvidence()` on the first original baseline row produces the same failure. A schema-2 final Dashboard that does contain clients passes as a legacy positive control.

`semanticOutcomes()` calls `fixtureState()` before checking the schema. That check turns an absent historical client field into `[]` and rejects it before returning `legacy; distinguishing outcomes not recorded`. Reading metadata, without rerunning the matrix, shows **all 360 original baseline rows plus the one initial probe lack `fixture.clients`**: 361 of the original 380 outputs have this incompatibility.

**Effect:** previously verifiable historical evidence can no longer use its documented weaker ANSI/cell/text/scene/geometry/privacy/paint path. Missing historical observations are treated as a failed client state rather than explicitly unavailable proof. The three historical samples chosen by the implementer all contain client metadata, so they do not exercise this boundary. The new historical-schema test downgrades a new, fully populated fixture and likewise misses the real old format. This is a verification compatibility regression, not evidence that the old images are corrupt or a request to invent historical semantics.

**Fix:** make legacy handling depend on what was actually recorded. For schema 1/2, retain the existing integrity/content checks and explicitly report unavailable distinguishing-state proof; validate optional historical client observations only when present. Keep schema 3 strict about required observations and state predicates. Add an unchanged real legacy row without `clients` as a positive regression control, and a current-schema missing-client/observation case that must fail. Do not modify old manifests, infer missing clients from a newer fixture or recapture/relabel the original matrix.

## Checked and fine

- **Opacity fixes exercise genuine pixel data.** Valid PNGs are produced by decoding real screenshot RGB and re-encoding with alpha 0, 128 or 255 and valid chunk CRCs. The transparent cases now fail for alpha/opacity, not hash mismatch or loss of glyph contrast. Both exporter and standalone verifier were freshly exercised. Rejected exporter evidence is not published as PNG/frame.
- **Semantic checks use production outcomes.** `visualScene` compares the actual Unicode name; Help page numbers come from production `modalRows`, not a duplicate pagination formula. For invalid edits, the real Enter attempt records refusal, retained editor and full comparable desired-state hashes. A private model/editor/draft copy calls the production `saveEditor` validator, and a focused test proves that probe leaves the actual draft unchanged. The shared verifier checks observations against actual frame text, including page markers, invalid input and refusal; before/after drift is refused.
- **Positive controls remain real.** All five clean representatives passed fresh standalone verification as schema 3, `production-view-and-ansi; semantics=observed-model-and-production-view`, with visual acceptance OPEN. Their source/helper/binary/renderer-code hashes and all listed production/fixture/tool input hashes match current bytes; dirtiness flags are false and untracked input lists empty.
- **Representative outcomes are inspectable.** The final Unicode sample contains the complete CJK/accented fixture name. Help records page 6 of 6 before/after. Budget, compaction and endpoint retain `invalid-input`, actual production refusal status and identical before/after full-draft hashes. Announced plan and decoded keys match in all five, with the ordinary order `tab,shift+tab`.
- **Actual pixels inspected.** Opened the final Unicode/ANSI-256 Dashboard and invalid endpoint PNGs. CJK/accented text, native styling and the endpoint refusal/retained invalid input are visible. No dummy route is displayed. This is bounded capture-content inspection, not application visual acceptance.
- **Protocol tests are meaningful.** Saved RED evidence shows the real production palette last-item fixed-order model test failed while first-item passed, and the reverse-order live PTY peer failed under the old fixed driver. Fresh tests now pass both directions. The announced plan precedes the completed-write epoch-0 marker; both keys still travel through the actual PTY instead of replacing capture input with synthetic model events.
- **Prior privacy/provenance boundaries survive.** No real backend/CLI/configuration is introduced. Dummy endpoint-sensitive draft fields are hashed without exporting their values. The initial probe's original source record remains intact, with its honest omission annotation; removing that additive annotation reproduces the stored original-manifest digest. No old artifact was upgraded to schema 3 or given a new production View.
- **More remains a handoff.** Current baseline lacks reviewed A3 navigation; unsupported `last-item` semantic verification fails closed. README explicitly requires implementing the integrated More outcome guard before claiming first/last-item captures. No More or all-screen P0 completion is claimed.

## RED/GREEN evidence

Read named logs under `.cache/tui-visual/fix2-logs/`:

| Log | Verified recorded behavior |
| --- | --- |
| `red-node.log` | 16 tests: nine pass, seven assertion failures. Old gate accepts transparent/partial-alpha real-RGB PNGs and five same-screen state relabelings; opaque controls pass. |
| `red-go.log` | Genuine positives pass; five wrong-outcome and three unsubmitted-invalid cases fail the intended rejection assertion. Existing wrong target/editor/client checks pass. |
| `red-semantic-suite.log` | 14 Python pass; 34 Node cases with 17 passes/17 assertion failures when independent semantic gates are absent. This demonstrates actual same-scene faults, not setup/import failure. |
| `red-boundary-go.log` | First-item passes; actual production palette last-item View restoration fails with fixed order. |
| `red-boundary-python.log` | Original eight controls pass; live reverse-plan peer case fails because the old driver sends the wrong initial key and reaches its deadline. |
| `green-fault-suite.log` | Eleven actual disposable fixture captures; 15 Python and 47 Node tests pass with no skips/failures, including opaque controls, matching-hash verifier cases, semantic outcomes and prior regressions. |
| `green-go-scene.log`, `green-tui.log` | Focused model/protocol checks and surrounding TUI suite pass in recorded implementer runs. |
| `historical-sample-verification.log` | Three selected populated legacy fixtures pass with weaker semantic labels. Does not cover the missing-client historical format in R1. |
| `final-representatives.log`, `final-provenance-audit.log` | Five exact clean-commit representatives verify; saved audit reports 157 distinct source inputs with matching hashes. Fresh review checks confirmed their inputs and outcomes. |

Intermediate RED working trees are not separate retained commits; logs remain local execution evidence. Fresh review execution below establishes current behavior independently. No complete fault-runner capture campaign or broad Go suite was repeated.

## Fresh commands and results

From the reviewed worktree:

```sh
git rev-parse HEAD
git status --porcelain
git diff --stat 0422864 86c3486
git diff --check 0422864 86c3486
git diff --name-only 0422864 86c3486 -- internal/tui cmd site go.mod go.sum
git ls-files -- .superpowers
git check-ignore -v .superpowers/sdd/product-completion/A2-fix2-report.md
node --check tests/tui-visual/evidence.mjs
node --check tests/tui-visual/render.test.mjs
node --check tests/tui-visual/semantic.test.mjs
GOMAXPROCS=2 .cache/tui-visual/native-capture.test -test.run '^TestVisual(Scene|Protocol)' -test.v
PYTHONDONTWRITEBYTECODE=1 python3 tests/tui-visual/verify.py artifacts/tui-native-fix2/unicode/manifest.json artifacts/tui-native-fix2/help-last/manifest.json artifacts/tui-native-fix2/budget-invalid/manifest.json artifacts/tui-native-fix2/compact-invalid/manifest.json artifacts/tui-native-fix2/endpoint-invalid/manifest.json
PYTHONDONTWRITEBYTECODE=1 python3 tests/tui-visual/verify.py artifacts/tui-native-probe/manifest.json
A2_TEST_MANIFEST=artifacts/tui-native-fix1/final-dashboard/manifest.json node --test --test-name-pattern '^(?!standalone verifier)' tests/tui-visual/render.test.mjs
```

Results: syntax/diff/ownership checks pass, hash-matched existing binary's focused model/protocol cases pass, all five new representatives pass, and unchanged old probe fails as R1 describes. The Node expression above matched the full reported test names and executed **all 20** cases, including three standalone verifier cases with disposable local-cache directories; all 20 passed. Those directories were removed by the tests. This incidental temporary output is explicitly disclosed rather than described as a memory-only run.

From `tests/tui-visual/`:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest test_capture test_evidence.OwnershipFailures.test_leader_exit_does_not_leave_live_descendant -v
```

All ten focused PTY/process tests pass, including actual forward/reverse delivery and failed-leader descendant cleanup. This excludes the temporary git-repository provenance test.

A further `node --test` invocation used `tests/tui-visual/semantic.test.mjs`, an `A2_SCENE_MANIFESTS` JSON map to the five final directories (`unicode`, `help-last`, `budget-invalid`, `compact-invalid`, `endpoint-invalid`), and name pattern:

```text
unicode positive|help-last positive|budget-invalid|compact-invalid|endpoint-invalid|new schema|post-key|historical semantic
```

All 17 selected semantic tests pass: five genuine positives, each invalid editor's refusal/value/draft faults, required observations, post-key drift and populated historical-schema control. Cases needing the runner's deleted additional fixtures were not rerun.

Additional read-only heredoc probes:

- `PYTHONDONTWRITEBYTECODE=1 python3 -`: exact diff/hash comparison, current source/helper/binary/fingerprint audit, before/after outcome equality, old manifest metadata enumeration and probe annotation reconstruction. Identified exactly 360 baseline rows and one probe lacking client metadata.
- `node --input-type=module -`: imported actual `validateEvidence`, read the first original baseline row and initial probe unchanged, and observed `Fixture client state baseline not reached` for each. A populated schema-2 Dashboard passed with the weaker semantic label. No files written by this probe.

## Limits / cannot verify

- No full matrix or all-380 verifier rerun. R1 was reproduced on the unchanged single probe and first baseline row; its affected count comes from metadata inspection and the unconditional predicate, not a new capture campaign.
- No native A3 Help/More reverse-order capture was executed. Current evidence separately exercises the real production model at the clamped palette boundary and actual PTY reverse byte delivery. All five retained native representatives chose forward order. The helper's actual decoded-order/View checks remain active, but reviewed A3 integration still needs its targeted final-page/first-last-menu captures.
- The temporary-repository provenance regression and broad tagged TUI suite were source/log reviewed, not freshly rerun. The existing binary's hashes match the new representative provenance; no rebuild occurred.
- Historical missing client/semantic observations cannot be reconstructed honestly. Preserve weaker proof instead of changing old manifests or substituting newer fixture data.
- No Windows/macOS terminal, live backend, full editor/async/child lifecycle journey, matched browser composition comparison or application visual PASS. PNG integrity, opacity and semantic checks do not establish P0 product acceptance.

The remaining fix is bounded: restore the actual legacy-format path with honest unavailable-proof labeling and a regression fixture, while retaining strict schema-3 state/opacity checks. No matrix recapture is needed.

=== DONE ===
