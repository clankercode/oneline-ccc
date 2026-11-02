#Requires -Version 5.1
<#
.SYNOPSIS
Optional, elevated Windows OpenSSH server setup for the local subnet.
.DESCRIPTION
Run -DryRun -NonInteractive on any platform to inspect the plan without OS commands.
Real execution requires native 64-bit Windows PowerShell 5.1+ or PowerShell 7.
#>
[CmdletBinding()]
param(
    [string]$UserName = $env:USERNAME,
    [string]$PublicKey,
    [string]$PublicKeyFile,
    [string]$GitHubUser,
    [switch]$CreateUser,
    [switch]$AllowPublicNetworkProfile,
    [ValidateSet('LocalSubnet')][string]$RemoteAddress = 'LocalSubnet',
    [switch]$DryRun,
    [switch]$NonInteractive
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if (@(@($PublicKey, $PublicKeyFile, $GitHubUser) | Where-Object { $_ }).Count -gt 1) { throw 'Specify only one of PublicKey, PublicKeyFile or GitHubUser.' }
if ($GitHubUser -and $GitHubUser -notmatch '^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,37}[a-zA-Z0-9])?$') { throw 'Invalid GitHub username.' }
if ($UserName -and $UserName -notmatch '^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,19}$') {
    throw 'UserName must be a simple local account name (1-20 letters, digits, dot, underscore or hyphen). Domain and Microsoft accounts are not supported.'
}
if ($PSBoundParameters.ContainsKey('UserName') -and -not $UserName) { throw 'UserName cannot be empty.' }
if ($PSBoundParameters.ContainsKey('PublicKey') -and -not $PublicKey -and -not $PublicKeyFile -and -not $GitHubUser) { throw 'PublicKey cannot be empty.' }
$profiles = @('Private')
if ($AllowPublicNetworkProfile) { $profiles += 'Public' }
function Assert-PublicKeys([string[]]$Keys) {
foreach ($key in $Keys) {
    if ($key -match '[\r\n]' -or $key -notmatch '^(ssh-ed25519|ssh-rsa|ecdsa-sha2-nistp(256|384|521)) [A-Za-z0-9+/]+={0,3}( [^\r\n]*)?$') {
        throw 'Supply valid OpenSSH public keys. Key options, private keys and multiline pasted input are not accepted.'
    }
    try {
        $parts = $key.Split(' ')
        $keyBytes = [Convert]::FromBase64String($parts[1])
        $typeLength = ([int]$keyBytes[0] * 16777216) + ([int]$keyBytes[1] * 65536) + ([int]$keyBytes[2] * 256) + [int]$keyBytes[3]
        if ($typeLength -lt 1 -or $typeLength -gt ($keyBytes.Length - 4)) { throw 'Invalid key header.' }
        if ([Text.Encoding]::ASCII.GetString($keyBytes, 4, $typeLength) -ne $parts[0]) { throw 'Key type mismatch.' }
        $hash = [Security.Cryptography.SHA256]::Create()
        try { $fingerprint = [Convert]::ToBase64String($hash.ComputeHash($keyBytes)).TrimEnd('=') } finally { $hash.Dispose() }
        Write-Host ("Public key: {0} SHA256:{1}" -f $parts[0], $fingerprint)
    } catch { throw 'Invalid OpenSSH public-key encoding.' }
}
}
if ($PSBoundParameters.ContainsKey('GitHubUser') -and -not $GitHubUser) { throw 'GitHubUser cannot be empty.' }
if ($GitHubUser -match '--') { throw 'Invalid GitHub username.' }
if ($PublicKeyFile) { $PublicKey = [IO.File]::ReadAllText((Resolve-Path -LiteralPath $PublicKeyFile).Path).Trim() }
if ($PublicKey) { Assert-PublicKeys @($PublicKey) }
if ($DryRun) {
    [pscustomobject]@{
        DryRun = $true; UserName = $UserName; CreateUser = [bool]$CreateUser
        RemoteAddress = $RemoteAddress; NetworkProfiles = $profiles
        Authentication = 'publickey'; Port = 22; RequiresAdministrator = $true
        Capability = 'OpenSSH.Server~~~~0.0.1.0'; GitHubUser = $GitHubUser
        KeySource = $(if ($GitHubUser) { 'GitHub' } elseif ($PublicKeyFile) { 'File' } elseif ($PublicKey) { 'Pasted' } else { 'Prompt' })
    }
    return
}
if ($env:OS -ne 'Windows_NT') { throw 'Real setup requires Windows. Use -DryRun -NonInteractive for a platform-independent plan.' }
$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($identity)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw 'Administrator elevation required. Open PowerShell with Run as administrator (UAC), then run this script with -UserName for the intended local account. A launcher can use Start-Process powershell.exe -Verb RunAs.'
}
if (-not [Environment]::Is64BitProcess) { throw 'Use native 64-bit PowerShell so LocalAccounts and Windows capabilities are available.' }
if (-not $NonInteractive) {
    $answer = Read-Host "Existing local account [$UserName], or type a new account name"
    if ($answer) { $UserName = $answer }
}
if (-not $UserName -or $UserName -notmatch '^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,19}$') { throw 'Supply a simple local -UserName (maximum 20 characters).' }
$UserName = $UserName.ToLowerInvariant()
if (-not $NonInteractive -and -not $CreateUser -and -not (Get-LocalUser -Name $UserName -ErrorAction SilentlyContinue)) {
    if ((Read-Host "Create '$UserName' as a dedicated non-administrator with an unlogged random password? Type YES") -ceq 'YES') { $CreateUser = $true }
    else { throw 'Account creation was not authorized.' }
}
if (-not $PublicKey -and -not $PublicKeyFile -and -not $GitHubUser -and -not $NonInteractive) {
    $source = Read-Host 'Import GitHub public keys? Enter GitHub username, or press Enter to paste one public key'
    if ($source) { $GitHubUser = $source }
    else { $PublicKey = Read-Host 'Paste ONE OpenSSH public key (.pub contents, never a private key)' }
}
if ($GitHubUser) {
    if ($GitHubUser -match '--' -or $GitHubUser -notmatch '^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,37}[a-zA-Z0-9])?$') { throw 'Invalid GitHub username.' }
    # No credentials, redirects or ambient Windows authentication are sent.
    $request = [Net.HttpWebRequest]::Create("https://github.com/$GitHubUser.keys")
    $request.AllowAutoRedirect = $false
    $request.UseDefaultCredentials = $false
    $request.Timeout = 15000
    $request.ReadWriteTimeout = 15000
    $request.UserAgent = 'oneline-ccc-ssh-setup'
    $oldProtocol = [Net.ServicePointManager]::SecurityProtocol
    try {
        [Net.ServicePointManager]::SecurityProtocol = $oldProtocol -bor [Net.SecurityProtocolType]::Tls12
        $response = $request.GetResponse()
        try {
            if ([int]$response.StatusCode -ne 200) { throw 'GitHub key endpoint did not return HTTP 200 (redirects are forbidden).' }
            $reader = New-Object IO.StreamReader($response.GetResponseStream())
            try {
                $characters = New-Object char[] 1048577
                $count = $reader.ReadBlock($characters, 0, $characters.Length)
                if ($count -gt 1048576) { throw 'GitHub key response exceeds 1 MiB.' }
                $keyText = New-Object string($characters, 0, $count)
            } finally { $reader.Dispose() }
        } finally { $response.Dispose() }
    } finally { [Net.ServicePointManager]::SecurityProtocol = $oldProtocol }
    $publicKeys = @($keyText -split '\r?\n' | Where-Object { $_.Trim() } | ForEach-Object { $_.Trim() } | Select-Object -Unique)
} else {
    if ($PublicKeyFile) { $PublicKey = [IO.File]::ReadAllText((Resolve-Path -LiteralPath $PublicKeyFile).Path).Trim() }
    $publicKeys = @($PublicKey)
}
if (-not $publicKeys.Count -or -not $publicKeys[0]) { throw 'No public keys supplied. Use PublicKey, PublicKeyFile or GitHubUser.' }
Assert-PublicKeys $publicKeys
if ($GitHubUser) {
    Write-Host "Source: https://github.com/$GitHubUser.keys ($($publicKeys.Count) keys). Anyone holding ANY corresponding private key gains access. Confirm the account and fingerprints through a trusted channel. This is a snapshot, not automatic synchronization."
    if (-not $NonInteractive -and (Read-Host 'Authorize these GitHub keys? Type YES') -cne 'YES') { throw 'GitHub key authorization declined.' }
}
$networks = @(Get-NetConnectionProfile)
foreach ($network in $networks) {
    if ($network.NetworkCategory -eq 'Public') {
        Write-Warning "Network '$($network.Name)' is Public. A trusted home LAN often should be Private; this script never changes its category. Default SSH rule will not admit traffic on Public networks."
    }
}
if (-not $NonInteractive -and -not $AllowPublicNetworkProfile -and @($networks | Where-Object NetworkCategory -eq 'Public').Count) {
    if ((Read-Host 'Also allow SSH on Public profile, still LocalSubnet only? Type YES to opt in') -ceq 'YES') { $profiles += 'Public' }
}
function Set-RestrictedAcl([string]$Path, [string]$OwnerSid, [bool]$Directory) {
    if ((Get-Item -LiteralPath $Path -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw "Refusing reparse point: $Path" }
    if ($Directory) { $acl = New-Object Security.AccessControl.DirectorySecurity } else { $acl = New-Object Security.AccessControl.FileSecurity }
    $acl.SetAccessRuleProtection($true, $false)
    $acl.SetOwner((New-Object Security.Principal.SecurityIdentifier($OwnerSid)))
    $sids = @('S-1-5-18', 'S-1-5-32-544', $OwnerSid) | Select-Object -Unique
    foreach ($sid in $sids) {
        $account = New-Object Security.Principal.SecurityIdentifier($sid)
        if ($Directory) {
            $rule = New-Object Security.AccessControl.FileSystemAccessRule($account, 'FullControl', 'ContainerInherit,ObjectInherit', 'None', 'Allow')
        } else { $rule = New-Object Security.AccessControl.FileSystemAccessRule($account, 'FullControl', 'Allow') }
        $acl.AddAccessRule($rule)
    }
    Set-Acl -LiteralPath $Path -AclObject $acl
}
function Assert-NoReparse([string]$Path) {
    $current = $Path
    while ($current) {
        if (Test-Path -LiteralPath $current) {
            if ((Get-Item -LiteralPath $current -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw "Refusing reparse point: $current" }
        }
        $current = Split-Path -Parent $current
    }
}
$user = Get-LocalUser -Name $UserName -ErrorAction SilentlyContinue
if ($CreateUser -and $user) { throw '-CreateUser refuses to reuse an existing account. Omit it to select that account.' }
if (-not $user -and -not $CreateUser) { throw "Local account '$UserName' does not exist. Use -CreateUser to explicitly create a non-administrator account." }
if ($user -and -not $user.Enabled) { throw 'Selected account is disabled; enable it explicitly before setup.' }
$sshRoot = Join-Path $env:ProgramData 'ssh'
Assert-NoReparse $sshRoot
$configPath = Join-Path $sshRoot 'sshd_config'
Assert-NoReparse $configPath
$previousRules = @(Get-NetFirewallRule -PolicyStore ActiveStore | Select-Object -ExpandProperty Name)
$freshRuleCreated = $false
$capability = Get-WindowsCapability -Online -Name 'OpenSSH.Server~~~~0.0.1.0'
if ($capability.State -ne 'Installed') {
    $install = Add-WindowsCapability -Online -Name 'OpenSSH.Server~~~~0.0.1.0'
    # Installation can create a broad allow rule. Narrow ONLY that newly created default immediately.
    if ($previousRules -notcontains 'OpenSSH-Server-In-TCP') {
        $fresh = Get-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -ErrorAction SilentlyContinue
        if ($fresh) {
            $freshRuleCreated = $true
            try { $fresh | Set-NetFirewallRule -Profile $profiles -RemoteAddress LocalSubnet }
            catch {
                $scopeFailure = $_
                try { Remove-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' }
                catch { throw "Unable to scope or remove the newly created OpenSSH rule. Review firewall immediately: $($_.Exception.Message)" }
                throw $scopeFailure
            }
        }
    }
    if ($install.RestartNeeded) { throw 'OpenSSH capability installed but Windows requires a reboot. Reboot and rerun; SSH setup is not complete.' }
}
$sshd = Join-Path $env:WINDIR 'System32\OpenSSH\sshd.exe'
if (-not (Test-Path -LiteralPath $sshd)) { throw 'Windows OpenSSH sshd.exe is missing after capability installation.' }
if (-not (Test-Path -LiteralPath $sshRoot)) { New-Item -ItemType Directory -Path $sshRoot | Out-Null }
if (Test-Path -LiteralPath $configPath) { $original = [IO.File]::ReadAllText($configPath) }
else { $original = [IO.File]::ReadAllText((Join-Path (Split-Path $sshd) 'sshd_config_default')) }
if ($original -match '(?im)^\s*Include\s+') { throw 'Existing Include directives require manual review; included authentication policy is outside this MVP.' }
# Address-dependent Match policies cannot be proven safe using a localhost -T probe.
foreach ($line in ($original -split '\r?\n')) {
    if ($line -match '^\s*Match\s+' -and $line -notmatch '(?i)^\s*Match\s+(all|(?:User|Group)\s+[^\s#]+)\s*(?:#.*)?$') {
        throw 'Existing conditional Match policy needs manual review; address/host-dependent authentication cannot be safely automated.'
    }
}
# Preserve established restrictions; never replace MFA or explicitly disabled public-key policy.
if ($original -match '(?im)^\s*AuthenticationMethods\s+(?!publickey\s*(?:#.*)?$)\S+' -or $original -match '(?im)^\s*PubkeyAuthentication\s+no\b') {
    throw 'Existing authentication restrictions conflict with automated key-only setup. Review sshd_config manually; no policy was replaced.'
}
# Fail closed before enabling the service if existing firewall policy could broaden access.
$firewallProfiles = @(Get-NetFirewallProfile -PolicyStore ActiveStore)
if (-not $firewallProfiles.Count) { throw 'Cannot inspect effective Windows Firewall profiles; refusing to enable SSH.' }
foreach ($firewallProfile in $firewallProfiles) {
    if ([string]$firewallProfile.Enabled -ne 'True' -or [string]$firewallProfile.DefaultInboundAction -ne 'Block') {
        throw "Effective firewall profile '$($firewallProfile.Name)' must be enabled with default inbound Block before SSH setup. Review Windows Defender Firewall policy; no profile is changed automatically."
    }
}
$unsafeRules = @()
foreach ($rule in @(Get-NetFirewallRule -PolicyStore ActiveStore -Enabled True -Direction Inbound -Action Allow)) {
    # Windows exposes UWP family scope on the rule even when its application
    # filter Package is null. This metadata restricts the rule to a packaged app.
    $familyProperty = $rule.PSObject.Properties['PackageFamilyName']
    if ($null -ne $familyProperty) {
        $family = ([string]$familyProperty.Value).Trim()
        if ($family -and $family -notin @('Any', '*')) { continue }
    }
    $port = $rule | Get-NetFirewallPortFilter -PolicyStore ActiveStore
    if ([string]$port.Protocol -notin @('TCP', '6', 'Any', '256')) { continue }
    $coversSshPort = $false
    foreach ($value in @($port.LocalPort)) {
        if ($value -in @('Any', '22')) { $coversSshPort = $true }
        elseif ($value -match '^(\d+)-(\d+)$' -and [int]$Matches[1] -le 22 -and [int]$Matches[2] -ge 22) { $coversSshPort = $true }
    }
    if (-not $coversSshPort) { continue }
    $application = $rule | Get-NetFirewallApplicationFilter -PolicyStore ActiveStore
    # Packaged-app rules cannot authorize this unpackaged desktop sshd process,
    # even when their Program and Service filters both say Any.
    $packageProperty = $application.PSObject.Properties['Package']
    if ($null -ne $packageProperty) {
        $package = ([string]$packageProperty.Value).Trim()
        if ($package -and $package -notin @('Any', '*')) { continue }
    }
    $service = $rule | Get-NetFirewallServiceFilter -PolicyStore ActiveStore
    if ([string]$application.Program -ne 'Any' -and [string]$application.Program -notlike '*\sshd.exe') { continue }
    if ([string]$service.Service -notin @('Any', 'sshd')) { continue }
    $address = @($rule | Get-NetFirewallAddressFilter -PolicyStore ActiveStore | Select-Object -ExpandProperty RemoteAddress)
    $ruleProfiles = @(([string]$rule.Profile) -split ',\s*')
    $profileTooBroad = @($ruleProfiles | Where-Object { $_ -notin $profiles }).Count -gt 0
    if ($address.Count -ne 1 -or $address[0] -ne 'LocalSubnet' -or $profileTooBroad) {
        Write-Warning "Existing rule '$($rule.Name)' can permit broader SSH access: profile=$($rule.Profile), remote=$($address -join ','). It remains unchanged."
        Write-Host ("Review and explicitly narrow with: Get-NetFirewallRule -Name '{0}' | Set-NetFirewallRule -Profile {1} -RemoteAddress LocalSubnet" -f ($rule.Name -replace "'", "''"), ($profiles -join ','))
        $unsafeRules += $rule.Name
    }
}
if ($unsafeRules.Count) { throw 'Existing inbound firewall policy permits SSH beyond the selected LocalSubnet/profiles. Review the identified rules and explicitly remediate before rerunning. sshd was not started or restarted.' }
if (-not $user) {
    $random = New-Object byte[] 48
    $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
    try { $rng.GetBytes($random) } finally { $rng.Dispose() }
    $password = ConvertTo-SecureString ('Aa1!' + [Convert]::ToBase64String($random)) -AsPlainText -Force
    try { $user = New-LocalUser -Name $UserName -Password $password -Description 'oneline-ccc SSH (non-administrator)' }
    finally { $password.Dispose(); [Array]::Clear($random, 0, $random.Length) }
    $usersGroup = Get-LocalGroup -SID 'S-1-5-32-545'
    Add-LocalGroupMember -Group $usersGroup -Member $user
    Write-Host "Created non-administrator local account '$UserName'. Generated password was discarded; use the public key."
}
$userSid = $user.SID.Value
$admins = @(Get-LocalGroupMember -SID 'S-1-5-32-544')
$isAdminUser = @($admins | Where-Object { $_.SID.Value -eq $userSid }).Count -gt 0
if ($isAdminUser) {
    Write-Warning 'Administrator accounts share administrators_authorized_keys. This key can authenticate other administrator accounts allowed by their SSH policy. Prefer a dedicated non-administrator account.'
    $keyPath = Join-Path $sshRoot 'administrators_authorized_keys'
    $authorizedFile = '__PROGRAMDATA__/ssh/administrators_authorized_keys'
    $ownerSid = 'S-1-5-32-544'
} else {
    # CreateProfile initializes a genuine Windows profile for new accounts without logging on or retaining its password.
    $profile = Get-CimInstance Win32_UserProfile | Where-Object SID -eq $userSid | Select-Object -First 1
    if (-not $profile) {
        if (-not ('OnelineSsh.UserProfile' -as [type])) {
            Add-Type -TypeDefinition @'
using System;
using System.Text;
using System.Runtime.InteropServices;
namespace OnelineSsh {
 public static class UserProfile {
  [DllImport("userenv.dll", CharSet = CharSet.Unicode, SetLastError = true)]
  public static extern int CreateProfile(string sid, string user, StringBuilder path, uint length);
 }
}
'@
        }
        # userenv profile RPC rejects buffers larger than MAX_PATH with 0x800706F7.
        # See https://pinvoke.net/default.aspx/userenv/CreateProfile.html.
        $buffer = New-Object Text.StringBuilder 260
        $result = [OnelineSsh.UserProfile]::CreateProfile($userSid, $UserName, $buffer, [uint32]$buffer.Capacity)
        if ($result -ne 0) { throw "Windows CreateProfile failed (HRESULT $result). Initialize this user's profile manually and rerun." }
        $profilePath = $buffer.ToString()
    } else { $profilePath = $profile.LocalPath }
    $keyDirectory = Join-Path $profilePath '.ssh'
    Assert-NoReparse $keyDirectory
    $keyPath = Join-Path $keyDirectory 'authorized_keys'
    $authorizedFile = '.ssh/authorized_keys'
    $ownerSid = $userSid
}
Assert-NoReparse $keyPath
$markerStart = "# BEGIN oneline-ccc key-only $UserName"
$markerEnd = "# END oneline-ccc key-only $UserName"
$block = @"
$markerStart
Match User $($UserName.ToLowerInvariant())
    AuthenticationMethods publickey
    PubkeyAuthentication yes
    PasswordAuthentication no
    AuthorizedKeysFile $authorizedFile
$markerEnd
"@
# A marked block is owned only while it is exactly the generated policy.
# Manual restrictions, malformed boundaries and duplicate blocks must not be erased.
$escapedUser = [regex]::Escape($UserName)
$beginMarkers = [regex]::Matches($original, '(?im)^# BEGIN oneline-ccc key-only ' + $escapedUser + '(?=[ \t\r\n]|$)[^\r\n]*\r?$')
$endMarkers = [regex]::Matches($original, '(?im)^# END oneline-ccc key-only ' + $escapedUser + '(?=[ \t\r\n]|$)[^\r\n]*\r?$')
$pattern = '(?msi)^' + [regex]::Escape($markerStart) + '\r?\n.*?^' + [regex]::Escape($markerEnd) + '\r?\n?'
$managed = [regex]::Matches($original, $pattern)
if ($beginMarkers.Count -or $endMarkers.Count) {
    if ($beginMarkers.Count -ne 1 -or $endMarkers.Count -ne 1 -or $managed.Count -ne 1 -or
        ($managed[0].Value.Replace("`r`n", "`n").TrimEnd() -cne $block.Replace("`r`n", "`n").TrimEnd())) {
        throw 'Existing oneline-ccc managed block was edited, malformed or duplicated. Review it manually; no authorization policy was replaced.'
    }
}
$baseConfig = [regex]::Replace($original, $pattern, '')
$candidate = Join-Path $sshRoot ('sshd_config.oneline-' + [guid]::NewGuid().ToString('N'))
$utf8 = New-Object Text.UTF8Encoding($false)
$keyExisted = Test-Path -LiteralPath $keyPath
$keyBefore = $null; $keyAclBefore = $null
if ($keyExisted) { $keyBefore = [IO.File]::ReadAllBytes($keyPath); $keyAclBefore = Get-Acl -LiteralPath $keyPath }
$configExisted = Test-Path -LiteralPath $configPath
$configBefore = $null; $configAclBefore = $null
if ($configExisted) { $configBefore = [IO.File]::ReadAllBytes($configPath); $configAclBefore = Get-Acl -LiteralPath $configPath }
$directoryExisted = $true; $directoryAclBefore = $null
if (-not $isAdminUser) {
    $directoryExisted = Test-Path -LiteralPath $keyDirectory
    if ($directoryExisted) { $directoryAclBefore = Get-Acl -LiteralPath $keyDirectory }
}
$serviceBefore = Get-Service sshd
$wasRunning = $serviceBefore.Status -eq 'Running'
$serviceMode = (Get-CimInstance Win32_Service -Filter "Name='sshd'").StartMode
$ruleName = $null
$authTouched = $false
$serviceTouched = $false
try {
    [IO.File]::WriteAllText($candidate, $baseConfig.TrimEnd() + "`r`n`r`n" + $block + "`r`n", $utf8)
    Set-RestrictedAcl $candidate 'S-1-5-32-544' $false
    # Host keys must exist for sshd -t, but no private keys are ever read or backed up.
    & (Join-Path (Split-Path $sshd) 'ssh-keygen.exe') -A
    if ($LASTEXITCODE -ne 0) { throw 'Host key generation failed.' }
    $keyCheck = Join-Path $sshRoot ('oneline-public-' + [guid]::NewGuid().ToString('N'))
    try {
        [IO.File]::WriteAllText($keyCheck, ($publicKeys -join "`n") + "`n", $utf8)
        Set-RestrictedAcl $keyCheck 'S-1-5-32-544' $false
        & (Join-Path (Split-Path $sshd) 'ssh-keygen.exe') -l -f $keyCheck | Out-Null
        if ($LASTEXITCODE -ne 0) { throw 'ssh-keygen rejected public-key material.' }
    } finally { if (Test-Path -LiteralPath $keyCheck) { Remove-Item -LiteralPath $keyCheck -Force } }
    & $sshd -t -f $candidate
    if ($LASTEXITCODE -ne 0) { throw 'sshd.exe -t rejected candidate configuration; live config is unchanged.' }
    $effective = @(& $sshd -T -f $candidate -C "user=$($UserName.ToLowerInvariant()),host=localhost,addr=127.0.0.1")
    if ($LASTEXITCODE -ne 0) { throw 'Unable to inspect effective SSH policy; live config is unchanged.' }
    $effectivePorts = @($effective | Where-Object { $_ -match '^port ' })
    $otherListeners = @($effective | Where-Object { $_ -match '^listenaddress ' -and $_ -notmatch ':22$' })
    if ($effectivePorts.Count -ne 1 -or $effectivePorts[0] -ne 'port 22' -or $otherListeners.Count) {
        throw 'Existing SSH listen ports differ from TCP 22; review policy manually before setup.'
    }
    foreach ($expected in @('authenticationmethods publickey', 'pubkeyauthentication yes', 'passwordauthentication no', "authorizedkeysfile $authorizedFile", 'port 22')) {
        if ($effective -cnotcontains $expected) { throw "Existing policy prevents required effective setting '$expected'. Review sshd_config manually; live config is unchanged." }
    }
# Never modify pre-existing operator rules, including an earlier run's default rule.
$ruleName = 'Oneline-CCC-SSH-' + [guid]::NewGuid().ToString('N')
New-NetFirewallRule -Name $ruleName -DisplayName 'oneline-ccc SSH (LocalSubnet)' -Direction Inbound -Action Allow -Enabled True -Protocol TCP -LocalPort 22 -RemoteAddress LocalSubnet -Profile $profiles | Out-Null
$actualRule = Get-NetFirewallRule -Name $ruleName
$actualAddress = @($actualRule | Get-NetFirewallAddressFilter -PolicyStore ActiveStore | Select-Object -ExpandProperty RemoteAddress)
if ($actualAddress.Count -ne 1 -or $actualAddress[0] -ne 'LocalSubnet' -or [string]$actualRule.Profile -ne ($profiles -join ', ')) {
    throw 'Firewall scope verification failed. Inspect the newly created rule before starting sshd.'
}
    $authTouched = $true
    if (-not $isAdminUser) {
        if (-not $directoryExisted) { New-Item -ItemType Directory -Path $keyDirectory | Out-Null }
        Set-RestrictedAcl $keyDirectory $userSid $true
    }
    $stamp = [DateTime]::UtcNow.ToString('yyyyMMddTHHmmssfffffffZ')
    if (Test-Path -LiteralPath $keyPath) {
        Copy-Item -LiteralPath $keyPath -Destination "$keyPath.oneline-backup-$stamp"
        Set-RestrictedAcl "$keyPath.oneline-backup-$stamp" $ownerSid $false
        $keys = [IO.File]::ReadAllText($keyPath)
    } else { $keys = '' }
    foreach ($key in $publicKeys) {
        if (@($keys -split '\r?\n' | Where-Object { $_.Trim() -eq $key.Trim() }).Count -eq 0) {
            $keys = $keys.TrimEnd() + "`r`n" + $key.Trim() + "`r`n"
        }
    }
    [IO.File]::WriteAllText($keyPath, $keys, $utf8)
    Set-RestrictedAcl $keyPath $ownerSid $false
    if (Test-Path -LiteralPath $configPath) {
        Copy-Item -LiteralPath $configPath -Destination "$configPath.oneline-backup-$stamp"
        Set-RestrictedAcl "$configPath.oneline-backup-$stamp" 'S-1-5-32-544' $false
    }
    Copy-Item -LiteralPath $candidate -Destination $configPath -Force
    Set-RestrictedAcl $configPath 'S-1-5-32-544' $false
    & $sshd -t -f $configPath
    if ($LASTEXITCODE -ne 0) { throw 'Final sshd.exe -t failed; service was not restarted. Restore the config backup before proceeding.' }
    $serviceTouched = $true
    Set-Service -Name sshd -StartupType Automatic
    Restart-Service -Name sshd
    if ((Get-Service sshd).Status -ne 'Running') { throw 'sshd did not reach Running state.' }
} catch {
    $failure = $_
    $rollbackErrors = @()
    # Stop any service that could be reading partially installed authorization before restoring it.
    if ($authTouched -or $serviceTouched) {
        try { Stop-Service sshd -Force } catch { $rollbackErrors += "stop sshd: $($_.Exception.Message)" }
    }
    if ($authTouched) {
        try {
            if ($keyExisted) { [IO.File]::WriteAllBytes($keyPath, $keyBefore); Set-Acl -LiteralPath $keyPath -AclObject $keyAclBefore }
            elseif (Test-Path -LiteralPath $keyPath) { Remove-Item -LiteralPath $keyPath -Force }
        } catch { $rollbackErrors += "restore authorized keys: $($_.Exception.Message)" }
        try {
            if ($configExisted) { [IO.File]::WriteAllBytes($configPath, $configBefore); Set-Acl -LiteralPath $configPath -AclObject $configAclBefore }
            elseif (Test-Path -LiteralPath $configPath) { Remove-Item -LiteralPath $configPath -Force }
        } catch { $rollbackErrors += "restore sshd_config: $($_.Exception.Message)" }
        if (-not $isAdminUser) {
            try {
                if ($directoryExisted) { Set-Acl -LiteralPath $keyDirectory -AclObject $directoryAclBefore }
                elseif (Test-Path -LiteralPath $keyDirectory) {
                    # Backups contain prior public material only; do not remove unrelated files.
                    if (@(Get-ChildItem -LiteralPath $keyDirectory -Force).Count -eq 0) { Remove-Item -LiteralPath $keyDirectory }
                }
            } catch { $rollbackErrors += "restore .ssh ACL: $($_.Exception.Message)" }
        }
    }
    if ($ruleName) {
        try { if (Get-NetFirewallRule -Name $ruleName -ErrorAction SilentlyContinue) { Remove-NetFirewallRule -Name $ruleName } }
        catch { $rollbackErrors += "remove new firewall rule: $($_.Exception.Message)" }
    }
    if ($freshRuleCreated) {
        try { Remove-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' }
        catch { $rollbackErrors += "remove invocation-created OpenSSH rule: $($_.Exception.Message)" }
    }
    if ($authTouched -or $serviceTouched) {
        try {
            $restoreMode = switch ($serviceMode) { 'Auto' { 'Automatic' } 'Disabled' { 'Disabled' } default { 'Manual' } }
            Set-Service sshd -StartupType $restoreMode
            if ($wasRunning -and -not $rollbackErrors.Count) { Start-Service sshd }
        } catch { $rollbackErrors += "restore service state: $($_.Exception.Message)" }
    }
    if ($rollbackErrors.Count) { throw ("SSH setup failed: {0}. ROLLBACK INCOMPLETE: {1}. Inspect server authorization and service immediately." -f $failure.Exception.Message, ($rollbackErrors -join '; ')) }
    throw $failure
} finally { if (Test-Path -LiteralPath $candidate) { Remove-Item -LiteralPath $candidate -Force } }
Write-Host "SSH configured for '$UserName': public-key-only policy verified for localhost; firewall rule $ruleName allows TCP 22 from LocalSubnet on $($profiles -join ', '). Test connectivity from the actual client."
Write-Host 'Keep the client private key on the client. Verify the server host-key fingerprint locally before accepting it. No router port-forwarding is needed.'
$addresses = @(Get-NetIPAddress | Where-Object { $_.AddressState -eq 'Preferred' -and $_.IPAddress -notmatch '^(127\.|169\.254\.|::1$|fe80:)' })
foreach ($address in $addresses) { Write-Host ('Connect from the same subnet: ssh -i <client-private-key> -l "{0}" {1}' -f $UserName, $address.IPAddress) }
