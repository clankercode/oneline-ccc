#requires -Version 5.1
[CmdletBinding()]
param([string]$Python = 'python')
$ErrorActionPreference = 'Stop'
$repository = Split-Path $PSScriptRoot -Parent
$installer = Join-Path $repository 'scripts/setup-codex.ps1'
$root = Join-Path $repository ('.cache/installer-tests-' + [Guid]::NewGuid().ToString('N'))
[IO.Directory]::CreateDirectory($root) | Out-Null
$shellExecutable = (Get-Process -Id $PID).Path
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
function Invoke-Installer([string]$HomeDirectory, [hashtable]$Overrides = @{}, [switch]$Failure, [switch]$Bootstrap, [switch]$NoCredential) {
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
    $escapedInstaller = $installer.Replace("'", "''")
    $escapedArguments = $argumentsPath.Replace("'", "''")
    $invocation = "& '$escapedInstaller' @options"
    if ($Bootstrap) { $invocation = "& ([ScriptBlock]::Create([IO.File]::ReadAllText('$escapedInstaller'))) @options" }
    Write-Utf8 $runner @"
`$ErrorActionPreference = 'Stop'
`$options = @{}
`$data = Get-Content -Raw -LiteralPath '$escapedArguments' | ConvertFrom-Json
foreach (`$property in `$data.PSObject.Properties) { `$options[`$property.Name] = `$property.Value }
try { $invocation; exit 0 } catch { Write-Output `$_.Exception.Message; exit 1 }
"@
    $start = New-Object Diagnostics.ProcessStartInfo
    $start.FileName = $shellExecutable
    $start.Arguments = '-NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "' + $runner + '"'
    $start.UseShellExecute = $false
    $start.RedirectStandardOutput = $true
    $start.RedirectStandardError = $true
    $start.EnvironmentVariables['CODEX_HOME'] = $HomeDirectory
    $start.EnvironmentVariables['LLMP_API_KEY'] = $dummyKey
    if ($NoCredential) { $start.EnvironmentVariables['LLMP_API_KEY'] = '' }
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
        if ($Failure) { Assert-True ($process.ExitCode -ne 0) 'Invalid input unexpectedly succeeded' }
        else { Assert-True ($process.ExitCode -eq 0) ('Installer failed: ' + $output) }
        return $output
    } finally { $process.Dispose() }
}
function Verify-Toml([string]$Path, [hashtable]$Extra = @{}) {
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
    $absent = @('profiles')
    foreach ($setting in @('model_catalog_json')) {
        if (-not $Extra.ContainsKey($setting)) { $absent += $setting }
    }
    Write-Utf8 $expectation (@{values = $values; forbidden = @($dummyKey); absent = $absent} | ConvertTo-Json -Depth 10)
    & $Python (Join-Path $PSScriptRoot 'verify_config.py') $Path $expectation
    Assert-True ($LASTEXITCODE -eq 0) 'TOML semantic verification failed'
}
function Test-Case([string]$Name, [scriptblock]$Body) {
    try { & $Body; $script:passed++; Write-Output ('PASS ' + $Name) }
    catch { $script:failures += $Name + ': ' + $_.Exception.Message; Write-Output ('FAIL ' + $script:failures[-1]) }
}

Test-Case 'fresh profile and environment authentication' {
    $homeDirectory = New-Case 'fresh'
    Invoke-Installer $homeDirectory | Out-Null
    Verify-Toml (Get-Profile $homeDirectory)
    Assert-True (-not (Test-Path (Join-Path $homeDirectory 'config.toml'))) 'Installer created base config'
}
Test-Case 'downloaded scriptblock bootstrap supports identical interface' {
    $homeDirectory = New-Case 'scriptblock'
    Invoke-Installer $homeDirectory -Bootstrap | Out-Null
    Verify-Toml (Get-Profile $homeDirectory)
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
Test-Case 'idempotent reruns' {
    $homeDirectory = New-Case 'idempotent'
    Invoke-Installer $homeDirectory | Out-Null
    $profile = Get-Profile $homeDirectory
    $original = [Convert]::ToBase64String([IO.File]::ReadAllBytes($profile))
    $before = @(Get-ChildItem $homeDirectory -File).Count
    Invoke-Installer $homeDirectory | Out-Null
    Assert-True ($original -ceq [Convert]::ToBase64String([IO.File]::ReadAllBytes($profile))) 'Rerun changed bytes'
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
    Assert-True ($baseBytes -ceq [Convert]::ToBase64String([IO.File]::ReadAllBytes($base))) 'Base config changed'
    Verify-Toml (Get-Profile $homeDirectory)
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
    Invoke-Installer $homeDirectory @{ContextWindow = 500000; AutoCompactTokenLimit = 450000} | Out-Null
    $backups = @(Get-ChildItem $homeDirectory -File | Where-Object { $_.Name -like 'llmp.config.toml.backup-*' })
    Assert-True ($backups.Count -eq 1) 'Expected one changed-profile backup'
    Assert-True ($originalBytes -ceq [Convert]::ToBase64String([IO.File]::ReadAllBytes($backups[0].FullName))) 'Backup differs from original bytes'
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
    Assert-True (@(Get-ChildItem $homeDirectory -File).Count -eq 1) 'Refusal created backup'
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
    Assert-True ([IO.File]::ReadAllText($base) -ceq $original) 'Tier reset changed base config'
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
    Verify-Toml (Get-Profile $homeDirectory) @{service_tier = 'fast'}
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
Test-Case 'native catalog is preserved and referenced at root with exact model tier' {
    $catalog = New-Catalog 'native' @(@{slug = 'fixture-model'; service_tiers = @(@{id = 'fast'})})
    $original = [IO.File]::ReadAllText($catalog)
    $homeDirectory = New-Case 'catalog-valid'
    Invoke-Installer $homeDirectory @{ModelCatalogPath = $catalog; ServiceTier = 'fast'} | Out-Null
    Verify-Toml (Get-Profile $homeDirectory) @{model_catalog_json = $catalog; service_tier = 'fast'}
    Assert-True ([IO.File]::ReadAllText($catalog) -ceq $original) 'Catalog source changed'
    Invoke-Installer $homeDirectory @{ModelCatalogPath = $catalog; ServiceTier = 'fast'} | Out-Null
    Assert-True (@(Get-ChildItem $homeDirectory -File).Count -eq 1) 'Catalog profile rerun was not idempotent'
}
Test-Case 'legacy catalog speed tiers are supported' {
    $catalog = New-Catalog 'legacy' @(@{slug = 'fixture-model'; additional_speed_tiers = @('fast')})
    $homeDirectory = New-Case 'catalog-legacy'
    Invoke-Installer $homeDirectory @{ModelCatalogPath = $catalog; ServiceTier = 'fast'} | Out-Null
    Verify-Toml (Get-Profile $homeDirectory) @{model_catalog_json = $catalog; service_tier = 'fast'}
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
Write-Output ('Results: ' + $script:passed + ' passed; ' + $script:failures.Count + ' failed. Artifacts: ' + $root)
if ($script:failures.Count -gt 0) { exit 1 }
