# Codex Windows gateway configuration research

Checked 2026-11-02 against fetched official documentation and installed `codex-cli 0.165.0`. Research only: no user configuration, credentials, authenticated catalog, or inference requests were read or used. Runtime probes used an empty cache `CODEX_HOME` outside the repository. Only this research document is owned by this task.

## Recommended configuration

For native Windows, user configuration lives at `%USERPROFILE%\.codex\config.toml` (or the selected `CODEX_HOME`). For a selectable gateway configuration, write **`$CODEX_HOME/llmp.config.toml`** and launch `codex --profile llmp`. The file is a configuration layer, with ordinary root keys; it is not a `[profiles.llmp]` table. Since **0.134.0**, `--profile` no longer reads the old table and the root `profile = "llmp"` selector is unsupported. Profile-file values override base user settings, but project/CLI layers have higher precedence. [S1, S2]

The MVP must configure the default launch as well: update root values in the main `config.toml`. A named profile alone does not meet the bare `codex` requirement. Preserve unrelated TOML through staged native `config/read` and `config/batchWrite`, then validate a fresh process before replacing the actual files.


Example **contents of `llmp.config.toml`**, using illustrative deployment values:

```toml
model = "company-coding-model"
model_provider = "company-gateway"
model_catalog_json = 'C:\ProgramData\OpenAI\Codex\models.json'
web_search = "disabled"
# Optional profile-wide overrides; use actual supported upstream limits:
# model_context_window = 128000
# model_auto_compact_token_limit = 100000
# Optional: only if the selected model and gateway advertise it:
# service_tier = "fast"

[model_providers.company-gateway]
name = "Company Gateway"
base_url = "https://gateway.example.com/v1"
wire_api = "responses"
env_key = "CODEX_GATEWAY_API_KEY"
requires_openai_auth = false
```

Put root settings before the first table. Use resolved absolute Windows paths; TOML single quotes preserve backslashes. Codex does not expand `%ProgramData%` in the catalog path. Install the catalog separately and restart Codex after changing it. WSL uses Linux paths and its own Linux `CODEX_HOME`, independently of native Windows. [S1, S2, S3]

`env_key` contains the **environment-variable name**, not the credential. Supply the variable to the actual process launching Codex through the existing secret-delivery mechanism. A terminal variable may not reach a desktop-launched app. This is bearer-token authentication; for a gateway requiring `X-API-Key`, use `env_http_headers = { "X-API-Key" = "CODEX_GATEWAY_API_KEY" }` instead. Do not combine `env_key` with command-backed `auth`. `requires_openai_auth = false` avoids the OpenAI login flow; setting it true is a different architecture that sends an OpenAI credential to the gateway. [S2, S4]

Use a custom provider ID rather than overriding built-in `openai`. Sensitive provider/auth settings belong in user/profile or managed configuration: current clients ignore those keys in project-local `.codex/config.toml`. [S1]

## Responses compatibility

With the example base URL, inference must accept **`POST /v1/responses`**. Do not put `/responses` itself in `base_url`. Current docs specify `responses` as the only supported `wire_api` value; a working Chat Completions endpoint is insufficient. Preserve incremental SSE events and terminal `response.completed`, function calls and results, reasoning/input replay, follow-up state, and useful errors. `/v1/models` success alone proves none of this. Do not advertise tools or tier capabilities that the actual route cannot provide. [S4, S5]

## Catalog format and version boundaries

The official catalog specification describes the complete **`ModelsResponse`** object with a top-level **`models` array**. Each model needs a unique, nonempty **`slug`** matching the gateway alias and configured `model`. This is neither the public OpenAI `/v1/models` response nor app-server `model/list` output. The local file replaces the bundled catalog and takes precedence over remote discovery. [S3, S6]

**A complete hand-written minimal catalog is deliberately not claimed here.** The official config JSON Schema validates the *path*, not the JSON catalog contents. The catalog specification identifies the version-matched Rust `ModelsResponse` / `ModelInfo` types as authoritative, and the bundled `models.json` as a complete example, not a JSON Schema. The fetched specification links `rust-v0.162.1` examples: a 0.165.0 deployment must use its own matching tag, and a custom build its exact commit. Those GitHub Rust sources were not fetched under this documentation-only source policy. Consequently, full required catalog fields, serde defaults, and context-field deserialization are not independently verified here. Preserve a complete provider-supplied, version-matched entry rather than manufacturing one from a picker schema. [S6]

