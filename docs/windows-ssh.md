# Optional Windows SSH server

SSH is optional and separate from Codex installation. Run this helper only on the Windows computer you want to connect **to**, in native 64-bit Windows PowerShell 5.1 or newer, **Run as administrator** (UAC). Installation requires Windows OpenSSH capability support and Windows Update availability. A required reboot stops setup; reboot and rerun. The helper does not elevate itself or change network categories.

```powershell
.\setup-ssh-server.ps1 -UserName alice
```

The interactive helper lets you select an existing local account, or explicitly confirm creation of a dedicated **non-administrator** account. It offers either a pasted OpenSSH public key or a GitHub username. Use a simple local account name, maximum 20 characters; domain/Microsoft account identities are outside this MVP. The launcher must capture the intended username before UAC, because elevation with different credentials changes the current username.

Generate a key on your **client**, for example `ssh-keygen -t ed25519`. Keep its private key on that client. Supply only the `.pub` file contents to the server. No password authentication is enabled. A dedicated account receives a cryptographically generated strong password that is discarded, never printed or saved. Windows initializes its profile with `CreateProfile`; no password logon is required. This account still has normal local-user privileges; SSH is not an application sandbox.

For explicit unattended setup:

```powershell
.\setup-ssh-server.ps1 -NonInteractive -UserName alice -PublicKeyFile C:\Users\Operator\client.pub
.\setup-ssh-server.ps1 -NonInteractive -CreateUser -UserName sshuser -PublicKeyFile C:\Users\Operator\client.pub
.\setup-ssh-server.ps1 -NonInteractive -UserName alice -GitHubUser your-github-name
```

Choose exactly one of `-PublicKey`, `-PublicKeyFile`, and `-GitHubUser`. Pasted/file input accepts one key. GitHub imports all supported keys from `https://github.com/<username>.keys`, with no authentication and no redirects. The helper shows SHA256 fingerprints and the source; interactive import requires typing `YES`. Verify the GitHub account and fingerprints independently: possession of **any** imported private key grants access. `-NonInteractive -GitHubUser` is explicit authorization to import that account's keys. Imports are snapshots; later GitHub removal does not revoke the server's copied keys. Empty/malformed responses fail. Supported types: Ed25519, RSA, and NIST P-256/P-384/P-521 ECDSA. Private keys, key options and multiline pasted input are rejected.

## Local subnet and firewall

The default rule permits inbound TCP 22 only from Windows Firewall's `LocalSubnet` on the **Private** network profile. `-RemoteAddress` accepts only `LocalSubnet`. This includes directly attached subnets, potentially on multiple adapters; it is not a list of trusted individual devices.

If a connected network is **Public**, the helper warns. A trusted home network often should be Private; review that choice in Windows Settings yourself. The script never changes it. For explicit Public-profile access, use `-AllowPublicNetworkProfile`, or confirm the interactive Public prompt. This still permits only `LocalSubnet`. Domain profile is not enabled. No router port forwarding is needed or configured.

Effective firewall profiles must be enabled with default inbound Block; disabled/allow-by-default profiles cause a refusal. Rules with a proven package restriction do not apply to the desktop SSH server and are excluded from this audit. Windows can expose this as `PackageFamilyName` on the rule even when the associated application filter has a null `Package`; the helper checks both metadata fields. An app-like rule name alone does not prove that restriction: existing broad or ambiguous app rules require manual review, and the helper preserves them and refuses setup when they could broaden SSH access. Existing inbound rules that could allow SSH more broadly are reported with their profile/source and a remediation command, and setup **stops before starting/restarting sshd**. Review and explicitly narrow or disable those rules yourself, then rerun. Existing operator rules are never silently disabled or rewritten. Only the standard `OpenSSH-Server-In-TCP` rule newly created by capability installation during this invocation may be narrowed automatically. Other firewall policy (including Group Policy), outbound policy, network routing and an already-running server remain operator responsibilities.

The disposable Windows CI smoke explicitly disables only the known preexisting broad OpenSSH default rule in PersistentStore and enables default inbound Block before exercising setup, key login and rollback. Other app firewall rules remain intact. The smoke also explicitly removes only the runner's administrator-only `AllowGroups` restriction while preserving other settings. This access-policy preparation is confined to the disposable test runner. The helper explicitly reads associated firewall filters from ActiveStore so their application and package restrictions match the effective rules being audited. The production helper never disables preexisting rules.

Future work: explicit trusted subnet lists, Tailscale and WireGuard. Those are not implemented here.

## Keys, policy and preservation

For a standard user the helper uses the registered Windows profile's `.ssh\authorized_keys`. The directory and file have protected ACLs for that user, Administrators and SYSTEM. For an administrator it uses `%ProgramData%\ssh\administrators_authorized_keys`, with only Administrators and SYSTEM. **Administrator keys are shared across administrator accounts** under Windows' default Match policy; prefer a dedicated non-administrator account.

