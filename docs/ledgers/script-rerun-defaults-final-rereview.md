# Scoped independent final re-review

SpecFAIL / QualityFAIL. Open findings: Critical 0; Important 1; Minor 1.

Frozen range: `5cf745b..b14af2c26143df4acf1d4e9a4720ab9680bd9ef8`. Read `final-fix-report.md`, `final-fix.diff`, the affected implementation/tests/docs and relevant existing merge seams. No source edits, registry/user-configuration access, real credentials, inference, delegation, attention notifications or commits. Writes are limited to this report and `.cache` scratch. No TUI content review or broader research redo.

## Re-review of accepted findings

| Item | Result | Evidence |
| --- | --- | --- |
| R1: unchanged priority/default selection retains Ultrafast menu | Original reproduction fixed; boundary regression below remains | Exact managed-default reader now retains the feature, native reader imports it only from active LLMP, and same-route inheritance respects explicit capability input. Both new unchanged-generated-config tests passed with complete byte/no-extra-backup assertions. However, arbitrary-main restrictions/route changes leave stale input that the new inheritance rule can promote on a subsequent rerun. |
| R2: isolate Windows User-key reads | Resolved in reviewed harness | `tests/run.ps1:50` always creates an AST-checked fixture copy before optional wrappers, replacing the sole User getter. Line 201 explicitly sets the synthetic User-key variable even for the absent-key case, rather than inheriting a workstation value. Normal harness invocations keep credential persistence disabled. |
| R3: compose User-key and Enter fixtures | Resolved | The User-key assertions append to the existing invocation instead of replacing the prompt wrapper. The new combined test passed, including hidden output and synthetic-key activation checks. |
| R4: tier omission documentation | Resolved | `README.md:175` now distinguishes saved same-route tiers from first-run/default behavior and documents `-ServiceTier default`. |
| PS5.1 JSON prompt queue | Source correction accepted; native CI reported passing | `tests/run.ps1:136` assigns decoded JSON before `foreach`, allowing the language to enumerate array elements regardless of the cmdlet's pipeline enumeration behavior. The fresh Linux PowerShell run passed both single-answer and multiple-answer cases. At finalization the controller reported all three verification jobs passing in run 41404807893, including PS5.1 core 99/0 and SSH 87/0. Native logs were not independently inspected by this reviewer. |

## Important finding RR1 — New inheritance re-enables a stale capability after restriction or route change

Locations: `scripts/setup-codex.ps1:967`, `scripts/setup-codex.ps1:1238`, with the existing one-way main-config feature edit at `scripts/setup-codex.ps1:849`. Relevant coverage gap: `tests/rerun-defaults.ps1:139`.

**Scenario:** Install a custom model without a catalog, with priority selected and both fast/Ultrafast advertised. Add an unrelated comment to `config.toml` to exercise the supported arbitrary-TOML native path. Then either (a) restrict `SupportedServiceTiers` to fast or (b) change the model and select default. On that apply, the named profile correctly omits `features.ultrafast_mode`, but the arbitrary main config retains its existing `true`: the native merger only writes this key when enabling it. On the next unchanged rerun, the new native defaults reader treats that leftover flag as the current route's capability, and line 1238 enables Ultrafast in the named profile again.

**Effect:** An explicit restriction does not survive an ordinary rerun; a changed model can gain an Ultrafast menu capability that was never confirmed for it. Main and named configurations disagree immediately after the restriction/change, then both advertise the stale capability on the next rerun. This violates route-bound capability inheritance. The new restriction test starts with exact generated main TOML and checks feature absence only in the named profile, so it misses the native-main sequence.

**Evidence:** `.cache/final-rereview-checks.ps1` reproduces both variants. In `.cache/final-rereview-boundaries.log`, both initially report `main-ultrafast=True; named-ultrafast=False`, followed by `subsequent unchanged rerun: named-ultrafast=True`. An independent Python `tomllib` read confirmed both final files contain `features.ultrafast_mode = true` for their selected model. Fixture root: `.cache/installer-tests-f706890acb8b4161b3aea1ebb8722f58`.

The identical sequence against an isolated `git show 5cf745b:scripts/setup-codex.ps1` copy leaves the named feature absent on the subsequent rerun for both variants; see `.cache/final-rereview-base-boundaries.log`. Thus the pre-existing stale-main flag becomes a new capability-inheritance regression in this fix range, rather than merely an unrelated old merger issue.

