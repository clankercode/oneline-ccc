# Generated catalog native verification

Result: **PASS native app-server verification for both generated presets.** Native TUI automation also passed both opposite `/ultrafast` toggles and exited cleanly for the generated Sol catalog. No model prompt or inference was submitted.

## Scope and isolation

Read-only production source: `scripts/setup-codex.ps1`. Imported only its exact catalog/config helper function definitions through the PowerShell AST; did not invoke installer main or modify production files. Invoked actual `New-ConfirmedLLMPCodexCatalog` with exact release `0.166.0`, each supported preset and confirmed `fast,ultrafast` tiers. This exercised its real bounded public upstream download, conversion and immutable catalog writer. Invoked actual `New-LLMPConfigBlock` with fixture.invalid endpoint, root service tier default and Ultrafast enabled.

All fixture catalogs, homes, logs, scripts and configuration are under `.cache`. Dummy credentials only. User config and private catalog were not read or changed. Public release download only; no hosted inference, paid API call, Windows installation, service/firewall/ACL/user mutation.

Native engine: cached Linux `codex-cli 0.166.0`. Catalog generation engine: portable PowerShell7.6.9. Native Windows PS5/7 execution is not established by this report.

## Exact app-server outputs

Fresh app-server `initialize` then `model/list` used each generated helper catalog, isolated CODEX_HOME and dummy key. Exactly one selected model appeared per catalog. Root service tier was default; config explicitly enabled features.ultrafast_mode using the actual helper's config block.

| Model | serviceTiers id/name | additionalSpeedTiers | defaultServiceTier | Catalog SHA256 |
| --- | --- | --- | --- | --- |
| `gpt-6.1-sol` | `priority/Fast`, `ultrafast/Ultrafast` | `fast`, `ultrafast` | null | `1649ec3ee58cdf43609f530ea96903c112978f5b6f8d3fd303a200e1bc14acb9` |
| `gpt-6-astra` | `priority/Fast`, `ultrafast/Ultrafast` | `fast`, `ultrafast` | null | `14ae9c3fe9df5c54c045603348f285fd0badfde4f30a3ec59d2f6614ad8e4427` |

Both native outputs match the required shapes. `name: "Ultrafast"` supplies the exact dynamic `/ultrafast` command, with ID `ultrafast` supplying request routing. `defaultServiceTier=null` permits the initial default selection rather than forcing accelerated mode.

Compared the complete generated entries against matching release originals: **only `service_tiers` and `additional_speed_tiers` changed** for either model. Model instructions, tools, reasoning options, context limits, modalities and all other fields remained equal. The generated wrapper advertises only the selected configured route.

## Actual TUI verification: PASS

Repaired the scratch PTY probe to use a fresh fixture home copied only from the generated fixture configuration, with root service tier reset to default. The probe accepts either first observed tier transition, then requires the second transition to be the opposite. Timed state changes run even when no terminal output arrives, eliminating the previous wait bug. Its deadline is 45 seconds with bounded cleanup.

Clean passing run:

1. Exact `/ultrafast` popup displayed the generated tier description.
2. Enter reported `Service tier set to ultrafast`.
3. Exact `/ultrafast` popup displayed again.
4. Enter reported `Service tier set to default`.
5. Asserted opposite transitions and clean native TUI exit **0**.

`native-tier/tui-events.json` records `passed=true`, `stage=done`, transitions `["ultrafast","default"]`, `exitCode=0`, and `noModelPromptSubmitted=true`. Full readable/raw terminal logs are retained. This passing run supersedes the earlier timeout observations.

Combined validation count: **2/2 generated catalogs passed native app-server checks; 1/1 native TUI probe passed with 2/2 opposite tier toggles.** No inference or user-configuration changes.

## Reproduce and artifacts

- Generator: `.cache/powershell/pwsh -NoProfile -File .cache/mvp-workflow/native-tier/generate.ps1`
- Passing native verifier: `python .cache/mvp-workflow/native-tier/verify.py`
- Structured checks: `native-tier/verification.json`
- Exact generated paths: `native-tier/manifest.json`
- Native outputs: `native-tier/gpt-6.1-sol-model-list.json`, `native-tier/gpt-6-astra-model-list.json`
- Full catalog files: content-addressed `.catalogs/oneline-<sha256>.json` under each fixture home.
- TUI harness/transcripts: `native-tier/tui_probe.py`, `native-tier/tui.raw.log`, `native-tier/tui-readable.log`, `native-tier/tui-events.json`.

Source SHA256 at report creation: `711b382b463594c0d31eed86623d452c313696d81a70597446ed1fa215e5460a`.
