#requires -Version 5.1
$ErrorActionPreference = 'Stop'
$repository = Split-Path $PSScriptRoot -Parent
$downloaded = Invoke-RestMethod 'https://clankercode.github.io/oneline-ccc/setup-codex.ps1'
$local = [IO.File]::ReadAllText((Join-Path $repository 'scripts/setup-codex.ps1'))
if ($downloaded.Replace("`r`n", "`n") -cne $local.Replace("`r`n", "`n")) { throw 'Hosted installer differs from checkout' }
$env:CODEX_HOME = Join-Path $repository ('.cache/public-download-' + [Guid]::NewGuid().ToString('N'))
& ([scriptblock]::Create($downloaded)) -BaseUrl 'https://fixture.invalid/v1' -Model 'fixture-model' -ContextWindow 500000 -DryRun -NonInteractive
if (Test-Path $env:CODEX_HOME) { throw 'Dry run modified configuration' }
$sshDownloaded = Invoke-RestMethod 'https://clankercode.github.io/oneline-ccc/setup-ssh-server.ps1'
$sshLocal = [IO.File]::ReadAllText((Join-Path $repository 'scripts/setup-ssh-server.ps1'))
if ($sshDownloaded.Replace("`r`n", "`n") -cne $sshLocal.Replace("`r`n", "`n")) { throw 'Hosted SSH installer differs from checkout' }
$blob = [byte[]](@(0, 0, 0, 11) + [Text.Encoding]::ASCII.GetBytes('ssh-ed25519') + @(0, 0, 0, 32) + (1..32))
$publicKey = 'ssh-ed25519 ' + [Convert]::ToBase64String($blob) + ' offline-fixture'
$plan = & ([scriptblock]::Create($sshDownloaded)) -UserName sshfixture -PublicKey $publicKey -DryRun -NonInteractive
if ($plan.RemoteAddress -ne 'LocalSubnet' -or @($plan.NetworkProfiles).Count -ne 1 -or $plan.NetworkProfiles[0] -ne 'Private') { throw 'Hosted SSH defaults are incorrect' }
