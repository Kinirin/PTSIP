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

**Developer policy entry:** Discover policy identities and paths from
`developer/policy/index.yaml` as a non-authoritative lookup aid. Read the
current policy documents directly, including `developer/policy/GOV/MPD-GOV-0006.yaml`
for lifecycle and governance source requirements. Confirm the referenced
policy's own effective status before using it as authority.

**Automatic existing-contract entry for developer policy work:** For a request
to inspect, author, interpret, convert, migrate, rewrite, or validate a
`developer/policy/**` policy, resolve the existing machine-contract route
**before** proposing new schemas or independent rule contracts:

```text
.ptsip/index.json                         namespaces.contracts.index
  -> .ptsip/contracts/index.json          policy
  -> .ptsip/contracts/policy/boundary.json
       -> routing_binding.data            policy_routes.json
       -> routing_binding.schema          Policy_Routing.schema.json
       -> rules_binding.selection_schema  Public_Structure_Rules.schema.json
       -> rules_binding.directory         {root_family}_rules.json
       -> rules_binding.responsibility_profile
```

Use the policy document's declared `Policy_Identity.policy_kind` and
`Policy_Identity.root_family`, and follow the registered routes and
family-selection contract; never select a family or schema merely by file
name, directory name, or resemblance to another Family. The current
policy public schema is registered through `policy_routes.json` at
`.ptsip/contracts/schemas/Policy/PTSIP_Public_Structure.schema.json`;
its Policy_Rules selection resolves within the existing
`.ptsip/contracts/schemas/Policy/` schema plane to the corresponding
`.ptsip/contracts/policy/Rules/{root_family}_rules.json`.
Resolve status, rule identities, conditions, and authorization through
their registered owners; a schema or validator result does not grant
policy approval or execution authority.

**Policy migration uses, rather than recreates, the registered contracts.**
For legacy machine policy -> human semantic review -> current machine policy,
use the existing selected schema and Family rules contract as the
serialization/validation target. Retired policy material is historical
evidence for review, not automatically effective authority. Do not default
to creating a new contract, a duplicate schema, a new independent rule body,
or a guessed Family structure. If the selected contract is missing,
incompatible, or insufficient for approved semantics, report the exact
unresolved contract/authority decision and fail closed for normative
mutation. The existing developer policy validators are routed through
`.ptsip/engine/developer/index.json`; their structural checks must not
be mistaken for semantic equivalence or approval.

**Temporary executable invocation for developer-policy contracts (entry guidance only):**
For a `developer/policy/**` task, resolve the current canonical policy
identity and registered contract route above first. To invoke the existing
PTSIP-backed repository-local checks from the repository root, use:

```powershell
go -C .ptsip/engine/developer run . --validator schema-check --root ../../..
go -C .ptsip/engine/developer run . --validator policy-rules --operation describe --family <registered-root-family> --root ../../..
go -C .ptsip/engine/developer run . --validator policy-check --root ../../.. --input <repository-relative-policy-path>
```

Replace the angle-bracket arguments using registered machine identities and
the exact target policy path; the commands are not a Family inference rule.
The first command checks limited contract/routing shape, the second describes
the selected Family contract, and the third validates the selected document
structure and references. These checks are non-mutating and do not assert
semantic equivalence, policy approval, Bootstrap Integrated Engine activation
or execution permission. Always prefer a current, registered machine invocation
if one becomes available; this Markdown neither registers an engine nor grants
an automatic runtime gate. Do not apply this developer-policy validator route
to `src/policy/**` Support Feature policy documents.

Do **not** use `developer/policy/registries/**` for current developer-policy
authority, schema, contract routing, lifecycle or approval. These retired
registry files are not active machine contracts; in particular,
`developer/policy/registries/public-policy-structure-governance.json`
does not impose current schema paths or reference obligations.

**Temporary reform authoring lookup (not policy authority):**
Before creating new repository policy, approval, schema, contract, automation,
plan, or registry files during the temporary reform, read the structured
`developer/Temporary_Record/legacy-dependency-replacement.json` via the
`LEGACY_REPLACEMENT_BLOCKERS / LG-04` temporary work binding. Apply its
directory-prefix/exact-file creation exclusions; do not infer permission
from a nonmatch. This is a disposable agent pre-write lookup only, not
an input to formal policy, runtime, verification or release contracts.
When the user directs removal of `developer/Temporary_Record/**`, remove
this temporary lookup and every link into that temporary root without
separate reference-validation; do not skip unrelated verification.

Do not reactivate the retired policy resolver, validators or legacy BOUND
semantics when no valid successor exists. An unresolved authoritative
policy or contract must fail closed rather than be reconstructed from a
retired registry or inferred by an agent.

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
