# Check: registry-consistency; migrated from static-validation
$ErrorActionPreference = 'Stop'

# Validate v3 Test Mode registry through Go
go -C developer/automation run ./cmd/test-modes
if ($LASTEXITCODE -ne 0) { throw 'Go Test Mode Registry v3 validation failed.' }
if ($LASTEXITCODE -ne 0) { throw 'registry-consistency failed' }
