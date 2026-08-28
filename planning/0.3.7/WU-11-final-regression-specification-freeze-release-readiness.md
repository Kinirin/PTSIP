# WU-11 — Tool 0.3.7 PP-Aware Specification and Release-Surface Preparation

> **Status:** ACTIVE  
> **Target Tool:** `0.3.7`  
> **Predecessor:** WU-10 — Project Profile Compatibility, Migration, and Adoption (`COMPLETE / VERIFIED`)  
> **Architecture authorities:** ADR-0017, ADR-0019, ADR-0020, ADR-0021, ADR-0022, ADR-0023  
> **Exact entry baseline:** `d7a19de22721b546c62809984b67c2ba1c723e7d`  
> **Predecessor verification authority:** `9a9159685a4f5de103d79e9d1c38bdbbada25d4c` via `tooling-test` run `33081718965`, job `98550433277`  
> **Current PP-aware normative baseline candidate:** `555af435a4bb68140c2c869efa34d12c624d51a4`  
> **Release role:** predecessor preparation for WU-12; not final Tool `0.3.7` release authority

## 0. Purpose

Prepare the Tool `0.3.7` PP-aware Specification and release surfaces after WU-09/WU-10, while keeping the newly selected long-term Specification-binding architecture out of WU-11.

During WU-11 review, the project identified an inherited responsibility coupling: generic Project Profile validation compares a profile's Specification revision directly with the installed Tool's single `SPEC_REVISION`. A proposed historical-bridge workaround was rejected because it would make migration compatibility a generic-validation bottleneck.

The project owner selected ADR-0023's long-term-maintainability architecture and explicitly added WU-12 for that work.

Therefore WU-11 is no longer the final release-readiness authority. It establishes the PP-aware normative/release baseline and hands the remaining binding architecture to WU-12.

## 1. PP-aware transition Specification baseline

Synchronize the `0.3.7-draft` transition companion with the completed WU-09/WU-10 behavior:

- independent Tool / PP identity;
- canonical PP `pp.1.01`;
- `0.3.6-draft -> pp.1.01` `IDENTITY_ONLY` bridge;
- direct latest-target convergence;
- legacy target-path continuity;
- explicit owner-decision and fail-closed boundaries;
- guarded PP-aware execution/promotion.

WU-11 created the PP-aware transition baseline candidate:

```text
555af435a4bb68140c2c869efa34d12c624d51a4
```

This SHA is retained as WU-11 normative evidence. It is not forced to remain the final Tool `0.3.7` release `SPEC_REVISION`, because WU-12 will change additional normative binding/schema surfaces.

## 2. Release-surface inventory and preparation

Review the surfaces that must eventually reflect Tool `0.3.7`, PP `pp.1.01`, and Specification `0.3.7-draft` independently:

```text
src/ptsip/constants.py
src/ptsip/spec_identity.py
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

WU-11 may prepare non-controversial release-surface changes, namespaced release-note behavior, and workflow cleanup that do not depend on the new SpecificationBinding architecture.

## 3. Responsibility-coupling inventory

Document and isolate inherited assumptions that WU-12 must remove.

Known examples include:

```text
profile.specification.revision == Tool.SPEC_REVISION
Tool release note path == releasenote/<tool-version>.md
publication workflow metadata hard-coded to 0.3.6
legacy validator assumptions that ptsip.version identifies the Specification family
```

Do not patch these by routing ordinary validation through WU-10 historical migration bridges.

## 4. WU-12 handoff contract

WU-11 must leave a precise implementation boundary for WU-12:

```text
Typed SpecificationBinding
    family
    source
    immutable revision

Independent capability registries
    Tool -> PP capability
    Tool -> Specification capability

Narrow composition
    no permanent Cartesian matrix unless semantics require it

Historical compatibility bridge
    migration interpretation only

Generic validator
    PP capability + Specification capability + narrow compatibility constraints
```

The handoff is governed by ADR-0023.

## 5. Tool identity preparation

Tool package/runtime `0.3.7` remains the target release identity, independent from PP and Specification identities.

WU-11 may prepare Tool-version and release-note changes, but it must not claim final release binding before WU-12 has completed the new binding architecture and selected the final immutable Specification snapshot.

The following implication remains invalid:

```text
Tool 0.3.7
    => Project Profile 0.3.7-draft
```

## 6. Repository self-adoption boundary

WU-11 does not perform the final PTSIP root Project Profile adoption.

The repository may remain on its historical source identity during WU-11:

```text
ptsip.yaml
    version: 0.3.6-draft
    revision: d6995ed232e845b88d8235b851e80ab54b7804ea
```

The project owner has approved the intended WU-12 final adoption to:

```text
Project Profile:       pp.1.01
Specification family: 0.3.7-draft
Specification revision: final WU-12 immutable snapshot
```

This is delayed to WU-12 so the profile can express and validate the new typed Specification binding without temporary coupling hacks.

## 7. Verification scope

WU-11 focused verification should establish that its Specification/release-surface preparation does not regress completed WU-09/WU-10 behavior.

At minimum verify as applicable:

- PP identity remains `pp.1.01` and independent from Tool SemVer;
- WU-10 direct-convergence tests still pass;
- historical `0.3.6-draft` source remains readable under the existing migration authority;
- PP-aware transition companion matches implemented WU-10 semantics;
- any touched release workflow remains self-hosted where previously required;
- release notes use the accepted namespace direction where modified.

A full final-release exact-SHA run belongs to WU-12.

## 8. Non-goals

WU-11 must not:

- implement typed SpecificationBinding;
- implement the new Specification capability registry;
- make historical migration bridges generic validation authority;
- perform final root `pp.1.01` adoption;
- select the final post-WU-12 `SPEC_REVISION`;
- claim final Tool `0.3.7` release readiness;
- publish Tool `0.3.7`;
- redesign PP version grammar or lifecycle classifications.

## 9. Completion gate

WU-11 is complete when:

- the PP-aware `0.3.7-draft` transition baseline is coherently recorded;
- WU-09/WU-10 completed semantics are represented without Tool/PP numeric coupling;
- known release/validator coupling points are inventoried and assigned to WU-12;
- ADR-0023 is accepted and WU-12 is pre-created with a precise implementation contract;
- any WU-11 release-surface preparation is internally consistent and focused tests pass;
- no final root adoption or release claim is made prematurely;
- a fresh exact WU-11 closure HEAD can be captured as the WU-12 entry baseline.

After WU-11 closes, the standing successor-entry rule may move pre-created WU-12 to `ACTIVE` if no new owner decision or architecture conflict remains. Entry does not itself authorize release execution.
