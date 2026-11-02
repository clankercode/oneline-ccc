# Native development build

The Go application connects the terminal UI to real Codex configuration. This is
the first native slice. The Windows PowerShell installer remains the simplest
setup for a new Windows machine, including installing Codex and optional SSH.

Build from source with Go 1.26.1 or later:

```sh
CGO_ENABLED=0 go build -trimpath -o oneline-ccc ./cmd/oneline-ccc
./oneline-ccc
```

On Windows, run this in PowerShell:

```powershell
$env:CGO_ENABLED = '0'
go build -trimpath -o oneline-ccc.exe ./cmd/oneline-ccc
.\oneline-ccc.exe
```

No separate language runtime is required; macOS still uses system libraries.
Build artifacts are also attached to
the [native CI runs](https://github.com/clankercode/oneline-ccc/actions/workflows/native.yml).
These development artifacts have no authenticated bootstrap or self-updater.

## First use

1. Press F6 to focus the client list. Select Codex with Up/Down, press Space to
   stage management, then Enter to configure it.
2. Keep the current values or select a model and context budget. Use the endpoint
   editor for your gateway URL. Confirm the route supports the chosen budget.
3. Use Gateway to enter a hidden session key, or retain an existing environment
   reference. A session key is delivered only to the launched client process.
4. Review the exact changes, acknowledge the pages, and select Apply.
5. Launch the selected client from the dashboard or command palette.

The current navigation bindings are:

| Input | Native behavior outside text entry |
| --- | --- |
| `?` / Ctrl+P | Open shortcut help / search commands and destinations |
| F6 / Shift+F6 | Move forward / backward between navigation, list, form, action and help regions; restore the region's focused control |
| Tab / Shift+Tab | Visit every control forward / backward, including across regions |
| Left / Right on the navigation bar | Move between horizontal tabs; Enter opens the focused tab's content |
| `[` / `]` | Switch to the previous / next section directly from content |
| Up / Down | Move within the current list, form or action region |
| Enter / Space | Activate the focused control / stage a checkbox choice |
| Escape | Close the top layer or return to the earlier visit, retaining safe drafts |
| `q` / Ctrl+C | Request safe exit; unsaved work and running operations retain their guards |

The visible **More** menu opens Gateway, Service, Repair, SSH, Utilities and
Settings. Help and Commands are also visible controls; Tab traversal remains
available when a terminal intercepts a shortcut. Text editors keep ordinary
characters as input. Contextual footers describe editor and overlay bindings.
Selection, management and installation are separate. Dark, light and monochrome
themes are available.

Native visual and interaction acceptance is **open** after the rejected
implementation. These navigation changes do not certify visual fidelity or a
smooth complete journey. The [P0 remediation](plans/remaining-items-so-far.md#p0-native-tui-visual-quality-and-interaction-remediation)
requires per-screen native/mockup comparisons and independent visual and
interaction review.

Existing choices prefill the editors. Budgets and feature flags imported from
Codex are not evidence of route capacity or tier entitlement: a first import
requires confirmation. The app retains its own confirmations only while the
exact associated configuration still matches. Unchanged applies preserve files.

The default request tier appears beside the model ID. **Up to Ultrafast** describes
available choices independently. The initial accelerated route requires explicit
confirmation of both Fast and Ultrafast for the exact Sol/Astra route, and uses a
complete official catalog matching the native version. Fast-only/other custom
tier-evidence workflows need a later adapter extension. GLM IDs are accepted
with Standard; their capabilities are not inferred from their names.

## Commands and isolation

```sh
oneline-ccc status --json
oneline-ccc doctor
oneline-ccc launch codex
oneline-ccc service status
oneline-ccc observe --once
oneline-ccc recover
oneline-ccc repair
```

Use `--state-dir PATH` to choose a separate private application root,
`--codex-home PATH` to select a Codex home, and `--codex-executable PATH` to select
its native executable. Shell/token-loading wrappers are skipped during detection.
Application state and client roots must be disjoint. `--plain`, redirected stdin
or stdout, and `TERM=dumb` use escape-free plain inspection output. Explicit
commands also run without the TUI. When `NO_COLOR` is present (including an empty
value), interactive terminals retain configuration and management with color
styling disabled. The TUI still uses cursor positioning; use `--plain` for
escape-free output. Help/version do not open application or client state. No
API-key command-line argument is provided.

Codex must already be installed. Configuration mutation initially supports native
**0.166.0 only**. Other versions and the five other detected clients remain read
only with reasons. This build does not install or update clients/utilities.

Apply stages native configuration edits, validates them in fresh processes and
commits through protected transactions. Foreign TOML/comments are preserved;
inline credentials, foreign named profiles, unsafe paths and unsupported native
permissions are refused. Backups/journals are private. Multi-file replacement is
recoverable rather than simultaneous. `recover` is explicit; the observer never
performs recovery. After an uncertain outcome, inspect it before retrying.

Keys remain out of configuration files, reviews and receipts. Existing selected
environment variables can be reused. Native persistent storage is initially
Windows Credential Manager only; Linux/macOS persistence is unavailable until
bounded providers are implemented. Session mode works without a vault. A plain
`codex` process outside the app needs its own selected environment reference;
the app's in-memory session key is available through its Launch action only.

## Optional features

The observer records configuration hashes/timestamps only. It never updates
clients, calls a gateway or runs a repair agent. Linux systemd user and macOS
LaunchAgent lifecycle adapters are implemented with fixture tests; native session
lifecycle acceptance remains separate from foreground configuration tests.
Windows observer/lifecycle is explicitly unavailable in this slice. Foreground
setup works without a service. See the [observer contract](../internal/monitor/README.md).

Windows SSH hands control to the embedded, tested PowerShell helper. Invoke it
from an Administrator terminal; the helper retains its account, public-key and
firewall review. It defaults to LocalSubnet on Private networks. No automatic
elevation or service installation occurs on startup. The standalone installer
also enables showing file extensions and hidden files by default; carrying that
preference control into the native app remains on the broader product plan.

Repair produces a redacted manual diagnostic brief. It does not run another
agent or import generated repairs. Additional adapters, automatic install/update,
native Windows observation, and authenticated release/bootstrap remain on the
[production plan](plans/tui-production-plan.md).
