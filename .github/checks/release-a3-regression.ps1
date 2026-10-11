# Check: release-a3-regression; migrated from test-build
$ErrorActionPreference = 'Stop'

# Run 0.3.8a3 Context Plane emergency regression
python -m pytest -q src/tests/ptsip/validation/test_local_profile_catalog.py
if ($LASTEXITCODE -ne 0) {
  throw "Canonical .ptsip local profile catalog regression failed."
}
python -m pytest -q src/tests/ptsip/repository/test_self_profile.py::test_repository_uses_canonical_ptsip_namespace_and_json_indexes
if ($LASTEXITCODE -ne 0) {
  throw "Canonical .ptsip repository namespace regression failed."
}
python -m pytest -q src/tests/ptsip/identity/test_tool_identity.py src/tests/ptsip/cli/test_runtime_surface.py
if ($LASTEXITCODE -ne 0) {
  throw "0.3.8a3 Tool/runtime identity regression failed."
}
python -m pytest -q src/tests/ptsip/test_release_readiness_030.py
if ($LASTEXITCODE -ne 0) {
  throw "0.3.8a3 release-readiness contract regression failed."
}
python -m pytest -q src/tests/ptsip/test_context_plane.py
if ($LASTEXITCODE -ne 0) {
  throw "0.3.8a3 Context Plane migration regression failed."
}
ptsip context check . --json
if ($LASTEXITCODE -ne 0) {
  throw "0.3.8a3 repository Context Plane release gate failed."
}
ptsip --version
ptsip spec
ptsip conform --help
if ($LASTEXITCODE -ne 0) { throw 'release-a3-regression failed' }
