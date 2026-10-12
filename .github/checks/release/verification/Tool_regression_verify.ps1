[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
$releaseRepository = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '../../../..'))
Push-Location $releaseRepository
try {
    foreach ($testRoot in @('src/tests/agent_contracts', 'src/tests/ptsip', 'src/tests/vpms')) {
        & python -m pytest -q $testRoot
        if ($LASTEXITCODE -ne 0) { throw "Tool shipping regression failed: $testRoot" }
    }
    & go -C developer/tests test -tags grammar_subset,grammar_subset_python ./release
    if ($LASTEXITCODE -ne 0) { throw 'Tool release contract/identity/gate/distribution Go regression failed.' }

} finally { Pop-Location }
