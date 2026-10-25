# PTSIP Tool T.0.3.8.b6 — source preparation (not published)

Expected PyPI prerelease identity: `0.3.8b6`. The canonical Package/Tool composition
has **not** been migrated or verified for this candidate. The current `main`
`pyproject.toml` still carries a separate older source composition, while
published `0.3.8b5` was built from a different release line. Do not infer
b6 readiness or publication from this note.

## Prepared on main

- Canonical six Compile-Sealed RBAC schema identities:
  `.ptsip/contracts/schemas/compile-sealed-rbac/*.schema.json`.
- Distribution resources:
  `src/ptsip/specdata/compile-sealed-rbac/*.schema.json`.
- Canonical central RBAC registry field requirement contract:
  `registry/compile-sealed-rbac-registry-fields.json`, mirrored in
  `src/ptsip/specdata/compile-sealed-rbac-registry-fields.json`.
- `setup.py`, `pyproject.toml`, and `MANIFEST.in` package the resources
  and check source/embedded file equality.
- `src/ptsip/specdata/ptsip-registry.yaml` matches its canonical
  `registry/ptsip-registry.yaml` source.

All six schema files are intentionally **fail-closed scaffolds**. Their
required/optional domain fields, registration grammar, Binding legitimacy,
Seal integrity, and Activation semantics have not yet been approved; no
RBAC instance is valid against the scaffolds. The Rust validation engine's
RBAC preflight is unregistered internal code and does not grant authority.

## Release blockers / explicit scope limits

- Approve complete field/lookup/serialization contracts before enabling
  `pp.2.00`, registering `validation.rbac-preflight`, or claiming RBAC support.
- Do not mark `pp.2.00` as current in `project-profile-contracts.yaml`
  until its full schema and compatible runtime/profile migration exist.
- Existing `src/ptsip/repository/schemas/engine-state.schema.json` and
  `repository-index.schema.json` do not own these RBAC contracts.
- Resolve the b5-to-b6 package and seven native-engine composition, then
  run source, wheel, sdist, installation and workflow verification against
  the exact release source. Publication is a separate verified action.

**Status: SOURCE_ASSETS_STAGED; RELEASE_NOT_VERIFIED; NOT_PUBLISHED.**
