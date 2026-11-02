# Native foundation verification

The first native development build implements the foreground Codex path in
[native-foundation.md](../plans/native-foundation.md). It retains the approved
colored dashboard, model/default-tier distinction, shortcut help, command search,
draft guards and explicit review/apply. See [usage](../native-app.md).

## Evidence and acceptance boundaries

Linux verification uses a native Codex **0.166.0** executable with explicit
disposable client/state roots and a complete version-matched catalog carrying
the fixture's explicit tier confirmations. No operator configuration or API keys
are development inputs. Native adapter and application journeys exercise first
setup, preservation of unrelated TOML, exact default/profile loading, rerun
without rewriting, stale-review refusal and session-key delivery. No inference
is needed for these checks.

The terminal journey uses tmux and a network-isolated Bubblewrap sandbox with a
private fixture home. It traverses Dashboard, help, command search, masked key
entry, client configuration, full review, Apply and receipt. No client config
exists before Apply. Native launch transfers the terminal to Codex; interrupting
the child returns to the dashboard. Exit restores the original terminal modes.
The final acceptance correction shares process ownership between direct launches,
native configuration calls and version probes. It stops inherited descendants
before removing private staging, and restores the pre-child terminal state before
Bubble Tea resumes. A real rendered-PTY regression covers SIGTERM while a child
has the terminal in raw mode; the isolated native Codex journey checks the same
shutdown path.
Captured layouts cover 120x40, 100x30, 80x24, 60x20 and 40x12. These are Linux
terminal results; native Windows/macOS console behavior requires its own proof.

The [native CI workflow](../../.github/workflows/native.yml) runs tests and static
checks on Linux, Windows and macOS, including isolated native Codex and Windows
Credential Manager checks. It also builds amd64/arm64 binaries for all three OS
families with cgo disabled. Compilation is separate from runtime acceptance.
Each uploaded development artifact includes its checksum.

The rendered-PTY SIGTERM regression runs without the race detector. Running that
specific fixture with race instrumentation reproduces a reader-close race in the
pinned Bubble Tea/cancelreader dependency. Headless shutdown and process ownership
remain race-tested. This dependency limitation is retained for wider production
acceptance; a race-suite pass does not imply that the rendered-PTY test ran.

Observer tests use fake native runners and disposable service definitions.
Windows explicitly tests Unavailable with no native actions or filesystem writes.
This does not certify real systemd/LaunchAgent lifecycle in a disposable session.
The SSH helper remains the existing PowerShell implementation with its separate
[native Windows evidence](windows-mvp.md).

## Remaining product work

Only the exact supported Codex version is writable. Other clients are detected
with reasons and remain read only. This slice does not install clients or
utilities, automatically update them, run repair agents, or install background
components on startup. Persistent native keys initially use Windows Credential
Manager; foreground session/environment credentials work without a vault.

Configuration/profile setup and inference are distinct: no paid gateway protocol
claim follows from a successful offline configuration check. Available tiers
require exact route confirmation; context presets remain user budgets requiring
capacity confirmation. Native binaries are development artifacts, with wider
platform/release and automatic distribution gates retained in the
[production plan](../plans/tui-production-plan.md).

The final integration revision, independent review disposition and exact final
gate results are recorded in the [native-foundation ledger](../ledgers/native-foundation-ledger.md).
