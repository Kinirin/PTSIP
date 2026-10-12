[CmdletBinding()]
param([string]$SelectionFile = '.artifacts/release/selection.json', [string]$Platform = 'any', [string]$OutputDirectory)
$ErrorActionPreference = 'Stop'
$repo = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../../../..'))
& go -C (Join-Path $repo '.github/checks/release') run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-release build --root $repo --selection $SelectionFile --target 'validation' --platform $Platform --output $OutputDirectory
if ($LASTEXITCODE -ne 0) { throw 'validation release build failed.' }
