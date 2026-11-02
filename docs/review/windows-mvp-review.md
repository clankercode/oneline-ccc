# Independent Windows Codex and optional SSH review

Final frozen-source review, 2026-11-02. Reviewed the MVP plan, product brief,
ultra-code-standards, installers, test helpers, and publication workflow.
Reviewer changed this report only, apart from isolated cache artifacts.

## Status

Current default-main-config targeted review is complete: the NullString fix
passed same-process rollback tests in all four original-file states. No unresolved
Important or Critical findings remain from this review. Native CI remains pending.

Scope update: the operator now requires LLMP for bare `codex`, including a
preserving merge into main `config.toml`. The previous profile-only clearance
below does not cover that forthcoming implementation. Two-file/credential
transaction and Windows bare-command resolution require a new review once ready.

**No unresolved Important or Critical findings in the reviewed frozen source.**
The material preservation, credential, policy, firewall, and transaction findings
were corrected. Source can proceed to native CI verification; this does not claim
native Windows or hosted deployment has passed.

## Independent evidence

- SSH harness: **71 passed, zero failed**, including mocked transaction rollback,
  policy conflicts, applicable firewall filtering, effective profile refusal,
  marker preservation, and repeat setup. Artifact directory:
  `.cache/ssh-tests-f162b240becf40caa0f7b25e77d12eaa`.
- Additional actual-source focused checks accepted the exact generated SSH block
  and rejected added DenyUsers, duplicate, missing, reversed, and nested markers.
  Final CRLF-specific edited-block refusal also passed after the line-ending fix.
- Actual-source effective firewall guard accepted enabled/default-Block policy;
  disabled, default-Allow, and missing policy were refused.
- Prior Codex harness independently passed **60/0**, including downloaded-text
  scriptblock, catalog binding, exact validation errors, unsafe prior URL refusal,
  preservation, and loopback discovery/redirect handling. Coordinator reports
  final frozen Codex harness **65/0** with optional SSH guards added.
- Independent credential failure injection retained/restored old process values.
  Unsafe prior URL query/userinfo/fragment variants refused without changing
  profile bytes, process credential, or creating backups/temp files. Dummy data
  only; no real API keys, operator config, or paid inference were used.

## Resolved material findings

- Codex credentials snapshot and restore previous User/Process values on commit
  failure; file preparation and concurrency checks precede final commit.
- Existing credential-bearing URLs are validated before backups. Generated
  profiles reference environment credentials and use the current separate-file
  profile contract with root context/compaction/catalog fields.
- Windows command handling chooses `codex.cmd` or `codex.exe`, avoiding `.ps1`
  execution-policy shims. Endpoint input is required; no private hostname/default
  endpoint is bundled.
- Edited Codex-owned profiles refuse without losing user additions. Invalid-input
  tests now assert the intended validation failure against an absent profile.
- Catalog tier binding selects one exact model slug and checks eligibility
  case-sensitively. An omitted tier explicitly resets inherited policy to default.
- SSH integration rejects unsupported/empty original accounts before download/UAC
  and always forwards a validated original account. Optional setup defaults No;
  explicit noninteractive SSH refuses before Codex mutation. SSH failure leaves
  working Codex configured and provides a safe warning.
- SSH refuses Include, unsupported conditional Match policies, and conflicting
  authentication restrictions. Managed blocks must match exact generated shape;
  edited/malformed/duplicate blocks refuse, including CRLF input. Marker accounts
  are normalized before construction.
- Firewall checks fail closed on disabled/unknown/default-Allow effective profile
  policy and broader applicable inbound rules before authorization/service start.
  Rule applicability checks TCP/Any protocol, port 22/Any/ranges, application,
  service, profile, and remote address, avoiding unrelated browser/service/UDP
  false positives. Production defaults remain Private plus LocalSubnet; Public
  requires explicit consent and the helper does not change network categories.
- Candidate policy and narrow firewall are verified before authentication writes.
  Rollback restores key/config bytes and ACLs, existing `.ssh` DIRECTORY ACL,
  startup/running state, and removes invocation-created firewall rules. Incomplete
  rollback is reported explicitly. Existing unrelated authorization is preserved.

## Native smoke and deployment review

`tests/ssh-native-smoke.ps1` refuses outside Windows with GITHUB_ACTIONS=true.
Only that guarded disposable test enables default-Block firewall profiles and
disables the baseline OpenSSH rule as fixture preparation; production helper
refuses unsafe operator policy instead of silently changing it.

The smoke creates fixture keys and a dedicated account, verifies real loopback
public-key login, effective authentication policy, service state, and the narrow
firewall rule. It injects a late Restart-Service failure and asserts exact prior
key/config bytes and file ACLs, owned rule names, and running state. This tests a
meaningful failure boundary. Directory ACL restoration exists in the helper;
the native smoke currently asserts file ACLs rather than separate directory or
unrelated-child SDDL. That remains a coverage limitation, not an observed defect.

