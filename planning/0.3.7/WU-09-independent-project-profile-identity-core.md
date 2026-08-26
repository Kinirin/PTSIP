# WU-09 — Independent Project Profile Identity Core

> **Status:** ACTIVE  
> **Target Tool:** `0.3.7`  
> **Predecessor:** WU-08 — Repository Self-Analysis and Package Baseline (`COMPLETE / VERIFIED`)  
> **Architecture authorities:** ADR-0017, ADR-0019, ADR-0020, ADR-0021  
> **Exact entry baseline:** `173483ac06a052f388581464364f5f65033f5587`  
> **Predecessor verification authority:** `56dd7399d2003892a2b0c02b23b5eb1aef63f527` via `tooling-test` run `32932963963`  
> **Successor:** WU-10 — Project Profile Compatibility, Migration, and Adoption

## 0. Purpose

Implement Project Profile contract identity as a machine-readable version axis that is independent from Tool SemVer and independent from one repository profile instance revision.

WU-09 establishes the identity substrate only. It does not migrate the real repository profile and does not perform canonical profile promotion.

Canonical identity model:

```text
Tool Version
    example: 0.3.7

Project Profile Contract Version
    example: pp.1.01

Project Profile Instance Revision
    immutable revision/content identity of one concrete declaration
```

Entry into WU-09 is state only under the standing successor-entry authorization. It does not by itself authorize implementation choices beyond the accepted architecture boundaries.

## 1. Project Profile version grammar

Implement a dedicated typed PP version model for:

```text
pp.<major>.<minor>
```

Requirements:

- `pp` is mandatory and case-sensitive unless the normative Specification explicitly decides otherwise;
- major and minor compare numerically;
- canonical serialization renders minor with a minimum width of two decimal digits;
- examples: `pp.0.00`, `pp.1.01`, `pp.1.10`, `pp.2.00`, `pp.2.100`;
- invalid, malformed, or unsupported identities fail closed;
- Tool package SemVer parsers MUST NOT be reused as PP identity authority.

The governing major rule is:

> **Project Profile major version은 governing lifecycle classification의 집합 또는 그 classification semantics가 변경되어 기존 프로젝트의 lifecycle 재분류를 요구할 수 있을 때 상승한다.**

Minor changes represent Project Profile contract-semantic or structural changes within one governing lifecycle generation.

## 2. Identity separation contract

Tests and runtime APIs must prove these are independent operations:

```text
Tool 0.3.7 -> Tool 0.3.8
PP pp.1.01 -> pp.1.01
```

and:

```text
Tool 0.3.7 -> Tool 0.3.7
PP pp.1.01 -> pp.1.02
```

A Project Profile instance edit that remains valid under the same PP contract changes only its immutable instance revision/content digest, not its PP contract version.

## 3. Runtime/schema identity representation

WU-09 must define one canonical runtime representation of PP identity and synchronize the schema/specdata surfaces that encode the identity itself.

Avoid parallel representations that can disagree. Public schema, embedded runtime schema, parser model, diagnostics, and CLI-visible identity must derive from one explicit contract.

Where a registry is appropriate, use an explicit supported-PP registry or typed contract rather than scattering string comparisons across validators/readers.

## 4. Identity-transition classification

ADR-0021 introduces an explicit distinction between identity transition and semantic migration.

The runtime identity layer must be able to represent at least:

```text
IDENTITY_ONLY
SEMANTIC_MIGRATION
```

WU-09 owns the reusable classification/typing capability, not historical source-to-target mapping execution.

For the accepted `0.3.6-draft -> pp.1.01` bridge, the architecture records:

```text
transition class: IDENTITY_ONLY
semantic contract delta: NONE
```

The concrete historical mapping and repository transition behavior remain WU-10 responsibilities.

## 5. Tool-to-PP compatibility declaration

Introduce an explicit compatibility declaration at the identity layer.

It must answer at least:

```text
Can this Tool identify this PP contract?
Is the PP supported for canonical validation?
Is it supported only as a historical migration source?
Is it unsupported/unknown?
```

The declaration must not itself perform migration or reinterpret historical semantics. Those belong to WU-10.

Unknown PP identities MUST NOT silently fall back to the newest known contract.

## 6. Filename serialization boundary

Canonical in-document Project Profile identity remains:

```text
pp.1.01
```

Where a temporary profile filename must encode a PP identity, WU-09 defines the filename-token serialization contract separately:

```text
pp.1.01 -> pp1.01
```

so a new-generation temporary profile may be represented as:

```text
ptsip_pp1.01.yaml
```

