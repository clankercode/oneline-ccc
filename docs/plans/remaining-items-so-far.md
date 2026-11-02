# Remaining items so far

Recorded: **2026-11-03** (Australia/Sydney).
Implementation baseline: **`40d02d4`**.
Visual/interaction feedback amendment: **2026-11-03**, after backlog commit
**`f051510`**.

The Windows MVP works, and the first native Codex workflow has functional
verification. **Native TUI visual and interaction acceptance is reopened.** The
operator rejected the native interface as visually unlike the approved preview,
"ugly, unimpressive, and cumbersome to use," with inconsistent controls and an
awkward transition from tabs to menus by pressing Down. That criticism was
missing from the original remaining-items list; it is now the explicit P0 below.

The baseline records 18 remaining areas: the P0 remediation and the original 17 areas
identified against the full project brief. All remain open at this baseline.
Update their status as implementation and verification finish, with links to
the resulting evidence. Earlier functional checks and browser-preview approval
do not establish that the native interface meets the visual or usability bar.
Original item numbers 1–17 are retained; item 18 is the highest-priority P0,
with its detailed requirements placed before them.

Execution started **2026-11-03** under the active full-backlog goal. The
[product completion campaign](product-completion-campaign.md) records ordered
milestones and task boundaries. Reviewed navigation, Linux terminal-input and
no-color and bounded startup/error corrections are integrated through `418283a`;
current work covers item 18's composition, item 20's constructor/platform gaps
and item 21's capture-evidence fixes. See the
[native remediation checkpoint](../verification/native-remediation-progress.md)
for the exact integrated gates and their limits.
All checkboxes remain open until the complete corresponding acceptance evidence
exists.
Discovered work is appended below with stable IDs; it is part of the same goal.
There are now **21 tracked areas**. The P0 remains open after the first source
corrections: improved navigation and added color do not establish preview fidelity,
comfortable compact layouts or complete interaction acceptance.
The item 19 implementation is integrated at `cbbaf77` with Linux PTY follow-ups
at `c1440cf`. Actual no-color entry, editor-focus/masking, draft/quit and redirect
checks pass; remaining platform and full-screen acceptance is still open.

Some areas extend working features; others require new implementation. Automatic
maintenance and automatic repair retain the policy decisions and explicit opt-in
described below. Use the current native usage, verification and delivery ledger
for shipped capabilities; older plans retain their original planning baselines.

## Foundational and supporting documents

