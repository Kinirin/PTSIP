# Check: readme-translation; migrated from static-validation
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

# Verify README translation registry layout
python -m py_compile .github/scripts/sync_readme_argos.py
if ($LASTEXITCODE -ne 0) {
  throw "README translation script syntax validation failed."
}

$targets = @(
  python .github/scripts/sync_readme_argos.py --registry docs/translated/index.yaml --list-targets
)
if ($LASTEXITCODE -ne 0) {
  throw "README translation registry resolution failed."
}
if ($targets.Count -ne 1 -or $targets[0] -ne "docs/translated/README.ko.md") {
  throw "README translation registry must resolve exactly the Korean translation target."
}
if (-not (Test-Path -LiteralPath "docs/translated/README.ko.md")) {
  throw "Registered Korean README translation is missing."
}
if (Test-Path -LiteralPath "README.ko.md") {
  throw "Legacy root README.ko.md must remain retired."
}

$rootReadme = Get-Content -LiteralPath "README.md" -Raw
if ($rootReadme -notlike '*href="docs/translated/README.ko.md"*') {
  throw "README.md does not link to the registered Korean translation."
}
$localized = Get-Content -LiteralPath "docs/translated/README.ko.md" -Raw
if ($localized -notlike '*href="../../README.md"*') {
  throw "Localized README does not link back to the canonical root README."
}
if ($LASTEXITCODE -ne 0) { throw 'readme-translation failed' }
