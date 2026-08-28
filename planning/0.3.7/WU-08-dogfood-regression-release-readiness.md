# WU-08 — Repository Self-Analysis and Package Baseline

> **Status:** COMPLETE / VERIFIED  
> **Target Tool:** `0.3.7`  
> **Predecessor:** WU-07 — Safe Sequential Apply and Promotion (`COMPLETE / FOCUSED TEST VERIFIED`)  
> **Architecture authorities:** ADR-0017, ADR-0018, ADR-0019, ADR-0020  
> **Exact entry baseline:** `874fa80a508f5901647d6d2df132a95f0eadda49`  
> **Exact verification authority:** `56dd7399d2003892a2b0c02b23b5eb1aef63f527` via `tooling-test` run `32932963963`  
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

The exact WU-08 verification run confirmed the repository remained clean and read-only during clarification/gate checks. The before/after tracked-content fingerprint remained identical and the repository snapshot status was `STABLE`.

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

WU-08 relies on the WU-07 focused migration evidence as predecessor evidence. Later PP-identity integration must be reverified in WU-10.

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

Workflow run `32923347579` first verified exact SHA:

```text
fc1f479d34cb9531180bdf111bfe0695ba0fc48b
```

with:

```text
439 passed
0 failed
repository validation warnings: []
responsibility_map_coverage.unassigned_count: 0
self-hosted/tooling-test: success
```

WU-08 final baseline verification then revalidated the expanded documentation/architecture state and package ownership workflow at exact SHA:

```text
56dd7399d2003892a2b0c02b23b5eb1aef63f527
```

through owner-dispatched `tooling-test` run:

```text
32932963963
```

with:

```text
439 passed in 337.30s
0 failed
repository validation valid: true
repository validation errors: []
repository validation warnings: []
component conflicts: []
component unmatched selectors: []
associated-artifact conflicts: []
associated-artifact unmatched selectors: []
responsibility_map_coverage.unassigned_count: 0
self-hosted/tooling-test: success
```

Therefore the inherited 18-file repository self-profile blocker is:

```text
CLOSED / VERIFIED
```

The component-only partition lists governance-support files as component-unassigned, while the associated-artifact partition owns those same files. The combined `responsibility_map_coverage` is the governing completeness result and reports zero unassigned tracked files.

## 6. Package artifact ownership baseline

ADR-0018 split three shipped product subsystems from `ptsip-core`, so wheel artifact evidence must preserve the same ownership.

The verified workflow classification order is:

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

The specific subsystem branches precede the generic `ptsip/*` fallback, and the workflow explicitly requires all three ADR-0018 Product components to be present in built-wheel artifact evidence.

Exact-SHA run `32932963963` verified this updated logic at `56dd7399d2003892a2b0c02b23b5eb1aef63f527`. The Product Artifact evidence step completed successfully, including exact-snapshot binding, with wheel SHA-256:

```text
b6f118d4bd5f4810b13ad81c8161a7a2d1d7c83b556a3218cbcaa78f11ce1a38
```

The artifact-aware conformance outcome was `INCOMPLETE`, which is an accepted workflow outcome under the existing evidence contract when the product-artifact evaluator and exact-snapshot binding run successfully, the binding is valid, no `artifact-evidence:*` blocking gap remains, and no `PTSIP-PKG-001` non-conformance exists. This result is not represented as full conformance success.

Package build also produced and passed `twine check` for:

```text
ptsip-0.3.6-py3-none-any.whl
ptsip-0.3.6.tar.gz
```

The built wheel was force-reinstalled and its public CLI surfaces and VPMS smoke contract passed. Tool/package identity remains `0.3.6` at this WU boundary by design; final Tool `0.3.7` release identity belongs to WU-11.

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

Historical Tool-numbered Project Profile labels remain actual historical source identities until WU-10 provides explicit compatibility mappings. WU-08 does not rewrite history by pretending old files were originally published as `pp.*` versions.

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

WU-08 does not implement or claim completion of:

- `pp.<major>.<minor>` parser/serializer;
- Tool-to-PP machine-readable compatibility registry;
- PP-aware transition ordering;
- `pp.0.00 -> pp.1.01` migration;
- PP-aware execution/promotion;
- real repository `pp.1.01` adoption;
- Tool package/runtime version bump to `0.3.7`;
- final Tool `0.3.7` Specification freeze;
- final release-readiness workflow authority.

Finding a defect in one of those future areas belongs to its owning successor WU rather than WU-08.

## 10. Completion gate

WU-08 completion gate is satisfied:

- inherited 18-file self-profile blocker remains closed;
- real repository validation is complete with combined unassigned tracked-file ownership equal to zero;
- ADR-0018 component ownership is explicit and non-overlapping;
- wheel artifact evidence attributes `evidence`, `source_compat`, and `migration` to their separate PRODUCT components;
- exact-SHA workflow run `32932963963` verifies the artifact-evidence ownership update at `56dd7399d2003892a2b0c02b23b5eb1aef63f527`;
- controlled migration fixtures remain valid predecessor evidence from WU-07 and the full WU-08 regression remains green;
- ADR-0019 independent PP architecture is recorded without prematurely activating `pp.1.01`;
- ADR-0020 responsibility-separated WU sequence is recorded;
- actual repository Project Profiles remain free from unauthorized Tool-number-driven migration;
- no final Tool release-readiness claim is made.

WU-08 is therefore:

```text
COMPLETE / VERIFIED
verification authority: 56dd7399d2003892a2b0c02b23b5eb1aef63f527
workflow run: 32932963963
```

This closure-documentation commit occurs after the verified SHA and does not replace that exact-SHA verification authority.

## 11. Successor entry boundary

Under the standing successor-entry authorization, WU-09 may enter `ACTIVE` after this completion record is committed and the resulting exact `dev/0.3.7` HEAD is freshly validated.

Automatic successor entry changes entry state only. It does not authorize WU-09 architecture choices beyond ADR-0019/ADR-0020 or authorize implementation outside the WU-09 plan.