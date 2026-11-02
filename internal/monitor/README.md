# Optional read-only observer

`New(stateRoot, codexHome, executable)` has no config, service or filesystem
side effects. `Observe(ctx, true)` samples once; false samples immediately and
then every 30 seconds until canceled or an unsafe/read/write error occurs.
Only the exact selected `config.toml` is read (maximum 5 MiB). A single private,
atomically replaced `stateRoot/monitor/observation.json` contains schema version,
observation timestamp and config existence/SHA-256/modification timestamp.
It contains no raw config, paths or endpoint and retains no history. Foreign,
malformed, linked or insecure existing observations are refused. No credential
files, network, update, inference, repair or client writes are involved.

`Status(ctx)` is read-only, bounded and returns native backend/state plus a checked
timestamp and observation freshness when available. `Action` accepts explicit
install/start/stop/restart/uninstall; status delegates to Status. Lifecycle and
observer errors never include raw command output, paths or config contents.

Linux uses a systemd user unit. Install writes and enables it; start is separate.
macOS uses a GUI-session LaunchAgent; bootstrap with RunAtLoad can start it during
explicit install. Both use literal argv and preserve spaces, Unicode and percent:

```
[executable, --state-dir, stateRoot, --codex-home, codexHome, observe]
```

The supplied executable is oneline-ccc. No shell is used. systemd specifier and
dollar expansion are escaped; LaunchAgent values use XML escaping. Automatic
restart is disabled. Existing safe service directories may be 0755; their modes
are never changed. Owned definitions must match a private ownership digest and
exact expected bytes. Foreign files, registrations, symlinks and permission
drift are refused. Configtxn protects installation, with private filesystem
helpers protecting fixed-schema state. Uninstall retains clients, credentials,
application state, observation and transaction/lock records.

Each lifecycle mutation has a durable reservation. Failed/timed-out/uncertain
native commands leave it in place; subsequent calls cannot blindly resend them.
Read-only native proof of the original postcondition can reconcile that reservation
without executing the command again; restart also needs a changed native invocation
identity. Otherwise local reconciliation is required. Do not delete pending.json
and retry without inspecting the native state and protected transaction records.
The observer never runs transaction recovery. Status and commands recheck artifact
ownership after native queries and before mutations. Advisory locks and final
filesystem checks do not defend against malicious code running as the same user.

Native queries/commands have a 10-second bound, 16 KiB output cap and filtered
OS/session environment. An action has an overall 30-second deadline. An uncertain
mutation is reported as Unknown. Missing or unsafe native support is Unavailable
or Conflict, rather than guessed successful state.

Windows explicitly returns Unavailable before filesystem or Scheduler actions.
Protected observer ACL/reparse/atomic-replacement and current-user logon Scheduled
Task ownership/absence/literal-percent behavior need native proof; there is no
SCM, password, elevation or other persistence fallback. Other unsupported OSes
also refuse observation/lifecycle. Windows/macOS cross-compilation is not native
integration evidence. Linux and macOS lifecycle tests use a fake runner and
scratch roots; no operator service/config/vault is touched.

Run the fixture tests with parent-provided GOTMPDIR/GOCACHE/TMPDIR:

```
GOMAXPROCS=2 go test -p 2 ./internal/monitor
```

Parent integration must exercise the frozen CLI and native disposable user-session
service lifecycle; this package's unit tests do not establish native release proof.

The reviewed lifecycle fixes also validate private state directories/records and
service path authority before and after every native await. Linux queries the
manager's typed UnitPath through bounded busctl JSON, scans every returned search
root for foreign main units and unit/dash-prefix/type-wide .conf drop-ins, and
requires empty cached DropInPaths/SourcePath, no foreign aliases and the exact
cached native ExecStart argv. The same proof is refreshed before each native
mutation, including daemon-reload/enable/start/restart. Overrides are preserved
and yield Conflict; they are never activated or adopted. Missing busctl or
unrecognized/incomplete native evidence yields Unavailable without activation.

A missing unit definition does not erase active/transition/enablement/invocation
state. Uninstall can clear a pending reservation only when the artifact, loaded
registration and all that evidence are gone. Inconsistent not-found registrations
are preserved as Conflict/Unknown. Runtime-only enablement is explicitly unsupported
and returns actionable Conflict before any native mutation; it is not interpreted
as disabled. Reconcile it through the native user manager before retrying.

POSIX fixtures skip only on unsupported platforms. A separate Windows test still
executes there and checks Unavailable, no native actions and no filesystem writes.
No native service commands were run to validate these changes; fake-runner proof
and cross-compilation remain distinct from disposable native-session acceptance.
