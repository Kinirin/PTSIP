# Check: release-full-regression; migrated from test-build
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

# Run complete shipping and release regression
python -m pytest -q src/tests/agent_contracts
if ($LASTEXITCODE -ne 0) {
  throw "Agent contract test corpus failed."
}
python -m pytest -q src/tests/ptsip
if ($LASTEXITCODE -ne 0) {
  throw "PTSIP product and release test corpus failed."
}
python -m pytest -q src/tests/vpms
if ($LASTEXITCODE -ne 0) {
  throw "VPMS test corpus failed."
}
if ($LASTEXITCODE -ne 0) { throw 'release-full-regression failed' }
