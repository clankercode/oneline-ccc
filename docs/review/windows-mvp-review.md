# Independent Windows Codex and optional SSH review

Final frozen-source review, 2026-11-02. Reviewed the MVP plan, product brief,
ultra-code-standards, installers, test helpers, and publication workflow.
Reviewer changed this report only, apart from isolated cache artifacts.

## Status

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
