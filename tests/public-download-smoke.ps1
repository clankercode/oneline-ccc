#requires -Version 5.1
$ErrorActionPreference = 'Stop'
$repository = Split-Path $PSScriptRoot -Parent
$downloaded = Invoke-RestMethod 'https://clankercode.github.io/oneline-ccc/setup-codex.ps1'
$local = [IO.File]::ReadAllText((Join-Path $repository 'scripts/setup-codex.ps1'))
if ($downloaded.Replace("`r`n", "`n") -cne $local.Replace("`r`n", "`n")) { throw 'Hosted installer differs from checkout' }
$env:CODEX_HOME = Join-Path $repository ('.cache/public-download-' + [Guid]::NewGuid().ToString('N'))
& ([scriptblock]::Create($downloaded)) -BaseUrl 'https://fixture.invalid/v1' -Model 'fixture-model' -ContextWindow 500000 -DryRun -NonInteractive
if (Test-Path $env:CODEX_HOME) { throw 'Dry run modified configuration' }