**Fix:** Synchronize the managed LLMP feature state in arbitrary main TOML when an explicit restriction or route change disables inheritance, so the next read cannot mistake stale state for a current confirmed capability. Preserve unrelated fields. Cover the arbitrary-main restriction/change and a subsequent omitted-input rerun, asserting the feature state in both main and named files. Merely checking the immediate named output does not prevent recurrence.

## Minor finding RR2 — New Explorer plan contains a broken relative link

Location: `docs/plans/windows-explorer-defaults.md:15`.

**Scenario:** Follow `[TUI plan](tui-production-plan.md)` from the newly added Explorer document in the frozen tree.

**Effect:** `docs/plans/tui-production-plan.md` does not exist and is not tracked at the reviewed HEAD. The intended cross-reference is unusable. This finding checks link existence only; separate TUI planning/review remains outside scope.

**Fix:** Point to the actual committed plan when it is integrated, or use an unlinked future-plan reference until the destination is present. No TUI-content change is required for this correction.

## Explorer helper and documentation review

No concrete defect found in the helper body or README behavior claims. `Set-ExplorerDefaults` uses the caller's original `Registry.CurrentUser`, after successful Codex publication and before optional SSH elevation. Opt-out, DryRun, SkipInstall and non-Windows guards precede registry access. The only proposed values are DWORD `HideFileExt=0` and `Hidden=1`; `ShowSuperHidden` and unrelated values are not read or written. It inspects value/type before opening the write handle, returns without writes when already correct, writes only mismatches, verifies both values, and disposes read/write handles in `finally` blocks.

Failure produces a fixed local warning with `-WarningAction Continue`, preserving the successful Codex result even if the preference update is partial. No Explorer restart, elevation or new download is added. README documents exact scope, skip modes, refresh guidance, opt-out and reversal through Explorer's View options. The plan explicitly discloses that actual registry/UI behavior has not been exercised.

Per the bounded task/developer instruction, no new Explorer unit tests were added or run, and this reviewer did not access a registry. PowerShell parsing/source inspection and the existing isolated installer cases were used. Actual Windows registry value types, access-denied behavior, visible refresh and the no-write path remain unverified runtime behavior.

## Fresh evidence and limits

- Fresh focused command: `DOTNET_PROCESSOR_COUNT=2 /home/xertrov/src/oneline-ccc/.cache/powershell/pwsh -NoProfile -File .cache/final-rereview-checks.ps1`. Exit 0; existing focused rerun group **15 passed / 0 failed**. Output: `.cache/final-rereview-checks.log`. The scratch script subsequently gained optional baseline/boundary switches; the first log preserves the original 15-case execution.
- Current boundary command: same script with `-SkipFocused`, exit 0; `.cache/final-rereview-boundaries.log`. Baseline comparison adds `-InstallerOverride /home/xertrov/src/oneline-ccc/.worktrees/script-rerun-defaults/.cache/final-rereview-base-installer.ps1`, exit 0; `.cache/final-rereview-base-boundaries.log`. These are observations reproducing RR1, not passing assertions that the defective behavior is correct.
- Frozen installer parsing returned **0 errors** and exactly one Explorer helper. `git diff --check 5cf745b..HEAD` passed with exit 0. HEAD remained `b14af2c`; tracked worktree status was clean at final inspection.
- Inspected the supplied controller logs: `.cache/rerun-explorer-core.log` **99/0**, `.cache/rerun-explorer-ssh.log` **87/0**, `.cache/rerun-explorer-native.log` **PASS** bare default loading. Those are prior controller runs, not fresh full-suite runs by this reviewer. Native smoke explicitly skips Explorer changes on the Linux host.
- At finalization the controller reported native feature run **41404807893** passing all three verification jobs, with PS5.1 core **99/0** and SSH **87/0** confirmed in its log. Feature-environment deployment was rejected as expected. This updates the earlier pending native gate with controller-reported evidence; the raw CI logs were not independently inspected here. Actual Explorer registry/UI behavior remains unverified by this review, and successful verification jobs do not establish deployment.
- Separately, the controller reported an operator Windows admin-account SSH run failing at `Restart-Service` line 493 with `StartServiceFailed` after key generation, without a rollback-incomplete message. Read-only service/event/port diagnostics are pending. This incident is outside this frozen rerun/Explorer re-review; no root cause or SSH code fix is inferred here, and SSH source was not changed.

=== DONE ===