Pages deployment depends on both the Windows shell matrix and real verify-ssh
job. Both installers are staged. The public-download smoke compares both hosted
files with the checkout before evaluating scriptblocks under both shell editions.
The site command and guidance agree with the installers on inspection. Visual
rendering and actual Codex parser/debug checks were reported by the coordinator.

## Validation limits

Linux PowerShell and mocked Windows cmdlets cannot establish real ACL semantics,
UAC, registry persistence, capability installation, packet reachability, or native
PowerShell 5.1 behavior. Those require the guarded Windows CI jobs. Loopback login
plus firewall inspection does not itself prove access from a separate LAN client.
GitHub key import is an explicit snapshot/authorization choice, not continuous
synchronization. No claim of native success or working hosted deployment is made
before the corresponding CI and public-download checks pass.

## Narrow CI repair review

Reviewed the packaged-application firewall filter exemption and reconstructed
test-body parser changes only. No new Important or Critical findings.

Rules restricted to a nonempty, non-Any application Package do not apply to the
unpackaged desktop sshd process and are excluded appropriately. Missing/Any
Package still receives the existing application/service/port/profile/address
audit, preserving detection of actual broad SSH allows. The new packaged-app
fixture passes alongside the genuine Any/sshd conflict fixtures.

The harness now parses and slices the identical full source string, checks every
replacement's original extent before editing, saves source-only reconstruction
for diagnostics, and exposes ParseOnly covering both original and reconstructed
parsing. No fixture secrets are embedded in these diagnostic source excerpts.

Independent portable PowerShell verification: ParseOnly **3/0** and full SSH
harness **72/0**. Latest artifacts:
`.cache/ssh-tests-cee30891ae484effa9325e0147b15bf8`.
These results do not substitute for the upcoming Windows PowerShell 5.1 CI run.

## Default main-config transaction review

Independent actual Codex app-server merge preserved unrelated feature, provider,
project trust settings, and comments while selecting LLMP at the root and writing
the optional profile. No inference or real credentials were used. Evidence:
`.cache/review-default-9dceaa44270e455ab7554b8facd4f09f`.

Parent-reported snapshot race is fixed on inspection: both decoded text and
comparison bytes come from one pre-merge snapshot per file. Prospective native
PATH now checks combined Machine/User resolution before persistence and refuses
fresh-terminal shadowing. Native PATH execution remains Windows CI evidence.

### Resolved: second-file commit failure could not restore the first file

Location: transaction rollback, `File.Replace($file.Backup, $file.Path, $null)`.

Independent runtime fault injection before profile publication, after successful
main publication, returned rollback failure and left main config changed. Profile,
old process key, and PATH were preserved. Directly invoking the restore showed
`The value cannot be an empty string (path)`: PowerShell converts the null string
argument to an empty string. Direct `[NullString]::Value` instead restored the
same isolated dummy file successfully. Reported to author and test owner.

Suggested fix: use correct null-string marshalling or another atomic restore
strategy. Add second-file failure tests for preexisting and initially absent
originals, asserting both exact bytes/absence and credential/PATH restoration.
Evidence: `.cache/review-twofile-e7b0593c7ac04255a23c9043df46616d`
(manually restored only after reproducing and diagnosing the failure).

## Final NullString closure

Current source restores existing files with `[NullString]::Value`. Independent
same-process injection failed the second file publish after the actual first
commit. All four states passed: both originals existing, both absent, main only,
and profile only. In each case both exact original bytes/absence, old process
credential, and process PATH were restored without a rollback-failure diagnostic.

Evidence: `.cache/review-nullstring-final-71712e83e978419c858d6f8b9f38f447`.
This closes the last Important default-config transaction finding. Verification
used portable PowerShell on Linux with dummy credentials; persistent Windows
registry/PATH behavior still requires native CI. No broader review round was run.

Final frozen-source confirmation: the tested NullString restore remains present.
Test owner reports 72/0 and native smoke success on Codex 0.166, including the
four rollback states; coordinator's independent 0.165 run subsequently passed 72/0 as well. No major review findings remain pending.

## Targeted staging cleanup and UTF-8 review

Reviewed only the new cleanup helpers and bounded Console.InputEncoding change.
No new Important or Critical finding. Cleanup requires an immediate GUID-named
child of the supplied scratch root, refuses a reparse stage root, removes child
links without recursion through their targets, normalizes read-only ordinary
entries, and retries with bounded delays. Failure categories expose no paths or
native exception payload. Cleanup runs before live configuration commit.

Independent actual-helper checks removed a stage containing file/directory
symlinks while preserving the external sentinel, refused a root symlink, and
refused a nonstage external directory. Evidence:
`.cache/review-stage-e3b3c6559e7542528d71bc34237f5b60`.
These are Linux filesystem checks, not native Windows junction/ACL validation.

