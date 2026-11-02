# Fresh final review — Windows Codex MVP

Reviewer: Codex, inherited GPT-6 model; fresh final-review seat identified as Hubble in the ledger. Independent reviewer, not implementer. No subagents, memory-derived specifications, attention/TTS, operator configuration reads, real credentials, or inference. Only this report and isolated `.cache` probes were written.

Reviewed checkpoint: `45acd35d12b305195d6f793c5baeb3004b40a1fc`; package `bb731915..45acd35`. Installer SHA256 remained `711b382b463594c0d31eed86623d452c313696d81a70597446ed1fa215e5460a`; tests SHA256 remained `7ebfab70d1f8aeea82fee896d2cdc0324c525e5fa2ccc19bacb8ccbf613a5637`. Concurrent changes observed were controller documentation updates only.

## Passes

1. Read the final-review brief first, then the plan/acceptance criteria, task brief and verbatim Review Focus, ledger/rulings, implementer/test handoffs, prior independent review and verification report. Read the packaged diff. Revisited the updated R7/R8 rulings and evidence as supplied during review.
2. Read the complete current installer and SSH helper, core harness, native Codex smoke, native SSH smoke, public-download smoke and all three workflows. Traced catalog inputs through generated metadata and both configuration files; traced native executable/version selection, credentials/PATH publication and rollback, cleanup authorization, and SSH authorization/service rollback.
3. Compared public README/site/research/verification/test-documentation claims with source and evidence. Assessed the supplied native-tier report and refreshed structured/TUI artifacts separately from tests run by this reviewer.

R1–R5 appropriately bound this to the existing MVP and honestly distinguish retrospective RED evidence. R6 permits only the guarded sharing-lock residue case. R7 authorizes concurrent CI/publication, without substituting for this review. R8 explicitly authorizes exact Sol/Astra preset defaults; it does not remove the Review Focus requirement that an explicit tier restriction cannot silently advertise Ultrafast.

## Verification run here

- `.cache/powershell/pwsh -NoProfile -File .cache/final-review-probes.ps1`: exit 0. Exact production installer invocation with a complete Sol catalog, `-SupportedServiceTiers fast` and `-ServiceTier ultrafast` unexpectedly succeeded. Actual native Codex 0.166.0 `config/read` returned `ultrafast`, `features.ultrafast_mode=true`; `model/list` advertised `priority,ultrafast`. Log: `.cache/final-review-probes.log`; fixtures: `.cache/final-review-58079916abe84ec5a1aca5652c4dddb1`.
- The same bounded probe explored an incomplete supplied catalog: installer reported success; native initialization/model listing accepted startup and returned nine fallback models. This did not demonstrate a hard startup failure and is not presented as one.
- `.cache/powershell/pwsh -NoProfile -File .cache/final-review-snapshot-probes.ps1`: exit 0, three checks passed using the actual AST-extracted collector: mismatched parent marks partial and remains sticky after a later successful empty scan; discovery exception marks partial; unverified child edge marks partial. Only a reviewer-created child was started/stopped. Log: `.cache/final-review-snapshot-probes.log`. These mocked CIM checks ran on Linux, not native Windows.
- `git diff --check bb731915 45acd35`: exit 0. Rechecked HEAD and production/test SHA256 values. Source/tests were unchanged by the reviewer.

Evidence inspected, not rerun here: coordinator logs show core **76/0**, SSH **87/0**, and bare-default native smoke success. BASE/current focused logs show **0/4** then **4/0**, with the retrospective timing disclosed. User and updated `docs/verification/windows-mvp.md:40` record publication run **41395265536**, all six jobs successful: Windows PS5.1/PS7 core and SSH harnesses/native smoke, real SSH login/rollback, deploy, and both public download/compare/dry-run jobs. The coordinator reports the hosted installer hash matches the frozen source. No full suite was repeated.

