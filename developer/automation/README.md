# Retiring developer automation

Current developer validation enters through `.ptsip/engine/developer/index.json`.
Use the developer module and its domain indexes for current policy, schema,
responsibility-authority, release-documentation and branch checks:

```powershell
go -C .ptsip/engine/developer run . --validator developer-layout-check --root ../../..
go -C .ptsip/engine/developer run . --validator branch-check --root ../../..
go -C .ptsip/engine/developer run ./Distribution/branch/cmd/ptsip-branch branch-control commands --repository ../../..
```

The examples above run from the repository root. Branch implementation and tests
have moved to `.ptsip/engine/developer/Distribution/branch/`. The existing
`ptsip-dev branch-control` and `branch-guard` commands are compatibility adapters
that call that module directly, without the retired command catalog or policy
resolver. Request-shape validation does not grant branch creation authority.

This directory remains a migration source for the older Planning, Agent and
policy implementations. Its retired policy validator/resolver commands do not
provide current developer-policy validation or authority. Their source presence
must not be treated as completed migration or operational approval. New files
belong in the successor developer engine, following its domain boundaries.

The existing `pp pre-commit` compatibility command remains the commit hook's PP
content/transition check. It is independently admitted and uses isolated PP data
schemas; this branch migration neither disables the hook nor converts it into a
current developer-policy validator.
