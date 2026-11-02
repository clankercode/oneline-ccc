#requires -Version 5.1
# Run only on disposable Windows CI machines: this creates a local test account
# and configures the real OpenSSH service/firewall with a generated fixture key.
$ErrorActionPreference = 'Stop'
if ($env:GITHUB_ACTIONS -ne 'true' -or $env:OS -ne 'Windows_NT') { throw 'This integration test requires a disposable Windows GitHub runner.' }
$repository = Split-Path $PSScriptRoot -Parent
$root = Join-Path $repository '.cache/ssh-native'
[IO.Directory]::CreateDirectory($root) | Out-Null
$keyPath = Join-Path $root 'fixture_ed25519'
$sshRoot = Join-Path $env:WINDIR 'System32\OpenSSH'
function New-FixtureKey([string]$Path) {
    $keygen = New-Object Diagnostics.ProcessStartInfo
    $keygen.FileName = Join-Path $sshRoot 'ssh-keygen.exe'
    $keygen.Arguments = '-q -t ed25519 -f "' + $Path + '" -N ""'
    $keygen.UseShellExecute = $false
    $process = [Diagnostics.Process]::Start($keygen)
    try { $process.WaitForExit(); if ($process.ExitCode -ne 0) { throw 'Fixture key generation failed' } }
    finally { $process.Dispose() }
}
New-FixtureKey $keyPath

# Operator preparation ONLY on this disposable Windows GitHub runner.
# Disable only the known broad OpenSSH default in its writable PersistentStore.
# Other app rules remain intact; the helper must inspect their ActiveStore filters.
$baselineSshRule = Get-NetFirewallRule -PolicyStore PersistentStore -Name 'OpenSSH-Server-In-TCP' -ErrorAction SilentlyContinue
if ($baselineSshRule) { Disable-NetFirewallRule -PolicyStore PersistentStore -Name 'OpenSSH-Server-In-TCP' }
Set-NetFirewallProfile -Profile Domain,Private,Public -Enabled True -DefaultInboundAction Block

& (Join-Path $repository 'scripts/setup-ssh-server.ps1') -UserName onelinefixture -CreateUser -PublicKeyFile ($keyPath + '.pub') -NonInteractive
$sshd = Join-Path $sshRoot 'sshd.exe'
& $sshd -t
if ($LASTEXITCODE -ne 0) { throw 'Installed sshd configuration is invalid' }
# Reuse the exact production probe: administrator -T cannot resolve a different
# user's group token, whereas the real sshd service and this probe run as SYSTEM.
$helperAst = [Management.Automation.Language.Parser]::ParseFile((Join-Path $repository 'scripts/setup-ssh-server.ps1'), [ref]$null, [ref]$null)
$probeDefinitions = @($helperAst.FindAll({ param($node)
    $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -in @('Set-RestrictedAcl', 'Get-SshEffectivePolicy')
}, $true))
if ($probeDefinitions.Count -ne 2) { throw 'Production effective-policy probe definitions are missing' }
$effective = @(& {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $sshRoot = Join-Path $env:ProgramData 'ssh'
    $utf8 = New-Object Text.UTF8Encoding($false)
    foreach ($definition in $probeDefinitions) { . ([scriptblock]::Create($definition.Extent.Text)) }
    Get-SshEffectivePolicy (Join-Path $sshRoot 'sshd_config') 'onelinefixture' (Get-LocalUser -Name onelinefixture).SID.Value
})
foreach ($setting in @('passwordauthentication no', 'authenticationmethods publickey', 'pubkeyauthentication yes')) {
    if ($effective -notcontains $setting) { throw 'Effective SSH authentication policy is incorrect' }
}
$rules = @(Get-NetFirewallRule -Name 'Oneline-CCC-SSH-*')
if ($rules.Count -ne 1 -or [string]$rules[0].Profile -ne 'Private') { throw 'Expected one Private SSH rule' }
$addresses = @($rules[0] | Get-NetFirewallAddressFilter -PolicyStore ActiveStore | Select-Object -ExpandProperty RemoteAddress)
if ($addresses.Count -ne 1 -or $addresses[0] -ne 'LocalSubnet') { throw 'SSH rule permits sources beyond the local subnet' }
if ((Get-Service sshd).Status -ne 'Running') { throw 'SSH service is not running' }

