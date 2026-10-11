# Check: policy-lifecycle; migrated from static-validation
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

# Verify policy lifecycle automation
go -C developer/tests test -tags grammar_subset,grammar_subset_python -count=1 -v ./policy/... ./planning/... ./repository/... ./workflows/...
if ($LASTEXITCODE -ne 0) {
  throw "Native Go policy and planning regression failed."
}
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-validator validate
if ($LASTEXITCODE -ne 0) {
  throw "Developer policy validation failed."
}

go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-plan-consistency verify
if ($LASTEXITCODE -ne 0) {
  throw "Policy-Plan consistency verification failed."
}

$rootPolicy = (& go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-lifecycle inspect MPD-GOV-0001) | ConvertFrom-Json
if ($LASTEXITCODE -ne 0 -or -not $rootPolicy.operationally_resolvable) {
  throw "Canonical Root governance policy is not operationally resolvable."
}
if ($LASTEXITCODE -ne 0) { throw 'policy-lifecycle failed' }