| Document | Purpose |
| --- | --- |
| [TUI production plan](tui-production-plan.md) | Full product scope, architecture, requirement map, phased tasks T0–T10 and exit tests |
| [Project brief](../product-brief.md) | Original MVP and broader product intent |
| [Broader utility brief](broader-utility-brief.md) | Retained multi-client, maintenance, context, repair and SSH requirements |
| [Native foundation plan](native-foundation.md) | Scope of the first implemented native slice |
| [Native usage](../native-app.md) | Current commands, workflows and implementation limits |
| [Native verification](../verification/native-foundation.md) | Actual configuration/terminal evidence and remaining production gates |
| [Native delivery ledger](../ledgers/native-foundation-ledger.md) | Delivered work, decisions, reviews and final gates |
| [Native remediation checkpoint](../verification/native-remediation-progress.md) | Integrated navigation, queued-input and no-color fixes with independent reviews; visual and platform acceptance still open |
| [Native acceptance review](../ledgers/native-foundation-acceptance-rereview.md) | Final process/terminal corrections and the disclosed dependency race |
| [Client compatibility contract](../compatibility/clients.md) | Client identities, supported Codex contract and adapter expansion gates |
| [Platform and distribution contract](../compatibility/platforms.md) | OS/architecture support, native proof and release/bootstrap trust |
| [TUI usability contract](../design/tui-usability.md) | Selection, management, drafts, review, tiers, help and command search |
| [TUI screen flows](../design/tui-screen-flows.md) | First-run, rerun, model, service, jobs, repair and SSH journeys |
| [TUI visual system](../design/tui-visual-system.md) | Approved color, hierarchy, focus and non-color cues |
| [Interactive mockup](https://clankercode.github.io/oneline-ccc/tui-preview.html) and [versioned preview source](../../site/tui-preview.html) | Approved visual reference to compare against actual native renders; record the source revision because the live page can change |
| [Preview styles](../../site/tui-preview.css), [views](../../site/tui-views.mjs) and [dialogs](../../site/tui-dialogs.mjs) | Mockup layout, emphasis, screen composition and overlay references for the native fidelity review |
| [Secondary-screen references](https://clankercode.github.io/oneline-ccc/tui-native-reference.html), [versioned source](../../site/tui-native-reference.html) and [verification boundary](../verification/tui-native-references.md) | Design targets for six secondary screens, eight layers and seven editors; browser targets remain separate from native acceptance |
| [Preview verification](../verification/tui-usability.md) | Historical browser-only checks; does not close native visual or interaction acceptance |
| [Native screen and navigation inventory](../review/native-tui-screen-inventory.md) | Source-backed inventory of 10 screens, 7 modal variants, 7 editors, current control inconsistencies and missing mockup counterparts; no visual PASS claimed |
| [Native capture infrastructure and limits](../verification/tui-native-captures.md) | Real PTY/ANSI/PNG tooling, controller fault gates, independent semantic/legacy reviews and outstanding final-capture requirements |
| [Stack and identity research](../research/tui-stack.md) | Framework choices, model metadata and client/utility identity evidence |
| [Existing-project research](../research/existing-projects.md) | Prior art and reuse options to consult before duplicating functionality |
| [Stage-zero verification](../verification/tui-stage-zero.md) | Native Codex contract and isolated existing-tool adoption trials |
| [Codex configuration research](../research/codex-configuration.md) | Configuration, model catalog and service-tier findings |
| [Windows MVP plan](windows-codex-mvp.md) and [verification](../verification/windows-mvp.md) | Existing one-line Windows setup and its tested behavior |
| [Windows SSH documentation](../windows-ssh.md) | Existing SSH helper, authentication, firewall and recovery behavior |
| [Windows Explorer defaults plan](windows-explorer-defaults.md) | File-extension and hidden-file preferences already implemented in the installer |
| [Script rerun/defaults plan](script-rerun-defaults.md) | Existing-setting prefill and idempotent installer behavior |
| [Configuration transaction contract](../../internal/configtxn/README.md) | Protected writes, drift, backup and recovery rules |
| [Observer contract](../../internal/monitor/README.md) | Current read-only observation and native lifecycle boundaries |
| [Test documentation](../../tests/README.md) | Existing verification harnesses |
| [Plan index](README.md) | Entry point for the repository's plans and status |
| [Product completion campaign](product-completion-campaign.md) | Full-backlog execution order, first milestone boundaries and dynamic-expansion rules |

## P0: Native TUI visual quality and interaction remediation

**Status: OPEN — native visual and interaction acceptance rejected.**


### Reported defects and required standard

The user explicitly rejected the native TUI's appearance and usability. Their
feedback is that it **does not look like the preview**, is **ugly, unimpressive
and cumbersome**, has **inconsistent controls**, and makes moving from tabs into
menus by pressing Down an unsatisfactory experience. Record these as major
product defects, not deferred cosmetic polish. The previously praised preview
does not constitute acceptance of the implemented native interface.

The native TUI must deliver the approved design's visual hierarchy, color,
texture, bold emphasis, spacing, alignment, list/detail structure, clear actions
and distinct navigation/selection/focus states. Rich terminal rendering is the
primary experience. A plain ASCII approximation or a wall of minimally styled
text does not meet this requirement. Keep intentional monochrome/ASCII fallbacks
usable, with clear hierarchy and state cues. Document genuine terminal constraints
and design a coherent native adaptation for them; do not weaken the visual target
or simplify the approved mockups merely to make the comparison pass.

### Required work

- [ ] **Establish an honest before/target comparison.** Pin the rejected native
  revision and the approved preview/mockup revision. Capture actual native output
  and matching mockups with the same fixture scenario, selected client, model,
  budget, request tier, available tiers and operation state. Distinguish visual
  mismatches, interaction defects and unavailable product capabilities. No
  screenshot or behavior comparison has been performed as part of this backlog
  amendment; the user's rejection is the evidence reopening acceptance.
- [ ] **Replace the flat control traversal with a coherent interaction model.**
  Treat section navigation, client/model lists, forms, action groups and overlays
  as deliberate focus regions. Provide direct, discoverable ways to switch
  sections and enter/leave content without repeatedly pressing Down through a
  single global sequence or walking every unrelated control. Horizontal tab
  navigation must support horizontal movement; Up/Down must behave locally in
  lists/menus. Tab/Shift+Tab must provide predictable forward/reverse access to
  controls and focus regions. Specify the exact bindings and transitions before
  implementation, then apply them consistently to every screen.
- [ ] **Make control behavior consistent.** Use Enter for deliberate activation,
  Space for checkbox/toggle choices, and Escape for the top overlay or the
  expected parent/back action while retaining drafts. Editing/searching must
  preserve ordinary text input. Keep current section, selected data row and
  keyboard focus visually distinct. Restore focus when a dialog closes; keep
  focus/selection stable through refresh, resize and child return. Navigation
  must not enable management, change a value or apply a plan accidentally.
- [ ] **Make every destination discoverable.** Give Dashboard, client detail,
  Models, Gateway, Review, Jobs, Service, Repair, SSH, Utilities and Settings a
  coherent place in navigation. Help and the command palette are accelerators;
  essential destinations and actions must also have an understandable visible
  route. Context-specific footer hints, help and command search must agree with
  actual bindings and the focused control. Do not claim a functioning help panel
  alone proves an intuitive interface.
- [ ] **Keep the main path effortless.** Group advanced settings behind
  deliberate disclosure; prioritize select client → enable management →
  configure sensible defaults → review → apply → launch. Explain unavailable
  actions where they occur and show the next useful step. Review keyboard effort
  and unnecessary backtracking, not just eventual reachability. Confirm Ctrl+P
  delivery and a visible command-search fallback in supported terminals.
- [ ] **Rebuild the presentation as a coherent component system.** Use consistent
  navigation, client rows, detail panels, field groups, choices, buttons, status
  badges, notices, diff rows, progress, dialogs and footer hints. Apply common
  color/weight/spacing rules across repeated components and themes. Preserve the
  approved model/default-tier adjacency and separate available-tier wording.
  Present the next useful action clearly, without debug text or implementation
  commentary in the main flow.
- [ ] **Review every screen, overlay and relevant state.** Complete the coverage
  matrix below, including unsupported and unavailable states. Existing browser
  mockups cover only part of the native app: create comparable mockups for missing
  surfaces using the approved visual system and screen-flow documents. A missing
  mockup is open work, not permission to waive that screen's visual review.
  Every coverage-manifest row must link native and reference captures, revision,
  fixture/state/size/theme, findings, fixes and each reviewer's final disposition.
- [ ] **Prove the ordinary journeys feel smooth.** Exercise fresh setup, rerun,
  select versus manage, configure, advanced settings, review/back/apply/receipt,
  launch/return, service management, help, command search, cancellation and safe
  exit. Users must be able to switch sections directly, reach content/actions
  without long traversal chains, see where focus is, and return without losing
  their selection or drafts. Test through real terminal input; source inspection
  and string assertions alone are insufficient.

### Screen and state coverage

The source inventory identifies **10 named native screens, 7 modal variants and
7 editor variants**. Client detail, first-run/rerun, receipts and operation
outcomes are states within these surfaces, not additional implemented screens.
These counts are a source-inspection baseline, not rendered visual acceptance.

| Surface | Required comparison and interaction coverage |
| --- | --- |
| First-run and rerun states | Initial inspection/loading, empty installation, existing choices, drift, monitor offer/decline and the next useful action |
| Dashboard and client detail | Wide list/detail composition, compact layout, selected versus managed versus focused, filtering/no results, ready/missing/unsupported/drift states, configure and launch routes |
| Models and client configuration | Model/default tier, available tiers, budget presets, custom values, capacity/tier evidence, discovery loading/error/empty results and disabled choices |
| Gateway and credentials | Endpoint editing, credential status, masked entry, session/persistent choices, validation/locked/unavailable states and cancel without exposing values |
| Review and result | Legible before/after changes, paging/acknowledgment, Back, no-op, stale/invalid review, Apply availability, in-progress/failure/conflict/success receipt |
| Jobs and diagnostics | Idle/history, active progress, cancel request versus final outcome, long details, interrupted/unknown/failure states and support/diagnostic presentation |
| Service / background monitor | Not installed/running/stopped/unknown/conflict/unavailable, install/uninstall review, action progress and useful recovery guidance |
| Repair | Brief preparation, preview/export, empty/error states and future agent/patch review surfaces as they are implemented |
| Windows SSH | Handoff/review, elevation requirements, cancellation/failure/success, and separate SSH/service status without implying enabled access prematurely |
| Utilities | Present truthful availability and platform reasons coherently; apply the same quality gate to install/update/remove flows when implemented |
| Settings | Theme, color/ASCII/motion choices, persistence feedback, grouped controls and access to related sections |
| Help and command palette | Visual grouping, context-accurate bindings, search/zero results/disabled explanations, scrolling, focus containment and return to invoking control |
| Editors and confirmation overlays | Model/budget/compaction/endpoint/credential/filter editors; API-key entry; discard, quit, service and SSH confirmations; validation, typing/paste, safe defaults, clipping and focus return |

The seven modal variants are **help, palette, discard, quit, SSH handoff, monitor
install and monitor uninstall**. The seven editor variants are **filter, model,
budget, compaction, endpoint, credential reference and masked API key**; editing
a credential reference is distinct from entering a secret. Include first/last
pages, zero search results, unavailable commands, validation errors, save/cancel
and focus return for the applicable variants. Review the shared application
shell, key footer and asynchronous status as well as the page bodies.
The navigation remediation adds a visible **More** menu for secondary screens;
include it as an eighth modal surface in the updated capture/reference matrix,
including first/last/reverse selection and Escape focus return. The original
seven-variant count describes the baseline, not a limit on review coverage.

The approved original browser has four screen counterparts: **Dashboard, Models,
Review and Jobs**. The added [reference companion](../verification/tui-native-references.md)
supplies **Gateway, Service, Repair, SSH, Utilities and Settings**, eight layers
and seven editor targets. Those targets still require comparisons with the exact
integrated native app; creating a reference does not close a native coverage row.
Extend partial references for native discovery/attestations, review pagination
and acknowledgment, real jobs/cancellation/recovery and terminal handoff/return.
Any remaining mockup gap stays open; do not simplify the approved references to
match a weaker native implementation.

For every base screen, capture **120x40, 100x30, 80x24, 60x20 and 40x12** terminal
cell sizes in dark, light and monochrome modes. Include overlay/state variants in
the coverage manifest; mark genuinely inapplicable combinations with a reason.
Check truecolor, 256-color, ANSI-16, `NO_COLOR`, ASCII fallback and long/Unicode
content with readable contrast and non-color state cues. Small terminals may
use deliberate scrolling/compact composition while keeping the current scope,
focused item and available actions understandable. Missing, clipped or unreadable
controls are failures. A Dashboard-only sample cannot certify the entire TUI.
Obtain real Windows and macOS terminal evidence alongside Linux; compiled
binaries and Linux captures do not certify those terminals' appearance or keys.

### Mandatory visual-review-and-fix and ultra-code-workflows execution

The remediation must use **`visual-review-and-fix`** and
**`ultra-code-workflows`**, as explicitly required by the user. The following
requirements carry those instructions into the backlog so they survive handoff:

1. **Plan and delegate deliberately.** Write the spec, component/navigation
   contracts, scoped implementation tasks and review-focus cases. Keep a durable
   ledger, per-agent briefs/reports and explicit screen ownership. Use strong
   subagents for visual judgment, navigation review and bounded implementation;
   keep implementers and reviewers independent. Use isolated worktrees and
   disjoint write sets for parallel implementation. Repeated screen-review
   fan-outs require a resumable workflow script with recorded failures and
   coverage; no screen disappears because an agent timed out or a cap was hit.
2. **Render the real app deterministically.** Build the native binary, drive
   fixture journeys through a PTY and capture its actual styled terminal output
   as PNGs using a real terminal renderer or a documented faithful ANSI/cell
   renderer. Record revision, fixture, terminal/font, cell and pixel dimensions,
   theme and state. Render matching browser mockups after fonts are ready. Do not
   hand-recreate the native UI in HTML and present that as evidence of the app.
3. **Alternate the two review modalities.** A vision-capable reviewer must open
   the native and mockup PNGs, compare them side by side, and crop/zoom suspicious
   regions. An independent geometry/source reviewer must examine layout bounds,
   alignment, clipping, spacing, repeated-component consistency and focus-region
   structure. Interleave vision → source/geometry → vision → source/geometry,
   with at least two passes of each during a real polish campaign. Every screen
   must receive both forms of review; an aggregate montage is a locator, not a
   substitute for inspecting readable individual screens and necessary crops.
4. **Fix, render and inspect again.** Record concrete findings with screen/state,
   revision, evidence, source location and expected improvement. Commit each
   coherent improving iteration before handing mutable work to another agent;
   keep repeated components consistent in each commit. Re-render changed screens
   and shared-component dependants, re-inspect the same suspect regions and
   update comparison artifacts. Runtime behavior changes need reproducing tests
   and independent review alongside visual work. No screenshot refresh or code
   diff alone establishes that the defect is fixed.
5. **Run dedicated interaction reviews.** Independently drive the tab/region/list/
   form/dialog routes and representative full journeys. Specifically reproduce
   and remove the reported cumbersome tab-to-content traversal problem. Check both directions,
   direct section switching from content, context-specific keys, typing guards,
   focus restoration and draft preservation. Automated key-path tests and human-
   readable terminal recordings must support the review.
6. **Demand convergence on the same revision.** Do not stop at the first adequate
   render, call an average ASCII imitation production-ready, or grade visual
   defects as minor merely because backend tests pass. Both visual and geometry/
   source reviewers must independently PASS the same final revision, with clean
   crops and complete screen/state coverage. Interaction review must also PASS.
   No unresolved major preview mismatch or cumbersome/inconsistent control path
   may be buried in a blanket approval; retain any genuine remaining limit
   explicitly and keep affected acceptance open.
7. **Deliver the actual improved app.** Preserve existing configuration, secret,
   transaction and terminal/process guarantees; run applicable integrated/native
   gates. Install the reviewed build from the integrated checkout and verify its
   revision. Provide before/after and mockup comparisons, the complete coverage
   manifest, reviewer dispositions and exact try-it command. The installed TUI
   must match the reviewed captures. Keep this remediation open until that
   evidence exists; functional CI and the old preview approval cannot close it.

### Execution findings to retain in acceptance

Recorded 2026-11-03 during independent reviews and actual baseline-image
inspection. These are concrete obligations within P0, not visual acceptance.
The first four findings below have independently reviewed source fixes in
`403ea35`, integrated as `da17aa0`; the focused TUI race gate passes. The merged
suite exposed two CLI terminal fixtures using the old control order; their
updated region-aware journeys now pass the focused and full merged Go gates.
Final rendered journey and all-screen acceptance remain open. Keep implementation
progress separate from visual acceptance so this list does not imply either
that the fixes are absent or that the rejected experience is now approved.

- [ ] **Repeated visits need visit-specific Back frames.** A Models → Review →
  Models → Back → Back journey must recover the earlier visit's control and
  scroll position; a mutable last-location-per-screen cache loses that history.
  Add the repeated-visit regression alongside the simpler single-Back path.
- [ ] **Discovered-model focus must follow identity.** Reordering or replacing
  discovery results while focused or away must retain the same model identity,
  and Enter must stage that exact model. Positional `discovered:0` identities
  are insufficient; disappearance needs an explicit safe fallback.
- [ ] **Help boundary navigation must reverse immediately.** Repeated forward
  paging at the final page must not accumulate invisible page indices. Verify
  first/last/reverse movement and resize with short terminal heights.
- [ ] **Credential hints must match the focused control.** Space is text while
  entering a key and toggles only the persistence checkbox. Input, persistence,
  Save and Back each need accurate hints, with the key masked throughout.
- [ ] **Replace the observed text-dump composition.** Actual 120×40 baseline
  Dashboard and Models captures show weak active-tab/action hierarchy, sparse
  single-line client rows, a long pipe separator and ungrouped model/Advanced
  controls extending below the viewport. Compare the corrected grouping,
  list/detail balance, action emphasis and compact behavior with the pinned
  approved references rather than treating existing ANSI color as sufficient.
- [ ] **Make compact Dashboard actions usable on the initial view.** Provisional
  native captures at 80×24 show only about three and a half client rows, with
  selected-client details and Configure/Review/Launch buried in a 56-row stacked
  body; at 40×12 the body is about 59 rows. Use a deliberate compact client
  window, concise selected-client summary and visible useful actions. Keep every
  control reachable, selection and drafts stable, and the focused action in view.
  Eventual access after repeated scrolling or F6 traversal is insufficient.
  Retain draft cues when focus and selection differ: at 40×12, Space → Down →
  Shift+Tab can focus Codex with a staged management change while Claude remains
  selected. Show Codex's pending draft and applied/staged distinction, and keep
  actions clearly targeted at Claude. A matching active filter also needs a
  bounded visible filter/edit/clear cue; `Selected 1/1` alone hides its scope.
- [ ] **Strengthen section, action and tier hierarchy against the approved
  reference.** Keep active section, keyboard focus and selected client visually
  distinct. Give the next useful action a coherent notice and primary/secondary
  action group appropriate to setup, configured/no-draft, draft and read-only
  states. Display the selected model with its default tier as **model · Standard**
  (or Fast/Ultrafast), with intentional monochrome/ASCII alternatives; retain the
  separate **up to Ultrafast** capability description and exact stored model ID.
  Footer labels must distinguish the current focus region from what pressing
  F6 will do. Verify each improvement in actual native captures, not source alone.
- [ ] **Paginate Help by readable wrapped rows.** At 40×12 the pinned heading can
  survive while the first binding is clipped into a fragment such as
  `form / actions`. Count actual wrapped content within the available viewport,
  retain complete binding descriptions and support first/last/reverse/resize
  journeys. Changing pages must not silently omit text. A capture named
  `help-last-page` that visibly says **Page 6 of 7** is failed evidence; prove the
  final rendered page and its content after real input, including boundary keys.
  Show useful, clearly grouped bindings immediately: the current first narrow
  page lists Dashboard/Models/Review/Jobs names without shortcut information.
  Distinguish Help's key explanations from command search's destination catalog,
  and give keys, descriptions and text-entry rules readable visual hierarchy.
- [ ] **Carry the redesign through every page body and layer.** Shared-shell and
  Dashboard improvements alone leave Models as a flat radio/control dump with
  exposed Advanced rows and leave secondary bodies, editors and overlays below
  the approved standard. Use deliberate grouping/disclosure, readable fields,
  stable focus, clear actions and matching references for every inventory row.
  Preserve Review's full before/after values, all pages, acknowledgment and
  stale/no-op/resize safeguards; styling must never clip authority-bearing text.

## Other remaining items

1. [ ] **Fix the known terminal shutdown race.** Resolve the Bubble Tea/cancelreader
   race and run the rendered terminal shutdown regression under the race detector.
   Independent review also reproduced an input goroutine surviving the 500 ms
   graceful-shutdown fallback when events are queued, on both cancellation and
   normal quit. Prove the actual input/decoder lifetime is joined before return;
   increasing a timeout or passing quiet-input repetitions does not close this.

2. [ ] **Finish native platform acceptance.** Test interactive input, resizing,
   interruption, client launch/return and terminal restoration on real Windows
   and macOS terminals. Establish minimum OS versions and runtime coverage for
   the advertised architectures.
   Diagnose the macOS race-gate failure in
   [run 41526198312](https://github.com/clankercode/oneline-ccc/actions/runs/41526198312):
   `TestRPCOwnsHelpersBeforeReturningAndStageRemoval/success` returns
   `cannot clean up owned Codex configuration processes` after the helper reports
   readiness. Preserve the requirement that owned helpers stop before returning
   or removing staging files; determine whether the failure is in ownership,
   process-state observation or the fixture. A prior green run or a rerun alone
   does not explain or close this failure.
   Keep diagnostic fixture cleanup within its ownership lifetime: complete all
   required group signals before releasing the final process identity anchor,
   and retire redundant deferred signaling after explicit cleanup. A recycled
   numeric PGID must never be treated as continued ownership of the old group.

3. [ ] **Ship the native app through one-line installers.** Add Windows, macOS and
   Linux commands that download the appropriate binary, install it and launch it
   without requiring Go. Add versioned releases, authenticated artifact
   verification, upgrade/rollback/uninstall, dependency inspection and clear
   signing status.

4. [ ] **Install and update coding clients.** The native app currently requires
   Codex to be installed already. Add prerequisite handling, installation
   previews, executable verification and updates that respect existing
   package-manager ownership. Expand Codex compatibility beyond **0.166.0**, with
   checks before accepting newer versions.

5. [ ] **Implement the other five client adapters.** Claude Code, Pi Agent,
   OpenCode, Grok Build and Hermes currently have detection only. Each needs
   verified configuration/authentication contracts, configuration and launch
   support, preservation of existing settings, rerun behavior and rollback tests.
   Hermes needs particular care around customizations.

6. [ ] **Complete persistent credentials across platforms.** Windows Credential
   Manager works; bounded macOS Keychain and Linux Secret Service implementations
   remain. Complete the experience for launching clients directly, such as bare
   `codex`, when the selected key lives in a vault rather than an existing
   environment variable.

7. [ ] **Finish model metadata presentation and caching.** Gateway discovery and
   models.dev enrichment already exist. Still needed: visible capacity/provenance
   information, persistent inventory and metadata caches, freshness indicators,
   offline fallback, explicit refresh failures and deliberate mappings for proxy
   aliases.

8. [ ] **Add per-model and shared context budgets.** Today there is one active
   budget per client profile. Add remembered budgets per model, links across
   clients/models, reserve and compaction constraints, and revalidation when
   switching. Preserve the 272000/353000/500000 presets.

9. [ ] **Generalize service-tier support.** Replace the current combined
   Fast/Ultrafast confirmation with individual capabilities for each model and
   route. Support Fast-only routes and models beyond the current Sol/Astra
   preset. Switching to a model without the selected tier should reset the
   default to Standard with an explanation.

10. [ ] **Complete the smooth first-run experience.** Connect missing-client
    Install/Skip through configuration and launch. Allow discovery against the
    endpoint currently being edited, before its first Apply. Remember
    theme/display preferences and the user's decision to decline the monitor.
    Help, command search and ordinary draft navigation have functional coverage;
    their consistency, discoverability and polish remain subject to the P0
    visual and interaction remediation above.

11. [ ] **Expand jobs, history and diagnostics.** Add phase-level progress,
    timestamps and durable records for failures, cancellations, conflicts and
    interrupted work, alongside successful Apply receipts. Connect restart
    recovery to visible job outcomes and provide previewable support bundles.
    Cancellation and transaction recovery already exist.

12. [ ] **Complete the optional background monitor.** Implement the Windows
    background-task backend. Validate actual Linux systemd-user and macOS
    LaunchAgent lifecycles, including login/logout, sleep/resume and unavailable
    backends. Extend observation as additional clients become supported.

13. [ ] **Settle and implement maintenance policy.** Decide what checks for client
    updates, gateway model changes and configuration drift; how often; and which
    changes need review. The current observer only records local configuration
    state. Automatic maintenance remains a design choice requiring explicit
    opt-in.

14. [ ] **Implement agent-assisted repair.** Extend the existing redacted
    diagnostic brief into: select a working agent → preview its inputs → run in
    an isolated workspace → inspect the proposed patch → validate and apply.
    Automatic repair remains a later optional feature.

15. [ ] **Finish native Windows integration.** SSH currently hands off to the
    working PowerShell helper. Add fuller native status, review, elevation and
    result handling; later support explicit additional subnet/Tailscale/WireGuard
    scopes while retaining Private + LocalSubnet defaults. Bring the
    file-extension and hidden-file preferences into the native app; the Windows
    installer already applies them.

16. [ ] **Implement utility management.** Add verified installation, status,
    update and removal paths for herdr, tmux, T3 and Paseo, respecting their
    platform limitations. Identify the intended Orca project before implementing
    its installer.

17. [ ] **Complete release acceptance and documentation.** Exercise fresh install,
    rerun, upgrade, interruption, recovery and uninstall on supported machines,
    plus an explicitly requested live gateway session. Refresh older
    planning/compatibility documents and publish an accurate
    client/version/platform support matrix.
    Publish the reviewed companion-browser Pages gate once a credential with
    GitHub `workflow` scope is available. The current OAuth credential rejected
    that workflow change; source/docs can be published separately and the same
    focused gate remains runnable locally. Preserve the reviewed change on local
    branch `wip/reference-ci-wiring` until it can be published and observed in CI.

18. [ ] **P0 — bring every native TUI screen and interaction up to the approved
    preview's standard.** Address the explicit visual rejection, inconsistent
    controls and cumbersome tab/menu navigation. Complete the
    [detailed remediation and acceptance gates](#p0-native-tui-visual-quality-and-interaction-remediation)
    above using `visual-review-and-fix`, `ultra-code-workflows`, independent
    subagents and actual native/mockup comparisons. **Highest priority; open.**

19. [ ] **Keep the console interactive when color is disabled.** Discovered
    2026-11-03: `useTUI` in `cmd/oneline-ccc/signals.go` treats `NO_COLOR` as
    read-only plain mode, so a user requesting no color loses all configuration
    controls. Separate presentation capability from entry mode: on a real TTY,
    `NO_COLOR` should keep keyboard-complete monochrome management; explicit
    `--plain`, redirected streams and incapable terminals retain plain status.
    Carry the same decision into signal ownership, help and native usage, with
    tests for entry mode, no-color rendering, typing and safe interruption.
    Observe populated credential-editor frames and each focus transition before
    cancellation; a burst of typing plus Escape can be coalesced by the renderer
    and cannot prove masking or correct editor navigation.
    Inspect reconstructed cells as output is processed and latch any dummy-key
    exposure, including during final drain. Cursor-addressed fragments can
    display a complete key and erase it in the same PTY/ConPTY read; raw/stripped
    transcripts and the final screen alone miss that case. Keep one-batch,
    split-batch and drain negative controls alongside valid masked journeys.

20. [ ] **Restore terminal ownership after startup and forced-error failures.**
    Discovered 2026-11-03 in the shutdown review: upstream initialization/error
    paths can return after changing terminal modes without the ordinary shutdown
    path, and forced Kill/panic paths skip graceful reader joining. Add real
    terminal failure injection after mode changes, including simultaneous
    cancellation; verify modes, foreground process ownership, readers and owned
    children are restored/joined on every affected exit. Cover supported Windows
    and macOS paths as well as Linux. A nil-model/headless startup-error test and
    successful cross-compilation do not prove these cases.
    Also close resources allocated during failed reader construction: the pinned
    cancelreader's Linux EpollCtl error path can leave its private wake-pipe files
    open after it returns no reader. A process exit or eventual finalization does
    not prove cleanup before return. Reproduce the constructor failure, verify
    descriptor closure and retain audited dependency provenance alongside the
    terminal-mode and input-join evidence.
    Check cancellation again before admitting a new model-initialization callback
    after renderer startup; keep callbacks outside terminal-ownership locks so
    synchronous cancellation cannot deadlock. State the observable-check boundary
    accurately rather than promising atomic exclusion of a racing cancellation.

21. [ ] **Make native visual evidence fail safely and remain reproducible.**
    Discovered 2026-11-03 in independent capture-tool review: valid blank PNGs,
    mismatched scenario/cell evidence and stale renderer metadata can currently
    pass verification; a failed capture leader can also leave descendants alive.
    Reject blank/header-only/wrong-target, fully transparent and inconsistent
    cell/text artifacts. Assert each named state and variant from its actual
    semantic outcome: Unicode content, Help last page and invalid-editor refusal
    cannot pass by repeating requested labels. Preserve these predicates as
    inspectable evidence as well as checking the live model. Independently
    verify the intended native screen/editor/overlay was reached, and clean up
    the entire owned process group even after leader exit. Preserve immutable
    renderer provenance when resuming partial batches; record untracked build
    inputs and distinguish capture-tool revision from served-reference source
    hashes. Pin these failure cases with executed negative regressions and retain
    honest provenance for existing captures rather than rewriting their history.
    Preserve schema 1/2 verification when old captures lack client observations:
    retain their original integrity/content checks and explicitly label absent
    semantic proof. Keep current-schema observations mandatory. The stricter
    predicate must not make the 361 original rows without client metadata
    unverifiable or fill their missing fields from a newer fixture.
    Validate the shape of recorded client observations: accept arrays and the
    helper's explicit nil-slice `null`, but reject objects/scalars. An empty-state
    label with `fixture.clients = {}` must not receive observed-state proof.
    This supports P0 acceptance; artifact integrity alone is not a visual PASS.
