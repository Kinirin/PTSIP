# PTSIP Status

## Current migration state

- Canonical repository: `kwaksinwoo01/PTSIP`
- Maturity: Experimental
- Current Tool/package source version: `0.3.4`
- Previous active Specification baseline on `main`: `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`
- `spec-v0.3.4-draft`: published GitHub Specification **design-release checkpoint**; tag remains unchanged
- Final `0.3.4-draft` coherent normative freeze revision in PR `#26`: `afba3531e23d96c21b7216e49614b839158ca7d5`
- Tool `0.3.4` source binding prepared in PR `#26`: `0.3.4-draft @ afba3531e23d96c21b7216e49614b839158ca7d5`
- Activation/rebind verification: pending PR `#26` merge CI
- Latest verified PyPI publication: `PTSIP==0.3.1`
- Tool `0.3.4` publication: **not published**; no `tool-v0.3.4`, GitHub Tool Release, or PyPI `0.3.4`
- Supported Python metadata: Python 3.11–3.14
- Routine hosted Tool CI: Python 3.14 to conserve GitHub Actions usage
- Tool release namespace: `tool-v*`
- Specification release/design namespace: `spec-v*`
- License: Apache License 2.0

The `0.3.4-draft` freeze becomes the canonical active draft snapshot only when PR `#26` is merged while preserving the freeze commit in history. The Tool binding commit intentionally follows the freeze commit and points backward to it; a Git commit cannot contain a literal self-reference to its own SHA.

The existing `spec-v0.3.4-draft` tag is not moved to the later normative freeze revision. Immutable tag provenance and active mutable-draft snapshot identity are separate concepts.

## Specification 0.3.4-draft activation scope

The migration combines the accepted Explicit Project Adoption design with Distributed Authority Consistency while preserving exactly three architecture classifications:

- `PRODUCT`
- `TOOLCHAIN`
- `NEUTRAL_CONTRACT`

The new durable structured adoption fact set is:

- `classification`
- `purpose`
- `shipped`
- `runtime_required`
- `lifecycle_owner`
- `executable`

Canonical lifecycle owners are `PRODUCT`, `DEVELOPMENT_TOOLING`, and `INDEPENDENT`. `release_owner` and `compatibility_owner` remain separate optional project metadata and are not aliases for canonical lifecycle ownership.

New stable Specification rules are:

- `PTSIP-ADP-001` — lossless explicit project adoption
- `PTSIP-AUT-001` — Decision Authority / Project Profile responsibility separation
- `PTSIP-AUT-002` — stable distributed decision identity
- `PTSIP-AUT-003` — first-valid-resolution-wins with ordered conditional mutation
- `PTSIP-AUT-004` — authority freshness
- `PTSIP-AUT-005` — safe authority/profile reconciliation
- `PTSIP-AUT-006` — fail-closed distributed coordination
- `PTSIP-AUT-007` — global decision state / local projection separation

`PTSIP-AUT-*` applies to implementations claiming distributed coordination capability. It does not create a fourth Consumer Repository architecture plane.

## Coherent migration assets

The immutable freeze `afba3531e23d96c21b7216e49614b839158ca7d5` contains the coherent normative/machine-readable migration state, including:

- `spec/PTSIP-SPEC.md`
- `spec/PTSIP-CONFORMANCE.md`
- `spec/PTSIP-TERMINOLOGY.md`
- `spec/PTSIP-GOVERNANCE.md`
- `schemas/ptsip-profile.schema.json`
- `schemas/ptsip-artifact-evidence.schema.json`
- `schemas/ptsip-agent-classification.schema.json`
- `schemas/ptsip-diagnostic.schema.json`
- `registry/ptsip-registry.yaml`
- `agents/AGENT-CONTRACT.md`
- `adoption/ADOPTION-GUIDE.md`
- `reference/REFERENCE-ARCHITECTURE.md`
- `profiles/example.ptsip.yaml`
- `decisions/ADR-0005-activate-spec-0.3.4-draft.md`
- `src/ptsip/specdata/*` packaged machine-readable contracts
- Tool profile-projection behavior required for lossless structured facts

PR `#26` then adds the Tool binding and regression fixtures that point back to that immutable snapshot.

All five canonical/embedded machine-readable pairs are required to be byte-identical by release-readiness tests:

1. Project Profile schema
2. registry
3. artifact-evidence schema
4. agent-classification schema
5. diagnostic schema

## Tool 0.3.4 distributed-authority implementation

Tool `0.3.4` already completed the distributed-authority behavior in PR `#24` (`555c528593f700a348d8da84545a62ce61291cae`) with the test identity follow-up in PR `#25` (`8cd0ddf16dc9b56f27f694138a37caae1c49bb4f`).

Implemented behavior includes:

- authority freshness even when local Project Profile declaration is complete;
- read-only authority absence lookup without fabricated history;
- first-valid-resolution-wins with non-force Git-ref CAS;
- deterministic missing/equivalent/conflicting reconciliation;
- `AUTHORITY_PROFILE_CONFLICT` without silent overwrite;
- stale repository/profile refusal;
- fail-closed `COORDINATION_UNAVAILABLE` instead of implicit Local DecisionStore fallback;
- global decision state separate from clone-local `LOCAL_PROJECTION` application state;
- action-time synchronization rather than continuous polling.

## Verification history

### Tool 0.3.4 pre-rebind verification

GitHub Actions run `31471025526` on Python `3.14.6` verified the original Tool `0.3.4` source while it was still bound to `0.2.0-draft @ a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`:

- `134 passed`
- Tool identity smoke passed
- exact then-current Specification binding passed
- build passed
- wheel/sdist `twine check` passed
- installed-wheel smoke passed

That run remains valid evidence for the distributed-authority implementation, but it is **not** the final verification for the new `0.3.4-draft @ afba3531...` binding.

### Post-rebind verification

Pending PR `#26` merge CI. Required checks are:

- complete pytest suite;
- `PTSIP Tool 0.3.4` identity;
- exact `0.3.4-draft @ afba3531e23d96c21b7216e49614b839158ca7d5` binding;
- canonical/embedded machine-readable contract equality;
- package build;
- wheel/sdist `twine check`;
- built-wheel reinstall and Tool/spec/CLI smoke checks.

## Tool lineage and publication policy

- Tool `0.3.0`: published
- Tool `0.3.1`: published as `tool-v0.3.1` / PyPI
- Tool `0.3.2`: source-only migration
- Tool `0.3.3`: permanently source-only; never tag, GitHub Release, or PyPI
- Tool `0.3.4`: distributed-authority implementation complete; Specification rebind in PR `#26`; publication remains a separate explicit decision

No Tool publication is implied by Specification activation or by successful CI.
