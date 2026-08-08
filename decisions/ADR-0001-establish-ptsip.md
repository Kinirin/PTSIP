# ADR-0001: Establish PTSIP

- **Status:** Accepted for draft specification
- **Date:** 2026-08-07
- **Decision:** Define Product–Toolchain SDK Isolation Policy (PTSIP) as the canonical project term and architecture policy.

## Context

Software projects can contain two fundamentally different SDK populations:

1. SDKs that belong to and may be packaged with the product;
2. SDKs and tools used to develop, validate, migrate, test, build, or release that product.

Conventional code-reuse pressure can cause these populations to share executable packages even though their packaging responsibility and lifecycle are different. This creates coupling that is difficult to reverse later.

Existing engineering concepts such as host/target separation, build-time/runtime separation, dependency isolation, and toolchain separation explain parts of the problem, but no single term was selected for this project's stronger SDK-governance policy.

## Decision

Adopt the term:

> **PTSIP — Product–Toolchain SDK Isolation Policy**

PTSIP classifies SDKs by purpose and lifecycle ownership and establishes a strong boundary between Product and Toolchain SDK planes.

The primary principle is **Purpose Before Reuse**.

PTSIP will be published as a versioned specification with:

- normative rules;
- terminology registry;
- governance;
- conformance requirements;
- reference architecture;
- machine-readable project profiles;
- coding-agent instructions;
- future automated validators.

## Consequences

### Positive

- Product packaging remains free of accidental development dependencies.
- Toolchain evolution is less constrained by Product compatibility.
- SDK ownership becomes explicit.
- coding agents receive deterministic architecture guidance.
- automated conformance checking becomes possible.

### Costs

- some apparently reusable executable code may remain separate;
- additional governance and classification work is required;
- contract drift must be controlled when implementations are separate;
- build environments and dependency manifests may require duplication.

## Non-claim

This ADR does not claim invention of host/target separation, build/runtime dependency separation, or toolchain isolation. PTSIP is the project's explicit synthesis and SDK-governance formulation of related established principles.
