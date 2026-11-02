#requires -Version 5.1
# Offline contract tests: never run a real capability, service, firewall or ACL command.
[CmdletBinding()]
param([switch]$ParseOnly)
$ErrorActionPreference = 'Stop'
$repository = Split-Path $PSScriptRoot -Parent
$installer = Join-Path $repository 'scripts/setup-ssh-server.ps1'
$root = Join-Path $repository ('.cache/ssh-tests-' + [Guid]::NewGuid().ToString('N'))
[IO.Directory]::CreateDirectory($root) | Out-Null
$script:passed = 0
$script:failures = @()

function Assert-True($Condition, [string]$Message) {
    if (-not $Condition) { throw $Message }
}
function Write-Utf8([string]$Path, [string]$Content) {
    [IO.File]::WriteAllText($Path, $Content, (New-Object Text.UTF8Encoding($false)))
}
function Test-Case([string]$Name, [scriptblock]$Body) {
    if ($ParseOnly -and $Name -notin @('SSH installer parses on the running PowerShell engine', 'installer requires PowerShell 5.1 and avoids known PS7-only syntax', 'setup mutation body has bounded native and ACL interception seams')) { return }
    try { & $Body; $script:passed++; Write-Output ('PASS ' + $Name) }
    catch { $script:failures += $Name + ': ' + $_.Exception.Message; Write-Output ('FAIL ' + $script:failures[-1]) }
}

if (-not (Test-Path -LiteralPath $installer)) { throw 'Missing scripts/setup-ssh-server.ps1' }
$parseTokens = $null
$parseErrors = $null
# AST offsets and mutation slices must refer to the identical full file string.
# Use one LF-normalized string for both parsing and edits; PS5 must not mix
# file CRLF offsets with reconstructed extent text. Production file stays untouched.
$sourceText = [IO.File]::ReadAllText($installer).Replace("`r`n", "`n").Replace("`r", "`n")
$sourceAst = [Management.Automation.Language.Parser]::ParseInput($sourceText, $installer, [ref]$parseTokens, [ref]$parseErrors)
Test-Case 'SSH installer parses on the running PowerShell engine' {
    Assert-True (@($parseErrors).Count -eq 0) ($parseErrors | Out-String)
}
Test-Case 'installer requires PowerShell 5.1 and avoids known PS7-only syntax' {
    Assert-True ($sourceAst.ScriptRequirements.RequiredPSVersion -eq [version]'5.1') 'Expected #requires -Version 5.1'
    $newSyntax = @($sourceAst.FindAll({ param($node)
        $node.GetType().Name -in @('TernaryExpressionAst', 'PipelineChainAst') -or
        ($node -is [Management.Automation.Language.BinaryExpressionAst] -and $node.Operator.ToString() -in @('QuestionQuestion')) -or
        ($node -is [Management.Automation.Language.AssignmentStatementAst] -and $node.Operator.ToString() -eq 'QuestionQuestionEquals')
    }, $true))
    Assert-True ($newSyntax.Count -eq 0) 'PS7-only syntax cannot run on Windows PowerShell 5.1'
}

# RFC4253 public blob with a deterministic, non-secret fixture payload.
$blob = [byte[]](@(0, 0, 0, 11) + [Text.Encoding]::ASCII.GetBytes('ssh-ed25519') + @(0, 0, 0, 32) + (1..32))
$publicKey = 'ssh-ed25519 ' + [Convert]::ToBase64String($blob) + ' offline-fixture'
$publicKeyPath = Join-Path $root 'fixture.pub'
Write-Utf8 $publicKeyPath ($publicKey + "`n")

function Invoke-DryRun([hashtable]$Overrides = @{}, [switch]$Failure) {
    Assert-True (@($parseErrors).Count -eq 0) 'Cannot validate behavior of an unparseable installer'
    # Functions shadow real cmdlets on BOTH Windows and Linux. Any accidental OS
    # access must fail the test, including access swallowed by installer catch blocks.
    $script:unexpectedCalls = @()
    $forbidden = @('Get-WindowsCapability', 'Add-WindowsCapability', 'Get-Service', 'Set-Service',
        'Start-Service', 'Restart-Service', 'Stop-Service', 'Get-NetFirewallRule', 'New-NetFirewallRule',
        'Set-NetFirewallRule', 'Remove-NetFirewallRule', 'Enable-NetFirewallRule', 'Disable-NetFirewallRule',
        'Get-NetFirewallAddressFilter', 'Get-NetFirewallPortFilter', 'Get-NetConnectionProfile',
        'Set-NetConnectionProfile', 'Get-LocalUser', 'New-LocalUser', 'Set-LocalUser',
        'Add-LocalGroupMember', 'Get-CimInstance', 'Get-Acl', 'Set-Acl', 'icacls', 'icacls.exe',
        'ssh-keygen', 'ssh-keygen.exe', 'sshd', 'sshd.exe', 'Install-Module', 'Invoke-WebRequest',
        'Start-Process', 'Remove-LocalUser', 'Remove-LocalGroupMember', 'Set-LocalGroup',
        'net', 'net.exe', 'netsh', 'netsh.exe', 'sc.exe', 'takeown', 'takeown.exe',
        'cmd.exe', 'powershell.exe', 'pwsh.exe', 'Invoke-Expression',
        'New-Item', 'Set-Content', 'Add-Content', 'Out-File', 'Copy-Item',
        'Move-Item', 'Remove-Item')
    foreach ($command in $forbidden) {
        Set-Item -Path ('Function:local:' + $command) -Value {
            $script:unexpectedCalls += $MyInvocation.MyCommand.Name
            throw ('Forbidden host operation: ' + $MyInvocation.MyCommand.Name)
        }
    }
    $options = @{UserName = 'sshfixture'; PublicKey = $publicKey; DryRun = $true; NonInteractive = $true}
    foreach ($name in $Overrides.Keys) {
        if ($null -eq $Overrides[$name]) { $options.Remove($name) }
        else { $options[$name] = $Overrides[$name] }
    }
    $errorMessage = $null
    $result = @()
    try { $result = @(& $installer @options) } catch { $errorMessage = $_.Exception.Message }
    Assert-True ($script:unexpectedCalls.Count -eq 0) ('DryRun/validation reached host commands: ' + ($script:unexpectedCalls -join ', '))
    if ($Failure) {
        Assert-True ($null -ne $errorMessage) 'Invalid input unexpectedly succeeded'
        Assert-True ($errorMessage -notmatch '(?i)requires Windows|elevation required|administrator|not supported on this platform') ('Input was rejected by OS gating instead of validation: ' + $errorMessage)
    }
    else { Assert-True ($null -eq $errorMessage) ('DryRun failed: ' + $errorMessage) }
    return [pscustomobject]@{ Output = $result; Error = $errorMessage }
}

