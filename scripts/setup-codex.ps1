#requires -Version 5.1
<#
.SYNOPSIS
Configure native Codex to use LLMP by default and through its named profile.
.DESCRIPTION
Run locally with -File or invoke a downloaded scriptblock. Supply LLMP_API_KEY
through the environment for automation; interactive entry uses a secure prompt.
-DryRun prints the proposed managed block without changing files, environment,
or software. -SkipInstall permits isolated configuration tests on non-Windows.
ContextWindow is an operator-selected budget, not a discovered model capability.
#>
[CmdletBinding()]
param(
    [string]$BaseUrl,
    [System.Security.SecureString]$ApiKey,
    [string]$Model,
    [long]$ContextWindow = 0,
    [long]$AutoCompactTokenLimit = 0,
    [string]$ServiceTier,
    [string[]]$SupportedServiceTiers = @(),
    [string]$ModelCatalogPath,
    [switch]$NonInteractive,
    [switch]$SkipInstall,
    [switch]$DryRun,
    [switch]$InstallNode,
    [switch]$SkipPersistCredential,
    [switch]$DiscoverModels,
    [switch]$EnableSshServer
)

$ErrorActionPreference = 'Stop'
$beginMarker = '# BEGIN oneline-ccc LLMP managed configuration'
$endMarker = '# END oneline-ccc LLMP managed configuration'

function Read-RequiredText([string]$Value, [string]$Prompt) {
    if (-not [string]::IsNullOrWhiteSpace($Value)) { return $Value.Trim() }
    if ($NonInteractive) { throw 'Required input missing. Supply BaseUrl, Model and ContextWindow explicitly in noninteractive mode.' }
    $answer = Read-Host $Prompt
    if ([string]::IsNullOrWhiteSpace($answer)) { throw 'Required input was empty.' }
    return $answer.Trim()
}

function ConvertTo-TomlString([string]$Value) {
    # A literal TOML string needs no escaping; reject delimiters and controls.
    if ($Value.Contains("'") -or $Value -match '[\x00-\x1f\x7f]') {
        throw 'Configuration values must not contain single quotes or control characters.'
    }
    return "'" + $Value + "'"
}

function Get-ValidatedEndpoint([string]$Url, [switch]$ExistingProfile) {
    $endpoint = $null
    if (-not [Uri]::TryCreate($Url, [UriKind]::Absolute, [ref]$endpoint) -or
        $endpoint.UserInfo -or $endpoint.Query -or $endpoint.Fragment -or
        ($endpoint.Scheme -ne 'https' -and -not ($endpoint.Scheme -eq 'http' -and $endpoint.IsLoopback))) {
        if ($ExistingProfile) { throw 'Existing config profile contains an unsafe base URL. Remove credentials, query and fragment; use HTTPS or HTTP loopback before rerunning. No backup was created.' }
        throw 'BaseUrl must be HTTPS (HTTP is allowed only for loopback), without credentials, query or fragment.'
    }
    return $endpoint
}

function Get-ConfigSnapshot([string]$Path) {
    $exists = [IO.File]::Exists($Path)
    $bytes = $null
    $text = ''
    if ($exists) {
        $bytes = [IO.File]::ReadAllBytes($Path)
        $stream = [IO.MemoryStream]::new($bytes, $false)
        $reader = [IO.StreamReader]::new($stream, [Text.Encoding]::UTF8, $true)
        try { $text = $reader.ReadToEnd() }
        finally { $reader.Dispose(); $stream.Dispose() }
    } elseif ([IO.Directory]::Exists($Path)) { throw 'Existing config path is a directory. Choose a writable CODEX_HOME.' }
    return @{Existed = $exists; Bytes = $bytes; Text = $text}
}

function Get-UpdatedConfig([string]$Existing, [string]$Block) {
    if (-not [string]::IsNullOrWhiteSpace($Existing)) {
        $lines = @($Existing -split '\r?\n')
        $nonempty = @($lines | Where-Object { $_.Trim().Length -gt 0 })
        if ($nonempty[0] -ne $beginMarker -or $nonempty[-1] -ne $endMarker -or
            @($lines | Where-Object { $_ -eq $beginMarker }).Count -ne 1 -or
            @($lines | Where-Object { $_ -eq $endMarker }).Count -ne 1) {
            throw 'Existing config profile is not owned by this installer. Rename llmp.config.toml or use a separate CODEX_HOME.'
        }
        # Only replace the exact shape we generate. Refuse hand-added settings
        # rather than dropping them, even inside an installer-owned file.
        $generatedShape = '\A' + [regex]::Escape($beginMarker) + '\r?\n' +
            "model_provider = 'oneline_llmp'\r?\nmodel = '[^'\r\n]*'\r?\n" +
            'model_context_window = [0-9]+\r?\nmodel_auto_compact_token_limit = [0-9]+\r?\n' +
            "(?:service_tier = '[^'\r\n]*'\r?\n)?(?:model_catalog_json = '[^'\r\n]*'\r?\n)?\r?\n" +
            "\[model_providers\.oneline_llmp\]\r?\nname = 'LLMP \(oneline-ccc\)'\r?\n" +
            "base_url = '[^'\r\n]*'\r?\nwire_api = 'responses'\r?\nenv_key = 'LLMP_API_KEY'\r?\n" +
            'requires_openai_auth = false\r?\n\r?\n' + [regex]::Escape($endMarker) + '\r?\n?\z'
        if ($Existing -cnotmatch $generatedShape) {
            throw 'Existing config profile has manual changes. Preserve them in a different profile before rerunning setup.'
        }
        $existingUrl = [regex]::Match($Existing, "(?m)^base_url = '([^'\r\n]*)'\r?$").Groups[1].Value
        Get-ValidatedEndpoint $existingUrl -ExistingProfile | Out-Null
        # Generated profiles never contain inline secrets. Refuse backup of an
        # owned profile that has acquired credential-like settings by hand.
        if ($Existing -match '(?im)^\s*["'']?(api_key|experimental_bearer_token|authorization|http_headers)["'']?\s*[=.]') {
            throw 'Existing config contains inline credentials. Move credentials to environment variables before creating a backup.'
        }
    }
    return $Block + "`n"
}

function Get-CodexCommand {
    if ($env:OS -eq 'Windows_NT') {
        $command = Get-Command codex.exe -CommandType Application -ErrorAction SilentlyContinue
        if (-not $command) { $command = Get-Command codex.cmd -CommandType Application -ErrorAction SilentlyContinue }
        return $command
    }
    return Get-Command codex -ErrorAction SilentlyContinue
}

