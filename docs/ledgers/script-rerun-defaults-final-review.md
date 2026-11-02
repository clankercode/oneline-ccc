# Independent fresh rerun-defaults review

SpecFAIL / QualityFAIL. Findings: Critical 0; Important 2; Minor 2.

Reviewed frozen `0d9bee6..5cf745b3b33ed6b515266e39ce7cdba68e77dd8a`, the plan, implementation report, supplied review.diff, current script seams, tests and documentation. Spec and quality passes were performed directly and independently, without subagents. Only this report and `.cache` scratch were written. No source edits, real credentials/user configuration, inference, commits, pushes or attention notifications. TUI planning/review and the subsequently requested Explorer defaults are outside scope.

## Important findings

### R1 — An unchanged rerun loses an already enabled Ultrafast menu setting

Locations: `scripts/setup-codex.ps1:920`, `scripts/setup-codex.ps1:1169`, `scripts/setup-codex.ps1:1201`.

Scenario: Install a custom model without a catalog using `ServiceTier = 'fast'` and `SupportedServiceTiers = @('fast','ultrafast')`. The generated main and named configurations contain `service_tier = 'priority'` and `features.ultrafast_mode = true`. Rerun with URL/model/context/compaction omitted and no capability changes. The defaults reader retains only the selected tier, and line 1169 reconstructs only that tier's capability. Consequently `$enableUltrafast` becomes false and the generated feature setting disappears.

Effect: An unchanged rerun changes both generated configurations, removes the previously enabled menu capability, and creates redundant backups. This contradicts the plan's preservation of route capabilities and README's unchanged-generated-config/no-redundant-backups claim. The existing omitted-input test selects Ultrafast itself, which reconstructs that capability and misses this case.

Evidence: `.cache/fresh-rerun-review.ps1` and `.cache/fresh-rerun-review.log` reproduce `initial-ultrafast=True; rerun-ultrafast=False; unchanged=False`. Fixture: `.cache/installer-tests-d80c68fc45a14bad9091b86d0b1262be/review-tier-menu-preservation`.

Fix: Retain the existing owned Ultrafast feature/capability when endpoint and model are unchanged and no explicit capability restriction supersedes it. Continue to honor explicit restrictions and route changes. Add a rerun case where Ultrafast is supported/enabled but the selected request tier is priority or default; verify complete configuration bytes and absence of new backups.

### R2 — Missing-credential tests can read the operator's real Windows User key

Locations: `tests/run.ps1:136`, `tests/run.ps1:199`, `tests/run.ps1:275`, `tests/rerun-defaults.ps1:116`; production fallback at `scripts/setup-codex.ps1:995`.

Scenario: Run the suite on Windows after setup has saved a valid User-scope `LLMP_API_KEY`. `-NoCredential` clears only the child process variable. The harness replaces `Get-LLMPUserApiKey` only when a nonempty `UserKeyFixture` is supplied. Both missing-key cases omit that fixture, so their production fallback reads the actual Windows User environment.

Effect: The tests violate the explicit isolation requirement and `tests/README.md:97` claim that no real saved User key is accessed. The noninteractive missing-key case can actually succeed and activate the real key inside its child; the interactive empty-key case can reuse it and fail later for an unexpected prompt instead of exercising missing-key refusal. Results depend on the workstation's saved credentials. This is source-confirmed; no actual Windows User key was accessed for this review.

Fix: Replace the User-key getter in every isolated invocation, using an explicit absent-key fixture by default and an opt-in dummy saved key. Compose that replacement with the other fixtures. Verify both missing-key cases with a simulated nonempty external User-key source that must never be consulted.

## Minor findings

### R3 — User-key fixture discards the interactive prompt fixture

Location: `tests/run.ps1:138` (overwriting the prompt wrapper built at `tests/run.ps1:121`).

Scenario: Call `Invoke-Installer` with `NonInteractive = $false`, `-NoCredential`, a dummy `-UserKeyFixture` and scripted `-PromptAnswers @('')`. The later User-key block replaces `$invocation`, discarding the mocked `Read-Host`, answer queue and unused-answer assertion.

Effect: The combined fixture invokes a real host prompt in the noninteractive child and fails. The required interactive Enter-keeps-saved-User-key path cannot currently be tested with these provided seams, although the individual process-key Enter and noninteractive User-key tests pass.

Evidence: `.cache/fresh-rerun-catalog-review.ps1` and `.cache/fresh-rerun-catalog-review.log` reproduce the composition failure using a dummy User key only.

Fix: Preserve the prompt wrapper when applying the User-key source replacement, or apply source replacements before wrapping the final invocation. Add the combined Enter-keeps-User-key case with output secrecy and credential-activation assertions.

