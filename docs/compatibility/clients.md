# Native client compatibility contract

Checked **2026-11-02** by the [stage-zero scout](../verification/tui-stage-zero.md).
This table specifies adapter eligibility for the first native build; it does not
claim the parallel Go adapter has already passed integration tests. No operator
configuration was inspected. PATH lookup and symlink resolution were performed
without executing the five other clients or reading their configuration.

| Client identity | Local observation | First native-build capability | Gate before widening capability |
| --- | --- | --- | --- |
| Codex, OpenAI | Installed native executable reports **`codex-cli 0.166.0`** in fresh isolated CODEX_HOME | Eligible for staged configuration mutation through native API, after transaction/adapter tests; no inference capability asserted | Exact version/executable selection, protected staging, native validation, preservation and drift/recovery gates |
| Claude Code, Anthropic (`claude`) | PATH candidate `/home/xertrov/.local/bin/claude`; runtime version not probed | Detected candidate; read-only/unsupported mutation | Verify exact product/version and native config/auth schema; foreign settings, round-trip and rollback tests |
| Pi (`pi`), canonical [`earendil-works/pi`](https://github.com/earendil-works/pi) | PATH candidate `/home/xertrov/.local/bin/pi`; package/version not verified | Detected candidate; read-only/unsupported mutation | Verify selected installed package against `@earendil-works/pi-coding-agent`, account for rename/older packages, prove native config semantics |
| OpenCode, [`anomalyco/opencode`](https://github.com/anomalyco/opencode) (`opencode`) | PATH candidate `/home/xertrov/.bun/bin/opencode`; runtime identity/version not probed | Detected candidate; read-only/unsupported mutation | Resolve launcher and actual runtime identity/platform/version; prove config schema/preservation/native load |
| Grok Build, [xAI](https://x.ai/build) (`grok`) | PATH candidate `/home/xertrov/.local/bin/grok`; runtime version not probed | Detected candidate; read-only/unsupported mutation | Verify release and official custom-endpoint/auth contract; do not infer config paths/keys from command name |
| Hermes Agent, [`NousResearch/hermes-agent`](https://github.com/NousResearch/hermes-agent) (`hermes`) | PATH candidate `/home/xertrov/.local/bin/hermes`; installed customization/schema not inspected | Detected candidate; read-only/unsupported mutation | Exact version, authorization for customized import, preservation preview, schema and native round-trip/rollback |

Identity references for the non-Codex rows come from retained
[stack identity research](../research/tui-stack.md#client-and-utility-identity-check)
and the [native plan](../plans/tui-production-plan.md); an executable with a
matching filename is not fresh proof it is that product. An absent, unresolved
or incompatible candidate must have a distinct unavailable/unsupported reason.
“Read-only” describes allowed capability, not a claim this scout opened client
files. No other client's schema was newly validated here. “PyAgent” does not
create a seventh identity; use Pi pending exact installed-package verification.

## Codex version and file contract

The initial allowlist is **0.166.0 only**, based on the installed Linux amd64
native executable and [recorded SHA256/experiments](../verification/tui-stage-zero.md).
No `>=` range, prerelease, custom build, desktop-bundled binary, or alternate
platform executable inherits this proof merely by name. Earlier 0.165.0 evidence
in [Windows MVP verification](../verification/windows-mvp.md) remains valid for
that workflow; it does not automatically expand the new native adapter's gate.
Unsupported versions remain detectable/read-only until a separate adapter trial.

Resolve wrappers explicitly: the local PATH Codex wrapper can load an operator
token, so it was bypassed. Every version/schema probe must use a fresh disposable
CODEX_HOME and sanitized child environment before native startup. Probe identity
must match the executable later selected for Validate/Apply/Launch.

Use `config/read` for effective settings and the exact writable user-layer
version; apply only owned key paths with `config/batchWrite`, `filePath`,
`expectedVersion`, and `reloadUserConfig:false` inside staging. Preserve foreign
TOML/comments through the native writer. Verify returned path/status/overrides
and fresh-process effective settings before the transaction layer can commit.
No independent TOML whole-file rewrite or guessed legacy profile table is allowed.

Default `config.toml` and named `<name>.config.toml` are separate activation
surfaces, both with ordinary root keys. **0.166.0 rejects `--profile` for
app-server.** The proven route validates a profile's copied bytes as
`config.toml` in a separate stage home, then verifies the actual named profile
with native offline runtime loading. Generated `ConfigReadParams` has no profile
argument. Do not label surrogate staging as a direct read of the selected named
profile; inherited/default/project layers require explicit adapter coverage.

Tested fields: model, provider, provider endpoint/environment-key reference,
context/compaction integers, tier string and default-versus-profile precedence.
Native parsing of `fast` and synthetic token counts establishes no backend tier
entitlement, model capacity, authentication or Responses compatibility. Real
keys must never enter TOML, diffs, receipts or logs. Synthetic/native loading
checks remain independent of any later authorized inference check.

Each of the five follow-up adapters must pass its own native
`TestET4<Name>RoundTrip` equivalent: preserve foreign fields, prefill/no-op rerun,
native effective config, unsupported schema refusal and injected-failure rollback.
No assumption that clients share tier/budget/auth fields. Missing adapters block
only their own mutation/install/launch capabilities; the milestone is Codex-only.

## Existing configuration-manager reuse trial

The [completed CC Switch CLI 5.11.3 trial](../verification/tui-stage-zero.md#cc-switch-released-headless-route-and-completed-isolated-trial)
used a fixed publisher-digest/checksum-verified Linux x64 musl executable in an
isolated, offline fixture home. It passed headless provider add/switch/rerun,
tested foreign TOML preservation and native Codex 0.166.0 default loading.
Prefer reuse of the proven inventory/switching workflow where it fits; no blanket
adoption rejection follows from this trial. This does not enable the other five
native app adapters or certify the parallel Go implementation.

The convenience `--api-key` route prints and persists an inline bearer token;
it is unsuitable for this app's secret-reference contract. The tested advanced
`--config-file` route with `auth:{}` and provider `env_key` works without putting
the fixture key in live TOML. It also carries explicit context/compaction values;
omitting those from a new provider removes prior provider-owned overrides.
Raw provider tier alone did not override an existing `fast` setting. An attached
common-config tier, explicitly refreshed after activation, reached native
`priority` and then survived a byte-identical rerun. Verify the effective result.

Separate named files survive ordinary switching and direct native profile loading
works. The CLI's temporary `start codex ... -- --profile adoption debug
prompt-input ...` exited 0 but omitted the fixture's named-profile sentinel;
direct native loading included it. Profile synchronization and temporary-home
profile delivery need explicit integration. Child-output redaction, native
validation, vault/session delivery and transaction/recovery contracts remain
requirements; parsing a fixture is not authentication or inference evidence.
