# WU-04F — Enforced Conformance Consumes the Effective Responsibility Map

> **Status:** COMPLETE — exact-SHA verified in combined WU-04E/F run  
> **Parent work unit:** WU-04 — template catalog + deterministic materialization  
> **Entry branch:** `tool-0.3.6-lifecycle-ownership`  
> **Entry predecessor:** WU-04E — profile validation consumes the effective Responsibility Map  
> **Entry baseline:** `5d52e452ce48de4d0e0d8251d906c1e0f15f82c2`  
> **Implementation completion baseline:** `991cecb58be1c6bfb063131ed04721015318e17f`  
> **Verification source SHA:** `48b75e699a592703e4e03a8462131e4932103677`  
> **Verification run/job:** `32240740753 / 96030499443`  
> **Bound Specification snapshot at entry:** `82abd09360df09a95fbbfb516855fa9ffb49f050`

## 1. Purpose

WU-04F removes the remaining source-mode/raw-profile dependency from Enforced Conformance.

The implemented pipeline is:

```text
source ptsip.yaml
    -> validate_profile()
    -> ValidationResult.resolved_profile
    -> ResolvedProfile.effective_payload
    -> conformance evaluators
```

Conformance no longer interprets template/hybrid declarations itself. It consumes the already validated effective Responsibility Map produced by the validation/materialization boundary.

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

For valid profiles, WU-04F consumers use:

```text
validation.resolved_profile.effective_payload
```

rather than re-reading `ptsip.yaml` and inspecting top-level source `components`.

## 4. Base conformance integration

Primary target:

```text
src/ptsip/conformance.py
```

Implemented changes:

- removed raw YAML reload after successful profile validation;
- components come from the resolved effective payload;
- `component_dependency_policy` comes from the resolved effective payload;
- partition/dependency/artifact/language checks run against effective components;
- immutable Specification revision enforcement reads the preserved source binding;
- removed the template/hybrid `ownership:materialization-required` blocking gap;
- removed `MATERIALIZED_COMPONENT_DECLARATIONS_REQUIRED` evaluator reasons;
- added fail-closed `profile:resolution-unavailable` handling if validation is ever valid without a resolved runtime object;
- added defensive `profile:effective-components-missing` handling for impossible/invalid effective component absence.

## 5. Complete conformance engine integration

Secondary target:

```text
src/ptsip/conformance_engine.py
```

The complete engine no longer performs a second raw-profile architecture read after invoking base conformance.

It uses effective components/policy for:

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

The conformance report continues to include:

```text
profile: ValidationResult.as_dict()
```

which carries serializable resolution details when materialization succeeds.

No public report field exposes the entire effective payload merely for convenience. The effective runtime object remains an in-process handoff.

Equivalent explicit/template/hybrid declarations are expected to produce equivalent architecture-dependent conformance behavior, except for explanatory source-mode/template identity already present under profile validation details.

## 7. Regression transition

WU-04F updated canonical conformance fixtures that were still generating Tool `0.3.4/0.3.5` profile state while claiming to test current conformance behavior.

Current-tool conformance fixtures use:

```text
ptsip.version: 0.3.6-draft
responsibility_map.mode: explicit/template/hybrid
PRODUCT | DEVELOPMENT_TOOLING | DELIVERY | OPERATIONS | NEUTRAL_CONTRACT
product_to_nonproduct_runtime_dependency
nonproduct_in_product_package
independent_build_resolution
current SPEC_REVISION
```

Updated current-tool regression surfaces include:

```text
tests/ptsip/test_profile_correctness_023.py
tests/ptsip/test_conformance_030.py
tests/ptsip/test_conformance_engine_030.py
tests/ptsip/test_merge_gate_remediation_030.py
tests/ptsip/test_remaining_030.py
```

Historical or intentionally conflicting inputs remain historical where the test actually exercises rejection/review behavior. For example, an agent decision that claims `TOOLCHAIN` may remain as conflicting evidence; canonical Project Profiles themselves do not emit that class.

