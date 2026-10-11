# Check: pra-lifecycle; migrated from pra-registry-validation
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

# Verify registry-first PRA lifecycle plane
go -C developer/tests test ./policy/lifecycle
if ($LASTEXITCODE -ne 0) {
  throw "Native Go policy responsibility lifecycle tests failed."
}
python -m pytest -q src/tests/ptsip/policy_responsibility_analysis/test_mode_registration.py
if ($LASTEXITCODE -ne 0) {
  throw "PRA Test Mode registration regression failed."
}
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-responsibility validate --analysis-id PRA-0014
if ($LASTEXITCODE -ne 0) {
  throw "PRA analysis-id resolution failed."
}
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-responsibility validate --subject-type POLICY_ANALYSIS --subject-id POLICY_ANALYSIS --analysis-kind REGISTRY_FIRST_DISCOVERY
if ($LASTEXITCODE -ne 0) {
  throw "PRA subject-key resolution failed."
}
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-validator validate
if ($LASTEXITCODE -ne 0) {
  throw "Developer policy validation failed after PRA registry migration."
}
if ($LASTEXITCODE -ne 0) { throw 'pra-lifecycle failed' }