The refreshed `.cache/mvp-workflow/native-tier/verification.json` records both generated presets with exact `priority/Fast` and `ultrafast/Ultrafast`, null default tier, and only the intended tier-field changes. Refreshed `tui-events.json` has `passed=true`, transitions `ultrafast,default`, exit 0 and `noModelPromptSubmitted=true`; the retained readable transcript contains both transitions. This supersedes the earlier scratch PTY timeout. These are inspected independent evidence, not TUI executions by this reviewer.

## Spec verdict and Quality verdict

**Spec verdict: changes requested.** The automatic Sol/Astra catalog and bare-default behavior meet the intended route under R8, but the explicit restriction requirement fails when a supplied catalog is also present. The version used for automatic catalog selection can also differ from the executable ultimately selected for bare Codex.

**Quality verdict: changes requested for two Important issues.** Transactional configuration handling, bounded cleanup uncertainty, SSH refusal/rollback, meaningful native tests and publication gates are otherwise sound within the stated limits. Two Minor documentation issues remain. Counts: **Critical 0; Important 2; Minor 2**.

## Review Focus

| Verbatim focus | Status and pin |
| --- | --- |
| 1. Automatic Sol setup exposes catalog tier id ultrafast and name Ultrafast, so /ultrafast exists. Pin: Sol automatically generates a complete catalog and retains Ultrafast UI metadata. | **Pinned** for the verified installed release: `scripts/setup-codex.ps1:236`, `scripts/setup-codex.ps1:1075`, `tests/run.ps1:516`, `tests/run.ps1:564`; native model/list and refreshed two-toggle TUI evidence agree. Mixed-install version selection is separately I2. |
| 2. Explicit tier restriction cannot silently advertise Ultrafast. Pin: complete catalog restricts advertised tiers to explicit confirmation. | **Half-pinned**: generated-catalog filtering is present at `scripts/setup-codex.ps1:215` and tested at `tests/run.ps1:582`; supplied-catalog integration overwrites the explicit restriction at `scripts/setup-codex.ps1:1028`. Native reproduction is I1. |
| 3. Catalog rerun is idempotent and refuses tampered cache without overwrite. Pin: complete catalog cache is idempotent and refuses tampered contents. | **Pinned**: hash-addressed create/move/read-back verification at `scripts/setup-codex.ps1:261`; rerun and tamper assertions at `tests/run.ps1:577` and `tests/run.ps1:595`; coordinator/current scoped evidence passes. |
| 4. Staging residue warning cannot authorize uncertain process tracking. Pin: focused parent/edge mismatch and discovery exception tests plus native Windows gate. | **Pinned**: sticky uncertainty at `scripts/setup-codex.ps1:518`, parent/edge/error branches at `scripts/setup-codex.ps1:530`, `scripts/setup-codex.ps1:540`, `scripts/setup-codex.ps1:557`; warning gate at `scripts/setup-codex.ps1:732`; reviewer probes above and supplied successful native gate. Native lock-fallback fault injection itself is not claimed. |
| 5. Bare codex uses LLMP while unrelated settings survive. Pin: native-smoke.ps1 effective config and existing full-suite preservation tests. | **Pinned** for normal command resolution: `tests/native-smoke.ps1:35`, `tests/native-smoke.ps1:63`, `tests/run.ps1:275`, `tests/run.ps1:292`; main/named coordinated publication at `scripts/setup-codex.ps1:1099`. Mixed installations are I2. |

## Findings

### Critical

None.

### Important

**I1 — Supplied catalog silently overrides an explicit tier restriction.**

