# Windows Codex MVP verification

Date: 2026-11-02. Repository: [clankercode/oneline-ccc](https://github.com/clankercode/oneline-ccc).

## Current rerun and Explorer update

Release checkpoint `8e2ac76` includes reviewed runtime correction `6fc00eb`.
[Publication run 41406318242](https://github.com/clankercode/oneline-ccc/actions/runs/41406318242)
passed all six jobs: Windows PowerShell 5.1 and PowerShell 7 each passed
**101 configuration cases and 87 SSH cases**, native bare-Codex loading passed,
real Windows SSH login/late-failure rollback passed, Pages deployed, and both
public scriptblock/download checks passed. The coordinator independently fetched
both hosted scripts and compared their bytes with the reviewed source.

The Codex script SHA256 is
`4f60f562d0496df7c8992e373005f503676fd4e8e40c089457d272d6fb7f632c`.
The unchanged SSH helper SHA256 is
`91aefa3fce6e09198e86a382ccbf7615fc923c803f2d8a1ea3b86bd1b19230b4`.

Enter retains saved LLMP URL, hidden process/User key, exact model, tier and
context budget. Unchanged generated configuration creates no extra backup.
Native arbitrary-TOML merges preserve unrelated fields and clear a disabled
Ultrafast flag so it cannot return on a later rerun. Explicit arguments and
capability restrictions take precedence. The isolated test harness substitutes
an absent or dummy Windows User key on every invocation.

Normal Windows setup now sets the original user's Explorer DWORD values
`HideFileExt=0` and `Hidden=1`, with `-SkipExplorerDefaults` opt-out. Protected
OS-file visibility is unchanged. Dry-run and configuration-only runs skip registry
mutation. Actual Explorer registry/UI behavior was not exercised by these native
configuration tests, which use isolated configuration mode.

An operator's actual Windows SSH run separately reached service restart and
failed to start `sshd`. Its cause remains unresolved pending read-only service,
event-log and listener diagnostics. Runner success does not establish SSH success
on that machine; this release makes no such claim and changes no SSH source.

The [TUI plan](../plans/tui-production-plan.md) passed a separate scoped review;
it remains a plan, with no Go TUI implementation. Earlier evidence below records
the previous MVP checkpoints.

## Delivered behavior

The standalone PowerShell installer configures native Codex against an
operator-supplied Responses-compatible gateway. It offers the requested model
choices, explicit context and compaction budgets, optional model discovery,
catalog-bound service tiers, and explicit manual tier overrides. It stores no
credential in TOML. The updated installer makes LLMP the main configuration default
while preserving unrelated user settings. A named LLMP profile is retained as well.

The static GitHub Pages guide serves the same installer source. No client-maintenance
service, Go binary, other-client adapter, or automatic repair was implemented.
Existing-project research is available before the broader design conversation.

## Local evidence

- Coordinator's final integrated Codex run: **84 passed, 0 failed**, portable PowerShell
  7.6.9 with Python `tomllib`. This includes complete-catalog tests and eight final seam regressions.
  The catalog tests fail against the earlier checkpoint without generation;
  the final regressions reproduced conflicting tier inputs and mixed executable
  versions before their fixes.
  The final run covers rich TOML/comment preservation, exact backups, and four
  injected second-file failure states with config, credential and PATH rollback.
- Actual Codex CLI 0.165.0 loaded LLMP using bare `codex debug prompt-input`,
  without profile or config overrides. The smoke test also queried the effective
  native app-server configuration. The test owner independently checked 0.166.0.
- Native 0.166.0 app-server checks passed for generated Sol and Astra catalogs,
  preserving every non-tier model field. A fresh fixture TUI displayed the exact
  `/ultrafast` command and toggled to ultrafast then back to default (two opposite
  transitions, exit 0, no model prompt submitted). An initial automation attempt
  assumed the starting state and timed out; a corrected state-aware probe passed.
- Independent review found no unresolved major findings after fixes and targeted
  repros. See [the review](../review/windows-mvp-review.md).
- Chromium screenshots were visually inspected at desktop/light and mobile/dark.
  Browser checks verified both themes, no horizontal document overflow, exact
  copied text, copy feedback, and denied-clipboard recovery.
- Both workflow files pass `actionlint` 1.7.12. An initial GitHub workflow
  expression error was corrected before native jobs could run.
- Every reachable commit was scanned for private deployment strings. The
  operator's endpoint is absent from published source and Git history.

## Native Windows and publication evidence

The [full publication workflow](https://github.com/clankercode/oneline-ccc/actions/runs/41397670175)
passed at corrective checkpoint `143006f`: Windows PowerShell 5.1 and PowerShell 7 each
passed **84 Codex cases and 87 SSH cases**, installed official native Codex with
a dummy credential, and loaded LLMP through bare `codex`. The new catalog cases
exercise complete version-matched metadata, native `model/list` Ultrafast name,
automatic catalog creation, explicit tier restrictions, immutable reruns, and
tamper refusal. The real SSH integration also passed in this same run. Conflicting supplied
catalog/tier restrictions refuse without changing state; mixed executable
fixtures verify that one selected native binary supplies version, catalog,
staging and bare-command activation.

Deployment then succeeded, followed by public download/byte comparison and
scriptblock dry-run execution in both Windows PowerShell editions. A separate
coordinator fetch confirmed the hosted Codex script SHA256 equals the tested
source: `4a0cc4cc35700f5d066304bc703e93ae092a68d8eae7303110ed2f9593e8540e`.
The site is https://clankercode.github.io/oneline-ccc/.

Optional SSH adds a separate Windows integration gate for capability, service,
firewall, account, effective authentication policy, actual loopback key login,
and injected late-failure restoration. The [dedicated native SSH run](https://github.com/clankercode/oneline-ccc/actions/runs/41392708855)
passed: actual non-admin loopback public-key login, effective SYSTEM policy
validation, and injected late service-restart failure with exact configuration,
key, ACL, firewall and service restoration. Its disposable runner disables only the known broad default OpenSSH rule
before setup and requires enabled firewall profiles with default inbound Block.
Other app rules stay intact. The production helper refuses broad or ambiguous
app/firewall rules requiring operator review.

## Limits

No real API key or inference request was used. Provider availability, actual
Responses streaming/tool behavior, the selected model's capacity and gateway tier
eligibility require an authorized deployment test. Optional supplied catalogs
need to be complete and compatible with the target Codex version. Automatic
Sol/Astra presets clone the complete official selected-model entry for the
installed release; they require its public version tag to remain available.
Preset Fast/Ultrafast metadata reflects operator-confirmed LLMP routing, not
automatic subscription entitlement discovery. Other routes can restrict tiers
or supply their own complete catalog.

Native configuration validation uses private temporary staging storage. The
installer stops and reaps its owned native processes and removes the copied
configuration. On Windows, an incidental native temporary-file sharing lock may
leave protected auxiliary files after bounded cleanup retries; setup reports a
warning only after confirming the copied configuration is gone, process tracking
is complete, and the staging directory remains private. Other cleanup failures
remain fatal.

Windows runners already include Node.js. The winget Node installation branch is
reviewed but is not exercised by that installation smoke test. Windows user
credential persistence/rollback uses the standard environment API; test suites
skip persistent user-key mutation. No test claims that user-environment storage
is an encrypted credential vault.

## Final workflow closeout

The fresh whole-source review found two Important seams and two Minor doc
issues. A fresh implementer fixed all four in one isolated pass; new tests
reproduced five failures before the fix, then all eight focused cases passed.
The same independent reviewer approved the scoped correction with no new
findings. Coordinator merged-tree gates passed before the corrective Windows
publication run above. [Ledger](../ledgers/windows-mvp-finalization-ledger.md),
[final review](../ledgers/windows-mvp-finalization-final-review.md),
[fix report](../ledgers/windows-mvp-finalization-final-fix.md), and
[scoped approval](../ledgers/windows-mvp-finalization-final-rereview.md) retain the
evidence and workflow adaptations. No broader TUI/daemon implementation began.
