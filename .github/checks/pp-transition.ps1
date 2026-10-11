# Check: pp-transition; migrated from pp-transition
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

# Resolve introduced commit baseline
$base = $env:PUSH_BEFORE

if (-not $base -or $base -match '^[0]+$') {
  if ($env:REF_TYPE -eq 'tag') {
    $base = (git rev-parse HEAD^).Trim()
  }
  else {
    git fetch origin $env:DEFAULT_BRANCH --no-tags
    if ($LASTEXITCODE -ne 0) {
      throw "Unable to fetch default branch for PP verification baseline."
    }
    $base = (git merge-base HEAD "origin/$env:DEFAULT_BRANCH").Trim()
  }
}
else {
  git merge-base --is-ancestor $base HEAD
  if ($LASTEXITCODE -ne 0) {
    $mergeBase = (git merge-base $base HEAD).Trim()
    if (-not $mergeBase) {
      throw "Unable to resolve PP verification baseline for non-fast-forward push."
    }
    $base = $mergeBase
  }
}

if (-not $base) {
  throw "PP transition verification baseline is empty."
}

"base=$base" >> $env:GITHUB_OUTPUT
Write-Host "PP transition verification base: $base"

# Verify every introduced PP transition commit
$env:PP_BASE = $base
$env:PP_HEAD = $env:GITHUB_SHA
go -C developer/automation run -tags grammar_subset,grammar_subset_python ./cmd/ptsip-dev pp range --base $env:PP_BASE --head $env:PP_HEAD
if ($LASTEXITCODE -ne 0) {
  throw "Remote PP transition verification failed."
}
if ($LASTEXITCODE -ne 0) { throw 'pp-transition failed' }
