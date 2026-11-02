# TUI screen flows and visual direction

Planning baseline: 2026-11-02. These are annotated wireframes, not screenshots or implemented views. [Production contracts and acceptance](../plans/tui-production-plan.md) govern behavior; [stack evidence](../research/tui-stack.md) governs framework claims.

## Direction: a calm operator console

Build a compact instrument panel with a clear page title, one main decision and a quiet status line. Use aligned text, spacing and short rules rather than nested ornamental boxes. Show “what works / what changed / what to do” before diagnostics. No splash art, gradients, emoji status, giant logo or assumed patched font. The product title is `oneline-ccc`; `ccc` remains a provisional command name.

The shell has four stable zones: title + current scope; section navigation; content with a single focused action; footer with context-specific keys and job count. At wide sizes, split a list from its detail; place the primary action at the bottom of the detail, never moving as status updates. One live announcement line changes without stealing focus. Badges are words: READY, UNVERIFIED, STALE, CHANGED, BLOCKED. “Configured” does not imply a successful inference request.

## Flow map and focus rules

First run: offer background monitor → choose/detect clients (review Install/Skip if missing) → gateway and API key entry → model/budget/tier → review plan → apply receipt → dashboard/Launch. Monitor acceptance records intent; installation is previewed/applied with the setup plan. Optional SSH is an independent later offer, with its own administrator review. Declining either optional offer leaves client setup available.

Rerun: inspect known supported settings → source-labelled summary → resolve differences or keep → review only changed values → apply/no-op receipt. Existing credentials appear as references/state, never as re-entered text. Invalid/unknown schema stays read-only with a useful reason. Back retains non-secret draft choices; leaving key entry drops its raw buffer and requires re-entry if it was not yet stored.

Dashboard → client detail / models and context / service / jobs / repair / SSH / settings. Utilities are a secondary Settings destination, not another first-run checklist. Each destination returns to the same selected row and scroll position.

- `Tab`/`Shift+Tab`: visible focusable controls in reading order. Arrow keys move lists, never submit. `Enter` opens/activates; `Space` toggles a checkbox. `Esc` closes the top layer or returns one level, preserving safe drafts.
- `/` filters the current list; `?` opens contextual help with all keys. `Ctrl+K` opens section/action search outside text entry; `F6` cycles shell/content/footer where the terminal supports it. Tab remains sufficient if function/control keys are intercepted.
- `q` exits only outside text entry and modal review; dirty/running work opens a leave/stay decision. `Ctrl+C` first requests cancellation/safe exit, displays its boundary and leaves rollback visible. Forced termination is handled by journal recovery on next launch.
- Focus appears as `>` plus emphasis/reverse video and an accessible text label; exactly one control has that marker. Persistent selection without focus uses labelled `Selected:` text, never another `>`. Enter activates only the focused control. Do not bind action to row selection alone. Lists use stable IDs so refresh never picks a different client under the cursor.
- Review/elevation screens initially focus **Back**, not Apply. Apply requires an explicit selected action/Enter; no automatic countdown. Significant authority changes have a local explanation and an action named for that change. Ordinary navigation adds no confirmation.
- Text fields consume typing literally; shortcuts are inactive there. Secrets are masked without reflecting their length, paste never triggers commands, and no key reveal/copy control exists. Public keys have fingerprints, not secret treatment.
- Only one modal/drawer layer is allowed; screen title + breadcrumb identifies context. Alerts do not trap focus. Errors link to the failing field/stage, retain safe choices and state whether anything changed.

## Annotated wireframes

All examples use ASCII, synthetic aliases and exact illustrative budget values. Whitespace, `>` focus, `[ ]` choices and `[Action]` buttons are semantic. Wide-screen samples omit repeated rows; they fit the 100x30 content pattern, not a fixed pixel grid.

### 1. First run — offer the service, keep the first decision small

```text
oneline-ccc                                      Setup 1 of 5
Background monitor
Keep local client and service status current while this window is closed.
Runs at sign-in as your account. No updates, repair or model calls.

> (o) Continue without monitor     recommended starting choice
  ( ) Enable background monitor   can be removed in Service
Backend: Windows background task | current account | no administrator

  [Continue]                                       [? Help]
Tab move   Space choose   Enter continue   Esc exit
```

