# ADR-0002: External Tooling and Consumer Repository Non-Intrusion

- **Status:** Accepted for draft specification
- **Date:** 2026-08-09
- **Decision:** PTSIP tooling is external architecture-governance tooling by default and MUST NOT impose PTSIP-specific directory hierarchies on Consumer Repositories solely to operate.

## Context

The first Pilot design initially placed PTSIP-specific scanners under a Consumer Repository's `tools/` hierarchy and Pilot reports under a `docs/` hierarchy. This creates an unintended architectural obligation: every adopting repository would have to reshape its own documentation or tooling conventions merely to use PTSIP.

This conflicts with PTSIP's purpose-based ownership model. The Consumer Repository owns its repository topology. PTSIP should govern Product/Toolchain SDK boundaries without taking ownership of unrelated directory conventions.

A second gap was identified in the 0.1.0-draft profile: it declared a PTSIP version but did not identify the canonical specification source used to interpret that version.

## Decision

1. Define **Consumer Repository**, **External PTSIP Tooling**, **Consumer Repository Non-Intrusion**, and **Specification Binding**.
2. Add `PTSIP-INT-001` requiring external inspection and Pilot operations to be read-only by default and prohibiting required PTSIP-specific repository hierarchies solely for tool operation.
3. Add `PTSIP-SPC-001` requiring machine-readable profiles used for automated conformance to identify canonical specification source and version.
4. Permit External PTSIP Tooling to be installed in an isolated/user development environment, including Python package environments, without automatically becoming part of the Consumer Repository's Product or project-owned Toolchain plane.
5. Keep tool-owned cache, Pilot state, and generated reports outside the Consumer Repository by default.
6. Allow projects to voluntarily commit profiles or reports wherever their own conventions require.
7. Separate the release lifecycle of the PTSIP specification from the release lifecycle of reference tooling.

## Consequences

### Positive

- PTSIP can be adopted without restructuring existing repositories.
- Repository ownership remains with the adopting project.
- Pilot and inspection tooling can operate safely as read-only external tooling.
- Python tooling can be distributed independently, including through PyPI.
- Automated conformance can identify which canonical PTSIP specification it evaluated.

### Costs

- Tooling needs an external state/cache location.
- A validator must distinguish an externally installed tool from a project-vendored copy.
- Specification/tool version compatibility must be tracked explicitly.
- Projects that want persistent profiles must choose their own profile/configuration location or use the reference convention.

## Compatibility

This is a backward-compatible normative addition to the experimental 0.x specification family and therefore advances the draft specification from `0.1.0-draft` to `0.2.0-draft` under the existing governance versioning policy.

## Affected rules

- `PTSIP-INT-001` — new
- `PTSIP-SPC-001` — new
