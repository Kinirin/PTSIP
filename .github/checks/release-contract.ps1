# Check: release-contract; migrated from test-build
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

# Verify Tool, PP, and Specification release contract
python .github/scripts/verify_release_contract.py
if ($LASTEXITCODE -ne 0) { throw 'release-contract failed' }
