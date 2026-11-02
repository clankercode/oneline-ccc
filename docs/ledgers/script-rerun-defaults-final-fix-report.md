# Bounded final fix and Windows preference addition

Initial independent review: final-review.md, 0 Critical / 2 Important / 2 Minor.
All accepted; no deferral.

- R1: retain the LLMP-scoped Ultrafast feature from exact generated/native user
  configuration when endpoint/model stay the same, without an explicit capability
  restriction. Route changes and explicit tier restrictions still take precedence.
  Two new tests (priority/default request with Ultrafast available) failed with
  extra backups before the fix; both now retain complete config bytes. Added
  compatibility checks for native-main feature import and explicit restriction.
- R2: every isolated test invocation creates an AST-checked copy replacing only
  the User-key getter with a synthetic fixture source. Absent is the default;
  no real Windows User key is read. The replacement precedes all optional fixture
  wrappers and the child explicitly overrides the fixture variable.
- R3: saved User-key selection no longer overwrites the interactive wrapper.
  The combined Enter/User-key test failed before the fix and passed afterward,
  including hidden output and activated synthetic credential checks.
- R4: corrected the tier reference and documented `-ServiceTier default` reset.
- Native PS5.1 gate: run41403148385 passed PS7/actual SSH but failed one simulated
  Enter case. Its JSON reader enqueued the entire array as one item on PS5.1;
  assign decoded JSON before foreach, compatible with both enumeration behaviors.
  Full native rerun remains required; no deployment claim from that failed run.

Before adding Explorer defaults, full local core99/0 passed. The three new bug
cases had a recorded RED of11/3; original full suite remained95/0 before those
additions. One compatibility case was added without a prior RED, explicitly.

Pascal implemented the subsequent user request in an isolated worktree, only
scripts/setup-codex.ps1 and README.md. Controller inspected/applied its patch:
original-account HKCU DWORD HideFileExt=0 and Hidden=1, read before write,
no ShowSuperHidden change, opt-out, dry-run/non-Windows/config-only skips,
post-success/pre-SSH placement and fixed local warning. No Explorer restart,
admin requirement or source downloads. See .cache/explorer-defaults-report.md
in the main checkout. Requirement also recorded for the future Go TUI.

Ruling R5: no new tests for the reversible low-impact Explorer preferences,
following the developer instruction; syntax/source review and existing native
installer coverage apply. Actual Explorer registry/UI behavior remains untested
until exercised on Windows; do not claim otherwise.

Source is frozen after final tests/commit for scoped independent re-review of
the four findings, PS5 harness fix and new Explorer function. No real credentials,
private endpoints, inference or operator configuration mutation occurred.

=== DONE ===