- **Location:** `scripts/setup-codex.ps1:1028`, consumed at `scripts/setup-codex.ps1:1050` and `scripts/setup-codex.ps1:1057`; public promise at `README.md:108`.
- **Scenario:** Supply a complete matching Sol catalog containing Fast and Ultrafast, together with `-SupportedServiceTiers fast`. The supplied list is replaced by all catalog tiers. Adding `-ServiceTier ultrafast` succeeds; omitting it still advertises Ultrafast through the catalog and enables its feature. The full conflicting-input/native-read reproduction above confirmed both the selected tier and advertised metadata.
- **Effect:** The operator's explicit restriction fails silently, and Codex permits a tier the operator expressly excluded. The generated-catalog helper test does not cover this combined-input seam. No inference, entitlement failure or actual extra charge is claimed.
- **Fix:** Preserve the explicit restriction when a catalog is supplied. A bounded fix can refuse conflicting catalog/list combinations before publication rather than silently ignoring the list; alternatively derive a complete restricted catalog without modifying the supplied source. Merely rejecting the selected tier leaves unauthorized picker advertisement unresolved. Add one integrated combined-input case, checking native advertised tiers or an unchanged-state refusal.
- **Ruling assessment:** Catalog precedence sensibly selects the supplied metadata source over automatic generation. R8 does not expressly authorize discarding a simultaneously supplied restriction; task Focus 2 and the README promise are unqualified. This differs from the now-authorized preset-default concern in the focused review.

**I2 — Catalog version and selected native executable use different resolution precedence.**

- **Location:** `scripts/setup-codex.ps1:101`, `scripts/setup-codex.ps1:363`, `scripts/setup-codex.ps1:1074`, `scripts/setup-codex.ps1:1081`.
- **Scenario:** On Windows, keep an older standalone `codex.exe` in a user PATH directory and a newer npm `codex.cmd` installation. `Assert-CodexVersion` uses `Get-CodexCommand`, which prefers the standalone `.exe`. `Get-NativeCodexExecutable` instead prefers the npm `.cmd` package's resolved native binary. Automatic metadata is therefore downloaded for the older executable's version, while staging and the final bare command select the newer npm binary. The new native directory can be prepended successfully when the older directory is only in User PATH, so the Machine-PATH shadowing refusal does not eliminate this scenario.
- **Effect:** The exact-version metadata guarantee is broken. If the older catalog lacks the selected preset, setup unnecessarily refuses despite an installed suitable native client; otherwise the catalog is validated/used by a different release than its source. This is a static control-flow finding; no mixed-install Windows runtime reproduction was performed here.
- **Fix:** Resolve the executable that will actually become bare Codex first, query that exact executable's version, and use that same identity for catalog download and staging. Add a focused resolver fixture with different standalone/npm versions to prevent the two precedence rules drifting apart.

### Minor

**M1 — Test instructions still promise an unchanged main config.**

- **Location:** `tests/README.md:28`, `tests/README.md:38`.
- **Scenario:** A contributor follows the statement that `config.toml` is preserved and existing base configuration survives byte-for-byte.
- **Effect:** The test documentation describes the preceding named-profile-only contract. Current setup intentionally changes main model/provider/context/tier settings; tests preserve unrelated semantics/comments and exact original backup bytes, not the entire live main file.
- **Fix:** Describe the main/named default transaction and distinguish preserved unrelated values from exact backup bytes. This is one duplicated false claim, counted once below.

**M2 — README's manual-tier statement omits the automatic Astra preset.**

- **Location:** `README.md:103`, `README.md:111`; implementation at `scripts/setup-codex.ps1:1023`.
- **Scenario:** A user supplies custom exact ID `gpt-6-astra`, no catalog and no supported-tier argument. README describes only Sol as automatic and says other models without catalogs use manual overrides; Astra actually receives Fast/Ultrafast defaults too.
- **Effect:** The user cannot predict tier advertisement from the public instructions for this supported preset. R8 authorizes the behavior, so the correction is documentation, not removal of Astra support.
- **Fix:** Name both exact automatic presets and limit the manual-override statement to other IDs.

## Seams checked and sound

