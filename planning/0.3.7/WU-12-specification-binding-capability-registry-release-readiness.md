# WU-12 — Typed Specification Binding, Capability Registries, Repository Adoption, and Final Release Readiness

> **Status:** PLANNED  
> **Target Tool:** `0.3.7`  
> **Predecessor:** WU-11 — PP-aware Specification / release-surface preparation  
> **Architecture authority:** ADR-0023  
> **Entry rule:** may enter ACTIVE only after WU-11 completion is recorded and a fresh exact `dev/0.3.7` HEAD is captured  
> **Release role:** final Tool `0.3.7` integration and release-readiness authority

## 0. Purpose

Complete the long-term-maintainability Specification-binding architecture selected by the project owner without turning WU-10 historical migration bridges into a generic validation bottleneck.

WU-12 separates Tool, Project Profile, Specification, and Project Profile instance identities all the way through runtime validation, repository adoption, release verification, and final exact-SHA evidence.

The target shape is:

```text
Tool 0.3.7
    ├─ explicit PP capabilities
    └─ explicit Specification capabilities

Project Profile pp.1.01
    └─ explicit SpecificationBinding
         family
         source
         immutable revision

historical migration bridge
    └─ used only when historical migration interpretation is actually required
```

## 1. Typed Specification binding

Introduce a typed `SpecificationBinding` model with explicit fields for:

```text
family
source
immutable revision
```

The model must preserve exact identity and provide stable malformed/unsupported diagnostics.

Canonical current-generation Project Profiles SHALL no longer infer Specification family from `ptsip.version`.

Expected canonical shape:

```yaml
ptsip:
  version: "pp.1.01"
  specification:
    family: "0.3.7-draft"
    source: "https://github.com/Kinirin/PTSIP"
    revision: "<immutable-specification-revision>"
```

## 2. Specification capability registry

Add an explicit typed registry for Specification capabilities.

The registry must be operation-aware and must not become a second copy of the PP registry.

At minimum distinguish operations such as:

```text
IDENTIFY
VALIDATE
ANALYZE
CONFORM
CREATE_TARGET
```

where the implementation genuinely needs different support boundaries.

The registry must support exact immutable release binding while allowing deliberately supported historical/current families without requiring a migration bridge for ordinary validation.

Unknown family/source/revision combinations fail closed.

## 3. Independent PP / Specification capability composition

Keep:

```text
Tool -> PP capability
Tool -> Specification capability
```

as independent authorities.

Do not create a permanent Cartesian matrix for every Tool × PP × Specification combination.

Normal composition:

```text
require PP capability
    +
require Specification capability
    +
apply explicit cross-contract constraint only if one exists
```

Where a PP contract genuinely requires or excludes a Specification family/semantic feature, represent that relationship through a narrow typed compatibility rule rather than duplicating complete registry entries.

## 4. Generic validator responsibility separation

Remove the inherited rule:

```text
profile.specification.revision == Tool.SPEC_REVISION
```

from generic validation.

Generic validation must not consult historical Project Profile migration bridges merely to determine whether a current declared Specification binding is supported.

The validator pipeline should instead be:

```text
parse PP identity
    -> require PP VALIDATE capability
parse SpecificationBinding
    -> require Specification VALIDATE capability
schema / Responsibility Map validation
    -> optional narrow PP-Spec compatibility constraint
```

Historical migration readers remain separate.

## 5. Canonical schema and embedded contract changes

Synchronize public and embedded `pp.1.01` schema/specdata so canonical current-generation profiles can represent Specification family explicitly.

Requirements:

- add the canonical Specification family field under the PP contract;
- keep public and embedded schema bytes synchronized;
- do not mutate immutable historical `0.3.6-draft` schemas as though they originally contained the new field;
- retain historical-source readers/fixtures for actual legacy shape;
- update maintained examples/templates only under the current PP contract.

## 6. Historical source compatibility remains migration-only

WU-10 historical compatibility remains valid and revision-bound.

Examples include:

```text
0.3.4-draft @ b5b17dd... -> pp.1.01 semantic migration
0.3.6-draft @ d6995ed... -> pp.1.01 identity-only migration
```

These bridges are not deleted, but their responsibility is narrowed explicitly to historical migration interpretation.

Regression must prove that removing them from generic validation authority does not weaken historical migration safety.

## 7. PTSIP repository self-adoption

After the new binding model and validator are implemented, migrate the PTSIP repository root from its historical source identity to the current canonical state.

Intended final state:

```text
ptsip.yaml
    PP contract:          pp.1.01
    Specification family: 0.3.7-draft
    Specification source: https://github.com/Kinirin/PTSIP
    Specification revision: final WU-12 immutable normative snapshot
```

The root migration is explicitly authorized by the project-owner decision that created WU-12.

It must still preserve normal mutation safety:

- exact source state;
- schema validity;
- no unintended component/relationship/artifact/policy semantic delta;
- repository coverage remains valid;
- post-write validation succeeds.

Do not create `ptsip_0.3.7.yaml`.

