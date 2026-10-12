[CmdletBinding()]
param([string]$SelectionFile = '.artifacts/release/selection.json', [string]$Platform = 'any', [string]$OutputDirectory)
$ErrorActionPreference = 'Stop'
$repo = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../../..'))
& go -C (Join-Path $repo '.github/checks/release') run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-release build --root $repo --selection $SelectionFile --target 'tool' --platform $Platform --output $OutputDirectory
if ($LASTEXITCODE -ne 0) { throw 'tool release build failed.' }

Push-Location $repo
try {
    & go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev pp release --sha HEAD
    if ($LASTEXITCODE -ne 0) { throw 'Exact-source Profile transition verification failed.' }
    & python .github/scripts/verify_release_contract.py
    if ($LASTEXITCODE -ne 0) { throw 'Specification release contract failed.' }
    & (Join-Path $PSScriptRoot 'Tool_artifact_verify.ps1') -DistPath $OutputDirectory
    & python .github/scripts/verify_distribution_contracts.py
    if ($LASTEXITCODE -ne 0) { throw 'Wheel/sdist registry projection verification failed.' }
} finally { Pop-Location }
