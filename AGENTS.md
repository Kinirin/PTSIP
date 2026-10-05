# AGENTS.md

These instructions apply to coding agents working anywhere in this repository.

## Role of this file

`AGENTS.md` is an operational entry projection for coding agents.

It does not own normative policy semantics, canonical policy identity, authority precedence,
classification semantics, current planning state, release verification authority, or other
repository state. Those responsibilities belong to their canonical machine-readable owners.

Rules:

- Do not create, change, or infer normative semantics from this file.
- Do not store current branch, work-unit, specification, release, verification, or other mutable
  repository state in this file.
- Do not copy canonical policy bodies into this file.
- Prefer exact machine resolution over natural-language interpretation.
- Generated or cached agent artifacts are projections only and must not become authority.
- If a derived projection conflicts with its canonical source, the canonical source wins and the
  projection must be treated as stale.
- Unresolved or ambiguous routing fails closed for mutation.
- Bounded read-only discovery may be used for a genuinely unregistered capability, but discovery
  does not authorize mutation or create authority.

During the transition to generated projection, this checked-in file is limited to the bootstrap
routing contract below. Do not expand it with policy or state semantics that belong in a canonical
machine plane.

## Top-level machine entry routing

### 1. PTSIP capability use

When an operation uses PTSIP functionality in a repository, enter the repository-local PTSIP
machine plane first through:

`.ptsip/index.json`

Use the namespaces and machine contracts reachable from that index. Do not begin by scanning
narrative documentation, implementation source, or similarly named files to infer a PTSIP entry
point.

The `.ptsip/` plane is the repository-local entry for using PTSIP capabilities. It is not the
developer-policy authority for the Kinirin/PTSIP repository.

### 2. Kinirin/PTSIP repository development

When:

`repository == Kinirin/PTSIP`

and the operation changes or governs the repository development environment, development process,
planning, verification, release preparation, CI, branch management, repository maintenance, or
other developer-control-plane behavior, enter:

`developer/policy/`

Use the canonical developer Policy Resolver before loading broader policy or planning context:

```text
python -m developer.automation.policy_resolver resolve --scope <repository-path> --operation <READ|MODIFY|PLAN|VERIFY|RELEASE>
```

Use only the exact policies, sections, contracts, and implementation bindings returned by the
machine path. If the required developer operation cannot be resolved exactly, fail closed rather
than reconstructing the rule from this file, historical documents, naming similarity, or model
confidence.

### 3. Kinirin/PTSIP consumer-facing feature semantics

Use the Support Feature policy plane only when both conditions are true:

```text
repository == Kinirin/PTSIP
AND
operation changes PTSIP consumer-facing feature semantics
```

Enter through:

`src/policy/index.yaml`

Use only registered Support Feature policy and machine-contract relationships reachable from the
canonical Support Feature plane. Do not infer Support Feature authority from shared names,
developer-policy similarity, historical documents, or implementation proximity.

The Support Feature plane must not inherit developer-policy authority implicitly. If exact Support
Feature resolution required for a mutation is unavailable, fail closed.

## Cross-plane work

A request may contain work belonging to more than one plane. Do not collapse such work into one
inferred authority route.

Segment cross-plane work into independently resolvable work units. Resolve each segment through its
own canonical plane, preserve its authority boundary, and only combine segment results through
machine-defined execution or join semantics when those semantics are available.

Authority or authorization resolved for one segment must not be reused as implicit authority for a
different plane.

## Machine routing and capability discipline

Routing is capability-owned and machine-resolved.

- Capabilities declare their own routing eligibility.
- Operations use a closed registered taxonomy; agents must not invent operation identities to make
  routing succeed.
- Multiple eligible capabilities may exist, but selection of a single execution candidate must be
  performed by registered deterministic selection and conflict-resolution machinery.
- Registered conflict policy, not prose similarity or agent preference, resolves supported routing
  conflicts.
- Canonical taxonomy may have a shared core and plane-owned extensions. An extension does not become
  common-core semantics merely because similar names appear in multiple planes.
- Resolver implementations may differ by plane, but agent-facing resolution must follow the common
  resolver protocol and machine-selected contract graph.
- Conditional contract branches, execution decisions, replanning boundaries, and implementation
  bindings belong to machine contracts and compiled projections, not to this file.

## Derived routing, state, and agent projections

Derived artifacts exist to reduce repeated reasoning and repeated repository traversal. They do not
create new authority.

The intended flow is:

```text
canonical capability / policy / state owners
        ↓
deterministic validation and compilation
        ↓
derived routing / selected contract / state projections
        ↓
agent execution image
        ↓
AGENTS.md and coding-agent consumption
```

A derived global registry may contain preselected routing, resolver, and conditional contract
information, but it remains a reproducible non-authoritative projection of canonical sources.

Current-state caches are derived state only. They must not become the source of planning,
specification, policy, release, or verification truth. A stale or unrecoverable derived state must
be reconciled from canonical sources rather than interpreted by the agent.

When present, `.agent/` is a generated, non-authoritative agent execution cache. It may contain
operation-scoped compiled routing, contracts, state, and execution information for efficient agent
consumption. It must not define policy, create authority, or become a fallback source when canonical
resolution fails.

## Generated projection discipline

The target operating model is that `AGENTS.md` is generated from admitted canonical routing and
agent-projection sources rather than maintained as a second policy database.

Generation must preserve these boundaries:

```text
Canonical source
    owns semantics and authority

Derived registry / state / .agent
    optimizes deterministic machine consumption

AGENTS.md
    projects coding-agent entry guidance
```

A generated projection must not add, weaken, reinterpret, or silently repair canonical semantics.
If a required machine relationship is not yet representable, keep it unresolved and route it
through the bounded discovery/admission process instead of encoding a guessed rule in
`AGENTS.md`.
