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
    # Reuse the production transport: PS5.1's default StandardInput writer can
    # emit a BOM; JSON-lines requires the explicit BOM-free UTF-8 writer.
    # Extract only definitions, without executing the installer's entry point.
    $tokens = $null
    $parseErrors = $null
    $ast = [Management.Automation.Language.Parser]::ParseFile(
        (Join-Path $repository 'scripts/setup-codex.ps1'), [ref]$tokens, [ref]$parseErrors)
    if ($parseErrors.Count -ne 0) { throw 'Native smoke could not parse installer helpers' }
    foreach ($helperName in @('Start-CodexConfigProcess', 'Stop-CodexConfigProcess', 'Invoke-CodexConfigRpc', 'Initialize-CodexConfigConnection')) {
        $definitions = @($ast.FindAll({ param($node)
            $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -ceq $helperName
        }, $false))
        if ($definitions.Count -ne 1) { throw 'Native smoke installer helper missing or ambiguous' }
        . ([ScriptBlock]::Create($definitions[0].Extent.Text))
    }
    $connection = $null
    try {
        $connection = Start-CodexConfigProcess $command.Source $env:CODEX_HOME
        Initialize-CodexConfigConnection $connection
        $effective = Invoke-CodexConfigRpc $connection 'config/read' @{includeLayers = $true}
        $config = $effective.config
        if ($config.model -cne 'fixture-model' -or $config.model_provider -cne 'oneline_llmp' -or
            $config.model_context_window -ne 500000 -or
            $config.model_providers.oneline_llmp.base_url -cne 'https://fixture.invalid/v1' -or
            $config.model_providers.oneline_llmp.env_key -cne 'LLMP_API_KEY') {
            throw 'Native effective configuration did not select LLMP defaults'
        }
    } finally {
        Stop-CodexConfigProcess $connection
    }
    Write-Output 'PASS bare codex loads the LLMP default without --profile'
} finally {
    $env:CODEX_HOME = $previousHome
    $env:LLMP_API_KEY = $previousKey
    $env:CODEX_PROFILE = $previousProfile
    $env:Path = $previousPath
}
