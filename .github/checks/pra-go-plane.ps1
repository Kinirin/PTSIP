# Check: pra-go-plane; migrated from pra-registry-validation
$ErrorActionPreference = 'Stop'

# Verify registry-first PRA Go plane
go -C developer/automation vet -tags grammar_subset,grammar_subset_python ./...
if ($LASTEXITCODE -ne 0) {
  throw "Go PRA registry vet failed."
}
go -C developer/automation test -tags grammar_subset,grammar_subset_python ./...
if ($LASTEXITCODE -ne 0) {
  throw "Go PRA registry tests failed."
}
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-responsibility validate --analysis-id PRA-0014
if ($LASTEXITCODE -ne 0) {
  throw "Go PRA analysis-id resolution failed."
}
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-responsibility validate --subject-type POLICY_ANALYSIS --subject-id POLICY_ANALYSIS --analysis-kind REGISTRY_FIRST_DISCOVERY
if ($LASTEXITCODE -ne 0) {
  throw "Go PRA subject-key resolution failed."
}
if ($LASTEXITCODE -ne 0) { throw 'pra-go-plane failed' }
