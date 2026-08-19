# WU-04F — Enforced Conformance Consumes the Effective Responsibility Map

> **Status:** ACTIVE  
> **Parent work unit:** WU-04 — template catalog + deterministic materialization  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-04E — profile validation consumes the effective Responsibility Map  
> **Entry baseline:** `5d52e452ce48de4d0e0d8251d906c1e0f15f82c2`  
> **Bound Specification snapshot at entry:** `82abd09360df09a95fbbfb516855fa9ffb49f050`

## 1. Purpose

WU-04F removes the remaining source-mode/raw-profile dependency from Enforced Conformance.

The target pipeline is:

```text
source ptsip.yaml
    -> validate_profile()
    -> ValidationResult.resolved_profile
    -> ResolvedProfile.effective_payload
    -> conformance evaluators
```

Conformance must not interpret template/hybrid declarations itself. It consumes the already validated effective Responsibility Map produced by the validation/materialization boundary.

## 2. Consumer authority boundary

Conformance is an evaluator, not declaration authority.

It may consume:

- effective components and classifications;
- effective component selectors;
- effective associated-artifact coverage as reflected by profile validation;
- effective component dependency policy;
- resolution identity already exposed by validation;
- repository/evidence observations required by conformance rules.

It must not:

- infer or select templates;
- apply hybrid overrides itself;
- rewrite the source profile;
- translate historical `TOOLCHAIN` into a canonical class;
- reconstruct lifecycle ownership from paths or evidence;
- silently repair invalid effective maps.

## 3. Runtime handoff

WU-04E retains the successfully materialized runtime object at:

```text
ValidationResult.resolved_profile
```

This field is deliberately not emitted by `ValidationResult.as_dict()`. Public validation/conformance reports continue to expose serializable `details.resolution` and provenance summaries without duplicating the full effective payload.

For valid profiles, WU-04F consumers must use:

```text
validation.resolved_profile.effective_payload
```

rather than re-reading `ptsip.yaml` and inspecting top-level source `components`.

## 4. Base conformance integration

Primary target:

```text
src/ptsip/conformance.py
```

Required changes:

- remove raw YAML reload after successful profile validation;
- obtain components from the resolved effective payload;
- obtain `component_dependency_policy` from the resolved effective payload;
- run partition/dependency/artifact/language checks against effective components;
- preserve immutable Specification revision enforcement;
- remove the template/hybrid `ownership:materialization-required` blocking gap;
- remove `MATERIALIZED_COMPONENT_DECLARATIONS_REQUIRED` evaluator reasons;
- fail closed if a profile is reported valid but no resolved runtime view is available.

## 5. Complete conformance engine integration

Secondary target:

```text
src/ptsip/conformance_engine.py
```

The complete engine currently re-reads the source profile independently after invoking base conformance. WU-04F must remove that second raw-profile architecture read.

The complete engine must use effective components/policy for:

- declared dependency boundaries;
- source-language coverage;
- component ownership gaps;
- component dependency policy evaluation;
- artifact component identity checks inherited from base conformance;
- agent-decision comparison against declared component identities;
- independent build resolution;
- lifecycle evidence evaluation.

This does not move clarification/decision-writing semantics into WU-04F. It only makes existing conformance-time read/evaluation paths operate on the effective declared architecture.

## 6. Report behavior

The conformance report should continue to include:

```text
profile: ValidationResult.as_dict()
```

which already carries serializable resolution details when materialization succeeds.

No new public report field is required merely to expose the full effective payload.

Equivalent explicit/template/hybrid declarations should produce equivalent architecture-dependent conformance behavior, except for explanatory source-mode/template identity already present under profile validation details.

## 7. Regression transition

WU-04F updates canonical conformance fixtures that still generate Tool `0.3.4/0.3.5` profile state while claiming to test current conformance behavior.

Canonical current-tool fixtures must use:

```text
ptsip.version: 0.3.6-draft
responsibility_map.mode: explicit/template/hybrid
PRODUCT | DEVELOPMENT_TOOLING | DELIVERY | OPERATIONS | NEUTRAL_CONTRACT
product_to_nonproduct_runtime_dependency
nonproduct_in_product_package
independent_build_resolution
current SPEC_REVISION
```

Historical/legacy fixtures remain historical only where the test explicitly exercises legacy rejection or future migration behavior. They must not be silently accepted by canonical conformance.

## 8. Focused tests

WU-04F should establish at least:

- explicit conformance continues to use canonical effective components;
- template profile reaches dependency/artifact/build/lifecycle evaluators without a materialization-required gap;
- hybrid profile uses overridden/extended/removed effective components;
- effective component dependency policy is evaluated for template/hybrid where present;
- artifact evidence component IDs are matched against effective component IDs;
- explicit/template profiles with equivalent effective architecture yield equivalent architecture-dependent conformance results;
- the source declaration remains unchanged;
- invalid/materialization-failed profiles remain fail-closed and do not reach architecture evaluators.

## 9. Out of scope

WU-04F does not migrate:

- clarification answer parsing or projection writes — WU-04G;
- adoption read/write flows — WU-04G;
- topology migration semantics unless required only to consume current validation output — later integration/migration work;
- VPMS — WU-04H;
- Tool `0.3.5` legacy reader — WU-07;
- broad cross-mode downstream closure — WU-04I.

## 10. Combined WU-04E/F verification gate

Per `planning/0.3.6.md`, this stage shares one exact-SHA self-hosted verification run with the final WU-04E implementation.

Before WU-04G can be entered:

- WU-04F implementation changes and focused tests are complete;
- one self-hosted `tooling-test.yml` run is dispatched from the exact final E/F branch SHA;
- WU-04E focused validation tests pass in that run;
- WU-04F focused conformance tests pass in that run;
- any remaining repository failures are classified and shown not to invalidate E/F completion;
- if the shared run exposes an E regression, E is reopened;
- if it exposes an F regression, F remains ACTIVE.
