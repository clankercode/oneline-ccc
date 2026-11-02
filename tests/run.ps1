#requires -Version 5.1
[CmdletBinding()]
param([string]$Python = 'python', [string]$TestFilter = '*')
$ErrorActionPreference = 'Stop'
$repository = Split-Path $PSScriptRoot -Parent
$installer = Join-Path $repository 'scripts/setup-codex.ps1'
$root = Join-Path $repository ('.cache/installer-tests-' + [Guid]::NewGuid().ToString('N'))
[IO.Directory]::CreateDirectory($root) | Out-Null
$shellExecutable = (Get-Process -Id $PID).Path
$nativeName = 'codex'
if ($env:OS -eq 'Windows_NT') {
    $nativeName = 'codex.cmd'
    $env:Path = (Join-Path $env:APPDATA 'npm') + ';' + $env:Path
}
$nativeCodex = (Get-Command $nativeName -ErrorAction Stop).Source
$dummyKey = 'fixture-only-not-a-real-credential'
$beginMarker = '# BEGIN oneline-ccc LLMP managed configuration'
$endMarker = '# END oneline-ccc LLMP managed configuration'
$script:passed = 0
$script:failures = @()

function Assert-True($Condition, [string]$Message) {
    if (-not $Condition) { throw $Message }
}
function Write-Utf8([string]$Path, [string]$Content) {
    [IO.File]::WriteAllText($Path, $Content, (New-Object Text.UTF8Encoding($false)))
}
function New-Case([string]$Name) {
    $directory = Join-Path $root $Name
    [IO.Directory]::CreateDirectory($directory) | Out-Null
    return $directory
}
function Get-Profile([string]$HomeDirectory) { return Join-Path $HomeDirectory 'llmp.config.toml' }
function Invoke-Installer([string]$HomeDirectory, [hashtable]$Overrides = @{}, [switch]$Failure, [switch]$Bootstrap, [switch]$NoCredential, [hashtable]$EnvironmentOverrides = @{}, [switch]$FaultAfterFirstCommit, [switch]$UseVersionFixture, [string]$OfficialCatalogFixture, [switch]$UseNativeIdentityFixture, [switch]$AssertUnchangedEnvironment, [object[]]$PromptAnswers, [string]$UserKeyFixture) {
    $arguments = @{
        BaseUrl = 'https://fixture.invalid/v1/'; Model = 'fixture-model'
        ContextWindow = 353000; AutoCompactTokenLimit = 300000
        NonInteractive = $true; SkipInstall = $true; SkipPersistCredential = $true
    }
    foreach ($name in $Overrides.Keys) {
        if ($null -eq $Overrides[$name]) { $arguments.Remove($name) }
        else { $arguments[$name] = $Overrides[$name] }
    }
    $identifier = [Guid]::NewGuid().ToString('N')
    $argumentsPath = Join-Path $root ($identifier + '.json')
    Write-Utf8 $argumentsPath ($arguments | ConvertTo-Json)
    $runner = Join-Path $root ($identifier + '.ps1')
    # Every child uses a protected synthetic User-key source, including absent-key
    # cases and calls using other seams. Never consult this workstation's saved key.
    $source = [IO.File]::ReadAllText($installer)
    $tokens = $null; $parseErrors = $null
    $ast = [Management.Automation.Language.Parser]::ParseInput($source, [ref]$tokens, [ref]$parseErrors)
    $getter = @($ast.FindAll({ param($node)
        $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -ceq 'Get-LLMPUserApiKey'
    }, $false))
    if ($parseErrors.Count -ne 0 -or $getter.Count -ne 1) { throw 'User credential fixture seam changed' }
    $body = $getter[0].Body.Extent
    $source = $source.Remove($body.StartOffset, $body.EndOffset - $body.StartOffset).Insert($body.StartOffset, '{ return $env:CODEX_USER_KEY_FIXTURE }')
    $fixtureInstaller = Join-Path $root ($identifier + '.installer.ps1')
    Write-Utf8 $fixtureInstaller $source
    $escapedInstaller = $fixtureInstaller.Replace("'", "''")
    $escapedArguments = $argumentsPath.Replace("'", "''")
    $invocation = "& '$escapedInstaller' @options"
    if ($Bootstrap) { $invocation = "& ([ScriptBlock]::Create([IO.File]::ReadAllText('$escapedInstaller'))) @options" }
    if ($OfficialCatalogFixture) {
        $invocation = @"
`$source = [IO.File]::ReadAllText('$escapedInstaller')
`$tokens = `$null; `$parseErrors = `$null
`$ast = [Management.Automation.Language.Parser]::ParseInput(`$source, [ref]`$tokens, [ref]`$parseErrors)
`$download = @(`$ast.FindAll({ param(`$node)
    `$node -is [Management.Automation.Language.FunctionDefinitionAst] -and `$node.Name -ceq 'Get-OfficialCodexCatalogText'
}, `$false))
if (`$parseErrors.Count -ne 0 -or `$download.Count -ne 1) { throw 'Catalog download fixture seam changed' }
`$body = `$download[0].Body.Extent
`$replacement = '{ return [IO.File]::ReadAllText(`$env:CODEX_OFFICIAL_CATALOG_FIXTURE) }'
`$source = `$source.Remove(`$body.StartOffset, `$body.EndOffset - `$body.StartOffset).Insert(`$body.StartOffset, `$replacement)
& ([ScriptBlock]::Create(`$source)) @options
"@
    }
    if ($UseVersionFixture) {
        # Resolve the executable explicitly in an isolated copy. Keep the real
        # version probe, native stderr handling, exit check and parser intact.
        $invocation = @"
`$source = [IO.File]::ReadAllText('$escapedInstaller')
`$tokens = `$null; `$parseErrors = `$null
`$ast = [Management.Automation.Language.Parser]::ParseInput(`$source, [ref]`$tokens, [ref]`$parseErrors)
if (`$parseErrors.Count -ne 0) { throw 'Version fixture could not parse installer' }
`$resolvers = @(`$ast.FindAll({ param(`$node)
    `$node -is [Management.Automation.Language.FunctionDefinitionAst] -and
    `$node.Name -cin @('Get-CodexCommand', 'Get-NativeCodexExecutable')
}, `$false))
if (`$resolvers.Count -ne 2 -or @(`$resolvers | Select-Object -ExpandProperty Name -Unique).Count -ne 2) { throw 'Version fixture resolver seam changed' }
foreach (`$resolver in @(`$resolvers | Sort-Object { `$_.Body.Extent.StartOffset } -Descending)) {
    `$replacement = '{ return `$null }'
    if (`$resolver.Name -ceq 'Get-CodexCommand') {
        `$replacement = '{ if (`$env:CODEX_VERSION_EXECUTABLE) { return [pscustomobject]@{ Source = `$env:CODEX_VERSION_EXECUTABLE; Name = [IO.Path]::GetFileName(`$env:CODEX_VERSION_EXECUTABLE) } }; return `$null }'
    }
    `$body = `$resolver.Body.Extent
    `$source = `$source.Remove(`$body.StartOffset, `$body.EndOffset - `$body.StartOffset).Insert(`$body.StartOffset, `$replacement)
}
& ([ScriptBlock]::Create(`$source)) @options
"@
    }
    if ($UseNativeIdentityFixture) {
        # Model mixed Windows discovery on either test host. Invoke the real
        # version probes; record catalog/staging/publication identity at their
        # boundaries without changing this host's PATH or downloading metadata.
        $invocation = @"
`$source = [IO.File]::ReadAllText('$escapedInstaller')
`$tokens = `$null; `$parseErrors = `$null
`$ast = [Management.Automation.Language.Parser]::ParseInput(`$source, [ref]`$tokens, [ref]`$parseErrors)
`$replacements = @{
    'Get-CodexCommand' = '{ return [pscustomobject]@{ Source = `$env:CODEX_OLD_EXECUTABLE } }'
    'Get-NativeCodexExecutable' = '{ return `$env:CODEX_SELECTED_EXECUTABLE }'
    'Get-OfficialCodexCatalogText' = '{ [IO.File]::WriteAllText(`$env:CODEX_IDENTITY_VERSION, `$Version); return [IO.File]::ReadAllText(`$env:CODEX_OFFICIAL_CATALOG_FIXTURE) }'
    'Get-CodexDefaultConfig' = '{ [IO.File]::WriteAllText(`$env:CODEX_IDENTITY_STAGE, `$CodexExecutable); return `$Generated }'
    'Enable-BareCodex' = '{ [IO.File]::WriteAllText(`$env:CODEX_IDENTITY_BARE, `$Executable) }'
}
`$definitions = @(`$ast.FindAll({ param(`$node)
    `$node -is [Management.Automation.Language.FunctionDefinitionAst] -and `$replacements.ContainsKey(`$node.Name)
}, `$false))
if (`$parseErrors.Count -ne 0 -or `$definitions.Count -ne `$replacements.Count) { throw 'Native identity fixture seam changed' }
foreach (`$definition in @(`$definitions | Sort-Object { `$_.Body.Extent.StartOffset } -Descending)) {
    `$body = `$definition.Body.Extent
    `$source = `$source.Remove(`$body.StartOffset, `$body.EndOffset - `$body.StartOffset).Insert(`$body.StartOffset, `$replacements[`$definition.Name])
}
& ([ScriptBlock]::Create(`$source)) @options
"@
    }
    if ($PSBoundParameters.ContainsKey('PromptAnswers')) {
        $answersPath = Join-Path $root ($identifier + '.answers.json')
        Write-Utf8 $answersPath (ConvertTo-Json -InputObject @($PromptAnswers))
        $escapedAnswers = $answersPath.Replace("'", "''")
        $invocation = @"
`$global:fixtureAnswers = New-Object 'System.Collections.Generic.Queue[object]'
`$answers = Get-Content -Raw -LiteralPath '$escapedAnswers' | ConvertFrom-Json
foreach (`$answer in `$answers) { `$global:fixtureAnswers.Enqueue(`$answer) }
function Read-Host([string]`$Prompt, [switch]`$AsSecureString) {
    Write-Host `$Prompt
    if (`$global:fixtureAnswers.Count -eq 0) { throw 'Unexpected interactive prompt' }
    `$answer = [string]`$global:fixtureAnswers.Dequeue()
    if (`$AsSecureString) {
        if (`$answer.Length -eq 0) { return New-Object Security.SecureString }
        return ConvertTo-SecureString `$answer -AsPlainText -Force
    }
    return `$answer
}
"@ + "`n" + $invocation + "`nif (`$global:fixtureAnswers.Count -ne 0) { throw 'Unused interactive answers' }"
    }
    if ($UserKeyFixture) {
        if ($NoCredential) { $invocation += "`nif (`$env:LLMP_API_KEY -cne `$env:CODEX_USER_KEY_FIXTURE) { throw 'Saved User credential was not activated' }" }
        else { $invocation += "`nif (`$env:LLMP_API_KEY -cne 'fixture-only-not-a-real-credential') { throw 'Process credential precedence changed' }" }
    }
    $rollbackCheck = ''
    if ($AssertUnchangedEnvironment) {
        $invocation = "`$oldPath = `$env:Path`n`$options.ApiKey = ConvertTo-SecureString 'fixture-replacement-credential' -AsPlainText -Force`n" + $invocation
        $rollbackCheck = @"
if (`$env:LLMP_API_KEY -cne 'fixture-only-not-a-real-credential' -or `$env:Path -cne `$oldPath) { Write-Output 'FAIL refusal changed credential or PATH'; exit 2 }
Write-Output 'PASS refusal preserved credential and PATH'
"@
    }
    if ($FaultAfterFirstCommit) {
        # Fault only the isolated in-memory copy; production transaction code stays intact.
        $invocation = @"
`$source = [IO.File]::ReadAllText('$escapedInstaller')
`$tokens = `$null; `$parseErrors = `$null
`$ast = [Management.Automation.Language.Parser]::ParseInput(`$source, [ref]`$tokens, [ref]`$parseErrors)
`$commits = @(`$ast.FindAll({ param(`$node)
    `$node -is [Management.Automation.Language.FunctionDefinitionAst] -and
    `$node.Name -ceq 'Publish-CodexConfigFile'
}, `$true))
if (`$parseErrors.Count -ne 0 -or `$commits.Count -ne 1) { throw 'Rollback fixture commit seam changed' }
`$fault = ' if ([IO.Path]::GetFileName(`$File.Path) -ceq "llmp.config.toml") { throw "fixture-second-promotion-failure" }; '
`$source = `$source.Insert(`$commits[0].Body.Extent.StartOffset + 1, `$fault)
`$options.ApiKey = ConvertTo-SecureString 'fixture-replacement-credential' -AsPlainText -Force
`$oldPath = `$env:Path
& ([ScriptBlock]::Create(`$source)) @options
"@
        $rollbackCheck = @"