Existing keys and unrelated configuration are retained. Account markers use a lowercase username. An existing managed block must exactly match the generated policy; manual edits, malformed boundaries or duplicates cause refusal rather than being erased. The helper appends a marked `Match User` block enforcing `AuthenticationMethods publickey`, `PubkeyAuthentication yes`, and `PasswordAuthentication no` for the selected account. It refuses active Include directives, conflicting MFA/disabled-public-key policy and address/host-dependent Match rules rather than replacing established restrictions. `sshd.exe -t` validates a candidate and `sshd.exe -T -C` verifies the selected user's effective key-only authentication, standard key path and port before replacement. The final configuration is tested again before restart. Windows OpenSSH requires SYSTEM to resolve another account's group token during `-T`. For a different target account, the helper uses a temporary SYSTEM Scheduled Task solely for that probe. Its script and outputs live in a protected GUID directory under `%ProgramData%\ssh`, the completion marker is published atomically after output, and execution has a deadline. The helper confirms the task has stopped before unregistering it or removing its files; any probe or cleanup failure refuses setup. If stopping cannot be confirmed within the cleanup deadline, it preserves the task and protected files and reports the task name for manual inspection. The bounded probe can start and continue on battery power. Same-user or already-SYSTEM checks run directly. Task Scheduler must be available. Other accounts' authentication policy is not changed.

Timestamped backups of existing configuration and public authorized-key files are kept beside those files, with restricted ACLs. Host private keys are never read, copied or backed up; `ssh-keygen -A` creates missing host keys. Reparse-point paths are refused. Rerunning retains existing keys, creates backups and a separate narrowly scoped firewall rule; it does not skip all work on repeated runs. On failure during authorization/service changes, the helper restores the previous key/configuration bytes and ACLs, directory ACLs, and service startup/running state, and removes firewall rules created by this invocation. Rollback failures are explicitly reported; the service stays stopped if safe restoration cannot be confirmed. Installed capability, created account/profile, missing host keys and protected public-material backups may remain. Read the error and inspect the state before rerunning. A policy refusal before authorization does not stop an already running service.

## Connect and verify

After successful setup, `sshd` is running with Automatic startup. The helper prints local non-loopback addresses and connection commands:

```text
ssh -i <client-private-key> -l "alice" 192.168.1.25
```

Use an address on the intended Private LAN. Locally inspect the server's public host-key fingerprint (for example `ssh-keygen -lf C:\ProgramData\ssh\ssh_host_ed25519_key.pub`) and compare it with the client's first connection prompt. Verify a real key login from your client and that a password-only attempt is refused. A key does not override disabled users, account expiration or Windows logon policy. Existing active `AllowUsers`, `DenyUsers`, `AllowGroups` or `DenyGroups` directives cause the helper to refuse before setup mutations: this MVP cannot reliably evaluate every account/group access policy. Review and reconcile those restrictions manually; the helper never edits them or grants administrator membership.

## Offline plan and validation limits

```powershell
.\setup-ssh-server.ps1 -DryRun -NonInteractive -UserName alice
.\setup-ssh-server.ps1 -DryRun -NonInteractive -UserName alice -GitHubUser your-github-name
```

Dry-run works on non-Windows PowerShell without capability, account, service, firewall, elevation or network commands. It validates supplied usernames and public-key syntax, then returns the requested plan. GitHub dry-run validates only the handle and source intent; it does not fetch keys. It cannot establish actual Windows permissions, policy, capability availability or connectivity. Real setup requires elevated Windows; Linux checks do not verify Windows behavior.

## Microsoft references

Consulted live on 2026-11-02:

- [Install OpenSSH for Windows](https://learn.microsoft.com/en-us/windows-server/administration/openssh/openssh_install_firstuse): server capability, automatic service and installation-created firewall rule.
- [Key-based authentication](https://learn.microsoft.com/en-us/windows-server/administration/openssh/openssh_keymanagement): standard-user versus shared administrator key paths and restricted administrator ACLs.
- [OpenSSH server configuration](https://learn.microsoft.com/en-us/windows-server/administration/openssh/openssh-server-configuration): Windows configuration path and administrator Match behavior.
- [CreateProfile](https://learn.microsoft.com/en-us/windows/win32/api/userenv/nf-userenv-createprofile): initialize a local user's profile using its SID.

- [New-NetFirewallRule](https://learn.microsoft.com/en-us/powershell/module/netsecurity/new-netfirewallrule): Private/Public profile values and the LocalSubnet address keyword.

- [MS-FASP FW_RULE](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-fasp/8c008258-166d-46d4-9090-f2ffaa01be4b): the distinct `wszPackageFamilyName` field identifies the package family of a UWP application.