- Complete catalog conversion retains the selected exact entry and other native metadata, supplies the exact `Ultrafast` name/id pair, normalizes Fast to `priority`, and does not invent GLM metadata (`scripts/setup-codex.ps1:202`). Fixed official download origin, semantic-version URL construction, redirect refusal, timeouts, strict UTF-8 and a 1 MiB bound are present (`scripts/setup-codex.ps1:164`). I2 concerns the version's executable identity, not URL construction.
- Generated cache files are immutable/hash-addressed, verified on reuse and refused on tamper; failed configuration updates may leave public metadata but do not overwrite the existing pointer (`scripts/setup-codex.ps1:261`).
- Both main/named snapshots derive text and comparison bytes from one read; named manual edits refuse, unrelated main TOML is merged natively in private staging, and changed files receive exact backups (`scripts/setup-codex.ps1:62`, `scripts/setup-codex.ps1:821`, `scripts/setup-codex.ps1:917`). Rechecks precede publication; reverse rollback handles existing/absent files, credentials and PATH, with correct null-string restore marshalling (`scripts/setup-codex.ps1:1121`, `scripts/setup-codex.ps1:1151`).
- Credential entry/environment storage and context/compaction bounds match the public contract. Omitted service tier explicitly resets to default; configuration does not claim to enlarge provider capacity (`scripts/setup-codex.ps1:993`, `scripts/setup-codex.ps1:1040`, `scripts/setup-codex.ps1:1056`).
- Stage protection precedes copying config; shutdown bounds wait/kill/reap and verifies captured child identities. Sharing-only warnings require validation, two successful stops, complete sticky tracking, removed config, GUID/root/reparse checks and a private DACL recheck. Uncertain or unsafe cleanup fails before live publication (`scripts/setup-codex.ps1:566`, `scripts/setup-codex.ps1:721`, `scripts/setup-codex.ps1:855`).
- SSH defaults to Private/LocalSubnet, preserves network classification, requires explicit Public/GitHub-key choices, refuses conflicting access/authentication/firewall policies, validates effective policy before authorization, and restores key/config bytes, ACLs, owned firewall rules and service state after late failure (`scripts/setup-ssh-server.ps1:30`, `scripts/setup-ssh-server.ps1:116`, `scripts/setup-ssh-server.ps1:232`, `scripts/setup-ssh-server.ps1:448`, `scripts/setup-ssh-server.ps1:495`). SYSTEM task cleanup confirms a stopped state before removing protected files (`scripts/setup-ssh-server.ps1:200`).
- Downloaded scriptblock and local-file entry points share the installer; optional SSH failure preserves working Codex. Pages deploy requires both shell-matrix and real SSH gates, stages both installers, then runs hosted text comparison and dry-run scriptblocks in both shell editions (`.github/workflows/pages.yml:57`, `.github/workflows/pages.yml:81`, `tests/public-download-smoke.ps1:4`).
- Reviewed source requires an operator-supplied endpoint and contains no private default route. The new official catalog fetch carries no API key. No source change was made during this review.

## Documentation claims checked

Counted **26 distinct behavioral/evidence claims; 4 wrong or incomplete**, with repeated wording counted once. Historical test counts are treated as historical evidence rather than invented current failures.