if (`$env:LLMP_API_KEY -cne 'fixture-only-not-a-real-credential') { Write-Output 'FAIL rollback process credential'; exit 2 }
if (`$env:Path -cne `$oldPath) { Write-Output 'FAIL rollback process PATH'; exit 2 }
Write-Output 'PASS rollback process credential and PATH'
"@
    }
    Write-Utf8 $runner @"
`$ErrorActionPreference = 'Stop'
`$options = @{}
`$data = Get-Content -Raw -LiteralPath '$escapedArguments' | ConvertFrom-Json
foreach (`$property in `$data.PSObject.Properties) { `$options[`$property.Name] = `$property.Value }
try { $invocation; exit 0 } catch { $rollbackCheck
Write-Output `$_.Exception.Message; exit 1 }
"@
    $start = New-Object Diagnostics.ProcessStartInfo
    $start.FileName = $shellExecutable
    $start.Arguments = '-NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "' + $runner + '"'
    $start.UseShellExecute = $false
    $start.RedirectStandardOutput = $true
    $start.RedirectStandardError = $true
    $start.EnvironmentVariables['CODEX_HOME'] = $HomeDirectory
    $start.EnvironmentVariables['LLMP_API_KEY'] = $dummyKey
    $start.EnvironmentVariables['CODEX_USER_KEY_FIXTURE'] = $UserKeyFixture
    if ($OfficialCatalogFixture) { $start.EnvironmentVariables['CODEX_OFFICIAL_CATALOG_FIXTURE'] = $OfficialCatalogFixture }
    if ($NoCredential) { $start.EnvironmentVariables['LLMP_API_KEY'] = '' }
    foreach ($name in $EnvironmentOverrides.Keys) { $start.EnvironmentVariables[$name] = [string]$EnvironmentOverrides[$name] }
    foreach ($name in @('TEMP', 'TMP', 'TMPDIR')) { $start.EnvironmentVariables[$name] = $root }
    $process = New-Object Diagnostics.Process
    $process.StartInfo = $start
    try {
        $process.Start() | Out-Null
        $stdoutTask = $process.StandardOutput.ReadToEndAsync()
        $stderrTask = $process.StandardError.ReadToEndAsync()
        if (-not $process.WaitForExit(30000)) { $process.Kill(); throw 'Installer exceeded 30 second deadline' }
        $output = $stdoutTask.GetAwaiter().GetResult() + $stderrTask.GetAwaiter().GetResult()
        Assert-True (-not $output.Contains($dummyKey)) 'Installer exposed dummy credential in output'
        if ($UserKeyFixture) { Assert-True (-not $output.Contains($UserKeyFixture)) 'Installer exposed saved User credential' }
        Assert-True (-not $output.Contains('fixture-replacement-credential')) 'Installer exposed replacement credential in output'
        if ($Failure) { Assert-True ($process.ExitCode -ne 0) 'Invalid input unexpectedly succeeded' }
        else { Assert-True ($process.ExitCode -eq 0) ('Installer failed: ' + $output) }
        return $output
    } finally { $process.Dispose() }
}
function Verify-Toml([string]$Path, [hashtable]$Extra = @{}, [switch]$AllowProfiles) {
    $values = @{
        model_provider = 'oneline_llmp'; model = 'fixture-model'; service_tier = 'default'
        model_context_window = 353000; model_auto_compact_token_limit = 300000
        'model_providers.oneline_llmp.base_url' = 'https://fixture.invalid/v1'
        'model_providers.oneline_llmp.wire_api' = 'responses'
        'model_providers.oneline_llmp.env_key' = 'LLMP_API_KEY'
        'model_providers.oneline_llmp.requires_openai_auth' = $false
    }
    foreach ($name in $Extra.Keys) { $values[$name] = $Extra[$name] }
    $expectation = Join-Path $root ([Guid]::NewGuid().ToString('N') + '.expected.json')
    $absent = @()
    if (-not $AllowProfiles) { $absent += 'profiles' }
    foreach ($setting in @('model_catalog_json')) {
        if (-not $Extra.ContainsKey($setting)) { $absent += $setting }
    }
    Write-Utf8 $expectation (@{values = $values; forbidden = @($dummyKey); absent = $absent} | ConvertTo-Json -Depth 10)
    & $Python (Join-Path $PSScriptRoot 'verify_config.py') $Path $expectation
    Assert-True ($LASTEXITCODE -eq 0) 'TOML semantic verification failed'
}
function Test-Case([string]$Name, [scriptblock]$Body) {
    if ($Name -notlike $TestFilter) { return }
    try { & $Body; $script:passed++; Write-Output ('PASS ' + $Name) }
    catch { $script:failures += $Name + ': ' + $_.Exception.Message; Write-Output ('FAIL ' + $script:failures[-1]) }
}

