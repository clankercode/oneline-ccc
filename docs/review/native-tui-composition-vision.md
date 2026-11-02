# Native composition: interim visual review

Recorded 2026-11-03. **Visual acceptance: OPEN; page bodies require further work.**
The operator's rejection and the full [P0 requirements](../plans/remaining-items-so-far.md#p0-native-tui-visual-quality-and-interaction-remediation)
remain authoritative. Adding color to the shell does not make the app match its
mockups or make its workflows comfortable.

## Actual inspected revision and evidence

The coordinator opened all ten individual native base-screen PNGs captured from
integrated **`40ba57b`**, using production `tui.New`, Bubble Tea in a disposable
PTY and xterm replay. Each is 120×40 cells, dark/truecolor, fixture `parity`.
The [index](../verification/assets/native-composition-20261103/index.json)
records source/tool provenance, terminal metrics and image hashes. The complete
local ANSI, cells, production View and lifecycle evidence remains in
`artifacts/native-integrated-40ba57b-screens/manifest.json`; the verifier accepted
all ten rows while explicitly leaving visual acceptance open.

The coordinator also opened the original Dashboard/Models/Review/Jobs target
PNGs from reference revision `0fab61e`, and the historical six secondary-screen
target PNGs. Those latter images predate the scoped reference fixes, so they
serve as structural design comparisons only. The [versioned primary target](../../site/tui-views.mjs)
and [reviewed secondary reference](../verification/tui-native-references.md)
remain the source authority for final comparisons. Review and Jobs reference
images were empty states while these native specimens contain a plan/receipt;
they are not matching-state acceptance pairs. Repair's reference had a prepared
brief while the native specimen had not prepared one. Missing matching state
references and their final source pins remain required work.

## Findings and required corrections

| Native specimen | Observed condition | Required result and disposition |
| --- | --- | --- |
| [Dashboard](../verification/assets/native-composition-20261103/dashboard.png) | Multiline clients, selected-client detail, model · default tier, active tab, filled next-action notice and distinct actions now have meaningful hierarchy | Improvement observed. Compact dirty-focus/filter images, all appearances and final same-revision independent visual/layout gates remain open |
| [Models](../verification/assets/native-composition-20261103/models.png) | A long radio/control list exposes every preset and Advanced editor; the main path and actions sit below the initial form | Concise grouped model/budget/tier fields, intentional Advanced disclosure, separate applied/draft summary and visible useful actions; implementation assigned as A9, not yet integrated |
| [Review](../verification/assets/native-composition-20261103/review.png) | Plan identity, desired settings and before/after values share nearly identical treatment; the page reads as a raw text listing | Distinct plan identity, field names and before/after roles, legible paging/acknowledgment and action hierarchy. Preserve every full value and exact-plan authority; no decorative clipping |
| [Jobs](../verification/assets/native-composition-20261103/jobs.png) | Receipt ID/client/result/message form one pipe-separated line; the initial focus highlights an unavailable cancellation action despite no active job | Readable receipt groups and status hierarchy, separate active-job and history areas, useful idle next step and an explained unavailable cancellation state |
| [Gateway](../verification/assets/native-composition-20261103/gateway.png) | Three summary sentences precede a plain action list; connection, credential storage and verification state have no distinct groups | Connection and credential groups, readable field/value rows, explicit session/persistent status and deliberate actions; retain masking and truthful unverified states |
| [Service](../verification/assets/native-composition-20261103/service.png) | Backend/observed/message lines and lifecycle commands form an undifferentiated stack | A clear observed-status surface, ownership/scope details and grouped inspection/install/lifecycle actions. Show unavailable-state reasons; never imply the monitor is running from a menu action alone |
| [Repair](../verification/assets/native-composition-20261103/repair.png) | A boundary sentence and Prepare/Export list provide little hierarchy | Separate selected target/scope, exact redacted brief or useful empty state, Prepare/Export actions and sharing guidance. Match prepared state with prepared state; preserve manual operation |
| [SSH](../verification/assets/native-composition-20261103/ssh.png) | Two sentences and a plain handoff command provide little structure | Distinct handoff status, steps/terminal ownership, access/verification state and a clear review action. Do not imply server/firewall setup or remote-login success prematurely |
| [Utilities](../verification/assets/native-composition-20261103/utilities.png) | Repeated unavailability prose and a highlighted disabled control occupy the initial body | Coherent availability/operation presentation, a useful return route and concise platform/source reasons. Keep unsupported operations truthful while lifecycle support is implemented |
| [Settings](../verification/assets/native-composition-20261103/settings.png) | Display choices, credentials, utilities and monitor routes are one undifferentiated list | Group display, credentials and related tools; align values and explain the scope/persistence of choices, with consistent controls and return focus |

The coordinator generated and opened two nearest-neighbor crops:
[Review values](../verification/assets/native-composition-20261103/review-values-zoom.png)
and [Jobs receipt/actions](../verification/assets/native-composition-20261103/jobs-receipt-actions-zoom.png).
They confirm weak before/after and receipt/action hierarchy. These are actionable
visual defects; readable strings and passing functional tests do not close them.

## Independent source review and integrated gates

The [initial composition review](native-composition-review.md) found hidden
compact drafts and filter scope. The [scoped rereview](native-composition-rereview.md)
approved their correction at `63006cf`, including Unicode and focus/selection
probes. The [Help readability review](native-help-readability-review.md) approved
wrapped-row pagination while explicitly rejecting any claim of finished Help
design. Those are source/geometry and behavior verdicts, not vision approvals.

The composition was integrated through `1ee8fc5`, followed by Help at `40ba57b`.
The first full merged normal gate caught a stale Linux PTY expectation for the
old `| Draft` separator. Test-only `b8d7d09` follows the actual pending-draft
message and retains staged-management, masking, guarded-exit and restoration
checks. The focused real-PTY test and fresh full normal/race/vet gates passed on
that integrated source. This does not certify native Windows/macOS behavior.
The independent [documentation/evidence review](native-composition-evidence-review.md)
checked saved review copies, hashes, source associations and gate claims; its
PASS concerns evidence accuracy, not product appearance.

This round is **not a final visual or interaction PASS**. It covers ten base
screens at one size/theme, not editors, layers, compact states or every journey.
Continue alternating actual PNG/crop inspection and independent source/geometry
review, fixing and recapturing each affected surface until both pass the same
final revision. Install and verify that reviewed build before a readiness claim.
