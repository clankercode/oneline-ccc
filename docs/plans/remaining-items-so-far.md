# Remaining items so far

Recorded: **2026-11-03** (Australia/Sydney).
Implementation baseline: **`40d02d4`**.

The Windows MVP and first native Codex workflow are complete. This records all
17 remaining areas identified against the full project brief, roughly in priority
order. All items are open at this baseline. Update their status as implementation
and verification finish, with links to the resulting evidence.

Some areas extend working features; others require new implementation. Automatic
maintenance and automatic repair retain the policy decisions and explicit opt-in
described below. Use the current native usage, verification and delivery ledger
for shipped capabilities; older plans retain their original planning baselines.

## Foundational and supporting documents

| Document | Purpose |
| --- | --- |
| [TUI production plan](tui-production-plan.md) | Full product scope, architecture, requirement map, phased tasks T0–T10 and exit tests |
| [Project brief](../product-brief.md) | Original MVP and broader product intent |
| [Broader utility brief](broader-utility-brief.md) | Retained multi-client, maintenance, context, repair and SSH requirements |
| [Native foundation plan](native-foundation.md) | Scope of the first implemented native slice |
| [Native usage](../native-app.md) | Current commands, workflows and implementation limits |
| [Native verification](../verification/native-foundation.md) | Actual configuration/terminal evidence and remaining production gates |
| [Native delivery ledger](../ledgers/native-foundation-ledger.md) | Delivered work, decisions, reviews and final gates |
| [Native acceptance review](../ledgers/native-foundation-acceptance-rereview.md) | Final process/terminal corrections and the disclosed dependency race |
| [Client compatibility contract](../compatibility/clients.md) | Client identities, supported Codex contract and adapter expansion gates |
| [Platform and distribution contract](../compatibility/platforms.md) | OS/architecture support, native proof and release/bootstrap trust |
| [TUI usability contract](../design/tui-usability.md) | Selection, management, drafts, review, tiers, help and command search |
| [TUI screen flows](../design/tui-screen-flows.md) | First-run, rerun, model, service, jobs, repair and SSH journeys |
| [TUI visual system](../design/tui-visual-system.md) | Approved color, hierarchy, focus and non-color cues |
| [Stack and identity research](../research/tui-stack.md) | Framework choices, model metadata and client/utility identity evidence |
| [Existing-project research](../research/existing-projects.md) | Prior art and reuse options to consult before duplicating functionality |
| [Stage-zero verification](../verification/tui-stage-zero.md) | Native Codex contract and isolated existing-tool adoption trials |
| [Codex configuration research](../research/codex-configuration.md) | Configuration, model catalog and service-tier findings |
| [Windows MVP plan](windows-codex-mvp.md) and [verification](../verification/windows-mvp.md) | Existing one-line Windows setup and its tested behavior |
| [Windows SSH documentation](../windows-ssh.md) | Existing SSH helper, authentication, firewall and recovery behavior |
| [Windows Explorer defaults plan](windows-explorer-defaults.md) | File-extension and hidden-file preferences already implemented in the installer |
| [Script rerun/defaults plan](script-rerun-defaults.md) | Existing-setting prefill and idempotent installer behavior |
| [Configuration transaction contract](../../internal/configtxn/README.md) | Protected writes, drift, backup and recovery rules |
| [Observer contract](../../internal/monitor/README.md) | Current read-only observation and native lifecycle boundaries |
| [Test documentation](../../tests/README.md) | Existing verification harnesses |
| [Plan index](README.md) | Entry point for the repository's plans and status |

## Remaining items

1. [ ] **Fix the known terminal shutdown race.** Resolve the Bubble Tea/cancelreader
   race and run the rendered terminal shutdown regression under the race detector.

2. [ ] **Finish native platform acceptance.** Test interactive input, resizing,
   interruption, client launch/return and terminal restoration on real Windows
   and macOS terminals. Establish minimum OS versions and runtime coverage for
   the advertised architectures.

