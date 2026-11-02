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

# These are operator actions on this disposable runner, not helper behavior.
# Existing broad SSH rules are deliberately disabled so the helper can enforce
# its refusal to start under an independently broad firewall policy.
Get-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -ErrorAction SilentlyContinue | Disable-NetFirewallRule
Set-NetFirewallProfile -Profile Domain,Private,Public -Enabled True -DefaultInboundAction Block

# Read-only metadata for the packaged-app rule misclassified by the SSH audit.
# Do not dump environment variables, key material, or unrelated machine state.
$diagnosticRules = @(Get-NetFirewallRule -PolicyStore ActiveStore -Name 'Microsoft.AAD.BrokerPlugin_cw5n1h2txyewy-In-Allow-ServerCapability' -ErrorAction SilentlyContinue)
Write-Output 'BEGIN SSH firewall applicability diagnostic'
if (-not $diagnosticRules.Count) { Write-Output 'AAD BrokerPlugin diagnostic rule is absent on this runner.' }
foreach ($diagnosticRule in $diagnosticRules) {
    $diagnosticApplication = $diagnosticRule | Get-NetFirewallApplicationFilter
    $diagnosticPackageProperty = $diagnosticApplication.PSObject.Properties['Package']
    $diagnosticPackage = $null
    if ($null -ne $diagnosticPackageProperty) { $diagnosticPackage = $diagnosticPackageProperty.Value }
    [pscustomobject]@{
        Rule = @($diagnosticRule | Select-Object Name, Enabled, Direction, Action, Profile, Owner, PolicyStoreSourceType)
        Application = @($diagnosticApplication | Select-Object Program, Package)
        ApplicationType = @($diagnosticApplication.PSObject.TypeNames)
        ApplicationPropertyNames = @($diagnosticApplication.PSObject.Properties | Select-Object -ExpandProperty Name)
        AdaptedPackagePropertyPresent = ($null -ne $diagnosticPackageProperty)
        AdaptedPackageValue = $diagnosticPackage
        Service = @($diagnosticRule | Get-NetFirewallServiceFilter | Select-Object Service)
        Port = @($diagnosticRule | Get-NetFirewallPortFilter | Select-Object Protocol, LocalPort, RemotePort)
        Address = @($diagnosticRule | Get-NetFirewallAddressFilter | Select-Object LocalAddress, RemoteAddress)
        Security = @($diagnosticRule | Get-NetFirewallSecurityFilter | Select-Object Authentication, Encryption, LocalUserAuthorizedList, RemoteUserAuthorizedList, RemoteMachineAuthorizedList)
    } | ConvertTo-Json -Depth 6
}
Write-Output 'END SSH firewall applicability diagnostic'

& (Join-Path $repository 'scripts/setup-ssh-server.ps1') -UserName onelinefixture -CreateUser -PublicKeyFile ($keyPath + '.pub') -NonInteractive
$sshd = Join-Path $sshRoot 'sshd.exe'
& $sshd -t
if ($LASTEXITCODE -ne 0) { throw 'Installed sshd configuration is invalid' }
$effective = @(& $sshd -T -C 'user=onelinefixture,host=localhost,addr=127.0.0.1')
foreach ($setting in @('passwordauthentication no', 'authenticationmethods publickey', 'pubkeyauthentication yes')) {
    if ($effective -notcontains $setting) { throw 'Effective SSH authentication policy is incorrect' }
}
$rules = @(Get-NetFirewallRule -Name 'Oneline-CCC-SSH-*')
if ($rules.Count -ne 1 -or [string]$rules[0].Profile -ne 'Private') { throw 'Expected one Private SSH rule' }
$addresses = @($rules[0] | Get-NetFirewallAddressFilter | Select-Object -ExpandProperty RemoteAddress)
if ($addresses.Count -ne 1 -or $addresses[0] -ne 'LocalSubnet') { throw 'SSH rule permits sources beyond the local subnet' }
if ((Get-Service sshd).Status -ne 'Running') { throw 'SSH service is not running' }

# BatchMode prevents fallback to password prompts. Known-host state stays in cache.
$knownHosts = Join-Path $root 'known_hosts'
$previousErrorAction = $ErrorActionPreference
try {
    $ErrorActionPreference = 'Continue'
    $identity = @(& (Join-Path $sshRoot 'ssh.exe') -o BatchMode=yes -o ConnectTimeout=15 -o StrictHostKeyChecking=accept-new -o "UserKnownHostsFile=$knownHosts" -i $keyPath -l onelinefixture 127.0.0.1 whoami 2>$null)
    $sshExit = $LASTEXITCODE
} finally { $ErrorActionPreference = $previousErrorAction }
if ($sshExit -ne 0 -or ($identity -join '').Trim() -notmatch '\\onelinefixture$') { throw 'Real SSH public-key login failed' }
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
