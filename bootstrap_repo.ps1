# Requires: GitHub CLI (gh), authenticated with `gh auth login`
# Purpose: one-time creation and publication of the canonical PTSIP repository.
$ErrorActionPreference = 'Stop'

$Repo = 'Kinirin/PTSIP'

if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
    throw 'GitHub CLI (gh) is not installed or not on PATH.'
}

$null = gh auth status 2>&1
if ($LASTEXITCODE -ne 0) {
    throw 'GitHub CLI is not authenticated. Run: gh auth login'
}

if (-not (Test-Path '.git')) {
    git init -b main
}

git add .
git diff --cached --quiet
if ($LASTEXITCODE -eq 1) {
    git commit -m 'docs: establish PTSIP draft specification'
    if ($LASTEXITCODE -ne 0) { throw 'Initial git commit failed.' }
}
elseif ($LASTEXITCODE -ne 0) {
    throw 'Unable to inspect staged git changes.'
}

gh repo create $Repo --public --source . --remote origin --push --description 'PTSIP — Product–Toolchain SDK Isolation Policy specification and reference tooling.'
