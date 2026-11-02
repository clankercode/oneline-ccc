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
codex.cmd --profile llmp
```

The installer prints the appropriate launch command. For a standalone native
installation, use `codex.exe --profile llmp`; npm installs use `codex.cmd` to
avoid PowerShell execution-policy restrictions on npm's `.ps1` shim.

## Configuration and context budgets

Setup writes `$HOME\.codex\llmp.config.toml`, or the same file under `CODEX_HOME`
when set. It leaves the main `config.toml` untouched. It requires Codex **0.134.0
or later**, whose named profiles are separate files. Existing installer-managed
profiles are backed up before changes; an unrelated file at that path is refused.

Profiles overlay your base user settings. If that base file contains a catalog or
other defaults specific to a different gateway, review them or use a separate
`CODEX_HOME` for LLMP. A separate home also separates Codex's other stored state.

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

Without a catalog, tiers are a manual override. Supply only the tiers your gateway
advertises for the exact selected model, for example:

```powershell
.\setup-codex.ps1 -BaseUrl 'https://gateway.example.com/v1' -Model 'EXACT-MODEL-ID' -ContextWindow 500000 -ServiceTier fast -SupportedServiceTiers fast
```

That example is not a capability claim. `fast` maps to the Responses API's
`priority` request tier. Leave the tier unset when eligibility is unknown. Model
IDs remain unchanged; setup does not invent tier-suffixed models. When you do not
select a tier, setup explicitly writes `service_tier = 'default'` so an unrelated
fast-tier preference in your base configuration is not inherited.

## Scope and research

The MVP does not install a daemon, update clients in the background, configure
other coding clients, query models.dev or ask agents to repair your machine.
Those requirements are recorded in [the product brief](docs/product-brief.md).

[Existing-project research](docs/research/existing-projects.md) compares CC Switch,
EasyCLIProxyAPI, CLIProxyAPI, OpenCodex, Claude Code Router and related tools before
we decide what the broader utility should build or reuse. The next step for that
broader utility is a design/grilling session.
