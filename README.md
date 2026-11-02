# oneline-ccc

Configure coding clients for a custom model gateway. The first deliverable is a
standalone **Windows PowerShell installer for Codex + LLMP**. The broader Go/TUI
and optional maintenance service are design work for a later phase.

## Windows quick start

The installer asks for your gateway URL and offers `gpt-6.1-sol`, `glm-5.3`,
`glm-5.3-flash`, or an exact custom model ID. It also asks for a **context budget**
and a privately entered **API key**. No deployment endpoint or credential is
bundled with the installer. Use an endpoint that supports
the **Responses API**; a Chat Completions-only endpoint is insufficient for Codex.
Include `/v1` in the URL if your gateway requires it.

Paste this into PowerShell on Windows:

```powershell
& ([scriptblock]::Create((Invoke-RestMethod 'https://clankercode.github.io/oneline-ccc/setup-codex.ps1')))
```

The [setup site](https://clankercode.github.io/oneline-ccc/) and installer are
published by GitHub Pages after Windows tests pass. The downloaded script runs with
your user permissions. The API key is entered after
download; do not put it in the URL or the command line.

You can use the installer immediately by copying `scripts/setup-codex.ps1` onto
the Windows machine, opening PowerShell in its directory, and running:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\setup-codex.ps1
```

This changes execution policy for that process only. No persistent policy change
is required. Windows PowerShell 5.1 or PowerShell 7 is required.

After setup, start the configured client with:

```powershell
codex
```

The installer makes LLMP the default. Reopen other terminals after setup so they
inherit the API key and PATH. No PowerShell execution-policy change is needed
to launch the native executable.

## Configuration and context budgets

Setup updates `$HOME\.codex\config.toml` (or `config.toml` under `CODEX_HOME`)
so bare `codex` uses LLMP. It preserves unrelated settings and creates backups
before replacing changed files. Existing TOML is merged using Codex's own
configuration API in an isolated staging directory.

Setup also maintains `llmp.config.toml` for an explicit `codex --profile llmp`
launch. Unrelated or manually changed files at that named-profile path are
refused. Codex **0.134.0 or later** is required.

The configuration points Codex at the LLMP endpoint using `wire_api = 'responses'`
and `env_key = 'LLMP_API_KEY'`. The key is stored in the Windows **user environment**
and the current process environment. This is normal persistent environment storage,
not an encrypted credential vault. No API key is written to generated TOML.
Other already-open terminals may need to be reopened to inherit the key.

Choose a budget at or below the selected model's actual input capacity. The setup
accepts explicit budgets such as **272000**, **353000**, and **500000**; these are
operator choices, not claims that every model supports those sizes. By default,
automatic compaction begins at **90%** of the selected budget. Increasing a client
setting cannot increase the provider's capacity. Reserve room for instructions,
tools and output as required by your model/provider.

To supply non-secret choices while keeping the API-key prompt private:

```powershell
.\setup-codex.ps1 -BaseUrl 'https://YOUR-LLMP-HOST/v1' -Model 'EXACT-MODEL-ID' -ContextWindow 500000 -AutoCompactTokenLimit 450000
```

`-DryRun` prints proposed TOML without installing software, persisting credentials,
or changing configuration. `-SkipInstall` configures only; `-NonInteractive`
requires all necessary inputs instead of prompting. For automation, supply
`LLMP_API_KEY` through your process environment or use a `SecureString` `-ApiKey`.
`-SkipPersistCredential` prevents persistent user-environment credential changes.

If Codex is absent, setup installs the official `@openai/codex` npm package into
your user npm directory and updates your user PATH. If Node.js is missing, setup
offers to install LTS through `winget`; Node's installer may request administrator
approval. If winget is unavailable, install [Node.js LTS](https://nodejs.org/),
reopen PowerShell, and rerun setup. Existing Codex versions below 0.134.0 must be
upgraded before using the separate profile.

## Windows Explorer defaults

After successful Codex setup, the installer enables **file name extensions** and
**hidden files** for the current Windows user, before offering optional SSH setup.
It sets only the DWORD values `HideFileExt=0` and `Hidden=1` under
`HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`; protected
operating-system file visibility (`ShowSuperHidden`) and unrelated preferences
are unchanged. Already-correct values are left untouched on reruns.

Add `-SkipExplorerDefaults` to keep your Explorer preferences. This step also
skips non-Windows and `-SkipInstall` configuration-only runs. `-DryRun` describes
the proposed defaults without accessing or changing the registry. No administrator
approval or Explorer restart is requested; refresh or reopen existing File Explorer
windows if needed. A preference failure prints a local warning and leaves Codex
setup successful. The standalone SSH helper never changes these preferences.

To reverse the preferences later, use File Explorer's View options to hide file
name extensions or hidden items again.

## Rerunning setup

Press Enter to keep the current LLMP URL, model, service tier and context budget.
The active `oneline_llmp` configuration supplies defaults; otherwise setup uses
its saved LLMP profile. Other providers' model/context/tier values are not
imported. Explicit command arguments take precedence. A custom compaction
threshold is retained while context stays the same; changing context recomputes
90% unless you explicitly set a threshold.

The key prompt accepts Enter to keep a saved key without displaying it. Setup
checks `LLMP_API_KEY` in the current process, then the Windows user environment,
so a PowerShell session without the saved variable can still reuse it. A typed
or explicitly supplied key replaces it. Noninteractive reruns reuse known values
and refuse missing required inputs.

Unchanged reruns of installer-generated configuration do not create redundant
backups. Changes still update
the managed LLMP fields in main and named configurations, preserving unrelated
settings and exact originals in backups. Switching endpoint/model does not
inherit the previous route's tier/catalog. Installer-generated catalogs refresh
against the selected native Codex release; external catalogs remain external.

Dry-run never writes files. Exact generated configuration can supply its defaults;
for arbitrary main TOML that needs native parsing, supply inputs explicitly in
dry-run mode. Normal setup uses private native staging to read that TOML safely.

To include the SSH offer in a rerun, add `-EnableSshServer`. You can also run the
SSH helper independently; it does not need or change the Codex URL/key.

## Model discovery and service tiers

Optional model discovery lists IDs from your base URL's `/models` route. It sends
your key only to that configured endpoint and refuses redirects. This inventory
does not establish context sizes or tier support.

With a complete, trusted, version-compatible Codex-native catalog from your gateway,
pass `-ModelCatalogPath 'C:\path\catalog.json'`. Setup uses its exact matching
model entry to offer eligible tiers and configures `model_catalog_json`. The
catalog remains in place and must be available when Codex starts. Setup checks
the selected model and tier metadata; Codex validates the complete catalog.
An OpenAI-compatible `/models` response is **not** a Codex-native catalog.

For the exact LLMP presets `gpt-6.1-sol` and `gpt-6-astra`, setup supplies Fast and
Ultrafast metadata, including the exact `/ultrafast` command name. It downloads
the complete official catalog matching the native Codex executable selected for
the bare command, retains the selected model's native metadata, and stores a local
content-addressed catalog. These presets reflect the operator-confirmed LLMP route;
setup does not discover subscription entitlement.
Explicit `-SupportedServiceTiers` restricts the advertised tiers. An explicit
`-ModelCatalogPath` supplies the metadata instead of automatic generation. When
both arguments are supplied, setup refuses a catalog advertising any tier excluded
by the explicit list before publishing configuration, credentials or PATH changes;
the supplied catalog is never modified. `fast` and `priority` are equivalent for
this check. GLM capabilities are not inferred from these presets.

For all other exact model IDs without a supplied catalog, tiers are a manual
override. Supply only the tiers your gateway advertises for the exact selected
model, for example:

```powershell
.\setup-codex.ps1 -BaseUrl 'https://gateway.example.com/v1' -Model 'EXACT-MODEL-ID' -ContextWindow 500000 -ServiceTier fast -SupportedServiceTiers fast
```

That example is not a capability claim. `fast` maps to the Responses API's
`priority` request tier. Leave the tier unset when eligibility is unknown. Model
IDs remain unchanged; setup does not invent tier-suffixed models. When you do not
select a tier, setup keeps a saved tier for the same LLMP route and model. Without
a saved LLMP tier, it writes `service_tier = 'default'`; an unrelated provider's
fast-tier preference is not inherited. Use `-ServiceTier default` to reset it.

## Optional Windows SSH server

After Codex setup, you can opt into Windows OpenSSH server setup. It asks for
administrator approval in a separate PowerShell window. Declining leaves the
Codex setup complete. To request the SSH offer explicitly, use `-EnableSshServer`
in interactive mode; the helper also supports a direct unattended invocation.

SSH setup installs the Windows capability, configures the service and a TCP 22
firewall rule, and sets up public-key authentication for the selected local user.
Choose a pasted public key, a `.pub` file, or a GitHub username whose published
keys you trust. GitHub import shows fingerprints for approval and is a one-time
snapshot. Never paste or transfer a private key to the server.

The MVP limits firewall sources to **LocalSubnet**, on the **Private** network
profile by default. It warns about active Public networks without changing their
classification. Public-profile access is a separate explicit opt-in, still
limited to the local subnet. Existing broader rules need review before enabling
the server. Other trusted subnets and VPN ranges belong to the later utility.

See [Windows SSH setup](docs/windows-ssh.md) for administrator invocation, account
creation, existing-server behavior, ACLs, and recovery instructions. Optional SSH
failure does not undo a working Codex profile.

## Scope and research

The MVP does not install a daemon, update clients in the background, configure
other coding clients, query models.dev or ask agents to repair your machine.
Those requirements are recorded in [the product brief](docs/product-brief.md).

[Existing-project research](docs/research/existing-projects.md) compares CC Switch,
EasyCLIProxyAPI, CLIProxyAPI, OpenCodex, Claude Code Router and related tools before
we decide what the broader utility should build or reuse. The next step for that
broader utility is a design/grilling session.
