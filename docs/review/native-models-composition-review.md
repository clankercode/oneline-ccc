# Independent A9 Models source and geometry review

## Spec verdict: PASS for the bounded source/geometry and input-behavior scope

The grouped Models form, three inline choice panels, Advanced disclosure and client/visit restoration implement the requested native adaptation. No Missing, Extra or Misunderstood requirement requiring a source change was found within this review. Hidden semantic leaves remain fixture-compatible while ordinary Tab follows disclosed controls. The authorized keyboard journey was adapted without a direct-activation bypass or removal of its original safeguards.

**Chooser/expanded-Advanced visual coverage remains OPEN.** This verdict does not close A9's visual gate, P0, the complete interaction campaign or native platform acceptance.

## Quality verdict: APPROVED within this scope

**No blocker, major/Important, Critical or Minor source finding identified.** The checks below support this exact revision, with the evidence limitations recorded separately. No production edits or fixes were made by the reviewer.

## Reviewed identity and authority

- Worktree: `/home/xertrov/src/oneline-ccc/.worktrees/native-models`.
- Exact range: `63006cf824517ac1a7cfd1c0d480c519aeb06ff5..014267d522659c4312428305811ff67b8cd26084`.
- HEAD was verified as `014267d522659c4312428305811ff67b8cd26084`. Four improving commits; eight changed tracked paths, 774 insertions and 51 deletions.
- Read the full supplied review brief, author report, saved package, directly relevant source/tests and author plan. Also read main's P0 requirements, A9 brief and `docs/review/native-tui-composition-vision.md`. The operator's native rejection and mandatory alternating review process remain authoritative.
- `A9-review-package.diff` contains commit/stat preamble followed by exactly `git diff --unified=10 63006cf 014267d`. Saved package SHA-256: `9e912902fde55478744de4cb7aefbdf7d826fcc97b3f0f58e37c2f24b2c18fc5`.
- Changed production scope is Models composition plus necessary input/navigation/state integration. The only changed existing test outside the new Models test file is the explicitly authorized `TestKeyboardOnlyJourneyAndReviewBack`. Backend, Review pagination/acknowledgment, shared shell/components/theme/Dashboard and capture tools are not changed by this package.

## Checked and accepted

### Visible controls and explicit choice intent

`models.go:30-112` retains semantic choices but filters ordinary navigation according to disclosure. The main path is management → model field → budget field → tier field → Advanced → Review, without visiting hidden radio/editor leaves. `modelsKey` at lines 152–197 handles forward/reverse Tab and local arrows against that visible set. Opening a field establishes the chooser and return location; Escape/Back-to-field restores its invoker (`models.go:113-150`).

Each explicit chooser contains actual visible choices and a reachable close control. The model panel includes custom-ID/discovery routes; the budget panel preserves a custom budget and its editor route; the tier panel keeps the current exact model/default and separate availability wording. Enter commits through existing profile-change handling; arrows and Tab do not change drafts. Preset/discovered commit closes to the appropriate field. Custom editor cancel/save returns to the invoking leaf within its containing choice/Advanced context.

Independent tests entered Models and the selectors using actual Tab/Enter messages, cycled visible Tab stops, reversed, crossed regions with F6/Shift+F6, canceled, opened/canceled the custom editor, and committed a model choice. All five sizes and all three themes passed frame/focus and unchanged-navigation-draft checks. Retained semantic controls were not used as a substitute for those keyboard paths.

### Authority, validation and unavailable states

The input delta invokes existing `changeProfile`, preserving capacity/tier invalidation and plan invalidation. A separate input-driven tier probe selected Fast: arrows preserved the draft; Enter changed the tier and cleared both route attestations. With management enabled but the route unconfirmed, Review returned synchronous refusal before any backend Plan/Apply call. This avoids interpreting a selected tier or advertised model as verified capability.

Custom budget tests retain positive-integer validation, exact stored value, compaction calculation and invalidation. Exact custom Unicode model IDs are stored unchanged; discovery labels retain source identity and positive advertised context is explicitly unverified. Empty/error discovery does not invent advertised inventory or entitlement. Existing reorder/disappearance identity regressions passed.

Unavailable controls retain their reason and are blocked by existing activation guards. Independent ordinary traversal/activation of the missing-client form at all five sizes found no draft changes, chooser opening or Plan/Apply calls from unavailable fields. No-client tests show explicit unselected values rather than a fabricated current zero-token/Standard profile. Advanced disclosure remains available for inspection. Missing/read-only compact forms still repeat long reasons and require scrolling; this disclosed presentation limitation is not promoted into a source blocker by these bounded checks. Its readability still needs actual image review.

### Client identity, disclosure and return

`navigation.go:16-84` stores Models disclosure/location per client and includes client/form state in visit-specific Back frames. New-client entry starts at its own form rather than exposing the prior client's Advanced leaf. Back restores the frame's client and disclosure before resolving the saved control identity. Editor/modal handling still precedes Models keys, preserving text input and ordinary close/restore behavior.

The new client/Back/resize regression and existing visit-specific Back tests pass. An independent actual-key probe saved a custom budget, switched section with `]`, returned with Escape, and recovered the same Codex draft and Advanced editor control. Collapse restores the Advanced disclosure control without mutating the draft. Resize retains focus and draft values; selected-client identity remains in the Models title, including the tested long Unicode name.