Test-Case 'SSH requires interactive choices before Codex mutations' {
    $homeDirectory = New-Case 'ssh-noninteractive-refusal'
    $output = Invoke-Installer $homeDirectory @{EnableSshServer = $true} -Failure
    Assert-True ($output.Contains('SSH server setup requires interactive mode')) 'Wrong SSH refusal reason'
    Assert-True (@(Get-ChildItem $homeDirectory -Force).Count -eq 0) 'SSH refusal changed Codex files'
}
Test-Case 'SSH flag in dry run does not install or download anything' {
    $homeDirectory = Join-Path $root 'ssh-dry-run-absent'
    Invoke-Installer $homeDirectory @{EnableSshServer = $true; DryRun = $true} | Out-Null
    Assert-True (-not (Test-Path $homeDirectory)) 'SSH dry run created a configuration directory'
}
Test-Case 'fresh profile and environment authentication' {
    $homeDirectory = Join-Path $root 'fresh'
    Invoke-Installer $homeDirectory | Out-Null
    Verify-Toml (Get-Profile $homeDirectory)
    Verify-Toml (Join-Path $homeDirectory 'config.toml')
}
Test-Case 'downloaded scriptblock bootstrap supports identical interface' {
    $homeDirectory = New-Case 'scriptblock'
    Invoke-Installer $homeDirectory -Bootstrap | Out-Null
    Verify-Toml (Get-Profile $homeDirectory)
    Verify-Toml (Join-Path $homeDirectory 'config.toml')
}
Test-Case 'omitted endpoint fails without selecting a host' {
    $homeDirectory = New-Case 'omitted-endpoint'
    $output = Invoke-Installer $homeDirectory @{BaseUrl = $null} -Failure
    Assert-True ($output.Contains('Required input')) 'Missing endpoint did not produce required-input error'
    Assert-True (@(Get-ChildItem $homeDirectory -Force).Count -eq 0) 'Missing endpoint created config'
}
Test-Case 'noninteractive missing credential fails without mutation' {
    $homeDirectory = New-Case 'missing-credential'
    $output = Invoke-Installer $homeDirectory -NoCredential -Failure
    Assert-True ($output.Contains('API key missing')) 'Missing credential was not diagnosed'
    Assert-True (@(Get-ChildItem $homeDirectory -Force).Count -eq 0) 'Missing credential created config'
}
Test-Case 'dry run works without credentials' {
    $homeDirectory = New-Case 'dry-run-no-credential'
    Invoke-Installer $homeDirectory @{DryRun = $true} -NoCredential | Out-Null
    Assert-True (@(Get-ChildItem $homeDirectory -Force).Count -eq 0) 'Credential-free dry run created config'
}
foreach ($versionCase in @(
    @{Name = 'stderr-warning'; Version = 'codex-cli 0.165.0'; ExitCode = 0; Fail = $false},
    @{Name = 'nonzero-exit'; Version = 'codex-cli 0.165.0'; ExitCode = 7; Fail = $true},
    @{Name = 'malformed-version'; Version = 'not-a-version'; ExitCode = 0; Fail = $true}
)) {
    Test-Case ('native version probe: ' + $versionCase.Name) {
        $homeDirectory = New-Case ('version-home-' + $versionCase.Name)
        $fakeDirectory = New-Case ('version-bin-' + $versionCase.Name)
        $marker = Join-Path $fakeDirectory 'called.txt'
        if ($env:OS -eq 'Windows_NT') {
            $fakeCommand = Join-Path $fakeDirectory 'codex.cmd'
            $commandText = '@echo off' + "`r`n" +
                'if not "%~1"=="--version" goto native' + "`r`n" +
                'echo called>"%CODEX_VERSION_MARKER%"' + "`r`n" +
                'echo fixture PATH alias warning 1>&2' + "`r`n" +
                'echo ' + $versionCase.Version + "`r`n" +
                'exit /b ' + $versionCase.ExitCode + "`r`n" +
                ':native' + "`r`n" + 'call "%CODEX_TEST_NATIVE%" %*' + "`r`n" + 'exit /b %errorlevel%' + "`r`n"
        } else {
            $fakeCommand = Join-Path $fakeDirectory 'codex'
            $commandText = '#!/bin/sh' + "`n" +
                'if [ "$1" != "--version" ]; then exec "$CODEX_TEST_NATIVE" "$@"; fi' + "`n" +
                'printf called > "$CODEX_VERSION_MARKER"' + "`n" +
                'echo fixture PATH alias warning >&2' + "`n" +
                'echo ' + $versionCase.Version + "`n" +
                'exit ' + $versionCase.ExitCode + "`n"
        }
        Write-Utf8 $fakeCommand $commandText
        if ($env:OS -ne 'Windows_NT') { & chmod +x $fakeCommand; Assert-True ($LASTEXITCODE -eq 0) 'Cannot make fixture executable' }
        $environment = @{
            CODEX_VERSION_EXECUTABLE = $fakeCommand
            CODEX_VERSION_MARKER = $marker
            CODEX_TEST_NATIVE = $nativeCodex
        }
        $output = Invoke-Installer $homeDirectory -EnvironmentOverrides $environment -Failure:$versionCase.Fail -UseVersionFixture
        Assert-True (Test-Path -LiteralPath $marker) 'Version fixture was not invoked'
        Assert-True (-not $output.Contains('fixture PATH alias warning')) 'Version probe exposed native stderr'
        if ($versionCase.Fail) {
            Assert-True ($output.Contains('version could not be verified')) 'Invalid native version did not receive a version error'
            Assert-True (@(Get-ChildItem $homeDirectory -Force).Count -eq 0) 'Invalid native version created config'
        } else {
            Verify-Toml (Get-Profile $homeDirectory)
            Verify-Toml (Join-Path $homeDirectory 'config.toml')
        }
    }
}
Test-Case 'idempotent reruns' {
    $homeDirectory = New-Case 'idempotent'
    Invoke-Installer $homeDirectory | Out-Null
    $profile = Get-Profile $homeDirectory
    $original = [Convert]::ToBase64String([IO.File]::ReadAllBytes($profile))
    $base = Join-Path $homeDirectory 'config.toml'
    $baseOriginal = [Convert]::ToBase64String([IO.File]::ReadAllBytes($base))
    $before = @(Get-ChildItem $homeDirectory -File).Count
    Invoke-Installer $homeDirectory | Out-Null
    Assert-True ($original -ceq [Convert]::ToBase64String([IO.File]::ReadAllBytes($profile))) 'Rerun changed bytes'
    Assert-True ($baseOriginal -ceq [Convert]::ToBase64String([IO.File]::ReadAllBytes($base))) 'Rerun changed main config bytes'
    Assert-True ($before -eq @(Get-ChildItem $homeDirectory -File).Count) 'Rerun created redundant backup'
    Verify-Toml $profile
}
Test-Case 'preserve base config including unrelated tables' {
    $homeDirectory = New-Case 'preserve'
    $base = Join-Path $homeDirectory 'config.toml'
    $original = "# user settings`r`nmodel = 'existing-default'`r`n[projects.'C:/example']`r`ntrust_level = 'trusted'`r`n[profiles.existing]`r`nmodel = 'another-model'`r`n"
    Write-Utf8 $base $original
    $baseBytes = [Convert]::ToBase64String([IO.File]::ReadAllBytes($base))
    Invoke-Installer $homeDirectory | Out-Null
    Verify-Toml $base @{'profiles.existing.model' = 'another-model'} -AllowProfiles
    $merged = [IO.File]::ReadAllText($base)
    foreach ($preserved in @('# user settings', "[projects.'C:/example']", "trust_level = 'trusted'", '[profiles.existing]', "model = 'another-model'")) {
        Assert-True ($merged.Contains($preserved)) 'Merge lost unrelated table or comment'
    }
    $backups = @(Get-ChildItem $homeDirectory -File -Filter 'config.toml.backup-*')
    Assert-True ($backups.Count -eq 1) 'Expected original main config backup'
    Assert-True ($baseBytes -ceq [Convert]::ToBase64String([IO.File]::ReadAllBytes($backups[0].FullName))) 'Main backup differs from original bytes'
    Verify-Toml (Get-Profile $homeDirectory)
}
Test-Case 'native merge preserves arbitrary TOML values and comments' {
    $homeDirectory = New-Case 'arbitrary-toml'
    $base = Join-Path $homeDirectory 'config.toml'
    $original = @'
# preserve heading
model = "old-model" # preserve model comment
model_provider = "openai"
custom_inline = { quoted = "keep", numbers = [1, 2] } # preserve inline comment
custom_date = 1979-05-27T07:32:00Z
custom_multiline = """line one
line two with [table] and model = 'decoy'
"""
"custom.quoted" = 'literal'
custom_dotted.child = 'keep dotted value'

[model_providers.oneline_llmp]
name = "Old provider"
base_url = "https://fixture.invalid/old"
wire_api = "responses"
request_max_retries = 7 # preserve provider comment

[profiles.existing]
model = "another-model" # preserve profile comment

[mcp_servers.fixture]
command = "fixture-command"
args = ["a", "b"] # preserve args comment

[[custom_array]]
name = 'literal'
'@
    Write-Utf8 $base $original
    $before = Join-Path $root 'arbitrary-before.toml'
    [IO.File]::Copy($base, $before)
    Invoke-Installer $homeDirectory | Out-Null
    Verify-Toml $base @{'profiles.existing.model' = 'another-model'; 'model_providers.oneline_llmp.request_max_retries' = 7} -AllowProfiles
    $check = @'
import pathlib, sys, tomllib
old = tomllib.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8-sig"))
new = tomllib.loads(pathlib.Path(sys.argv[2]).read_text(encoding="utf-8-sig"))
for key in ("custom_inline", "custom_date", "custom_multiline", "custom.quoted", "custom_dotted", "custom_array", "profiles", "mcp_servers"):
    assert old[key] == new[key], "Unrelated TOML value changed"
'@
    $checkPath = Join-Path $root 'verify-arbitrary-toml.py'
    Write-Utf8 $checkPath $check
    & $Python $checkPath $before $base
    Assert-True ($LASTEXITCODE -eq 0) 'Arbitrary TOML semantics changed'
    $merged = [IO.File]::ReadAllText($base)
    foreach ($comment in @('# preserve heading', '# preserve model comment', '# preserve inline comment', '# preserve provider comment', '# preserve profile comment', '# preserve args comment')) {
        Assert-True ($merged.Contains($comment)) 'Native merge lost a comment'
    }
    $backups = @(Get-ChildItem $homeDirectory -File -Filter 'config.toml.backup-*')
    Assert-True ($backups.Count -eq 1) 'Arbitrary TOML merge did not back up main config'
    Assert-True ([Convert]::ToBase64String([IO.File]::ReadAllBytes($before)) -ceq [Convert]::ToBase64String([IO.File]::ReadAllBytes($backups[0].FullName))) 'Arbitrary TOML backup changed bytes'
    Verify-Toml (Get-Profile $homeDirectory)
}
foreach ($invalidBase in @('invalid-toml', 'conflicting-auth')) {
    Test-Case ('native merge refuses ' + $invalidBase + ' without mutation') {
        $homeDirectory = New-Case $invalidBase
        $base = Join-Path $homeDirectory 'config.toml'
        $original = "model = [`n"
        if ($invalidBase -eq 'conflicting-auth') {
            $original = "[model_providers.oneline_llmp]`nname = 'existing'`nbase_url = 'https://fixture.invalid/v1'`nwire_api = 'responses'`nhttp_headers = { Authorization = 'fixture-inline-secret' }`n"
        }
        Write-Utf8 $base $original
        $output = Invoke-Installer $homeDirectory -Failure
        Assert-True (-not $output.Contains('fixture-inline-secret')) 'Merge refusal leaked dependency payload'
        Assert-True ([IO.File]::ReadAllText($base) -ceq $original) 'Refused main config changed'
        Assert-True (-not (Test-Path (Get-Profile $homeDirectory))) 'Refused main config created named profile'
        Assert-True (@(Get-ChildItem $homeDirectory -File).Count -eq 1) 'Refused main config created backups or temporary files'
    }
}
foreach ($rollbackState in @('both-existing', 'both-absent', 'main-only', 'named-only')) {
    Test-Case ('second file promotion fault rolls back ' + $rollbackState) {
        $homeDirectory = New-Case ('rollback-' + $rollbackState)
        Invoke-Installer $homeDirectory | Out-Null
        $base = Join-Path $homeDirectory 'config.toml'
        $profile = Get-Profile $homeDirectory
        if ($rollbackState -in @('both-absent', 'named-only')) { [IO.File]::Delete($base) }
        if ($rollbackState -in @('both-absent', 'main-only')) { [IO.File]::Delete($profile) }
        $snapshots = @{}
        foreach ($path in @($base, $profile)) {
            $snapshots[$path] = $null
            if ([IO.File]::Exists($path)) { $snapshots[$path] = [Convert]::ToBase64String([IO.File]::ReadAllBytes($path)) }
        }
        $beforeFiles = @(Get-ChildItem $homeDirectory -File).Count
        $output = Invoke-Installer $homeDirectory @{ContextWindow = 500000; AutoCompactTokenLimit = 450000} -Failure -FaultAfterFirstCommit
        Assert-True ($output.Contains('PASS rollback process credential and PATH')) 'Rollback fixture did not verify old credential and PATH'
        Assert-True ($output.Contains('Setup failed.')) ('Injected promotion fault was not reached: ' + $output)
        foreach ($path in @($base, $profile)) {
            if ($null -eq $snapshots[$path]) { Assert-True (-not [IO.File]::Exists($path)) 'Rollback left a newly created configuration' }
            else { Assert-True ($snapshots[$path] -ceq [Convert]::ToBase64String([IO.File]::ReadAllBytes($path))) 'Rollback did not restore original configuration bytes' }
        }
        Assert-True (@(Get-ChildItem $homeDirectory -File).Count -eq $beforeFiles) 'Rollback left backup or temporary files'
    }
}
Test-Case 'unmanaged profile refused without losing existing tables' {
    $homeDirectory = New-Case 'unmanaged'
    $profile = Get-Profile $homeDirectory
    $original = "# own profile`r`nmodel = 'existing'`r`n[features]`r`nexample = true`r`n"
    Write-Utf8 $profile $original
    Invoke-Installer $homeDirectory -Failure | Out-Null
    Assert-True ([IO.File]::ReadAllText($profile) -ceq $original) 'Unmanaged profile changed'
    Assert-True (@(Get-ChildItem $homeDirectory -File).Count -eq 1) 'Refusal wrote backup'
}
Test-Case 'changed context updates managed block and backs up' {
    $homeDirectory = New-Case 'update'
    Invoke-Installer $homeDirectory | Out-Null
    $originalBytes = [Convert]::ToBase64String([IO.File]::ReadAllBytes((Get-Profile $homeDirectory)))
    $base = Join-Path $homeDirectory 'config.toml'
    $originalMainBytes = [Convert]::ToBase64String([IO.File]::ReadAllBytes($base))
    Invoke-Installer $homeDirectory @{ContextWindow = 500000; AutoCompactTokenLimit = 450000} | Out-Null
    $backups = @(Get-ChildItem $homeDirectory -File | Where-Object { $_.Name -like 'llmp.config.toml.backup-*' })
    Assert-True ($backups.Count -eq 1) 'Expected one changed-profile backup'
    Assert-True ($originalBytes -ceq [Convert]::ToBase64String([IO.File]::ReadAllBytes($backups[0].FullName))) 'Backup differs from original bytes'
    $mainBackups = @(Get-ChildItem $homeDirectory -File -Filter 'config.toml.backup-*')
    Assert-True ($mainBackups.Count -eq 1) 'Expected one changed-main backup'
    Assert-True ($originalMainBytes -ceq [Convert]::ToBase64String([IO.File]::ReadAllBytes($mainBackups[0].FullName))) 'Main backup differs from original bytes'
    Verify-Toml $base @{model_context_window = 500000; model_auto_compact_token_limit = 450000}
    Verify-Toml (Get-Profile $homeDirectory) @{model_context_window = 500000; model_auto_compact_token_limit = 450000}
}
Test-Case 'manual edits inside owned block are preserved by refusal' {
    $homeDirectory = New-Case 'edited-owned'
    Invoke-Installer $homeDirectory | Out-Null
    $profile = Get-Profile $homeDirectory
    $edited = [IO.File]::ReadAllText($profile).Replace($endMarker, "[features]`nexample = true`n" + $endMarker)
    Write-Utf8 $profile $edited
    Invoke-Installer $homeDirectory -Failure | Out-Null
    Assert-True ([IO.File]::ReadAllText($profile) -ceq $edited) 'Owned manual edit was overwritten'
    Assert-True (@(Get-ChildItem $homeDirectory -File).Count -eq 2) 'Refusal created backup'
}
foreach ($unsafeUrlKind in @('query', 'userinfo')) {
    Test-Case ('owned profile refuses secret-bearing base URL: ' + $unsafeUrlKind) {
        $homeDirectory = New-Case ('existing-url-' + $unsafeUrlKind)
        Invoke-Installer $homeDirectory | Out-Null
        $profile = Get-Profile $homeDirectory
        $base = Join-Path $homeDirectory 'config.toml'
        Write-Utf8 $base "# unrelated settings`nmodel = 'original'`n"
        $secretSentinel = 'fixture-url-secret'
        $unsafeUrl = 'https://fixture.invalid/v1?key=' + $secretSentinel
        if ($unsafeUrlKind -eq 'userinfo') { $unsafeUrl = 'https://fixture:' + $secretSentinel + '@fixture.invalid/v1' }
        $edited = [IO.File]::ReadAllText($profile).Replace('https://fixture.invalid/v1', $unsafeUrl)
        Write-Utf8 $profile $edited
        $profileBytes = [Convert]::ToBase64String([IO.File]::ReadAllBytes($profile))
        $baseBytes = [Convert]::ToBase64String([IO.File]::ReadAllBytes($base))
        $output = Invoke-Installer $homeDirectory @{ContextWindow = 500000} -Failure
        Assert-True ($output.Contains('unsafe base URL')) 'Existing secret-bearing endpoint was not diagnosed'
        Assert-True (-not $output.Contains($secretSentinel)) 'Existing URL credential appeared in diagnostics'
        Assert-True ($profileBytes -ceq [Convert]::ToBase64String([IO.File]::ReadAllBytes($profile))) 'Rejected profile changed bytes'
        Assert-True ($baseBytes -ceq [Convert]::ToBase64String([IO.File]::ReadAllBytes($base))) 'Rejected update changed base config'
        Assert-True (@(Get-ChildItem $homeDirectory -Force).Count -eq 2) 'Rejected update created backup or temporary files'
    }
}
Test-Case 'default tier overrides unrelated inherited service tier' {
    $homeDirectory = New-Case 'inherited-tier'
    $base = Join-Path $homeDirectory 'config.toml'
    $original = "service_tier = 'ultrafast'`n"
    Write-Utf8 $base $original
    Invoke-Installer $homeDirectory | Out-Null
    Verify-Toml (Get-Profile $homeDirectory) @{service_tier = 'default'}
    Verify-Toml $base @{service_tier = 'default'}
}
Test-Case 'default compaction is ninety percent' {
    $homeDirectory = New-Case 'default-compaction'
    Invoke-Installer $homeDirectory @{AutoCompactTokenLimit = 0} | Out-Null
    Verify-Toml (Get-Profile $homeDirectory) @{model_auto_compact_token_limit = 317700}
}
Test-Case 'dry run has no filesystem mutations' {
    $homeDirectory = Join-Path $root 'dry-run-absent'
    Invoke-Installer $homeDirectory @{DryRun = $true} | Out-Null
    Assert-True (-not (Test-Path $homeDirectory)) 'Dry run created directory'
}
Test-Case 'explicit service tier is a root field' {
    $homeDirectory = New-Case 'tier'
    Invoke-Installer $homeDirectory @{ServiceTier = 'fast'; SupportedServiceTiers = @('fast')} | Out-Null
    Verify-Toml (Get-Profile $homeDirectory) @{service_tier = 'priority'}
}
Test-Case 'loopback HTTP is accepted without a network request' {
    $homeDirectory = New-Case 'loopback'
    Invoke-Installer $homeDirectory @{BaseUrl = 'http://127.0.0.1:9/v1'} | Out-Null
    Verify-Toml (Get-Profile $homeDirectory) @{'model_providers.oneline_llmp.base_url' = 'http://127.0.0.1:9/v1'}
}
Test-Case 'minimum supported context and compaction boundary' {
    $homeDirectory = New-Case 'minimum'
    Invoke-Installer $homeDirectory @{ContextWindow = 1024; AutoCompactTokenLimit = 1} | Out-Null
    Verify-Toml (Get-Profile $homeDirectory) @{model_context_window = 1024; model_auto_compact_token_limit = 1}
}
function New-Catalog([string]$Name, $Models) {
    $catalog = Join-Path $root ($Name + '.catalog.json')
    Write-Utf8 $catalog (@{models = @($Models)} | ConvertTo-Json -Depth 12)
    return [IO.Path]::GetFullPath($catalog)
}
function Initialize-CompleteCatalogFixture {
    if ($script:completeCatalogFixture) { return }
    $tokens = $null; $parseErrors = $null
    $ast = [Management.Automation.Language.Parser]::ParseFile($installer, [ref]$tokens, [ref]$parseErrors)
    Assert-True ($parseErrors.Count -eq 0) 'Catalog tests could not parse installer'
    foreach ($name in @('Get-OfficialCodexCatalogText', 'ConvertTo-ConfirmedCodexCatalog', 'Get-CatalogSha256', 'Save-ImmutableCodexCatalog', 'Start-CodexConfigProcess', 'Stop-CodexConfigProcess', 'Invoke-CodexConfigRpc', 'Initialize-CodexConfigConnection')) {
        $definitions = @($ast.FindAll({ param($node)
            $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -ceq $name
        }, $false))
        Assert-True ($definitions.Count -eq 1) ('Catalog test helper missing or ambiguous: ' + $name)
        # Import definitions only; the installer entry point must not run here.
        $definition = $definitions[0].Extent.Text.Replace(('function ' + $name), ('function script:' + $name))
        . ([ScriptBlock]::Create($definition))
    }
    $homeDirectory = New-Case 'catalog-version-probe'
    $previousHome = $env:CODEX_HOME
    $previousErrorAction = $ErrorActionPreference
    try {
        $env:CODEX_HOME = $homeDirectory
        $ErrorActionPreference = 'Continue'
        $versionText = (& $nativeCodex --version 2>$null | Out-String)
        $versionExit = $LASTEXITCODE
    } finally {
        $env:CODEX_HOME = $previousHome
        $ErrorActionPreference = $previousErrorAction
    }
    Assert-True ($versionExit -eq 0 -and $versionText -match '(\d+\.\d+\.\d+)') 'Cannot identify installed native catalog version'
    # Fetch the exact public upstream fixture once. No LLMP request or key is used.
    $script:completeCatalogFixture = Get-OfficialCodexCatalogText $Matches[1]
    $script:completeCatalogFixturePath = Join-Path $root 'complete-upstream-catalog.json'
    Write-Utf8 $script:completeCatalogFixturePath $script:completeCatalogFixture
}
Test-Case 'final fix: mixed native selection uses exact path and version for catalog and staging' {
    Initialize-CompleteCatalogFixture
    $homeDirectory = New-Case 'identity-home'
    $environment = @{
        CODEX_IDENTITY_VERSION = (Join-Path $root 'identity-version.txt')
        CODEX_IDENTITY_STAGE = (Join-Path $root 'identity-stage.txt')
        CODEX_IDENTITY_BARE = (Join-Path $root 'identity-bare.txt')
    }
    foreach ($identity in @(@{Name = 'old'; Version = '0.134.0'}, @{Name = 'selected'; Version = '0.166.0'})) {
        $directory = New-Case ('identity-' + $identity.Name)
        $marker = Join-Path $directory 'version-called.txt'
        if ($env:OS -eq 'Windows_NT') {
            $executable = Join-Path $directory 'codex.cmd'
            $text = '@echo off' + "`r`n" + 'echo called>"' + $marker + '"' + "`r`n" +
                'echo codex-cli ' + $identity.Version + "`r`n" + 'exit /b 0' + "`r`n"
        } else {
            $executable = Join-Path $directory 'codex'
            $text = '#!/bin/sh' + "`n" + 'printf called > "' + $marker + '"' + "`n" +
                'echo codex-cli ' + $identity.Version + "`n"
        }
        Write-Utf8 $executable $text
        if ($env:OS -ne 'Windows_NT') { & chmod +x $executable; Assert-True ($LASTEXITCODE -eq 0) 'Cannot make identity fixture executable' }
        $environment[('CODEX_' + $identity.Name.ToUpperInvariant() + '_EXECUTABLE')] = $executable
    }
    Invoke-Installer $homeDirectory @{Model = 'gpt-6.1-sol'} -EnvironmentOverrides $environment -OfficialCatalogFixture $script:completeCatalogFixturePath -UseNativeIdentityFixture | Out-Null
    Assert-True ([IO.File]::ReadAllText($environment.CODEX_IDENTITY_VERSION) -ceq '0.166.0') 'Catalog used standalone 0.134.0 instead of selected native 0.166.0'
    foreach ($path in @($environment.CODEX_IDENTITY_STAGE, $environment.CODEX_IDENTITY_BARE)) {
        Assert-True ([IO.File]::ReadAllText($path) -ceq $environment.CODEX_SELECTED_EXECUTABLE) 'Catalog/staging/bare executable identities differ'
    }
    Assert-True (Test-Path (Join-Path (Split-Path $environment.CODEX_SELECTED_EXECUTABLE) 'version-called.txt')) 'Selected native executable was not version-probed'
    Assert-True (-not (Test-Path (Join-Path (Split-Path $environment.CODEX_OLD_EXECUTABLE) 'version-called.txt'))) 'Unselected standalone executable was version-probed'
}
Test-Case 'final fix: SkipInstall without any native command still writes isolated config' {
    $homeDirectory = New-Case 'no-native'
    Invoke-Installer $homeDirectory -UseVersionFixture -EnvironmentOverrides @{CODEX_VERSION_EXECUTABLE = ''} | Out-Null
    Verify-Toml (Get-Profile $homeDirectory)
    Verify-Toml (Join-Path $homeDirectory 'config.toml')
}
Test-Case 'complete native catalog retains upstream metadata and exposes Ultrafast name' {
    Initialize-CompleteCatalogFixture
    $homeDirectory = New-Case 'complete-catalog-native'
    $catalogJson = ConvertTo-ConfirmedCodexCatalog $script:completeCatalogFixture 'gpt-6.1-sol' @('fast', 'ultrafast')
    $catalog = Save-ImmutableCodexCatalog $homeDirectory $catalogJson
    $entry = ($catalogJson | ConvertFrom-Json).models[0]
    $ultrafast = @($entry.service_tiers | Where-Object { $_.id -ceq 'ultrafast' })
    Assert-True ($ultrafast.Count -eq 1 -and $ultrafast[0].name -ceq 'Ultrafast') 'Catalog lost Ultrafast display name'
    Assert-True (-not [string]::IsNullOrWhiteSpace($ultrafast[0].description)) 'Catalog lost Ultrafast description'
    Assert-True (@($entry.service_tiers | Where-Object { $_.id -ceq 'priority' }).Count -eq 1) 'Fast tier was not normalized to native priority'
    Assert-True ($entry.additional_speed_tiers -ccontains 'fast' -and $entry.additional_speed_tiers -ccontains 'ultrafast') 'Legacy speed tiers were not preserved'
    $check = @'
import json, pathlib, sys
upstream = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8-sig"))
generated = json.loads(pathlib.Path(sys.argv[2]).read_text(encoding="utf-8-sig"))
assert len(generated["models"]) == 1, "Unconfirmed models advertised"
original = next(m for m in upstream["models"] if m["slug"] == "gpt-6.1-sol")
selected = generated["models"][0]
changed = {"service_tiers", "additional_speed_tiers", "default_service_tier"}
assert {k: v for k, v in original.items() if k not in changed} == {k: v for k, v in selected.items() if k not in changed}, "Complete upstream metadata changed"
assert len(selected) > 10, "Catalog fixture is only a partial model"
'@
    $checkPath = Join-Path $root 'verify-complete-catalog.py'
    Write-Utf8 $checkPath $check
    & $Python $checkPath $script:completeCatalogFixturePath $catalog
    Assert-True ($LASTEXITCODE -eq 0) 'Generated catalog did not preserve complete upstream model metadata'
    # Force the arbitrary-TOML merge path, then validate the native API's actual UI metadata.
    Write-Utf8 (Join-Path $homeDirectory 'config.toml') "# preserve catalog user settings`n[features]`n"
    # An explicit complete catalog must bypass automatic downloading entirely.
    Invoke-Installer $homeDirectory @{Model = 'gpt-6.1-sol'; ModelCatalogPath = $catalog; ServiceTier = 'ultrafast'} -OfficialCatalogFixture (Join-Path $root 'must-not-download.json') | Out-Null
    foreach ($path in @((Get-Profile $homeDirectory), (Join-Path $homeDirectory 'config.toml'))) {
        Verify-Toml $path @{model = 'gpt-6.1-sol'; model_catalog_json = $catalog; service_tier = 'ultrafast'; 'features.ultrafast_mode' = $true}
    }
    $connection = $null
    try {
        $connection = Start-CodexConfigProcess $nativeCodex $homeDirectory $root
        Initialize-CodexConfigConnection $connection
        $models = Invoke-CodexConfigRpc $connection 'model/list' @{}
        $selected = @($models.data | Where-Object { $_.model -ceq 'gpt-6.1-sol' })
        Assert-True ($selected.Count -eq 1) 'Native model/list did not expose the selected complete model'
        $nativeTier = @($selected[0].serviceTiers | Where-Object { $_.id -ceq 'ultrafast' })
        Assert-True ($nativeTier.Count -eq 1 -and $nativeTier[0].name -ceq 'Ultrafast') 'Native model/list lost Ultrafast display name'
        Assert-True (-not [string]::IsNullOrWhiteSpace($nativeTier[0].description)) 'Native model/list lost Ultrafast description'
        $effective = Invoke-CodexConfigRpc $connection 'config/read' @{}
        Assert-True ($effective.config.service_tier -ceq 'ultrafast') 'Native effective tier differs from selected Ultrafast tier'
        Assert-True ($effective.config.features.ultrafast_mode -eq $true) 'Native effective config did not enable the Ultrafast menu'
    } finally { Stop-CodexConfigProcess $connection }
}
Test-Case 'Sol automatically generates a complete catalog and retains Ultrafast UI metadata' {
    Initialize-CompleteCatalogFixture
    $homeDirectory = New-Case 'auto-complete-catalog'
    Invoke-Installer $homeDirectory @{Model = 'gpt-6.1-sol'; ServiceTier = 'ultrafast'} -OfficialCatalogFixture $script:completeCatalogFixturePath | Out-Null
    $catalogs = @(Get-ChildItem (Join-Path $homeDirectory '.catalogs') -File -Filter 'oneline-*.json')
    Assert-True ($catalogs.Count -eq 1) 'Preset setup did not create exactly one complete catalog'
    $catalog = $catalogs[0].FullName
    foreach ($path in @((Get-Profile $homeDirectory), (Join-Path $homeDirectory 'config.toml'))) {
        Verify-Toml $path @{model = 'gpt-6.1-sol'; model_catalog_json = $catalog; service_tier = 'ultrafast'; 'features.ultrafast_mode' = $true}
    }
    $entry = ([IO.File]::ReadAllText($catalog) | ConvertFrom-Json).models[0]
    Assert-True (@($entry.service_tiers | Where-Object { $_.id -ceq 'ultrafast' -and $_.name -ceq 'Ultrafast' }).Count -eq 1) 'Automatic catalog lost Ultrafast name metadata'
    $before = [Convert]::ToBase64String([IO.File]::ReadAllBytes($catalog))
    Invoke-Installer $homeDirectory @{Model = 'gpt-6.1-sol'; ServiceTier = 'ultrafast'} -OfficialCatalogFixture $script:completeCatalogFixturePath | Out-Null
    Assert-True (@(Get-ChildItem (Join-Path $homeDirectory '.catalogs') -File).Count -eq 1) 'Automatic catalog rerun created extra cache files'
    Assert-True ($before -ceq [Convert]::ToBase64String([IO.File]::ReadAllBytes($catalog))) 'Automatic catalog rerun changed bytes'
    Assert-True (@(Get-ChildItem $homeDirectory -File -Filter '*.backup-*').Count -eq 0) 'Automatic catalog rerun created redundant config backups'
}
Test-Case 'complete catalog restricts advertised tiers to explicit confirmation' {
    Initialize-CompleteCatalogFixture
    $limited = ConvertTo-ConfirmedCodexCatalog $script:completeCatalogFixture 'gpt-6.1-sol' @('fast') | ConvertFrom-Json
    Assert-True (@($limited.models[0].service_tiers | Where-Object { $_.id -ceq 'ultrafast' }).Count -eq 0) 'Unconfirmed Ultrafast tier was advertised'
    Assert-True ($limited.models[0].additional_speed_tiers -cnotcontains 'ultrafast') 'Unconfirmed legacy Ultrafast tier was advertised'
    Assert-True (@($limited.models[0].service_tiers | Where-Object { $_.id -ceq 'priority' }).Count -eq 1) 'Confirmed priority tier disappeared'
    foreach ($model in @('glm-5.3', 'GPT-6.1-SOL')) {
        $refused = $false
        try { ConvertTo-ConfirmedCodexCatalog $script:completeCatalogFixture $model @('ultrafast') | Out-Null }
        catch { $refused = $true }
        Assert-True $refused 'Builder invented native metadata for an unsupported or mismatched model'
    }
}
foreach ($selection in @('default', 'fast', 'ultrafast', 'empty-restriction')) {
    Test-Case ('final fix: supplied catalog refuses excluded advertisement with ' + $selection) {
        Initialize-CompleteCatalogFixture
        $homeDirectory = New-Case ('catalog-conflict-' + $selection)
        Invoke-Installer $homeDirectory | Out-Null
        $catalog = Join-Path $homeDirectory 'supplied.json'
        Write-Utf8 $catalog (ConvertTo-ConfirmedCodexCatalog $script:completeCatalogFixture 'gpt-6.1-sol' @('fast', 'ultrafast'))
        $paths = @((Join-Path $homeDirectory 'config.toml'), (Get-Profile $homeDirectory), $catalog)
        $before = @($paths | ForEach-Object { [Convert]::ToBase64String([IO.File]::ReadAllBytes($_)) })
        $entries = @(Get-ChildItem $homeDirectory -Force | Select-Object -ExpandProperty Name)
        $options = @{Model = 'gpt-6.1-sol'; ModelCatalogPath = $catalog; SupportedServiceTiers = @('fast')}
        if ($selection -eq 'empty-restriction') { $options.SupportedServiceTiers = @() }
        elseif ($selection -ne 'default') { $options.ServiceTier = $selection }
        $output = Invoke-Installer $homeDirectory $options -Failure -AssertUnchangedEnvironment
        Assert-True ($output.Contains('catalog advertises tiers excluded by explicit SupportedServiceTiers')) 'Conflicting catalog/list did not receive the restriction error'
        Assert-True ($output.Contains('PASS refusal preserved credential and PATH')) 'Refusal did not check credential/PATH state'
        for ($i = 0; $i -lt $paths.Count; $i++) {
            Assert-True ($before[$i] -ceq [Convert]::ToBase64String([IO.File]::ReadAllBytes($paths[$i]))) 'Refusal changed main/named config or supplied catalog bytes'
        }
        Assert-True ((@(Get-ChildItem $homeDirectory -Force | Select-Object -ExpandProperty Name) -join '|') -ceq ($entries -join '|')) 'Refusal created backups, staging or catalog files'
    }
}
foreach ($restriction in @('fast', 'priority')) {
    Test-Case ('final fix: supplied catalog accepts normalized ' + $restriction + ' restriction') {
        Initialize-CompleteCatalogFixture
        $homeDirectory = New-Case ('catalog-alias-' + $restriction)
        $json = ConvertTo-ConfirmedCodexCatalog $script:completeCatalogFixture 'gpt-6.1-sol' @('fast')
        $catalog = Join-Path $homeDirectory 'supplied.json'
        Write-Utf8 $catalog $json
        $before = [Convert]::ToBase64String([IO.File]::ReadAllBytes($catalog))
        Invoke-Installer $homeDirectory @{Model = 'gpt-6.1-sol'; ModelCatalogPath = $catalog; SupportedServiceTiers = @($restriction); ServiceTier = 'fast'} | Out-Null
        foreach ($path in @((Get-Profile $homeDirectory), (Join-Path $homeDirectory 'config.toml'))) {
            Verify-Toml $path @{model = 'gpt-6.1-sol'; model_catalog_json = $catalog; service_tier = 'priority'}
        }
        Assert-True ($before -ceq [Convert]::ToBase64String([IO.File]::ReadAllBytes($catalog))) 'Compatible catalog changed source bytes'
    }
}
Test-Case 'complete catalog cache is idempotent and refuses tampered contents' {
    Initialize-CompleteCatalogFixture
    $homeDirectory = New-Case 'complete-catalog-cache'
    $json = ConvertTo-ConfirmedCodexCatalog $script:completeCatalogFixture 'gpt-6.1-sol' @('ultrafast')
    $path = Save-ImmutableCodexCatalog $homeDirectory $json
    Assert-True ($path -ceq (Save-ImmutableCodexCatalog $homeDirectory $json)) 'Catalog cache rerun changed its content-addressed path'
    Assert-True (@(Get-ChildItem (Split-Path $path) -File).Count -eq 1) 'Catalog cache rerun left redundant files'
    Write-Utf8 $path 'fixture tampered catalog'
    $refused = $false
    try { Save-ImmutableCodexCatalog $homeDirectory $json | Out-Null } catch { $refused = $true }
    Assert-True $refused 'Catalog cache accepted tampered content'
    Assert-True ([IO.File]::ReadAllText($path) -ceq 'fixture tampered catalog') 'Catalog cache overwrote tampered content'
}
Test-Case 'native catalog is preserved and referenced at root with exact model tier' {
    $catalog = New-Catalog 'native' @(@{slug = 'fixture-model'; service_tiers = @(@{id = 'fast'})})
    $original = [IO.File]::ReadAllText($catalog)
    $homeDirectory = New-Case 'catalog-valid'
    Invoke-Installer $homeDirectory @{ModelCatalogPath = $catalog; ServiceTier = 'fast'} | Out-Null
    Verify-Toml (Get-Profile $homeDirectory) @{model_catalog_json = $catalog; service_tier = 'priority'}
    Assert-True ([IO.File]::ReadAllText($catalog) -ceq $original) 'Catalog source changed'
    Invoke-Installer $homeDirectory @{ModelCatalogPath = $catalog; ServiceTier = 'fast'} | Out-Null
    $before = @(Get-ChildItem $homeDirectory -File).Count
    Invoke-Installer $homeDirectory @{ModelCatalogPath = $catalog; ServiceTier = 'fast'} | Out-Null
    Assert-True (@(Get-ChildItem $homeDirectory -File).Count -eq $before) 'Catalog profile rerun was not idempotent'
    Verify-Toml (Join-Path $homeDirectory 'config.toml') @{model_catalog_json = $catalog; service_tier = 'priority'}
}
Test-Case 'legacy catalog speed tiers are supported' {
    $catalog = New-Catalog 'legacy' @(@{slug = 'fixture-model'; additional_speed_tiers = @('fast')})
    $homeDirectory = New-Case 'catalog-legacy'
    Invoke-Installer $homeDirectory @{ModelCatalogPath = $catalog; ServiceTier = 'fast'} | Out-Null
    Verify-Toml (Get-Profile $homeDirectory) @{model_catalog_json = $catalog; service_tier = 'priority'}
}
$catalogFailures = @{
    'missing-model' = @(@{slug = 'other-model'; service_tiers = @(@{id = 'fast'})})
    'case-mismatch' = @(@{slug = 'FIXTURE-MODEL'; service_tiers = @(@{id = 'fast'})})
    'duplicate-slug' = @(@{slug = 'fixture-model'}, @{slug = 'fixture-model'})
    'tier-other-model' = @(@{slug = 'fixture-model'; service_tiers = @(@{id = 'flex'})}, @{slug = 'other-model'; service_tiers = @(@{id = 'fast'})})
    'invalid-tier' = @(@{slug = 'fixture-model'; service_tiers = @(@{id = 42})})
}
foreach ($catalogCase in $catalogFailures.Keys) {
    Test-Case ('catalog rejects ' + $catalogCase) {
        $catalog = New-Catalog $catalogCase $catalogFailures[$catalogCase]
        $homeDirectory = New-Case ('catalog-' + $catalogCase)
        Invoke-Installer $homeDirectory @{ModelCatalogPath = $catalog; ServiceTier = 'fast'; SupportedServiceTiers = @('fast')} -Failure | Out-Null
        Assert-True (@(Get-ChildItem $homeDirectory -Force).Count -eq 0) 'Invalid catalog created config'
    }
}
Test-Case 'malformed catalog fails without config mutation' {
    $catalog = Join-Path $root 'malformed.catalog.json'
    Write-Utf8 $catalog '{invalid json'
    $homeDirectory = New-Case 'catalog-malformed'
    Invoke-Installer $homeDirectory @{ModelCatalogPath = $catalog} -Failure | Out-Null
    Assert-True (@(Get-ChildItem $homeDirectory -Force).Count -eq 0) 'Malformed catalog created config'
}
Test-Case 'missing catalog fails without config mutation' {
    $homeDirectory = New-Case 'catalog-missing-file'
    Invoke-Installer $homeDirectory @{ModelCatalogPath = (Join-Path $root 'absent.json')} -Failure | Out-Null
    Assert-True (@(Get-ChildItem $homeDirectory -Force).Count -eq 0) 'Missing catalog created config'
}
$invalidInputs = @(
    @{BaseUrl = 'http://remote.invalid/v1'}, @{BaseUrl = 'https://user:pass@fixture.invalid/v1'},
    @{BaseUrl = 'https://fixture.invalid/v1?key=secret'}, @{BaseUrl = 'https://fixture.invalid/v1#fragment'},
    @{BaseUrl = 'file:///example'}, @{BaseUrl = 'relative/path'}, @{BaseUrl = ''},
    @{Model = "bad'model"}, @{Model = "bad`nmodel"}, @{Model = ''},
    @{ContextWindow = -1}, @{ContextWindow = 1023}, @{ContextWindow = 2147483648}, @{ContextWindow = 0},
    @{AutoCompactTokenLimit = -1}, @{AutoCompactTokenLimit = 353000},
    @{ServiceTier = 'fast'}
)
$index = 0
foreach ($invalid in $invalidInputs) {
    $index++
    Test-Case ('invalid input ' + $index) {
        $homeDirectory = New-Case ('invalid-' + $index)
        $output = Invoke-Installer $homeDirectory $invalid -Failure
        $expectedError = 'Required input'
        if ($invalid.ContainsKey('BaseUrl') -and $invalid.BaseUrl) { $expectedError = 'BaseUrl must' }
        elseif ($invalid.ContainsKey('Model') -and $invalid.Model) { $expectedError = 'Configuration values' }
        elseif ($invalid.ContainsKey('ContextWindow') -and $invalid.ContextWindow -ne 0) { $expectedError = 'ContextWindow must' }
        elseif ($invalid.ContainsKey('AutoCompactTokenLimit')) { $expectedError = 'AutoCompactTokenLimit must' }
        elseif ($invalid.ContainsKey('ServiceTier')) { $expectedError = 'ServiceTier requires' }
        Assert-True ($output.Contains($expectedError)) ('Expected validation error: ' + $expectedError + '; got: ' + $output)
        Assert-True (@(Get-ChildItem $homeDirectory -Force).Count -eq 0) 'Invalid input created files'
    }
}
$badBlocks = @{
    'missing-end' = "$beginMarker`nmodel = 'x'`n"
    'missing-start' = "$endMarker`n"
    'reversed' = "$endMarker`n$beginMarker`n"
    'duplicate' = "$beginMarker`n$endMarker`n$beginMarker`n$endMarker`n"
    'nested' = "$beginMarker`n$beginMarker`n$endMarker`n$endMarker`n"
    'trailing-table' = "$beginMarker`n$endMarker`n[features]`nx = true`n"
    'reserved-provider' = "[model_providers.oneline_llmp]`nname = 'existing'`n"
    'inline-credential' = "$beginMarker`napi_key = 'fixture-inline-credential'`n$endMarker`n"
    'inline-provider' = "model_providers = { oneline_llmp = { name = 'existing' } }`n"
}
foreach ($blockName in $badBlocks.Keys) {
    Test-Case ('reject malformed or conflicting block: ' + $blockName) {
        $homeDirectory = New-Case $blockName
        $profile = Get-Profile $homeDirectory
        Write-Utf8 $profile $badBlocks[$blockName]
        Invoke-Installer $homeDirectory -Failure | Out-Null
        Assert-True ([IO.File]::ReadAllText($profile) -ceq $badBlocks[$blockName]) 'Refused config changed'
        Assert-True (@(Get-ChildItem $homeDirectory -File).Count -eq 1) 'Refusal created backup or scratch file'
    }
}
$fixture = $null
try {
    $readyFile = Join-Path $root 'discovery.port'
    $requestsFile = Join-Path $root 'discovery.requests'
    $start = New-Object Diagnostics.ProcessStartInfo
    $start.FileName = (Get-Command $Python -ErrorAction Stop).Source
    $start.Arguments = '"' + (Join-Path $PSScriptRoot 'discovery_server.py') + '" --ready-file "' + $readyFile + '" --requests-file "' + $requestsFile + '"'
    $start.UseShellExecute = $false
    foreach ($variable in @('TEMP', 'TMP', 'TMPDIR')) { $start.EnvironmentVariables[$variable] = $root }
    $fixture = [Diagnostics.Process]::Start($start)
    $deadline = [DateTime]::UtcNow.AddSeconds(10)
    while (-not (Test-Path $readyFile)) {
        if ($fixture.HasExited -or [DateTime]::UtcNow -gt $deadline) { throw 'Discovery fixture failed to start' }
        Start-Sleep -Milliseconds 50
    }
    $origin = 'http://127.0.0.1:' + [IO.File]::ReadAllText($readyFile)
    Test-Case 'offline model discovery uses bearer and ignores capability metadata' {
        $homeDirectory = New-Case 'discovery-success'
        $output = Invoke-Installer $homeDirectory @{BaseUrl = $origin + '/ok'; DiscoverModels = $true}
        Assert-True ($output.Contains('fixture-alternative')) 'Discovery did not list fixture IDs'
        Verify-Toml (Get-Profile $homeDirectory) @{'model_providers.oneline_llmp.base_url' = $origin + '/ok'}
    }
    foreach ($route in @('redirect', 'malformed', 'missing', 'invalid-ids')) {
        Test-Case ('offline discovery rejects ' + $route) {
            $homeDirectory = New-Case ('discovery-' + $route)
            Invoke-Installer $homeDirectory @{BaseUrl = $origin + '/' + $route; DiscoverModels = $true} -Failure | Out-Null
            Assert-True (@(Get-ChildItem $homeDirectory -Force).Count -eq 0) 'Failed discovery created configuration'
        }
    }
    Test-Case 'dry run skips requested discovery entirely' {
        $before = [IO.File]::ReadAllText($requestsFile)
        $homeDirectory = New-Case 'discovery-dry-run'
        Invoke-Installer $homeDirectory @{BaseUrl = $origin + '/ok'; DiscoverModels = $true; DryRun = $true} | Out-Null
        Assert-True ([IO.File]::ReadAllText($requestsFile) -ceq $before) 'Dry run performed discovery'
        Assert-True (@(Get-ChildItem $homeDirectory -Force).Count -eq 0) 'Dry run created configuration'
    }
    Test-Case 'discovery never follows redirects' {
        $requests = [IO.File]::ReadAllText($requestsFile)
        Assert-True (-not $requests.Contains('/redirect-target')) 'Discovery followed redirect'
        Assert-True ($requests.Contains('/redirect/models')) 'Redirect test did not reach fixture'
    }
} catch {
    $script:failures += 'Discovery fixture: ' + $_.Exception.Message
    Write-Output ('FAIL ' + $script:failures[-1])
} finally {
    if ($fixture) {
        if (-not $fixture.HasExited) { $fixture.Kill(); $fixture.WaitForExit() }
        $fixture.Dispose()
    }
}
. (Join-Path $PSScriptRoot 'rerun-defaults.ps1')
Write-Output ('Results: ' + $script:passed + ' passed; ' + $script:failures.Count + ' failed. Artifacts: ' + $root)
if ($script:failures.Count -gt 0) { exit 1 }