Officially documented edits to a complete provider-supplied entry:

```json
{
  "upgrade": null,
  "service_tiers": [],
  "additional_speed_tiers": [],
  "default_service_tier": null
}
```

This is an **entry patch, not a loadable catalog**. The docs prescribe `upgrade: null` for an alias without migration, and clearing all three tier fields for a route without tiers. Preserve model instructions, tool settings, reasoning choices and actual context limits unless a tested provider-specific change is necessary. Catalog metadata cannot create inference access or capabilities. [S5, S6, S7]

Remote discovery is optional: API-key-only custom providers need an explicit `model_catalog_url` with `features.api_key_model_discovery` enabled (default). An API key alone does not trigger discovery at the default `/models` path. Catalog URLs receive provider credentials and headers; explicit URLs reject redirects and responses over 1 MiB. A local `model_catalog_json` suppresses remote fetching. No authenticated fetch was performed. [S6]

## Context sizes: root overrides versus per-model metadata

| Surface | Verified behavior / limitation |
| --- | --- |
| Root `model_context_window` | Integer token count for the active model, not a model-to-size map. |
| Root `model_auto_compact_token_limit` | Integer token threshold for automatic compaction; distinct from model capacity. |
| Named `llmp.config.toml` | Ordinary root context keys load when that profile is selected. One root value still applies when selecting another model within that profile. |
| Legacy schema `definitions.ConfigProfile` | Both context keys are absent; `additionalProperties: false`. Do not emit them into `[profiles.llmp]`. Legacy schema presence does not establish runtime support. |
| Catalog | Intended place for each model's own context metadata. Use complete version-matched upstream entries; do not apply one model's size to all models. |
| Public documented size | GPT-6.1 Sol defaults to **500,000 tokens in Codex and ChatGPT Work**. This does not establish every gateway alias's capacity. [S8] |

Root field types and the legacy-profile mismatch were inspected in the fetched config schema. [S9] A real 0.165.0 parsing probe used `llmp.config.toml`: `model_context_window = "deliberately-invalid"` failed with `expected i64`; replacing it with integer `128000` and adding integer compaction `100000` progressed to the deliberately configured `Model provider research-nonexistent not found` error. This confirms profile-file parsing without contacting an inference endpoint. The example numbers are illustrative, not discovered limits.

No authenticated per-model catalog was available. Exact current context sizes for other models, route-specific limits, effective reserved capacity and compaction defaults remain unverified. Do not infer these from model names or apply Sol's size to Astra/Luna/custom aliases. Global overrides can defeat otherwise correct per-model metadata; omit them when catalog-driven model switching is required.

## Service tiers and speed tiers

`service_tier` is a **string**, selecting `fast` or another tier advertised by the active model; **`fast` maps to request value `priority`**. The config schema describes examples `default`, `priority`, `flex`, and legacy `fast`; a string passing config validation does not prove backend support. Model IDs stay separate from tier selection. No current account-specific `ultrafast` availability was verified. Reasoning effort **Ultra** is a separate feature, not evidence for an `ultrafast` service tier. [S4, S8, S9]

Using the official documented command, generated the installed 0.165.0 app-server schema with a fresh empty `CODEX_HOME`:

```sh
codex app-server generate-json-schema --out ./schemas
```

Observed in `v2/ModelListResponse.json`:

| Client-facing field | Exact generated shape |
| --- | --- |
| `serviceTiers` | Array of `ModelServiceTier`; each object requires string `id`, `name`, `description`. Defaults to `[]`. |
| `additionalSpeedTiers` | Array of **strings**, defaults to `[]`; explicitly deprecated in favor of `serviceTiers`. |
| `defaultServiceTier` | String or null; defaults to null. |

Verified client-facing shape example (illustrative values, not advertised account capabilities):

```json
{
  "serviceTiers": [
    {"id": "priority", "name": "Priority", "description": "Illustrative tier"}
  ],
  "additionalSpeedTiers": ["priority"],
  "defaultServiceTier": null
}
```

