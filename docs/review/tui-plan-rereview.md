# Scoped independent TUI plan re-review

Date: 2026-11-02. Scope: Pascal's frozen fix round for I1/I2/M1/M2, checked against `docs/review/tui-plan-review.md` and `.cache/tui-planning/report.md`, including concrete regressions introduced by the fixes. Applied the contract-review method directly; no delegation.

**Overall: PASS. Spec verdict: PASS. Quality verdict: PASS, within this scoped planning re-review.**

All four accepted findings are resolved at planning level. Remaining findings: **0 Critical, 0 Important, 0 Minor**. No new meaningful issue was identified in the fixes or their integration with the surrounding contracts.

## Finding disposition

| Finding | Disposition | Independently checked evidence |
| --- | --- | --- |
| I1 — Missing-client installation | Resolved | `docs/plans/tui-production-plan.md:23` and `:25` retain install-to-launch in the journey map. T3a at `:146` assigns files, PlanInstall/Install, verified source/version/artifact, prerequisites, destination/owner, privilege review, cancellation and failure receipts. `TestET4MissingClientInstall` at `:151` covers absence and failure paths; T4 at `:156` wires Install/Skip; T4a at `:165` completes installation/configuration/synthetic launch. `docs/design/tui-screen-flows.md:77` and `:85` expose the missing-client action, review and truthful failure state. Later clients supply the same verified capability in their gated slices. |
| I2 — Vault-backed launch contract/gates | Resolved | `docs/plans/tui-production-plan.md:58` defines LaunchSpec/LaunchReceipt, executable/config rechecks, literal arguments, working directory, credential resolution and session lifetime. T4a at `:159` assigns credential/application/TUI files; `:162` and `:163` define terminal handoff/restore and owned-child exit/cancellation. Tests at `:164` and `:165` cover exact synthetic credential delivery, environment isolation, locked/missing/expired references, session-only success, native PTY return and fresh installation through launch. `docs/design/tui-screen-flows.md:106` consumes the same behavior. |
| M1 — API credential wording | Resolved | `docs/design/tui-screen-flows.md:13` and `:47` now say “API key entry”. No “private key entry” remains. Public-key SSH handling remains separate. |
| M2 — Ambiguous focus markers | Resolved | `docs/design/tui-screen-flows.md:22` distinguishes the single focused control from persistent `Selected:` state and defines Enter behavior. Models at `:113`/`:120` and Jobs at `:147`/`:153` implement that distinction. `TestET1SingleFocusMarker` at `:257` pins it through keyboard transitions. |

Locations abbreviated as `:line` in a table cell refer to the immediately preceding full file path.

## Scoped regression checks

- **Installation authority and recovery:** T3a preserves pre-existing installations/configuration/credentials/PATH, keeps gateway credentials away from installers and delegates updates to the existing owner. Partial package-manager prerequisite effects are explicitly recorded; the plan does not promise atomic rollback of shared dependencies or remove them automatically (`docs/plans/tui-production-plan.md:149`–`:151`). This closes I1 without weakening the existing ownership/recovery rules.
- **Launch authority and terminal lifecycle:** The application operation, credential launcher and TUI handoff have compatible responsibilities. Secrets stay outside arguments, persistent configuration and receipts; missing/locked references refuse spawn unless the operator explicitly supplies session credentials. Ctrl+C reaches the foreground client during handoff, and exit/failure/cancellation restores terminal ownership. Non-TTY interactive launch has an explicit refusal path (`docs/plans/tui-production-plan.md:58`, `:162`–`:165`). No concrete conflict with the existing foreground-mutation or observer boundaries was found.
- **Evidence claims:** The fixture tests prove credential delivery to a synthetic child, while a separately pinned real client opens without a model prompt. Client exit zero is explicitly insufficient to claim authentication. `docs/research/tui-stack.md:58` carries the installation/launch gate dependencies without adding a fabricated library API or claiming executed runtime proof. Gateway inference/protocol/capacity verification remains a separate authorized gate.
- **UI consistency:** Install/Skip and locked-credential outcomes connect to the existing first-run/detail flows; selection and focus remain distinct in monochrome. No screen-size or navigation regression was found in the amended examples.
- **Fix report:** The I1/I2/M1/M2 mapping at `.cache/tui-planning/report.md:30`–`:35` agrees with the actual documents. Its stated pending independent acceptance and unexecuted implementation tests are accurate. No reliance was placed on the fixer's self-review verdict.

## Verification and limits

Read all three amended outputs, the original review and the fix report. Independently checked named contracts/test gates, local links, balanced fences, trailing whitespace, credential terminology and every wireframe. The package totals **566 lines**; all **11** wireframes contain exactly one focus marker. Maximum example width is **73 ASCII cells**; the narrow example is **26**, within its 40-column budget. These are document checks, not execution of the proposed tests.

No implementation/native installer, child-process credential test, PTY handoff, vault operation, service, SSH login, inference, release-signing or accessibility test was run. Native compatibility, trust/signing, executable repair confinement and unresolved client/utility identity gates remain future implementation/release requirements, as already accepted in the original review. PASS accepts the corrected planning package within scope; it does not certify a production implementation. No repeat upstream retrieval was needed because the fixes add application contracts and gates rather than new upstream factual claims.

## Frozen input integrity

SHA256 values identify the exact reviewed inputs:

| Input | SHA256 |
| --- | --- |
| `docs/plans/tui-production-plan.md` | `5da89c40d2f4d0a49f110ad26032b05b86d8994a07784b2ec13b473c7cdffbc8` |
| `docs/design/tui-screen-flows.md` | `02fe79fe19beccd5314b5252f291c221902e66f27002ef55203e67b17e809e2c` |
| `docs/research/tui-stack.md` | `d07b1ecec850d97ec4a656fdbcfd2ce039fb337ded040329cf3949b6add7f8d1` |
| `docs/review/tui-plan-review.md` | `dcd9ca9d06edecfdde45700f2ef52781fd04f34db3c8231c34fe5929af1adbf5` |
| `.cache/tui-planning/report.md` | `ac93dc23749493b379de84a5a357d960f24a64193f7941173845e627550f20d1` |

Only `docs/review/tui-plan-rereview.md` was written. No source/planning edits, scratch writes, implementation, production configuration/credential reads, delegation, commits, pushes or attention/TTS calls. Controller owns final verification and notification.

=== DONE ===
