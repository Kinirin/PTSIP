# PTSIP Reference Architecture

This document is **informative** unless a project profile adopts a recommendation as a required local rule. PTSIP does not require a Consumer Repository to use any directory shown here.

## 1. Illustrative project topology

```text
repository/
│
├─ product/
│  ├─ app/
│  └─ sdk/
│
├─ toolchain/
│  ├─ sdk/
│  └─ automation/
│
└─ contracts/
```

A real project MAY use completely different names and locations. In particular, PTSIP does not require `product/`, `toolchain/`, `contracts/`, `docs/`, `tools/`, `.ptsip/`, or a root-level profile file.

## 2. External PTSIP Tooling topology

The preferred Pilot/inspection model keeps PTSIP implementation and state outside the Consumer Repository:

```text
External development environment
│
├─ PTSIP tooling installation
│  └─ ptsip CLI / validator / pilot engine
│
└─ PTSIP local state
   ├─ cache/
   └─ pilots/

             read-only by default
                     |
                     v
              Consumer Repository
              └─ existing project topology unchanged
```

For a Python implementation, the tool MAY be installed in an isolated virtual environment, user-level tool environment, or other package-managed environment. The physical `site-packages` location is an implementation detail, not a PTSIP repository requirement.

## 3. Dependency model

```text
                    contracts
                   /         \
                  v           v
          product/sdk       toolchain/sdk
               |                 |
               v                 v
           product/app     validators/build tools

Forbidden default edge:
product/sdk  -X->  toolchain/sdk
```

External PTSIP Tooling may inspect either plane without becoming a project-owned dependency simply by being installed in a developer environment.

## 4. Separate build contexts

A strong implementation keeps at least dependency resolution separate. One possible Python arrangement is:

```text
product build context
  dependency manifest / lock / environment

toolchain build context
  dependency manifest / lock / environment
```

Equivalent structures are valid for npm, Gradle, Cargo, Go, Bazel, containers, or other systems.

The objective is not directory aesthetics. The objective is that Toolchain-only dependencies cannot appear in Product builds merely because both happen to be installed in one developer environment.

## 5. Shared semantics without shared executable ownership

When both planes must understand the same object model:

```text
contract/schema
      |             |
      v             v
Product adapter   Toolchain validator
```

This preserves one semantic source while allowing independent implementations and lifecycles.

## 6. Anti-pattern: shared executable common package

```text
             shared/common
             /           \
            v             v
      Product SDK     Toolchain SDK
```

This is not automatically forbidden in every possible case, but it is architecture-sensitive. It frequently creates release coupling and therefore requires explicit classification and ownership rather than being accepted merely as DRY refactoring.

## 7. Recommended ownership test

Before classifying a module, ask in order:

1. Why does this module exist?
2. Who consumes it?
3. Is it shipped with the product?
4. Which plane owns compatibility?
5. Which plane should trigger releases when it changes?
6. Can the semantic contract be shared without sharing executable code?
7. Would sharing create a cross-plane dependency edge?

The answer to question 1 has priority over superficial code similarity.
