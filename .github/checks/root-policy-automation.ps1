# Check: root-policy-automation; migrated from static-validation
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

# Verify direct Root policy Go automation
go -C developer/automation vet -tags grammar_subset,grammar_subset_python ./...
if ($LASTEXITCODE -ne 0) {
  throw "Go Developer automation vet failed."
}
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev policy-resolver validate
if ($LASTEXITCODE -ne 0) {
  throw "Direct Root policy Go resolver validation failed."
}
go -C developer/tests/rootfamily test -count=1 -v -run '^TestGo(DirectRootResolver|ResolverNeeds|ResolverRejects|CutoverReports|StateAndPlanning|RootIdentityInspection)' ./...
if ($LASTEXITCODE -ne 0) {
  throw "Go automation boundary verification failed."
}
if ($LASTEXITCODE -ne 0) { throw 'root-policy-automation failed' }
