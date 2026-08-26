# WU-08 — Repository Self-Analysis and Package Baseline

> **Status:** ACTIVE  
> **Target Tool:** `0.3.7`  
> **Predecessor:** WU-07 — Safe Sequential Apply and Promotion (`COMPLETE / FOCUSED TEST VERIFIED`)  
> **Architecture authorities:** ADR-0017, ADR-0018, ADR-0019, ADR-0020  
> **Exact entry baseline:** `874fa80a508f5901647d6d2df132a95f0eadda49`  
> **Successor:** WU-09 — Independent Project Profile Identity Core

## 0. Purpose

Stabilize the real PTSIP repository and package baseline after WU-07 without expanding WU-08 into Project Profile identity implementation, Project Profile semantic migration, or final Tool release readiness.

WU-08 owns:

```text
real repository self-analysis
+ controlled migration fixture regression baseline
+ repository Responsibility Map completeness
+ package artifact ownership accuracy
+ architecture decisions required by later PP work
```

WU-08 does **not** own:

```text
pp.<major>.<minor> runtime implementation        -> WU-09
PP compatibility/migration/adoption             -> WU-10
Tool 0.3.7 final version/spec/release readiness  -> WU-11
```

This separation is mandated by ADR-0020 so failures can be localized to a narrow responsibility boundary while the long-term-maintainability rules remain active across all successor WUs.

## 1. Tool / Project Profile independence boundary

Tool and Project Profile versions are independent authorities.

```text
PTSIP Tool version
    !=
Project Profile Contract version
    !=
Project Profile Instance revision
```

Tool `0.3.7` does not authorize a Project Profile `0.3.7-draft` profile and does not authorize a `pp.1.01` migration by itself.

ADR-0019 establishes the future independent PP namespace:

```text
pp.<major>.<minor>
current intended PP contract: pp.1.01
```

with the normative major rule:

> **Project Profile major version은 governing lifecycle classification의 집합 또는 그 classification semantics가 변경되어 기존 프로젝트의 lifecycle 재분류를 요구할 수 있을 때 상승한다.**

WU-08 records this architecture decision only. Machine-readable PP identity implementation belongs to WU-09.

## 2. Real repository profile boundary

Until WU-09 and WU-10 implement and verify the independent PP identity/migration model, the actual repository source profile remains on its currently supported historical identity:

```text
ptsip.yaml
ptsip.version = 0.3.6-draft
ptsip.specification.revision = d6995ed232e845b88d8235b851e80ab54b7804ea
```

WU-08 MUST NOT:

- create `ptsip_0.3.7.yaml` merely because Tool `0.3.7` exists;
- rewrite `ptsip.yaml` to `0.3.7-draft` for numeric symmetry;
- rewrite `ptsip.yaml` to `pp.1.01` before WU-09/WU-10 implementation and authorization;
- mechanically relabel public example profiles;
- execute real-repository canonical profile promotion.

The real repository may be analyzed read-only while mutation/deletion/promotion behavior remains verified through controlled fixtures.

## 3. Inherited WU-07 migration verification

WU-07 already verified the mutation capability through controlled fixture repositories at exact SHA:

```text
68b29308e8531f93267acc2aec585f6e751021f1
```

Owner-dispatched workflow run:

```text
32825710974
```

produced:

```text
437 passed
1 failed
```

The sole failure was not a WU-07 migration-executor failure. It was the WU-08 repository self-profile coverage regression.

WU-08 may rely on the WU-07 focused migration evidence as predecessor evidence, while later PP-identity integration must be reverified in WU-10.

## 4. Repository self-profile regression

The inherited failure was:

```text
tests/ptsip/test_repository_self_profile_035.py::
test_repository_self_profile_is_valid_complete_and_revision_pinned
```

Repository validation reported 18 tracked files outside declared component/associated-artifact selectors.

The 18 files were exactly the new Tool `0.3.7` product subpackages:

```text
src/ptsip/evidence/**        4 files
src/ptsip/source_compat/**   4 files
src/ptsip/migration/**       10 files
```

The project owner selected the long-term-maintainability ownership model under ADR-0018:

```text
ptsip-evidence
ptsip-source-compat
ptsip-migration
```

as separate PRODUCT components rather than broadening `ptsip-core`.

This keeps future unclassified `src/ptsip/<new-subsystem>/` roots visible as architecture gaps instead of silently absorbing them into core ownership.

## 5. Verified self-profile resolution

Workflow run:

```text
32923347579
```

verified exact SHA:

