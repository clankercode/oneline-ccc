# Loaded by run.ps1 with its isolated process/native/TOML helpers.
$keepInputs = @{BaseUrl = $null; Model = $null; ContextWindow = $null; AutoCompactTokenLimit = $null}
function Get-SetupFileBytes([string]$Directory) {
    $result = @{}
    foreach ($file in @(Get-ChildItem $Directory -File)) { $result[$file.Name] = [Convert]::ToBase64String([IO.File]::ReadAllBytes($file.FullName)) }
    return $result
}
function Assert-SetupFileBytes([string]$Directory, [hashtable]$Expected) {
    $actual = Get-SetupFileBytes $Directory
    Assert-True ($actual.Count -eq $Expected.Count) 'Rerun created extra files/backups'
    foreach ($name in $Expected.Keys) { Assert-True ($actual[$name] -ceq $Expected[$name]) ('Rerun changed ' + $name) }
}
Test-Case 'rerun defaults: omitted inputs keep managed LLMP values and bytes' {
    $directory = New-Case 'reuse-managed'
    Invoke-Installer $directory @{ServiceTier = 'ultrafast'; SupportedServiceTiers = @('ultrafast')} | Out-Null
    $before = Get-SetupFileBytes $directory
    Invoke-Installer $directory $keepInputs | Out-Null
    Assert-SetupFileBytes $directory $before
    Verify-Toml (Join-Path $directory 'config.toml') @{service_tier = 'ultrafast'}
}
Test-Case 'rerun defaults: Enter retains URL key model tier context and custom compaction' {
    $directory = New-Case 'reuse-enter'
    Invoke-Installer $directory @{ServiceTier = 'fast'; SupportedServiceTiers = @('fast')} | Out-Null
    $before = Get-SetupFileBytes $directory
    $options = @{}; foreach ($name in $keepInputs.Keys) { $options[$name] = $null }; $options.NonInteractive = $false
    $answers = @('', '', '', '', '')
    if ($env:OS -eq 'Windows_NT') { $answers += 'n' }
    $output = Invoke-Installer $directory $options -PromptAnswers $answers
    Assert-True ($output.Contains('https://fixture.invalid/v1')) 'URL default was not shown'
    Assert-True ($output.Contains('fixture-model')) 'Model default was not shown'
    Assert-True ($output.Contains('353000')) 'Context default was not shown'
    Assert-True ($output.Contains('priority')) 'Tier default was not shown'
    Assert-SetupFileBytes $directory $before
}
Test-Case 'rerun defaults: active arbitrary LLMP main overrides stale named defaults' {
    $directory = New-Case 'reuse-active-main'
    Invoke-Installer $directory | Out-Null
    $main = Join-Path $directory 'config.toml'
    Write-Utf8 $main @'
# retain this unrelated setting
model_provider = "oneline_llmp"
model = "current-fixture-model"
model_context_window = 272000
model_auto_compact_token_limit = 220000
service_tier = "priority"
[model_providers.oneline_llmp]
name = "LLMP (oneline-ccc)"
base_url = "https://current.fixture.invalid/v1"
wire_api = "responses"
env_key = "LLMP_API_KEY"
requires_openai_auth = false
[features]
web_search_request = false
'@
    Invoke-Installer $directory $keepInputs | Out-Null
    Verify-Toml $main @{model = 'current-fixture-model'; service_tier = 'priority'; model_context_window = 272000; model_auto_compact_token_limit = 220000; 'model_providers.oneline_llmp.base_url' = 'https://current.fixture.invalid/v1'; 'features.web_search_request' = $false}
    Assert-True ([IO.File]::ReadAllText($main).Contains('# retain this unrelated setting')) 'Rerun dropped unrelated comment'
}
Test-Case 'rerun defaults: another active provider cannot supply LLMP defaults' {
    $directory = New-Case 'reuse-other-provider'
    Invoke-Installer $directory | Out-Null
    $main = Join-Path $directory 'config.toml'
    Write-Utf8 $main @'
model_provider = "other"
model = "other-model"
model_context_window = 128000
service_tier = "flex"
[model_providers.other]
name = "Other"
base_url = "https://other.fixture.invalid/v1"
wire_api = "responses"
'@
    Invoke-Installer $directory $keepInputs | Out-Null
    Verify-Toml $main @{'model_providers.other.base_url' = 'https://other.fixture.invalid/v1'}
}
Test-Case 'rerun defaults: explicit changes win and changed context recomputes compaction' {
    $directory = New-Case 'reuse-overrides'
    Invoke-Installer $directory @{ServiceTier = 'ultrafast'; SupportedServiceTiers = @('ultrafast')} | Out-Null
    Invoke-Installer $directory @{BaseUrl = 'https://changed.fixture.invalid/v1'; Model = 'changed-model'; ContextWindow = 500000; AutoCompactTokenLimit = $null; ServiceTier = 'default'} | Out-Null
    Verify-Toml (Join-Path $directory 'config.toml') @{model = 'changed-model'; model_context_window = 500000; model_auto_compact_token_limit = 450000; 'model_providers.oneline_llmp.base_url' = 'https://changed.fixture.invalid/v1'}
}
Test-Case 'rerun defaults: saved User key works when process key is absent and stays hidden' {
    $directory = New-Case 'reuse-user-key'
    $output = Invoke-Installer $directory -NoCredential -UserKeyFixture 'fixture-saved-user-key'
    Verify-Toml (Join-Path $directory 'config.toml')
}
Test-Case 'rerun defaults: DryRun reuses recognized settings without writing files' {
    $directory = New-Case 'reuse-dry-run'
    Invoke-Installer $directory | Out-Null
    $before = Get-SetupFileBytes $directory
    $options = @{}; foreach ($name in $keepInputs.Keys) { $options[$name] = $null }; $options.DryRun = $true
    $output = Invoke-Installer $directory $options -NoCredential
    Assert-True ($output.Contains("base_url = 'https://fixture.invalid/v1'")) 'DryRun omitted URL default'
    Assert-SetupFileBytes $directory $before
}

