#requires -Version 5.1
# Destructive fixture preparation: disposable GitHub-hosted Windows runners ONLY.
# This is a reproduction probe, not evidence of a known cause or a repair.
param([switch]$SeedExistingHostKeys)
$ErrorActionPreference = 'Stop'
if ($env:GITHUB_ACTIONS -ne 'true' -or $env:OS -ne 'Windows_NT' -or $env:RUNNER_ENVIRONMENT -ne 'github-hosted') {
    throw 'Fresh SSH probe requires a disposable GitHub-hosted Windows runner.'
}
if (-not [Environment]::Is64BitProcess -or -not $env:RUNNER_TEMP) { throw 'Native 64-bit PowerShell and RUNNER_TEMP are required.' }
$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($identity)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { throw 'Runner must already be elevated.' }
$account = Get-LocalUser -SID $identity.User
$userName = $account.Name.ToLowerInvariant()
if (-not $account.Enabled -or $userName -notmatch '^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,19}$' -or
    -not @(Get-LocalGroupMember -SID 'S-1-5-32-544' | Where-Object { $_.SID.Value -eq $identity.User.Value }).Count) {
    throw 'Current identity must be an enabled local account directly in Administrators, as expected by the helper.'
}

$repository = Split-Path $PSScriptRoot -Parent
$started = Get-Date
$sshBin = Join-Path $env:WINDIR 'System32\OpenSSH'
$sshd = Join-Path $sshBin 'sshd.exe'
$sshRoot = Join-Path $env:ProgramData 'ssh'
$config = Join-Path $sshRoot 'sshd_config'
$fixture = Join-Path $env:RUNNER_TEMP ('oneline-ssh-fresh-' + [guid]::NewGuid().ToString('N'))
$keyPath = Join-Path $fixture 'client_ed25519'

function Invoke-BoundedNative([string]$Executable, [string]$Arguments, [int]$Seconds = 30) {
    # Process redirection avoids PS5.1 turning ordinary native stderr into a
    # terminating NativeCommandError under ErrorActionPreference=Stop.
    $info = New-Object Diagnostics.ProcessStartInfo
    $info.FileName = $Executable
    $info.Arguments = $Arguments
    $info.UseShellExecute = $false
    $info.CreateNoWindow = $true
    $info.RedirectStandardOutput = $true
    $info.RedirectStandardError = $true
    $process = [Diagnostics.Process]::Start($info)
    try {
        $stdout = $process.StandardOutput.ReadToEndAsync()
        $stderr = $process.StandardError.ReadToEndAsync()
        if (-not $process.WaitForExit($Seconds * 1000)) {
            $process.Kill()
            [void]$process.WaitForExit(5000)
            throw ('Native fixture command exceeded {0}s: {1}' -f $Seconds, [IO.Path]::GetFileName($Executable))
        }
        if (-not $stdout.Wait(5000) -or -not $stderr.Wait(5000)) { throw 'Native fixture output did not close within 5s.' }
        [pscustomobject]@{ ExitCode = $process.ExitCode; Stdout = $stdout.Result; Stderr = $stderr.Result }
    } finally { $process.Dispose() }
}

function Write-Diagnostic([string]$Label, [scriptblock]$Body) {
    Write-Output ('Diagnostic: ' + $Label)
    try { & $Body }
    catch { Write-Output ('Diagnostic unavailable: ' + $_.Exception.Message) }
}

function Write-RecentEvents([string]$Log, [string]$Provider, [string]$Pattern) {
    $filter = @{ LogName = $Log; StartTime = $started }
    if ($Provider) { $filter.ProviderName = $Provider }
    Write-Diagnostic $Log {
        Get-WinEvent -FilterHashtable $filter -MaxEvents 40 -ErrorAction Stop |
            Where-Object { -not $Pattern -or $_.Message -match $Pattern } |
            ForEach-Object { Write-Output ('{0:o} [{1}] {2}: {3}' -f $_.TimeCreated, $_.ProviderName, $_.Id, $_.Message) }
    }
}

