# Scoped final re-review — 143006f

Reviewer: Codex, inherited GPT-6, independent final-review seat. Reviewed only the resolution of I1/I2/M1/M2 and regressions introduced by `45acd35..143006f23da4d69a0074add3dad96b9c49e194cd`. No subagents, source edits, broad test reruns, real credentials, inference, operator configuration reads or publication actions.

## Inputs and source identity

Read `.superpowers/sdd/windows-mvp-finalization/final-fix-review.diff`, the prior `final-review.md`, `.worktrees/windows-mvp-final-fix/.cache/final-fix/report.md`, changed source/test/documentation seams and the relevant R9/R10 rulings. Main had already fast-forwarded to `143006f` when inspected. The four reviewed files matched that commit; the fix worktree was clean. Concurrent controller changes outside those four files were not reviewed.

Production SHA256: `4a0cc4cc35700f5d066304bc703e93ae092a68d8eae7303110ed2f9593e8540e`.

## Original findings

| Finding | Status | Evidence and assessment |
| --- | --- | --- |
| I1: supplied catalog overrides explicit restriction | **ADDRESSED** | `scripts/setup-codex.ps1:1029` checks explicitly bound supported tiers, including an empty list, against every advertised selected-model tier returned by the existing catalog reader. Both sides normalize `fast` to `priority`; comparison remains case-sensitive. Excluded tiers cause a safe refusal before snapshots, installation, configuration/cache publication, credential changes or PATH changes. Supplied bytes remain untouched. `tests/run.ps1:667` covers omitted/default, Fast, Ultrafast and empty-list refusal with exact main/named/catalog bytes, directory entries and process credential/PATH preservation; `tests/run.ps1:689` covers both compatible aliases. This implements the bounded refusal selected in R9. |
| I2: version differs from selected native executable | **ADDRESSED** | `scripts/setup-codex.ps1:1083` resolves the native executable once; `scripts/setup-codex.ps1:1089` passes that exact path into the version probe at `scripts/setup-codex.ps1:139`. That version supplies the catalog; the same path supplies staging at `scripts/setup-codex.ps1:1108` and bare activation at `scripts/setup-codex.ps1:1147`. The old later re-resolution is removed. Generic-command fallback and no-command SkipInstall remain explicit. `tests/run.ps1:550` executes distinct real version fixtures, verifies selected 0.166.0 rather than old 0.134.0, records staging/bare boundaries and checks the old executable was never probed. This implements R10 without changing npm-native discovery precedence. |
| M1: test docs promise unchanged live main config | **ADDRESSED** | `tests/README.md:28` and `tests/README.md:42` now describe the coordinated main/named transaction, preservation of unrelated semantics/comments, and exact original backup bytes. Neither passage promises byte-identical live main configuration. |
| M2: README omits automatic Astra preset | **ADDRESSED** | `README.md:103` names both exact Sol/Astra presets and their operator-confirmed capability assumption. `README.md:116` limits manual overrides to other exact IDs. `README.md:109` also explains the combined catalog/list refusal and Fast/priority equivalence accurately. |

All four previously incorrect/incomplete claims associated with these findings were rechecked; **4 checked, 0 remain wrong**.

## Verification run here

Executed only the eight scoped regression cases on merged `143006f`:

```sh
PATH="$PWD/.cache/codex-merge-research/0.166.0-linux-x64/package/vendor/x86_64-unknown-linux-musl/bin:$PATH" .cache/powershell/pwsh -NoProfile -File tests/run.ps1 -TestFilter 'final fix:*' > .cache/final-rereview-focused.log 2>&1
```

Exit **0**, **8 passed / 0 failed**. Log: `.cache/final-rereview-focused.log`; isolated artifacts: `.cache/installer-tests-ffa5625fb0864cc6a4281c8f68546674`.

Also ran `git diff --check 45acd35 143006f` successfully, checked the changed-path list, checked main/fix-worktree status, and confirmed the reviewed four files match the committed candidate. Searched version-probe call sites: the production caller passes the selected executable explicitly.

Inspected worker RED/GREEN logs: the new focused cases failed **5**, passed **3** on original source, then passed **8/0** after the fix. The five RED failures correspond to the mixed-version case and four conflicting catalog/list cases. Worker report discloses the prospective sequence and intermediate safe-diagnostic correction; no reconstructed historical RED claim is needed for this fix.

Inspected worker gate logs/status: core **84/0**, SSH **87/0**, native bare-default smoke success, all recorded exits 0. Separately inspected coordinator merged-tree logs `.cache/coordinator-final-fix-core.log`, `.cache/coordinator-final-fix-ssh.log`, `.cache/coordinator-final-fix-native.log`: core **84/0**, SSH **87/0**, native bare-default smoke success; coordinator supplied all-zero exit status. Those full suites were not rerun by this reviewer.

## New-regression review

No new regression found in the scoped diff. The restriction check uses the existing exact-selected-model tier reader, including legacy advertisements, and refuses before mutating state. Compatible alias handling preserves legitimate catalog use; omission of the restriction preserves authoritative supplied-catalog behavior. Automatic preset generation retains its existing explicit-list behavior.

The executable change retains resolver precedence, version/error checks and the no-native isolated fixture path, while eliminating the identity split. Native version stderr/nonzero/malformed cases remain in the harness; the reported full merged gate includes them. Configuration transaction, backups, rollback, cleanup uncertainty guards, SSH code and publication workflows are unchanged by this fix. The test filter defaults to all cases; fixture interception is bounded to named helpers in isolated in-memory source and does not replace the actual version-probe logic.

## Limits and assessment

The independent scoped run used Linux PowerShell, dummy credentials and fixtures. The mixed-install case models resolver outputs and observes real version probing plus staging/bare boundaries; it does not claim actual Windows npm discovery/PATH persistence. The corrective commit's Windows PS5.1/PS7, real SSH and hosted-byte/public-bootstrap pipeline remains the coordinator's next gate. The prior successful deployment at `45acd35` is not presented as deployment verification for `143006f`.

No broader behaviors or previously declined scope were reopened. No source/test changes were made; reviewer writes were this report and the scoped harness's `.cache` log/artifacts.

**Spec verdict: PASS for the scoped corrections. Quality verdict: PASS.**

**Assessment: Ready to publish — Yes, proceed with the corrective commit's gated publication pipeline.** I1/I2/M1/M2 are **ADDRESSED**. New findings: **Critical 0; Important 0; Minor 0**. This is source-review approval, with final native/publication completion still owned by the coordinator.

=== DONE ===
