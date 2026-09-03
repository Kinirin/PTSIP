# PTSIP Specification 0.3.4-draft Release Notes

**Status:** Active draft family after coherent normative migration  
**Design predecessor:** `0.3.3-draft`  
**Previous active baseline:** `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`  
**Implementation evidence:** PTSIP Reference Tool `0.3.4`  
**Tool implementation merge:** `555c528593f700a348d8da84545a62ce61291cae`  
**Tool verification completion before rebind:** `8cd0ddf16dc9b56f27f694138a37caae1c49bb4f`  
**Normative freeze revision:** `b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`<br>
**Identity model:** draft family label + immutable Git revision

The existing `spec-v0.3.4-draft` tag/GitHub Release is the earlier public design-release checkpoint and remains immutable historical provenance. It is not moved to the later activation snapshot.

The exact active normative identity of this mutable draft family is:

```text
0.3.4-draft @ b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e
```

The repository-identity-migration freeze is canonical while preserving the earlier freeze commit in repository history.

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

The immutable freeze `b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e` aligns:

- `spec/PTSIP-SPEC.md`;
- `spec/PTSIP-CONFORMANCE.md`;
- `spec/PTSIP-TERMINOLOGY.md`;
- `spec/PTSIP-GOVERNANCE.md`;
- `schemas/ptsip-profile.schema.json`;
- `schemas/ptsip-artifact-evidence.schema.json`;
- `schemas/ptsip-agent-classification.schema.json`;
- `schemas/ptsip-diagnostic.schema.json`;
- `registry/ptsip-registry.yaml`;
- `agents/AGENT-CONTRACT.md`;
- `adoption/ADOPTION-GUIDE.md`;
- `reference/REFERENCE-ARCHITECTURE.md`;
- `profiles/example.ptsip.yaml`;
- `decisions/ADR-0005-activate-spec-0.3.4-draft.yaml`;
- Reference Tool profile projection behavior required to preserve new durable facts; and
- all packaged `src/ptsip/specdata/*` counterparts used by the Tool.

At the freeze state, every packaged canonical/embedded machine-readable pair is byte-identical:

- profile schema;
- registry;
- artifact-evidence schema;
- agent-classification schema;
- diagnostic schema.

The artifact, agent-classification, and diagnostic contract semantics themselves did not require a 0.3.4 format revision; their packaged copies were normalized to the canonical source so one bound Tool cannot carry divergent Specification bytes.

## 7. Tool binding boundary

Reference Tool `0.3.4` was originally implemented and verified while bound to `0.2.0-draft` revision `a877b2f66a7f94c1b844c979e1b08fb08a9a8e45`.

The following Tool binding commit points to `b5b17dd16667cc1afaf1d23054b6e5dd773e3f5e`. The complete regression/build verification must succeed under that binding before Tool `0.3.4` is published.

## 8. Compatibility

Existing `0.2.0-draft` profiles remain historical declarations interpreted under the immutable predecessor revision. They are not silently reinterpreted as `0.3.4-draft`.

The Product/Toolchain/Neutral classification model, artifact-owner/producer separation, evidence-relative conformance, no-waiver remediation policy, and Consumer Repository non-intrusion remain preserved.
