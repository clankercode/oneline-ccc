# Existing projects before Go/TUI/daemon expansion

Retrieved **2026-11-02**. Scope: existing tools for coding-agent clients, endpoints, keys, models and context, with emphasis on a Windows Codex MVP. This is a live primary-source desk review, not an installation or compatibility test. No local user configuration, credentials or session histories were read. No implementation is proposed as completed.

## Recommendation

**Evaluate CC Switch first for direct Windows Codex configuration management. Evaluate EasyCLIProxyAPI + CLIProxyAPI when a local gateway is actually required. Compare OpenCodex and Claude Code Router before writing any routing daemon.** All four now document meaningful Windows/Codex overlap; describing CCR as Claude-only or CC Switch as only a file switcher would miss their current scope. Sources: [CC Switch README](https://github.com/farion1231/cc-switch#readme), [EasyCLIProxyAPI README](https://github.com/router-for-me/EasyCLIProxyAPI#readme), [OpenCodex README](https://github.com/lidge-jun/opencodex#readme), [CCR README](https://github.com/musistudio/claude-code-router#readme).

For a Codex-only MVP using an already compatible endpoint, first check whether **Codex's own provider configuration and profiles** remove the need for a persistent manager. A thin Windows setup/launch workflow may be sufficient. Cross-client rule synchronization is a different problem: **Rulesync** is a reusable candidate for that layer, not a proxy or secret manager.

The reason to build something new should be a demonstrated gap in the intended Windows workflow, not a general desire for Go, a TUI or a daemon. No candidate was established here as satisfying every possible requirement; the exact MVP acceptance criteria and target Codex version still need to be matched against an isolated Windows trial.

## Keep these responsibilities separate

| Responsibility | What it changes | Relevant candidates |
| --- | --- | --- |
| Client configuration manager | Writes a client's endpoint/auth reference/model configuration; can work with direct connections | CC Switch direct mode; native Codex profiles; parts of EasyCLIProxyAPI, CCR and OpenCodex |
| Runtime proxy/gateway | Receives inference requests, holds upstream credentials, translates protocols and/or selects upstreams | CLIProxyAPI; CCR; OpenCodex; CC Switch routing/aggregation |
| Rules/context configuration synchronizer | Generates instructions, skills, MCP and related configuration for different clients | Rulesync; portions of CC Switch |
| Conversation/session management | Reads, resumes or edits existing client history | CC Switch session views; EasyCLIProxyAPI's experimental Codex context editor; RyensX/OpenCodex remote interface |
| Replacement coding agent | Provides its own model selection, tools and conversation state | OpenCode; Crush |

“Context” can mean a model's context-window metadata, project instructions, or actual conversation state. These are not interchangeable. Model switching and instruction synchronization do not establish lossless conversation transfer between agents. A proxy's OpenAI-compatible label alone also does not establish Codex Responses API, streaming or tool-call compatibility.

## Candidate comparison

Features below are upstream documentation claims unless explicitly identified as repository/release metadata. “Unknown” means not established in this review, not necessarily absent.

| Project | Type and documented overlap | Platforms established | Windows Codex fit and limits |
| --- | --- | --- | --- |
| [CC Switch](https://github.com/farion1231/cc-switch) | Desktop configuration manager plus optional gateway. Endpoint/key/model switching, model aggregation, MCP, skills, per-tool prompts, project snapshots, session browsing and usage/quota views | README: Windows 10+, macOS 12+, Linux; current release includes Windows x64 and ARM64 installers/portable archives | **Strongest first trial for direct configuration MVP.** Codex is explicitly supported. Documentation says restart terminal/CLI after switching; not evidence of live mutation of a running Codex session |
| [Claude Code Router (CCR)](https://github.com/musistudio/claude-code-router) | Local gateway/control plane plus agent profiles; provider/model discovery, credential pools, retries, ordered fallbacks, protocol conversion and request observation | README desktop downloads: Windows, macOS, Linux; CLI/npm and Docker paths. Windows `.exe` release verified | **Strong gateway alternative.** Current README explicitly supports Codex CLI & App and multiple other agents; substantially broader than its name suggests. Adds a local service rather than simply writing a direct provider profile |
| [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) | Proxy server with compatible API interfaces, OAuth accounts, multi-account balancing, upstream provider configuration, Management API and reusable Go SDK | Current release has Windows amd64/aarch64 archives; project also supplies other OS builds | **Reusable proxy core**, especially before custom Go daemon work. By itself, not a full Windows client-configuration UX |
| [EasyCLIProxyAPI](https://github.com/router-for-me/EasyCLIProxyAPI) | Desktop console for CLIProxyAPI: core lifecycle, credentials, providers, quotas, model aliases, analytics and agent configuration | README: Windows amd64/aarch64 ZIP, macOS amd64/aarch64 DMG, Linux amd64/aarch64 TAR.GZ | **Strong turnkey gateway + configuration candidate.** Explicit Codex integration, model sync, default selection and configuration backup/restore. Also accepts WSL UNC configuration paths; that does not change executable launch behavior |
| [OpenCodex / `ocx`](https://github.com/lidge-jun/opencodex) | Local gateway/dashboard plus managed client integrations, provider/model catalogs, failover/weighted combos, account pools and runtime lifecycle | README: Windows x64 without WSL, macOS arm64/x64, Linux x64/arm64 CLI; desktop packaging is narrower | **Strong Codex-focused gateway candidate.** Explicit Codex CLI/App/SDK overlap and other client integrations. Changes client integration and runs a gateway; evaluate restoration and version sensitivity |
| [CCS](https://github.com/kaitranntt/ccs) | Multi-provider profile/runtime manager; isolated Claude subscription contexts, API/local-model profiles, CLIProxy runtime and optional CCR integration | npm host and Docker installation documented in retrieved README; Windows support not established here | Useful wrapper/profile design reference; README now mentions Codex. Native Windows Codex parity remains **unknown**, so secondary to candidates with verified Windows release assets |
| [Rulesync](https://github.com/dyoshikawa/rulesync) | Generate/import/convert rules, commands, MCP, subagents, skills, hooks and permissions across coding tools | Node CLI; current release includes `rulesync-windows-x64.exe`; macOS/Linux Homebrew path | **Complement for shared instructions/context configuration.** Codex CLI target is explicit. Not endpoint routing, API-key pooling or conversation-history transfer |
| [OpenCode](https://github.com/anomalyco/opencode) | Replacement coding agent with its own UI/configuration; not a manager of existing Codex installations | README Windows/macOS/Linux desktop packages; Windows release assets verified | Alternative if replacing the client is acceptable. Does not meet “keep using Codex and manage its configuration” by itself |
| [Crush](https://github.com/charmbracelet/crush) | Replacement Go/TUI coding agent; custom providers, model context metadata, MCP, skills and session handling | README explicitly includes Windows PowerShell/WSL, macOS, Linux and BSD variants | Existing Go/TUI reference if the task expands to a whole agent. **License is FSL-1.1-MIT, not simply MIT**; not the default code-reuse choice for a competing product |

Sources for the comparison are each project's linked repository README and the release/license evidence below. OpenCode and Crush are included to prevent confusing reusable agent implementations with configuration managers.

## Detailed findings and reuse boundaries

### CC Switch: configuration first, optional routing

The current README describes twelve supported tools including Codex, Claude Code, Gemini CLI, OpenCode and Pi. Direct mode writes connection details; routing mode forwards and translates requests locally; aggregation exposes models from multiple providers in a client model list. The Codex row explicitly includes providers, routing, aggregation, tray switching, MCP, skills, `AGENTS.md`, sessions and usage statistics. [README and feature matrix](https://github.com/farion1231/cc-switch#readme).

The README claims preservation of unrelated configuration and comments, a resulting-file preview, and automatic backup before the first rewrite of each configuration file. It documents importing existing configuration on first launch, project switching for Codex/Claude and a return to official login. These are particularly relevant to a reversible setup MVP, but are **claims to validate**, not proof of byte-preserving behavior across every TOML structure or concurrent edit. [Provider management / quick start](https://github.com/farion1231/cc-switch#readme).

**Reuse:** prefer adopting the existing application if its workflow fits; its MIT repository is also a potential adapter/reference source. **Gaps:** headless automation/CLI suitability, exact supported Codex version range, credential-at-rest protection, concurrent-edit behavior and crash recovery were not established. Its broad UI may exceed a minimal one-command installer, but that is a workflow tradeoff rather than evidence to rebuild its features.

### CCR: current scope includes Codex

CCR now calls itself a local model gateway and control plane for multiple agents. Its quick start is provider setup → start gateway → apply agent profile. It documents OpenAI Chat/Responses, Anthropic Messages and Gemini formats; provider credentials/pools; route conditions and rewrites; retries/fallbacks; client keys; usage/cost/log inspection; and Codex CLI & App support. Desktop, CLI and Docker installation paths exist. [Current README](https://github.com/musistudio/claude-code-router#readme).

**Reuse:** compare as a complete existing gateway/control plane rather than implementing equivalent routing. **Gaps:** no Windows Codex execution was performed; full protocol behavior, config rollback, minimum Windows version and credential encryption were not established. The README describes request/response detail logging; inspect default retention/redaction before adopting with real traffic. This review makes no claim that logs are secret-safe by default.

### CLIProxyAPI plus EasyCLIProxyAPI: core and desktop manager are distinct

CLIProxyAPI explicitly supplies compatible inference endpoints, Codex/Claude OAuth support, multiple-account round-robin and configurable OpenAI-compatible upstreams. It documents a reusable Go SDK and Management API. This is the closest direct reuse route if a later Go utility genuinely needs a proxy engine. [README](https://github.com/router-for-me/CLIProxyAPI#readme), [SDK usage](https://github.com/router-for-me/CLIProxyAPI/blob/main/docs/sdk-usage.md).

Important scope detail: the README says built-in usage statistics were removed since v6.10.0; do not assume the bare core includes a persistent analytics dashboard. Management/UI extensions are separate projects. The linked hosted [Management API documentation](https://help.router-for.me/management/api) returned **HTTP 403** in this retrieval; its detailed contract was not verified.

EasyCLIProxyAPI is upstream's recommended desktop companion. Its own README documents detecting agent clients, synchronizing model catalogs, selecting a default model, backing up original client configuration and restoring previous settings. It can manage the core and credentials and provides analytics separately from the bare proxy. It explicitly accepts Windows WSL UNC configuration directories, while warning that configuration location does not control executable launch. [EasyCLIProxyAPI README](https://github.com/router-for-me/EasyCLIProxyAPI#readme).

EasyCLIProxyAPI also documents **Codex Session Management → Edit Context (Experimental)**. Saving changes real local history and creates backups; upstream says to close the target session first. This is unusually direct context-management overlap, but **not** evidence of safe concurrent history editing or portable context across clients. A configuration-only MVP need not touch that feature.

**Reuse:** evaluate the packaged pair before writing service lifecycle, tray UI, model synchronization or backup/restore. **Gaps:** exact client schema compatibility, unattended install/update operation, least-privilege behavior, Windows filesystem edge cases and secret storage guarantees remain untested.

### OpenCodex: identify the repository, not just the name

The relevant provider gateway is **[`lidge-jun/opencodex`](https://github.com/lidge-jun/opencodex)** (`ocx`, npm package `@bitkyc08/opencodex`). It documents managed Codex integration, other client connections, providers/models, protocol bridging, weighted/failover combos, ChatGPT account pools and a local web dashboard. It claims `ocx stop` restores Codex's original configuration. Its platform table documents Windows x64 without WSL, Task Scheduler lifecycle and an opt-in native service via WinSW. [README](https://github.com/lidge-jun/opencodex#readme), [client support reference](https://opencodex.me/reference/client-support/) (HTTP 200 verified).

**Reuse:** existing Codex-focused gateway and lifecycle tooling could replace much of a proposed daemon. **Gaps:** restoration under competing edits/crashes, integration behavior on the target Codex version, installer signing status for the selected artifact and protocol fidelity remain trial questions. Streaming/tool/reasoning/image support is conditional on the client/model/provider; do not turn “any model” marketing into a universal compatibility guarantee.

A distinct project, **[`RyensX/OpenCodex`](https://github.com/RyensX/OpenCodex)**, is a middleware/browser access layer for an installed Codex/ChatGPT Desktop runtime, not the same provider gateway. Its README describes remote access from phones/other computers, sessions and intelligent model scheduling; it requires an installed desktop runtime. Windows/macOS launcher and Linux command-line paths are documented. It is **AGPL-3.0**, with release `2.1.0` published 2026-09-13 and a Windows installer. Useful only if remote desktop/runtime access becomes a requirement; not a substitute for endpoint/key configuration management. An attempted guessed URL, `beshkenadze/opencodex`, returned HTTP 404; it was not treated as a real candidate.

### Rulesync and replacement agents

Rulesync generates tool-specific files from unified rule sources and can import or directly convert existing rule formats. Its support table includes Codex CLI rules, MCP, commands, subagents, skills, hooks and permissions, with mode-specific limitations (for example, Codex commands are global-only). This addresses **instruction/configuration context**, not a shared live conversation or credential-routing layer. [README](https://github.com/dyoshikawa/rulesync#readme).

For replacement agents, use canonical identities: `sst/opencode` redirects to **`anomalyco/opencode`**, while **`opencode-ai/opencode` is archived** and its README points to Crush. The archived repository had MIT metadata, but current Crush uses **Functional Source License 1.1 with an MIT future license** and a Competing Use restriction. Do not infer current Crush reuse rights from its predecessor's license. [Current OpenCode](https://github.com/anomalyco/opencode), [archived OpenCode README](https://github.com/opencode-ai/opencode#readme), [Crush license](https://github.com/charmbracelet/crush/blob/main/LICENSE.md).

## Native Codex baseline: check before building a manager

Official documentation retrieved successfully from `developers.openai.com` redirected to the `learn.chatgpt.com` pages below. It documents custom providers, endpoint/auth configuration, model selection, context-window metadata and named CLI configuration profiles. This can cover direct-endpoint selection without a proxy. A profile is configuration layering; it does not by itself establish independent accounts, independent history or cross-client context synchronization.

- [Advanced configuration](https://learn.chatgpt.com/docs/config-file/config-advanced): current live docs describe `--profile profile-name` overlaying `$CODEX_HOME/profile-name.config.toml` on the main config. **They explicitly say not to nest that file under `[profiles.profile-name]`.** They also describe `CODEX_HOME` as a state root containing more than configuration.
- [Configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference): `model_provider`, custom `model_providers`, provider auth fields, `model_context_window`, and `model_catalog_json` are documented. Current docs distinguish user-level provider configuration from project-local settings and say built-in provider IDs cannot be overridden.
- [Windows sandbox documentation](https://learn.chatgpt.com/docs/windows/windows-sandbox): native Windows configuration is documented. A manager's Windows installer alone does not prove Codex sandbox or terminal-launch behavior.

**Version caveat:** live docs describe current behavior, while latest GitHub stable release metadata showed Codex `rust-v0.166.0`. This review did not establish that every current documented profile/config feature exists in that release or in the MVP's target executable. Pin the intended binary version and validate its actual schema before selecting a file-writing strategy; do not copy an old `[profiles.*]` example or a new separate-profile-file example without checking compatibility.

## License and maintenance snapshot

Repository metadata (`archived`, `pushed_at`, license detection) and latest non-prerelease release records were fetched from GitHub REST on the retrieval date. Release dates are UTC. Activity is evidence of recent publication, not a support/SLA or quality guarantee. README features may be ahead of a released binary. License labels below describe the repository, not every dependency, provider service or distributable component.

| Repository | Declared/detected license | Latest stable release observed | Published UTC | Last push UTC date |
| --- | --- | --- | --- | --- |
| farion1231/cc-switch | MIT | v4.0.24 | 2026-11-01 | 2026-11-02 |
| musistudio/claude-code-router | MIT | v3.1.3 | 2026-10-09 | 2026-10-10 |
| router-for-me/CLIProxyAPI | MIT | v8.0.51 | 2026-11-01 | 2026-11-02 |
| router-for-me/EasyCLIProxyAPI | MIT | v0.3.65 | 2026-11-01 | 2026-11-01 |
| lidge-jun/opencodex | MIT | v2.96.0 | 2026-11-01 | 2026-11-01 |
| kaitranntt/ccs | MIT | Not retrieved | Unknown | 2026-10-27 |
| dyoshikawa/rulesync | MIT | v43.4.2 | 2026-11-01 | 2026-11-01 |
| anomalyco/opencode | MIT | v1.19.5 | 2026-10-28 | 2026-11-02 |
| charmbracelet/crush | FSL-1.1-MIT, verified in LICENSE.md; GitHub reports “Other” | v0.99.0 | 2026-10-29 | 2026-11-02 |
| RyensX/OpenCodex | AGPL-3.0 | 2.1.0 | 2026-09-13 | 2026-09-13 |
| openai/codex | Apache-2.0 | rust-v0.166.0 | 2026-10-29 | 2026-11-01 |

All listed active candidates had `archived: false`. Separately, `opencode-ai/opencode` had `archived: true`, last push 2025-09-18.

Primary machine-readable evidence is reproducible at `https://api.github.com/repos/<owner>/<repo>` and `https://api.github.com/repos/<owner>/<repo>/releases/latest`, using the exact repository identities in the table. Release pages: [CC Switch](https://github.com/farion1231/cc-switch/releases/tag/v4.0.24), [CCR](https://github.com/musistudio/claude-code-router/releases/tag/v3.1.3), [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI/releases/tag/v8.0.51), [EasyCLIProxyAPI](https://github.com/router-for-me/EasyCLIProxyAPI/releases/tag/v0.3.65), [OpenCodex](https://github.com/lidge-jun/opencodex/releases/tag/v2.96.0), [Rulesync](https://github.com/dyoshikawa/rulesync/releases/tag/v43.4.2), [OpenCode](https://github.com/anomalyco/opencode/releases/tag/v1.19.5), [Crush](https://github.com/charmbracelet/crush/releases/tag/v0.99.0), [Codex](https://github.com/openai/codex/releases/tag/rust-v0.166.0).

## Remaining decision and validation gaps

These are proposed evaluation criteria, **not features already tested or authorization to expand implementation**:

1. **Direct endpoint or translation?** If the selected upstream already supports the target Codex wire protocol, compare native profiles and CC Switch direct mode. Require a concrete translation/account-pooling need before selecting a gateway.
2. **Reversible Windows configuration:** use synthetic credentials and an isolated Windows user/config root; inspect exact changed files, preservation of unrelated TOML, backups, rollback after interruption, and behavior if another process edits configuration. Do not point candidate auto-import at real user state during evaluation.
3. **Actual Codex behavior:** pin a client version; validate launch, selected endpoint/model, Responses streaming, tool calls, reasoning metadata and surfaced errors against a controlled test endpoint. Model-list visibility alone is insufficient.
4. **Secrets and logging:** establish storage location, encryption/OS credential-store use, file permissions and log redaction. None was independently audited here. Keep proxy listener/admin exposure explicit; no remote access is needed merely to manage local configuration.
5. **Windows lifecycle:** check PowerShell/native terminal launch, paths containing spaces/non-ASCII, portable versus installer packaging, non-admin operation, shutdown and restoration. Treat WSL configuration access separately from launching a WSL process.
6. **Context requirement:** choose explicitly between model context limits, `AGENTS.md`/skills/MCP, session resume and editing history. Prefer existing configuration synchronization; avoid history mutation unless required and separately validated.
7. **Automation interface:** if the MVP must be a one-command script or headless utility, verify each candidate's supported CLI/API and install automation. A desktop feature matrix does not establish that interface. CLIProxyAPI's Go SDK is a promising reuse seam, not yet an integration decision.

**Decision gate:** trial the smallest existing path first; record an observed blocker for each rejected candidate. A later custom utility should own only the remaining Windows setup/configuration/launch gap. This research found no reason to begin a broad new TUI, model gateway or daemon before those trials.
