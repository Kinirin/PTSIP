# Check: readme-translation. Registry-driven; no fixed target language or Python script.
$ErrorActionPreference = 'Stop'

go -C developer/automation test ./cmd/readme-translate
if ($LASTEXITCODE -ne 0) {
  throw 'Go README translator regression tests failed.'
}

$targets = @(go -C developer/automation run ./cmd/readme-translate --action list-targets)
if ($LASTEXITCODE -ne 0 -or $targets.Count -lt 1) {
  throw 'README translation registry has no valid targets.'
}

foreach ($target in $targets) {
  if (-not (Test-Path -LiteralPath $target -PathType Leaf)) {
    throw "Registered README translation is missing: $target"
  }
}

go -C developer/automation run ./cmd/readme-translate --action validate
if ($LASTEXITCODE -ne 0) {
  throw 'README translation structure, inline code or links are invalid.'
}
