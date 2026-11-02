#requires -Version 5.1
$ErrorActionPreference = 'Stop'
$repository = Split-Path $PSScriptRoot -Parent
$previousHome = $env:CODEX_HOME
$previousKey = $env:LLMP_API_KEY
$previousProfile = $env:CODEX_PROFILE
$previousPath = $env:Path
try {
    # Deliberately absent: version verification must tolerate first-launch warnings.
    $env:CODEX_HOME = Join-Path $repository ('.cache/native-codex-smoke-' + [Guid]::NewGuid().ToString('N'))
    $env:LLMP_API_KEY = 'fixture-only-not-a-real-credential'
    $env:CODEX_PROFILE = $null
    if ($env:OS -eq 'Windows_NT') {
        # Previous CI child processes may have installed npm without updating this PATH.
        $npmPrefix = Join-Path $env:APPDATA 'npm'
        $env:Path = $npmPrefix + ';' + $env:Path
    }
    $options = @{
        BaseUrl = 'https://fixture.invalid/v1'; Model = 'fixture-model'; ContextWindow = 500000
        NonInteractive = $true; SkipPersistCredential = $true; Verbose = $true
    }
    if ($env:OS -ne 'Windows_NT') { $options.SkipInstall = $true }
    & (Join-Path $repository 'scripts/setup-codex.ps1') @options
    foreach ($name in @('config.toml', 'llmp.config.toml')) {
        $path = Join-Path $env:CODEX_HOME $name
        if (-not [IO.File]::Exists($path)) { throw 'Native setup did not create both configurations' }
        $text = [IO.File]::ReadAllText($path)
        if ($text.Contains($env:LLMP_API_KEY)) { throw 'Native setup exposed fixture credential in configuration' }
    }
    $command = Get-Command codex -ErrorAction Stop
    if ($command.CommandType -ne 'Application') { throw 'Bare codex does not resolve to an application' }
    if ($env:OS -eq 'Windows_NT' -and [IO.Path]::GetExtension($command.Source) -ine '.exe') {
        throw 'Bare codex must resolve to the native executable without PowerShell execution-policy changes'
    }
    # Exercise the default configuration with no --profile or config overrides.
    $prompt = (& codex debug prompt-input 'Offline configuration smoke test' | Out-String)
    $debugExit = $LASTEXITCODE
    if ($debugExit -ne 0) { throw 'Bare native Codex configuration loading failed' }
    if ([string]::IsNullOrWhiteSpace($prompt)) { throw 'Bare native Codex did not produce an offline prompt' }
    if ($prompt.Contains($env:LLMP_API_KEY)) { throw 'Bare native Codex exposed fixture credential' }
    [IO.File]::WriteAllText((Join-Path $env:CODEX_HOME 'native-prompt.json'), $prompt, (New-Object Text.UTF8Encoding($false)))
    # Inspect effective settings through the native offline configuration API.
    $start = New-Object Diagnostics.ProcessStartInfo
    $start.FileName = $command.Source
    $start.Arguments = 'app-server --listen stdio://'
    $start.WorkingDirectory = $env:CODEX_HOME
    $start.UseShellExecute = $false
    $start.RedirectStandardInput = $true
    $start.RedirectStandardOutput = $true
    $start.RedirectStandardError = $true
    foreach ($name in @($start.EnvironmentVariables.Keys)) {
        if ([string]$name -match '(?i)(API_?KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|AUTHORIZATION)') {
            $start.EnvironmentVariables.Remove([string]$name)
        }
    }
    $start.EnvironmentVariables['LLMP_API_KEY'] = 'fixture-only-not-a-real-credential'
    $process = New-Object Diagnostics.Process
    $process.StartInfo = $start
    $started = $false
    try {
        $process.Start() | Out-Null
        $started = $true
        $stderr = $process.StandardError.ReadToEndAsync()
        foreach ($request in @(
            @{id = 1; method = 'initialize'; params = @{clientInfo = @{name = 'native_smoke'; version = '1.0'}}},
            @{id = 2; method = 'config/read'; params = @{includeLayers = $true}}
        )) {
            $process.StandardInput.WriteLine(($request | ConvertTo-Json -Depth 8 -Compress))
            $process.StandardInput.Flush()
            $deadline = [DateTime]::UtcNow.AddSeconds(20)
            do {
                $remaining = [int][Math]::Max(1, ($deadline - [DateTime]::UtcNow).TotalMilliseconds)
                $read = $process.StandardOutput.ReadLineAsync()
                if (-not $read.Wait($remaining)) { throw 'Native config smoke timed out' }
                $line = $read.GetAwaiter().GetResult()
                if ($null -eq $line -or [DateTime]::UtcNow -gt $deadline) { throw 'Native config smoke response missing' }
                $reply = $line | ConvertFrom-Json
            } while ($reply.id -ne $request.id)
            if ($reply.PSObject.Properties['error']) { throw 'Native config smoke request failed' }
            if ($request.id -eq 1) {
                $process.StandardInput.WriteLine('{"method":"initialized","params":{}}')
                $process.StandardInput.Flush()
            }
        }
        $config = $reply.result.config
        if ($config.model -cne 'fixture-model' -or $config.model_provider -cne 'oneline_llmp' -or
            $config.model_context_window -ne 500000 -or
            $config.model_providers.oneline_llmp.base_url -cne 'https://fixture.invalid/v1' -or
            $config.model_providers.oneline_llmp.env_key -cne 'LLMP_API_KEY') {
            throw 'Native effective configuration did not select LLMP defaults'
        }
    } finally {
        if ($started -and -not $process.HasExited) {
            $process.StandardInput.Close()
            if (-not $process.WaitForExit(2000)) { $process.Kill(); $process.WaitForExit() }
        }
        $process.Dispose()
    }
    Write-Output 'PASS bare codex loads the LLMP default without --profile'
} finally {
    $env:CODEX_HOME = $previousHome
    $env:LLMP_API_KEY = $previousKey
    $env:CODEX_PROFILE = $previousProfile
    $env:Path = $previousPath
}