### R4 — Service-tier documentation still says omission always resets to default

Location: `README.md:155` (claim continues at lines 156–157).

Scenario: Rerun the unchanged LLMP route with a saved priority/Ultrafast tier, omit `-ServiceTier`, and accept Enter where prompted. The implementation preserves that saved tier, while this paragraph says that not selecting a tier explicitly writes `service_tier = 'default'`.

Effect: The service-tier reference contradicts the new rerun section and actual behavior. An operator relying on omission to reset a paid/accelerated tier would retain it. This candidate was raised by the controller during review and independently confirmed against source and the fresh passing rerun tests.

Fix: State that first runs and changed routes use default absent an explicit selection, while unchanged LLMP routes retain the saved tier; document `-ServiceTier default` as the explicit reset.

## Evidence and checks

- Fresh focused execution: `DOTNET_PROCESSOR_COUNT=2 /home/xertrov/src/oneline-ccc/.cache/powershell/pwsh -NoProfile -File .cache/fresh-rerun-review.ps1`, exit 0. The existing rerun group passed **11/0** before the additional R1 reproduction. Covers Enter defaults, process/User precedence with dummy keys, active-main precedence, unrelated-provider isolation, explicit changes, context/compaction and generated-config DryRun. The additional empty-explicit-key probe is not reported as a defect: rejecting an empty explicitly supplied credential is safe.
- Fresh catalog/DryRun checks: `.cache/fresh-rerun-catalog-review.ps1`, exit 0; output `.cache/fresh-rerun-catalog-review.log`. An inherited managed priority-only catalog refreshed to changed upstream fixture bytes without broadening tiers. An inherited external catalog kept its exact pointer and bytes and bypassed upstream download. Arbitrary-main TOML with explicit DryRun inputs produced no staging, profile or backup. R3 was reproduced separately in the same script. Catalog data came from the controller's existing public upstream fixture; no hosted model/inference call was made.
- Fresh guard checks: `.cache/fresh-rerun-guards-review.ps1`, final exit 0; output `.cache/fresh-rerun-guards-review.log`. A deterministic main-config edit at the first prompt was refused by the saved byte snapshot guard; the concurrent edit, named profile and dummy process key were preserved, with no config backups or staging. Native version probing can create its ordinary `tmp` directory, which is not a configuration publication. A malformed native config/read input produced sanitized refusal and removed its copied TOML and GUID stage. Earlier scratch-harness failures were corrected before the final passing run; they were not production findings.
- Static safety pass: snapshots are taken before prompts; exact generated grammar gates the regex extraction; arbitrary TOML uses the isolated native user layer; foreign/edited/unsafe named profiles are refused before staging. The new reader imports no inline credential fields and uses the existing private-directory, scrubbed-child-environment, stop/reap and final cleanup machinery. `RequiredStops` remains 2 by default; only the new one-process reader requests 1, preserving the privacy/config-deletion/completeness requirements for tolerated Windows sharing residue.
- `git diff --check 0d9bee6..5cf745b` passed. Tracked worktree status remained clean and HEAD remained `5cf745b` during verification. No repository-specific coding-standard file was found; quality findings concern concrete behavior and test isolation rather than speculative refactoring.
- Inspected saved controller logs `.cache/rerun-core-final.log` (**95/0**), `.cache/rerun-ssh-final.log` (**87/0**) and `.cache/rerun-native-final.log` (**PASS** bare Codex loading the LLMP default). These are prior controller evidence, not fresh full-suite reruns by this reviewer. The frozen full suite was not repeated because focused checks sufficed to reproduce the identified issues.

## Limits and outstanding evidence

This reviewer ran Linux PowerShell/native checks only. Actual Windows PS5.1/PS7 semantics, Windows ACL/sharing-residue behavior for `RequiredStops = 1`, and the real elevated optional-SSH integration were not independently executed. Cleanup failure/reap uncertainty and publication/rollback races were inspected in source; the focused concurrent-change reproduction covers a prompt-time edit, not every possible filesystem interleaving.

During finalization the controller reported CI run **41403148385**: PS5.1 installer job **125139173591 failed**, while PS7 and real SSH passed. Exact failure logs were still being fetched by the controller and were not available to this reviewer. Treat the PS5.1 failure as an additional unresolved acceptance gate; its cause is not attributed to any finding here. The reported passing Windows jobs are not independently inspected evidence in this report. Public-bootstrap deployment/hosted-byte comparison also remains unverified here. These limitations prevent a complete Windows/deployment acceptance claim even after the four findings are corrected.

=== DONE ===
