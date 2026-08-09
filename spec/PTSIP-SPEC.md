# PTSIP Specification

**Name:** Product–Toolchain SDK Isolation Policy  
**Acronym:** PTSIP  
**Version:** 0.2.0-draft  
**Status:** Draft normative specification

## 1. Scope

PTSIP defines how a software project classifies, builds, packages, depends on, releases, validates, and evolves SDKs that belong either to the product or to the development toolchain.

PTSIP governs **architectural ownership and lifecycle**, not programming language, build system, repository topology, package manager, deployment platform, documentation layout, or development-tool directory layout.

A project MAY use one repository or multiple repositories. A project MAY use one programming language or several. These choices do not change PTSIP conformance provided the required boundaries are preserved.

## 2. Core model

PTSIP defines two primary planes and one shared contract category.

### 2.1 Product SDK Plane

The Product SDK Plane contains SDKs, libraries, schemas, clients, domain modules, plugins, runtime support modules, or other components whose primary responsibility belongs to the product and whose artifacts may be distributed, embedded, loaded, or depended on by the product.

### 2.2 Toolchain SDK Plane

The Toolchain SDK Plane contains SDKs and tools whose primary responsibility is the development lifecycle, including validation, migration, build generation, test tooling, code generation, release preparation, static analysis, repository transformation, compatibility auditing, and development automation.

### 2.3 Neutral Contract Artifact

A Neutral Contract Artifact is a declarative or generated representation of a contract that MAY be consumed by both planes without becoming a shared executable SDK dependency.

Examples include:

- schema definitions;
- interface description files;
- field registries;
- test vectors;
- protocol definitions;
- generated immutable manifests.

A Neutral Contract Artifact MUST NOT become a hidden shared runtime implementation.

### 2.4 External PTSIP Tooling

External PTSIP Tooling is an implementation used to inspect, pilot, validate, or report on a Consumer Repository without becoming part of that repository's Product or project-owned Toolchain implementation.

A package installed in an isolated Python environment, user-level package environment, CI tool image, or equivalent external development environment is outside the Consumer Repository classification scope unless the project intentionally vendors, embeds, packages, or takes lifecycle ownership of that tooling.

### 2.5 Component and decision status

A **PTSIP Component** is the architectural unit to which ownership is assigned. A component MAY be a package, module group, generated artifact group, executable, single build/release script, protocol bundle, or another project-defined unit whose purpose and lifecycle can be evaluated coherently.

The only PTSIP architectural classifications are:

- `PRODUCT`
- `TOOLCHAIN`
- `NEUTRAL_CONTRACT`

Inspection or agent tooling MAY use decision statuses such as `UNKNOWN`, `CONFLICT`, or `INCOMPLETE` while evidence is insufficient. Such statuses describe the state of a decision and MUST NOT be represented as additional architectural classifications or planes.

## 3. Foundational principle

### PTSIP-CORE-001 — Purpose Before Reuse

Every SDK, module, package, library, executable component, or equivalent PTSIP Component MUST be classified by **primary purpose and lifecycle ownership before reuse is considered**.

Code similarity, convenience, or DRY pressure MUST NOT by itself justify crossing the Product/Toolchain boundary.

When reuse conflicts with lifecycle isolation, lifecycle isolation SHOULD take precedence.

## 4. Normative rules

### PTSIP-CLS-001 — Mandatory classification

Every SDK or SDK-like component within PTSIP scope MUST ultimately be classified as one of:

- `PRODUCT`
- `TOOLCHAIN`
- `NEUTRAL_CONTRACT` for non-executable shared contract artifacts only.

Classification MUST be made at a component granularity capable of representing the repository's actual ownership boundaries. A directory root MAY be used as shorthand when ownership is uniform, but directory naming alone is not proof of classification.

Ambiguous ownership MAY be represented as an unresolved decision status during inspection or migration, but it MUST be resolved before the component is introduced as a shared dependency or before strict conformance is claimed.

