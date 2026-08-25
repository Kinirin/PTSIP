# ADR Governance

This directory stores Architecture Decision Records (ADRs) and the small set of governance/navigation files used to create and locate them.

## Responsibility split

The `decisions/` directory has four distinct responsibilities:

```text
decisions/
├─ README.md
│   ADR governance: how ADRs are operated
├─ INDEX.yaml
│   navigation registry: what topic exists, which ADR is current, and where it is
├─ ADR-TEMPLATE.md
│   authoring contract for new ADRs
└─ ADR-NNNN-*.md
    historical decision records: what was decided and why at that time
```

These responsibilities MUST remain separate. `README.md`, `INDEX.yaml`, and `ADR-TEMPLATE.md` are mutable current governance/navigation files. They are not historical ADR records and are not subject to the rule that Accepted ADRs remain unchanged merely to match newer terminology or templates.

## Historical ADR records

An Accepted `ADR-NNNN-*.md` records the architecture decision as accepted at that time.

Accepted ADRs MUST NOT be rewritten only to:

- adopt current-version terminology;
- match a newer ADR template;
- remove superseded reasoning;
- make historical wording uniform with current documentation.

When a later decision changes an earlier one, create a new ADR and express the relationship explicitly. Do not erase the older decision history.

Historical immutability MUST NOT force readers to traverse an entire supersession chain merely to determine current architecture. A replacing ADR should be self-contained for the scope it governs and should state which prior semantics it changes and which relevant semantics remain preserved.

## When an ADR is required

Create or update the decision history when a material architecture decision is accepted, including decisions that establish, replace, amend, extend, or materially constrain an architecture boundary.

Editorial changes, routine implementation details, test-only changes, release-note updates, and other changes that do not create a material architecture decision do not require a new ADR merely for documentation completeness.

## ADR numbering and filenames

New records use monotonically increasing four-digit IDs and a concise kebab-case slug:

```text
ADR-NNNN-short-decision-title.md
```

The greatest ADR number MUST NOT be interpreted as the current policy for every topic. ADR numbers identify records; current-topic routing is owned by `INDEX.yaml`.

## Decision relationships

Use explicit relationship vocabulary when a new ADR is connected to an earlier decision:

- `Supersedes` — replaces the earlier decision for the stated scope.
- `Amends` — changes only a stated part of an earlier decision.
- `Extends` — adds new semantics while preserving the earlier decision.
- `Depends on` — relies on an earlier decision that remains effective.

For `Amends` and other partial relationships, state the affected scope clearly enough that readers can distinguish replaced semantics from preserved semantics.

## Rejected Alternatives

`Rejected Alternatives` is required for new ADRs created under this governance model.

The section records materially plausible alternatives that were considered and why they were not selected. It exists to prevent maintainers and coding agents from repeatedly proposing an already-rejected architecture without understanding the earlier tradeoff.

If no material alternative existed, state that explicitly rather than silently omitting the section.

## Current-decision routing

`INDEX.yaml` is the canonical navigation registry for current ADR discovery.

Its responsibility is intentionally narrow:

```text
What is this topic?
        ↓
Which ADR is current for it?
        ↓
Where is that ADR file?
```

`INDEX.yaml` MUST NOT restate ADR rationale, Specification rules, implementation behavior, or historical lineage. Those responsibilities belong to the ADR record and applicable normative/implementation sources.

Topic identity in `INDEX.yaml` uses stable machine-readable topic IDs. Human-readable titles may evolve without requiring the stable topic ID to change.

When an accepted ADR changes which ADR is current for an indexed topic, the same architecture change MUST update `INDEX.yaml`. A stale current-ADR pointer is not acceptable.

## Template evolution

`ADR-TEMPLATE.md` defines the current format for newly created ADRs.

The template is updated in place as ADR governance evolves. A template change applies prospectively and MUST NOT trigger bulk rewriting of earlier Accepted ADRs.

## Authority boundary

ADRs preserve decision rationale and architecture history. They do not replace the applicable bound Specification, project-owned architecture declarations, Decision Authority state, observed evidence, or conformance evaluation.

When current normative behavior differs from historical ADR wording, the applicable current authority takes precedence; the historical ADR remains unchanged as history.
