# WU-04E — Profile Validation Consumes the Effective Responsibility Map

> **Status:** IMPLEMENTATION COMPLETE — combined WU-04E/F exact-SHA verification pending  
> **Parent work unit:** WU-04 — template catalog + deterministic materialization  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-04D — `ResolvedProfile`, effective-map digest, and derived provenance  
> **Entry baseline:** `2f0c7db2d20fb6d88ab5c4ab10707f50d486351f`  
> **Implementation completion baseline:** `b27938288aea712c52db5ee29c5c2f5a092cb325`  
> **Bound Specification snapshot at entry:** `82abd09360df09a95fbbfb516855fa9ffb49f050`

## 1. Purpose

WU-04E moves Project Profile validation from source-mode-specific partial behavior onto the Canonical Effective Responsibility Map produced by WU-04D.

The required validation pipeline is:

```text
source ptsip.yaml
    -> parse source document
    -> validate canonical source schema
    -> validate Tool / Specification binding
    -> validate source declaration mechanics
    -> deterministically materialize
    -> validate effective Responsibility Map semantics
    -> validate effective selector ownership / coverage
    -> return validation details including resolution identity
```

Template and hybrid declarations must no longer stop after structural validation with a warning that component-level validation requires a later materialization layer.

## 2. Validation authority boundary

Validation does not gain architecture authority.

The source Project Profile remains the project-owned declaration. `ResolvedProfile.effective_payload` is a deterministic runtime view used to evaluate the architecture that the project actually selected through explicit/template/hybrid declaration semantics.

Validation MUST NOT:

- infer or auto-select a template;
- rewrite source declarations into explicit form;
- repair dangling relationships or anchors;
- cascade hybrid removals;
- change lifecycle classifications;
- promote observed repository evidence into project declarations;
- serialize derived provenance back into `ptsip.yaml`.

## 3. Source validation before materialization

Before materialization, validation must continue to reject malformed source declarations.

This includes:

- YAML/JSON document parse failures;
- canonical schema failures;
- unsupported Tool/Specification source/version/revision binding;
- malformed source-mode structure;
- duplicate hybrid override IDs;
- one stable ID being both overridden and removed;
- malformed template reference syntax.

The source declaration must be valid enough to resolve deterministically before effective-map validation begins.

## 4. Materialization failure handling

`materialize_profile()` is the single mode-resolution entry point.

`TemplateMaterializationError` is a validation failure, not a warning or an internal exception leak.

Examples:

```text
unknown template ID
unknown immutable template revision
remove unknown template entity
replace and remove same stable ID
malformed stable-ID collection
```

Validation must fail closed and report the materialization defect as a profile error.

## 5. Effective-map semantic validation

After successful materialization, semantic Responsibility Map validation must operate on `ResolvedProfile.effective_payload` for all three source modes.

The same effective rules therefore apply to explicit, template, and hybrid declarations:

- component IDs unique;
- associated-artifact IDs unique;
- relationship IDs unique;
- component/artifact endpoint namespace collision rejected;
- relationship endpoints must exist;
- duplicate semantic relation edges rejected;
- associated-artifact anchor must be a component;
- associated artifact must have a typed relationship to its anchor;
- component dependency policy endpoints must exist;
- duplicate allow/deny policy relations rejected;
- same allow/deny relation conflict rejected.

A source mode must not change the semantic validity of an otherwise identical effective map.

## 6. Effective selector validation

Repository partitioning and selector checks must use effective components and associated artifacts, not only top-level source `components`.

Therefore template/hybrid profiles must receive the same checks currently available to explicit profiles:

- component selector partition;
- equal-specificity component ownership conflicts;
- unmatched component selectors;
- associated-artifact selector partition;
- associated-artifact equal-specificity conflicts;
- component/artifact tracked-path overlap;
- effective Responsibility Map coverage;
- repository scan warnings where applicable.

Unassigned tracked files remain a warning rather than an automatic PTSIP violation.

## 7. Validation details / resolution identity

When materialization succeeds, `ValidationResult.details` exposes a stable serializable resolution summary:

```text
resolution:
  source_mode
  materialized
  template: {id, revision} | null
  effective_map_digest
```

Derived provenance is also exposed as explanatory details while remaining clearly non-authoritative.

For downstream in-process consumers, `ValidationResult.resolved_profile` retains the already materialized `ResolvedProfile` runtime object. This field is intentionally omitted from `ValidationResult.as_dict()` so public validation reports continue to expose only serializable validation details rather than duplicating the effective payload.

The validation result does not replace or mutate the source profile with the effective payload.

## 8. Existing implementation transition

Primary implementation target:

```text
src/ptsip/validation/profile.py
```

Removed behavior:

```text
template/hybrid
    -> schema/binding valid
    -> warning: materialization layer required
    -> skip component-level validation
```

Implemented behavior:

```text
explicit/template/hybrid
    -> source validation
    -> materialize_profile()
    -> common effective semantic validation
    -> common effective selector validation
    -> retain ResolvedProfile for downstream in-process consumption
```

The existing semantic validation logic is shared over the explicit-form effective map. Source-only hybrid declaration mechanics remain a separate pre-materialization check.

## 9. Focused regression scope

WU-04E focused tests cover:

- valid template profile fully validated against effective components;
- valid hybrid profile fully validated against override/removal result;
- template/hybrid no longer emit the old “materialization required” warning;
- unknown template revision becomes a validation error;
- dangling effective relationship endpoint fails validation;
- dangling effective associated-artifact anchor fails validation;
- effective component dependency policy allow/deny conflict fails validation;
- effective selectors participate in repository partition/coverage details;
- explicit/template declarations with the same effective map have equivalent digest and partition results;
- validation details expose source mode, exact template identity where applicable, and effective-map digest;
- caller/source profile remains unchanged.

The exact-SHA run `32227306170` at source SHA `33e26cebd845970c6e52fb0a9194a272f0e50228` executed the first WU-04E focused tranche without any `test_profile_validation_036.py` failure and produced `230 passed / 37 failed` repository-wide. The additional completion cases and downstream runtime handoff added after that run are intentionally deferred to the shared WU-04E/F exact-SHA verification run authorized in the master plan.

## 10. Out of scope

WU-04E does **not** migrate:

- conformance evaluation — WU-04F;
- clarification/adoption reads — WU-04G;
- VPMS bridge — WU-04H;
- broad cross-mode downstream regression closure — WU-04I;
- Tool `0.3.5` legacy profile reading/migration — WU-07 and later.

Conformance must consume the resolved/effective validation surface in WU-04F rather than teaching itself separate template/hybrid semantics.

## 11. Completion gate

WU-04E implementation gate is satisfied because:

- source schema and Specification binding are checked before materialization;
- materialization failures are surfaced as fail-closed validation errors;
- all source modes receive common semantic validation over the effective map;
- all source modes receive common effective selector/coverage validation;
- the old template/hybrid “materialization required” warning is removed;
- validation details expose deterministic resolution identity;
- the in-process validation result retains the resolved profile for the next consumer stage;
- focused validation tests cover explicit/template/hybrid behavior and required failure cases.

Per the maintainer-directed combined verification tranche recorded in `planning/0.3.6.md`, exact-SHA verification of the final WU-04E implementation is deferred and shared with WU-04F. WU-04E is not to be described as exact-SHA verified until that shared run succeeds. If the shared run reports an E regression, this stage reopens before WU-04G can be entered.
