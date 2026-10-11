# Check: markdown-cleanup-regression; migrated from static-validation
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

# Verify Markdown cleanup automation
go -C developer/automation test -tags grammar_subset,grammar_subset_python -count=1 -run '^TestCleanup' ./internal/machine
if ($LASTEXITCODE -ne 0) {
  throw "Markdown cleanup automation tests failed."
}
if ($LASTEXITCODE -ne 0) { throw 'markdown-cleanup-regression failed' }