# BatchMode prevents fallback to password prompts. Known-host state stays in cache.
$knownHosts = Join-Path $root 'known_hosts'
$clientError = Join-Path $root 'ssh-client.err'
$previousErrorAction = $ErrorActionPreference
try {
    $ErrorActionPreference = 'Continue'
    $identity = @(& (Join-Path $sshRoot 'ssh.exe') -vv -o IdentitiesOnly=yes -o BatchMode=yes -o ConnectTimeout=15 -o StrictHostKeyChecking=accept-new -o "UserKnownHostsFile=$knownHosts" -i $keyPath -l onelinefixture 127.0.0.1 whoami 2>$clientError)
    $sshExit = $LASTEXITCODE
} finally { $ErrorActionPreference = $previousErrorAction }
if ($sshExit -ne 0 -or ($identity -join '').Trim() -notmatch '\\onelinefixture$') {
    Write-Output ("SSH login diagnostic: exit={0}, whoami={1}" -f $sshExit, (($identity -join '').Trim()))
    Write-Output ('SSH service identity: {0}' -f (Get-CimInstance Win32_Service -Filter "Name='sshd'").StartName)
    if (Test-Path -LiteralPath $clientError) {
        Get-Content -LiteralPath $clientError | Where-Object { $_ -match 'connect|authenticat|permission|offering public key|server accepts key|load key|bad permissions|host key verification|sign_and_send_pubkey|no mutual|warning|error' } | Write-Output
    }
    Get-WinEvent -LogName 'OpenSSH/Operational' -MaxEvents 10 -ErrorAction SilentlyContinue |
        Where-Object { $_.Message -match 'onelinefixture' -and $_.Message -match 'authentication|ownership|permission|publickey|invalid user|not allowed|unable|error' } |
        ForEach-Object { Write-Output ('OpenSSH event {0}: {1}' -f $_.Id, $_.Message) }
    throw 'Real SSH public-key login failed'
}
Write-Output 'Real SSH public-key login, user policy, running service, and local-subnet firewall verified.'

# Inject a late service failure into the real helper. New authorization must be
# rolled back even though the service was already running when setup started.
$secondKey = Join-Path $root 'rollback_ed25519'
New-FixtureKey $secondKey
$account = Get-LocalUser -Name onelinefixture
$profile = Get-CimInstance Win32_UserProfile | Where-Object SID -eq $account.SID.Value | Select-Object -First 1
$authorized = Join-Path $profile.LocalPath '.ssh\authorized_keys'
$config = Join-Path $env:ProgramData 'ssh\sshd_config'
$oldKeys = [Convert]::ToBase64String([IO.File]::ReadAllBytes($authorized))
$oldConfig = [Convert]::ToBase64String([IO.File]::ReadAllBytes($config))
$oldKeyAcl = (Get-Acl $authorized).Sddl
$oldConfigAcl = (Get-Acl $config).Sddl
$oldRuleNames = @(Get-NetFirewallRule -Name 'Oneline-CCC-SSH-*' | Select-Object -ExpandProperty Name | Sort-Object)
$script:restartCalls = 0
function Restart-Service {
    [CmdletBinding()]param([string]$Name)
    $script:restartCalls++
    if ($script:restartCalls -eq 1) { throw 'Forced fixture service failure' }
    Microsoft.PowerShell.Management\Restart-Service -Name $Name
}
$refused = $false
try {
    & (Join-Path $repository 'scripts/setup-ssh-server.ps1') -UserName onelinefixture -PublicKeyFile ($secondKey + '.pub') -NonInteractive
} catch { $refused = $true }
finally { Remove-Item Function:Restart-Service }
if (-not $refused -or $script:restartCalls -lt 1) { throw 'Late-failure regression did not reach the intended service boundary' }
if ([Convert]::ToBase64String([IO.File]::ReadAllBytes($authorized)) -cne $oldKeys -or (Get-Acl $authorized).Sddl -cne $oldKeyAcl) { throw 'Failed setup retained new authorization or changed key ACLs' }
if ([Convert]::ToBase64String([IO.File]::ReadAllBytes($config)) -cne $oldConfig -or (Get-Acl $config).Sddl -cne $oldConfigAcl) { throw 'Failed setup did not restore exact config and ACLs' }
$newRuleNames = @(Get-NetFirewallRule -Name 'Oneline-CCC-SSH-*' | Select-Object -ExpandProperty Name | Sort-Object)
if (($newRuleNames -join ',') -cne ($oldRuleNames -join ',') -or (Get-Service sshd).Status -ne 'Running') { throw 'Failed setup changed firewall rules or previous service state' }
Write-Output 'Late SSH setup failure restored authorization, config, ACLs, firewall rules, and running service.'

if (@(Get-ScheduledTask -TaskName 'Oneline-CCC-SSH-Probe-*' -ErrorAction SilentlyContinue).Count -or
    @(Get-ChildItem -LiteralPath (Join-Path $env:ProgramData 'ssh') -Filter 'Oneline-CCC-SSH-Probe-*').Count) {
    throw 'Temporary SYSTEM policy probe tasks or files remained after setup/rollback'
}
Write-Output 'Temporary SYSTEM policy probe tasks and files were removed.'
