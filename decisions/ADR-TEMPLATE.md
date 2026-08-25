# ADR-NNNN — Short Decision Title

**Status:** Proposed  
**Date:** YYYY-MM-DD  
**Decision:** One-sentence statement of the accepted architecture decision.  
**Topic ID:** `stable_topic_id`  

Optional relationship metadata; remove entries that do not apply:

**Supersedes:** `ADR-NNNN`  
**Amends:** `ADR-NNNN`  
**Extends:** `ADR-NNNN`  
**Depends on:** `ADR-NNNN`  
**Applies to:** concise scope or work/version boundary  

## Context

Describe the architecture problem, constraints, evidence, and the reason a decision is required.

The context should be sufficient for a future maintainer or coding agent to understand the problem without reconstructing every earlier ADR in the lineage.

## Decision

State the architecture decision precisely.

If this ADR supersedes or amends an earlier ADR, make the effective scope self-contained:

- identify the semantics being replaced or changed;
- identify relevant semantics that remain preserved;
- avoid requiring the reader to traverse the complete historical chain merely to determine the current decision.

## Consequences

### Positive

- Describe the material benefits of the decision.

### Tradeoffs

- Describe the accepted costs, constraints, migration burden, or loss of flexibility.

## Rejected Alternatives

This section is required for new ADRs.

### Alternative name

Explain the materially plausible alternative and why it was not selected.

If no material alternative existed, state that explicitly instead of omitting this section.

## Affected Rules / Assets

Use this section when the decision affects Specification rules, schemas, registries, profiles, agent contracts, workflows, source boundaries, or other named assets.

- `asset-or-rule-id` — effect

If no named rule or asset is affected, state that explicitly or remove this optional section.

## Implementation Sequencing

Use this section only when the decision requires ordering constraints that later implementation work must preserve.

Do not turn the ADR into a general implementation plan; detailed execution planning belongs in the applicable planning documents.