Test-Case 'real dry-run works offline before Windows OS calls' {
    $result = Invoke-DryRun
    $plan = @($result.Output | Where-Object { $null -ne $_.PSObject.Properties['RemoteAddress'] })
    Assert-True ($plan.Count -eq 1) 'DryRun must emit one inspectable plan'
    Assert-True ($plan[0].UserName -ceq 'sshfixture') 'Plan lost the requested username'
    Assert-True ($plan[0].RemoteAddress -eq 'LocalSubnet') 'Default permits a broader source than LocalSubnet'
    Assert-True (@($plan[0].NetworkProfiles).Count -eq 1 -and $plan[0].NetworkProfiles[0] -eq 'Private') 'Default must select only Private'
    Assert-True (-not $plan[0].CreateUser) 'Default requests user creation'
    Assert-True (($plan[0] | Out-String) -notmatch [regex]::Escape($publicKey.Split(' ')[1])) 'DryRun echoes supplied key material'
}
Test-Case 'Public profile requires explicit opt-in and still limits source to subnet' {
    $result = Invoke-DryRun @{AllowPublicNetworkProfile = $true; CreateUser = $true}
    $plan = @($result.Output | Where-Object { $null -ne $_.PSObject.Properties['RemoteAddress'] })[0]
    Assert-True (@($plan.NetworkProfiles) -contains 'Private' -and @($plan.NetworkProfiles) -contains 'Public') 'Explicit Public opt-in missing from plan'
    Assert-True ($plan.RemoteAddress -eq 'LocalSubnet') 'Public opt-in broadened remote addresses'
    Assert-True $plan.CreateUser 'CreateUser intent missing from plan'
}
Test-Case 'public key file accepted offline' {
    Invoke-DryRun @{PublicKey = $null; PublicKeyFile = $publicKeyPath} | Out-Null
}
$invalidInputs = @(
    @{UserName = 'bad user'}, @{UserName = 'bad;whoami'},
    @{UserName = "bad`nuser"}, @{UserName = '../escape'}, @{UserName = 'domain\user'},
    @{PublicKey = 'password'}, @{PublicKey = '-----BEGIN OPENSSH PRIVATE KEY-----'},
    @{PublicKey = '-----BEGIN RSA PRIVATE KEY-----'}, @{PublicKey = 'ssh-ed25519 invalid-base64'},
    @{PublicKey = "${publicKey}`ncommand=evil"}, @{PublicKey = 'command="whoami" ' + $publicKey},
    @{PublicKeyFile = $publicKeyPath},
    @{PublicKey = $null; PublicKeyFile = (Join-Path $root 'missing.pub')},
    @{RemoteAddress = 'Any'}, @{RemoteAddress = '0.0.0.0/0'},
    @{Password = 'fixture-password'}, @{UserName = '$(whoami)'}
)
$index = 0
foreach ($invalid in $invalidInputs) {
    $index++
    Test-Case ('reject unsafe input before OS access ' + $index) {
        Invoke-DryRun $invalid -Failure | Out-Null
    }
}
Test-Case 'private key file rejected without OS access' {
    $privateFile = Join-Path $root 'private-fixture'
    Write-Utf8 $privateFile "-----BEGIN OPENSSH PRIVATE KEY-----`nfixture-not-a-secret`n-----END OPENSSH PRIVATE KEY-----`n"
    Invoke-DryRun @{PublicKey = $null; PublicKeyFile = $privateFile} -Failure | Out-Null
}
Test-Case 'multiple public keys rejected rather than silently selecting one' {
    $multiple = Join-Path $root 'multiple.pub'
    Write-Utf8 $multiple ($publicKey + "`n" + $publicKey + "`n")
    Invoke-DryRun @{PublicKey = $null; PublicKeyFile = $multiple} -Failure | Out-Null
}

function Get-FixtureSnapshot([string]$Directory) {
    $snapshot = @()
    foreach ($file in @(Get-ChildItem -LiteralPath $Directory -File -Recurse | Sort-Object FullName)) {
        $acl = ''
        if ($env:OS -eq 'Windows_NT') { $acl = (Get-Acl -LiteralPath $file.FullName).Sddl }
        $snapshot += [pscustomobject]@{
            Path = $file.FullName.Substring($Directory.Length)
            Bytes = [Convert]::ToBase64String([IO.File]::ReadAllBytes($file.FullName))
            LastWriteTicks = $file.LastWriteTimeUtc.Ticks
            Acl = $acl
        }
    }
    return ConvertTo-Json -InputObject @($snapshot) -Compress -Depth 5
}

Test-Case 'GitHub import dry-run plans offline without fetching keys' {
    $result = Invoke-DryRun @{PublicKey = $null; GitHubUser = 'fixture-user'}
    $plan = @($result.Output | Where-Object { $null -ne $_.PSObject.Properties['RemoteAddress'] })
    Assert-True ($plan.Count -eq 1) 'GitHub dry-run did not produce a plan'
    Assert-True ($plan[0].RemoteAddress -eq 'LocalSubnet') 'GitHub import broadened the source scope'
}
foreach ($handle in @('', '../escape', 'a/b', 'a?token=x', 'a#fragment', 'a@host', '-leading', 'trailing-', 'a--b', 'a b', "bad`nuser", '$(whoami)', ('a' * 40))) {
    Test-Case ('reject unsafe GitHub handle ' + ($handle -replace '[\r\n]', '_')) {
        Invoke-DryRun @{PublicKey = $null; GitHubUser = $handle} -Failure | Out-Null
    }
}
Test-Case 'GitHub handle and inline key are mutually exclusive' {
    Invoke-DryRun @{GitHubUser = 'fixture-user'} -Failure | Out-Null
}
Test-Case 'GitHub handle and key file are mutually exclusive' {
    Invoke-DryRun @{PublicKey = $null; PublicKeyFile = $publicKeyPath; GitHubUser = 'fixture-user'} -Failure | Out-Null
}