The service offer is part of first run, not hidden in settings. The selected backend description changes by OS; Windows task and OpenSSH service are different objects. Later setup steps reuse this shell. Choosing clients shows installed/version status; endpoint uses a local alias until explicitly edited; API key entry shows “Provided for this session”. Model choice permits an exact custom ID when discovery is absent.

### 2. Rerun — show what was found and what changed

```text
oneline-ccc / Setup                              Existing settings
Welcome back                       Inspected just now | no writes
Field              Current choice          Source
Gateway            Gateway A               native + saved agree
Model              exact-model-id          native client
Budget             353000                  native; saved 500000
Key                Stored                  credential reference
> Budget differs   [Keep current 353000] [Review saved choice]

[Review changes: 0]  [Open dashboard]
Enter inspect difference   Tab move   Esc back
```

The focused difference explains consequences; there is no silent “reset to defaults”. Reusing stored credentials does not open a value field. A no-change result reads “Already configured; no files changed”, with an inspection time and launch action.

### 3. Dashboard — current status before management controls

```text
oneline-ccc                                      This account
Dashboard  Clients  Models  Service  Jobs  Repair  SSH  Settings
2 clients configured     Gateway A: not checked     1 needs attention

CLIENT        CONFIG         MODEL            ACTION
> Codex       CHANGED        exact-model-id    Review external change
  Claude Code UNVERIFIED     another-id        Inspect
  Pi          Not installed  --                Review install / support

Monitor: Running (user task)   Observed 12 s ago   Jobs: 0 active
Enter client   / filter   Tab sections   ? help   q quit
```

No metric implies inference health: gateway inventory, authentication and protocol checks have separate labels. The priority row is attention-needed but refresh never moves the user's selection. An empty installation offers “Choose a client”; offline mode keeps last-known entries with timestamps.

A missing client with a verified install capability offers `[Review installation]` and `[Skip]`; otherwise show the exact unsupported reason. Installation review names trusted source/version, prerequisites, install owner/destination and any administrator request, then explicit `[Install]`. Success verifies the executable and continues configuration/Launch; decline/failure retains existing state and shows a receipt, including partial prerequisite effects.

### 4. Client detail — observation, desired change, safe action

```text
oneline-ccc / Clients / Codex                      CHANGED
Installed: version verified by adapter    Config: supported schema
Gateway: Gateway A    Key: Stored         Launch: managed launcher

                         Observed          Desired
Model                    exact-model-id    exact-model-id
Context budget           353000            500000
Tier                     Default           Default
Other configuration      Preserved         No planned changes

> [Review plan]  [Launch]  [Back]              Native check: not run
Tab move   Enter action   Esc clients
```

Review shows owned fields, backups, privileges, validation and conflicts before Apply. Endpoint reveal/edit is a deliberate local-only action; exported diagnostics use the alias. Unsupported versions show “Read only: adapter supports …” and disable Apply with an explanation reachable by keyboard. Hermes customizations require a separate preservation/import preview.

Launch checks the selected executable/config and vault/session reference, then suspends this TUI so the interactive client owns the terminal. A locked/missing credential offers explicit session API key entry or Back before starting any child. Client exit/interrupt returns to the same detail/selection with an exit receipt and restored terminal; launching does not imply an inference/authentication check has passed. T4a pins synthetic credential delivery and native PTY handoff/return.

### 5. Models and context — separate facts from chosen budgets

```text
oneline-ccc / Models and context                 Gateway A
Inventory: refreshed now    Metadata: cached, 2 h old
Selected: exact-model-id    Advertised | protocol unverified
Canonical mapping: not confirmed        Gateway capacity: unknown

Budget: [272000] [353000] [500000] [Custom]
Shared with: [x] Codex  [ ] Pi           Revalidated on each switch
Tier: Default  (Fast / Ultrafast need exact model + route evidence)
500000 is a chosen budget, not verified model capacity.
> [Save draft]  [Record capacity evidence]  [Refresh inventory]
Tab move   Enter action   Esc back
```

For verified mappings, replace unknowns with source, revision/time, input/total/output distinctions and limiting linked client. A budget above a confirmed limit shows a field error and disables Apply. Unknown capacity requires explicit per-model attestation to apply; stale metadata never silently expands a budget. A custom model ID remains exact. No fuzzy match imports another model's limits or tiers.

