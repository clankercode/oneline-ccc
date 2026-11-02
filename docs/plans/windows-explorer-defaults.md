# Windows Explorer defaults

Subsequent requirement, 2026-11-02: the Windows MVP and future Go application
enable file name extensions and hidden files by default for the setup user.

The MVP sets only DWORD `HideFileExt=0` and `Hidden=1` under
`HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`, after
successful Codex setup and before optional SSH elevation. Existing correct values
are not rewritten. `ShowSuperHidden` and unrelated settings remain unchanged.
`-SkipExplorerDefaults` opts out; dry-run never accesses the registry and
configuration-only `-SkipInstall` also skips preferences. Failure warns without
undoing successful client setup. Refresh or reopen existing Explorer windows if
needed; setup never restarts Explorer.

The future [TUI plan](tui-production-plan.md) must carry the same default in its
Windows setup review, with an explicit opt-out, current-value inspection and an
independent operation receipt. Apply in the original operator's HKCU context,
before or separately from elevated OpenSSH work; do not use an alternate
administrator's HKCU. Keep these settings distinct from public-key authentication
and firewall scope. Cancellation before apply preserves existing values.

Verification is scaled to this reversible preference change: syntax and source
review, with native behavior a disclosed limit until exercised on Windows. No new
unit-test suite is required for the two registry values.
