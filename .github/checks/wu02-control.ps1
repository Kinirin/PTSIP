# Check: wu02-control; migrated from static-validation
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

# Resolve WU-02 lane validation applicability
$controlContext = (& go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev wu02 control-context) | ConvertFrom-Json
$controlBranch = $controlContext.control_branch
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($controlBranch)) {
  throw "Unable to resolve the canonical WU-02 control branch."
}

$laneRoot = "developer/planning/0.4.0/WU-02"
$laneMaterialized = Test-Path -LiteralPath $laneRoot
$required = (($env:PTSIP_EXECUTION_BRANCH -eq $controlBranch) -and $laneMaterialized).ToString().ToLowerInvariant()
"required=$required" >> $env:GITHUB_OUTPUT
Write-Host "WU-02 lane validation required: $required (execution=$env:PTSIP_EXECUTION_BRANCH control=$controlBranch materialized=$laneMaterialized)"

# Validate WU-02 control branch from explicit workflow context
if ($required -eq 'true') {
  go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev wu02 validate --execution-branch "$env:PTSIP_EXECUTION_BRANCH"
  if ($LASTEXITCODE -ne 0) {
    throw "WU-02 control branch validation failed."
  }
}
if ($LASTEXITCODE -ne 0) { throw 'wu02-control failed' }