### 6. Service — one owned lifecycle, truthful platform wording

```text
oneline-ccc / Service                            Background monitor
Backend: Windows Task Scheduler    Scope: current user at sign-in
State: Running                     Observed 12 s ago
Purpose: local status only         Permission: no elevation
Client writes: Off   Automatic updates: Off   Automatic repair: Off

> [Stop]  [Restart]  [Inspect installation]  [Uninstall]
Unavailable after sign-out; foreground inspection still works.
OpenSSH has its own elevated service page under SSH.
Tab move   Enter action   Esc back
```

Not installed: show offer and backend; no duplicate install. Starting/stopping shows requested versus observed state and a timeout. Failed status includes native error plus “Retry inspection”; no success until the backend confirms. Uninstall explains that client configuration and credentials remain managed separately. Non-systemd Linux shows a foreground alternative, not a fake service.

### 7. Jobs — evidence of progress and safe cancellation

```text
oneline-ccc / Jobs                               1 active
Selected: Configure Codex          Applying: 2 of 4 stages
  Inspect    done      Validate   done
  Replace    running   Verify     pending
Last event: waiting for configuration file lock (3 s)
No files replaced yet. Cancellation is safe at this stage.

> [Request cancel]  [View redacted events]  [Back to dashboard]
Ctrl+C request safe exit   Enter action   Esc back
```

Use a determinate bar only when stages/count are known; otherwise show a static “Working” label and elapsed time, with optional restrained animation. After commit begins, say “Cancellation requested; finishing safe rollback” rather than “Cancelled”. Receipts distinguish Succeeded / Rolled back / Conflict / Unknown. Interrupted work opens recovery before another mutation, with the exact pending stage and reviewed recovery choices.

### 8. Repair — operator scope and prompt before execution

```text
oneline-ccc / Repair                             Manual operation
Target: Codex                     Working agent: select verified client
Evidence: redacted diagnostics    Workspace: disposable copy
Production writes: none           Network egress: selected agent provider

> [Preview exact prompt and allowed files]
  [Export prompt only]            [Run in verified sandbox]
Run unavailable until confinement is verified for this agent.
Returned changes require review, native validation and ordinary Apply.
Tab move   Enter action   Esc back
```

No automatic repair toggle is enabled. Missing working agent offers export and diagnostics; no credential request is forwarded. Prompt preview omits real endpoint/config/key values, even if the agent asks for them. The result view pairs a patch/allowlist violation with accept/reject reasons; Apply never means “execute whatever the agent printed”.

### 9. SSH — access scope and account/key before elevation

```text
oneline-ccc / SSH                                Windows OpenSSH
Current network: Public   WARNING: classification will not change
Firewall: [x] Private  [ ] Public (explicit opt-in)  Source: LocalSubnet
Private-only access will not work on this Public connection.
User: [Choose existing standard user]  [Create standard user]
Public key: [Paste]  [.pub file]  [GitHub username]
Fingerprint: shown after validation; import requires local acceptance

> [Review administrator changes]  [Back]
Client setup is complete; declining SSH leaves it usable.
```

Review lists effective authentication policy, key ownership/ACLs, service changes and firewall rule/scope. Any existing broad/ambiguous rule blocks enablement with recovery guidance. SSH server status and “remote login verified” are separate. Public profile does not broaden LocalSubnet; wider CIDRs/tunnels are labelled future scope. Key import is a one-time snapshot, not continuous GitHub trust.

### 10. Settings — explicit defaults, sources and reversible ownership

```text
oneline-ccc / Settings                           Local account
Appearance        Auto theme / ASCII fallback / Reduced motion
Credentials       OS vault; session only when unavailable
Updates           Notify only; managed installs keep their owner
Diagnostics       Local redacted receipts; preview before export
Background        Local observation only; off unless chosen
> Utilities       herdr | tmux | T3 Code | Paseo | orca (unresolved)
Config sources    Inspect native / saved / action precedence

[Review setting changes]  [Back]
Tab move   Enter edit   Esc back
```

Utilities show repository identity, platform, dependencies, privileges, install owner and removal behavior before install. A supported project's existence is not proof it is the operator's intended “t3” or “orca”. Orca has no installer until resolved. No remote desktop/tunnel capability is enabled incidentally by utility selection.