function Get-ModelIds([string]$Url, [System.Security.SecureString]$Key) {
    Add-Type -AssemblyName System.Net.Http
    $handler = New-Object System.Net.Http.HttpClientHandler
    $handler.AllowAutoRedirect = $false
    $client = New-Object System.Net.Http.HttpClient($handler)
    $client.Timeout = [TimeSpan]::FromSeconds(20)
    $pointer = [IntPtr]::Zero
    try {
        $pointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($Key)
        $token = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($pointer)
        $client.DefaultRequestHeaders.Authorization = New-Object System.Net.Http.Headers.AuthenticationHeaderValue('Bearer', $token)
        $response = $client.GetAsync($Url + '/models').GetAwaiter().GetResult()
        try {
            if (-not $response.IsSuccessStatusCode) { throw 'Model discovery failed. Redirects are disabled; supply Model explicitly or verify the endpoint.' }
            $payload = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult() | ConvertFrom-Json
            # Only IDs are used. OpenAI-compatible /models does not standardize
            # context sizes or service-tier capabilities.
            return @($payload.data | ForEach-Object { $_.id } | Where-Object {
                $_ -is [string] -and $_ -notmatch '[\x00-\x1f\x7f]' -and $_.Length -gt 0
            } | Select-Object -Unique)
        } finally { $response.Dispose() }
    } finally {
        $token = $null
        if ($pointer -ne [IntPtr]::Zero) { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($pointer) }
        $client.Dispose()
        $handler.Dispose()
    }
}

function Assert-CodexVersion {
    if (-not (Get-CodexCommand)) {
        if ($SkipInstall) { return }
        throw 'Codex installed but was not found on PATH. Reopen PowerShell.'
    }
    $codexCommand = Get-CodexCommand
    $previousErrorAction = $ErrorActionPreference
    try {
        # Windows PowerShell 5.1 treats native stderr as an error even when
        # redirected. A fresh CODEX_HOME can cause a harmless PATH warning.
        $ErrorActionPreference = 'Continue'
        $versionOutput = (& $codexCommand.Source --version 2>$null | Out-String)
        $versionExitCode = $LASTEXITCODE
    } finally { $ErrorActionPreference = $previousErrorAction }
    if ($versionExitCode -ne 0 -or $versionOutput -notmatch '(\d+\.\d+\.\d+)') {
        throw 'Codex installed but its version could not be verified. Install current @openai/codex using npm.'
    }
    if ([version]$Matches[1] -lt [version]'0.134.0') {
        throw 'Codex installed is older than 0.134.0. Upgrade using npm install --global @openai/codex, then rerun.'
    }
}

function Get-CatalogTiers([string]$Path, [string]$Slug) {
    try {
        $catalog = [IO.File]::ReadAllText($Path) | ConvertFrom-Json
        if ($null -eq $catalog -or $catalog -is [Array] -or $catalog.models -isnot [Array]) { throw 'invalid' }
        $selected = @($catalog.models | Where-Object { $_.slug -is [string] -and $_.slug -ceq $Slug })
        if ($selected.Count -ne 1) { throw 'invalid' }
        $tiers = @()
        foreach ($property in @('service_tiers', 'additional_speed_tiers')) {
            $values = $selected[0].$property
            if ($null -eq $values) { continue }
            if ($values -isnot [Array]) { throw 'invalid' }
            foreach ($item in $values) {
                $tier = $item
                if ($property -eq 'service_tiers') {
                    if ($null -eq $item -or $item -is [string]) { throw 'invalid' }
                    $tier = $item.id
                }
                if ($tier -isnot [string] -or [string]::IsNullOrWhiteSpace($tier) -or $tier -match '[\x00-\x20\x7f]') { throw 'invalid' }
                $tiers += $tier
            }
        }
        return @($tiers | Select-Object -Unique)
    } catch {
        throw 'Model catalog invalid: require readable JSON with models array, exactly one matching slug, service_tiers objects with string ids and legacy additional_speed_tiers strings. Full native catalog validation remains Codex responsibility.'
    }
}

function Add-UserPath([string]$Directory) {
    $env:Path = $Directory + ';' + $env:Path
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (@($userPath -split ';') -notcontains $Directory) {
        [Environment]::SetEnvironmentVariable('Path', ($Directory + ';' + $userPath), 'User')
    }
}

function Install-NodeDependency {
    if (-not (Get-Command winget.exe -ErrorAction SilentlyContinue)) {
        throw 'Install Node.js LTS from https://nodejs.org, reopen PowerShell and rerun setup; winget is unavailable.'
    }
    $approved = $InstallNode
    if (-not $approved -and -not $NonInteractive) {
        $approved = (Read-Host 'Install Node.js LTS using winget (may request administrator approval)? [y/N]') -match '^(?i)y(es)?$'
    }
    if (-not $approved) { throw 'Install Node.js LTS and reopen PowerShell, or rerun with -InstallNode.' }
    & winget.exe install --id OpenJS.NodeJS.LTS --exact --accept-package-agreements --accept-source-agreements | Out-Host
    if ($LASTEXITCODE -ne 0) { throw 'Node.js installation failed or administrator approval was declined. Install Node.js LTS manually and reopen PowerShell.' }
    $machinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $env:Path = $machinePath + ';' + $userPath + ';' + $env:Path
}

function Install-NativeCodex {
    if (Get-CodexCommand) { return }
    if (-not (Get-Command npm.cmd -ErrorAction SilentlyContinue)) { Install-NodeDependency }
    if (-not (Get-Command npm.cmd -ErrorAction SilentlyContinue)) {
        throw 'npm is unavailable after PATH refresh. Reopen PowerShell after installing Node.js LTS and rerun setup.'
    }
    # A user-owned prefix avoids administrator permissions even with a system Node install.
    $npmPrefix = Join-Path $env:APPDATA 'npm'
    & npm.cmd install --global --prefix $npmPrefix '@openai/codex' | Out-Host
    if ($LASTEXITCODE -ne 0) { throw 'Codex npm installation failed. Check connectivity and install @openai/codex with npm.' }
    Add-UserPath $npmPrefix
    if (-not (Get-CodexCommand)) { throw 'Codex installed but was not found on PATH. Reopen PowerShell.' }
}

function Get-NativeCodexExecutable {
    if ($env:OS -ne 'Windows_NT') { return $null }
    $shim = Get-Command codex.cmd -CommandType Application -ErrorAction SilentlyContinue
    if ($shim) {
        $packageRoot = Join-Path ([IO.Path]::GetDirectoryName($shim.Source)) 'node_modules/@openai/codex'
        $node = Get-Command node.exe -CommandType Application -ErrorAction SilentlyContinue
        if ($node -and [IO.File]::Exists((Join-Path $packageRoot 'package.json'))) {
            $resolver = @'
const fs=require('fs'),path=require('path'),{createRequire}=require('module');
const root=path.resolve(process.argv[1]);
const target={x64:'x86_64-pc-windows-msvc',arm64:'aarch64-pc-windows-msvc'}[process.arch];
if(process.platform!=='win32'||!target)process.exit(2);
const resolve=createRequire(path.join(root,'package.json'));
const vendors=[];
try{vendors.push(path.join(path.dirname(resolve.resolve('@openai/codex-win32-'+process.arch+'/package.json')),'vendor'));}catch{}
vendors.push(path.join(root,'vendor'));
for(const vendor of vendors)for(const layout of ['bin','codex']){
 const exe=path.join(vendor,target,layout,'codex.exe');
 if(fs.existsSync(exe)){process.stdout.write(exe);process.exit(0);}
}
process.exit(3);
'@
            $previousErrorAction = $ErrorActionPreference
            try {
                $ErrorActionPreference = 'Continue'
                $resolved = (& $node.Source -e $resolver $packageRoot 2>$null | Out-String).Trim()
                $resolverExit = $LASTEXITCODE
            } finally { $ErrorActionPreference = $previousErrorAction }
            if ($resolverExit -eq 0 -and [IO.File]::Exists($resolved)) { return [IO.Path]::GetFullPath($resolved) }
            throw 'Codex native executable could not be located. Reinstall @openai/codex with npm, then rerun setup.'
        }
    }
    $native = Get-Command codex.exe -CommandType Application -ErrorAction SilentlyContinue
    if ($native) { return $native.Source }
    return $null
}

