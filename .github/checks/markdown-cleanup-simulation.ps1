# Check: markdown-cleanup-simulation; migrated from static-validation
$ErrorActionPreference = 'Stop'

# Run MPD-0012 Markdown cleanup simulation
New-Item -ItemType Directory -Force -Path ".artifacts" | Out-Null
$outputPath = ".artifacts/markdown-cleanup-simulation.json"

go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev markdown-cleanup simulate --output $outputPath
if ($LASTEXITCODE -ne 0) {
  throw "Markdown cleanup simulation execution failed."
}

$result = Get-Content -LiteralPath $outputPath -Raw | ConvertFrom-Json
$status = [string]$result.status
$removeCount = @($result.plan.remove).Count
$blockedCount = @($result.plan.blocked).Count
$unresolvedCount = @($result.plan.unresolved).Count

"cleanup_status=$status" >> $env:GITHUB_OUTPUT
"remove_count=$removeCount" >> $env:GITHUB_OUTPUT
"blocked_count=$blockedCount" >> $env:GITHUB_OUTPUT
"unresolved_count=$unresolvedCount" >> $env:GITHUB_OUTPUT

Write-Host "Markdown cleanup simulation status: $status"
Write-Host "Remove candidates: $removeCount"
Write-Host "Blocked targets: $blockedCount"
Write-Host "Unresolved targets: $unresolvedCount"

@(
  "## MPD-0012 Markdown cleanup simulation"
  ""
  "- Source SHA: $env:GITHUB_SHA"
  "- Simulation status: **$status**"
  "- Remove candidates: $removeCount"
  "- Blocked targets: $blockedCount"
  "- Unresolved targets: $unresolvedCount"
  "- Mutation performed: **false**"
) | Add-Content -LiteralPath $env:GITHUB_STEP_SUMMARY -Encoding UTF8
if ($LASTEXITCODE -ne 0) { throw 'markdown-cleanup-simulation failed' }
