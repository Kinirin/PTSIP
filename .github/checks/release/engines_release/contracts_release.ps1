[CmdletBinding()]
param([string]$SelectionFile = '.artifacts/release/selection.json', [Parameter(Mandatory)][string]$Target, [Parameter(Mandatory)][string]$OutputDirectory)
$ErrorActionPreference = 'Stop'
$repo = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../../../..'))
& go -C (Join-Path $repo '.github/checks/release') run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-release contracts --root $repo --selection $SelectionFile --target $Target --output $OutputDirectory
if ($LASTEXITCODE -ne 0) { throw 'Neutral engine contract projection failed.' }
