Spec verdict: PASS — no Missing/Extra/Misunderstood requirement established in the two-file integration correction.

Quality verdict: APPROVED — no actionable Critical, Important or Minor finding identified within the scoped change.

# Independent A4 integration verification review

Reviewed worktree: `/home/xertrov/src/oneline-ccc/.worktrees/native-references`.
Exact range: `46b50cd016917f2b7e45b898c749530db04c5ea1..f285ff8ea5db4b64c96754ac8b64f422894454bd`.

Read the review brief, saved two-file diff, complete `A4-integration-fix-report.md`, and main `A4-rereview1.md` for the approved UI boundary. Inspected the changed verifier and Pages workflow with their surrounding module-resolution, request/error handling, scratch and exit paths. The saved review diff exactly matches this range with eight context lines and explicit `a/`/`b/` prefixes. The worktree was clean at the specified HEAD.

## Checked and fine

- **Correct boundary replacement:** `tests/tui-native-reference-browser.mjs:209` removes the historical git diff/report-tracking dependency rather than repinning its SHA. Source scope belongs to the implementation review, and unrelated approved native changes no longer fail a browser consumer gate. There are no remaining `execFileSync`, `0fab61e` or ignored-handoff-report references in the verifier.
- **All 13 consumer journeys remain:** a fresh byte comparison confirms the complete block from the first F1 journey through the end of F5 is unchanged. It contains exactly 13 journeys. The replacement F0 remains a substantive fourteenth check rather than an unconditional pass.
- **Fixture persistence is checked behaviorally:** F0 captures the initial credential-reference value, saves the valid dummy environment-variable name `PAGE_ONLY_DUMMY_REF`, verifies the page reflects it, asserts empty local/session storage, reloads, and verifies restoration of the initial fixture value. It also checks visible focus after reload. This uses a dummy reference name, not an operator credential. The existing source no-fetch/no-WebSocket/no-storage check and `connect-src 'none'` assertion remain at lines 210–212.
- **Request/error rejection remains effective:** the companion serves only allowlisted site assets from its own ephemeral loopback origin. Its fresh browser context aborts and records requests to another origin, records page errors and console errors, and asserts both arrays empty at line 224. Journey assertion failures set a nonzero process exit status; an uncaught global assertion also fails the process. The unchanged `finally` closes the browser and companion server. This correction does not weaken these checks or start native code.
- **Hosted module resolution is correct:** `.github/workflows/pages.yml:31` installs pinned Playwright 1.61.1 under `.cache/browser`. Both browser scripts live under `tests/`; the explicit `../.cache/browser/node_modules/playwright/index.mjs` override therefore resolves to the installed module relative to each importing script. The companion sets its scratch environment before importing Playwright. The CI override avoids its pre-existing machine-local fallback. Fresh inspection confirms the local equivalent target exists and reports version 1.61.1.
- **Scratch and workflow ordering are correct:** the browser step creates `${{ github.workspace }}/.cache/browser/scratch` before running either browser command. The primary preview inherits that `TMPDIR`; the companion deliberately replaces its own process's `TMPDIR` with its checkout-local artifact scratch. Artifacts and server logs remain under checkout `.cache`. The pinned browser installation precedes the exercise step, the existing primary-preview command precedes the new companion `--fix1 --record=ci` command at workflow lines 45–46, and Pages deployment still depends on `verify-preview`. No `continue-on-error`, case filter or shell construct hides a failed consumer gate.
- **Approved source is unchanged:** fresh exact-range comparison confirms only the workflow and browser verifier changed. `site/`, `internal/`, `go.mod` and `go.sum` have no diff in this correction. The original approved primary-preview HTML/CSS/JS/views/dialogs also remain unchanged against `0fab61e`. This does not impose that historical restriction on separately reviewed native integration work.

## Evidence checked

Evidence root: `.cache/artifacts/native-references/integration-fix/` in the reviewed worktree.

- The copied coordinator results contain 13 passing consumer journeys and exactly the old F0 source-boundary failure, with empty browser-error and external-request arrays.
- The disposable merged-input RED contains zero passes and one failure at the expected historical native-source assertion. It has no browser errors or external requests.
- The disposable merged-input GREEN contains 14 distinct named passes and zero failures, errors or external requests. Its first 13 names match the coordinator's successful consumer journeys; the last is `F0-fixture-boundary`.
- Fresh SHA-256 checks confirm all 13 copied native inputs still match `overlay-provenance.json`. The corrected verifier in that disposable tree is byte-identical to the reviewed verifier. Thus the supplied RED/GREEN comparison actually retains the native changes that triggered the integration failure.
- The local CI-equivalent companion results independently record the same 14 passes with empty error/remote arrays. `ci-preview.log` records 30 primary-preview passes and zero failures; `preview-state.log` records nine passes and zero failures. The workflow RED/GREEN notes agree with the actual workflow ordering and environment inspected here.
- Fresh `node --check tests/tui-native-reference-browser.mjs` returned 0. PyYAML successfully parsed the workflow, and read-only comparisons verified its module override, scratch creation and deploy dependency. These were the only execution checks needed for this small review; no browser or render matrix was rerun.

## Limits

GitHub-hosted Actions was not executed by this reviewer or claimed by the implementer. The supplied browser execution records are historical local/merged-input evidence, checked against their provenance and current source; this review did not generate new browser captures or rerun the 159-render matrix. No new UI design, native all-screen acceptance, native runtime behavior or full product completion is inferred from this test/CI correction. The earlier approved browser-reference boundary remains the boundary of this approval.

Review was read-only except for this designated ignored report, authored last. No source/git edits, agents, notifications, network calls or operator-configuration access occurred. Final checks confirm the reviewed worktree remains clean at `f285ff8ea5db4b64c96754ac8b64f422894454bd`.

=== DONE ===
