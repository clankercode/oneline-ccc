# Native foundation stage-zero scout

Checked **2026-11-02**. This is the bounded scout in
`.cache/native-app/stage-zero-brief.md`, consuming the native plan's
[T0](../plans/tui-production-plan.md#t0--verify-the-adoption-and-platform-contract)
and [foundation](../plans/native-foundation.md). It establishes a first Codex
configuration contract, not completion of every production T0 or release gate.
Sources, fixture experiments, decisions and deferred runtime checks are separated
below. No operator configuration, real credential or inference was used.

## First build contract

| Item | Pinned contract | Evidence on this date |
| --- | --- | --- |
| Executable | `oneline-ccc` (`oneline-ccc.exe` on Windows) | PATH already contains unrelated `/home/xertrov/.cargo/bin/ccc`; do not replace it |
| Toolchain | Go **1.26.1** | Installed tool reports `go version go1.26.1 linux/amd64`; [official download index](https://go.dev/dl/?mode=json&include=all) contains stable `go1.26.1` |
| Renderer | `charm.land/bubbletea/v2` **v2.1.1** | [Release](https://github.com/charmbracelet/bubbletea/releases/tag/v2.1.1), published 2026-10-28; [tagged module](https://github.com/charmbracelet/bubbletea/blob/v2.1.1/go.mod) requires Go 1.26.0 |
| Styling | `charm.land/lipgloss/v2` **v2.0.6** | [Release](https://github.com/charmbracelet/lipgloss/releases/tag/v2.0.6), published 2026-08-11; [tagged module](https://github.com/charmbracelet/lipgloss/blob/v2.0.6/go.mod) requires Go 1.25.0 |
| First writable client | Installed native **`codex-cli 0.166.0`** only | Fresh isolated version, schema, config API and offline profile trials below |
| Configuration writer | Codex `config/read` + `config/batchWrite` in protected staging | Reuse the existing PowerShell adapter approach; no independent TOML whole-file serializer |
| Other clients | Detection and read-only/unsupported status | [Client matrix](../compatibility/clients.md); each adapter has its own gate |
| Distribution | Source install, builds and CI artifacts | [Platform/trust contract](../compatibility/platforms.md); automatic bootstrap/self-update unavailable pending authenticated distribution |

These are implementation inputs, not a claim that this scout built or tested the
parallel Go implementation. The dependency pins' minimum Go versions fit 1.26.1;
integrated module/build compatibility belongs to that implementation's checks.

## Isolation and installed executable

PATH resolves `codex` to `/home/xertrov/.local/bin/codex`, a shell wrapper whose
code can read `$HOME/.opencodex/api-token`. It was inspected as executable source
and **not run**. No token file was opened. The probe bypassed that wrapper and
invoked the already installed native executable:

```text
/home/xertrov/.bun/install/global/node_modules/@openai/codex-linux-x64/vendor/x86_64-unknown-linux-musl/bin/codex
SHA256 b8a11fedf24b2dfb59a70c9e52b030f9a508dcd49cd2dd0d41956f4c785156b8
--version => codex-cli 0.166.0 (exit 0)
```

The original Codex API probe's generated files are in
`.cache/native-app/stage-zero-probe-lrfqj60m/` in this worktree. The directory was
created with mode 0700. Before the version probe, `codex-home` was freshly
created and empty. Bubblewrap cleared the child environment, disabled network
access (`--unshare-net`), isolated PIDs, overlaid `/home`, `/root`, `/etc`, `/run`
and `/tmp` with empty filesystems, and exposed only the disposable trial tree
and installed Codex executable inside those overlays. The executable was bound
at the namespace-only path `/tmp/stage-zero-codex`; no host `/tmp` scratch was
created. Child CWD, HOME, CODEX_HOME, XDG config/cache/state and TMPDIR all pointed
inside the trial. The only credential variables were explicit dummy
`OPENAI_API_KEY` and `LLMP_API_KEY` values. No inherited credential or profile
environment reached Codex. Synthetic endpoint: `http://127.0.0.1:9/v1`.

The exact bubblewrap/allowlisted-environment argument array is retained in
`invocation.json`. `rpc-transcript.json`, `results.json`, generated `schemas/`,
offline prompt inputs and parser error evidence are local, disposable artifacts;
none contains real credentials. They are not release artifacts.

## Observed native configuration experiments

Commands below were executed through that isolated invocation, not through the
PATH wrapper. The app-server exchange used newline-delimited JSON over stdio,
`initialize` with client name `stage_zero_scout`, then `initialized`.

| Experiment / command | Observed result |
| --- | --- |
| `--version` | Exit 0, `codex-cli 0.166.0` |
| `app-server generate-json-schema --out <trial>/schemas` | Exit 0; schemas generated from the selected binary |
| `app-server --listen stdio://`, then `config/read {"includeLayers":true}` | Effective config plus exact user layer path and `sha256:` version returned |
| `config/batchWrite` with explicit staged `filePath`, read layer's `expectedVersion`, `reloadUserConfig:false`, and per-key `mergeStrategy:"replace"` | `status:"ok"`, expected staged file path, no `overriddenMetadata` |
| Fresh app-server, `config/read` after each write | Default and separate profile-stage values matched every planned field, including provider endpoint and environment-key reference |
| Retry with old `expectedVersion` and a different model | Error `-32600`, `config_write_error_code:"configVersionConflict"`; exact file bytes unchanged, in both stage roots |
| Foreign root/profile comments and `[history] persistence = "none"` | Preserved in both native edited fixtures; this demonstrates these fixtures, not arbitrary TOML coverage |
| `--profile stagezero app-server --listen stdio://` | Exit 1: `--profile` only applies to listed runtime commands, not app-server |
| `debug prompt-input 'Stage zero offline fixture; do not submit.'` | Exit 0; default instruction sentinel present, named-profile sentinel absent |
| `--profile stagezero debug prompt-input 'Stage zero offline fixture; do not submit.'` | Exit 0; named-profile instruction sentinel present, default sentinel absent |
| Same named command with fixture `model_context_window = "invalid-stage-zero"` | Exit 1, profile file path identified, `expected i64`; valid fixture bytes restored afterwards |

The final harness records **seven grouped checks passed** in `results.json`:
native edits/preservation/conflict refusal and fresh effective config for each of
two roots, bare/default precedence, named/profile precedence, and invalid profile
type rejection. It checked that dummy credential values did not enter either
configuration. No thread or turn was started; `debug prompt-input` only assembled
offline input. Networking was disabled throughout all Codex probes.

Effective values were deliberately synthetic, with no capacity/tier entitlement
claim:

| Field | Default config | Named profile |
| --- | --- | --- |
| `model` | `fixture-default-final` | `fixture-profile-final` |
| `model_provider` | `stage_zero_fixture` | `stage_zero_fixture` |
| `model_context_window` | 256000 | 128000 |
| `model_auto_compact_token_limit` | 200000 | 100000 |
| `service_tier` | `fast` | `fast` |
| Provider endpoint / credential reference | Loopback port 9 / `LLMP_API_KEY` | Loopback port 9 / `LLMP_API_KEY` |

Named profiles are ordinary root-key files, `stagezero.config.toml`, not legacy
`[profiles.stagezero]` tables. The installed generated `ConfigReadParams` exposes
`cwd` and `includeLayers`, with no profile selector. Consequently, the profile
fixture was copied into a separate fresh `profile-stage/config.toml`, edited and
verified through native config RPCs, then its exact resulting bytes copied into
the fixture's `stagezero.config.toml`. The actual named runtime was checked using
the offline command above. The profile includes explicit provider fields so its
isolated validation does not depend on operator/default settings. This is not
evidence of app-server directly selecting a named profile or of every layered
configuration combination.

The production adapter must preserve field ownership and original bytes, operate
on staged copies, use the native layer version for edits, validate with a fresh
native process, and only hand verified staged bytes to the transaction layer.
Default activation and named-profile activation remain distinct. Do not pass
`--profile` to app-server. Preserve supported inherited fields when constructing
profile-validation fixtures; refuse an unsupported layer/schema rather than
flattening arbitrary user TOML. Transactions, recovery and final drift checks are
separate implementation gates; a successful native single-file write does not
prove a multi-file commit is atomic.

This follows the existing repository's
[`Start-CodexConfigProcess` / native merge](../../scripts/setup-codex.ps1) and
[native smoke](../../tests/native-smoke.ps1), inspected here without running the
installer or its system side effects. Existing
[Windows evidence](windows-mvp.md) remains historical evidence, not a scout rerun.

Official OpenAI documentation fetched on this date:
[app-server](https://learn.chatgpt.com/docs/app-server) (readable `.md` body)
describes the handshake, effective `config/read`, and batch configuration edits.
Exact generated request schemas and runtime behavior above take precedence over
assuming the live docs match another installed version. The prior
[configuration research](../research/codex-configuration.md) and
[stack](../research/tui-stack.md) remain inputs, not newly repeated proxy research.

## CC Switch: released headless route and completed isolated trial

The latest stable desktop release observed through GitHub's release API was
[farion1231/cc-switch **v4.0.24**](https://github.com/farion1231/cc-switch/releases/tag/v4.0.24),
published **2026-11-01 15:05:37 UTC**, tag commit
`c0189078aacdd80fd2462f59b345f55f62b41cf2`. Its
[tagged README FAQ](https://github.com/farion1231/cc-switch/blob/c0189078aacdd80fd2462f59b345f55f62b41cf2/README.md)
explicitly says the desktop app requires a graphical interface and recommends
the community-maintained **SaladDay/cc-switch-cli** for servers/SSH/headless use.
The desktop release's assets are GUI packages/archives, not a separate headless
CLI. Its [main entry](https://github.com/farion1231/cc-switch/blob/c0189078aacdd80fd2462f59b345f55f62b41cf2/src-tauri/src/main.rs)
calls the Tauri library; [startup code](https://github.com/farion1231/cc-switch/blob/c0189078aacdd80fd2462f59b345f55f62b41cf2/src-tauri/src/lib.rs)
contains automatic live-config import. No GUI was launched or installed.

The separately released fork is real:
[SaladDay/cc-switch-cli **v5.11.3**](https://github.com/SaladDay/cc-switch-cli/releases/tag/v5.11.3),
published **2026-10-22 11:03:36 UTC**, tag commit
`a285cd7d02fe893a72e9b47340f3893908bcabb6`. The
[tagged README](https://github.com/SaladDay/cc-switch-cli/blob/a285cd7d02fe893a72e9b47340f3893908bcabb6/README.md)
documents `cc-switch --app codex provider list`,
`cc-switch provider switch <id>` and `cc-switch start codex <id>` as CLI commands,
in addition to a TUI. The
[tagged entry](https://github.com/SaladDay/cc-switch-cli/blob/a285cd7d02fe893a72e9b47340f3893908bcabb6/src-tauri/src/main.rs)
parses with Clap, dispatches commands, and runs startup recovery for applicable
commands. The [manifest](https://github.com/SaladDay/cc-switch-cli/blob/a285cd7d02fe893a72e9b47340f3893908bcabb6/src-tauri/Cargo.toml)
defines a CLI binary and Rust **1.91.1** minimum; the README's older Rust 1.85+
build advice is not the selected release's authoritative compiler floor.
This is a distinct project's version, not a desktop v5 release.

The release API lists Linux x64/arm64 glibc and musl archives, macOS x64/arm64/
universal archives, Windows x64 ZIP and `checksums.txt`. No Windows arm64 CLI
artifact appeared in that release. The follow-up brief authorized a fixed
publisher-digest-verified executable inside a strong sandbox. The earlier
independent-signer prerequisite for this evaluation was too strict; it is
superseded here. Shipping bootstrap trust remains a separate release requirement.

### Selected executable and sandbox

Downloaded only the fixed
[`cc-switch-cli-v5.11.3-linux-x64-musl.tar.gz`](https://github.com/SaladDay/cc-switch-cli/releases/download/v5.11.3/cc-switch-cli-v5.11.3-linux-x64-musl.tar.gz)
and [`checksums.txt`](https://github.com/SaladDay/cc-switch-cli/releases/download/v5.11.3/checksums.txt).
Archive size **8,919,026 bytes**; SHA256
`42531873444c8b3b41ac5969a70752550ef4c307438d7ca7ee45bcddf201061c`
matched both the
[fixed release API asset digest](https://api.github.com/repos/SaladDay/cc-switch-cli/releases/tags/v5.11.3)
and published checksum. The checksum file's own API digest also matched. The
archive contains one `cc-switch` executable; `--version` returned
**`cc-switch 5.11.3`**, exit 0, inside the sandbox. No host installation, GUI,
installer script or source build was used.

Artifacts are local to `.cache/native-app/adoption-trial-74ljyfde/` (0700 root).
The executable ran only through saved `sandbox.json`: network/PID/IPC/UTS
isolation, dropped capabilities, read-only `/usr`, namespace-only `/eval`
executable binds, isolated `/proc` and `/dev`, empty operator `/home`, `/root`,
`/etc`, `/run`, and writable disposable `/trial` state. Namespace HOME,
USERPROFILE, CODEX_HOME, CC_SWITCH_CONFIG_DIR, XDG roots, CWD and TMPDIR all
resolve inside `/trial`. The environment was cleared; only an explicit synthetic
provider key was supplied. The repository and operator roots were not mounted.
Boundary assertions found only loopback, zero network routes, PID 1 `bwrap`,
read-only `/usr`, and no operator home/config paths. All host scratch is under
this worktree's cache. The installed native Codex 0.166.0 was separately bound
inside this sandbox; its wrapper was bypassed.

Inspected tagged startup/source before provider listing, then ran `--help`,
`provider --help`, `provider add --help` and relevant config/start help.
`provider list` can initialize/recover app state and import live settings, so
even the list command used fixture-only homes. No network probe, speedtest,
stream-check, OAuth, model request or interactive agent session was run.

### Observed commands and results

All CC Switch commands below use `--app codex` where applicable. Synthetic key
arguments/output are deliberately omitted from public evidence; protected local
command/output records are redacted. Models and token counts are fixture values,
not capability or entitlement claims.

| Trial | Actual result |
| --- | --- |
| `provider list` on seeded foreign TOML | Exit 0; imported `default` plus `codex-official` appeared. Main config bytes unchanged by list and subsequent add; app state initialized |
| `provider add --id adoption-fixture --name 'Adoption Fixture' --base-url http://127.0.0.1:9/v1 --model fixture-selected-model --api-key <fixture>` then `provider switch adoption-fixture` | Both exit 0; fresh native config/read selected exact endpoint/model. Field-mode auth became `model_providers.custom.experimental_bearer_token` in 0600 TOML; add/current printed the fixture key. This route does not meet the native app's secret-reference policy |
| Same-provider field-mode switch rerun | Exit 0; main and named profile bytes unchanged. Existing `fast` tier, root/table comments and foreign history/notice settings survived |
| Field-mode context handling | Existing 256000/200000 context/compaction overrides were removed when the new provider omitted them. They are provider-owned fields upstream, not evidence of arbitrary foreign-TOML loss |
| `provider add --id adoption-reference --name 'Adoption Reference' --config-file /trial/reference-provider.json`, then switch | Exit 0. Raw settings contained `auth:{}`, Responses provider `env_key:"ADOPTION_API_KEY"`, model, explicit 353000/280000 budgets and `service_tier:"priority"`. Native config/read confirmed endpoint/model/reference/budgets; no fixture key in live TOML. Existing `fast` tier remained: raw provider tier is not projected as a provider-owned field |
| Reference-provider switch rerun; restore imported `default` | Exit 0, reference rerun byte-identical. Restoring `default` recovered original model and 256000/200000 budgets; upstream normalized its route ID to `custom` |
| Tier through `config common set --snippet 'service_tier = "priority"'`, provider added with `--common-config`, switch, then common set again with `--apply` | Each exit 0. First attached-provider switch left the tier absent; refreshing common config after activation produced native effective `priority`. Subsequent same-provider switch and common refresh were byte-identical. Use the explicit final refresh and verify effective config; do not equate initial success text with the requested tier |
| Bare Codex `debug prompt-input` after switching | Exit 0. Default instruction sentinel retained. Native config/read confirms final reference model, endpoint, environment-key reference, explicit budgets and `priority` tier |
| Actual `adoption.config.toml`, direct `codex --profile adoption debug prompt-input` | Named file remained byte-identical throughout. Exit 0, named instruction sentinel present; separate-profile convention works with Codex 0.166.0 |
| `start codex adoption-common --dry-run`; `start codex adoption-common -- debug prompt-input ...` | Exit 0; dry run identified `/eval/codex` and temporary CODEX_HOME. Offline default launch returned input successfully, leaving live config bytes unchanged |
| `start codex adoption-common -- --profile adoption debug prompt-input ...` | Exit 0, but named instruction sentinel absent. Direct Codex with the same profile still includes it. Temporary launch does not carry the fixture's separate profile file; passing `--profile` alone did not reproduce direct native profile selection |

Foreign main/root/table comments, `[history] persistence="none"` and the notice
setting survived the tested switches; the named profile was unchanged. This is
fixture coverage, not an arbitrary-config preservation or crash-recovery claim.
The earlier field-mode key remains in CC Switch's fixture app database even after
switching to the reference-only provider; choosing the reference route is not
credential deletion or a vault integration. No fixture key value is in these
docs/report. Offline outputs and summaries are in `commands.json`, redacted
command output files and `observations.json` under the trial cache.

Source explains the relevant boundaries at tag commit
`a285cd7d02fe893a72e9b47340f3893908bcabb6`:
[Codex projection/auth modes](https://github.com/SaladDay/cc-switch-cli/blob/a285cd7d02fe893a72e9b47340f3893908bcabb6/src-tauri/src/live/project/codex.rs)
select row-owned fields and support both env references and inline bearer tokens;
[exclusive fields](https://github.com/SaladDay/cc-switch-cli/blob/a285cd7d02fe893a72e9b47340f3893908bcabb6/src-tauri/src/live/floor.rs)
include context/compaction;
[temporary launch](https://github.com/SaladDay/cc-switch-cli/blob/a285cd7d02fe893a72e9b47340f3893908bcabb6/src-tauri/src/cli/codex_temp_launch.rs)
writes a new config/auth home, rather than copying the operator's profile tree.
No claim is made that its other launch modes solve or fail that profile case.

**Adoption decision:** the released CLI passes basic headless provider switching,
the tested foreign-config preservation, native default loading and ordinary
rerun checks. Recommend reusing/evaluating its provider inventory and switching
workflow instead of rebuilding those features by default. It is not a drop-in
for this app's full contract: use raw environment-key references, redact child
output, verify final common tier application, supply named-profile handling, and
retain required native validation/transaction gates. The temporary named launch
gap and convenience credential behavior are concrete, scoped differences, not a
reason to reject the entire tool. Integration/library boundaries and broader
transaction, vault and platform cases remain untested. The custom native UI may
proceed around those bounded needs; no Go application/runtime support is proven
by this third-party Linux fixture trial. No proxy survey was repeated.

## Remaining gates are local to their slices

- Exact installed Codex version is the first adapter eligibility boundary, not a
  version range; broader versions need their own fixtures and native checks.
- Other five clients need identity/version/schema and preservation/native gates
  before mutation. PATH candidates alone do not pass those gates.
- CC Switch CLI's isolated adoption trial is complete. Reuse integration still
  needs secret-safe invocation/output, named-profile handling and final tier/
  transaction validation; these are bounded adapter/launch concerns.
- Six cross-builds, native terminal/credential/lifecycle proof and actual minimum
  OS floors belong to platform integration/release, not this configuration scout.
- Signer identity, key custody and bootstrap trust are unavailable. Automatic
  self-update and executable-fetch bootstrap remain unavailable; source/build/CI
  surfaces can proceed with truthful artifact labels.
- Utility manifests (herdr/T3/Paseo, ambiguous orca, native Windows tmux) retain
  their existing identity/license/install-ownership gates. None is enabled here.

The existing Windows user-tested SSH path and its documented limitations remain
current. No SSH script, test, configuration or support claim was changed.
