# Windows Codex / LLMP MVP

## Scope and authorization

Build the urgently requested Windows installer now. Defer the broader Go binary,
TUI, service, automatic maintenance, multi-client setup and agent-driven repair to
research and a subsequent design discussion. Do not use real API credentials or
modify the operator's Codex configuration during development.

## Acceptance criteria

- A PowerShell one-liner downloads a standalone installer and runs an interactive
  setup on Windows; a local-file invocation also works.
- Setup accepts a proxy base URL, a privately entered API key and an exact model ID.
- Install native Codex when necessary or give actionable dependency instructions.
- Write valid Codex configuration using the Responses API and environment-based
  credentials, preserving unrelated settings and backing up configuration changes.
- Make reruns predictable and support an isolated configuration home for testing.
- Let the user choose a context budget and an earlier compaction threshold. Never
  claim a context cap expands the model's actual capacity.
- Represent eligible service tiers using actual supported Codex configuration and
  capability metadata; do not invent models or tier eligibility.
- Verify PowerShell parsing, generated TOML, reruns, preservation, validation,
  error handling and real Codex configuration loading without paid inference.
- Independently review and fix material findings before reporting completion.
- Document hosting and use honestly: a public one-liner needs an actual hosted URL.
- Offer optional elevated Windows OpenSSH installation/service/user/public-key
  setup. Use LocalSubnet sources and Private profile by default; make Public an
  explicit choice, warn on active Public networks, and preserve classification.
- Accept pasted/file public keys or explicitly approved GitHub public-key import.
  Refuse conflicting existing policies; restore authorization and service changes
  on failure. Verify a real SSH key login and rollback on a disposable Windows runner.

## Work lanes

1. Research official Codex configuration, catalog and tier capabilities.
2. Implement the standalone Windows PowerShell installer.
3. Research existing multi-client/proxy configuration projects for the later design.
4. Integrate documentation and an executable isolated test harness.
5. Independent review, fixes and final validation.

## Design decisions to verify

Use a dedicated `$CODEX_HOME/llmp.config.toml` profile file for Codex 0.134.0 or
later, with root-level context and compaction fields. Current official docs say
legacy `[profiles.llmp]` tables and the root `profile` selector are no longer read.
Update the main `config.toml` so bare `codex` uses LLMP, preserving unrelated
settings via the native Codex configuration API in staging. Maintain the named
profile as an optional explicit launch. Back up changed files and roll back both
configuration files and credentials on a failed update. Store the API
key in the Windows user environment and process environment; keep it out of TOML.
Use explicit model/context choices when proxy metadata is insufficient. Test with
dummy credentials and local fixtures only. The optional SSH server is explicitly offered; no client-maintenance service is installed.

## Status

Installer, executable test harness, independent review, official documentation
research, existing-project comparison and static setup page are implemented.
Portable PowerShell tests and actual Codex strict parsing/profile loading pass.
Native Windows verification and public download checks are publication gates;
their final results belong in the release verification report.

## Workflow continuation (2026-11-02)

The operator requested `ultra-code-workflows` during MVP finalization. Continue
with its SDD controller loop: exact file ownership, on-disk briefs and reports,
independent task review, fresh whole-source final review, tested commits, Windows
publication gates, and an executable try-it command. The in-flight checkout is
retained to avoid disrupting active workers; subsequent app work uses isolated
feature worktrees. No Workflow tool is available, so native subagent tools and
file artifacts provide the workflow. Do not invent RED evidence for completed
historical work. New catalog tests were run against the preceding checkpoint
and fail because its catalog helpers are absent.

Task 1 finishes automatic complete, version-matched Sol tier metadata and safe
native staging cleanup. Task 2 publishes the frozen reviewed result, waits for
Windows PowerShell 5.1/7 and real SSH checks, verifies public script bytes, and
notifies the operator with `attn`. Broader TUI work remains a subsequent design
phase. Finalization ledger and review copies will be committed alongside this
plan after verification.