Console input encoding is changed only around Process.Start and restored in a
finally block, with the explicit stdin writer retaining BOM-free UTF-8. Failure
still reaches existing process shutdown and staging cleanup. Actual .NET
Framework stdin/BOM behavior remains a Windows CI result to collect.

## Targeted shutdown and private staging review

Reviewed only the latest shutdown, staging-permission, and cleanup-diagnostic
changes. No new Important or Critical findings. A close/flush exception no longer
skips bounded wait/kill/reap; redirected output/error streams close in finally.
Caller cleanup also runs if shutdown throws. New stage permissions are restricted
before existing config is copied. Cleanup diagnostics select fixed categories
and never expose the native path or exception payload.

Independent actual-function fault injection confirmed a throwing input Close
still executes underlying-stream close, wait, kill, second wait/reap, output/error
close, and process disposal in that order. Actual POSIX staging helper produced
mode 0700. Evidence:
`.cache/review-private-stage-242659fa72a64a599570da7b86846b72`.
Windows ACL/.NET Framework sharing behavior remains native CI evidence; no broad
review or test round was run.

## SYSTEM SSH effective-policy probe review

Privileged probe targeted review is now complete; the cleanup finding below is resolved. Protected directory,
wrapper/output ACLs, fixed executable, individually quoted file arguments, and
validated account context look sound on inspection. Candidate policy is already
restricted before use. The registered task has no trigger and a one-minute
execution limit; completion polling is bounded and happens before authorization.

### Resolved: task shutdown was requested but not confirmed before file removal

Location: `Get-SshEffectivePolicy` finally block.

The current code checks Running once, requests Stop, immediately unregisters,
then deletes wrapper/output files. It never observes that shutdown completed;
Queued state is omitted. A successful stop request is not evidence that no
SYSTEM action can still use those paths. This conflicts with the stated cleanup
invariant and leaves uncertain process/file lifetime on timeout.

Suggested fix: request stop for the owned temporary task, boundedly confirm it
has left Running/Queued before unregistering/deleting files, and retain protected
files with an explicit cleanup error if termination cannot be proved. Add delayed
stop and queued-state fixture checks. Native task/login verification remains CI
evidence; no SYSTEM task was executed by this reviewer on Linux.

## Final SYSTEM-probe cleanup closure

Frozen cleanup now handles Running and Queued, requests Stop, polls up to ten
seconds, and requires Ready/Disabled before unregistering or deleting protected
probe files. Failed or unconfirmed shutdown preserves those files/task and
returns an explicit cleanup error before authorization. This closes the finding.

Independent current SSH harness: **83 passed, zero failed**, including delayed
stop confirmation and stop-failure preservation/refusal. Evidence:
`.cache/ssh-tests-21da45c1e6dd433186affe4f6274db43`.
No unresolved Important or Critical finding from this focused review. Actual
SYSTEM scheduled-task and login execution remains the next native CI validation;
this reviewer executed mocks on Linux only.

## Final working-directory and diagnostic delta

Focused review found no new Important or Critical issue. App-server CWD now uses
the stable scratch parent while CODEX_HOME still selects the private GUID stage;
both initial and fresh-validation processes use this pairing. Cleanup diagnostics
expose fixed categories, counts, and whitelisted process/transport kinds, not
paths, command lines, or credentials. Targeted parent-PID CIM snapshots only
observe processes; they do not kill descendants. Start-time comparison reduces
PID-reuse misclassification in later diagnostic counts.

Independent actual Stop-function injection made both descendant diagnostics and
input Close throw. Shutdown still performed two waits, kill/reap, and disposal,
while marking the snapshot partial. No broader test round or native CIM execution
was run. Existing native Windows CI remains the runtime verification gate.

## Captured-descendant shutdown and SSH access-list delta

Focused review found no new Important or Critical issue. Discovery pins each
parent/child process, rechecks CIM parentage while pinned, requires child start
time at least the captured parent's, and bounds traversal by count/time. Shutdown
uses only captured records, reopens/pins each process, requires exact start-time
identity, and terminates deepest first with bounded waits. No name-only, reused
PID, or broad system-process killing is introduced.

Independent controlled-process test of the actual Stop function killed/reaped an
exact captured identity while leaving a deliberately mismatched start-time record
alive. Both test processes were created and cleaned up by this isolated test.
This checks the kill identity boundary on Linux; Windows handle/CIM enumeration
semantics remain native CI evidence.

SSH now refuses active AllowUsers/DenyUsers/AllowGroups/DenyGroups before setup
without rewriting operator access policy. The guarded disposable CI fixture
explicitly removes runner AllowGroups restrictions; that operation is not in
production helper code. Existing access lists cannot silently be broadened by
this new refusal path. Shared managed-profile recognizer extraction in the same
diff preserves the previous exact grammar on inspection.
