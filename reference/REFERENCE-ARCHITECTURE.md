# PTSIP Reference Architecture

This document is informative unless a project profile adopts a recommendation as a required local rule.

## 1. Reference topology

```text
repository/
│
├─ product/
│  ├─ app/
│  └─ sdk/
│     ├─ core/
│     ├─ schema/
│     ├─ domain/
│     ├─ client/
│     └─ integration/
│
├─ toolchain/
│  ├─ sdk/
│  │  ├─ validation/
│  │  ├─ migration/
│  │  ├─ build/
│  │  ├─ release/
│  │  └─ test/
│  └─ automation/
│
├─ contracts/
│  ├─ schema/
│  ├─ registry/
│  └─ test-vectors/
│
├─ ptsip.yaml
└─ docs/
```

## 2. Dependency model

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

## 3. Separate build contexts

A strong implementation keeps at least dependency resolution separate:

```text
product/
  pyproject.toml
  .venv-product/

toolchain/
  pyproject.toml
  .venv-toolchain/
```

Equivalent structures are valid for npm, Gradle, Cargo, Go, Bazel, containers, or other systems.

The objective is not directory aesthetics. The objective is that Toolchain-only dependencies cannot appear in Product builds merely because both happen to be installed in one developer environment.

## 4. Shared semantics without shared executable ownership

When both planes must understand the same object model:

```text
contracts/schema.yaml
      |             |
      v             v
Product adapter   Toolchain validator
```

This preserves one semantic source while allowing independent implementations and lifecycles.

## 5. Anti-pattern: shared executable common package

```text
             shared/common
             /           \
            v             v
      Product SDK     Toolchain SDK
```

This is not automatically forbidden in every possible case, but it is architecture-sensitive. It frequently creates release coupling and therefore requires explicit classification and ownership rather than being accepted merely as DRY refactoring.

## 6. Recommended ownership test

Before placing a module, ask in order:

1. Why does this module exist?
2. Who consumes it?
3. Is it shipped with the product?
4. Which plane owns compatibility?
5. Which plane should trigger releases when it changes?
6. Can the semantic contract be shared without sharing executable code?
7. Would sharing create a cross-plane dependency edge?

The answer to question 1 has priority over superficial code similarity.
