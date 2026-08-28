# WU-11 — Tool 0.3.7 PP-Aware Specification and Release-Surface Preparation

> **Status:** COMPLETE / HANDOFF VERIFIED  
> **Target Tool:** `0.3.7`  
> **Predecessor:** WU-10 — Project Profile Compatibility, Migration, and Adoption (`COMPLETE / VERIFIED`)  
> **Successor:** WU-12 — Typed Specification Binding, Capability Registries, Repository Adoption, and Final Release Readiness  
> **Architecture authorities:** ADR-0017, ADR-0019, ADR-0020, ADR-0021, ADR-0022, ADR-0023  
> **Exact entry baseline:** `d7a19de22721b546c62809984b67c2ba1c723e7d`  
> **Predecessor verification authority:** `9a9159685a4f5de103d79e9d1c38bdbbada25d4c` via `tooling-test` run `33081718965`, job `98550433277`  
> **PP-aware normative baseline:** `555af435a4bb68140c2c869efa34d12c624d51a4`  
> **Release role:** predecessor preparation only; WU-12 is the final Tool `0.3.7` release-readiness authority

## 0. Closure summary

WU-11 was originally created as the final Tool `0.3.7` regression / Specification freeze / release-readiness WU. During release-readiness review, that scope exposed a deeper inherited coupling between Tool Specification identity and Project Profile validation.

The project owner selected the long-term-maintainability architecture recorded in ADR-0023 and explicitly moved that work into a new WU-12 rather than forcing it into WU-11.

WU-11 therefore closes as a **PP-aware normative/release-surface preparation and architecture-handoff WU**, not as final release authority.

No Tool `0.3.7` publication, final root Project Profile adoption, final `SPEC_REVISION` binding, or final release-readiness claim is made by WU-11.

## 1. Completed PP-aware transition baseline

WU-11 synchronized the `0.3.7-draft` transition companion with the behavior already implemented and verified by WU-09/WU-10.

The normative baseline is:

```text
555af435a4bb68140c2c869efa34d12c624d51a4
spec: finalize pp-aware 0.3.7 transition semantics
```

It records:

- Tool / Project Profile identity independence;
- canonical PP identity `pp.1.01`;
- `0.3.6-draft -> pp.1.01` as `IDENTITY_ONLY`;
- direct latest-target convergence instead of mandatory intermediate traversal;
- legacy physical target continuity where explicitly authorized;
- equivalent-target collision fail-closed behavior;
- owner-decision boundaries for non-deterministic semantic migration;
- PP-aware guarded execution and promotion semantics.

`555af435...` is retained as WU-11 normative evidence. It is not required to remain the final Tool `0.3.7` release `SPEC_REVISION` because WU-12 changes additional normative binding/schema surfaces.

## 2. Release-surface inventory completed

WU-11 reviewed the surfaces that ultimately need to express Tool, PP, and Specification identities independently, including:

```text
src/ptsip/constants.py
src/ptsip/spec_identity.py
src/ptsip/validation/profile.py
pyproject.toml
schemas/
src/ptsip/specdata/
spec/
releasenote/
profiles/
adoption/ADOPTION-GUIDE.md
agents/AGENT-CONTRACT.md
README.md
README.ko.md
STATUS.md
MEMORY.md
AGENTS.md
.github/scripts/
.github/workflows/
```

The review established that final release synchronization cannot safely be completed while the generic validator still assumes one installed Tool `SPEC_REVISION` is the only valid Project Profile Specification revision.

## 3. Responsibility coupling discovered and assigned

The inherited coupling is:

```text
profile.specification.revision == Tool.SPEC_REVISION
```

Additional release-surface assumptions include:

```text
Tool release note path == releasenote/<tool-version>.md
publication metadata hard-coded to 0.3.6
legacy assumptions that ptsip.version also identifies Specification family
```

A proposed workaround that routed ordinary Project Profile validation through WU-10 historical migration bridges was rejected because it would make migration compatibility a central generic-validation bottleneck.

ADR-0023 instead establishes three independent authorities:

```text
Tool -> Project Profile capability
Tool -> Specification capability
historical source -> migration compatibility bridge
```

Historical migration bridges remain migration interpretation authority only.

## 4. WU-12 handoff contract

WU-12 owns the implementation of:

```text
Typed SpecificationBinding
    family
    source
    immutable revision

Specification capability registry
    operation-aware Tool -> Specification support

Independent capability composition
    PP capability
    + Specification capability
    + narrow cross-contract constraints only where required

Generic validator responsibility separation
    remove profile revision == Tool SPEC_REVISION coupling
    do not use historical migration bridge as generic validation registry

PTSIP repository adoption
    Project Profile: pp.1.01
    Specification family: 0.3.7-draft
    Specification revision: final WU-12 immutable normative snapshot

Final release work
    final immutable SPEC_REVISION
    Tool runtime/package 0.3.7
    release-note/workflow/package synchronization
    full exact-SHA regression
    package/distribution verification
    final release handoff
```

This handoff is governed by ADR-0023.

## 5. Repository adoption boundary preserved

WU-11 intentionally leaves the repository canonical Project Profile on its historical declaration:

```text
ptsip.yaml
    version: 0.3.6-draft
    specification revision: d6995ed232e845b88d8235b851e80ab54b7804ea
```

This is not a rejection of current PP adoption. The project owner has explicitly authorized WU-12 to move the repository to:

```text
Project Profile:       pp.1.01
Specification family: 0.3.7-draft
Specification revision: final WU-12 immutable normative snapshot
```

after the typed binding and capability architecture exists.

No `ptsip_0.3.7.yaml` is created merely to mirror Tool `0.3.7`.

## 6. Verification evidence

WU-11 introduced no production runtime implementation after the PP-aware normative baseline. Exact comparison:

```text
base: 555af435a4bb68140c2c869efa34d12c624d51a4
head before closure: 66edd17390d3d108710448b3b12276bc2d832d6d
```

shows only:

```text
decisions/ADR-0023-typed-specification-binding-and-capability-registries.md
planning/0.3.7.md
planning/0.3.7/WU-11-final-regression-specification-freeze-release-readiness.md
planning/0.3.7/WU-12-specification-binding-capability-registry-release-readiness.md
```

No production source, schema, workflow, package metadata, or repository profile changed in that interval.

Therefore the last full implementation verification remains the WU-10 exact-SHA authority:

```text
9a9159685a4f5de103d79e9d1c38bdbbada25d4c
workflow run: 33081718965
workflow job: 98550433277
482 passed / 0 failed
self-hosted/tooling-test: success
```

A new final full regression is deliberately deferred to WU-12 because WU-12 owns the production binding architecture and final release state.

## 7. Completion gate result

WU-11 completion requirements are satisfied:

- PP-aware `0.3.7-draft` transition baseline recorded at `555af435...`;
- WU-09/WU-10 semantics represented without Tool/PP numeric coupling;
- release/validator coupling points identified and assigned to WU-12;
- historical bridge generic-validation workaround explicitly rejected;
- ADR-0023 accepted;
- WU-12 pre-created with a typed implementation contract;
- no production implementation change was hidden in the WU-11 split;
- no unauthorized root adoption occurred;
- no premature final `SPEC_REVISION` or release-readiness claim occurred.

WU-11 is therefore `COMPLETE / HANDOFF VERIFIED`.

The standing successor-entry rule may activate WU-12 after this closure commit is captured as its exact entry baseline. WU-12 entry state does not by itself authorize publication; it authorizes execution of the already accepted ADR-0023 implementation scope.