function Get-PrependedPath([string]$Directory, [string]$Original) {
    $remaining = @($Original -split ';' | Where-Object {
        $_ -and -not [string]::Equals($_.TrimEnd('\'), $Directory.TrimEnd('\'), [StringComparison]::OrdinalIgnoreCase)
    })
    return (@($Directory) + $remaining) -join ';'
}

function Enable-BareCodex([string]$Executable, [bool]$Persist) {
    if (-not $Executable) { return }
    $directory = [IO.Path]::GetDirectoryName($Executable)
    $env:Path = Get-PrependedPath $directory $env:Path
    $resolved = Get-Command codex -ErrorAction SilentlyContinue
    if (-not $resolved -or $resolved.CommandType -ne 'Application' -or
        -not [string]::Equals($resolved.Source, $Executable, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Codex bare command is shadowed by another command. Remove the conflicting codex alias/function or PATH entry and rerun setup.'
    }
    if ($Persist) {
        $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
        $proposedPath = Get-PrependedPath $directory $userPath
        $currentPath = $env:Path
        try {
            $machinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
            $env:Path = [Environment]::ExpandEnvironmentVariables($machinePath + ';' + $proposedPath)
            $freshCommand = Get-Command codex -ErrorAction SilentlyContinue
            if (-not $freshCommand -or $freshCommand.CommandType -ne 'Application' -or
                -not [string]::Equals($freshCommand.Source, $Executable, [StringComparison]::OrdinalIgnoreCase)) {
                throw 'Codex bare command would be shadowed in new terminals by a machine PATH entry. Remove the conflicting system Codex shim or PATH entry before retrying.'
            }
        } finally { $env:Path = $currentPath }
        [Environment]::SetEnvironmentVariable('Path', $proposedPath, 'User')
    }
}

# PowerShell 5.1 compatible. Only staging files are written; callers own commit/backup.
function Test-CodexManagedDefault([string]$Text) {
    # Recognize only the legacy installer's complete, exact generated grammar.
    # This regex is a fast-path recognizer, never a TOML parser or merger.
    $shape = '\A# BEGIN oneline-ccc LLMP managed configuration\r?\n' +
        "model_provider = 'oneline_llmp'\r?\nmodel = '[^'\r\n]*'\r?\n" +
        'model_context_window = [0-9]+\r?\nmodel_auto_compact_token_limit = [0-9]+\r?\n' +
        "(?:service_tier = '[^'\r\n]*'\r?\n)?(?:model_catalog_json = '[^'\r\n]*'\r?\n)?\r?\n" +
        "\[model_providers\.oneline_llmp\]\r?\nname = 'LLMP \(oneline-ccc\)'\r?\n" +
        "base_url = '[^'\r\n]*'\r?\nwire_api = 'responses'\r?\nenv_key = 'LLMP_API_KEY'\r?\n" +
        'requires_openai_auth = false\r?\n\r?\n# END oneline-ccc LLMP managed configuration\r?\n?\z'
    return $Text -cmatch $shape
}

function Start-CodexConfigProcess([string]$Executable, [string]$Stage, [string]$WorkingDirectory) {
    $process = $null
    try {
        $info = New-Object System.Diagnostics.ProcessStartInfo
        $info.FileName = [IO.Path]::GetFullPath($Executable)
        $info.Arguments = 'app-server --listen stdio://'
        if ([IO.Path]::GetExtension($Executable) -ieq '.cmd') {
            $directory = [IO.Path]::GetDirectoryName($info.FileName)
            $entry = Join-Path $directory 'node_modules/@openai/codex/bin/codex.js'
            $node = Join-Path $directory 'node.exe'
            if (-not [IO.File]::Exists($node)) {
                $command = Get-Command node.exe -CommandType Application -ErrorAction SilentlyContinue
                if ($command) { $node = $command.Source }
            }
            if ([IO.File]::Exists($entry) -and [IO.File]::Exists($node)) {
                $info.FileName = $node
                $info.Arguments = '"' + $entry + '" app-server --listen stdio://'
            } else {
                # cmd expands metacharacters even inside some quoted contexts.
                if ($info.FileName -match '[%!&|<>^"\r\n]') { throw 'Unsafe shim path.' }
                $shim = $info.FileName
                $info.FileName = Join-Path ([Environment]::GetFolderPath('System')) 'cmd.exe'
                $info.Arguments = '/d /v:off /s /c ""' + $shim + '" app-server --listen stdio://"'
            }
        }
        $info.UseShellExecute = $false
        $info.CreateNoWindow = $true
        $info.RedirectStandardInput = $true
        $info.RedirectStandardOutput = $true
        $info.RedirectStandardError = $true
        $utf8 = New-Object System.Text.UTF8Encoding($false)
        $info.StandardOutputEncoding = $utf8
        $info.StandardErrorEncoding = $utf8
        $info.WorkingDirectory = $Stage
        # Keep child CWD outside the GUID home that must be deleted after RPCs.
        if ($WorkingDirectory) { $info.WorkingDirectory = $WorkingDirectory }
        $info.EnvironmentVariables['CODEX_HOME'] = $Stage
        # Do not expose inherited credentials to the config-only subprocess.
        foreach ($key in @($info.EnvironmentVariables.Keys)) {
            if ([string]$key -match '(?i)(API_?KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|AUTHORIZATION|^CODEX_PROFILE$)') {
                $info.EnvironmentVariables.Remove([string]$key)
            }
        }
        $info.EnvironmentVariables['LLMP_API_KEY'] = 'dummy-fixture-key'
        $info.EnvironmentVariables['OPENAI_API_KEY'] = 'dummy-fixture-key'
        $process = New-Object System.Diagnostics.Process
        $process.StartInfo = $info
        $previousInputEncoding = [Console]::InputEncoding
        try {
            [Console]::InputEncoding = $utf8
            if (-not $process.Start()) { throw 'Start failed.' }
        } finally { [Console]::InputEncoding = $previousInputEncoding }
        $inputWriter = New-Object System.IO.StreamWriter($process.StandardInput.BaseStream, $utf8)
        $inputWriter.AutoFlush = $true
        # Drain concurrently; never forward raw stderr or protocol to the console.
        $drain = $process.StandardError.ReadToEndAsync()
        $transportKind = 'other'
        switch -Regex ([IO.Path]::GetFileName($info.FileName)) {
            '^codex(?:\.exe)?$' { $transportKind = 'native' }
            '^node(?:\.exe)?$' { $transportKind = 'node' }
            '^cmd(?:\.exe)?$' { $transportKind = 'cmd' }
        }
        return @{ Process = $process; Input = $inputWriter; Stderr = $drain; NextId = 1; TransportKind = $transportKind }
    } catch {
        if ($process) { Stop-CodexConfigProcess @{Process = $process} }
        throw 'Codex configuration process could not start.'
    }
}

# Diagnostic only: targeted parent-PID queries, no command lines or process killing.
function Get-CodexConfigDescendants([int]$RootProcessId, [hashtable]$Diagnostics) {
    $items = @()
    if ($env:OS -ne 'Windows_NT') { return $items }
    $Diagnostics.Snapshot = 'complete'
    $pending = New-Object 'System.Collections.Generic.Queue[int]'
    $pending.Enqueue($RootProcessId)
    $clock = [Diagnostics.Stopwatch]::StartNew()
    while ($pending.Count -gt 0 -and $items.Count -lt 32 -and $clock.ElapsedMilliseconds -lt 2000) {
        $ownerPid = $pending.Dequeue()
        try {
            $children = @(Get-CimInstance Win32_Process -Filter ('ParentProcessId=' + $ownerPid) -Property ProcessId,Name -OperationTimeoutSec 1 -ErrorAction Stop)
            foreach ($child in $children) {
                if ($items.Count -ge 32) { $Diagnostics.Snapshot = 'partial'; break }
                $observed = $null
                try {
                    $observed = [Diagnostics.Process]::GetProcessById([int]$child.ProcessId)
                    $kind = 'other'
                    switch -Regex ($child.Name) {
                        '^codex(?:\.exe)?$' { $kind = 'codex' }
                        '^(powershell|pwsh)(?:\.exe)?$' { $kind = 'powershell' }
                        '^node(?:\.exe)?$' { $kind = 'node' }
                        '^cmd(?:\.exe)?$' { $kind = 'cmd' }
                    }
                    $items += @{Id = $observed.Id; Started = $observed.StartTime.ToUniversalTime(); Kind = $kind}
                    $pending.Enqueue($observed.Id)
                } catch { } finally { if ($observed) { $observed.Dispose() } }
            }
        } catch { $Diagnostics.Snapshot = 'partial'; break }
    }
    if ($pending.Count) { $Diagnostics.Snapshot = 'partial' }
    return $items
}

function Stop-CodexConfigProcess($Connection) {
    if (-not $Connection) { return }
    $process = $Connection.Process
    $reaped = $false
    try {
        # A failed Start has no owned PID to stop.
        try { $null = $process.Id } catch { return }
        if ($Connection.Diagnostics) {
            try { $Connection.Diagnostics.Children += @(Get-CodexConfigDescendants $process.Id $Connection.Diagnostics) }
            catch { $Connection.Diagnostics.Snapshot = 'partial' }
        }
        # A close/flush failure must never skip wait, kill, and reap.
        try {
            if ($Connection.Input) { $Connection.Input.Close() }
            else { $process.StandardInput.Close() }
        } catch { }
        finally {
            try { $process.StandardInput.BaseStream.Close() } catch { }
        }
        try { $reaped = $process.WaitForExit(2000) } catch { }
        if ($Connection.Diagnostics) { $Connection.Diagnostics.Stop = 'graceful' }
        if (-not $reaped) {
            if ($Connection.Diagnostics) { $Connection.Diagnostics.Stop = 'killed' }
            try { $process.Kill() } catch { }
            try { $reaped = $process.WaitForExit(2000) } catch { }
        }
        if (-not $reaped) { throw 'Codex configuration process could not be stopped.' }
        if ($Connection.Diagnostics) { $Connection.Diagnostics.Stops += $Connection.Diagnostics.Stop }
    } finally {
        # Framework Process.Dispose does not close redirected streams.
        try { $process.StandardOutput.Close() } catch { }
        try { $process.StandardError.Close() } catch { }
        $process.Dispose()
    }
}

function Invoke-CodexConfigRpc($Connection, [string]$Method, [hashtable]$Parameters) {
    try {
        $requestId = $Connection.NextId
        $Connection.NextId++
        $message = @{ id = $requestId; method = $Method; params = $Parameters }
        $Connection.Input.WriteLine(($message | ConvertTo-Json -Depth 40 -Compress))
        $clock = [Diagnostics.Stopwatch]::StartNew()
        $bytes = 0
        while ($clock.ElapsedMilliseconds -lt 20000) {
            $read = $Connection.Process.StandardOutput.ReadLineAsync()
            $remaining = [Math]::Max(1, 20000 - [int]$clock.ElapsedMilliseconds)
            if (-not $read.Wait($remaining)) { throw 'Timeout.' }
            $line = $read.GetAwaiter().GetResult()
            if ($null -eq $line) { throw 'Closed.' }
            $bytes += $line.Length
            if ($bytes -gt 4194304) { throw 'Response limit.' }
            $reply = $line | ConvertFrom-Json
            if ($null -ne $reply.id -and $reply.id -eq $requestId) {
                if ($reply.PSObject.Properties['error']) { throw 'RPC error.' }
                if (-not $reply.PSObject.Properties['result']) { throw 'Invalid response.' }
                return $reply.result
            }
            if ($null -ne $reply.id -and $reply.method) { throw 'Unexpected server request.' }
        }
        throw 'Timeout.'
    } catch { throw 'Codex configuration request failed or timed out.' }
}

function Initialize-CodexConfigConnection($Connection) {
    $null = Invoke-CodexConfigRpc $Connection 'initialize' @{
        clientInfo = @{name = 'oneline_ccc_setup'; version = '1.0'}
    }
    $Connection.Input.WriteLine('{"method":"initialized","params":{}}')
}

# Delete links themselves, never enumerate them or change target attributes.
function Remove-CodexConfigStageTree([string]$Path, [string]$StageRoot) {
    if (-not $StageRoot) { $StageRoot = $Path }
    $isDirectory = $false
    try {
        $attributes = [IO.File]::GetAttributes($Path)
        $isDirectory = ($attributes -band [IO.FileAttributes]::Directory) -ne 0
        if (($attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
            if ($isDirectory) { [IO.Directory]::Delete($Path, $false) }
            else { [IO.File]::Delete($Path) }
            return
        }
        if (($attributes -band [IO.FileAttributes]::ReadOnly) -ne 0) {
            [IO.File]::SetAttributes($Path, ($attributes -band (-bnot [IO.FileAttributes]::ReadOnly)))
        }
        if ($isDirectory) {
            foreach ($entry in [IO.Directory]::GetFileSystemEntries($Path)) {
                Remove-CodexConfigStageTree $entry $StageRoot
            }
            [IO.Directory]::Delete($Path, $false)
        } else { [IO.File]::Delete($Path) }
    } catch {
        $cause = $_.Exception.GetBaseException()
        if (-not $cause.Data.Contains('CodexCleanupEntry')) {
            $category = 'other-file'
            $name = [IO.Path]::GetFileName($Path)
            if ($isDirectory) {
                if ([string]::Equals($Path, $StageRoot, [StringComparison]::OrdinalIgnoreCase)) { $category = 'stage-root' }
                elseif ($Path -match '[/\\]arg0(?:-windows)?([/\\]|$)') { $category = 'directory-arg0' }
                elseif ($Path -match '[/\\]skills([/\\]|$)') { $category = 'directory-skills' }
                else { $category = 'directory-other' }
            }
            elseif ($name -ceq 'config.toml') { $category = 'config' }
            elseif ($Path -match '[/\\]arg0(?:-windows)?[/\\]') {
                if ($name -ceq '.lock') { $category = 'arg0-lock' }
                elseif ($name -match '\.exe$') { $category = 'arg0-executable' }
                else { $category = 'arg0-alias' }
            } elseif ($Path -match '[/\\]skills[/\\]') { $category = 'skills' }
            elseif ($name -match '\.(db|sqlite)(-wal|-shm)?$') { $category = 'database' }
            elseif ($name -match '\.(log|jsonl)$') { $category = 'log' }
            $cause.Data['CodexCleanupEntry'] = $category
        }
        throw
    }
}

function Remove-CodexConfigStage([string]$Stage, [string]$ScratchRoot, [hashtable]$Diagnostics) {
    try {
        $full = [IO.Path]::GetFullPath($Stage)
        $root = [IO.Path]::GetFullPath($ScratchRoot)
        $parent = [IO.Path]::GetDirectoryName($full)
        $comparison = [StringComparison]::Ordinal
        if ($env:OS -eq 'Windows_NT') { $comparison = [StringComparison]::OrdinalIgnoreCase }
        if (-not [string]::Equals($parent, $root.TrimEnd([IO.Path]::DirectorySeparatorChar), $comparison) -or
            [IO.Path]::GetFileName($full) -cnotmatch '^codex-default-[0-9a-f]{32}$') {
            throw 'Invalid staging directory.'
        }
        for ($attempt = 0; $attempt -lt 5; $attempt++) {
            try {
                try { $attributes = [IO.File]::GetAttributes($full) }
                catch [IO.FileNotFoundException] { return }
                catch [IO.DirectoryNotFoundException] { return }
                # The stage root itself must remain the real directory we created.
                if (($attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
                    throw 'Staging directory was replaced.'
                }
                Remove-CodexConfigStageTree $full $full
                return
            } catch {
                if ($attempt -eq 4) { throw }
                Start-Sleep -Milliseconds (100 * [Math]::Pow(2, $attempt))
            }
        }
    } catch {
        # Fixed diagnostic classes only: no native exception text or paths.
        $cause = $_.Exception.GetBaseException()
        $category = 'other'
        if ($cause -is [UnauthorizedAccessException]) { $category = 'access-denied' }
        elseif ($cause -is [IO.IOException]) {
            $category = 'io'
            switch ($cause.HResult -band 65535) {
                32 { $category = 'sharing-violation' }
                33 { $category = 'lock-violation' }
                145 { $category = 'directory-not-empty' }
            }
        }
        $entry = $cause.Data['CodexCleanupEntry']
        if ($entry -cmatch '^(arg0-lock|arg0-alias|arg0-executable|skills|config|log|database|stage-root|directory-arg0|directory-skills|directory-other|other-file)$') {
            $category += '; entry=' + $entry
        }
        $cwd = 'unknown'
        try {
            $current = [IO.Path]::GetFullPath([Environment]::CurrentDirectory)
            $comparison = [StringComparison]::OrdinalIgnoreCase
            $cwd = 'outside'
            if ([string]::Equals($current, $full, $comparison)) { $cwd = 'stage' }
            elseif ($current.StartsWith($full + [IO.Path]::DirectorySeparatorChar, $comparison)) { $cwd = 'inside' }
        } catch { }
        $category += '; parent-cwd=' + $cwd
        if ($Diagnostics) {
            $live = 0
            $kinds = @()
            foreach ($child in $Diagnostics.Children) {
                $observed = $null
                try {
                    $observed = [Diagnostics.Process]::GetProcessById($child.Id)
                    if (-not $observed.HasExited -and $observed.StartTime.ToUniversalTime() -eq $child.Started) {
                        $live++
                        if ($child.Kind -cin @('codex','powershell','cmd','node','other')) { $kinds += $child.Kind }
                    }
                } catch { } finally { if ($observed) { $observed.Dispose() } }
            }
            $category += '; live-descendants=' + $live
            if ($Diagnostics.Snapshot -cin @('complete','partial','unknown')) { $category += '; child-snapshot=' + $Diagnostics.Snapshot }
            if ($kinds.Count) { $category += '; child-kinds=' + (($kinds | Sort-Object -Unique) -join ',') }
            if ($Diagnostics.Stop -cin @('graceful','killed')) { $category += '; stop=' + $Diagnostics.Stop }
            $transports = @($Diagnostics.Transports | Where-Object { $_ -cin @('native','node','cmd','other') })
            if ($transports.Count) { $category += '; transport=' + ($transports -join ',') }
            $stops = @($Diagnostics.Stops | Where-Object { $_ -cin @('graceful','killed') })
            if ($stops.Count) { $category += '; stops=' + ($stops -join ',') }
        }
        throw ('Default Codex configuration staging cleanup failed (' + $category + '). Existing configuration was not changed.')
    }
}

function Protect-CodexConfigStage([string]$Stage) {
    if ($env:OS -eq 'Windows_NT') {
        $owner = [Security.Principal.WindowsIdentity]::GetCurrent().User
        $acl = New-Object Security.AccessControl.DirectorySecurity
        $acl.SetAccessRuleProtection($true, $false)
        $acl.SetOwner($owner)
        foreach ($sid in @($owner.Value, 'S-1-5-18', 'S-1-5-32-544') | Select-Object -Unique) {
            $identity = New-Object Security.Principal.SecurityIdentifier($sid)
            $rule = New-Object Security.AccessControl.FileSystemAccessRule(
                $identity, 'FullControl', 'ContainerInherit,ObjectInherit', 'None', 'Allow')
            $acl.AddAccessRule($rule)
        }
        Set-Acl -LiteralPath $Stage -AclObject $acl -ErrorAction Stop
    } else {
        $chmod = Get-Command chmod -CommandType Application -ErrorAction Stop
        & $chmod.Source 700 -- $Stage 2>$null
        if ($LASTEXITCODE -ne 0) { throw 'Staging permissions could not be restricted.' }
    }
}

function Get-CodexDefaultConfig(
    [string]$CodexExecutable, [string]$Existing, [string]$Generated,
    [hashtable]$Settings, [string]$ScratchRoot
) {
    $stage = $null
    $connection = $null
    $diagnostics = @{Children = @(); Stop = 'unknown'; Snapshot = 'unknown'; Transports = @(); Stops = @()}
    try {
        if ([string]::IsNullOrWhiteSpace($Existing) -or (Test-CodexManagedDefault $Existing)) {
            return $Generated
        }
        if (-not $Settings -or -not $Settings.ContainsKey('model_providers')) { throw 'Settings missing.' }
        $provider = $Settings.model_providers.oneline_llmp
        if ($provider -isnot [System.Collections.IDictionary]) { throw 'Provider missing.' }
        $edits = @()
        foreach ($key in @('model', 'model_provider', 'model_context_window', 'model_auto_compact_token_limit')) {
            if (-not $Settings.ContainsKey($key) -or $null -eq $Settings[$key]) { throw 'Settings missing.' }
            $edits += @{ keyPath = $key; value = $Settings[$key]; mergeStrategy = 'replace' }
        }
        foreach ($key in @('service_tier', 'model_catalog_json')) {
            $value = $null
            if ($Settings.ContainsKey($key)) { $value = $Settings[$key] }
            $edits += @{keyPath = $key; value = $value; mergeStrategy = 'replace'}
        }
        foreach ($key in @('name', 'base_url', 'wire_api', 'env_key', 'requires_openai_auth')) {
            if (-not $provider.Contains($key) -or $null -eq $provider[$key]) { throw 'Provider missing.' }
            $edits += @{keyPath = ('model_providers.oneline_llmp.' + $key); value = $provider[$key]; mergeStrategy = 'replace'}
        }
        # A GUID child is the only directory this helper may remove.
        $root = [IO.Path]::GetFullPath($ScratchRoot)
        $stage = Join-Path $root ('codex-default-' + [Guid]::NewGuid().ToString('N'))
        $null = [IO.Directory]::CreateDirectory($stage)
        Protect-CodexConfigStage $stage
        $path = Join-Path $stage 'config.toml'
        [IO.File]::WriteAllText($path, $Existing, (New-Object Text.UTF8Encoding($false)))
        $connection = Start-CodexConfigProcess $CodexExecutable $stage $root
        $connection.Diagnostics = $diagnostics
        $diagnostics.Transports += $connection.TransportKind
        Initialize-CodexConfigConnection $connection
        $read = Invoke-CodexConfigRpc $connection 'config/read' @{includeLayers = $true}
        $layers = @($read.layers | Where-Object {
            $_.name.type -eq 'user' -and -not $_.name.profile -and
            [IO.Path]::GetFullPath($_.name.file) -eq $path
        })
        if ($layers.Count -ne 1 -or -not $layers[0].version) { throw 'User layer missing.' }
        $oldProvider = $layers[0].config.model_providers.oneline_llmp
        if ($oldProvider) {
            if ($oldProvider.base_url) { Get-ValidatedEndpoint ([string]$oldProvider.base_url) -ExistingProfile | Out-Null }
            foreach ($property in $oldProvider.PSObject.Properties) {
                if ($property.Name -match '(?i)(auth|bearer|token|api_?key|headers)' -and
                    $property.Name -ne 'requires_openai_auth' -and $null -ne $property.Value) {
                    throw 'Conflicting provider authentication.'
                }
            }
        }
        $write = Invoke-CodexConfigRpc $connection 'config/batchWrite' @{
            filePath = $path; expectedVersion = $layers[0].version
            edits = $edits; reloadUserConfig = $false
        }
        if ($write.status -cne 'ok' -or $write.overriddenMetadata -or
            [IO.Path]::GetFullPath($write.filePath) -ne $path) { throw 'Overridden write.' }
        Stop-CodexConfigProcess $connection
        $connection = $null
        # Fresh process validates the actual startup config, including catalog content.
        $connection = Start-CodexConfigProcess $CodexExecutable $stage $root
        $connection.Diagnostics = $diagnostics
        $diagnostics.Transports += $connection.TransportKind
        Initialize-CodexConfigConnection $connection
        $verified = Invoke-CodexConfigRpc $connection 'config/read' @{includeLayers = $true}
        foreach ($edit in $edits) {
            $actual = $verified.config
            foreach ($part in $edit.keyPath.Split('.')) {
                if ($null -eq $actual) { break }
                $actual = $actual.$part
            }
            if ($null -eq $edit.value) {
                if ($null -ne $actual) { throw 'Effective config mismatch.' }
            } elseif (($actual | ConvertTo-Json -Depth 20 -Compress) -cne
                      ($edit.value | ConvertTo-Json -Depth 20 -Compress)) {
                throw 'Effective config mismatch.'
            }
        }
        return [IO.File]::ReadAllText($path)
    } catch { throw 'Default Codex configuration could not be merged safely. Existing configuration was not changed.' }
    finally {
        try { Stop-CodexConfigProcess $connection }
        finally { if ($stage) { Remove-CodexConfigStage $stage $root $diagnostics } }
    }
}

function Publish-CodexConfigFile([hashtable]$File) {
    if ($File.Existed) {
        $File.Backup = $File.Path + '.backup-' + [Guid]::NewGuid().ToString('N')
        [IO.File]::Replace($File.Temp, $File.Path, $File.Backup)
    } else { [IO.File]::Move($File.Temp, $File.Path) }
}

function Invoke-SshServerSetup {
    if ($env:USERNAME -notmatch '^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,19}$') {
        throw 'SSH server setup requires an explicit supported local account. Run the SSH helper separately with -UserName; the elevated account will not be selected automatically.'
    }
    $cacheDirectory = Join-Path $env:LOCALAPPDATA 'oneline-ccc'
    [IO.Directory]::CreateDirectory($cacheDirectory) | Out-Null
    $sshScript = Join-Path $cacheDirectory ('setup-ssh-' + [Guid]::NewGuid().ToString('N') + '.ps1')
    try {
        Invoke-WebRequest -UseBasicParsing -MaximumRedirection 0 -Uri 'https://clankercode.github.io/oneline-ccc/setup-ssh-server.ps1' -OutFile $sshScript | Out-Null
        $windowsIdentity = [Security.Principal.WindowsIdentity]::GetCurrent()
        try {
            $principal = New-Object Security.Principal.WindowsPrincipal($windowsIdentity)
            $isAdministrator = $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
        }
        finally { $windowsIdentity.Dispose() }
        $arguments = '-NoProfile -ExecutionPolicy Bypass -File "' + $sshScript + '"'
        # The helper receives the original account even if UAC uses another admin.
        $arguments += ' -UserName "' + $env:USERNAME + '"'
        $processOptions = @{
            FilePath = (Join-Path $env:WINDIR 'System32\WindowsPowerShell\v1.0\powershell.exe')
            ArgumentList = $arguments; Wait = $true; PassThru = $true
        }
        if (-not $isAdministrator) { $processOptions.Verb = 'RunAs' }
        $process = Start-Process @processOptions
        try {
            if ($process.ExitCode -ne 0) { throw 'SSH server setup did not complete. Codex remains configured; run the SSH helper separately to retry.' }
        } finally { $process.Dispose() }
    } finally {
        if (Test-Path -LiteralPath $sshScript) { [IO.File]::Delete($sshScript) }
    }
}

try {
    if ($EnableSshServer -and $NonInteractive -and -not $DryRun) {
        throw 'SSH server setup requires interactive mode for user, key and firewall choices. Run the SSH helper directly for unattended setup.'
    }
    if ($EnableSshServer -and $env:OS -ne 'Windows_NT' -and -not $DryRun) { throw 'SSH server setup requires Windows.' }
    $BaseUrl = Read-RequiredText $BaseUrl 'LLMP OpenAI-compatible base URL (include /v1 if required)'
    $endpoint = Get-ValidatedEndpoint $BaseUrl
    $BaseUrl = $endpoint.AbsoluteUri.TrimEnd('/')
    if (-not $ApiKey -and -not [string]::IsNullOrWhiteSpace($env:LLMP_API_KEY)) {
        $ApiKey = ConvertTo-SecureString $env:LLMP_API_KEY -AsPlainText -Force
    }
    if (-not $DryRun -and -not $ApiKey) {
        if ($NonInteractive) { throw 'API key missing. Set LLMP_API_KEY in the environment or supply a SecureString ApiKey.' }
        $ApiKey = Read-Host 'LLMP API key (stored in your Windows user environment)' -AsSecureString
    }
    $wantDiscovery = $DiscoverModels
    if (-not $DryRun -and -not $NonInteractive -and -not $Model -and -not $wantDiscovery) {
        $wantDiscovery = (Read-Host 'List available model IDs from this endpoint? [y/N]') -match '^(?i)y(es)?$'
    }
    if ($wantDiscovery -and -not $DryRun) {
        if (-not $ApiKey -or $ApiKey.Length -eq 0) { throw 'API key missing for model discovery.' }
        $ids = @(Get-ModelIds $BaseUrl $ApiKey)
        if ($ids.Count -eq 0) { throw 'Model discovery failed. No model IDs were returned; supply Model explicitly.' }
        Write-Host 'Available model IDs:'
        $ids | ForEach-Object { Write-Host ('  ' + $_) }
    }
    if (-not $NonInteractive -and [string]::IsNullOrWhiteSpace($Model)) {
        Write-Host 'Choose model: 1) gpt-6.1-sol  2) glm-5.3  3) glm-5.3-flash  4) Exact custom ID'
        $choice = Read-Host 'Model choice [1-4]'
        switch ($choice) {
            '1' { $Model = 'gpt-6.1-sol' }
            '2' { $Model = 'glm-5.3' }
            '3' { $Model = 'glm-5.3-flash' }
            '4' { }
            default { throw 'Required input: choose a model number from 1 through 4.' }
        }
    }
    $Model = Read-RequiredText $Model 'Exact model ID supported by your LLMP endpoint'
    if ($ModelCatalogPath) {
        try {
            $ModelCatalogPath = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($ModelCatalogPath)
            $SupportedServiceTiers = @(Get-CatalogTiers $ModelCatalogPath $Model)
        } catch { throw 'Model catalog invalid: verify the path, models array, exact selected slug and tier field shapes.' }
        if ($SupportedServiceTiers.Count -gt 0) { Write-Host ('Catalog eligible tiers: ' + ($SupportedServiceTiers -join ', ')) }
    }
    if (-not $NonInteractive -and -not $ServiceTier) {
        $ServiceTier = (Read-Host 'Optional service tier (Enter to omit; use only an advertised tier)').Trim()
        if ($ServiceTier -and $SupportedServiceTiers.Count -eq 0 -and -not $ModelCatalogPath) {
            $advertised = Read-Host 'Advertised eligible tiers for this exact model, comma separated (operator attestation)'
            $SupportedServiceTiers = @($advertised.Split(',') | ForEach-Object { $_.Trim() } | Where-Object { $_ })
        }
    }
    if ($ContextWindow -eq 0) {
        $contextInput = Read-RequiredText '' 'Context budget in tokens (for example 272000, 353000 or 500000; confirm model capacity)'
        if (-not [long]::TryParse($contextInput, [ref]$ContextWindow)) { throw 'ContextWindow must be an integer.' }
    }
    if ($ContextWindow -lt 1024 -or $ContextWindow -gt 2147483647) { throw 'ContextWindow must be between 1024 and 2147483647 tokens.' }
    if ($AutoCompactTokenLimit -eq 0) { $AutoCompactTokenLimit = [long][Math]::Floor($ContextWindow * 0.9) }
    if ($AutoCompactTokenLimit -lt 1 -or $AutoCompactTokenLimit -ge $ContextWindow) {
        throw 'AutoCompactTokenLimit must be positive and below ContextWindow.'
    }
    if ($ServiceTier -and $SupportedServiceTiers -cnotcontains $ServiceTier) {
        throw 'ServiceTier requires explicit matching SupportedServiceTiers capability input for the selected model.'
    }
    # Explicitly reset a base config's tier instead of inheriting a tier intended
    # for another provider/model. 'default' is the standard request-tier reset.
    if (-not $ServiceTier) { $ServiceTier = 'default' }
    $providerUrl = ConvertTo-TomlString $BaseUrl
    $modelValue = ConvertTo-TomlString $Model
    $tierLine = ''
    if ($ServiceTier) { $tierLine = 'service_tier = ' + (ConvertTo-TomlString $ServiceTier) + "`n" }
    if ($ModelCatalogPath) { $tierLine += 'model_catalog_json = ' + (ConvertTo-TomlString $ModelCatalogPath) + "`n" }
    $block = @"
$beginMarker
model_provider = 'oneline_llmp'
model = $modelValue
model_context_window = $ContextWindow
model_auto_compact_token_limit = $AutoCompactTokenLimit
$tierLine
[model_providers.oneline_llmp]
name = 'LLMP (oneline-ccc)'
base_url = $providerUrl
wire_api = 'responses'
env_key = 'LLMP_API_KEY'
requires_openai_auth = false

"@
    $block += "`n" + $endMarker
    $configHome = $env:CODEX_HOME
    if ([string]::IsNullOrWhiteSpace($configHome)) { $configHome = Join-Path ([Environment]::GetFolderPath('UserProfile')) '.codex' }
    $configHome = [IO.Path]::GetFullPath($configHome)
    $configPath = Join-Path $configHome 'llmp.config.toml'
    $defaultPath = Join-Path $configHome 'config.toml'
    $profileSnapshot = Get-ConfigSnapshot $configPath
    $defaultSnapshot = Get-ConfigSnapshot $defaultPath
    $existing = $profileSnapshot.Text
    $defaultExisting = $defaultSnapshot.Text
    $updated = Get-UpdatedConfig $existing $block
    if ($DryRun) { Write-Output $block; return }
    $onWindows = $env:OS -eq 'Windows_NT'
    if (-not $onWindows -and -not $SkipInstall) { throw 'Native install requires Windows. Use -SkipInstall with isolated CODEX_HOME for testing.' }
    if ($ApiKey.Length -eq 0) { throw 'API key must not be empty.' }
    if (-not $SkipInstall) { Install-NativeCodex }
    Assert-CodexVersion
    $nativeExecutable = Get-NativeCodexExecutable
    if (Test-CodexManagedDefault $defaultExisting) { Get-UpdatedConfig $defaultExisting $block | Out-Null }
    $settings = @{
        model = $Model; model_provider = 'oneline_llmp'; model_context_window = $ContextWindow
        model_auto_compact_token_limit = $AutoCompactTokenLimit; service_tier = $ServiceTier
        model_providers = @{oneline_llmp = @{
            name = 'LLMP (oneline-ccc)'; base_url = $BaseUrl; wire_api = 'responses'
            env_key = 'LLMP_API_KEY'; requires_openai_auth = $false
        }}
    }
    if ($ModelCatalogPath) { $settings.model_catalog_json = $ModelCatalogPath }
    $command = Get-CodexCommand
    $mergeExecutable = ''
    if ($command) { $mergeExecutable = $command.Source }
    if ($nativeExecutable) { $mergeExecutable = $nativeExecutable }
    $scratchRoot = Join-Path $configHome '.setup-staging'
    $defaultUpdated = Get-CodexDefaultConfig $mergeExecutable $defaultExisting ($block + "`n") $settings $scratchRoot
    $files = @(
        @{Path = $defaultPath; Existed = $defaultSnapshot.Existed; Bytes = $defaultSnapshot.Bytes; Original = $defaultExisting; Updated = $defaultUpdated; Temp = $null; Backup = $null; Committed = $false},
        @{Path = $configPath; Existed = $profileSnapshot.Existed; Bytes = $profileSnapshot.Bytes; Original = $existing; Updated = $updated; Temp = $null; Backup = $null; Committed = $false}
    )
    $keyPointer = [IntPtr]::Zero
    $previousProcessKey = [Environment]::GetEnvironmentVariable('LLMP_API_KEY', 'Process')
    $previousProcessPath = $env:Path
    $persistUserKey = $onWindows -and -not $SkipPersistCredential
    $previousUserKey = $null
    $previousUserPath = $null
    if ($persistUserKey) { $previousUserKey = [Environment]::GetEnvironmentVariable('LLMP_API_KEY', 'User') }
    if ($onWindows) { $previousUserPath = [Environment]::GetEnvironmentVariable('Path', 'User') }
    $credentialsAttempted = $false
    $pathAttempted = $false
    try {
        foreach ($file in $files) {
            if ($file.Updated -cne $file.Original) {
                [IO.Directory]::CreateDirectory($configHome) | Out-Null
                $file.Temp = Join-Path $configHome ('.llmp-' + [Guid]::NewGuid().ToString('N') + '.tmp')
                [IO.File]::WriteAllText($file.Temp, $file.Updated, (New-Object Text.UTF8Encoding($false)))
            }
        }
        # Both original byte snapshots must still match before changing auth or PATH.
        foreach ($file in $files) {
            $existsNow = [IO.File]::Exists($file.Path)
            if ($existsNow -ne $file.Existed) { throw 'Config changed during setup. Rerun setup to preserve concurrent changes.' }
            if ($existsNow -and [Convert]::ToBase64String([IO.File]::ReadAllBytes($file.Path)) -cne [Convert]::ToBase64String($file.Bytes)) {
                throw 'Config changed during setup. Rerun setup to preserve concurrent changes.'
            }
        }
        $keyPointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($ApiKey)
        $plainKey = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($keyPointer)
        if ($plainKey -match '[\x00-\x20\x7f]') { throw 'API key must not contain whitespace or control characters.' }
        $credentialsAttempted = $true
        if ($persistUserKey) { [Environment]::SetEnvironmentVariable('LLMP_API_KEY', $plainKey, 'User') }
        [Environment]::SetEnvironmentVariable('LLMP_API_KEY', $plainKey, 'Process')
        if ($nativeExecutable) {
            $pathAttempted = $true
            Enable-BareCodex $nativeExecutable (-not $SkipPersistCredential)
        } elseif ($onWindows -and -not $SkipInstall) {
            throw 'Codex native executable could not be located. Reinstall @openai/codex with npm, then rerun setup.'
        }
        foreach ($file in $files) {
            if (-not $file.Temp) { continue }
            $existsNow = [IO.File]::Exists($file.Path)
            if ($existsNow -ne $file.Existed -or ($existsNow -and
                [Convert]::ToBase64String([IO.File]::ReadAllBytes($file.Path)) -cne [Convert]::ToBase64String($file.Bytes))) {
                throw 'Config changed during setup. Rerun setup to preserve concurrent changes.'
            }
            Publish-CodexConfigFile $file
            $file.Committed = $true
        }
    } catch {
        $rollbackFailed = $false
        for ($index = $files.Count - 1; $index -ge 0; $index--) {
            $file = $files[$index]
            if (-not $file.Committed) { continue }
            try {
                # Never clobber a later third-party edit while rolling back.
                if ([IO.File]::ReadAllText($file.Path) -cne $file.Updated) { throw 'Concurrent rollback conflict.' }
                if ($file.Existed) { [IO.File]::Replace($file.Backup, $file.Path, [NullString]::Value) }
                else { [IO.File]::Delete($file.Path) }
            } catch { $rollbackFailed = $true }
        }
        if ($credentialsAttempted) {
            try {
                [Environment]::SetEnvironmentVariable('LLMP_API_KEY', $previousProcessKey, 'Process')
                if ($persistUserKey) { [Environment]::SetEnvironmentVariable('LLMP_API_KEY', $previousUserKey, 'User') }
            } catch { $rollbackFailed = $true }
        }
        if ($pathAttempted) {
            try {
                $env:Path = $previousProcessPath
                if (-not $SkipPersistCredential) { [Environment]::SetEnvironmentVariable('Path', $previousUserPath, 'User') }
            } catch { $rollbackFailed = $true }
        }
        if ($rollbackFailed) { throw 'Config rollback failed. Restore configuration from its backup and check LLMP_API_KEY and user PATH before retrying.' }
        throw
    } finally {
        $plainKey = $null
        $previousProcessKey = $null
        $previousUserKey = $null
        if ($keyPointer -ne [IntPtr]::Zero) { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($keyPointer) }
        foreach ($file in $files) {
            if ($file.Temp -and [IO.File]::Exists($file.Temp)) { [IO.File]::Delete($file.Temp) }
        }
    }
    Write-Output 'LLMP is configured as the Codex default. Start with: codex'
    Write-Output 'The named configuration is also available with: codex --profile llmp'
    Write-Output 'The selected context budget does not increase the actual model capacity.'
    $wantSsh = $EnableSshServer
    if ($onWindows -and -not $NonInteractive -and -not $wantSsh) {
        $wantSsh = (Read-Host 'Also set up Windows SSH server for local-subnet access? Administrator approval required [y/N]') -match '^(?i)y(es)?$'
    }
    if ($wantSsh) {
        try { Invoke-SshServerSetup }
        catch {
            Write-Warning 'Codex is configured, but optional SSH setup did not complete. Retry using the SSH instructions in the project README.'
            if ($_.Exception.Message.StartsWith('SSH server setup requires', [StringComparison]::Ordinal)) { Write-Warning $_.Exception.Message }
        }
    }
} catch {
    # Never emit exception payloads from dependencies, URLs, files or credentials.
    Write-Verbose ('Setup failure type: ' + $_.Exception.GetType().FullName + '; line: ' + $_.InvocationInfo.ScriptLineNumber)
    $safeMessages = @(
        'Required input', 'Configuration values', 'Existing config', 'Managed configuration',
        'Managed block', 'An existing LLMP', 'Existing inline', 'BaseUrl must',
        'ContextWindow must', 'AutoCompactTokenLimit must', 'ServiceTier requires',
        'Native install requires', 'API key', 'Install Node.js', 'Node.js installation',
        'npm is unavailable', 'Codex npm installation', 'Codex installed', 'Codex native executable', 'Codex bare command', 'Config rollback failed', 'Default Codex configuration', 'Config changed', 'Model discovery failed', 'Model catalog invalid', 'SSH server setup requires'
    )
    $message = 'Setup failed. Check configuration directory permissions and dependency availability; no diagnostic payload was printed.'
    foreach ($prefix in $safeMessages) {
        if ($_.Exception.Message.StartsWith($prefix, [StringComparison]::Ordinal)) { $message = $_.Exception.Message; break }
    }
    throw $message
}
