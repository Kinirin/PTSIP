[CmdletBinding()]
param([string]$SelectionFile = '.artifacts/release/selection.json', [string]$InputDirectory = '.artifacts/release/components', [string]$OutputDirectory = '.artifacts/release/package', [switch]$Publish)
$ErrorActionPreference = 'Stop'
$repo = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../../..'))
if (-not $OutputDirectory) { $OutputDirectory = '.artifacts/release/package' }
$inputs = [IO.Path]::GetFullPath((Join-Path $repo $InputDirectory))
if (-not $inputs.StartsWith($repo + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw 'Package input must stay inside the repository.' }
if (-not (Test-Path -LiteralPath $inputs)) { New-Item -ItemType Directory -Path $inputs | Out-Null }
# GitHub artifact names preserve target/platform boundaries. Join only their
# verified files into component directories; the native verifier checks hashes.
foreach ($artifact in @(Get-ChildItem -LiteralPath $inputs -Directory)) {
    if ($artifact.Name -notmatch '^release-(tool|profile|context|storage|validation|discovery|planning|execution|projection)-') { continue }
    $target = $Matches[1]
    $destination = Join-Path $inputs $target
    New-Item -ItemType Directory -Force -Path $destination | Out-Null
    foreach ($file in @(Get-ChildItem -LiteralPath $artifact.FullName -File)) {
        $targetPath = Join-Path $destination $file.Name
        if (Test-Path -LiteralPath $targetPath) { throw "Duplicate component artifact: $targetPath" }
        Copy-Item -LiteralPath $file.FullName -Destination $targetPath
    }
}
$argsNative = @('-C',(Join-Path $repo '.github/checks/release'),'run','-tags','grammar_subset,grammar_subset_python','./cmd/ptsip-release','assemble','--root',$repo,'--selection',$SelectionFile,'--input',$InputDirectory,'--output',$OutputDirectory)
if ($Publish) { $argsNative += '--publish' }
& go @argsNative
if ($LASTEXITCODE -ne 0) { throw 'Verified Package assembly failed.' }