function Write-SystemSyntaxDiagnostic([string]$SyntaxConfig) {
    # Diagnostic only: -t never listens. No script runs with elevated identity;
    # the scheduled action directly executes the native Windows binary.
    $taskName = 'Oneline-CCC-Fresh-Syntax-' + [guid]::NewGuid().ToString('N')
    $directory = Join-Path $env:ProgramData $taskName
    $log = Join-Path $directory 'syntax.log'
    New-Item -ItemType Directory -Path $directory | Out-Null
    $acl = New-Object Security.AccessControl.DirectorySecurity
    $acl.SetAccessRuleProtection($true, $false)
    $acl.SetOwner((New-Object Security.Principal.SecurityIdentifier('S-1-5-32-544')))
    foreach ($sid in @('S-1-5-18', 'S-1-5-32-544')) {
        $rule = New-Object Security.AccessControl.FileSystemAccessRule(
            (New-Object Security.Principal.SecurityIdentifier($sid)), 'FullControl',
            'ContainerInherit,ObjectInherit', 'None', 'Allow')
        $acl.AddAccessRule($rule)
    }
    Set-Acl -LiteralPath $directory -AclObject $acl
    $registered = $false
    try {
        $action = New-ScheduledTaskAction -Execute $sshd -Argument ('-ddd -t -f "' + $SyntaxConfig + '" -E "' + $log + '"')
        $principal = New-ScheduledTaskPrincipal -UserId 'S-1-5-18' -LogonType ServiceAccount -RunLevel Highest
        $settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit (New-TimeSpan -Seconds 30)
        Register-ScheduledTask -TaskName $taskName -Action $action -Principal $principal -Settings $settings | Out-Null
        $registered = $true
        Start-ScheduledTask -TaskName $taskName
        $deadline = [DateTime]::UtcNow.AddSeconds(40)
        do {
            $state = [string](Get-ScheduledTask -TaskName $taskName).State
            if ($state -eq 'Ready') {
                # Query results after observing completion; an earlier query can
                # race the short action and report its in-progress status.
                $info = Get-ScheduledTaskInfo -TaskName $taskName
                if ($info.LastRunTime.Year -ge 2000 -and $info.LastTaskResult -notin @(0x41301, 0x41303, 0x41325)) { break }
            }
            if ([DateTime]::UtcNow -ge $deadline) { throw 'SYSTEM syntax diagnostic exceeded 40s.' }
            Start-Sleep -Milliseconds 200
        } while ($true)
        Write-Output ('SYSTEM syntax task result: ' + $info.LastTaskResult)
        if (Test-Path -LiteralPath $log) { Get-Content -LiteralPath $log -TotalCount 100 | Write-Output }
        else { Write-Output 'SYSTEM syntax log absent; native execution was not established.' }
    } finally {
        if ($registered) {
            Stop-ScheduledTask -TaskName $taskName
            if ([string](Get-ScheduledTask -TaskName $taskName).State -in @('Running', 'Queued')) {
                throw ('SYSTEM syntax task cleanup unconfirmed; protected directory retained: ' + $directory)
            }
            Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
        }
        Remove-Item -LiteralPath $directory -Recurse -Force
    }
}

function Write-FailureDiagnostics {
    # Called only after the production helper has returned/thrown, including
    # its own rollback. No daemon is launched and no ACL is changed here.
    Write-Diagnostic 'sshd service after helper return/rollback' {
        Get-CimInstance Win32_Service -Filter "Name='sshd'" |
            Select-Object State, StartMode, StartName, PathName, ExitCode, ServiceSpecificExitCode, ProcessId |
            Format-List | Out-String | Write-Output
    }
    Write-Diagnostic 'Windows and OpenSSH versions' {
        Get-CimInstance Win32_OperatingSystem | Select-Object Caption, Version, BuildNumber, OSArchitecture |
            Format-List | Out-String | Write-Output
        (Get-Item -LiteralPath $sshd).VersionInfo | Select-Object FileVersion, ProductVersion |
            Format-List | Out-String | Write-Output
    }
    Write-Diagnostic 'TCP 22 listeners' {
        $listeners = @(Get-NetTCPConnection -State Listen -LocalPort 22 -ErrorAction SilentlyContinue)
        if (-not $listeners.Count) { Write-Output 'No TCP 22 listener.' }
        $listeners | Select-Object LocalAddress, LocalPort, OwningProcess | Format-Table | Out-String | Write-Output
    }
    Write-RecentEvents 'OpenSSH/Admin' '' ''
    Write-RecentEvents 'OpenSSH/Operational' '' ''
    Write-RecentEvents 'System' 'Service Control Manager' 'sshd|OpenSSH'
    Write-RecentEvents 'Application' 'Application Error' 'sshd|OpenSSH'
    Write-RecentEvents 'Application' 'Windows Error Reporting' 'sshd|OpenSSH'
    Write-Diagnostic 'SSH root, logs, config and hostkey ACL metadata only' {
        $paths = @($sshRoot, (Join-Path $sshRoot 'logs'), $config)
        if (Test-Path -LiteralPath $sshRoot) {
            $paths += @(Get-ChildItem -LiteralPath $sshRoot -Filter 'ssh_host_*_key*' -File -Force | Select-Object -ExpandProperty FullName)
        }
        foreach ($path in $paths) {
            Write-Diagnostic $path {
                if (Test-Path -LiteralPath $path) {
                    $acl = Get-Acl -LiteralPath $path
                    Write-Output ('Owner={0}; Protected={1}; SDDL={2}' -f $acl.Owner, $acl.AreAccessRulesProtected, $acl.Sddl)
                } else { Write-Output 'Path absent after helper return/rollback.' }
            }
        }
    }
    Write-Diagnostic 'bounded verbose sshd syntax check (no listener)' {
        $syntaxConfig = $config
        if (-not (Test-Path -LiteralPath $syntaxConfig)) { $syntaxConfig = Join-Path $sshBin 'sshd_config_default' }
        Write-Output ('Syntax configuration: ' + $syntaxConfig)
        $syntax = Invoke-BoundedNative $sshd ('-ddd -t -f "' + $syntaxConfig + '"') 20
        Write-Output ('sshd -ddd -t exit: ' + $syntax.ExitCode)
        Write-Output $syntax.Stdout
        Write-Output $syntax.Stderr
        Write-SystemSyntaxDiagnostic $syntaxConfig
    }
}