## 8. Final Specification freeze

`555af435a4bb68140c2c869efa34d12c624d51a4` is the WU-11 PP-aware transition Specification baseline, not a permanently forced release revision.

Because WU-12 changes normative Specification-binding/schema surfaces, WU-12 must create or select a later immutable normative snapshot after all affected canonical assets are coherent.

Required chain:

```text
WU-11 transition baseline
    ↓
WU-12 typed binding + schema/validator contract
    ↓
coherent normative assets
    ↓
immutable final Specification snapshot
    ↓
Tool 0.3.7 SPEC_REVISION binding
    ↓
root pp.1.01 + 0.3.7-draft adoption
```

The final release `SPEC_REVISION` must point to that immutable WU-12 snapshot.

## 9. Release-contract verifier

Refactor release verification to check independent identities rather than numeric/equality coupling.

The release gate must verify at least:

```text
Tool package/runtime == 0.3.7
Tool bound Specification family == 0.3.7-draft
Tool bound SPEC_REVISION == immutable final WU-12 snapshot
current supported PP target == pp.1.01
root Project Profile == pp.1.01
root SpecificationBinding == supported 0.3.7-draft exact binding
release note namespace == releasenote/tool/0.3.7.md
public/embedded PP schema synchronization
```

Do not infer PP identity from Tool SemVer.

## 10. Workflow and package synchronization

Synchronize:

```text
.github/scripts/verify_release_contract.py
.github/workflows/tooling-test.yml
.github/workflows/release.yml
.github/workflows/tooling-release.yml
pyproject.toml
src/ptsip/constants.py
src/ptsip/spec_identity.py
releasenote/
STATUS.md
README.md / README.ko.md where user-visible identity requires it
```

Preserve self-hosted verification policy and exact-SHA release gates.

Publication artifact evidence must continue to preserve separate Product ownership for:

```text
ptsip-core
ptsip-evidence
ptsip-source-compat
ptsip-migration
ptsip-embedded-contracts
vpms
```

## 11. Focused tests

Add/organize focused tests for at least:

- typed SpecificationBinding parse/serialization;
- malformed/missing family/source/revision diagnostics;
- supported and unsupported Specification capability lookup;
- PP capability independent from Specification capability;
- narrow cross-contract constraint behavior;
- generic validator does not require historical bridge;
- generic validator does not require exact equality with Tool `SPEC_REVISION` unless the selected capability says so;
- historical source migration still requires explicit historical bridge;
- canonical pp.1.01 schema requires/accepts explicit family as selected by the final contract;
- immutable legacy schemas remain unchanged;
- root PTSIP adoption is identity/binding-safe and semantically stable;
- release contract reports Tool / PP / Specification identities independently.

Touched/new tests should continue the gradual purpose/role-based test organization rather than trigger a wholesale test migration.

## 12. Full regression and distribution verification

Run the complete repository regression against the exact final WU-12 source SHA.

At minimum verify:

- WU-09 PP identity tests;
- WU-10 direct-convergence / execution / recovery tests;
- WU-12 binding/capability tests;
- repository root profile validation with zero unexpected coverage gaps;
- CLI `ptsip --version` and `ptsip spec` identities;
- package build/twine;
- Product Artifact exact-snapshot binding;
- installed-wheel smoke;
- no tests or local migration ledgers/checkpoints packaged;
- VPMS boundary remains unchanged unless separately authorized.

## 13. Exact-SHA release-readiness gate

The final self-hosted workflow must verify the exact dispatched SHA and record success only after all required tests, release-contract checks, build/artifact validation, and installed-wheel smoke pass.

Earlier WU-11 or WU-10 runs remain evidence for their own SHAs but cannot substitute for final WU-12 evidence.

## 14. Non-goals

WU-12 must not:

- redesign PP version grammar;
- invent new lifecycle classifications;
- make historical bridge availability equivalent to generic Specification support;
- create automatic migration authority from registry presence;
- require a Cartesian Tool × PP × Specification table where independent capability composition is sufficient;
- retroactively rewrite historical release/schema identities;
- publish Tool `0.3.7` before final exact-SHA evidence is complete.

## 15. Completion gate

WU-12 is complete only when:

- typed SpecificationBinding exists and is used by canonical validation;
- Specification capability registry is explicit, typed, and operation-aware;
- PP and Specification capabilities are independently composed;
- historical migration bridge is no longer generic validation authority;
- current PP schema/specdata and maintained examples are synchronized;
- PTSIP root is validly adopted to `pp.1.01` with explicit `0.3.7-draft` binding;
- final immutable WU-12 Specification revision is selected and Tool `0.3.7` binds it;
- generic validation, historical migration, and release verification all pass their focused suites;
- complete regression passes at the exact final SHA;
- package/distribution/artifact/smoke verification passes;
- final self-hosted exact-SHA workflow succeeds;
- release documentation records Tool, PP, Specification, source SHA, and artifact identity independently;
- final release handoff is ready without Tool/PP/Specification coupling.