## Responsive layouts and fallback

| Terminal size | Layout and invariant |
| --- | --- |
| 120x40 and larger | 20-cell section rail, list/detail split (about 34 / remaining cells); max content width 116; whitespace outside it, no stretching for decoration |
| 100x30 | Compact top navigation and one principal table/detail area; titles/action/footer always visible; secondary evidence in a drawer |
| 80x24 | One pane; sections in a labelled selector; list row expands to detail on Enter; primary action visible, details vertically scroll |
| 60x20 | Stacked label/value rows; one compact title, one status line, scrollable content, two-line key footer; no mandatory horizontal scroll |
| 40x12 | Narrow mode: one field/decision per step; “Item n of m”; compact Back/Next; full values in vertically wrapped detail |
| Below 40 columns or 12 rows | Inline/plain guided mode, no fullscreen cursor tricks; one prompt/action at a time, all decisions retained; `status --json` remains available |

Narrow first-run example (40x12 budget, 11 content lines):

```text
oneline-ccc / Setup 1 of 5
Background monitor
Optional. Runs at sign-in.
No repair or model calls.

> (o) Continue without
  ( ) Enable monitor

[Continue]
Tab move  Space choose
Enter next  Esc exit
```

Measure rendered cell width with the framework's Unicode-aware facilities; never byte length. Truncate a table value only with an explicit detail path; show its full value wrapped before a consequential confirmation. Long paths/IDs, combining marks, CJK and right-to-left text need fixtures. Sanitize control/escape sequences from remote/native output; do not interpret OSC links, terminal title changes or clipboard controls. Resize preserves selected ID, draft and scroll anchor; fast status updates cannot reflow the active field. Apply stays unavailable if the whole authority review cannot be read, even in narrow mode.

## Tokens, theme and accessibility

| Token | Dark starting value | Light starting value | Non-color meaning |
| --- | --- | --- | --- |
| surface / text | `#111820` / `#E6EAF0` | `#F5F6F8` / `#17212B` | paired explicit themes; auto respects terminal |
| muted | `#B3BDC9` | `#465463` | secondary label, never the sole instruction |
| accent / focus | `#91C7FF` | `#164F91` | `>` and underline/reverse; current action |
| success | `#91D6A5` | `#23613A` | READY / DONE text |
| caution | `#F0CC81` | `#704700` | WARNING / STALE text |
| danger | `#F5A3A3` | `#9B2727` | BLOCKED / FAILED + corrective action |

Values are starting tokens, not a contrast certification on arbitrary terminal backgrounds. Set an explicit paired surface only in an explicit light/dark theme; auto mode respects the user's terminal. Target at least 4.5:1 text contrast on the tested paired palettes; test selected/error states too. No status depends on red/green distinction. Spacing tokens are 1 cell inline, 2 cells between groups, 1 blank line between sections; use a single optional rule below page title, no box around every value.

Truecolor → 256 color → 16 ANSI → monochrome must retain labels, focus marker, order and emphasis. Honor `NO_COLOR`, dumb/non-TTY output and explicit `--color=never`; do not send terminal capability probes that can hang a pipe. Default ASCII symbols work with ordinary fonts; optional Unicode rules/arrows have exact `-`, `>`, `...` replacements. No icons, ligatures, Nerd Fonts, terminal font installation or width assumptions.

Reduced motion disables spinners, blinking and animated transitions; elapsed/stage text remains. Mouse is an optional convenience, never the only path. Provide a linear `--plain` guided mode for screen-reader users; fullscreen accessibility is not assumed. Test keyboard order, focus retention, terminal restoration and high-contrast/no-color modes with native PTYs, and obtain an actual screen-reader pass before claiming accessibility support.

Visual exit gate: inspect recordings/captures for every requested screen plus empty/loading/error/offline states at 120x40, 80x24 and 40x12; mechanical clipping/focus checks cover every size in the table. Pin `TestET1SingleFocusMarker`: at most one `>` focus marker per view, persistent `Selected:` state distinct, and Enter activates only that focused control through Models/Jobs Tab transitions. Record Windows Terminal, macOS Terminal, Linux terminal and SSH/tmux trials. Assess hierarchy/readability with a separate review pass; wireframes and palette choices alone are not production visual evidence.