| # | Claim/source | Result |
| --- | --- | --- |
| 1 | README: downloaded scriptblock and local standalone invocation | Checked |
| 2 | README: Windows PS5.1/PS7 support | Checked against supplied native gates |
| 3 | README: offered IDs/custom exact ID | Checked |
| 4 | README: endpoint/private key prompt, no bundled deployment endpoint | Checked |
| 5 | README: npm install/dependency instructions | Checked on source; winget runtime excluded below |
| 6 | README: bare default plus optional named profile | Checked |
| 7 | README: unrelated main settings and changed-file backups | Checked |
| 8 | README: manually changed named profile refusal | Checked |
| 9 | README: environment credential, no key in generated TOML | Checked |
| 10 | README/site: explicit context budget, 90% compaction, no capacity expansion | Checked |
| 11 | README: dry run does not change software/config/credentials | Checked |
| 12 | README: discovery sends bearer only to configured endpoint, refuses redirects | Checked |
| 13 | README: supplied catalog path/exact slug and source precedence | Checked |
| 14 | README: automatic catalog matches the client release used | **Wrong for mixed installations — I2** |
| 15 | README/site/research: exact `/ultrafast` command metadata | Checked, including refreshed TUI artifacts |
| 16 | README: explicit supported tiers restrict advertisement | **Wrong with supplied catalog — I1** |
| 17 | README: other models without catalogs require manual tier override | **Incomplete for Astra — M2** |
| 18 | README: Fast maps to priority; omitted selection resets default | Checked |
| 19 | README: content-addressed local catalog | Checked |
| 20 | Verification: protected auxiliary residue only under narrow guards | Checked |
| 21 | README/SSH guide: Private/LocalSubnet, explicit Public, no reclassification | Checked |
| 22 | SSH guide: pasted/file or approved GitHub public-key snapshot | Checked |
| 23 | SSH guide: policy refusal and authorization/service rollback | Checked against source and supplied native evidence |
| 24 | README/verification: Windows gates before deploy, hosted script comparison | Checked against workflow and supplied successful run |
| 25 | Verification: current 76/0, 87/0 and retrospective four-case RED/GREEN | Checked against local logs; native results supplied separately |
| 26 | Test README: live main configuration remains byte-for-byte unchanged | **Wrong — M1** |

## Declined to judge

- **Gateway inference/entitlement/capacity/performance:** no real key or request was used; local metadata/TUI behavior cannot establish these. Under R8, preset metadata is an explicitly authorized route assumption with public disclosure, not discovery. The earlier focused I1 is closed on that basis; finding I1 in this report is the different explicit-restriction interaction.
- **Personally reproduced Windows behavior:** native PS5.1/PS7, ACL/junction/sharing semantics, UAC, capability/account/service/firewall operations and real login are supported by the supplied completed CI evidence, not executions on this Linux review host. No separate LAN-client reachability or native lock-fallback fault-injection result is claimed.
- **Winget installation and persistent User-environment failure rollback at runtime:** source inspected, but the documented CI runner already has Node and tests skip persistent user-key mutation. No production defect was inferred solely from those coverage limits.
- **Every possible supplied catalog:** operator-supplied catalogs must be complete and version-compatible. The incomplete-catalog probe demonstrated fallback, not a startup crash; it is not evidence that arbitrary partial catalogs are supported. Complete Sol/Astra native evidence is much stronger and is assessed above.
- **Every Codex release/custom build:** inspected the exact-version mechanism and 0.165/0.166 evidence; did not fetch or test all releases or untagged custom builds. I2 identifies a concrete resolver mismatch independently of that broader matrix.
- **Exhaustive private-string history/publication audit and live CI requery:** the coordinator records the history scan, public hashes and completed run. Reviewed source contains no private default endpoint; no operator configuration/keys were read and no source was changed. The coordinator's exhaustive historical scan and remote execution claims were not independently repeated.
- **Earlier resolved findings, historical RED process and broader product work:** checked their relevant final seams, without reopening fixed issues absent contrary evidence, inventing pre-implementation RED timing, or reviewing deferred Go/TUI/services/multi-client work. No redesign or subagent review was performed.
- **Visual/browser behavior:** site behavioral text/command inspected; previous screenshots/browser evidence was not rerun because this review made no visual changes.

## Assessment

**Ready to publish: With fixes.** Native/publication gates are recorded green and the generated preset TUI behavior is now supported by clean evidence. Release-review approval still requires I1 and I2 to be resolved and relevant focused regression checks to pass; M1/M2 are bounded documentation corrections. No source/test fixes were made by this reviewer.

=== DONE ===