## 8. Focused tests

Focused contract:

```text
tests/ptsip/test_conformance_effective_map_036.py
```

It covers:

- template profiles reach conformance evaluators without a source-mode/materialization branch;
- hybrid override/removal results are the component identities consumed by conformance;
- effective artifact producer/component identity is recognized;
- explicit/template equivalent effective maps produce equivalent architecture-dependent base conformance results;
- invalid template bindings remain fail-closed before architecture evaluation.

WU-04E focused validation coverage remains in:

```text
tests/ptsip/test_profile_validation_036.py
```

Repository search after the implementation found no remaining `materialization-required`, `MATERIALIZED_COMPONENT_DECLARATIONS_REQUIRED`, or `yaml.safe_load` raw-profile path in the conformance integration targets.

## 9. Combined exact-SHA verification

Final verification was shared with WU-04E as authorized by the master plan.

```text
workflow run: 32240740753
job:          96030499443
source SHA:   48b75e699a592703e4e03a8462131e4932103677
Python:       3.14.3
pytest:       260 passed / 13 failed
```

The workflow confirmed exact checkout of `48b75e699a592703e4e03a8462131e4932103677` and ran the full repository pytest collection. `tests/ptsip/test_conformance_effective_map_036.py` did not appear in the exhaustive 13-failure summary, so the WU-04F focused contract passed at the exact verification SHA. The same is true for WU-04E's `test_profile_validation_036.py`.

The remaining 13 failures were reviewed:

```text
3  decision/clarification-control expectations still using historical TOOLCHAIN/lifecycle_owner
1  pilot/evidence fixture still using a pre-0.3.6 Project Profile
1  lifecycle-evidence regression expectation contradicting the current authority rule
1  agent-decision conflict expectation belonging to later decision integration
7  topology/legacy-profile migration fixtures still bound to old revision/schema semantics
```

The lifecycle-evidence failure does not demonstrate an effective-map conformance regression. Current `evaluate_lifecycle_evidence()` explicitly treats release-like workflow naming/trigger scope as evidence rather than lifecycle authority; an unscoped release-like workflow is recorded as an observation and does not itself create a classification failure. The old test expected the opposite behavior.

None of the 13 failures show that conformance re-read source-mode architecture, failed to consume `effective_payload`, or mishandled template/hybrid component identity. They belong to WU-04G or later integration/migration/regression work.

The repository-wide workflow still concluded failure, so no `self-hosted/tooling-test` success commit status was recorded for the SHA and Tool `0.3.6` is not globally regression-clean. This does not invalidate the stage-scoped E/F completion gate previously authorized by the master plan.

## 10. Out of scope

WU-04F does not migrate:

- clarification answer parsing or projection writes — WU-04G;
- adoption read/write flows — WU-04G;
- topology migration semantics — later integration/migration work;
- VPMS — WU-04H;
- Tool `0.3.5` legacy reader — WU-07;
- broad cross-mode downstream closure — WU-04I.

Stale tests whose purpose belongs to clarification/decision projection, pilot/adoption, topology migration, or legacy profile interpretation remain future-stage debt rather than being weakened merely to make this stage green.

## 11. Completion gate

WU-04F is COMPLETE because:

- conformance consumes `ValidationResult.resolved_profile.effective_payload`;
- base and complete conformance no longer re-read raw source architecture;
- source-mode/materialization-specific conformance branches are removed;
- template/hybrid component and artifact identities reach the same evaluators as explicit mode;
- equivalent explicit/template effective maps produce equivalent architecture-dependent focused results;
- invalid materialization remains fail-closed;
- the final focused contract passed at exact SHA `48b75e699a592703e4e03a8462131e4932103677` in run `32240740753`;
- remaining repository failures were classified and do not invalidate the E/F effective-map boundary.

WU-04G may be entered only in a later session after a fresh branch-HEAD read and creation of its own stage document. It has not been entered by this completion record.
