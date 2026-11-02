# Targeted independent capability-boundary re-review

SpecPASS / QualityPASS for the bounded fix. Findings: Critical 0; Important 0; Minor 0. RR1 is resolved. RR2 requires the final merged-tree integration check described below.

Frozen range: `b14af2c..6fc00eb2110344a60ae58e84edc6437fbb7cc174`. Read `boundary-fix-report.md`, `boundary-fix.diff`, all three changed files at their relevant seams, and the Python TOML assertion helper. No source edits, delegation, commits, attention notifications, inference, real credentials, registry access or operator-configuration mutation. Only this report and `.cache` scratch were written.

## RR1 verification

The native merger now always emits a `replace` edit for `features.ultrafast_mode`: true when enabled, otherwise JSON null to remove the setting (`scripts/setup-codex.ps1:849`). It edits this dotted key alone; it does not replace the features table. Existing post-write validation starts a fresh native process and checks every edit, including requiring the deleted setting to be absent (`scripts/setup-codex.ps1:899`). The generated-config fast path and named-profile generation continue to produce the corresponding feature state.

Fresh independent command:

```text
DOTNET_PROCESSOR_COUNT=2 /home/xertrov/src/oneline-ccc/.cache/powershell/pwsh -NoProfile -File .cache/boundary-rereview-checks.ps1
```

Exit 0. Output: `.cache/boundary-rereview-checks.log`. Fixture root: `.cache/installer-tests-6d411a1ed7384c1ba26c3163f5a979e1`.

- Existing focused rerun group: **17 passed / 0 failed**, including both new arbitrary-main restriction/route sequences and the existing enabled-menu preservation, active native-main import, saved-key/Enter, overrides and context cases.
- Two additional independent sequences started with a custom model, priority selected, Ultrafast available, and arbitrary main TOML containing an unrelated comment plus `features.web_search_request = false`.
- Before disabling anything, an omitted-input native merge retained `ultrafast_mode = true` in both main and named TOML and retained the unrelated feature in main.
- Applying either a fast-only restriction or a changed model/default request removed the Ultrafast setting from both files. A subsequent omitted-input rerun kept it absent from both files. The unrelated main feature remained false and the comment remained present after both applications.
- Assertions used the shared Python `tomllib` parser for values and nested-key absence, not text matching for feature removal. Inspection confirms `Verify-Toml -Absent` passes dotted paths into `verify_config.py`, whose absence assertion checks actual parsed key membership. The independent comment-preservation assertion separately checked the text.

These results directly cover the previously reproduced sequence that reintroduced an unconfirmed capability. No concrete regression was found in this bounded correction.

## RR2 integration dependency

Verified read-only Git objects:

- `main` resolves to `4ee2553326ff3e9f606f98efea45507f1ec55cb1`.
- Both `4ee2553:docs/plans/tui-production-plan.md` and `main:docs/plans/tui-production-plan.md` exist as blob `06903aa67d2f312058222069cd95f39da769cd85`.
- The feature HEAD contains `docs/plans/windows-explorer-defaults.md` with its relative `tui-production-plan.md` link, while the destination remains absent from this feature branch by design.

No link rewrite or TUI-content edit is required. **Before publishing the final integration, verify the actual merged commit contains both documents and that the relative link resolves to `docs/plans/tui-production-plan.md`.** In the merged checkout, `git cat-file -e HEAD:docs/plans/tui-production-plan.md` must succeed, and the Explorer document's link must still resolve to that path. Verifying the separate main object here does not substitute for that final merged-tree check. No TUI content was reviewed.

## Other evidence and limits

`git diff --check b14af2c..HEAD` passed with exit 0. The change remains confined to the installer, rerun tests and their shared assertion wrapper. Tracked worktree status was clean and HEAD remained `6fc00eb` at final inspection.

Inspected `.cache/rerun-boundary-core-green.log`: controller's full local core gate reports **101/0**. As requested, this reviewer did not repeat the full suite. Fresh independent evidence here is the 17-case focused run plus the two preservation/boundary sequences.

This is source and Linux PowerShell/native verification of the frozen boundary fix. No current-commit Windows CI, actual Explorer registry/UI behavior, deployment or hosted-byte verification was performed here. The separate operator SSH startup incident is outside scope; no diagnosis, hypothesis or SSH source change is supplied by this review.

=== DONE ===