```text
fc1f479d34cb9531180bdf111bfe0695ba0fc48b
```

and produced:

```text
439 passed
0 failed
repository validation warnings: []
responsibility_map_coverage.unassigned_count: 0
self-hosted/tooling-test: success
```

Therefore the inherited 18-file repository self-profile blocker is:

```text
CLOSED / VERIFIED
```

This run remains exact-SHA evidence for the repository baseline it verified. It is not final Tool `0.3.7` release-readiness authority because later WU-09/WU-10/WU-11 work changes the integrated source state.

## 6. Package artifact ownership baseline

ADR-0018 split three shipped product subsystems from `ptsip-core`, so wheel artifact evidence must preserve the same ownership.

The workflow classification order is now required to distinguish:

```text
ptsip/specdata/*       -> ptsip-embedded-contracts
ptsip/evidence/*       -> ptsip-evidence
ptsip/source_compat/*  -> ptsip-source-compat
ptsip/migration/*      -> ptsip-migration
ptsip/*                -> ptsip-core
vpms/*                 -> vpms
```

The workflow synchronization was implemented in commit:

```text
4329e48127003151a6f256320da236943e674911
```

The specific subsystem branches must precede the generic `ptsip/*` fallback so shipped subsystem files are not collapsed back into `ptsip-core`.

The workflow must also require the three ADR-0018 Product components to appear in the built wheel artifact evidence.

A later exact-SHA run must verify this workflow change before WU-08 completion.

## 7. Project Profile version architecture decision

ADR-0019 is accepted during WU-08 as architecture authority for successor work.

It defines three independent identities:

```text
Tool Version
Project Profile Contract Version
Project Profile Instance Revision
```

and establishes the PP grammar direction:

```text
pp.<major>.<minor>
```

WU-08 does not implement this grammar in the runtime/schema/transition engine. That work is intentionally delegated to WU-09.

Historical Tool-numbered Project Profile labels remain actual historical source identities until WU-10 provides explicit compatibility mappings. WU-08 must not rewrite history by pretending old files were originally published as `pp.*` versions.

## 8. Responsibility-segmented successor sequence

ADR-0020 establishes the post-WU-08 sequence:

```text
WU-08
Repository Self-Analysis and Package Baseline
        ↓
WU-09
Independent Project Profile Identity Core
        ↓
WU-10
Project Profile Compatibility, Migration, and Adoption
        ↓
WU-11
Tool 0.3.7 Final Regression, Specification Freeze, and Release Readiness
```

The larger number of WUs is intentional. Debugging scope and regression ownership are more important than minimizing WU count.

Long-term maintainability remains mandatory through:

- typed boundaries;
- explicit supported-version contracts/registries;
- independent focused verification;
- fail-closed compatibility;
- stable ownership between identity, migration, adoption, and release layers;
- no Tool/PP numeric coupling.

## 9. WU-08 non-goals

WU-08 MUST NOT implement or claim completion of:

- `pp.<major>.<minor>` parser/serializer;
- Tool-to-PP machine-readable compatibility registry;
- PP-aware transition ordering;
- `pp.0.00 -> pp.1.01` migration;
- PP-aware execution/promotion;
- real repository `pp.1.01` adoption;
- Tool package/runtime version bump to `0.3.7`;
- final Tool `0.3.7` Specification freeze;
- final release-readiness workflow authority.

Finding a defect in one of those future areas should be recorded for its owning successor WU rather than patched into WU-08 without changing responsibility ownership.

## 10. Completion gate

WU-08 is complete only when:

- the inherited 18-file self-profile blocker remains closed;
- real repository validation remains complete with no unassigned tracked-file ownership;
- ADR-0018 component ownership remains explicit and non-overlapping;
- the wheel artifact-evidence workflow attributes `evidence`, `source_compat`, and `migration` to their separate PRODUCT components;
- an exact-SHA workflow run verifies the artifact-evidence ownership update;
- existing controlled migration fixtures remain a valid predecessor baseline;
- ADR-0019 independent PP architecture is recorded without prematurely activating `pp.1.01`;
- ADR-0020 responsibility-separated WU sequence is recorded;
- actual repository Project Profiles remain free from unauthorized Tool-number-driven migration;
- no final Tool release-readiness claim is made.

When WU-08 is documented complete and the exact `dev/0.3.7` HEAD is freshly validated, WU-09 may automatically enter `ACTIVE` under the standing successor-entry authorization. Automatic entry is state only and does not itself authorize WU-09 implementation.
