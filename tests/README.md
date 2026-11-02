# Windows installer executable tests

Run from the repository root with Python 3.11+ and either Windows PowerShell
5.1 or PowerShell 7:

```powershell
./tests/run.ps1
# Optional Python executable selection:
./tests/run.ps1 -Python python3
# Focused final-fix regressions:
./tests/run.ps1 -TestFilter 'final fix:*'
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
`config.toml` and `llmp.config.toml` and checks semantic values independently of
installer text. Setup updates both files in one transaction so bare Codex and the
named configuration select LLMP. Unrelated main-config semantics and comments are
preserved; each changed existing file receives an exact backup of its original bytes.
No package install, inference request, or paid API call is performed. Optional model discovery uses `discovery_server.py`, a loopback-only fixture
that never forwards requests and records request paths without credentials.
The separate loopback URL-validation test uses port 9 without contacting it.

## Acceptance coverage

- Fresh TOML with root model/provider/context/compaction fields, Responses API,
  environment authentication, and no embedded dummy credential or output leak.
- Byte-stable reruns without duplicate tables or redundant backups.
- Main and named configurations receive matching model/provider/context/tier
  controls together, with rollback if publication of either file fails.
- Unrelated main-config settings, nested project/profile tables and comments
  survive the update. Existing unmanaged named files are refused unchanged.
- Changed existing main and named files receive byte-for-byte original backups.
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
- Combined catalog/tier inputs refuse excluded advertisements before config,
  credential or PATH publication, preserving both config files and supplied bytes.
  Compatible `fast`/`priority` aliases succeed. A differing-version resolver
  fixture checks the selected executable's version, catalog and staging identity;
  version stderr/error fixtures and no-native `SkipInstall` remain covered.
- Missing, reversed, nested, duplicate, and misplaced managed markers, unmanaged
  provider definitions, and inline credential settings fail closed without
  backups or scratch files.

## Validation status

65 cases passed locally with portable PowerShell 7.6.9 and Python `tomllib`.
The final run includes catalogue fixtures, scriptblock bootstrap, missing
endpoint/authentication checks, and explicit absence assertions for unintended
legacy profile tables or optional settings. Additional regression cases cover
unsafe previous-profile URLs, resetting an inherited nondefault tier, and real
native-command shims emitting stderr during version checks.
`.github/workflows/windows.yml` runs this harness independently using native
Windows PowerShell 5.1 and PowerShell 7. The Codex suite, real npm installation,
profile loading, Pages deployment and hosted-script bootstrap all passed in both
shells in [the verified core run](https://github.com/clankercode/oneline-ccc/actions/runs/41384021098).
Local Codex strict startup parsing also passed. These checks use dummy
credentials and do not make inference requests.

`tests/ssh.ps1` checks optional SSH planning/validation and intercepted service,
firewall and authorization paths without changing the host. The separate
`tests/ssh-native-smoke.ps1` is guarded for disposable Windows GitHub runners:
it creates a test account, installs/configures OpenSSH, generates a fixture key,
authenticates over loopback, and injects a late failure to test rollback. Never
run that integration smoke on a workstation; its fixture resources are discarded
with the CI runner. SSH-specific final CI evidence is recorded in the verification
report.

The shared `run.ps1` harness also loads `rerun-defaults.ps1`. Its cases exercise
omitted inputs and scripted Enter responses, hidden process/User key selection,
active LLMP main-vs-profile precedence, unrelated-provider isolation, context
and tier changes, and read-only dry-run reuse. Sequence cases cover a native
arbitrary-TOML merge followed by another rerun, so a disabled Ultrafast capability
cannot return from stale main-file state. Python parses both configurations and
checks nested-key absence while unrelated comments and feature values survive.
Every invocation replaces the User-environment seam in its process-local source
copy with an absent or dummy key source, including when other fixtures are used;
tests never access a real saved User key or persist fixture keys.
