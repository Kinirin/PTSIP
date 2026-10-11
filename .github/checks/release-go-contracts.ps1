# Check: release-go-contracts; migrated from test-build
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

# Verify Go contracts during full verification
go -C developer/tests test -tags grammar_subset,grammar_subset_python -count=1 -p=1 -v ./...
if ($LASTEXITCODE -ne 0) {
  throw "Root Family Go verification failed."
}
if ($LASTEXITCODE -ne 0) { throw 'release-go-contracts failed' }