### PTSIP-DEP-001 — Product-to-Toolchain runtime dependency prohibition

A Product SDK MUST NOT import, link, load, vendor, or otherwise depend on a Toolchain SDK as a runtime or shipped dependency.

### PTSIP-DEP-002 — Toolchain inspection is permitted

A Toolchain SDK MAY inspect, parse, validate, transform, generate, test, or package Product artifacts.

This permission does not convert a Toolchain SDK into a Product dependency.

A Toolchain-to-Product executable dependency is not automatically permitted merely because its direction is Toolchain-to-Product. Its purpose and dependency phase MUST still be consistent with the declared ownership and lifecycle boundary.

### PTSIP-DEP-003 — Cross-boundary executable sharing is denied by default

A shared executable package used directly by both Product and Toolchain planes SHOULD NOT be introduced unless an explicit architecture decision demonstrates that the package has a lifecycle independent of both planes and does not create release coupling.

If this condition cannot be demonstrated, separate implementations or a Neutral Contract Artifact SHOULD be preferred.

### PTSIP-PKG-001 — Packaging isolation

Toolchain SDK code and Toolchain-only dependencies MUST NOT be included in the product's deployable or distributable artifact unless the component has been reclassified through governance as Product-owned.

### PTSIP-BLD-001 — Independently resolvable build environments

Product and Toolchain planes MUST have independently resolvable build environments.

At minimum, a conforming project MUST be able to determine each plane's direct dependencies without relying on undeclared dependencies from the other plane.

A project SHOULD use separate dependency manifests, lock scopes, virtual environments, containers, workspaces, or equivalent mechanisms when the ecosystem supports them.

### PTSIP-BLD-002 — Independent buildability

A Product build MUST NOT require development-only Toolchain packages merely because they share a repository.

A Toolchain build MAY require Product source or Product contract artifacts as analysis inputs, but such inputs MUST be declared explicitly.

### PTSIP-LCY-001 — Lifecycle independence

Product SDK and Toolchain SDK versioning and release decisions MUST be independently governable.

A Toolchain-only change SHOULD NOT force a Product release unless the change produces or modifies a Product artifact that is itself release-relevant.

### PTSIP-LCY-002 — Compatibility ownership

Backward-compatibility requirements MUST be owned by the plane whose consumers require them.

Product compatibility requirements MUST NOT automatically prevent breaking changes inside a Toolchain-only interface that has no Product consumer.

### PTSIP-CMN-001 — No unclassified common package

A package named or treated as `common`, `shared`, `core`, `utils`, or equivalent MUST NOT be exempt from classification.

A generic name does not create neutral architectural ownership.

### PTSIP-CNT-001 — Contract-first cross-boundary reuse

When Product and Toolchain need the same semantic definition, projects SHOULD prefer a Neutral Contract Artifact over a shared executable implementation.

### PTSIP-INT-001 — Consumer Repository Non-Intrusion

External PTSIP Tooling MUST NOT require a Consumer Repository to create or adopt PTSIP-specific documentation directories, tooling directories, cache directories, report directories, or equivalent repository hierarchy solely so the tooling can operate.

Inspection and pilot operations MUST be read-only with respect to the Consumer Repository by default.

Any operation that writes or modifies Consumer Repository content MUST require an explicit user action or explicit write-enabled mode.

Tool-owned cache, pilot state, and generated reports SHOULD be stored outside the Consumer Repository by default. A project MAY voluntarily commit a PTSIP profile, report, or other PTSIP artifact according to its own repository conventions.

### PTSIP-SPC-001 — Specification Binding

A machine-readable PTSIP Project Profile used for automated conformance MUST identify the PTSIP specification version and canonical specification source that govern the profile.

For a mutable draft specification family, a validator or conformance report SHOULD record the exact immutable specification revision used for evaluation. Reference tooling implementing a draft specification MUST record the exact revision it implements.

