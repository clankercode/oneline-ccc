# Independent bounded TUI planning review

Date: 2026-11-02. Scope: the frozen planning package, independently reviewed against the delegated brief and retained product requirements. This is a planning review, not implementation acceptance.

**Overall: FAIL. Spec verdict: FAIL. Quality verdict: FAIL.**

Findings: **0 Critical, 2 Important, 2 Minor**. The package is a usable design baseline, but the two Important coverage gaps need correction before it is accepted as the complete production plan. They require small additions to the existing slices, not an architecture rewrite or another broad approval round.

## Evidence and coverage

Read `.cache/tui-planning/{review-brief,brief,report}.md`, `docs/plans/broader-utility-brief.md`, all three frozen outputs, `README.md`, `docs/product-brief.md`, `docs/research/existing-projects.md`, and `docs/verification/windows-mvp.md`. Applied the contract-review skill directly; no delegated reviewers were used. Existing Windows results were read as historical evidence and were not rerun.

| Requirement / boundary | Assessment |
| --- | --- |
| Cross-platform one-line bootstrap and static Go intent | T0/T10 name six build pairs, native smoke, artifact authentication, rollback and the explicit macOS linkage interpretation gate. Appropriate planning coverage. |
| First-run/rerun and preservation | Source-labelled prefill, no-op behavior, owned-field changes, protected backups, stable first-write locks and interrupted recovery have concrete contracts and ET2/ET3 checks. |
| Client support | All six client identities remain in scope, with separate adapter expansion slices. Client installation is omitted: I1. |
| Credentials and usable client launch | Vault/session policy and secrecy boundaries are explicit. The default managed launch path lacks its own execution contract and acceptance coverage: I2. |
| Inventory, metadata, budgets and tiers | Exact IDs, provenance, unknown aliases, capacity attestation, unequal linked capacities and tier translation are separated and covered by ET5. No invented provider capacity is required. |
| Optional background operation | Foreground mutation ownership, read-only observer, actual OS backends, login lifetime, version mismatch and IPC identity checks are coherent. Windows Task Scheduler is distinguished from SCM OpenSSH. |
| Jobs, repair and SSH | Receipts/cancellation/recovery, sandbox-or-export repair, independent elevation, Private + LocalSubnet defaults and GitHub key acceptance are concrete. Native proof remains explicitly required. |
| Utilities and adoption | Existing-tool trial precedes custom implementation; utility identity/platform/install ownership gates and a later install queue are retained. Unresolved orca is honestly unavailable. |
| Terminal visual design | Ten requested screen examples plus a narrow example establish hierarchy, actionable status, restrained styling and semantic color tokens. Focus-marker ambiguity is M2. |
| Keyboard, resize and accessibility | Reading-order focus, safe review defaults, literal text entry, five concrete sizes, cell-width handling, ASCII/no-color/reduced-motion/plain modes and native accessibility/visual gates are specified. |
| Review/fix/release | Per-slice and integrated native gates distinguish proposed tests from executed evidence. Wayfinding is prioritized and does not block every reversible choice. |

Mechanical review: 542 lines across the three outputs; balanced Markdown fences; all relative links resolve; 11 ASCII wireframes with a maximum line width of 73 characters. These checks establish document consistency and example width, not a working responsive renderer.

## Findings

### I1 — Important — Required client installation has no planned implementation or exit test