$stopOnExit = $false
try {
    $stopOnExit = $true
    $capability = Get-WindowsCapability -Online -Name 'OpenSSH.Server~~~~0.0.1.0'
    if ($capability.State -ne 'Installed') {
        $install = Add-WindowsCapability -Online -Name 'OpenSSH.Server~~~~0.0.1.0'
        if ($install.RestartNeeded) { throw 'OpenSSH capability requires a reboot; fresh startup was not tested.' }
    }
    foreach ($name in @('sshd.exe', 'ssh.exe', 'ssh-keygen.exe', 'sshd_config_default')) {
        if (-not (Test-Path -LiteralPath (Join-Path $sshBin $name))) { throw ('Missing built-in OpenSSH file: ' + $name) }
    }
    # Stop before moving the directory so helper rollback records Stopped,
    # never a previously running password-policy service to restore.
    Stop-Service -Name sshd -Force
    (Get-Service sshd).WaitForStatus('Stopped', [TimeSpan]::FromSeconds(15))
    if ((Get-Service sshd).Status -ne 'Stopped') { throw 'Cannot prepare a fresh SSH directory while sshd is active.' }

    # Same disposable-runner firewall preparation as the existing native smoke.
    $baseline = Get-NetFirewallRule -PolicyStore PersistentStore -Name 'OpenSSH-Server-In-TCP' -ErrorAction SilentlyContinue
    if ($baseline) { Disable-NetFirewallRule -PolicyStore PersistentStore -Name 'OpenSSH-Server-In-TCP' }
    Set-NetFirewallProfile -Profile Domain,Private,Public -Enabled True -DefaultInboundAction Block

    if (Test-Path -LiteralPath $sshRoot) {
        if ((Get-Item -LiteralPath $sshRoot -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Refusing a reparse-point SSH fixture root.' }
        $retainedName = 'ssh.oneline-fresh-retained-' + [guid]::NewGuid().ToString('N')
        Rename-Item -LiteralPath $sshRoot -NewName $retainedName
        Write-Output ('Prior SSH directory retained intact at: ' + (Join-Path $env:ProgramData $retainedName))
    }
    if (Test-Path -LiteralPath $sshRoot) { throw 'Fresh SSH root still exists before helper setup.' }
    # Hosted images also patch the built-in fallback template. Remove only
    # their known AllowGroups line from fixture data, as the existing native
    # smoke does. Keep fresh default directory ACLs and no hostkeys; this also
    # models retrying a previously failed first run. Never change the OS template.
    $template = [IO.File]::ReadAllText((Join-Path $sshBin 'sshd_config_default'))
    $imageRule = '(?im)^[ \t]*AllowGroups[ \t]+administrators[ \t]+"openssh users"[ \t]*\r?$'
    if ([regex]::Matches($template, $imageRule).Count -eq 1) {
        New-Item -ItemType Directory -Path $sshRoot | Out-Null
        [IO.File]::WriteAllText($config, [regex]::Replace($template, $imageRule, ''), (New-Object Text.UTF8Encoding($false)))
        Write-Output 'Fixture only: omitted the hosted-image AllowGroups entry from a fresh config; directory ACLs remain untouched and hostkeys are absent.'
    }
    [IO.Directory]::CreateDirectory($fixture) | Out-Null
    $keygen = Invoke-BoundedNative (Join-Path $sshBin 'ssh-keygen.exe') ('-q -t ed25519 -f "' + $keyPath + '" -N "" -C "oneline-fresh-fixture"')
    if ($keygen.ExitCode -ne 0) { throw ('Synthetic client key generation failed: ' + $keygen.Stderr) }

    $priorPublicKeys = @{}
    if ($SeedExistingHostKeys) {
        # Models a retry after the old helper generated hostkeys and failed.
        # Native keygen owns private contents; the fixture reads public files only.
        $seed = Invoke-BoundedNative (Join-Path $sshBin 'ssh-keygen.exe') '-A'
        if ($seed.ExitCode -ne 0) { throw ('Existing hostkey fixture generation failed: ' + $seed.Stderr) }
        foreach ($type in @('rsa', 'ecdsa', 'ed25519')) {
            $publicPath = Join-Path $sshRoot ('ssh_host_' + $type + '_key.pub')
            $priorPublicKeys[$publicPath] = [IO.File]::ReadAllText($publicPath)
        }
        Write-Output 'Fixture: administrator-generated hostkeys already exist before helper setup.'
    }

    # No account creation, production shims, or real GitHub keys.
    # SID equality above forces the helper's current-account -T policy branch.
    & (Join-Path $repository 'scripts/setup-ssh-server.ps1') -UserName $userName -PublicKeyFile ($keyPath + '.pub') -NonInteractive
    $serviceRunning = (Get-Service sshd).Status -eq 'Running'
    if (-not $serviceRunning) { throw 'Fresh administrator setup did not leave sshd Running.' }
    foreach ($publicPath in $priorPublicKeys.Keys) {
        if ([IO.File]::ReadAllText($publicPath) -cne $priorPublicKeys[$publicPath]) { throw 'Retry replaced a preexisting public host key.' }
    }
    foreach ($type in @('rsa', 'ecdsa', 'ed25519')) {
        foreach ($suffix in @('', '.pub')) {
            if (-not (Test-Path -LiteralPath (Join-Path $sshRoot ('ssh_host_' + $type + '_key' + $suffix)))) {
                throw ('Helper did not generate fresh ' + $type + ' hostkey files.')
            }
        }
    }

    # Ignore ambient client config/agents; require only this synthetic key.
    $knownHosts = Join-Path $fixture 'known_hosts'
    $arguments = '-vv -F NUL -o IdentityAgent=none -o IdentitiesOnly=yes -o BatchMode=yes -o PreferredAuthentications=publickey -o PasswordAuthentication=no -o ConnectTimeout=15 -o ConnectionAttempts=1 -o StrictHostKeyChecking=accept-new -o "UserKnownHostsFile=' + $knownHosts + '" -i "' + $keyPath + '" -l ' + $userName + ' 127.0.0.1 whoami /user /fo csv /nh'
    $login = Invoke-BoundedNative (Join-Path $sshBin 'ssh.exe') $arguments 30
    if ($login.ExitCode -ne 0) {
        Write-Output ('Loopback SSH exit: ' + $login.ExitCode)
        Write-Output $login.Stderr
        throw 'Real loopback public-key login failed after fresh administrator setup.'
    }
    $remoteIdentity = @($login.Stdout | ConvertFrom-Csv -Header Account, Sid)
    if ($remoteIdentity.Count -ne 1 -or $remoteIdentity[0].Sid -ne $identity.User.Value) { throw 'Loopback SSH authenticated an unexpected account SID.' }
    if ((Get-Service sshd).Status -ne 'Running') { throw 'sshd stopped during the loopback login.' }
} catch {
    $originalFailure = $_
    Write-Output ('Fresh SSH probe failed: ' + $originalFailure.Exception.Message)
    try { Write-FailureDiagnostics }
    catch { Write-Output ('Failure diagnostics could not complete: ' + $_.Exception.Message) }
    throw $originalFailure
} finally {
    # Leave the old directory intact. Never restore or restart the old service.
    # Diagnostics above observe helper rollback before this final shutdown.
    if ($stopOnExit) {
        try {
            Stop-Service -Name sshd -Force
            Set-Service -Name sshd -StartupType Manual
        } catch { Write-Warning ('Disposable runner SSH shutdown failed: ' + $_.Exception.Message) }
    }
}
Write-Output 'Fresh RSA/ECDSA/ED25519 hostkeys, current-administrator policy branch, serviceRunning=True and real loopback public-key login as the intended SID verified.'