An implementation MUST NOT silently evaluate a project against a different normative specification version or revision than the version/revision it declares support for.

### PTSIP-EVD-001 — Evidence snapshot integrity

Automated PTSIP evidence used for conformance MUST identify the Consumer Repository revision or equivalent snapshot being evaluated.

If the repository revision or observed tracked content changes during evidence collection, the evidence set MUST be marked invalidated, incomplete, or otherwise unsuitable for a strict conformance claim. A tool MUST NOT silently combine repository metadata from one snapshot with inventory, dependency, or artifact evidence from another snapshot and present the result as stable evidence.

A non-intrusion claim SHOULD be based on observable before/after repository state rather than a hard-coded assertion.

### PTSIP-EVD-002 — Declaration and observation are distinct

A PTSIP Project Profile declares intended ownership and policy. A declaration is not by itself proof that the repository behaves consistently with that declaration.

An automated validator MUST distinguish declared ownership from observed evidence and MUST report material contradictions such as dependency edges, packaging contents, or build behavior that violate the declaration or a normative PTSIP rule.

Agent or heuristic classifications MAY assist review, but they MUST NOT silently override an explicit project declaration or convert unresolved evidence into a conformance fact.

### PTSIP-EXC-001 — Explicit exceptions

Any intentional violation of a MUST or MUST NOT rule MUST be recorded as a PTSIP exception decision.

The exception record MUST identify:

- violated rule ID;
- reason;
- affected components;
- coupling introduced;
- owner;
- review or expiry condition;
- remediation or permanent-acceptance decision.

A project with an active violation MAY NOT claim strict PTSIP conformance unless the conformance profile explicitly allows that category of exception.

## 5. Dependency direction

The default allowed information flow is:

```text
Toolchain SDK  --->  Product source/artifact
     inspect / validate / generate / migrate

Product SDK    -X->  Toolchain SDK
     runtime/package dependency prohibited
```

A Neutral Contract Artifact MAY be consumed by both:

```text
                   Neutral Contract
                    /            \
                   v              v
            Product SDK      Toolchain SDK
```

The contract MUST remain declarative or otherwise non-owning with respect to executable lifecycle.

External PTSIP Tooling observes the Consumer Repository from outside its project-owned dependency graph unless explicitly vendored or adopted by the project:

```text
External PTSIP Tooling  --->  Consumer Repository
         inspect / pilot / validate
```

Dependency evidence SHOULD preserve both relationship type and lifecycle phase when they can be determined. Examples include `IMPORTS`, `LINKS`, `INVOKES`, `READS`, `GENERATES`, `PACKAGES`, `TESTS`, and `PUBLISHES`, with phases such as `RUNTIME`, `BUILD`, `TEST`, `RELEASE`, and `INSPECTION`. Unknown phase or dynamic resolution MUST remain explicitly unresolved rather than being guessed into a conformance result.

## 6. Reuse policy

PTSIP does not require intentional code duplication. It requires that reuse be subordinate to boundary ownership.

Acceptable reuse strategies include:

1. shared declarative contracts;
2. generation of separate Product and Toolchain implementations from one contract;
3. independent implementations validated against the same conformance vectors;
4. a separately governed third-party or platform dependency that neither plane owns.

Risky strategies include:

1. a project-local executable `shared` package imported by both planes;
2. Toolchain validation code imported into Product runtime for convenience;
3. Product runtime libraries imported into Toolchain solely to avoid defining a stable contract;
4. one dependency lock/environment whose accidental transitive state makes both planes build.

## 7. Repository topology

PTSIP does not require a monorepo or multirepo and does not prescribe `docs/`, `tools/`, `.ptsip/`, or any other PTSIP-specific directory.

The following is an illustrative topology only:

```text
repo/
├─ product/
├─ toolchain/
├─ contracts/
└─ optional-project-profile
```

