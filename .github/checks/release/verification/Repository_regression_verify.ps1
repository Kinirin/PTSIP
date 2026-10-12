[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
$releaseRepository = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../../../..'))
Push-Location $releaseRepository
try {
    # Explicit repository-wide verification provider; never an implicit engine release dependency.
    & go -C developer/tests test -tags grammar_subset,grammar_subset_python -count=1 -p=1 -v ./...
    if ($LASTEXITCODE -ne 0) { throw 'Repository-wide Go regression failed.' }

} finally { Pop-Location }