- **Locations:** `docs/plans/tui-production-plan.md:25`, `docs/plans/tui-production-plan.md:138`, `docs/plans/tui-production-plan.md:141`, `docs/plans/tui-production-plan.md:182`; corresponding absent-client state at `docs/design/tui-screen-flows.md:77`.
- **Requirement / evidence:** `docs/product-brief.md:31` explicitly requires TUI controls for client selection/**installation** and current status. R1 also promises a configured client after first launch (`docs/plans/tui-production-plan.md:23`). The adapter contract at line 53 is Detect/Plan/Validate/Verify, T3 and its expansion queue cover configuration of native clients, and T10 installs this utility and optional utilities. None owns installing a missing coding client, its prerequisites, or the corresponding UI action. The absent Pi example offers only support status.
- **Scenario / Effect:** On a fresh supported machine without Codex, the bootstrap can install `ccc` and detect the missing client, but the planned tasks and all named tests can finish without delivering an installable/runnable selected client. The full retained installation requirement silently disappears despite the report's claim that all requirements are mapped.
- **Smallest fix:** Add a bounded client-install slice and a missing-client action to T3/T4 or the expansion queue. Name ownership for trusted source/version, native prerequisites/package-manager delegation, privilege preview, existing-install preservation, cancellation and failure receipts. Add an exit case starting with the selected client absent: explicit Install produces the verified executable and reaches configured launch, while decline/failure leaves existing state intact. Other clients can acquire this capability in their existing gated slices; no detailed six-client installer design is needed now.

### I2 — Important — Default vault-backed launch is not carried through to a process contract and acceptance gate

- **Locations:** `docs/plans/tui-production-plan.md:46`, `docs/plans/tui-production-plan.md:77`, `docs/plans/tui-production-plan.md:131`, `docs/plans/tui-production-plan.md:146`; consuming UI at `docs/design/tui-screen-flows.md:90` and `docs/design/tui-screen-flows.md:98`.
- **Requirement / evidence:** The intended outcome is usable clients (`docs/plans/tui-production-plan.md:9`), and the chosen default explicitly makes credential delivery depend on a managed launcher rather than arbitrary bare-client invocation. The boundary table names `internal/credentials/launch.go`, but T2's deliverables/tests cover storage and transactions; T3/ET4 check effective configuration; T4 wires inspection, plans, apply and status. There is no corresponding launch operation contract or exit test that proves the selected client receives the credential and owns a usable terminal. The native compatibility gate at T0 and ergonomic question at line 203 are legitimate, but neither specifies this downstream behavior.
- **Scenario / Effect:** With a synthetic key stored in a vault and no inherited key environment, ET2/ET4 can pass and the UI can show “Stored” plus a valid `env_key` reference while Launch still cannot authenticate or cannot hand the fullscreen terminal to the interactive client. This is the normal proposed default path, so configuration-only acceptance leaves the core outcome untested.
- **Smallest fix:** Assign the existing launcher file and an application Launch operation to a scoped T2/T4 follow-up. Define selected executable/arguments, config/profile selection, working directory, minimal secret-bearing child environment, vault/session lifetime, terminal suspend/restore and child exit/cancellation behavior. Add a synthetic child-process test proving exact credential delivery without persistence/logging, a locked/session-only refusal path, and native PTY handoff/return checks. This requires neither real inference nor a promise of vault support for bare client commands.

### M1 — Minor — Setup calls the API credential a private key

- **Locations:** `docs/design/tui-screen-flows.md:13` and `docs/design/tui-screen-flows.md:47`.
- **Requirement / evidence:** The requested gateway credential is an API key (`docs/plans/broader-utility-brief.md:12`). SSH accepts public keys and explicitly never imports private keys (`docs/plans/tui-production-plan.md:92`). The first-run flow and its annotation instead say “private key entry”.
- **Scenario / Effect:** A future implementer can carry this wording into setup, misleading a user into supplying an SSH private key where the gateway API key is expected.
- **Smallest fix:** Replace both instances with “API key entry”; reserve private/public key terminology for the SSH workflow.

### M2 — Minor — Two wireframes show two simultaneous focus markers

- **Locations:** `docs/design/tui-screen-flows.md:109` and `docs/design/tui-screen-flows.md:116`; repeated at `docs/design/tui-screen-flows.md:143` and `docs/design/tui-screen-flows.md:149`.
- **Requirement / evidence:** The design promises one focused action at line 9 and defines `>` as the focus marker at lines 22 and 29. Both Models and Jobs put `>` on a content row and an action in the same example, without identifying a separate selection marker.
- **Scenario / Effect:** A keyboard user or implementer cannot tell whether Enter activates the row or Save draft/Request cancel. In monochrome mode the marker is a primary focus signal, so ambiguity also weakens the accessible fallback.
- **Smallest fix:** Keep `>` on the one focused control; mark persistent row selection with a distinct labelled state, or explicitly annotate the examples as different focus moments. Pin this distinction in the keyboard/focus visual checks.

## Independent primary-source spot checks

Fetched the following supplied public sources over HTTPS during this review. Source snapshots are confined to `.cache/tui-planning/review-sources/`; they are review scratch, not pinned dependency inputs.

| Source | Observed result / limit |
| --- | --- |
| [Bubble Tea module](https://github.com/charmbracelet/bubbletea/blob/main/go.mod) and [README](https://github.com/charmbracelet/bubbletea/blob/main/README.md) | Current module is `charm.land/bubbletea/v2`; README documents the cell renderer and `View() tea.View`. Matches the dated stack claim; no released-version/native compatibility result inferred. |
| [Lip Gloss README](https://github.com/charmbracelet/lipgloss/blob/main/README.md) | Uses `charm.land/lipgloss/v2` and documents color profiles. Color downsampling is appropriately not presented as accessibility proof. |
| [go-keyring README](https://github.com/zalando/go-keyring) and [Darwin implementation](https://github.com/zalando/go-keyring/blob/master/keyring_darwin.go) | Documents Windows, Secret Service/DBus and macOS support; Darwin Set invokes `/usr/bin/security -i` and writes its command to stdin. The package correctly retains subprocess cleanup and chosen-release audit gates. |
| [Go Darwin runtime](https://github.com/golang/go/blob/master/src/runtime/sys_darwin.go) | Contains dynamic imports from `/usr/lib/libSystem.B.dylib`. Supports the plan's refusal to equate no-cgo with literal fully static macOS. Actual release binaries still need linkage inspection. |
| [models.dev README](https://github.com/anomalyco/models.dev/blob/dev/README.md) | Documents `api.json`, generated `canonical_model_id` where applicable, and context-limit metadata. Supports the separation from gateway identity/availability and actual capacity evidence. |
| [Microsoft task security contexts](https://learn.microsoft.com/en-us/windows/win32/taskschd/security-contexts-for-running-tasks) | Documents registration principal, logon types and password/privilege distinctions. Supports treating scheduled tasks as a distinct identity/backend contract; does not prove the future installer's exact account permissions or logout behavior. |

No fabricated library API was identified in these consequential checks. This was a bounded spot check, not independent revalidation of every identity or every link in the prior project survey.

## Explicit future gates, not additional findings

- Released dependency pins, actual Go/OS floors and native PTY/vault/background-manager compatibility are named T0 and platform-test gates. Reversible framework/theme preferences do not require another approval stop.
- Signing identity, bootstrap trust root, Authenticode/notarization availability and literal macOS static-link interpretation must be resolved before the affected release. The package explicitly identifies these limits; an unproven artifact is not represented as already secure or portable.
- Grok/Hermes schema behavior, older Pi compatibility, intended utility matches and orca identity block only their dependent adapter/installer slices. The full six-client requirement remains intact through the expansion queue.
- Executable repair requires demonstrated confinement; export-only repair is the stated safe fallback. Actual gateway protocol/capacity/tier support needs separately authorized evidence. These are explicit implementation/deployment gates, not reasons to reject every other planning decision.
- The documented foreign-editor race is honestly bounded: advisory locks and pre-replace hashes are not claimed to provide universal compare-and-swap against noncooperating writers.

## What a plan cannot verify

No Go implementation, dependency build, native client install/launch, real credential store, service lifecycle, SSH login, inference call, release signature, rollback execution, screenshot or screen-reader session was exercised. Wireframes cannot prove polished rendering, correct clipping/cell width, stable focus under live updates, cancellation safety, actual contrast on arbitrary terminals or accessible screen-reader behavior. ET1–ET10 and the companion visual/accessibility gate must supply that evidence later; historic Windows script results do not establish it for a future Go application.

## Review disposition and integrity

Accept after I1/I2 acquire concrete task ownership/contracts and behavioral exit criteria, then perform a scoped document rereview. M1/M2 are local documentation fixes. No broad architecture redesign or resolution of every wayfinding preference is required.

Reviewed output SHA256 values:

- `docs/plans/tui-production-plan.md`: `1265a13d9211acd045eadca9f2cf6df7560d19a0f4929689f6124bf56adea94f`
- `docs/design/tui-screen-flows.md`: `316d448b0e64715d2f5a9282e397ee9855b802b066c3f9baad72f268ac254fb5`
- `docs/research/tui-stack.md`: `78a65de28bdb4a137992e3e053d835fccf5281461fae1baccf0cfd7de2f34d15`

Only this named review report and permitted review scratch were written. No source/planning artifacts, production configuration, credentials, scripts/tests, services, commits, pushes, subagents or attention/TTS operations were touched.

=== DONE ===
