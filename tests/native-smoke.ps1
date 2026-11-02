#requires -Version 5.1
$ErrorActionPreference = 'Stop'
$repository = Split-Path $PSScriptRoot -Parent
$env:CODEX_HOME = Join-Path $repository '.cache/native-codex-smoke'
& (Join-Path $repository 'scripts/setup-codex.ps1') -BaseUrl 'https://fixture.invalid/v1' -Model 'fixture-model' -ContextWindow 500000 -NonInteractive -SkipPersistCredential
if ($LASTEXITCODE -ne 0) { throw 'Native Codex installation failed' }
$npmPrefix = Join-Path $env:APPDATA 'npm'
& (Join-Path $npmPrefix 'codex.cmd') --profile llmp debug prompt-input 'Offline configuration smoke test' | Out-File -Encoding utf8 (Join-Path $repository '.cache/native-prompt.json')
if ($LASTEXITCODE -ne 0) { throw 'Native Codex profile loading failed' }
