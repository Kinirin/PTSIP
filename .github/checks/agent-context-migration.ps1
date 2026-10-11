# Check: agent-context-migration; migrated from static-validation
$ErrorActionPreference = 'Stop'

# Verify bounded machine context migration
New-Item -ItemType Directory -Force -Path ".artifacts" | Out-Null
$log = ".artifacts/agent-context-migration-verification.log"

go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev agent-context-migration verify --stage AUTO 2>&1 |
  Tee-Object -FilePath $log
if ($LASTEXITCODE -ne 0) {
  throw "Agent context migration M5 verification failed."
}

python -m pytest -q `
  src/tests/agent_contracts/test_contract_plane.py `
  src/tests/agent_contracts/test_resolver.py `
  src/tests/ptsip/test_modes/test_agent_context_routing.py 2>&1 |
  Tee-Object -FilePath $log -Append
if ($LASTEXITCODE -ne 0) {
  throw "Agent context migration focused tests failed."
}

go -C developer/automation test -tags grammar_subset,grammar_subset_python -count=1 -run '^TestAgent(Profile|OperationReferences|ContextNative)' ./internal/machine 2>&1 |
  Tee-Object -FilePath $log -Append
if ($LASTEXITCODE -ne 0) {
  throw "Native Agent context migration regression failed."
}
if ($LASTEXITCODE -ne 0) { throw 'agent-context-migration failed' }
