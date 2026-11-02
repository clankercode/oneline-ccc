# Product completion campaign

Started: 2026-11-03. Baseline: `0fab61e`.
Authority: [remaining items](remaining-items-so-far.md), [product brief](../product-brief.md), [production plan](tui-production-plan.md).

Complete every original and subsequently discovered backlog requirement. This is
an execution map; the backlog remains the acceptance authority. Keep original
item IDs stable, preserve evidence of partial completion and never close a broad
item using a narrow gate.

## Ordered milestones

| Milestone | Backlog IDs | Deliverable and evidence |
| --- | --- | --- |
| A: native usability and visual repair | 18, 1, 19–20, relevant 10/17 | Predictable focus regions and section switching, polished native screens/overlays, real styled captures against pinned mockups, repeated independent visual/source reviews; interactive no-color mode, race-free joined input and startup/error cleanup |
| B: remembered choices and model policy | 6–10 | Bounded platform vaults, persistent preferences and inventory, per-model/shared budgets, per-route tier capabilities, first-run discovery/configuration and effective direct client launch |
| C: distribution and client lifecycle | 3–5 | Authenticated binary releases and bootstraps, safe update/rollback/uninstall, prerequisite/client install and version support, five independently verified client adapters |
| D: durable work and maintenance | 11–13 | Durable job states/recovery/diagnostics, Windows task backend, actual service lifecycle evidence and explicit opt-in maintenance policy |
| E: repair, Windows and utilities | 14–16 | Operator-invoked isolated agent repair and patch review, native SSH/Explorer controls, identified and verified utility lifecycle |
| F: complete acceptance | 2, 17, all | Real native machine trials, supported OS/version matrix, install/rerun/upgrade/interruption/recovery/uninstall, requested live session, final integrated and per-screen evidence |

Independent tasks may proceed concurrently in isolated worktrees with disjoint
ownership. Integrate reviewed tasks serially and verify the merged tree. Native
machines, signing/release identity and ambiguous utility identities require
evidence before their affected acceptance claims, while independent work proceeds.

## Milestone A task boundaries

1. **A1 — Terminal shutdown race.** Own dependency selection/local fix and
   rendered foreground-shutdown regression only. Reproduce the current race,
   correct its lifetime ownership, enable the actual PTY test under `-race`,
   and run ordinary/race process and terminal gates. Do not hide the regression
   behind `!race` or remove the renderer. Independent review must assess both
   behavior and the dependency/platform implications.
2. **A2 — Native capture infrastructure.** Own fixture/capture tools under
   `tests/tui-visual/` and a narrowly scoped test helper in `internal/tui/`.
   Capture real `tui.New` output from a PTY with deterministic fixture state;
   faithfully render captured ANSI/cells to PNG. Record build revision, theme,
   size, font, state and capture hash. Capture all ten screens, modal/editor
   variants and state cases; allow baseline/final builds and mockup comparisons.
   No operator state, credential store or network calls. A static HTML imitation
   of the app cannot be used as native evidence.
3. **A3 — Native navigation and composition.** Own `internal/tui/` production
   and behavior tests, excluding the A2 capture helper. Replace flat arrow
   traversal with explicit navigation/content regions; implement horizontal
   section movement, forward/reverse region/control traversal, stable return
   focus, local list/form keys, visible routes to every destination and accurate
   contextual help. Preserve guards, typing, exact review/apply authority and
   draft retention. Match the approved color/surface/hierarchy/list-detail
   language across every native screen, with intentional compact fallbacks.
4. **A4 — Alternating native visual and interaction loops.** Capture the exact
   integrated A2/A3 revision at the required sizes/themes. Inspect individual
   native and mockup images plus crops; independent source/geometry review;
   fix, commit, render and repeat at least twice per modality for substantive
   polish. Extend reference designs for native surfaces without mockups. Every
   applicable matrix row needs both independent passes on the same final
   revision, and real native keyboard journey evidence. Review shared changes
   across all affected screens. Install/smoke the reviewed integrated build.

Review Focus: input text and paste never trigger global navigation; arrows in
tab navigation never toggle management; reverse traversal and dialog Back return
to the prior control; narrow/Unicode content keeps focused actions visible;
SIGTERM during raw-mode child ownership restores terminal state without a race.

## Execution and evidence

Use `ultra-code-workflows` with durable per-agent briefs/reports, isolated
worktrees, independent implementer/reviewer seats, explicit rulings and
implement/review/fix/re-review gates. Use `visual-review-and-fix` for rendered
quality. The ignored controller workspace is
`.superpowers/sdd/product-completion/`; preserve milestone ledgers and final
evidence under `docs/ledgers/` and `docs/verification/` when integrated.

The global gate includes appropriate Go tests/race/vet, pinned native Codex
journeys, six static builds, browser preview tests and native visual/keyboard
captures. Run targeted gates per task and complete milestone gates before
publication. Use disposable roots and scratch under the checkout, with build
concurrency bounded to two. Never use real client configuration as a fixture.

## Dynamic backlog expansion

Record each discovered defect or missing capability in the remaining-items file
with trigger/effect, intended result, status and evidence/owner. Fold details
into an existing item when it is the same obligation; assign a new stable ID
when independently testable work is discovered. Do not inflate speculative
features, duplicate existing obligations, or quietly relax the approved UX.
Routine implementation choices proceed under the user's full-goal authorization.
