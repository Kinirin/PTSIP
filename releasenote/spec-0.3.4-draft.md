# PTSIP Specification 0.3.4-draft Release Notes

**Status:** Active draft family after coherent normative migration  
**Design predecessor:** `0.3.3-draft`  
**Previous active baseline:** `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`  
**Implementation evidence:** PTSIP Reference Tool `0.3.4`  
**Tool implementation merge:** `555c528593f700a348d8da84545a62ce61291cae`  
**Tool verification completion before rebind:** `8cd0ddf16dc9b56f27f694138a37caae1c49bb4f`  
**Identity model:** draft family label + immutable Git revision

The `spec-v0.3.4-draft` GitHub Release originally published the design record for Explicit Project Adoption plus Distributed Authority Consistency. The tag/release is historical design provenance; the exact active normative identity is the immutable coherent-migration revision that contains the canonical Specification, schema, registry, agent contract, ADR, and embedded resources.

Because a Git commit cannot safely contain its own literal SHA, the immutable normative migration commit is created first. A following Tool binding commit records that already-existing revision as `SPEC_REVISION`.

## 1. What becomes normative

PTSIP `0.3.4-draft` preserves the established Product/Toolchain/Neutral Contract architecture and adds two connected capabilities:

1. **Explicit Project Adoption with lossless durable architecture facts.**
2. **Backend-neutral Distributed Authority Consistency for implementations that claim distributed coordination.**

PTSIP still has exactly three architecture classifications:

- `PRODUCT`;
- `TOOLCHAIN`;
- `NEUTRAL_CONTRACT`.

Distributed authority state does not create another plane.

## 2. Durable explicit adoption facts

The component-level Project Profile can represent the structured fact set used by adoption/resolution:

- `classification`;
- `purpose`;
- `shipped`;
- `runtime_required`;
- `lifecycle_owner`;
- `executable`.

Canonical lifecycle owners are:

- `PRODUCT`;
- `DEVELOPMENT_TOOLING`;
- `INDEPENDENT`.

`release_owner` and `compatibility_owner` remain separate optional project metadata and are not aliases for `lifecycle_owner`.

A write-enabled structured adoption/resolution workflow must preserve supplied facts losslessly. Boundary-root shorthand may remain structurally valid, but the Tool must refuse structured mutation when shorthand cannot preserve those facts rather than silently dropping them.

## 3. Decision Authority separation

PTSIP distinguishes:

```text
Decision Authority
    -> which explicit architecture answer won

Project Profile
    -> durable project-owned architecture declaration

Observed evidence
    -> what the repository/artifacts actually do

Conformance Evaluation
    -> whether declaration + observed evidence satisfy PTSIP rules
```

A Decision Authority is not a conformance oracle.

## 4. Distributed coordination requirements

A backend claiming distributed coordination must provide:

- stable coordination-domain + component-scope decision identity;
- ordered authority revision/state;
- first-valid-resolution-wins protected by conditional mutation;
- authority freshness at architecture-sensitive operation boundaries;
- read-only absence observation that does not fabricate decision history merely to prove absence;
- deterministic missing/equivalent/conflicting reconciliation;
- semantic-equivalence comparison independent from serialization formatting;
- explicit conflict rather than silent overwrite;
- stale profile/evidence refusal;
- fail-closed behavior instead of implicit isolated Local fallback; and
- separation of global decision state from clone-local projection/application state.

Continuous background polling is not required; action-time synchronization is sufficient.

## 5. New rule IDs

The coherent migration assigns:

- `PTSIP-ADP-001` — Explicit project adoption preserves architecture intent losslessly.
- `PTSIP-AUT-001` — Decision Authority and Project Profile responsibilities are distinct.
- `PTSIP-AUT-002` — Stable distributed decision identity.
- `PTSIP-AUT-003` — First valid resolution wins through ordered conditional mutation.
- `PTSIP-AUT-004` — Authority freshness at architecture-sensitive boundaries.
- `PTSIP-AUT-005` — Safe authority/profile reconciliation.
- `PTSIP-AUT-006` — Fail-closed distributed coordination.
- `PTSIP-AUT-007` — Global decision state is distinct from local projection state.

`PTSIP-AUT-*` rules apply to implementations claiming distributed coordination capability. They are not additional Consumer Repository architecture requirements when distributed coordination is not used.

## 6. Coherent migration assets

The activation migration aligns at one repository state:

- `spec/PTSIP-SPEC.md`;
- `spec/PTSIP-CONFORMANCE.md`;
- `spec/PTSIP-TERMINOLOGY.md`;
- `spec/PTSIP-GOVERNANCE.md`;
- `schemas/ptsip-profile.schema.json`;
- `registry/ptsip-registry.yaml`;
- `agents/AGENT-CONTRACT.md`;
- `adoption/ADOPTION-GUIDE.md`;
- `reference/REFERENCE-ARCHITECTURE.md`;
- `src/ptsip/specdata/ptsip-profile.schema.json`;
- `src/ptsip/specdata/ptsip-registry.yaml`;
- `decisions/ADR-0005-activate-spec-0.3.4-draft.md`;
- Reference Tool profile projection behavior required to preserve new durable facts.

Canonical and embedded profile schema copies are byte-identical in the migration state. Canonical and embedded registry copies are likewise byte-identical.

## 7. Tool binding boundary

Reference Tool `0.3.4` was originally implemented and verified while bound to `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`.

The Tool may claim `0.3.4-draft` only after its binding constants point to the immutable coherent-migration revision and the complete regression/build verification succeeds again with the new embedded resources and lossless profile projection.

## 8. Compatibility

Existing `0.2.0-draft` profiles remain historical declarations interpreted under the immutable predecessor revision. They are not silently reinterpreted as `0.3.4-draft`.

The Product/Toolchain/Neutral classification model, artifact-owner/producer separation, evidence-relative conformance, no-waiver remediation policy, and Consumer Repository non-intrusion remain preserved.
