# Check: release-contract; migrated from test-build
$ErrorActionPreference = 'Stop'

# Verify Tool, PP, and Specification release contract
python .github/scripts/verify_release_contract.py
if ($LASTEXITCODE -ne 0) { throw 'release-contract failed' }
