# Bounded final fix pass

Status: DONE. I1, I2, M1 and M2 are addressed. Frozen commit:
`143006f23da4d69a0074add3dad96b9c49e194cd`
(`Fix native catalog tier restrictions and executable identity`).

- Worktree: `/home/xertrov/src/oneline-ccc/.worktrees/windows-mvp-final-fix`
- Branch: `fix/native-catalog-resolution`
- Base: `45acd35d12b305195d6f793c5baeb3004b40a1fc`
- Committed paths, exactly: `scripts/setup-codex.ps1`, `tests/run.ps1`,
  `README.md`, `tests/README.md`.
- Final tracked worktree status was clean; pre- and post-commit diff checks passed.
  No other source/doc files were changed. No subagents, publication, attention,
  branch switches, resets, rebases or amendments were used.

## Changes

I1: An explicit supported-tier list is checked against every advertised tier of
the supplied catalog's exact selected model, including legacy speed tiers.
`fast` and `priority` normalize to the same request tier on both sides. A catalog
advertising anything outside that explicit list is refused before configuration,
credential or PATH publication, including when no service tier is selected or the
explicit list is empty. The supplied catalog is never rewritten. The refusal
uses the existing safe diagnostic prefix rather than widening diagnostic output.

Four integrated refusal cases cover omitted/default selection, Fast selection,
Ultrafast selection and an empty restriction. They assert unchanged bytes of both
existing configuration files and the complete supplied catalog, unchanged directory
entries (no backup/cache/staging publication), and unchanged process credential and
PATH despite a different fixture credential being supplied. Two success cases
cover compatible `fast` and `priority` restrictions and unchanged source bytes.

I2: Setup resolves the eventual native executable first, then passes that exact
path to version verification and staging, retaining it for bare-command activation.
Catalog generation consumes that executable's verified version. The existing
npm-native selection precedence is unchanged. Generic-command fallback remains
available when there is no resolved native executable; no-command `SkipInstall`
still supports isolated configuration fixtures.

The differing-version regression models an older standalone selection at
`0.134.0` and an eventual npm-native selection at `0.166.0`. It executes the real
version-probe code against distinct executable fixtures and records the catalog
version and staging/bare activation paths. It requires the selected path, version
`0.166.0`, and no invocation of the older executable. Boundary interception avoids
host PATH mutation. The existing stderr, nonzero-exit and malformed-version
fixtures remain intact and pass; the no-command fixture was made explicit.

M1: Test documentation now describes the main/named transaction, preserved
unrelated semantics/comments, and exact original backup bytes rather than promising
unchanged live main-file bytes. M2: Public documentation names both exact automatic
Sol/Astra presets, limits manual-tier instructions to other IDs, and explains
combined catalog/list refusal and alias normalization. No capability metadata,
backup/rollback or cleanup guards were changed.

## Commands and evidence

All commands below ran from the isolated worktree. PowerShell was the requested
parent portable binary, version `7.6.9`; Python was `3.13.12`. Temporary-directory
variables pointed into this worktree's `.cache/final-fix`:

```sh
export TMPDIR="$PWD/.cache/final-fix" TEMP="$PWD/.cache/final-fix" TMP="$PWD/.cache/final-fix"
```

The focused command ran before any production source edit, then again after the
fix. RED/GREEN timing is prospective, not reconstructed from a later checkout:

```sh
/home/xertrov/src/oneline-ccc/.cache/powershell/pwsh -NoProfile -File tests/run.ps1 -TestFilter 'final fix:*' > .cache/final-fix/focused-red.log 2>&1
/home/xertrov/src/oneline-ccc/.cache/powershell/pwsh -NoProfile -File tests/run.ps1 -TestFilter 'final fix:*' > .cache/final-fix/focused-green.log 2>&1
```

RED exited 1: **3 passed, 5 failed**. Excerpts:

```text
FAIL final fix: mixed native selection uses exact path and version for catalog and staging: Catalog used standalone 0.134.0 instead of selected native 0.166.0
FAIL final fix: supplied catalog refuses excluded advertisement with default: Invalid input unexpectedly succeeded
FAIL final fix: supplied catalog refuses excluded advertisement with fast: Invalid input unexpectedly succeeded
FAIL final fix: supplied catalog refuses excluded advertisement with ultrafast: Invalid input unexpectedly succeeded
FAIL final fix: supplied catalog refuses excluded advertisement with empty-restriction: Invalid input unexpectedly succeeded
Results: 3 passed; 5 failed.
```

GREEN exited 0: **8 passed, 0 failed**. Excerpts:

```text
PASS final fix: mixed native selection uses exact path and version for catalog and staging
PASS final fix: SkipInstall without any native command still writes isolated config
PASS final fix: supplied catalog refuses excluded advertisement with default
PASS final fix: supplied catalog refuses excluded advertisement with fast
PASS final fix: supplied catalog refuses excluded advertisement with ultrafast
PASS final fix: supplied catalog refuses excluded advertisement with empty-restriction
PASS final fix: supplied catalog accepts normalized fast restriction
PASS final fix: supplied catalog accepts normalized priority restriction
Results: 8 passed; 0 failed.
```

The intermediate post-edit check is retained as `focused-first-check.log`:
**4 passed, 4 failed**, because the existing safe-message filter hid the new
refusal diagnostic. Reusing its existing `Model catalog invalid` prefix resolved
those failures before the final focused GREEN and full gates.

One full gate run was used (below the limit of two); all three commands exited 0:

```sh
/home/xertrov/src/oneline-ccc/.cache/powershell/pwsh -NoProfile -File tests/run.ps1 > .cache/final-fix/core.log 2>&1
/home/xertrov/src/oneline-ccc/.cache/powershell/pwsh -NoProfile -File tests/ssh.ps1 > .cache/final-fix/ssh.log 2>&1
/home/xertrov/src/oneline-ccc/.cache/powershell/pwsh -NoProfile -File tests/native-smoke.ps1 > .cache/final-fix/native-smoke.log 2>&1
git diff --check
git diff --cached --check
git diff --check HEAD^
```

```text
core.log: Results: 84 passed; 0 failed.
ssh.log: Results: 87 passed; 0 failed.
native-smoke.log: PASS bare codex loads the LLMP default without --profile
```

Core count is the baseline 76 plus eight regressions. `gates.status` records
`core_exit=0`, `ssh_exit=0`, `native_smoke_exit=0`. Core includes all three existing
native-version cases and the full complete-catalog/native API checks. Logs and
fixture directories remain under this worktree's `.cache`; no scratch used `/tmp`.
No real credentials, operator configuration, private endpoint or inference request
were used. The catalog tests download complete official public metadata.

## Limits and handoff

No known unresolved issue remains within this bounded pass. This execution host
is Linux: native Windows PowerShell 5.1/7 behavior and a real mixed Windows
installation were not rerun here. The mixed-install regression models resolver
outputs and observes actual version probing plus staging/publication boundaries;
it does not claim to execute Windows npm discovery or PATH persistence. New code
retains PowerShell 5.1-compatible syntax and the existing version-test seams.
The coordinator's merged gate and Hubble's scoped re-review remain separate
downstream checks. The frozen SHA was signaled immediately after commit.

This report was written after the commit, gates and final read-only checks, as
the last worktree write of this pass. No commits follow it.

=== DONE ===