### Geometry and useful Review access

At 100/120 columns, field and applied/draft summary columns are explicitly measured. At compact sizes the body stacks deliberately, with the editable 40x12 form pairing distinct Advanced and Review controls on one row. The five-size composition regression verifies initial tiny Review access, 80-column applied/draft comparison, selection identity and focused-control bounds.

Independent chooser/Advanced navigation covered 120x40, 100x30, 80x24, 60x20 and 40x12 in dark/light/mono. All observed frames retained one focus marker, valid Unicode and cell-width/height bounds. Long current exact IDs wrap into scrollable content rather than rewriting the stored value. The tiny initial viewport need not show the entire long value or chooser introduction; later document rows remain available through paging. These checks establish bounded geometry and reachability, not readability/contrast or visual polish of the uncaptured panels.

### Authorized full journey remains input-driven

The `journey_test.go` diff changes only `TestKeyboardOnlyJourneyAndReviewBack`. It adds field entry and Advanced disclosure with actual Tab/Enter, retaining management staging, exact model/budget choices, capacity/tier attestations, initial Review Back focus, Back preserving the draft, explicit read acknowledgment, one Apply and successful snapshot reconciliation. The additional combined choice/attestation assertion strengthens that journey. There is no introduced `focusID` or `activate` bypass in it. Fresh execution passed.

## Execution and test evidence

Fresh focused suite passed, package **0.791s**:

```sh
GOMAXPROCS=2 GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
GOCACHE="$PWD/.cache/a9/review-go-build" \
GOTMPDIR="$PWD/.cache/a9/tmp" TMPDIR="$PWD/.cache/a9/tmp" \
go test -p 2 ./internal/tui \
  -run '^(TestModels.*|TestKeyboardOnlyJourneyAndReviewBack|TestFix1.*|TestET1SingleFocusMarker)$' \
  -count=1
```

This covers the eleven new Models tests, authorized keyboard journey, focus-marker regression and existing relevant Back/discovery/region/credential/Help boundary regressions. An initial focused run also passed in 0.169s using the inherited standard Go cache; the quoted repeat and independent probes explicitly used checkout-local build/temp paths.

Independent test sources and overlay JSON were held in Linux memory file descriptors, without writing source/test files. Using the same bounded/offline/local settings and `go test -p 2 -overlay /proc/<live-python-pid>/fd/<overlay-fd> ./internal/tui -run '^TestA9Independent' -v -count=1`:

- `TestA9IndependentPaths`: 15 size/theme cases, actual main-field/selector/Advanced navigation, complete visible Tab cycles and reverse/region checks, cancel/editor return, explicit model commit and invalidation.
- `TestA9IndependentUnavailable`: missing-client visible traversal/blocked activation across all five sizes, with unchanged draft and zero Plan/Apply calls.
- `TestA9IndependentLongSelector`: long CJK/combining exact ID at 40x12, repeated visible Tab traversal and return.
- These three passed together in **0.847s**. The separate corrected tier/refusal/custom-budget/section-Back probe passed in **0.004s**. The initial version of that negative probe incorrectly required an asynchronous command from the deliberately synchronous validation refusal; correcting that probe assertion required no application change.

Author final JSON logs `.cache/a9/tui-normal-final.jsonl` and `tui-race-final.jsonl` each contain **90 top-level / 145 total passing events, zero failures/skips**, matching the report. Initial normal logs preserve genuine failures of the former flat journey, missing installed/managed evidence, tier adjacency, scroll-to-evidence and region restoration. The claimed earlier new-Models RED cycles are described in the plan/report but no separate raw RED log was located in `.cache/a9`; their historical execution is not independently established here. Fresh meaningful behavior assertions pass. No fresh full package/race/repository suite is claimed.

`git diff --check 63006cf 014267d` and read-only `gofmt -l` on the eight changed files passed. Tracked source remained clean; untracked campaign/development capture paths were preserved.

## Capture limits and required handoff

All four final manifest directories identify exact production `014267d`, clean production scope and dirty/untracked development fixture/tool scopes. The 19 generated PNGs comprise 15 parity, 2 empty, 1 Unicode and 1 NoColor baseline frames; their PNG hashes match their manifests. **Every final capture is a baseline variant.** None establishes a model/budget/tier chooser or expanded Advanced image state.

No PNG was opened for a new visual judgment in this source/geometry review. The parent's three final dark full images and implementer's disclosed image/crop inspections do not close the missing state coverage. Continue with three inline chooser panels, Advanced collapsed/expanded, actual custom-editor return, unavailable/discovery/long-value states and visible keyboard journeys through reviewed capture tooling. Preserve production/fixture/tool provenance and prove post-protocol state rather than inferring it from semantic setup IDs.

The actual chooser/Advanced pixel quality, full compact unavailable-state readability, matching-reference comparisons, integrated A3b/Help/A9 behavior, complete alternating same-revision visual/source convergence and final installed build remain unverified. This branch is based on `63006cf`; main's separate Help correction is not assumed present or reapproved. No actual gateway/discovery/auth/inference, child/service work, Windows/macOS/native-platform or all-screen acceptance is claimed.

No agents, attention/notification tools, network, pushes, operator configuration or real credentials were used. This designated report was written last and is the only authored persistent review artifact; Go verification populated ordinary caches/temporary files.

=== DONE ===
