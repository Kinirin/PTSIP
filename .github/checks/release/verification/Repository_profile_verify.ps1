[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
$releaseRepository = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../../../..'))
Push-Location $releaseRepository
try {
    foreach ($arguments in @(@('--version'), @('spec'), @('validate','.','--profile','.ptsip/profiles/main.ptsip.yaml','--json'), @('clarify','.','--profile','.ptsip/profiles/main.ptsip.yaml','--json'), @('gate','.','--profile','.ptsip/profiles/main.ptsip.yaml','--coordination','local','--json'), @('conform','--help'))) {
        & ptsip @arguments
        if ($LASTEXITCODE -ne 0) { throw "Repository self-profile verification failed: $arguments" }
    }

} finally { Pop-Location }
