# Windows installer executable tests

Run from the repository root with Python 3.11+ and either Windows PowerShell
5.1 or PowerShell 7:

```powershell
./tests/run.ps1
# Optional Python executable selection:
./tests/run.ps1 -Python python3
```

Portable Linux execution used during development:

```sh
.cache/powershell/pwsh -NoProfile -File tests/run.ps1
```

The harness invokes the real `scripts/setup-codex.ps1` in child PowerShell
processes with `-NonInteractive -SkipInstall -SkipPersistCredential`, isolated
`CODEX_HOME`, and dummy `LLMP_API_KEY`. Child temporary-directory variables
point into repository `.cache/`. It neither reads real user configuration nor
persists credentials to the Windows user environment. Each invocation has a
30-second deadline. Artifacts remain under `.cache/installer-tests-<unique-id>`
for inspection; the harness exits nonzero if any case fails.

Python's standard-library `tomllib` validates the actual generated
`llmp.config.toml` and checks semantic values independently of installer text.
Codex >=0.134 profiles use root keys in that file. `config.toml` is preserved.
No package install, inference request, or paid API call is performed. Optional model discovery uses `discovery_server.py`, a loopback-only fixture
that never forwards requests and records request paths without credentials.
The separate loopback URL-validation test uses port 9 without contacting it.

## Acceptance coverage

- Fresh TOML with root model/provider/context/compaction fields, Responses API,
  environment authentication, and no embedded dummy credential or output leak.
- Byte-stable reruns without duplicate tables or redundant backups.
- Existing base config with root settings and nested project/profile tables
  survives byte-for-byte. Existing unmanaged profile files are refused unchanged.
- Changed managed profiles receive byte-for-byte backups and updated controls.
- Default compaction, minimum context boundary, explicit service tier capability.
- Dry run leaves an absent configuration directory absent and works without
  credentials. Missing authentication fails noninteractive setup unchanged.
- Invalid URL schemes, remote HTTP, embedded URL credentials, query/fragment,
  model delimiters/control characters/missing model, context and compaction
  limits, missing endpoint, and unsupported service tier fail without file mutations.
- Scriptblock bootstrap executes the same parameter interface. Child PowerShell
  explicitly uses `ExecutionPolicy Bypass`.
- Offline discovery exercises successful bearer authentication, ignores claimed
  capability metadata, rejects redirects/malformed JSON/missing or invalid IDs,
  and skips all requests during dry run.
- Native catalogue fixtures check root path, unchanged source, exact model slug
  tier binding, legacy speed tiers, and refusal of missing/malformed catalogues.
- Missing, reversed, nested, duplicate, and misplaced managed markers, unmanaged
  provider definitions, and inline credential settings fail closed without
  backups or scratch files.

## Validation status

60 cases passed locally with portable PowerShell 7.6.9 and Python `tomllib`.
The final run includes catalogue fixtures, scriptblock bootstrap, missing
endpoint/authentication checks, and explicit absence assertions for unintended
legacy profile tables or optional settings. Additional regression cases cover
unsafe previous-profile URLs and resetting an inherited nondefault tier.
`.github/workflows/windows.yml` runs this harness independently using native
Windows PowerShell 5.1 and PowerShell 7. Those Windows jobs have not been run
locally; CI results remain pending. Real Codex runtime profile selection and
strict startup validation are being verified separately by the parent agent;
this harness checks installer behavior and TOML semantics.
