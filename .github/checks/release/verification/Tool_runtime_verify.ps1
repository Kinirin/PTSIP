[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
$releaseRepository = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../../../..'))
Push-Location $releaseRepository
try {
    foreach ($arguments in @(@('--version'), @('spec'), @('conform','--help'), @('context','--help'), @('context','status','.','--json'), @('context','check','.','--json'))) {
        & ptsip @arguments
        if ($LASTEXITCODE -ne 0) { throw "Installed Tool runtime verification failed: $arguments" }
    }
    & python -c "from vpms import VerificationPurpose; assert VerificationPurpose.PRODUCT.value == 'PRODUCT'; assert VerificationPurpose.TOOLCHAIN.value == 'TOOLCHAIN'"
    if ($LASTEXITCODE -ne 0) { throw 'Existing VPMS public enum compatibility failed.' }

} finally { Pop-Location }