The filename token is not a second Project Profile version syntax and must not be accepted as canonical `ptsip.version` text.

Legacy temporary-filename alias resolution and migration continuity belong to WU-10.

## 7. Release-note namespace contract

ADR-0021 selects the long-term-maintainable release-note layout while preserving historical paths.

Existing flat historical records remain where they are. New-generation records use:

```text
releasenote/tool/<tool-version>.md
releasenote/project-profile/<pp-version>.md
releasenote/specification/<spec-family>.md
```

WU-09 owns the identity/discoverability contract needed for this separation. It must not bulk-move published or historical release-note documents.

The future `releasenote/project-profile/pp.1.01.md` must explicitly disclose that the accepted `0.3.6-draft -> pp.1.01` bridge is identity-only and does not require users to redesign unchanged Project Profile declarations.

Final Tool release-note publication/freeze remains WU-11 responsibility.

## 8. Diagnostics

Provide stable fail-closed diagnostics for at least:

- malformed PP identity;
- non-canonical PP spelling where canonical text is required;
- unsupported PP contract;
- PP identity missing when required by the new contract;
- incompatible identity context;
- accidental Tool-version/PP-version conflation;
- misuse of a filename token as canonical PP identity.

Diagnostics should distinguish identity failure from semantic migration incompatibility so WU-10 failures are not misclassified as parser failures.

## 9. Backward-compatibility boundary

Historical Tool-numbered profile labels such as `0.3.4-draft` and `0.3.6-draft` remain source identities for compatibility readers until WU-10 maps them to explicit PP generations.

WU-09 MUST NOT silently rewrite historical files or pretend those historical labels were originally published as `pp.*` versions.

`pp.0.00` and `pp.1.01` are PP-generation identities defined by ADR-0019; historical source-label mapping and the accepted `0.3.6-draft <-> pp.1.01` bridge execution are WU-10 responsibilities.

## 10. Maintained example-profile identity integration

Once the WU-09 parser/schema/runtime surfaces can validate canonical `pp.1.01`, the maintained public example profiles are intended to switch identity to `pp.1.01` without redesigning their unchanged contract contents:

```text
profiles/example.ptsip.yaml
profiles/hybrid-python-package.ptsip.yaml
profiles/template-python-package.ptsip.yaml
```

Any stale comments or binding text that still describe `0.3.6-draft` as the active Project Profile identity must be updated consistently.

This example-surface update is not authority to migrate the real repository canonical `ptsip.yaml`.

## 11. Test structure

Focused tests should be grouped by identity responsibility and gradually follow the role/purpose-based test organization direction.

Minimum cases:

- parse valid PP versions;
- canonical serialization;
- numeric ordering;
- minor width preservation;
- malformed prefix/segments;
- unsupported identity;
- Tool version independent from PP version;
- PP version independent from Tool version;
- instance revision independent from PP version;
- `IDENTITY_ONLY` distinct from semantic migration;
- filename token serialization distinct from canonical PP serialization;
- public schema / embedded schema identity synchronization;
- compatibility-registry lookup behavior;
- no historical source relabeling;
- maintained example profiles validate under `pp.1.01` after their identity-only update.

## 12. Non-goals

WU-09 does not own:

- execution of the historical `0.3.6-draft -> pp.1.01` bridge;
- `pp.0.00 -> pp.1.01` semantic migration;
- legacy temporary profile alias handling;
- temporary profile or Final Point migration mechanics;
- repository canonical profile promotion;
- real repository adoption of `pp.1.01`;
- final Tool `0.3.7` package version bump;
- final Specification freeze or release handoff.

## 13. Completion gate

WU-09 is complete only when:

- PP identity has one typed parser/serializer model;
- canonical `pp.<major>.<minor>` representation is deterministic;
- Tool/PP/instance identities are independently represented and tested;
- identity-only versus semantic-migration transition classification is representable without performing migration;
- canonical PP identity and PP filename token serialization are distinct and deterministic;
- supported/unsupported PP identity handling fails closed;
- schema/specdata/runtime identity surfaces are synchronized;
- Tool-to-PP identity compatibility is explicit and testable;
- release-note namespace ownership is explicit without relocating historical records;
- historical Tool-numbered source labels remain unmodified and are not falsely treated as originally published `pp.*` identities;
- maintained example-profile surfaces can use `pp.1.01` without semantic redesign;
- focused identity tests pass at an exact source SHA;
- no real repository Project Profile migration is performed.

Completion of WU-09 authorizes automatic entry state into WU-10 under the standing successor-entry rule, but does not authorize WU-10 implementation by itself.
