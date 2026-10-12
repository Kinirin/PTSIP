[CmdletBinding()]
param(
<<<<<<< Updated upstream
    [ValidateSet('Plan','Build','Verify','Publish','Package','Dispatch','SyncVersions')][string]$Action = 'Plan',
=======
    [ValidateSet('Plan','Build','Verify','Publish','Package','Dispatch','VerifyRepository')][string]$Action = 'Plan',
>>>>>>> Stashed changes
    [string]$Targets = 'all', [string]$DisabledTargets = '', [string]$Target = '',
    [string]$PackageBase = '', [string]$Platform = '', [string]$ReleaseTag = '',
    [string]$GitHubRepository = $(if ($env:GITHUB_REPOSITORY) { $env:GITHUB_REPOSITORY } else { 'Kinirin/PTSIP' }),
    [string]$SelectionFile = '.artifacts/release/selection.json',
    [string]$OutputDirectory = '', [string]$InputDirectory = '.artifacts/release/components',
    [ValidateSet('verify','publish')][string]$ReleasePhase = 'verify',
    [switch]$Offline, [switch]$PublishPackage
)
$ErrorActionPreference = 'Stop'
$repo = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../../..'))
$native = Join-Path $repo '.github/checks/release'
$argsNative = @('-C', $native, 'run', '-tags', 'grammar_subset,grammar_subset_python', './cmd/ptsip-release')
switch ($Action) {
<<<<<<< Updated upstream
    'SyncVersions' {
        $argsNative += @('sync-versions','--root',$repo,'--targets',$Targets)
=======
    'VerifyRepository' {
        & (Join-Path $PSScriptRoot 'verification/Repository_regression_verify.ps1')
        & (Join-Path $PSScriptRoot 'verification/Repository_profile_verify.ps1')
        return
>>>>>>> Stashed changes
    }
    'Dispatch' {
        $dispatch = @('workflow','run','tooling-release.yml','--ref','main','-f',"phase=$ReleasePhase",'-f',"targets=$Targets",'-f',"disabled_targets=$DisabledTargets")
        if ($PackageBase) { $dispatch += @('-f',"package_base=$PackageBase") }
        & gh @dispatch
        if ($LASTEXITCODE -ne 0) { throw 'Independent release dispatch failed.' }
        return
    }
    'Plan' {
        $argsNative += @('plan','--root',$repo,'--targets',$Targets,'--selection',$SelectionFile,'--github-repository',$GitHubRepository)
        if ($DisabledTargets) { $argsNative += @('--disabled',$DisabledTargets) }
        if ($PackageBase) { $argsNative += @('--package-base',$PackageBase) }
        if ($ReleaseTag) { $argsNative += @('--release-tag',$ReleaseTag) }
        if ($Offline) { $argsNative += '--offline' }
    }
    'Build' {
        if (-not $Target) {
            $selectionText = (& go -C $native run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-release selection --root $repo --selection $SelectionFile | Out-String)
        if ($LASTEXITCODE -ne 0) { throw 'Selected release binding is stale or modified.' }
        $selection = $selectionText | ConvertFrom-Json
            $selected = @($selection.components | Where-Object { $_.state -eq 'BUILD' })
            if ($Targets -ne 'all') { $names = $Targets.Split(','); $selected = @($selected | Where-Object { $_.id -in $names }) }
            $selected | ForEach-Object -Parallel {
                $component = $_
                $releaseRoot = $using:native
                $output = if ($component.id -eq 'tool') { 'dist' } else { ".artifacts/release/build/$($component.id)/native" }
                & (Join-Path $releaseRoot $component.script) -SelectionFile $using:SelectionFile -Platform $(if($component.kind -eq 'ENGINE'){$using:Platform}else{'any'}) -OutputDirectory $output
                if ($LASTEXITCODE -ne 0) { throw "Parallel release build failed: $($component.id)" }
            } -ThrottleLimit 9
            return
        }
        $selectionText = (& go -C $native run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-release selection --root $repo --selection $SelectionFile | Out-String)
        if ($LASTEXITCODE -ne 0) { throw 'Selected release binding is stale or modified.' }
        $selection = $selectionText | ConvertFrom-Json
        $component = @($selection.components | Where-Object { $_.id -eq $Target })
        if ($component.Count -ne 1) { throw "Unregistered release target: $Target" }
        if ($component[0].state -ne 'BUILD') { throw "Target is switched off or unchanged: $Target" }
        $script = Join-Path $PSScriptRoot $component[0].script
        if (-not (Test-Path -LiteralPath $script -PathType Leaf)) { throw "Registered target script missing: $script" }
        if (-not $OutputDirectory) { $OutputDirectory = if ($Target -eq 'tool') { 'dist' } else { ".artifacts/release/build/$Target/$Platform" } }
        & $script -SelectionFile $SelectionFile -Platform $Platform -OutputDirectory $OutputDirectory
        if ($LASTEXITCODE -ne 0) { throw "Target build failed: $Target" }
        return
    }
    'Verify' { $argsNative += @('verify','--root',$repo,'--selection',$SelectionFile,'--target',$Target,'--output',$OutputDirectory,'--github-repository',$GitHubRepository) }
    'Publish' { $argsNative += @('publish','--root',$repo,'--selection',$SelectionFile,'--target',$Target,'--output',$OutputDirectory,'--github-repository',$GitHubRepository) }
    'Package' {
        & (Join-Path $PSScriptRoot 'Package_release.ps1') -SelectionFile $SelectionFile -InputDirectory $InputDirectory -OutputDirectory $OutputDirectory -Publish:$PublishPackage
        if ($LASTEXITCODE -ne 0) { throw 'Package assembly failed.' }
        return
    }
}
& go @argsNative
if ($LASTEXITCODE -ne 0) { throw "Native release $Action failed." }
