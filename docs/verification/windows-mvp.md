# Windows Codex MVP verification

Date: 2026-11-02. Repository: [clankercode/oneline-ccc](https://github.com/clankercode/oneline-ccc).

## Delivered behavior

The standalone PowerShell installer configures native Codex against an
operator-supplied Responses-compatible gateway. It offers the requested model
choices, explicit context and compaction budgets, optional model discovery,
catalog-bound service tiers, and explicit manual tier overrides. It stores no
credential in TOML and leaves the base configuration unchanged.

The static GitHub Pages guide serves the same installer source. No client-maintenance
service, Go binary, other-client adapter, or automatic repair was implemented.
Existing-project research is available before the broader design conversation.

## Local evidence

- Coordinator's final integrated Codex run: **65 passed, 0 failed**, portable PowerShell
  7.6.9 with Python `tomllib`. This followed the last installer changes; a prior
  concurrent run mixed versions and was discarded, then repeated on frozen code.
- Actual Codex CLI 0.165.0 loaded the generated named profile through
  `debug prompt-input`; a separate isolated base-file copy passed
  `app-server --print-startup-config --strict-config`.
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

The [core publication workflow](https://github.com/clankercode/oneline-ccc/actions/runs/41384021098)
passed in Windows PowerShell 5.1 and PowerShell 7 with the then-current 63-case suite:
installs official native Codex through npm using a dummy credential, loads the
generated profile without inference, deploys Pages only after those pass, and
then compares/downloads/executes the hosted installer in dry-run mode in both
PowerShell editions. The initial Pages environment allowed `master`; it was
corrected to `main` and the failed deployment jobs rerun successfully.

Optional SSH adds a separate Windows integration gate for capability, service,
firewall, account, effective authentication policy, actual loopback key login,
and injected late-failure restoration. That feature's final CI result is recorded
after the expanded workflow completes.

## Limits

No real API key or inference request was used. Provider availability, actual
Responses streaming/tool behavior, the selected model's capacity and gateway tier
eligibility require an authorized deployment test. Optional supplied catalogs
need to be complete and compatible with the target Codex version; installer
fixtures exercise metadata checks, not every native catalog field.

Windows runners already include Node.js. The winget Node installation branch is
reviewed but is not exercised by that installation smoke test. Windows user
credential persistence/rollback uses the standard environment API; test suites
skip persistent user-key mutation. No test claims that user-environment storage
is an encrypted credential vault.
