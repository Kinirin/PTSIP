[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
$releaseRepository = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../../../..'))
Push-Location $releaseRepository
try {
    & python .github/scripts/verify_release_contract.py
    if ($LASTEXITCODE -ne 0) { throw 'Tool/Profile/Specification release contract verification failed.' }

} finally { Pop-Location }
