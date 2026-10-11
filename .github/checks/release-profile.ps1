# Check: release-profile; migrated from test-build
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

# Verify Tool, Specification, and repository profile
ptsip --version
ptsip spec
ptsip validate . --profile .ptsip/profiles/main.ptsip.yaml --json
ptsip clarify . --profile .ptsip/profiles/main.ptsip.yaml --json
ptsip gate . --profile .ptsip/profiles/main.ptsip.yaml --coordination local --json
ptsip conform --help
if ($LASTEXITCODE -ne 0) { throw 'release-profile failed' }
