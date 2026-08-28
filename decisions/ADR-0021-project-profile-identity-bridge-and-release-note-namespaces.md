# ADR-0021 — Project Profile Identity Bridge and Release-Note Namespaces

> **Status:** Accepted  
> **Date:** 2026-08-26  
> **Target Tool:** `0.3.7`  
> **Governing work units:** WU-09, WU-10, WU-11  
> **Related authorities:** ADR-0017, ADR-0019, ADR-0020

## 1. Decision

PTSIP adopts a long-term-maintainable separation between Tool release history, Project Profile contract history, and Specification history while preserving historical paths already used by prior releases.

Existing flat `releasenote/` records remain in place as historical records. New-generation records use separate namespaces:

```text
releasenote/
├─ README.md
├─ tool/
│  └─ <tool-version>.md
├─ project-profile/
│  └─ <pp-contract-version>.md
├─ specification/
│  └─ <spec-family>.md
└─ historical flat records retained in place
```

Published or historical release-note files are not bulk-moved merely to make the directory visually uniform.

## 2. Three independent release-history axes

The release-note structure reflects the independent version authorities already established by ADR-0019:

```text
Tool Version
Project Profile Contract Version
Specification family / immutable normative revision
```

A Tool release document MUST NOT substitute for a Project Profile contract note, and a Project Profile contract note MUST NOT be treated as a Tool release note.

For Tool `0.3.7`, intended new-generation paths are conceptually:

```text
releasenote/tool/0.3.7.md
releasenote/project-profile/pp.1.01.md
releasenote/specification/0.3.7-draft.md
```

Exact creation/finalization timing remains owned by the corresponding WU and release boundary.

## 3. `0.3.6-draft` to `pp.1.01` transition classification

The transition from historical Project Profile label `0.3.6-draft` to canonical Project Profile contract identity `pp.1.01` is classified as:

```text
IDENTITY_ONLY
```

For this bridge:

```text
components delta:             NONE
relationships delta:          NONE
associated_artifacts delta:   NONE
policies delta:               NONE
Responsibility Map delta:     NONE
lifecycle classification delta: NONE
```

The purpose is to separate Project Profile contract identity from Tool-numbered historical labels. It is not, by itself, a semantic Project Profile migration.

Accordingly, an existing Project Profile that is valid under `0.3.6-draft` does not require lifecycle redesign or reclassification merely because the canonical contract identity becomes `pp.1.01`.

Post-rewrite identity/schema validation is still required; `IDENTITY_ONLY` does not mean validation is skipped.

## 4. User-facing disclosure requirement

The `pp.1.01` Project Profile release note MUST clearly disclose that `pp.1.01` and historical `0.3.6-draft` have equivalent Project Profile contract semantics for the identity bridge covered by this ADR.

The note must make clear that users are not expected to re-review unchanged `components`, `relationships`, `associated_artifacts`, `policies`, or lifecycle classifications merely because the identity namespace changed.

Historical labels remain historical facts and MUST NOT be rewritten as though they were originally published as `pp.*` identities.

## 5. Canonical PP identity and filename token

The canonical contract serialization remains:

```text
pp.<major>.<minor>
```

Example:

```text
pp.1.01
```

When a temporary profile filename needs to encode that identity, the filesystem token is:

```text
pp1.01
```

so the new-generation temporary path is:

```text
ptsip_pp1.01.yaml
```

The absence of the dot between `pp` and the major number is a filename serialization rule only. It does not change the canonical in-document identity `pp.1.01`.

## 6. In-progress migration continuity rules

The PP identity transition MUST NOT discard or restart valid migration work that is already targeting historical `0.3.6-draft`.

### 6.1 Existing `ptsip_0.3.6.yaml`

If `ptsip_0.3.6.yaml` already exists as an active/in-progress target, the migration continues using that file. Its internal contract identity may be rewritten from:

```text
0.3.6-draft
```

to:

```text
pp.1.01
```

without creating a second `ptsip_pp1.01.yaml` target.

The legacy filename becomes an accepted alias path for the equivalent canonical PP target during that migration lineage.

### 6.2 No existing `ptsip_0.3.6.yaml`

If migration has not yet created a `0.3.6-draft` temporary target, the system MUST NOT create an obsolete intermediate solely to preserve the old Tool-numbered sequence.

It may create the actual current PP target directly:

```text
ptsip_pp1.01.yaml
```

subject to the normal migration/adoption authority and validation requirements.

### 6.3 Canonical `ptsip.yaml` already at `0.3.6-draft`

If canonical `ptsip.yaml` is already valid at historical identity `0.3.6-draft`, transition to `pp.1.01` is an in-place identity rewrite rather than a semantic migration:

```text
ptsip.version: 0.3.6-draft
        ↓ IDENTITY_ONLY
ptsip.version: pp.1.01
```

No new temporary target is required solely for this identity change.

## 7. Equivalent-target collision rule

Because `0.3.6-draft` and `pp.1.01` are equivalent identities for this bridge, repositories that contain both an active legacy target and a new PP target representing the same logical destination must fail closed until the ambiguity is resolved.

Conceptually:

```text
ptsip_0.3.6.yaml
ptsip_pp1.01.yaml
        ↓
DUPLICATE_EQUIVALENT_TARGET
```

The transition engine MUST NOT choose one implicitly.

## 8. Responsibility split

WU-09 owns the reusable identity substrate required to represent this architecture, including:

- typed `pp.<major>.<minor>` identity;
- canonical parser/serializer;
- filename-token representation where appropriate;
- Tool / PP / instance-revision independence;
- identity-transition classification representation;
- schema/runtime support for PP identity;
- release-note namespace contract and discoverability expectations.

WU-10 owns historical compatibility and transition behavior, including:

- explicit `0.3.6-draft` <-> `pp.1.01` identity-equivalence registration;
- active legacy temporary-profile alias handling;
- direct creation of `ptsip_pp1.01.yaml` when no legacy target exists;
- in-place canonical identity rewrite for already-current `0.3.6-draft` profiles;
- equivalent-target collision diagnostics;
- semantic migration of genuinely older Project Profile generations;
- adoption authority and migration execution.

WU-11 owns final Tool `0.3.7` integration, final release-note publication readiness, complete regression, Specification freeze, and release handoff.

## 9. Example profiles

The following maintained example surfaces are intended to adopt canonical `pp.1.01` once the WU-09 identity layer can validate that identity:

```text
profiles/example.ptsip.yaml
profiles/hybrid-python-package.ptsip.yaml
profiles/template-python-package.ptsip.yaml
```

Their Project Profile contract content is not to be redesigned merely for the identity change. Any comments or Specification-binding text that still claims `0.3.6-draft` as the active contract identity must be updated consistently when these examples are switched.

This example update is not authority to migrate the real repository canonical `ptsip.yaml`.

## 10. Rationale

This decision preserves historical stability while making future version ownership explicit. It also prevents a namespace change from being mistaken for a semantic migration, avoids restarting valid in-progress migration work, and provides a deterministic boundary for later PP generations where genuine contract semantics do change.