# Execute the installer's actual mutation statements, with platform/native/ACL
# boundaries replaced by test doubles. Public parsing and DryRun above run the
# unmodified script. No copied policy implementation lives in this harness.
function Get-MockedSetupBody {
    $start = @($sourceAst.EndBlock.Statements | Where-Object {
        $_.Extent.Text -match '^\$networks\s*='
    })
    Assert-True ($start.Count -eq 1) 'Cannot locate setup boundary; review harness before executing changed source'
    $offset = $start[0].Extent.StartOffset
    $extentStart = $sourceAst.Extent.StartOffset
    $body = $sourceText.Substring($offset)
    $edits = @()
    $aclFunction = @($sourceAst.FindAll({ param($node)
        $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq 'Set-RestrictedAcl'
    }, $true))
    Assert-True ($aclFunction.Count -eq 1) 'Expected one mockable ACL boundary'
    $edits += @{Start = $aclFunction[0].Extent.StartOffset; End = $aclFunction[0].Extent.EndOffset; Original = $aclFunction[0].Extent.Text; Line = $aclFunction[0].Extent.StartLineNumber; Column = $aclFunction[0].Extent.StartColumnNumber; Text = @'
function Set-RestrictedAcl([string]$Path, [string]$OwnerSid, [bool]$Directory) {
    Assert-FixturePath $Path
    $state.Events.Add('ACL:' + $Path) | Out-Null
    $state.Acls[$Path] = 'restricted:' + $OwnerSid
}
'@}
    $nativeCommands = @($sourceAst.FindAll({ param($node)
        $node -is [Management.Automation.Language.CommandAst] -and $node.InvocationOperator.ToString() -eq 'Ampersand'
    }, $true) | Where-Object { $_.Extent.StartOffset -ge $offset })
    Assert-True ($nativeCommands.Count -ge 3) 'Expected native SSH validation calls'
    foreach ($command in $nativeCommands) {
        $target = $command.CommandElements[0].Extent.Text
        Assert-True ($target -eq '$sshd' -or $target -match "^\(Join-Path \(Split-Path \`$sshd\) 'ssh-keygen.exe'\)$") ('Unrecognized native target: ' + $target)
        # Replace only the invocation operator. Keep executable expressions and
        # every argument byte-for-byte from sourceText, including interpolated -T
        # context strings; never reconstruct commands from PS5 Extent.Text.
        $operators = @($parseTokens | Where-Object {
            $_.Kind.ToString() -eq 'Ampersand' -and
            $_.Extent.StartOffset -ge $command.Extent.StartOffset -and
            $_.Extent.EndOffset -le $command.CommandElements[0].Extent.StartOffset
        })
        Assert-True ($operators.Count -eq 1) ('Expected one invocation token at source line ' + $command.Extent.StartLineNumber)
        $operator = $operators[0].Extent
        $edits += @{Start = $operator.StartOffset; End = $operator.EndOffset; Original = '&'; Line = $operator.StartLineNumber; Column = $operator.StartColumnNumber; Text = 'Invoke-FixtureNative'}
    }
    foreach ($edit in @($edits | Sort-Object { [int]$_.Start } -Descending)) {
        $relative = $edit.Start - $offset
        $actual = $body.Substring($relative, $edit.End - $edit.Start)
        if ($actual -cne $edit.Original) {
            Write-Utf8 (Join-Path $root 'mocked-setup-body.ps1') $body
            Write-Utf8 (Join-Path $root 'normalized-source.ps1') $sourceText
            $raw = $sourceText.Substring($edit.Start, $edit.End - $edit.Start)
            $expectedEscaped = $edit.Original.Replace("`r", '\r').Replace("`n", '\n').Replace("`t", '\t')
            $actualEscaped = $actual.Replace("`r", '\r').Replace("`n", '\n').Replace("`t", '\t')
            $rawEscaped = $raw.Replace("`r", '\r').Replace("`n", '\n').Replace("`t", '\t')
            throw ('AST replacement extent mismatch; engine=' + $PSVersionTable.PSVersion +
                '; source offsets=' + $edit.Start + '-' + $edit.End +
                '; line=' + $edit.Line + ':' + $edit.Column + '; relative=' + $relative +
                '; expected length=' + $edit.Original.Length + '; actual length=' + $actual.Length +
                '; expected=<' + $expectedEscaped + '>; actual=<' + $actualEscaped +
                '>; unedited raw=<' + $rawEscaped + '>')
        }
        $body = $body.Remove($relative, $edit.End - $edit.Start).Insert($relative, $edit.Text)
    }
    # Save BEFORE parsing so PS5-only reconstruction failures remain inspectable.
    # This contains source/test code only, never fixture key contents or host secrets.
    $body = 'Set-StrictMode -Version Latest' + "`n" + $body
    $mockPath = Join-Path $root 'mocked-setup-body.ps1'
    Write-Utf8 $mockPath $body
    $mockTokens = $null
    $mockErrors = $null
    $mockAst = [Management.Automation.Language.Parser]::ParseInput($body, [ref]$mockTokens, [ref]$mockErrors)
    if (@($mockErrors).Count) {
        $diagnostics = @($mockErrors | ForEach-Object {
            '{0} line {1}:{2}-{3}:{4}: {5}; extent=<{6}>' -f $_.ErrorId,
                $_.Extent.StartLineNumber, $_.Extent.StartColumnNumber,
                $_.Extent.EndLineNumber, $_.Extent.EndColumnNumber,
                $_.Message, $_.Extent.Text
        }) -join "`n"
        throw ('Intercepted setup body does not parse. Engine=' + $PSVersionTable.PSVersion +
            '; source extent start=' + $extentStart + '; setup offset=' + $offset +
            '; source chars=' + $sourceText.Length + '; mock chars=' + $body.Length +
            '; saved=' + $mockPath + "`n" + $diagnostics)
    }
    Write-Host ('Mock body parses on ' + $PSVersionTable.PSVersion + '; source extent start=' + $extentStart + '; setup offset=' + $offset + '; saved=' + $mockPath)
    # Fail closed when the source gains a new command. Every mutator in this
    # allowlist is overridden below; only bounded scratch-file commands pass through.
    $allowed = @('Add-LocalGroupMember', 'Add-Type', 'Add-WindowsCapability', 'Assert-FixturePath',
        'Assert-NoReparse', 'ConvertTo-SecureString', 'Copy-Item', 'Get-Acl', 'Get-ChildItem',
        'Get-CimInstance', 'Get-Item', 'Get-LocalGroup', 'Get-LocalGroupMember', 'Get-LocalUser',
        'Get-NetConnectionProfile', 'Get-NetFirewallProfile', 'Get-NetFirewallAddressFilter', 'Get-NetFirewallApplicationFilter',
        'Get-NetFirewallPortFilter', 'Get-NetFirewallServiceFilter', 'Get-NetFirewallRule',
        'Get-NetIPAddress', 'Get-Service', 'Get-WindowsCapability', 'Invoke-FixtureNative',
        'Join-Path', 'New-Item', 'New-LocalUser', 'New-NetFirewallRule', 'New-Object', 'Out-Null',
        'Read-Host', 'Remove-Item', 'Remove-NetFirewallRule', 'Restart-Service', 'Select-Object',
        'Set-Acl', 'Set-NetFirewallRule', 'Set-RestrictedAcl', 'Set-Service', 'Set-StrictMode', 'Split-Path',
        'Start-Service', 'Stop-Service', 'Test-Path', 'Where-Object', 'Write-Host', 'Write-Warning')
    foreach ($command in $mockAst.FindAll({ param($node) $node -is [Management.Automation.Language.CommandAst] }, $true)) {
        Assert-True ($allowed -contains $command.GetCommandName()) ('Review new/unbounded setup command before executing: ' + $command.Extent.Text)
        Assert-True ($command.InvocationOperator.ToString() -ne 'Dot') 'Dot-sourcing setup code is not permitted in mock execution'
    }
    return [scriptblock]::Create($body)
}

function New-SetupFixture([string]$Name) {
    $directory = Join-Path $root $Name
    $programData = Join-Path $directory 'ProgramData'
    $profile = Join-Path $directory 'profile'
    $windows = Join-Path $directory 'Windows'
    foreach ($path in @((Join-Path $programData 'ssh'), (Join-Path $profile '.ssh'), (Join-Path $windows 'System32/OpenSSH'))) {
        [IO.Directory]::CreateDirectory($path) | Out-Null
    }
    Write-Utf8 (Join-Path $windows 'System32/OpenSSH/sshd.exe') 'not an executable; all native calls must be intercepted'
    Write-Utf8 (Join-Path $windows 'System32/OpenSSH/sshd_config_default') "# fixture default`nPort 22`n"
    Write-Utf8 (Join-Path $programData 'ssh/sshd_config') "# operator comment`nPort 22`n"
    Write-Utf8 (Join-Path $profile '.ssh/authorized_keys') "# existing authorized keys`n"
    Write-Utf8 (Join-Path $programData 'ssh/ssh_host_ed25519_key') 'opaque fixture sentinel: never read or replace host private keys'
    return @{
        Root = $directory; ProgramData = $programData; Profile = $profile; Windows = $windows
        Events = (New-Object 'System.Collections.Generic.List[string]')
        Rules = @{}; UserExists = $true; Enabled = $true; Capability = 'Installed'
        PublicNetwork = $false; FailNative = $false; BadEffective = $false
        FailFirewall = $false; FailService = $false; Acls = @{}
        Service = [pscustomobject]@{Status = 'Stopped'; StartType = 'Manual'}
        FirewallEnabled = 'True'; DefaultInboundAction = 'Block'; EmptyFirewallProfiles = $false
    }
}

function Invoke-MockedSetup([hashtable]$State, [switch]$Public, [switch]$Create, [switch]$Failure) {
    Assert-True ($mockedSetupBody -is [scriptblock]) 'No executable mock body: safety-seam initialization failed; no setup assertions are valid'
    $state.Events.Clear()
    $state.ServiceFailureConsumed = $false
    $UserName = 'sshfixture'
    $PublicKey = $publicKey
    $publicKeys = @($publicKey)
    $CreateUser = [bool]$Create
    $NonInteractive = $true
    $AllowPublicNetworkProfile = [bool]$Public
    $RemoteAddress = 'LocalSubnet'
    $profiles = @('Private')
    if ($Public) { $profiles += 'Public' }
    function Assert-FixturePath([string]$Path) {
        $full = [IO.Path]::GetFullPath($Path)
        $prefix = [IO.Path]::GetFullPath($state.Root) + [IO.Path]::DirectorySeparatorChar
        Assert-True ($full.StartsWith($prefix, [StringComparison]::OrdinalIgnoreCase)) ('Refusing path outside fixture: ' + $Path)
    }
    function Get-NetFirewallProfile {
        param($PolicyStore)
        if (-not $state.EmptyFirewallProfiles) {
            [pscustomobject]@{Name = 'Private'; Enabled = $state.FirewallEnabled; DefaultInboundAction = $state.DefaultInboundAction}
        }
    }
    function Get-NetConnectionProfile {
        [pscustomobject]@{Name = 'fixture'; NetworkCategory = $(if ($state.PublicNetwork) { 'Public' } else { 'Private' })}
    }
    function Get-LocalUser { param($Name) if ($state.UserExists) { [pscustomobject]@{Name = $Name; Enabled = $state.Enabled; SID = [pscustomobject]@{Value = 'S-1-5-21-111-222-333-1001'}} } }
    function Get-LocalGroupMember { param($SID) }
    function Get-CimInstance {
        param($ClassName, $Filter)
        if ($ClassName -eq 'Win32_Service') {
            $mode = switch ($state.Service.StartType) { 'Automatic' {'Auto'} 'Disabled' {'Disabled'} default {'Manual'} }
            [pscustomobject]@{StartMode = $mode}
        } else { [pscustomobject]@{SID = 'S-1-5-21-111-222-333-1001'; LocalPath = $state.Profile} }
    }
    function Get-WindowsCapability { param([switch]$Online, $Name) [pscustomobject]@{State = $state.Capability} }
    function Add-WindowsCapability { param([switch]$Online, $Name) $state.Events.Add('CapabilityInstall'); $state.Capability = 'Installed'; [pscustomobject]@{RestartNeeded = $false} }
    function Get-NetFirewallRule {
        param($Name, $PolicyStore, $Enabled, $Direction, $Action)
        if ($Name) { if ($state.Rules.ContainsKey($Name)) { $state.Rules[$Name] } }
        else { $state.Rules.Values }
    }
    function New-NetFirewallRule {
        param($Name, $DisplayName, $Direction, $Action, $Enabled, $Protocol, $LocalPort, $RemoteAddress, $Profile)
        $state.Events.Add('FirewallCreate')
        if ($state.FailFirewall) { throw 'Fixture firewall creation failed' }
        $rule = [pscustomobject]@{Name = $Name; Profile = ($Profile -join ', '); RemoteAddress = $RemoteAddress; LocalPort = $LocalPort; Protocol = $Protocol; Program = 'Any'; Package = 'Any'; Service = 'Any'; Enabled = $Enabled; Direction = $Direction; Action = $Action}
        $state.Rules[$Name] = $rule
        $rule
    }
    function Get-NetFirewallAddressFilter { param([Parameter(ValueFromPipeline = $true)]$InputObject, $PolicyStore) process { Assert-True ($PolicyStore -eq 'ActiveStore') 'Associated firewall filter must explicitly query ActiveStore'; [pscustomobject]@{RemoteAddress = $InputObject.RemoteAddress} } }
    function Get-NetFirewallPortFilter { param([Parameter(ValueFromPipeline = $true)]$InputObject, $PolicyStore) process { Assert-True ($PolicyStore -eq 'ActiveStore') 'Associated firewall filter must explicitly query ActiveStore'; [pscustomobject]@{LocalPort = $InputObject.LocalPort; Protocol = $InputObject.Protocol} } }
    function Get-NetFirewallApplicationFilter { param([Parameter(ValueFromPipeline = $true)]$InputObject, $PolicyStore) process { Assert-True ($PolicyStore -eq 'ActiveStore') 'Associated firewall filter must explicitly query ActiveStore'; [pscustomobject]@{Program = $InputObject.Program; Package = $InputObject.Package} } }
    function Get-NetFirewallServiceFilter { param([Parameter(ValueFromPipeline = $true)]$InputObject, $PolicyStore) process { Assert-True ($PolicyStore -eq 'ActiveStore') 'Associated firewall filter must explicitly query ActiveStore'; [pscustomobject]@{Service = $InputObject.Service} } }
    function Remove-NetFirewallRule { param($Name) $state.Events.Add('FirewallRemove'); $state.Rules.Remove($Name) }
    function Set-Service { param($Name, $StartupType) $state.Events.Add('ServiceConfigure'); $state.Service.StartType = [string]$StartupType }
    function Restart-Service {
        param($Name)
        $state.Events.Add('ServiceRestart')
        if ($state.FailService -and -not $state.ServiceFailureConsumed) {
            $state.ServiceFailureConsumed = $true
            $state.Service.Status = 'Stopped'
            throw 'Fixture service restart failed'
        }
        $state.Service.Status = 'Running'
    }
    function Start-Service { param($Name) $state.Events.Add('ServiceStart'); $state.Service.Status = 'Running' }
    function Stop-Service { param($Name, [switch]$Force) $state.Events.Add('ServiceStop'); $state.Service.Status = 'Stopped' }
    function Get-Service { param($Name) $state.Service }
    function Get-Acl {
        param($LiteralPath, $Path)
        if (-not $LiteralPath) { $LiteralPath = $Path }
        Assert-FixturePath $LiteralPath
        $sddl = 'fixture-original-acl'
        if ($state.Acls.ContainsKey($LiteralPath)) { $sddl = $state.Acls[$LiteralPath] }
        [pscustomobject]@{Sddl = $sddl; FixturePath = $LiteralPath}
    }
    function Set-Acl {
        param($LiteralPath, $Path, $AclObject)
        if (-not $LiteralPath) { $LiteralPath = $Path }
        Assert-FixturePath $LiteralPath
        $state.Events.Add('ACLRestore:' + $LiteralPath)
        $state.Acls[$LiteralPath] = $AclObject.Sddl
    }
    function Get-NetIPAddress { }
    function Invoke-FixtureNative {
        param([string]$Executable)
        Assert-FixturePath $Executable
        $state.Events.Add('Native:' + [IO.Path]::GetFileName($Executable) + ':' + ($args -join ' '))
        $global:LASTEXITCODE = 0
        if ($args -contains '-t' -and $state.FailNative) { $global:LASTEXITCODE = 1 }
        if ($args -contains '-T') {
            'authenticationmethods publickey'; 'pubkeyauthentication yes'
            if ($state.BadEffective) { 'passwordauthentication yes' } else { 'passwordauthentication no' }
            'authorizedkeysfile .ssh/authorized_keys'; 'port 22'
        }
    }
    # These operations are never needed for the existing-user fixture. Block them
    # rather than risk a platform branch mutating the host or runner.
    foreach ($command in @('New-LocalUser', 'Set-LocalUser', 'Remove-LocalUser', 'Add-LocalGroupMember',
        'Set-NetConnectionProfile', 'Set-NetFirewallRule',
        'Enable-NetFirewallRule', 'Disable-NetFirewallRule', 'Start-Process', 'Add-Type', 'Invoke-WebRequest', 'Read-Host', 'Get-LocalGroup')) {
        Set-Item -Path ('Function:local:' + $command) -Value {
            $state.Events.Add('FORBIDDEN:' + $MyInvocation.MyCommand.Name)
            throw ('Forbidden host operation: ' + $MyInvocation.MyCommand.Name)
        }
    }
    function Copy-Item {
        param($LiteralPath, $Destination, [switch]$Force)
        Assert-FixturePath $LiteralPath; Assert-FixturePath $Destination
        Microsoft.PowerShell.Management\Copy-Item -LiteralPath $LiteralPath -Destination $Destination -Force:$Force
    }
    function New-Item { param($Path, $ItemType) Assert-FixturePath $Path; Microsoft.PowerShell.Management\New-Item -Path $Path -ItemType $ItemType }
    function Remove-Item { param($LiteralPath, [switch]$Force) Assert-FixturePath $LiteralPath; Microsoft.PowerShell.Management\Remove-Item -LiteralPath $LiteralPath -Force:$Force }
    $oldProgramData = $env:ProgramData
    $oldWindows = $env:WINDIR
    $errorMessage = $null
    $warnings = @()
    try {
        $env:ProgramData = $state.ProgramData
        $env:WINDIR = $state.Windows
        & $mockedSetupBody 3>&1 | ForEach-Object {
            if ($_ -is [Management.Automation.WarningRecord]) { $warnings += $_.Message }
        }
    } catch { $errorMessage = $_.Exception.Message }
    finally { $env:ProgramData = $oldProgramData; $env:WINDIR = $oldWindows }
    Assert-True (@($state.Events | Where-Object { $_ -like 'FORBIDDEN:*' }).Count -eq 0) ($state.Events -join ', ')
    if ($Failure) { Assert-True ($null -ne $errorMessage) 'Refused setup unexpectedly succeeded' }
    else { Assert-True ($null -eq $errorMessage) ('Mocked setup failed: ' + $errorMessage) }
    return [pscustomobject]@{Error = $errorMessage; Warnings = $warnings}
}

$mockedSetupBody = $null
Test-Case 'setup mutation body has bounded native and ACL interception seams' {
    $bodyResult = @(Get-MockedSetupBody)
    Assert-True ($bodyResult.Count -eq 1 -and $bodyResult[0] -is [scriptblock]) 'Mock factory must return exactly one ScriptBlock and no pipeline noise'
    $script:mockedSetupBody = $bodyResult[0]
}
Test-Case 'mocked setup installs key-only policy with Private LocalSubnet firewall' {
    Assert-True ($null -ne $mockedSetupBody) 'Missing safely intercepted setup body'
    $state = New-SetupFixture 'initial'
    $hostKey = Join-Path $state.ProgramData 'ssh/ssh_host_ed25519_key'
    $hostBytes = [IO.File]::ReadAllText($hostKey)
    Invoke-MockedSetup $state | Out-Null
    $keys = [IO.File]::ReadAllText((Join-Path $state.Profile '.ssh/authorized_keys'))
    Assert-True ($keys.Contains('# existing authorized keys') -and $keys.Contains($publicKey)) 'Setup lost existing key content or failed to add supplied key'
    $config = [IO.File]::ReadAllText((Join-Path $state.ProgramData 'ssh/sshd_config'))
    Assert-True ($config.Contains('# operator comment') -and $config -match '(?im)^\s*PasswordAuthentication no\s*$') 'Key-only config or operator content missing'
    Assert-True ([IO.File]::ReadAllText($hostKey) -ceq $hostBytes) 'Host private-key fixture changed'
    Assert-True ($state.Rules.Count -eq 1) 'Expected one owned firewall rule'
    $rule = @($state.Rules.Values)[0]
    Assert-True ($rule.RemoteAddress -eq 'LocalSubnet' -and $rule.Profile -eq 'Private' -and $rule.LocalPort -eq 22 -and $rule.Direction -eq 'Inbound' -and $rule.Action -eq 'Allow') 'Firewall plan is broader than Private LocalSubnet TCP 22'
    $validation = @($state.Events | Where-Object { $_ -like 'Native:sshd.exe:*-t*' })
    Assert-True ($validation.Count -ge 1) 'Native config validation was skipped'
    Assert-True ($state.Events -contains 'ServiceRestart') 'Successful plan did not reach mocked service restart'
}
Test-Case 'active Public network warns without changing its category or broadening defaults' {
    $state = New-SetupFixture 'public-default'
    $state.PublicNetwork = $true
    $result = Invoke-MockedSetup $state
    Assert-True (($result.Warnings -join ' ') -match 'Public') 'Active Public network warning missing'
    Assert-True (@($state.Rules.Values)[0].Profile -eq 'Private') 'Active Public network automatically broadened the rule'
}
Test-Case 'explicit Public opt-in remains LocalSubnet only' {
    $state = New-SetupFixture 'public-opt-in'
    $state.PublicNetwork = $true
    Invoke-MockedSetup $state -Public | Out-Null
    $rule = @($state.Rules.Values)[0]
    Assert-True ($rule.Profile -eq 'Private, Public' -and $rule.RemoteAddress -eq 'LocalSubnet') 'Explicit Public rule has incorrect scope'
}
Test-Case 'candidate rejection preserves live config, keys and service state' {
    $state = New-SetupFixture 'candidate-rejected'
    $state.FailNative = $true
    $before = Get-FixtureSnapshot $state.Root
    Invoke-MockedSetup $state -Failure | Out-Null
    Assert-True ((Get-FixtureSnapshot $state.Root) -ceq $before) 'Rejected candidate changed fixture files or ACLs'
    Assert-True (-not ($state.Events -contains 'ServiceRestart') -and $state.Rules.Count -eq 0) 'Rejected candidate reached service/firewall mutation'
}
Test-Case 'effective password policy conflict preserves live config and keys' {
    $state = New-SetupFixture 'effective-rejected'
    $state.BadEffective = $true
    $before = Get-FixtureSnapshot $state.Root
    Invoke-MockedSetup $state -Failure | Out-Null
    Assert-True ((Get-FixtureSnapshot $state.Root) -ceq $before) 'Policy conflict changed fixture files or ACLs'
    Assert-True (-not ($state.Events -contains 'ServiceRestart')) 'Policy conflict restarted service'
}
Test-Case 'existing user is never silently created or replaced' {
    $state = New-SetupFixture 'create-conflict'
    $before = Get-FixtureSnapshot $state.Root
    Invoke-MockedSetup $state -Create -Failure | Out-Null
    Assert-True ((Get-FixtureSnapshot $state.Root) -ceq $before) 'CreateUser conflict changed files'
    Assert-True ($state.Events.Count -eq 0) 'CreateUser conflict reached mutations'
}
Test-Case 'missing user requires explicit creation before any mutations' {
    $state = New-SetupFixture 'missing-user'
    $state.UserExists = $false
    Invoke-MockedSetup $state -Failure | Out-Null
    Assert-True ($state.Events.Count -eq 0) 'Missing user reached mutations'
}
Test-Case 'dry-run preserves existing keys, config and ACLs without OS access' {
    $state = New-SetupFixture 'dryrun-no-op'
    $before = Get-FixtureSnapshot $state.Root
    $oldProgramData = $env:ProgramData
    $oldWindows = $env:WINDIR
    try {
        $env:ProgramData = $state.ProgramData
        $env:WINDIR = $state.Windows
        Invoke-DryRun | Out-Null
    } finally { $env:ProgramData = $oldProgramData; $env:WINDIR = $oldWindows }
    Assert-True ((Get-FixtureSnapshot $state.Root) -ceq $before) 'DryRun changed keys/config/ACLs'
}
Test-Case 'repeat apply preserves existing key content without duplicating supplied key' {
    $state = New-SetupFixture 'repeat'
    Invoke-MockedSetup $state | Out-Null
    Invoke-MockedSetup $state | Out-Null
    $keys = [IO.File]::ReadAllText((Join-Path $state.Profile '.ssh/authorized_keys'))
    Assert-True ($keys.Contains('# existing authorized keys')) 'Repeat setup lost existing authorized-key content'
    Assert-True (@($keys -split '\r?\n' | Where-Object { $_ -ceq $publicKey }).Count -eq 1) 'Repeat setup duplicated supplied key'
}
function Get-LiveFixtureState([hashtable]$State) {
    $result = @()
    foreach ($path in @((Join-Path $State.ProgramData 'ssh/sshd_config'), (Join-Path $State.Profile '.ssh/authorized_keys'))) {
        $acl = 'fixture-original-acl'
        if ($State.Acls.ContainsKey($path)) { $acl = $State.Acls[$path] }
        $result += [pscustomobject]@{Path = $path; Bytes = [Convert]::ToBase64String([IO.File]::ReadAllBytes($path)); Acl = $acl}
    }
    $directoryPath = Join-Path $State.Profile '.ssh'
    $directoryAcl = 'fixture-original-acl'
    if ($State.Acls.ContainsKey($directoryPath)) { $directoryAcl = $State.Acls[$directoryPath] }
    $result += [pscustomobject]@{DirectoryAcl = $directoryAcl}
    $result += [pscustomobject]@{ServiceStatus = $State.Service.Status; ServiceStartType = $State.Service.StartType; RuleCount = $State.Rules.Count}
    return ConvertTo-Json -InputObject @($result) -Depth 5 -Compress
}
Test-Case 'firewall failure refuses authorization and preserves live bytes, ACLs and service state' {
    $state = New-SetupFixture 'firewall-rollback'
    $state.FailFirewall = $true
    $before = Get-LiveFixtureState $state
    $result = Invoke-MockedSetup $state -Failure
    Assert-True ($result.Error -match 'firewall creation failed') ('Did not reach injected firewall failure: ' + $result.Error)
    Assert-True ($state.Events -contains 'FirewallCreate') 'Firewall failure injection was not exercised'
    Assert-True ((Get-LiveFixtureState $state) -ceq $before) 'Firewall failure left changed key/config/ACL/service/firewall state'
}
Test-Case 'service failure after key stage removes new rule and restores live state' {
    $state = New-SetupFixture 'service-rollback'
    $state.FailService = $true
    $before = Get-LiveFixtureState $state
    $result = Invoke-MockedSetup $state -Failure
    Assert-True ($result.Error -match 'service restart failed') ('Did not reach injected service failure: ' + $result.Error)
    Assert-True ($state.Events -contains 'ServiceRestart') 'Service failure injection was not exercised'
    Assert-True ($state.Events -contains ('ACL:' + (Join-Path $state.Profile '.ssh/authorized_keys'))) 'Service failure happened before authorization stage'
    Assert-True ((Get-LiveFixtureState $state) -ceq $before) 'Service failure left changed key/config/ACL/service/firewall state'
}
Test-Case 'Include directives refuse automated setup before live changes' {
    $state = New-SetupFixture 'include-refused'
    Write-Utf8 (Join-Path $state.ProgramData 'ssh/sshd_config') "Include conf.d/*.conf`nPort 22`n"
    $before = Get-LiveFixtureState $state
    $result = Invoke-MockedSetup $state -Failure
    Assert-True ($result.Error -match 'Include') ('Include refusal missing: ' + $result.Error)
    Assert-True ((Get-LiveFixtureState $state) -ceq $before) 'Include refusal changed live state'
}
foreach ($serviceCase in @(
    @{Status = 'Running'; StartType = 'Automatic'},
    @{Status = 'Stopped'; StartType = 'Disabled'}
)) {
    Test-Case ('late failure restores service ' + $serviceCase.Status + '/' + $serviceCase.StartType) {
        $state = New-SetupFixture ('service-' + $serviceCase.Status + '-' + $serviceCase.StartType)
        $state.Service.Status = $serviceCase.Status
        $state.Service.StartType = $serviceCase.StartType
        $state.FailService = $true
        $before = Get-LiveFixtureState $state
        $result = Invoke-MockedSetup $state -Failure
        Assert-True ($result.Error -match 'service restart failed') ('Wrong failure: ' + $result.Error)
        Assert-True ((Get-LiveFixtureState $state) -ceq $before) 'Service mode/status or authorization not restored'
    }
}
$firewallCases = @(
    @{Name = 'wildcard application package remains broad'; Program = 'Any'; Package = '*'; Service = 'Any'; Protocol = 'TCP'; Port = '22'; Remote = 'Any'; Profile = 'Private'; Conflict = $true},
    @{Name = 'package family with null application package'; PackageFamilyName = 'Microsoft.AAD.BrokerPlugin_cw5n1h2txyewy'; Program = 'Any'; Package = $null; Service = 'Any'; Protocol = 'Any'; Port = 'Any'; Remote = 'Any'; Profile = 'Domain, Private'; Conflict = $false},
    @{Name = 'empty package family remains broad'; PackageFamilyName = ''; Program = 'Any'; Package = $null; Service = 'Any'; Protocol = 'TCP'; Port = '22'; Remote = 'Any'; Profile = 'Private'; Conflict = $true},
    @{Name = 'null package family remains broad'; PackageFamilyName = $null; Program = 'Any'; Package = $null; Service = 'Any'; Protocol = 'TCP'; Port = '22'; Remote = 'Any'; Profile = 'Private'; Conflict = $true},
    @{Name = 'Any package family remains broad'; PackageFamilyName = 'Any'; Program = 'Any'; Package = $null; Service = 'Any'; Protocol = 'TCP'; Port = '22'; Remote = 'Any'; Profile = 'Private'; Conflict = $true},
    @{Name = 'packaged app catchall'; Program = 'Any'; Package = 'S-1-15-2-12345'; Service = 'Any'; Protocol = 'TCP'; Port = 'Any'; Remote = 'Any'; Profile = 'Any'; Conflict = $false},
    @{Name = 'browser catchall'; Program = 'C:\Apps\browser.exe'; Service = 'Any'; Protocol = 'TCP'; Port = 'Any'; Remote = 'Any'; Profile = 'Any'; Conflict = $false},
    @{Name = 'unrelated service'; Program = 'Any'; Service = 'Dnscache'; Protocol = 'TCP'; Port = 'Any'; Remote = 'Any'; Profile = 'Any'; Conflict = $false},
    @{Name = 'UDP22'; Program = 'Any'; Service = 'Any'; Protocol = 'UDP'; Port = '22'; Remote = 'Any'; Profile = 'Any'; Conflict = $false},
    @{Name = 'TCP22 broad'; Program = 'Any'; Service = 'Any'; Protocol = 'TCP'; Port = '22'; Remote = 'Any'; Profile = 'Private'; Conflict = $true},
    @{Name = 'TCP range includes22'; Program = 'Any'; Service = 'Any'; Protocol = 'TCP'; Port = '20-24'; Remote = 'Any'; Profile = 'Private'; Conflict = $true},
    @{Name = 'sshd catchall'; Program = 'C:\Windows\System32\OpenSSH\sshd.exe'; Service = 'Any'; Protocol = 'TCP'; Port = 'Any'; Remote = 'Any'; Profile = 'Private'; Conflict = $true},
    @{Name = 'safe existing subnet'; Program = 'Any'; Service = 'Any'; Protocol = 'TCP'; Port = '22'; Remote = 'LocalSubnet'; Profile = 'Private'; Conflict = $false},
    @{Name = 'Public outside optin'; Program = 'Any'; Service = 'Any'; Protocol = 'TCP'; Port = '22'; Remote = 'LocalSubnet'; Profile = 'Private, Public'; Conflict = $true}
)
$firewallIndex = 0
foreach ($firewallCase in $firewallCases) {
    $firewallIndex++
    Test-Case ('existing firewall scope audit: ' + $firewallCase.Name) {
        $state = New-SetupFixture ('firewall-audit-' + $firewallIndex)
        $package = 'Any'
        if ($firewallCase.ContainsKey('Package')) { $package = $firewallCase.Package }
        $operatorRule = [pscustomobject]@{
            Name = 'operator-rule'; Profile = $firewallCase.Profile; RemoteAddress = $firewallCase.Remote
            LocalPort = $firewallCase.Port; Protocol = $firewallCase.Protocol
            Program = $firewallCase.Program; Package = $package; Service = $firewallCase.Service
            Enabled = 'True'; Direction = 'Inbound'; Action = 'Allow'
        }
        if ($firewallCase.ContainsKey('PackageFamilyName')) {
            $operatorRule | Add-Member -NotePropertyName PackageFamilyName -NotePropertyValue $firewallCase.PackageFamilyName
        }
        $state.Rules[$operatorRule.Name] = $operatorRule
        $ruleBefore = $operatorRule | ConvertTo-Json -Compress
        $before = Get-LiveFixtureState $state
        $result = Invoke-MockedSetup $state -Failure:$firewallCase.Conflict
        if ($firewallCase.Conflict) {
            Assert-True ($result.Error -match 'firewall policy') ('Wrong conflict error: ' + $result.Error)
            Assert-True ((Get-LiveFixtureState $state) -ceq $before) 'Unsafe existing firewall policy was not refused before live changes'
        }
        Assert-True (($state.Rules['operator-rule'] | ConvertTo-Json -Compress) -ceq $ruleBefore) 'Operator firewall rule was changed'
    }
}
foreach ($firewallProfileCase in @(
    @{Name = 'disabled'; Enabled = 'False'; Action = 'Block'; Empty = $false},
    @{Name = 'default allow'; Enabled = 'True'; Action = 'Allow'; Empty = $false},
    @{Name = 'unknown'; Enabled = 'True'; Action = 'NotConfigured'; Empty = $false},
    @{Name = 'absent profiles'; Enabled = 'True'; Action = 'Block'; Empty = $true}
)) {
    Test-Case ('effective firewall profile refuses ' + $firewallProfileCase.Name) {
        $state = New-SetupFixture ('firewall-profile-' + $firewallProfileCase.Name.Replace(' ', '-'))
        $state.FirewallEnabled = $firewallProfileCase.Enabled
        $state.DefaultInboundAction = $firewallProfileCase.Action
        $state.EmptyFirewallProfiles = $firewallProfileCase.Empty
        $before = Get-LiveFixtureState $state
        $result = Invoke-MockedSetup $state -Failure
        Assert-True ($result.Error -match 'firewall') ('Wrong firewall-profile refusal: ' + $result.Error)
        Assert-True ((Get-LiveFixtureState $state) -ceq $before) 'Uncertain effective firewall profile changed live state'
    }
}
$markerCases = @(
    @{Name = 'edited DenyUsers'; Text = "# BEGIN oneline-ccc key-only sshfixture`nMatch User sshfixture`n    DenyUsers sshfixture`n# END oneline-ccc key-only sshfixture`n"},
    @{Name = 'missing end'; Text = "# BEGIN oneline-ccc key-only sshfixture`nMatch User sshfixture`n"},
    @{Name = 'missing begin'; Text = "# END oneline-ccc key-only sshfixture`n"},
    @{Name = 'duplicate'; Text = ("# BEGIN oneline-ccc key-only sshfixture`nMatch User sshfixture`n# END oneline-ccc key-only sshfixture`n" * 2)}
)
$markerIndex = 0
foreach ($markerCase in $markerCases) {
    $markerIndex++
    Test-Case ('managed authorization block refuses ' + $markerCase.Name) {
        $state = New-SetupFixture ('marker-' + $markerIndex)
        Write-Utf8 (Join-Path $state.ProgramData 'ssh/sshd_config') ("Port 22`n" + $markerCase.Text)
        $before = Get-LiveFixtureState $state
        $result = Invoke-MockedSetup $state -Failure
        Assert-True ($result.Error -match 'managed block|marker') ('Wrong managed block refusal: ' + $result.Error)
        Assert-True ((Get-LiveFixtureState $state) -ceq $before) 'Refused managed block changed live state'
    }
}
# FINAL_RESULTS
Write-Output ('Results: ' + $script:passed + ' passed; ' + $script:failures.Count + ' failed. Artifacts: ' + $root)
if ($script:failures.Count -gt 0) { exit 1 }