3. [ ] **Ship the native app through one-line installers.** Add Windows, macOS and
   Linux commands that download the appropriate binary, install it and launch it
   without requiring Go. Add versioned releases, authenticated artifact
   verification, upgrade/rollback/uninstall, dependency inspection and clear
   signing status.

4. [ ] **Install and update coding clients.** The native app currently requires
   Codex to be installed already. Add prerequisite handling, installation
   previews, executable verification and updates that respect existing
   package-manager ownership. Expand Codex compatibility beyond **0.166.0**, with
   checks before accepting newer versions.

5. [ ] **Implement the other five client adapters.** Claude Code, Pi Agent,
   OpenCode, Grok Build and Hermes currently have detection only. Each needs
   verified configuration/authentication contracts, configuration and launch
   support, preservation of existing settings, rerun behavior and rollback tests.
   Hermes needs particular care around customizations.

6. [ ] **Complete persistent credentials across platforms.** Windows Credential
   Manager works; bounded macOS Keychain and Linux Secret Service implementations
   remain. Complete the experience for launching clients directly, such as bare
   `codex`, when the selected key lives in a vault rather than an existing
   environment variable.

7. [ ] **Finish model metadata presentation and caching.** Gateway discovery and
   models.dev enrichment already exist. Still needed: visible capacity/provenance
   information, persistent inventory and metadata caches, freshness indicators,
   offline fallback, explicit refresh failures and deliberate mappings for proxy
   aliases.

8. [ ] **Add per-model and shared context budgets.** Today there is one active
   budget per client profile. Add remembered budgets per model, links across
   clients/models, reserve and compaction constraints, and revalidation when
   switching. Preserve the 272000/353000/500000 presets.

9. [ ] **Generalize service-tier support.** Replace the current combined
   Fast/Ultrafast confirmation with individual capabilities for each model and
   route. Support Fast-only routes and models beyond the current Sol/Astra
   preset. Switching to a model without the selected tier should reset the
   default to Standard with an explanation.

10. [ ] **Complete the smooth first-run experience.** Connect missing-client
    Install/Skip through configuration and launch. Allow discovery against the
    endpoint currently being edited, before its first Apply. Remember
    theme/display preferences and the user's decision to decline the monitor.
    Help, command search and ordinary draft navigation already work.

11. [ ] **Expand jobs, history and diagnostics.** Add phase-level progress,
    timestamps and durable records for failures, cancellations, conflicts and
    interrupted work, alongside successful Apply receipts. Connect restart
    recovery to visible job outcomes and provide previewable support bundles.
    Cancellation and transaction recovery already exist.

12. [ ] **Complete the optional background monitor.** Implement the Windows
    background-task backend. Validate actual Linux systemd-user and macOS
    LaunchAgent lifecycles, including login/logout, sleep/resume and unavailable
    backends. Extend observation as additional clients become supported.

13. [ ] **Settle and implement maintenance policy.** Decide what checks for client
    updates, gateway model changes and configuration drift; how often; and which
    changes need review. The current observer only records local configuration
    state. Automatic maintenance remains a design choice requiring explicit
    opt-in.

14. [ ] **Implement agent-assisted repair.** Extend the existing redacted
    diagnostic brief into: select a working agent → preview its inputs → run in
    an isolated workspace → inspect the proposed patch → validate and apply.
    Automatic repair remains a later optional feature.

15. [ ] **Finish native Windows integration.** SSH currently hands off to the
    working PowerShell helper. Add fuller native status, review, elevation and
    result handling; later support explicit additional subnet/Tailscale/WireGuard
    scopes while retaining Private + LocalSubnet defaults. Bring the
    file-extension and hidden-file preferences into the native app; the Windows
    installer already applies them.

16. [ ] **Implement utility management.** Add verified installation, status,
    update and removal paths for herdr, tmux, T3 and Paseo, respecting their
    platform limitations. Identify the intended Orca project before implementing
    its installer.

17. [ ] **Complete release acceptance and documentation.** Exercise fresh install,
    rerun, upgrade, interruption, recovery and uninstall on supported machines,
    plus an explicitly requested live gateway session. Refresh older
    planning/compatibility documents and publish an accurate
    client/version/platform support matrix.