This is a **partial app-server Model object**, not catalog JSON. The catalog's documented names are `service_tiers`, `additional_speed_tiers`, `default_service_tier`, but the generated app-server schema alone does not prove identical deserialization rules for those catalog fields. In particular, do not put tier objects into a field expecting a string array or copy camelCase picker objects into a snake_case catalog. The app-server schema does not expose catalog context fields. [S6, S10]

## Verified `/ultrafast` command behavior

A fresh native Codex 0.166.0 home enables `fast_mode` and `ultrafast_mode`
by default. A complete matching catalog containing the selected model with
`{"id":"ultrafast","name":"Ultrafast","description":"Lower latency"}` in
`service_tiers` exposes `/ultrafast`. The TUI derives the slash command from the
lowercased tier **name**, and sends its **id**. Setting only root
`service_tier` or legacy `additional_speed_tiers` does not supply this metadata.
A dummy-key native TUI test toggled `/ultrafast` between default and ultrafast;
no inference request was made.

The complete release catalog and implementation were checked at
[`rust-v0.166.0`](https://github.com/openai/codex/tree/rust-v0.166.0):
`codex-rs/models-manager/models.json`, `codex-rs/protocol/src/openai_models.rs`,
`codex-rs/tui/src/chatwidget/service_tiers.rs`, and
`codex-rs/features/src/lib.rs`. The bundled Sol entry advertises priority;
Ultrafast is an operator-confirmed gateway capability added to that complete
entry. This verifies client eligibility and command behavior, not gateway
entitlement or inference performance.

## Validation scope and deployment checks

Completed: fetched cited official pages, inspected config schema field definitions, generated 0.165.0 app-server schemas, parsed this document's TOML/JSON examples, and exercised named-profile context parsing in an isolated home. The later 0.166.0 catalog/TUI verification above extends that initial research. Windows installer validation is recorded separately in [the MVP verification report](../verification/windows-mvp.md); no backend compatibility claim is made.

Before distributing a real catalog: match the desktop-bundled/CLI version; parse the complete catalog against matching Rust types; start a fresh client; inspect `model/list` for expected advertised tiers; then verify streaming, local tool results, follow-up replay, route attribution and error handling with an authorized test credential. Catalog metadata and request routing must agree. [S5, S6, S10]

## Exact sources

All documentation below was fetched with `curl -fsSL` on 2026-11-02; `.md` variants provided readable bodies. Downloads and generated schemas were kept outside the repository in the research cache.

- **S1:** [Advanced configuration](https://developers.openai.com/codex/config-advanced/) — profile files, 0.134.0 migration, layer precedence, custom providers and project restrictions.
- **S2:** [Connect to a gateway](https://learn.chatgpt.com/docs/enterprise/connect-to-a-gateway) — Windows path, environment auth, custom headers and Windows TOML.
- **S3:** [Roll out a gateway](https://learn.chatgpt.com/docs/enterprise/roll-out-a-gateway) — local replacement catalogs, startup, absolute Windows paths, WSL separation.
- **S4:** [Configuration reference](https://developers.openai.com/codex/config-reference/) — provider auth/transport, `service_tier`, profile-overridable startup catalog path.
- **S5:** [Gateway compatibility](https://learn.chatgpt.com/docs/enterprise/gateway-compatibility) — Responses URL, streaming, tools, conversation and alias requirements.
- **S6:** [Implementing model catalogs](https://learn.chatgpt.com/docs/model-catalogs) — complete response wrapper, version matching, remote discovery and documented tier edits; links authoritative Rust types and bundled catalog.
- **S7:** [Bedrock through LiteLLM](https://learn.chatgpt.com/docs/enterprise/bedrock-through-litellm) — concrete official catalog-edit table, preserving instructions/context limits.
- **S8:** [Codex models](https://developers.openai.com/codex/models/) — Sol 500,000-token default and distinction of reasoning Ultra.
- **S9:** [Official config JSON Schema](https://developers.openai.com/codex/config-schema.json) — inspected root properties, `AbsolutePathBuf`, `ModelProviderInfo`, legacy `ConfigProfile` and `WireApi`.
- **S10:** [App-server documentation](https://developers.openai.com/codex/app-server/) — version-specific schema-generation commands and `model/list`; tier field details additionally verified in locally generated 0.165.0 `v2/ModelListResponse.json`.