A project's existing directory conventions remain project-owned. Directory layout alone is not sufficient for conformance; dependency and packaging behavior MUST agree with the declared boundary.

## 8. Project profile

A project MAY operate PTSIP inspection or pilot tooling without adding a PTSIP profile to the repository.

A project claiming **PTSIP Enforced Conformance** MUST provide a machine-readable PTSIP Project Profile or equivalent declaration containing at least:

- PTSIP specification version;
- canonical specification source;
- Product and Toolchain ownership declarations;
- Neutral Contract ownership declarations when applicable;
- dependency policy;
- packaging policy;
- build-environment policy;
- approved exceptions.

The reference profile supports two declaration forms:

1. **boundary-root shorthand** for repositories where ownership is uniform by root; and
2. **component declarations** for nested, mixed, file-level, or lifecycle-specific ownership.

A component declaration identifies a stable component ID, one of the three PTSIP classifications, one or more include selectors, a purpose, and optional lifecycle/packaging metadata such as shipped state, executable state, manifests, release owner, compatibility owner, consumers, and analysis inputs.

When component selectors overlap, the reference profile resolves ownership deterministically:

1. exact-file selectors take precedence over glob selectors;
2. selectors with greater literal/path specificity take precedence over broader selectors;
3. exclusions are applied before ownership selection;
4. an equal-specificity tie between different components is a validation error rather than an automatic choice.

A file not matched by a component declaration is not automatically a PTSIP violation because a repository may contain files outside PTSIP component scope. However, a project claiming coverage over such files MUST not hide them as implicitly classified.

The profile is a **declaration of intended ownership**, not evidence that the declared boundary is actually respected. Automated validation compares declaration with observed dependency, build, packaging, and lifecycle evidence.

The reference profile format is defined by `schemas/ptsip-profile.schema.json`.

The profile's physical location is not an architectural boundary. Implementations MAY use a conventional `ptsip.yaml` at the repository root, an explicit user-supplied path, or an equivalent project configuration mechanism.

## 9. Enforcement

A PTSIP validator SHOULD be capable of checking at least:

- project profile validity when a profile is supplied;
- specification binding, including immutable revision for draft evaluation when available;
- component classification and selector conflicts;
- dependency edges with unresolved/dynamic evidence preserved;
- shipped artifact contents;
- build manifest separation;
- undeclared cross-plane imports;
- active exception records;
- evidence snapshot integrity;
- observable non-intrusion evidence for external inspection/pilot operations.

The validator SHOULD report rule IDs in diagnostics.

Example:

```text
PTSIP-DEP-001 ERROR
product/sdk/schema.py imports toolchain/validation/common.py
```

External PTSIP Tooling implementing inspection or pilot analysis MUST preserve the read-only default required by `PTSIP-INT-001`.

A coding agent used during classification SHOULD receive bounded evidence for one component or decision at a time and SHOULD return a schema-constrained decision containing a decision status, optional PTSIP classification, confidence, evidence IDs, rationale, and counter-evidence. `UNKNOWN`, `CONFLICT`, and `INCOMPLETE` are valid decision states but not PTSIP classifications.

## 10. Non-goals

PTSIP does not prescribe:

- microservices versus monoliths;
- object-oriented versus functional design;
- a specific package manager;
- a specific build system;
- source repository count;
- coding style;
- test framework;
- CI provider;
- documentation hierarchy;
- development-tool directory hierarchy;
- installation of PTSIP tooling inside the Consumer Repository.

PTSIP also does not claim that every shared dependency is harmful. It governs **project-owned SDK responsibility boundaries**.

## 11. Status and novelty statement

PTSIP is a project-defined architecture policy. The acronym and exact rule set in this specification are defined by this project.

PTSIP does **not** claim invention of the underlying ideas of host/target separation, build-time/runtime separation, toolchain isolation, dependency isolation, or independent lifecycle management. Its contribution is the explicit combination of these ideas into an SDK-oriented governance and conformance model.
