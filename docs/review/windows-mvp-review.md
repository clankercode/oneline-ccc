# Independent Windows Codex MVP review

Reviewed 2026-11-02 against the MVP plan, product brief, and
ultra-code-standards. Scope: installer, isolated harness, README, and Pages
workflow. Reviewer changed this report only, apart from isolated cache artifacts.

## Current status

**No unresolved Important or Critical findings in the reviewed installer/tests.**
All material review findings were corrected and independently rechecked where
portable execution allowed. Native Windows and hosted deployment limits remain
listed below; they are not observed defects.

## Resolved findings

- Prior credential-bearing URLs are now rejected by the shared
  `Get-ValidatedEndpoint` validator before backup or credential mutation.
  Independently tested query, userinfo, and fragment variants with dummy data:
  each refused safely, retained exact profile bytes and the old process key,
  created no backup/temporary files, and did not echo the dummy secret.
  Evidence: `.cache/review-old-url-final-5a0cac96cdaa4a18a04d21a62ba81608`.

- Credential mutation now snapshots previous User/Process values, prepares the
  temporary file before mutation, rechecks the profile, and restores credentials
  on commit failure. Independent early-directory-failure test retained the old
  process key; an injected failure after mutation restored it. Persistent User
  rollback was inspected, not natively executed. Evidence:
  `.cache/review-rollback-0c1e202332a44369b24ab6c53b800c79`.
- Windows command handling prefers `codex.cmd`, then `codex.exe`, avoiding the
  npm PowerShell shim. README uses the same commands.
- Manually edited owned profiles are refused rather than silently overwritten;
  independent repro retained the edited bytes. Base `config.toml` is untouched.
- Invalid-input tests now use an absent profile and assert the intended safe
  error. Malformed-fixture loop variables no longer suffer dynamic-scope
  shadowing. These are resolved coverage defects, not current issues.
- `ModelCatalogPath` now binds tier eligibility to exactly one case-sensitive
  model slug, validates tier array shapes, references root `model_catalog_json`,
  and preserves the supplied catalog. Explicit operator attestation remains
  available without a catalog. No capability inference is made from `/models`.
- An omitted tier now writes root `service_tier = 'default'` to avoid inheriting
  an unrelated base tier. The focused inherited-tier test passes; nondefault
  tiers still require exact capability eligibility.
- The installer prompts for the endpoint; no URL default remains. This report
  contains no private operator hostname.

## Validation and limits

Independent portable PowerShell 7 harness run: **60 passed, zero failed**.
Artifacts: `.cache/installer-tests-32c718c1f23049c28d4125d64e799eb3`.
Coverage includes downloaded-text scriptblock invocation, exact invalid-input
errors, catalog binding, reruns/backups/preservation, and loopback discovery with
bearer credentials, malformed responses, redirect refusal, and dry-run behavior.
No real keys, operator configuration, or paid inference were used.

Current profile structure agrees with the supplied Codex >=0.134.0 contract:
`llmp.config.toml`, root context/compaction fields, Responses transport, and an
environment credential reference. Context values are explicitly operator budgets.
Full provider-supplied catalog validity remains Codex's responsibility; minimal
fixtures do not prove a complete native catalog loads.

Native installation currently uses winget without forced user scope, warns about
possible administrator approval, refreshes machine/user PATH, checks `npm.cmd`,
and provides actionable failure instructions. Linux execution does not validate
Windows PowerShell 5.1, actual winget/npm installation, registry persistence, or
execution policies. These remain validation limits, not observed defects.

The Pages deploy job depends on both Windows shell verification jobs and stages
the same checkout's installer after success. Actual Windows CI results and the
hosted bootstrap must be checked before claiming a working public Windows
one-liner. Brief inspection of the completed site confirms the same scriptblock
command, private-key prompt guidance, Responses requirement, and cmd-based launch.
Desktop/mobile rendering was reported by the coordinator, not independently run
by this reviewer. Coordinator also reports strict actual Codex parsing and named
profile debug rendering passed. No unresolved major review finding blocks the
initial source push. Hosted
operation still requires successful Windows CI and deployment checks.
