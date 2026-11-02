#requires -Version 5.1
<#
.SYNOPSIS
Configure native Codex for an LLMP endpoint without changing the default profile.
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
        $command = Get-Command codex.cmd -ErrorAction SilentlyContinue
        if (-not $command) { $command = Get-Command codex.exe -ErrorAction SilentlyContinue }
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
    $existing = ''
    if (Test-Path -LiteralPath $configPath) { $existing = [IO.File]::ReadAllText($configPath) }
    $updated = Get-UpdatedConfig $existing $block
    if ($DryRun) { Write-Output $block; return }
    $onWindows = $env:OS -eq 'Windows_NT'
    if (-not $onWindows -and -not $SkipInstall) { throw 'Native install requires Windows. Use -SkipInstall with isolated CODEX_HOME for testing.' }
    if ($ApiKey.Length -eq 0) { throw 'API key must not be empty.' }
    if (-not $SkipInstall) { Install-NativeCodex }
    Assert-CodexVersion
    $temporaryPath = $null
    $keyPointer = [IntPtr]::Zero
    $previousProcessKey = [Environment]::GetEnvironmentVariable('LLMP_API_KEY', 'Process')
    $persistUserKey = $onWindows -and -not $SkipPersistCredential
    $previousUserKey = $null
    if ($persistUserKey) { $previousUserKey = [Environment]::GetEnvironmentVariable('LLMP_API_KEY', 'User') }
    $credentialsAttempted = $false
    try {
        if ($updated -cne $existing) {
            [IO.Directory]::CreateDirectory($configHome) | Out-Null
            $temporaryPath = Join-Path $configHome ('.llmp-' + [Guid]::NewGuid().ToString('N') + '.tmp')
            [IO.File]::WriteAllText($temporaryPath, $updated, (New-Object Text.UTF8Encoding($false)))
        }
        # Validate filesystem access and recheck immediately before credential mutation.
        $current = ''
        if (Test-Path -LiteralPath $configPath) { $current = [IO.File]::ReadAllText($configPath) }
        if ($current -cne $existing) { throw 'Config changed during setup. Rerun setup to preserve concurrent changes.' }
        $keyPointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($ApiKey)
        $plainKey = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($keyPointer)
        if ($plainKey -match '[\x00-\x20\x7f]') { throw 'API key must not contain whitespace or control characters.' }
        $credentialsAttempted = $true
        if ($persistUserKey) { [Environment]::SetEnvironmentVariable('LLMP_API_KEY', $plainKey, 'User') }
        [Environment]::SetEnvironmentVariable('LLMP_API_KEY', $plainKey, 'Process')
        if ($temporaryPath) {
            $current = ''
            if (Test-Path -LiteralPath $configPath) { $current = [IO.File]::ReadAllText($configPath) }
            if ($current -cne $existing) { throw 'Config changed during setup. Rerun setup to preserve concurrent changes.' }
            if (Test-Path -LiteralPath $configPath) {
                $backupPath = $configPath + '.backup-' + [Guid]::NewGuid().ToString('N')
                [IO.File]::Replace($temporaryPath, $configPath, $backupPath)
            } else { [IO.File]::Move($temporaryPath, $configPath) }
        }
    } catch {
        if ($credentialsAttempted) {
            try {
                [Environment]::SetEnvironmentVariable('LLMP_API_KEY', $previousProcessKey, 'Process')
                if ($persistUserKey) { [Environment]::SetEnvironmentVariable('LLMP_API_KEY', $previousUserKey, 'User') }
            } catch { throw 'API key rollback failed. Restore LLMP_API_KEY in your user/process environment before retrying setup.' }
        }
        throw
    } finally {
        $plainKey = $null
        $previousProcessKey = $null
        $previousUserKey = $null
        if ($keyPointer -ne [IntPtr]::Zero) { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($keyPointer) }
        if ($temporaryPath -and (Test-Path -LiteralPath $temporaryPath)) { [IO.File]::Delete($temporaryPath) }
    }
    if ($onWindows) {
        $launchCommand = 'codex.cmd'
        $installedCommand = Get-CodexCommand
        if ($installedCommand) { $launchCommand = $installedCommand.Name }
        Write-Output ('LLMP profile configured. Start with: ' + $launchCommand + ' --profile llmp')
    }
    else { Write-Output 'LLMP profile configured. Start with: codex --profile llmp' }
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
        'npm is unavailable', 'Codex npm installation', 'Codex installed', 'Config changed', 'Model discovery failed', 'Model catalog invalid', 'SSH server setup requires'
    )
    $message = 'Setup failed. Check configuration directory permissions and dependency availability; no diagnostic payload was printed.'
    foreach ($prefix in $safeMessages) {
        if ($_.Exception.Message.StartsWith($prefix, [StringComparison]::Ordinal)) { $message = $_.Exception.Message; break }
    }
    throw $message
}
