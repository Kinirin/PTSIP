# ADR Governance

`decisions/` stores current machine-readable Architecture Decision Records and the small set of governance assets used to route and validate them.

## Canonical current form

Current ADR authority is YAML only:

```text
decisions/
├─ README.md
├─ INDEX.yaml
├─ ADR-TEMPLATE.yaml
├─ AUTHORITY-SCHEMA-REGISTRY.yaml
└─ ADR-NNNN-*.yaml
```

`README.md` is a human operating guide. It is not an Architecture Decision Record and is not machine authority.

Every `ADR-NNNN-*.yaml` contains exactly one `authority_contract` and exactly one `authority_semantics` object. One ADR therefore represents one Authority semantic. A record must resolve through `AUTHORITY-SCHEMA-REGISTRY.yaml` to one registered language-neutral schema definition before its semantics may be consumed automatically.

## Natural-language boundary

Opaque natural-language prose is forbidden inside `authority_semantics`.

Human rationale, commentary, tradeoffs, and explanatory text are not machine semantic authority. They may exist in a separate presentation/reporting surface or in Git history, but CORE automatic reasoning must not consume them to infer governance semantics.

The P03 migration removed the historical Markdown ADR files from the current tree. Each migrated YAML record contains `representation_migration.source_blob_sha`, which preserves the exact original Markdown blob identity in Git history without making historical prose a current automatic-reasoning input.

## Historical immutability

The earlier rule prohibiting changes to Accepted ADRs was temporarily relaxed only for the P03 representation migration from Markdown prose to machine-readable YAML.

That exception is consumed by this migration.

After migration:

```text
representation migration != semantic change
accepted semantic change -> new governance decision
```

An Accepted ADR must not be silently weakened, strengthened, or reinterpreted in place. Material semantic change requires a new ADR and an explicit `supersedes`, `amends`, `extends`, or `depends_on` relationship where applicable.

## Registry boundary

`AUTHORITY-SCHEMA-REGISTRY.yaml` is a Tool capability catalog. It is not Project Authority.

It maps:

```text
authority_type
+ stable schema_id
+ integer schema_version
    -> exact machine semantic schema definition
```

`schema_id` is logical identity, not a physical file path. Published schema versions are immutable. Semantic contents must never be used to guess or repair contract identity.

## Current-decision routing

`INDEX.yaml` owns stable current-topic routing.

It answers only:

```text
topic_id
    -> current ADR ID
    -> current YAML record path
```

The greatest ADR number is not implicitly current for every topic.

## Authority separation

Governance ADR authority remains distinct from:

```text
Evidence
Derived Fact
Normative Specification
Project Profile declaration
Distributed Owner Decision
Conformance result
```

A Governance Authority Projection may consume registered machine semantics in a later runtime implementation. Unsupported registered semantics must fail closed as Tool capability gaps rather than being reconstructed from natural-language history.
