# Go TUI production plan

Planning baseline: 2026-11-02. The proposal below retains that baseline; current
implementation and evidence are recorded in the [native foundation](native-foundation.md)
and its [delivery ledger](../ledgers/native-foundation-ledger.md).
Authority: [broader brief](broader-utility-brief.md), [product brief](../product-brief.md), [existing research](../research/existing-projects.md), [Windows evidence](../verification/windows-mvp.md).
Companions: [screen flows](../design/tui-screen-flows.md), [stack evidence and IGC decisions](../research/tui-stack.md).

Visual follow-up: [navigation and color pass](../design/tui-visual-system.md).
Carry its hierarchy, selection and non-color cues into T1/T4; its browser
specimen is a design aid, not evidence of a running Go TUI.
Interaction follow-up: [usability contract](../design/tui-usability.md) connects
management choices, model settings, review, simulated receipts and keyboard
help/search. Carry these paths into T1/T4; browser tests do not satisfy native gates.

**Acceptance correction, 2026-11-03:** the operator rejected the native TUI's
visual fidelity and cumbersome, inconsistent controls, including the awkward
movement from tabs into menus via Down. Native visual and interaction acceptance is
reopened. The [P0 remediation backlog](remaining-items-so-far.md#p0-native-tui-visual-quality-and-interaction-remediation)
requires actual native/mockup comparisons for every screen and overlay,
consistent navigation, alternating `visual-review-and-fix` passes and
independent subagent reviews under `ultra-code-workflows`. Functional native
checks and approval of the browser specimen do not close this gate.

Production follow-up from native acceptance: resolve the pinned Bubble Tea/
cancelreader reader-close race in rendered SIGTERM shutdown and prove native
Windows/macOS interactive terminal behavior before wider console acceptance.
The development slice's Linux rendered-PTY and separate headless race gates are
documented in [native verification](../verification/native-foundation.md).

## Goal, scope and evidence labels

Deliver one-line Windows/macOS/Linux setup of a self-contained Go utility that makes existing coding clients usable with a custom gateway, preserves their settings, and explains current status and recoverable failures. Offer an optional background monitor during first run; ordinary use must work without it.

- **Verified:** this repository currently delivers a Windows PowerShell Codex/SSH workflow. Its verification document records native Windows evidence and specific limits; those are historical claims, not tests rerun for this plan. No Go module exists in the inspected tree.
- **Verified upstream:** facts marked V in the stack document were checked against public primary sources on the baseline date; source review does not establish runtime compatibility.
- **Decision:** choices below are proposed implementation defaults within the retained requirements, not implemented behavior.
- **Assumption:** one interactive operator per OS account, multiple concurrent invocations possible; local client configuration, not fleet management. Target six OS/architecture pairs: Windows/macOS/Linux, amd64/arm64. Minimum OS versions remain a release gate.
- **Unknown:** gateway protocol/capacity/tier contracts, exact installed client schemas, customized Hermes behavior, and signing resources require evidence before the affected build slice ships.

Non-goals: a new inference proxy, replacement coding agent, conversation transfer/history editing, rules synchronization engine, remote management server, telemetry, automatic model calls, unattended repair or unattended updates. Evaluate native profiles/CC Switch first and an existing gateway only if translation or pooling is required; use the existing research, without repeating its proxy survey.

## User journeys and requirement map

| ID / journey | Observable outcome | Build tasks / exit tests |
| --- | --- | --- |
| R1 First useful launch | One-line bootstrap opens a native TUI; optional monitor is offered; missing selected client can be installed and launched after configuration | T0,T1,T3a,T4,T4a,T7,T10 / ET1,ET2,ET4,ET6,ET10 |
| R2 Return and reconcile | Rerun prefills discovered/current choices; zero-change apply is a no-op; external changes are explicit | T2,T3,T4 / ET2,ET3 |
| R3 Choose clients | Codex, Claude Code, Pi, OpenCode, Grok Build, Hermes each progress through verified configuration/install capabilities; only completed capabilities offer Apply/Install | T0,T3,T3a and expansion queue,T4,T4a / ET4 |
| R4 Connect a gateway | Exact custom endpoint/key/model ID; gateway inventory and protocol evidence stay distinct | T2,T3,T5 / ET2,ET4,ET5 |
| R5 Share budgets and tiers | 272000/353000/500000 presets are budgets; per-model capacity and tier evidence remain visible | T5 / ET5 |
| R6 Understand/manage service | Current client, monitor and SSH status have timestamps; install/start/stop/restart/uninstall show actual backend | T4,T7 / ET6 |
| R7 Follow work | Jobs show stages, cancellation boundaries, receipts and interrupted/unknown outcomes | T2,T6 / ET3,ET7 |
| R8 Repair deliberately | Operator invokes a working agent with a bounded, redacted brief, then reviews any resulting patch | T8 / ET8 |
| R9 Enable Windows SSH | User/key choice, GitHub import/paste, Private + LocalSubnet default; explicit Public choice; no network reclassification | T9 / ET9 |
| R10 Add useful utilities | herdr/tmux/T3/Paseo have verified identities before install; ambiguous orca remains disabled with explanation | T0,T10 / ET4,ET10 |
| R11 Keep control and secrets | OS-backed storage where available; exact backup/rollback; foreign settings and external installs retain their owners | T2,T3,T6,T8,T10 / ET2,ET3,ET8,ET10 |
| R12 Read and operate anywhere | Keyboard-complete, coherent TUI at concrete sizes, ASCII/no-color/plain mode, offline recovery | T1,T4,T6 / ET1,ET7 |

## Domain, adapters and ownership

Proposed package/file names and contracts below are application design, not claims about third-party APIs. `ccc` is a provisional executable name; freeze it at T0 before publishing commands.

| Boundary / proposed files | Owns / contract | Must not own |
| --- | --- | --- |
| `internal/domain/{profile,model,operation}.go` | Profile, ClientRef, CredentialRef, ModelEvidence, BudgetPolicy, ChangePlan, OperationReceipt; typed statuses and errors | Terminal, filesystem, network or OS calls |
| `internal/app/{inspect,plan,apply}.go` | Inspect → Plan → Validate → Apply → Verify; capability gating and operation authorization | Client syntax or service-manager command construction |
| `internal/adapters/client/{adapter,codex}.go` | Version-bounded client detection and lossless config edits; declare owned fields and paths | Broad home-directory scanning, rewriting unrelated settings or installing arbitrary plugins |
| `internal/configtxn/{transaction,recovery}.go` | Per-operation journal, protected backups, stable locks, drift checks, crash recovery | Inference, UI state or optimistic whole-file overwrite |
| `internal/credentials/{store,launch}.go` | OS vault/session references, secret entry and minimal child environment | Secret-bearing events, command arguments or persistent global environment changes by default |
| `internal/install/{client,manifest}.go` | Trusted client install plans, prerequisite/package-manager delegation and executable verification | Replacing existing installs silently, changing ownership or receiving gateway credentials |
| `internal/catalog/{gateway,metadata,policy}.go` | Exact inventory IDs, explicit identity mapping, capacity/tier provenance and stale cache | Entitlement claims from names or cached metadata |
| `internal/jobs/{runner,store}.go` | Bounded foreground jobs, receipts, cancellation and redacted event stream | Blind retries of uncertain mutations |
| `internal/monitor/{observe,lifecycle}.go` | Read-only local observation; native install/start/stop/status adapters | Client writes, elevated actions, inference or repair at startup |
| `internal/repair/{brief,review}.go`, `internal/ssh/{plan,windows}.go` | Explicit repair workspace/review and typed elevated SSH plan | Reusing a general shell as an unrestricted privilege broker |
| `internal/tui/{app,navigation,theme}.go` | View state, focus and rendering; invokes application commands | Credential storage, direct config writes or shell execution from widgets |

Client adapter contract: `Detect(ctx, Root) -> ClientSnapshot`; `Plan(ctx, ClientSnapshot, Profile) -> ChangePlan`; `Validate(ctx, StagedChange) -> ValidationResult`; `Verify(ctx, OperationReceipt) -> ObservedState`. Root identifies an approved config root; StagedChange identifies protected staged outputs for a ChangePlan. Every result carries schema/version support, paths considered and typed refusal reasons. T3 implements Codex only; later client slices consume this same contract rather than designing all adapters now.

`ChangePlan` contains operation ID, target principal, source revision hashes, owned fields, proposed changes, credential references, preconditions, required privileges and rollback steps. `Apply(ctx, planID, expectedRevision) -> OperationReceipt` reloads the stored plan and rechecks preconditions under lock. IDs are generated by this application only for its own records; never invent a remote model/client ID.

Launch contract: `Launch(ctx, LaunchSpec) -> LaunchReceipt` is an application operation. LaunchSpec identifies ClientRef, the verified absolute executable/identity and literal argument array, selected config root/profile, approved working directory, expected config revision and CredentialRef or ephemeral session handle. Recheck executable/config identity before spawn; derive argument/profile conventions from the pinned client contract, never shell-interpolate user input. LaunchReceipt contains child exit/cancellation/terminal-restoration outcomes only, with no environment values. The credential launcher resolves the key just before spawn into an allowlisted platform/client child environment; reject missing, locked or expired references without starting a child or persisting a fallback. A locked vault can be bypassed only by explicit session entry; that handle is memory-only, scoped to the foreground session and discarded on session end. Release launch-owned secret buffers after use; child environment remains sensitive until child exit.

`ModelEvidence` separates exact gateway ID, optional canonical identity, metadata source/revision/time, advertised availability, protocol evidence, reported capacity and tier evidence. `BudgetPolicy` carries selected budget, reserve policy, confirmation source and linked clients. `OperationReceipt` records phases, changed paths/hashes, backup references and verified/rolled-back/conflict/unknown outcome, with no config content or secrets.

Dependencies point TUI/CLI → application → domain and ports; adapters implement ports. No adapter imports TUI; no service manager owns a client profile. `Profile` owns desired non-secret settings; each client remains owner of its native file. The transaction layer owns only the fields explicitly accepted in its plan.

## Lifecycle and durable state

- Foreground TUI/CLI owns all mutations and credential access. One mutation job per account runs at a time, using a stable account-level coordinator lock; also lock affected stable parent directories, including first writes where a destination is absent. Interoperating app invocations honor locks; foreign editors require final drift checks and explicit conflict handling.
- The optional daemon is an observer. Linux defaults to a systemd **user** unit, macOS to a LaunchAgent, Windows to a current-user Task Scheduler job at logon, without stored login password. The UI names the Windows backend “Background task”; it does not claim it is an SCM service. No login-session keyring access is required by this observer. Boot-without-login support is deferred.
- Windows OpenSSH is a separate SCM service with separate elevation and rollback. Absence of systemd, a locked user session or a denied Task Scheduler policy means “Unavailable”; foreground use continues. Never enable lingering or install a system service silently.
- Foreground owns versioned non-secret `settings.json`, operation journals and receipts in OS user config/state directories. The observer owns a separate timestamped observation snapshot. POSIX permissions are owner-only; Windows ACLs name the intended account. Network/shared/cloud-synced filesystems are unsupported for writable state until locking/atomicity are established.
- Plan/apply lifecycle: Draft → Validated → Applying → Verifying → Succeeded. Alternative terminal outcomes: Cancelled, RolledBack, Conflict, NeedsAttention, Unknown. “Process exited 0” alone never establishes success.
- Each transaction journals intent before side effects, takes exact protected backups, validates staged output with the selected native client, rechecks original hashes, then replaces each target using platform-safe primitives. Multiple files/credential changes form a recoverable transaction; they are not simultaneously atomic. Recover after each interruption point, in reverse order, restoring only app-owned changes whose current hashes still match its outputs.
- An external edit after apply prevents blind rollback: preserve both states, report Conflict and offer a reviewed merge. Journal recovery runs before a new mutation; ambiguous outcomes require reconciliation. Windows sharing violations get bounded retries then refusal; no fallback to truncate-in-place. POSIX directory durability and Windows replacement/ACL behavior need native proof.
- Advisory locks cannot exclude a foreign editor that ignores them; a hash check and rename are not a universal compare-and-swap. Ask users to close active client/config editors before Apply where exclusive ownership cannot be enforced, retain exact backups, and disclose this residual race rather than promise arbitrary multi-writer atomicity.
- Jobs belong to the foreground process in the first release. On exit, wait or request cancellation; active mutations finish their safe boundary/rollback before exit. Forced termination leaves a recoverable journal. Closing the TUI does not silently transfer mutation authority to the monitor.
- Observation IPC uses a user-owned Unix socket or Windows named pipe with peer-identity/ACL checks; no TCP listener. Version mismatch falls back to foreground inspection, never auto-replaces a running daemon. Same-user IPC is not a security boundary against arbitrary code running as that user.

## Credentials, defaults and drift

- Store API keys in Windows Credential Manager, macOS Keychain or Linux Secret Service, addressed by opaque CredentialRef. Locked/missing vault: explain and offer session-only entry; no plaintext fallback. Pin and audit the selected Go keyring implementation, including size limits and subprocess cleanup, before release.
- Default direct connection uses a launcher that injects the key only into the selected child environment; arbitrary bare-client launches cannot inherit a vault entry automatically. Where clients support native credential references, prefer them. Offer persistent client-supported storage/environment only as a separately described compatibility mode, including its protection and new-terminal behavior.
- Import the current Windows MVP environment-key **reference**, not its value into app settings. Do not delete/migrate persistent credentials without a specific reviewed operation. Display “Stored”, “Session only”, “Missing” or “Locked”; never reveal the key in a TUI view, diff, log, journal, repair prompt or clipboard action.
- Endpoint and config content can also be private. Render a local alias by default; explicit endpoint editing stays local, diagnostics replace it with an alias. Reject URL userinfo and query/fragment credentials. Require HTTPS except explicit loopback/local-development HTTP choice; no TLS verification bypass, redirect-following credential forwarding or automatic external hostname probes.
- Existing supported configuration prefills endpoint reference, exact model, context, tier, chosen clients and service choice. Precedence: explicit current action → saved per-client desired settings → detected native settings → safe defaults. Each field shows its source; disagreements are drift, not permission to overwrite. New/unknown native schemas are read-only.
- Defaults: service offered but off; updates notify only after an explicit check; repair manual; no inference or telemetry; network discovery only on explicit refresh. Gateway inventory requests are bounded (10 s request timeout, 5 MiB response cap, no redirects); metadata cache has a 24 h freshness target and remains visibly stale offline. These are reversible product defaults, not provider facts.
- Gateway `/models` supplies availability IDs, never context or protocol guarantees. models.dev enriches only exact verified mappings (including its explicit canonical ID where applicable); unknown aliases stay unknown. Auth status, route reachability and paid protocol verification are distinct states; model visibility alone is not “Ready”.
- Context presets 272000/353000/500000 are selectable user budgets. A linked shared budget must satisfy every linked client/model's confirmed applicable capacity and reserve constraints; never silently raise a smaller limit or silently reduce an existing choice. Unknown capacity can be saved as an unverified draft; apply requires explicit per-model operator attestation or gateway evidence. Switching models revalidates the link. A 90% compaction target is a proposed default only where the client supports it; input/total/output limits remain distinct.
- Model ID, request service tier and native catalog/display metadata are separate fields. Default tier is explicit default/unset according to adapter semantics; preserve a detected existing tier until the operator changes it. Offer Fast/Ultrafast only for the exact model/gateway/version with verified mapping or labelled operator attestation; no tier suffixes added to model IDs. Reuse the MVP's catalog semantics only after version-specific tests; no blanket extension of its presets.

## Update, repair and SSH authority

Update ownership follows installation ownership: app-managed portable binaries may be staged, verified and rolled back by an explicit Update action; package-manager installs delegate to that manager with a preview; unmanaged installs show instructions. Never silently change ownership, runtime prerequisites or PATH. Self-update occurs after active mutations settle, through a verified replacement helper, with previous executable retained. Observe unsupported new client versions as read-only until compatibility is established.

Repair is a capability-bounded, operator-invoked workflow: choose a working agent → collect an allowlisted redacted diagnostic bundle → preview exact prompt/files/egress → run in a scratch workspace → inspect proposed patch → validate and apply through the transaction layer. The agent never receives raw credential/config backups, SSH keys, production config roots or direct elevation. Use a disposable config root and no write tools outside it; if that cannot be enforced for the chosen agent, provide an export-only prompt and import a reviewed patch instead. Output is untrusted; reject symlinks, traversal, secret changes and paths outside the approved plan. Cancellation/timeout terminates owned children and records uncertain effects without retry. Sandbox strength and agent CLI contracts are implementation blockers for executable repair, not for the planning artifacts.

Windows SSH flow reuses the documented MVP behavior as a contract, not its source as presumed production-ready Go code. Inspect effective policy and broad/ambiguous firewall rules before enabling access; refuse unsafe ambiguity. Select existing local user or explicitly create a standard user; account passwords are privately entered only in the elevated flow. Accept pasted public keys, `.pub` files or a GitHub username; show fingerprints and require local acceptance of the fetched snapshot. Never fetch/import private keys, change network classification, or remove unrelated authorized keys.

Default firewall plan is Private + LocalSubnet. Warn if the active network is Public and explain that Private-only access will not work there; Public is a distinct explicit opt-in, still LocalSubnet. Broader CIDRs, sibling networks, Tailscale and WireGuard are separately gated later options. Preview local user/key paths, ACLs, service/firewall changes and administrator scope; the elevated helper accepts a strictly validated typed plan, rechecks principal/paths/state at execution, performs no arbitrary commands, then independently verifies effective authentication and restores app-owned changes on failure. SSH failure does not undo client setup; Public access and broader scopes are never defaults.

## Production failure contract

| Trigger | Required behavior / recovery | Owning task |
| --- | --- | --- |
| First-write race or concurrent rerun | Serialize app writers; recheck absent/existing file under stable lock; refuse stale plan | T2 |
| Disk full, read-only root, sharing lock, kill between writes | Preserve originals/journal; bounded retries; recover or display NeedsAttention | T2,T6 |
| Foreign edit, symlink/reparse-point swap, unknown schema | Stop before overwrite; retain foreign bytes/ACLs; show conflict/read-only reason | T2,T3 |
| Missing/locked vault, expired/revoked credential | Show actionable state, allow session entry; never leak or persist by fallback | T2,T5 |
| Gateway timeout, TLS failure, 401/403/429, malformed/oversize inventory | Distinct typed errors; stale inventory labelled; no mutation or inference retry | T5 |
| False capacity mapping or unsupported tier | Block unacknowledged apply; keep exact IDs, show evidence and per-client limit | T5 |
| No terminal, resize, lost SSH session, Ctrl+C | Plain output or stable layout; restore terminal; journal safe cancellation | T1,T6 |
| Monitor unavailable, version skew, sleep/resume | Timestamp stale status; bounded reconnect; foreground fallback | T7 |
| Updater signature failure or replacement restart failure | Retain previous binary/config; report refusal/rollback; no silent upgrade | T10 |
| Repair injects command/path or leaves child alive | Enforce workspace/policy, reject patch, reap owned process tree; show receipt | T8 |
| SSH UAC denied, invalid key, broad firewall or restart failure | Keep client setup; refuse/restore SSH changes; never claim reachability untested | T9 |

## Phased build tasks and gates

All paths below are **future proposed implementation ownership**, not permission to edit them in this planning run. T0 precedes implementation; each T1–T10 is a slice to refine into at most 1–5 source/test files per implementer, split further if needed. Preserve controller-owned scripts/tests until explicitly assigned. No client besides Codex receives detailed adapter design here.

### T0 — Verify the adoption and platform contract

Files: create future `docs/verification/tui-stage-zero.md`, `docs/compatibility/clients.md`, `docs/compatibility/platforms.md`.
Consume: this plan, existing research and Windows evidence. Produce: pinned Go/dependency versions, actual supported OS floors/architectures, client identity/version/schema matrix, release signer/bootstrap trust contract, and observed existing-tool adoption gaps. Run isolated native-profile/CC Switch trial before committing to custom configuration work; use fixtures only.
Exit: sources and disposable native trials establish the first Codex contract; Pi rename/package is checked against the selected installed release; each remaining client is verified or explicitly unavailable. Confirm herdr/T3/Paseo intended identities, resolve orca before any installer, and verify each utility's platform/license/install ownership. Unknowns block only their dependent slice.

### T1 — Establish the terminal shell and public command contract

Files: create future `cmd/ccc/main.go`, `internal/tui/{app,navigation,theme}.go`, `internal/tui/app_test.go`; add `go.mod`/`go.sum` in a separately scoped foundation change.
Produce proposed commands `ccc`, `ccc status --json`, `ccc plan`, `ccc apply <plan-id>`, `ccc doctor`, `ccc service`, `ccc repair`, `ccc ssh`, `ccc update`; commands besides TUI have explicit help/usage and return nonzero for refused/failed operations. Unsupported commands are not offered as working features.
Produce global terminal options `--plain`, `--color=auto|always|never`, `--ascii` and `--reduced-motion`; plain/redirected output never requires mouse, Unicode or fullscreen features. These are proposed app options, not library APIs.
ET1 `TestET1TerminalModes`: all ten screen skeletons and focus paths render at 120x40, 100x30, 80x24, 60x20, 40x12; no escape output to redirected stdout; resize/Ctrl+C restore terminal. PTY trial on Windows Terminal, macOS Terminal and a Linux terminal is required before acceptance.
ET1 `TestET1HelpAndCommands`: `?` help and `Ctrl+P` search share the action catalog;
typing/composition guards, arrow/search selection, unavailable reasons, modal
focus and Escape restore work through native PTY input. Every action remains
reachable through Tab without intercepted control/function keys.

### T2 — Build the protected state, credential and transaction core

Files: future `internal/domain/profile.go`, `internal/credentials/store.go`, `internal/configtxn/{transaction,recovery}.go`, `internal/configtxn/transaction_test.go`; credential platform tests are a following slice.
Produce domain types/ports and `Apply`/receipt contract defined above; versioned protected JSON state plus one journal per operation, not a custom general-purpose database. T3 consumes these exact types.
ET2 `TestET2RerunPreservesUnownedState`: no-op rerun does not rewrite; unknown fields/comments/permissions survive; vault failure leaves config unchanged. ET3 `TestET3CrashDriftAndFirstWrite`: interrupt after each durable step, race two absent-file writers, inject final foreign edit and rollback conflict; originals or explicit recoverable conflict remain, never silent loss. Include native Windows ACL/reparse/sharing and POSIX symlink/parent-lock cases.

### T3 — Ship one verified Codex configuration path

Files: future `internal/adapters/client/{adapter,codex}.go`, `internal/adapters/client/codex_test.go`, `docs/compatibility/codex.md`.
Consume T2 contracts; produce Detect/Plan/Validate/Verify for a pinned native version and disposable root. Preserve unrelated TOML and exact backups; distinguish main/default activation, named profile and selected executable. Verify native effective configuration and catalog semantics without submitting a model prompt. Refuse unsupported versions/custom files rather than guessing.
ET4 `TestET4AdapterCapabilities`: selected endpoint reference/model/budget/tier reaches effective Codex config; other five clients show truthful detection/unsupported states. Follow-up adapter slices each require their own identity/schema/license/native verification before Apply is enabled, including a customization-preserving Hermes import preview.

Expansion queue, after each client's T0 contract is verified: create `internal/adapters/client/<name>.go` and `<name>_test.go` for `claude`, `pi`, `opencode`, `grokbuild` and `hermes`, one reviewed slice per client. Each produces the same four adapter operations and pins `TestET4<Name>RoundTrip` (preserve foreign fields, prefill/no-op rerun, validate selected endpoint/model/auth reference with its native client, refuse unsupported schema and roll back injected failure). No assumption that every client supports the same tier/budget fields; unsupported capability is explicit. A Codex-only milestone is labelled accordingly; full requested client support requires all five expansion slices, not permanently disabled rows.

### T3a — Install a missing selected client

Files: future `internal/install/{client,manifest}.go`, `internal/install/client_test.go`, `internal/tui/client_install.go`.
Consume T0 identity/version/platform trust and T3 detection; produce `PlanInstall(ctx, ClientRef, InstallChoice) -> InstallPlan` and `Install(ctx, planID, expectedRevision) -> OperationReceipt`. InstallPlan fixes trusted upstream/version/artifact verification, package-manager ownership, required native runtimes/prerequisites, exact destination and privilege scope. Preview dependencies/elevation before explicit Install; verify resulting executable path/version/architecture before offering configuration. Installers receive no gateway credential.
Preserve detected existing installs/config/credentials/PATH; offer inspection or explicit owner-delegated update rather than reinstall. Cancellation stops owned processes at a safe boundary; decline/failure preserves pre-existing state and yields a redacted receipt. For delegated package managers, record any new partial prerequisite install and its owner; never auto-remove shared dependencies or claim whole-machine atomic rollback. Each later client's existing gated slice supplies its verified install manifest and prerequisite/platform checks, without a new installer architecture.
ET4 `TestET4MissingClientInstall`: start with Codex absent and fixture install sources; explicit Install produces the verified executable; decline, failed prerequisite/download/signature, denied elevation and cancellation preserve pre-existing state and report any partial dependency effects. T4a's `TestET4FreshInstallToLaunch` completes the absent → install → configure → authenticated synthetic launch journey; other advertised client-install capabilities require the same cases in their gated slices.

### T4 — Connect first-run, rerun, dashboard and client detail

Files: future `internal/app/{inspect,plan,apply}.go`, `internal/tui/{setup,dashboard}.go`, `internal/tui/journeys_test.go`; split application wiring and view wiring into separate owned slices if needed.
Consume T2/T3 snapshots/plans and T3a install capability; produce source-labelled prefill, client choices, missing-client Install/Skip path, staged diff/privilege review, apply receipt and dashboard status. Preserve drafts on back/navigation; service decline and optional SSH failure do not block client success.
Gate: ET1/ET2/ET4 plus keyboard-only fresh-root/rerun journeys; selected-client absence reaches installation review and configured Launch after success, or a truthful skipped/failed state. No writes while editing or merely navigating; unknown client schema shows a useful refusal.
ET4 `TestET4ClientDraftJourneys`: selecting never enables; enable/settings/review/
apply/receipt preserve client identity and reviewed values; disabling retains
configuration and safe drafts; navigation/filter/refresh retain independent
drafts; invalid or stale reviews cause no partial apply; discard/cancel affects
only the named draft. Exercise unsupported/missing clients and zero-result filters.

### T4a — Deliver authenticated launch and terminal handoff

Files: future `internal/credentials/{launch,launch_test}.go`, `internal/app/launch.go`, `internal/tui/launch.go`, `internal/tui/launch_test.go`.
Consume T2 credential references, T3 selected executable/config and T3a install receipt; produce the LaunchSpec/LaunchReceipt operation above, wired to Launch after native configuration validation. Suspend fullscreen/raw mode, restore normal console settings and hand inherited stdin/stdout/stderr plus foreground terminal control to the selected client; on exit, spawn failure or cancellation, restore the TUI, selection and terminal exactly once. Never run the client inside the active fullscreen renderer. Non-TTY mode requires an explicit compatible noninteractive action or refuses interactive Launch.
Track only the owned child/process group using native platform primitives; interactive Ctrl+C reaches the foreground client first. Cancellation/close requests stop/reap owned processes with bounded escalation, report the child's actual exit outcome and return terminal ownership. Do not claim successful authentication solely from exit zero, modify native config to inject a secret, or persist the launch environment.
ET2 `TestET2LaunchCredentialDelivery`: with a synthetic vault entry and no inherited key variable, a fixture executable compares the exact selected API key/env name, literal arguments, config/profile and working directory without printing secret values; unrelated secret variables are absent, and parent environment/config/logs/receipts stay unchanged. `TestET2LaunchLockedOrSessionOnly` covers locked/missing vault, missing/expired session handle (no child), and explicitly entered session-only success without persistence. No inference occurs.
ET1 `TestET1LaunchTerminalHandoff`: native Windows/macOS/Linux PTYs prove suspend → interactive child input/output → exit/interrupt/spawn-failure → restored TUI and usable terminal, with no orphan child or secret echo. ET4 `TestET4FreshInstallToLaunch` combines missing-client installation, configuration and the authenticated fixture child; separately open the pinned real client to its initial UI without a model prompt. Release requires these gates as well as configuration-only ET4.

### T5 — Add inventory, metadata, budgets and service tiers

Files: future `internal/catalog/{gateway,metadata,policy}.go`, `internal/catalog/policy_test.go`, `internal/tui/models.go`.
Consume ModelEvidence/BudgetPolicy; produce timeout/size/redirect-safe discovery and source-labelled exact mappings; explicit shared-budget linking/revalidation and adapter-specific tier translation.
ET5 `TestET5EvidenceDoesNotInventCapacity`: unknown alias, stale metadata, advertised-but-unauthorized model, unequal linked capacities, exact presets 272000/353000/500000, unknown capacity attestation, unsupported tier and unchanged exact model ID. Fixture endpoint tests include 401/403/429, bad JSON and redirects with credential non-forwarding. No real inference is needed; an authorized later deployment probe separately verifies actual protocol/capacity/tier support.

### T6 — Make jobs, cancellation and diagnostics recoverable

Files: future `internal/jobs/{runner,store}.go`, `internal/jobs/runner_test.go`, `internal/tui/jobs.go`, `internal/diagnostics/bundle.go`.
Consume transaction receipts; produce bounded redacted events, truthful stage transitions, timeout/cancellation, restart reconciliation and previewable support export. Default history: last 100 receipts/30 days; redact endpoints/paths and exclude contents. Protected rollback backups remain separately managed until explicit cleanup; do not purge an unresolved recovery.
ET7 `TestET7CancelAndRedaction`: cancel before/after irreversible boundary, kill/restart, stale status, hostile ANSI text and long Unicode paths; no secret/endpoint content reaches logs/support exports and no uncertain job auto-retries. Demonstrate empty/loading/error/offline states and event retention bounds.

### T7 — Offer and manage the optional observer

Files: future `internal/monitor/{observe,lifecycle}.go`, `internal/monitor/lifecycle_test.go`, `internal/tui/service.go`, `cmd/ccc/daemon.go`; native backends split into reviewed OS slices.
Consume read-only inspection/receipts; produce `ccc daemon` and `ccc service install|status|start|stop|restart|uninstall` with explicit backend/user scope; templates and ACLs are versioned implementation assets. Native integration tests use disposable user sessions.
ET6 `TestET6OptionalMonitorLifecycle`: install declined/accepted, repeat install, stop/restart/uninstall, login/logout, missing systemd, denied policy, IPC wrong-user/version mismatch and sleep/resume. No client/credential writes or network inference; uninstall leaves client config and credentials intact.

### T8 — Deliver manual repair with bounded execution

Files: future `internal/repair/{brief,review}.go`, `internal/repair/review_test.go`, `internal/tui/repair.go`, `docs/repair.md`.
Consume redacted diagnostics and ChangePlan; produce operator-reviewed brief, optional sandboxed agent run and imported patch plan. Export-only mode ships if process sandbox guarantees are unverified. Automatic repair remains disabled.
ET8 `TestET8RepairAuthority`: injected prompt/path/symlink, credential request, elevation attempt, timeout, cancellation, stale patch and missing working agent all preserve production state; valid patch still passes native validation and ordinary reviewed Apply. Verify exact transmitted prompt contains no secrets/private endpoint and lists egress destinations.

### T9 — Integrate Windows SSH as an independent elevated operation

Files: future `internal/ssh/{plan,windows}.go`, `internal/ssh/windows_test.go`, `internal/tui/ssh.go`, `docs/verification/tui-ssh.md`.
Consume typed elevated plan and transaction receipts; produce user/key/policy/firewall preview and independent result. Arrange controller-reviewed reuse of current SSH helper only after its contract is frozen; this planning worker does not alter it.
ET9 `TestET9SshScopeAndRecovery`: default Private+LocalSubnet, active-Public warning with classification unchanged, explicit Public opt-in, GitHub fingerprint acceptance, pasted/public-file keys, duplicate key no-op, standard-user selection/creation, UAC denial, planted path, broad-rule refusal and late-failure restoration. Native disposable Windows VM verifies actual public-key login/effective ACLs; loopback alone does not prove LAN reachability.

### T10 — Bootstrap, update and release the verified slices

Files: future `scripts/bootstrap.{ps1,sh}`, `.goreleaser.yaml`, `.github/workflows/tui-release.yml`, `docs/verification/tui-release.md`; utility manifests/install adapters are separately scoped subsequent slices.
Consume T0 trust/platform/ownership decisions and reviewed binary. Produce version-pinned one-line HTTPS bootstraps with OS/architecture detection, authenticated artifact verification before execution, user-local install, prefill rerun and actionable denied-download/proxy/no-terminal failures. Publish actual commands only after release location is settled. Optional utilities install only after exact project/platform/license verification and explicit selection; native Windows tmux is not promised (offer a separately identified WSL route later).
Utility queue: one reviewed manifest/adapter pair per verified utility under `internal/utilities/<name>.go` and `<name>_test.go`, with identity/version/platform/license, native installer ownership, preview, install/status/remove and failure receipts. Orca enters only after its repository is identified; adding remote access or another daemon requires a separate permission preview. T0 must specify the bootstrap trust root itself, not only a checksum downloaded beside the binary.
ET10 `TestET10ReleaseInstallAndRollback`: fresh/rerun/upgrade/uninstall at paths with spaces/non-ASCII, tampered artifact/signature, interrupted self-replace and failed restart, package-manager ownership, unavailable signer, all six build pairs and native smoke on all three OSes. Check linked libraries: Linux static and no external language runtime; macOS only documented system dependencies. If literal fully static macOS is mandatory, release is blocked rather than claiming equivalence. Pin provenance/SBOM/checksums, test release archives and rollback, and retain accurate platform/signing limits.

## Implement → review → fix → release gates

Historical planning adaptation of `ultra-code-workflows`: the original plan was a bounded single-worker planning task. Its no-agent/no-deployment scope described that planning pass only. For the current native TUI remediation, the operator explicitly requires independent subagents and rendered visual-review-and-fix loops; the linked P0 is the current acceptance requirement.

For development, use the skill's per-task loop with explicit ownership: spec/ETs → pinned plan/contracts/preflight → implement → review → fix/re-review → integrated release gate. Choose independent implementation and review seats for each bounded task. Record rulings, known errors and handoffs in the controller's workflow records, never silently weaken acceptance to pass.

1. **Preflight:** record dependency versions, known unsupported platforms, task interface producer/consumer rows, baseline and scratch roots. Stage-0 probes resolve every consequential UNVERIFIED dependency before its task. Real credentials, production state and elevation stay with an explicitly authorized operator/controller.
2. **Per task:** name failing behavioral tests, observe RED where code is involved, implement smallest slice, run targeted GREEN and full applicable gate. Proposed Go gate: `go test ./...`, `go test -race ./...`, `go vet ./...`; release builds use `CGO_ENABLED=0`, while race checks use a supported native toolchain. Capture actual results, not worker claims.
3. **Review:** check spec compliance and quality, typed contracts, secret paths, crash/race cases, UI states and exact changed files. Critical/Important findings block; Minor findings have named destinations. Fix rounds 1–3 retain context; rounds 4–5 seek a fresh authorized review/implementation seat; unresolved load-bearing failures require a ruling and blocked dependent slice.
4. **Whole release:** one whole-branch review → one scoped fix pass → scoped re-review. A new Critical/Important issue after this gate reopens the affected task; it does not disappear under a release deadline. Run merged-tree Go/native-client/PTY/installer/SSH gates; live inference only in a separately authorized deployment check.
5. **Try and ship:** install/start/smoke the built release in disposable native environments, execute the published one-liners there, verify dashboard/client effective config/service controls and rollback. No “production ready” claim until ET1–ET10 and every advertised adapter/platform gate pass. Then controller handles any separately authorized commit/publication/notification.

Review Focus pins: absent destination/concurrent first write → ET3; foreign edit immediately before rollback → ET3; cached large capacity on an unknown alias → ET5; repair patch escapes scratch → ET8; active Public network with Private-only firewall → ET9. These pins are prospective tests, not completed verification.

## Prioritized wayfinding agenda and blockers

1. Confirm the observed gap after isolated native-profile/CC Switch trials. Default: build only this manager's remaining headless/configuration/lifecycle gap; an existing tool satisfying the journey can replace a slice.
2. Confirm direct-launch credential ergonomics and required bare-client compatibility. Default: vault + managed launcher, preserving the MVP's existing reference; no promise that arbitrary terminals can resolve a vault reference.
3. Confirm monitor utility and login-only lifetime. Default: local observation, no mutation authority; system services/boot persistence require a separate least-privilege design.
4. Verify exact versions/products: Pi identity change, Grok Build adapter API, customized Hermes, intended utility repositories, orca. Default: unsupported/read-only until evidenced; no guessed installer or schema.
5. Set support floors, release identity/signing credentials, and whether macOS system-library linkage meets “static”. Default: self-contained no-cgo binaries with explicit macOS platform exception; strict fully-static requirement blocks that target.

Only native compatibility/credential behavior, bootstrap trust/signing, repair confinement, and unresolved identity/schema facts block their implementation/release slices. Palette, token names, cache age, provisional CLI name and library choice are reversible routine choices. None prevents delivery of this plan.

## Self-review notes

R1–R12 map to named tasks and ET1–ET10; later unsupported adapters/utilities remain visible, gated scope rather than claimed functionality. All ten requested screens, narrow fallback and terminal states are specified in the companion. Contracts use the same ChangePlan/ModelEvidence/BudgetPolicy/OperationReceipt names throughout. No real deployment, credentials, config inspection, inference, service change or live SSH test occurred. Planning review and source-check evidence are recorded in the final report; runtime acceptance remains future work.
