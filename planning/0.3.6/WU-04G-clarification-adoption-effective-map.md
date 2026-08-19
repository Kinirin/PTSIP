# WU-04G — Clarification / Adoption Effective-Map Integration and Decision-Protocol Upgrade

> **Status:** ACTIVE — maintainer decisions D1-B through D9-B accepted; implementation not yet started  
> **Parent work unit:** WU-04 — template catalog + deterministic materialization + effective-map consumers  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-04F — conformance consumes the effective Responsibility Map  
> **Entry baseline:** `52a455115d191123504c2fd690ffe499caf0ff6a`  
> **Pre-entry review:** `planning/0.3.6/pre-entry-WU-04G-decision-review.md`  
> **Bound Specification snapshot at entry:** `82abd09360df09a95fbbfb516855fa9ffb49f050`

## 1. Accepted maintainer decision package

The maintainer selected the following options for WU-04G:

```text
D1-B  ResolvedProfile.effective_payload is the clarification/adoption read authority
D2-B  invalid/unresolvable profiles fail closed, with post-failure recovery work extended
D3-B  effective associated-artifact coverage suppresses duplicate component questions
D4-B  selector/candidate coverage uses one shared canonical primitive
D5-B  create ptsip-clarification-answer/v2 and remove lifecycle_owner from the canonical answer
D6-B  accepted adoption decisions may extend hybrid and convert template -> hybrid when required
D7-C  eligible accepted decisions automatically perform that hybrid conversion/application
D8-B  exact selected profile path travels through the complete decision protocol and remote CAS
D9-B  optimize/migrate only WU-04G-owned tests with coverage-preservation proof
```

This package intentionally expands WU-04G beyond the provisional read-only recommendation. The expansion is accepted and must be implemented without weakening the authority boundaries frozen by WU-04C.

## 2. Non-negotiable authority boundary

WU-04G must preserve all of the following:

- `classification` remains canonical lifecycle ownership;
- canonical PTSIP Tool 0.3.6 never restores `TOOLCHAIN` as a classification alias;
- template identity (`id + immutable revision`) remains project-selected authority;
- materialization remains deterministic and non-authoritative;
- an effective map is a runtime/read model and must never be dumped wholesale back as project-owned explicit source;
- associated artifacts remain non-component architecture;
- invalid/unresolvable profiles fail closed;
- VPMS remains outside WU-04G.

### Accepted-decision authority for D6-B / D7-C

Automatic `template -> hybrid` conversion is permitted only when there is an accepted project-owned clarification/adoption decision that requires a new source declaration.

The required authority chain is:

```text
repository evidence
    -> discovers candidate only
    -> does NOT authorize architecture

maintainer / authorized project decision
    -> supplies canonical component decision
    -> authorizes the project-owned declaration change

Tool safe-apply layer
    -> may encode that accepted decision as a hybrid project extension/replacement
    -> may change source mode template -> hybrid when required to represent the accepted decision

materializer
    -> only reproduces the resulting declaration deterministically
```

Discovery, path heuristics, confidence, candidate identity, or the materializer itself MUST NOT trigger a source-mode conversion.

## 3. G0 — normative authority freeze before runtime implementation

D6-B and D7-C add a new normative meaning that is not explicitly frozen by the current `PTSIP-RMAP-013` through `PTSIP-RMAP-016`: an accepted project decision may authorize a source-mode transition and project-owned hybrid extension.

Therefore WU-04G begins with a normative precondition:

1. update the canonical Responsibility Map Specification/registry contract to define accepted clarification/adoption decisions as project-owned declaration authority for the exact accepted change;
2. state that automatic template -> hybrid conversion is allowed only as a representation of that accepted project decision;
3. retain exact template `id + immutable revision` during conversion;
4. prohibit materialize-to-explicit writeback;
5. prohibit unrelated implicit overrides/removals or cascading repair;
6. align canonical and embedded registry copies;
7. select a new immutable `SPEC_REVISION` only after the normative change is committed;
8. update Tool constants/root binding after that immutable revision exists.

Runtime implementation that depends on D6-B/D7-C MUST NOT precede this freeze.

## 4. D1-B — effective-map read authority

Clarification/adoption coverage must use:

```text
selected profile
    -> validate_profile()
    -> ValidationResult.resolved_profile
    -> ResolvedProfile.effective_payload
```

Effective components and effective associated artifacts are the only architecture coverage authority for explicit/template/hybrid source modes.

Raw source remains available for source identity, provenance, projection, and safe-write decisions, but it is not a second coverage interpretation.

## 5. D2-B — fail closed plus recovery workflow

If parse/schema/binding/materialization/effective semantic/selector validation cannot produce a valid `ResolvedProfile`, clarification/adoption must not fall back to raw source or manufacture candidate questions.

The failure path is extended beyond a terminal error:

```text
profile resolution failure
    -> block architecture-dependent clarification/adoption
    -> preserve validation/materialization errors
    -> classify the failing stage
    -> expose the exact selected profile path
    -> expose a non-authoritative remediation/retry plan
    -> require project/source correction
    -> re-run validation against a fresh repository/profile snapshot
    -> resume clarification/adoption only after a valid ResolvedProfile exists
```

The remediation surface may explain required corrections but MUST NOT infer architecture or silently repair declarations. Partial effective state from a failed attempt must not be reused as authority on retry.

Public status names may be finalized during implementation, but failure and recovery state must be machine-readable enough for local CLI and remote control-plane callers to continue deterministically after the project fixes the profile.

## 6. D3-B / D4-B — associated-artifact suppression and canonical selector coverage

Effective associated-artifact coverage suppresses duplicate component clarification for the same declared support scope. This does not permanently prevent later explicit promotion/re-evaluation if independent lifecycle responsibility emerges.

Clarification-local selector interpretation must be replaced by/reduced to a shared canonical candidate-coverage primitive owned by the validation/selector layer.

Required semantic result shape should distinguish at least:

```text
covered by one component
covered by one associated artifact
uncovered
ambiguous/conflicting
```

Ambiguity must fail/review rather than guess ownership. The shared primitive must preserve include/exclude/specificity semantics used by canonical validation rather than creating a second selector dialect.

## 7. D5-B — clarification answer v2

WU-04G introduces:

```text
ptsip-clarification-answer/v2
```

Canonical v2 decision fields are:

```text
classification
purpose
shipped
runtime_required
executable
```

`lifecycle_owner` is removed from the canonical answer contract and is not serialized into canonical Tool 0.3.6 profiles.

Required migration behavior:

- newly generated clarification requests/answers use v2;
- new canonical decision storage uses v2 semantics;
- existing v1 data may be read only through an explicit compatibility/migration path where needed for already-persisted decisions;
- v1 `lifecycle_owner` must never override canonical `classification`;
- legacy PTSIP `TOOLCHAIN` remains migration input only and is not automatically translated to `DEVELOPMENT_TOOLING`;
- replacement tests must prove the v1-to-v2 migration boundary rather than silently deleting historical coverage.

WU-04G owns this decision-protocol migration because the maintainer explicitly selected D5-B.

## 8. D6-B / D7-C — automatic template -> hybrid safe apply

For an accepted decision requiring a declaration not already represented by the effective map:

### explicit source

Apply through the canonical explicit projection path.

### template source

The Tool may automatically transform:

```text
template(id, revision)
    -> hybrid(same id, same revision)
         + exact project-owned accepted extension/replacement
```

Only the minimum project-owned delta needed to represent the accepted decision may be written.

### hybrid source

The Tool may add/update the exact project-owned hybrid extension/replacement needed by the accepted decision while preserving unrelated overrides/removals.

### Prohibited writes

The Tool must not:

- serialize the entire effective map into explicit form;
- copy template-owned entities into project overrides merely because they exist;
- change the selected template revision;
- create unrelated removals/relationships;
- overwrite a conflicting existing project declaration silently;
- repair invalid downstream relationships heuristically.

A conflict that prevents lossless safe apply remains fail-closed and must report the conflict.

## 9. D8-B — exact profile path through decision protocol and CAS

The selected Project Profile path becomes part of the decision/control-plane identity.

It must flow through:

```text
clarification/gate creation
    -> persisted decision record
    -> issue / agent resolution context
    -> exact subject revision
    -> remote file read
    -> projection validation
    -> branch-head stale check
    -> CAS write to the same exact profile path
```

Rules:

- store a normalized repository-relative profile path;
- reject paths outside the repository;
- do not silently substitute root `ptsip.yaml` when a different profile was selected;
- stale/retry logic must preserve the selected path;
- a changed selected profile path requires a new/rebound gate rather than applying an old decision to a different file;
- local and remote application semantics should converge on the same path identity.

## 10. D9-B — bounded G-owned test migration/optimization

WU-04G may optimize only tests directly changed because of this stage.

Primary surfaces include:

```text
tests/ptsip/test_adoption_033.py
tests/ptsip/test_decision_control_plane.py
clarification-focused tests directly touched by G
tests/ptsip/test_clarification_adoption_effective_map_036.py
```

Before optimization, collect diagnostic evidence such as:

```text
python -m pytest -q --durations=30 --durations-min=0.05
```

Allowed work:

- consolidate duplicated immutable repository/profile construction;
- parameterize equivalent semantic variants;
- introduce narrow reusable helpers;
- migrate stale G-owned expectations to canonical 0.3.6/v2 behavior;
- record a replacement mapping for any removed/merged historical test.

Forbidden work:

- no semantic contract removal for speed;
- no sharing away mutable CAS/snapshot/stale-state isolation;
- no topology/VPMS/pilot/repository-wide migration;
- no pytest-xdist/worker-count/workflow change under D9-B;
- no reduced-suite replacement for the final exact-SHA full regression.

Runtime improvement is an observation, not the completion gate. Coverage preservation is the gate.

## 11. Implementation tracks

### G0 — normative contract

Specification/registry rule for accepted-decision hybrid authority and new immutable `SPEC_REVISION`.

### G1 — effective read + selector coverage

Targets:

```text
src/ptsip/clarification/generator.py
src/ptsip/clarification/generator_core.py
src/ptsip/validation/components.py (or a narrow shared selector module)
```

Deliver D1-B, D2-B read behavior, D3-B, D4-B.

### G2 — clarification-answer/v2

Targets:

```text
src/ptsip/clarification/model.py
src/ptsip/clarification/resolution/model.py
src/ptsip/clarification/resolution/parser.py
src/ptsip/clarification/resolution/resolver.py
render/transport/store surfaces as required
```

Deliver D5-B and explicit v1 migration handling.

### G3 — projection + automatic hybrid conversion

Targets:

```text
src/ptsip/clarification/resolution/profile_projection.py
src/ptsip/adoption.py
```

Deliver D6-B/D7-C with minimal project-owned delta serialization.

### G4 — exact profile-path control plane

Targets:

```text
src/ptsip/app/service.py
src/ptsip/app/store.py
gate/CLI payload construction and GitHub CAS surfaces
```

Deliver D8-B and integrate G3 remote safe apply.

### G5 — recovery behavior + tests/optimization

Deliver the D2 post-failure recovery surface, D9-B bounded migration, focused tests, and cross-track integration.

Parallel work may be used only after G0 fixes the shared normative meaning and file ownership is partitioned to avoid concurrent edits.

## 12. Focused regression contract

Primary new focused target:

```text
tests/ptsip/test_clarification_adoption_effective_map_036.py
```

Required coverage includes:

1. equivalent explicit/template/hybrid effective coverage yields equivalent clarification suppression;
2. template-provided component prevents duplicate clarification;
3. hybrid override selectors are the selectors clarification consumes;
4. hybrid removals expose genuinely uncovered candidates;
5. effective associated artifacts suppress duplicate component questions;
6. invalid/unknown template revision fails closed and exposes recovery information without raw fallback;
7. retry after a corrected profile uses a fresh resolved profile and can continue;
8. canonical shared selector semantics report ambiguity rather than guessing;
9. clarification-answer/v2 omits `lifecycle_owner` and accepts only canonical 0.3.6 classification vocabulary;
10. explicit v1 compatibility/migration tests retain historical data handling without restoring `TOOLCHAIN` authority;
11. accepted template decision converts source to hybrid while retaining exact template id/revision;
12. automatic conversion writes only the accepted project delta and never materializes the full effective map into source;
13. accepted hybrid decision preserves unrelated overrides/removals;
14. conflicting safe-apply remains fail-closed and does not partially mutate source;
15. exact non-root profile path is preserved through local/remote gate, store, stale check, projection, and CAS;
16. stale branch/profile state does not apply an accepted decision to a different revision/path;
17. repeated clarification/gate operations do not reopen architecture already represented by the new effective declaration;
18. read-only clarification does not mutate the source profile;
19. D9-B migrated tests have traceable replacement coverage and preserve mutable-state isolation.

## 13. Completion gate

WU-04G is complete only when:

- G0 normative accepted-decision authority is frozen and the Tool binds to the resulting immutable Specification snapshot;
- clarification/adoption reads use only validated effective architecture;
- invalid profiles fail closed and expose deterministic recovery/retry information;
- associated-artifact and selector coverage use canonical shared semantics;
- clarification-answer/v2 is canonical and `lifecycle_owner` is removed from new decisions;
- required v1 compatibility/migration handling is explicit and tested;
- accepted template/hybrid decisions apply as minimal project-owned hybrid changes without materialize-to-explicit writeback;
- exact selected profile path is preserved through local/remote decision state and CAS;
- G-owned test optimization preserves semantic coverage with before/after diagnostic evidence;
- focused WU-04G tests pass;
- one exact-SHA self-hosted complete repository regression is reviewed and remaining failures are classified;
- WU-04H remains locked until this gate is reviewed.

## 14. Out of scope

WU-04G still does not authorize:

- Tool 0.3.5 legacy architecture migration beyond the narrow clarification-answer v1 compatibility needed for D5-B;
- blind `TOOLCHAIN -> DEVELOPMENT_TOOLING` translation;
- topology migration;
- VPMS effective-map integration;
- materialize-to-explicit writeback;
- repository-wide test restructuring;
- unrelated workflow/runner optimization.

WU-04H MUST NOT be entered or pre-created before WU-04G completion is reviewed.