Test-Case 'rerun defaults: a smaller context replaces an oversized old compaction limit' {
    $directory = New-Case 'reuse-smaller-context'
    Invoke-Installer $directory | Out-Null
    Invoke-Installer $directory @{ContextWindow = 172000; AutoCompactTokenLimit = $null} | Out-Null
    Verify-Toml (Join-Path $directory 'config.toml') @{model_context_window = 172000; model_auto_compact_token_limit = 154800}
}
Test-Case 'rerun defaults: changing model does not inherit the old request tier' {
    $directory = New-Case 'reuse-changed-model'
    Invoke-Installer $directory @{ServiceTier = 'ultrafast'; SupportedServiceTiers = @('ultrafast')} | Out-Null
    Invoke-Installer $directory @{Model = 'different-model'} | Out-Null
    Verify-Toml (Join-Path $directory 'config.toml') @{model = 'different-model'}
}
Test-Case 'rerun defaults: process credential takes precedence over saved User credential' {
    $directory = New-Case 'reuse-process-key'
    Invoke-Installer $directory -UserKeyFixture 'fixture-saved-user-key' | Out-Null
    Verify-Toml (Join-Path $directory 'config.toml')
}
Test-Case 'rerun defaults: Enter without any saved key refuses before configuration writes' {
    $directory = New-Case 'reuse-empty-key'
    $output = Invoke-Installer $directory @{NonInteractive = $false} -NoCredential -PromptAnswers @('') -Failure
    Assert-True ($output.Contains('API key must not be empty')) 'Missing saved key did not receive the required-key error'
    Assert-True (@(Get-ChildItem $directory -Force).Count -eq 0) 'Missing key changed configuration'
}
