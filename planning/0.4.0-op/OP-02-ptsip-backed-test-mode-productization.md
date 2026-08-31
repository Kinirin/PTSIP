# OP-02 — PTSIP-backed Test Mode Productization

> **Status:** DRAFT / OPTIONAL PRODUCTIZATION PLAN  
> **Classification:** `OPTIONAL / NON-BLOCKING`  
> **Reference implementation precedent:** `Kinirin/turbo-system`  
> **Reference surfaces:** `.github/test-mode/`, `.github/workflows/tooling-test.yml`, repository `ptsip.yaml`  
> **Implementation authorization:** not granted by this planning document

## 1. Product objective

PTSIP should make the successful `turbo-system` development-verification pattern reusable by other Consumer Repositories.

The intended product capability is not "copy this repository's GitHub Actions files". It is a supported PTSIP development orchestration feature in which the Project Profile declares durable verification responsibility and a separate Test Mode execution declaration maps that responsibility to focused runnable verification.

Target relationship:

```text
Project Profile / ptsip.yaml
    = project-owned verification responsibility and architecture metadata
             ↓
component_ref
             ↓
Test Mode registry
    = execution selection and runtime declaration
             ↓
PTSIP Test Mode resolver / validator
             ↓
local execution plan or CI matrix
             ↓
GitHub Actions adapter or other execution adapter
```

The feature should let users benefit from PTSIP architecture declarations in day-to-day development work without forcing a coding agent or maintainer to reconstruct repository test ownership manually.

## 2. Why `turbo-system` is a strong precedent

The current `Kinirin/turbo-system` implementation demonstrates a useful vertical slice:

1. `ptsip.yaml` uses an explicit Responsibility Map with verification components carrying `classification`, `roles`, `include`, and `purpose`.
2. `.github/test-mode/test_modes.yaml` declares execution-oriented information such as `component_ref`, automatic/manual selection, watch paths, Python version, pytest targets, dependencies, environment, and timeout.
3. `.github/test-mode/resolve_test_modes.py` loads both the Test Mode registry and `ptsip.yaml`.
4. The resolver rejects a `component_ref` that does not exist in the Project Profile.
5. The resolver rejects a referenced component that does not carry the `VERIFICATION` role.
6. Changed files drive focused mode selection; control-plane changes select all automatic modes.
7. The normalized CI matrix derives `classification`, `roles`, and `purpose` from the referenced PTSIP component instead of copying that architecture metadata into the Test Mode registry.
8. `.github/workflows/tooling-test.yml` resolves modes first, then executes the selected matrix on self-hosted Windows runners and records PTSIP provenance in the job summary.

This is a concrete example of PTSIP moving beyond passive architecture documentation into architecture-backed development verification while retaining the authority firewall.

## 3. Governing separation

The reusable feature must preserve:

```text
Verification Responsibility
    ≠ Physical Test Location
    ≠ CI Execution Mode
    ≠ Watch-path heuristic
```

Authority responsibilities:

```text
ptsip.yaml
    = project truth / verification responsibility / architecture authority

Test Mode registry
    = execution declaration

resolver
    = deterministic validation and selection

workflow / local runner
    = execution mechanism
```

The rejected direction is:

```text
test path or watch glob
    ↓
infer lifecycle ownership or verification architecture
```

Watch paths may select execution. They must never manufacture or override Project Profile authority.

## 4. Productization scope

The planned reusable capability should eventually provide the following surfaces.

### 4.1 Test Mode registry contract

A stable, versioned registry/schema for mode declarations with fields comparable to:

```text
id
component_ref
selection
watch
execution
runtime / language version
targets
arguments
dependencies
environment
timeout
```

Architecture metadata such as component classification, roles, purpose, lifecycle ownership, or other Project Profile facts should not be duplicated as independent Test Mode authority.

### 4.2 Project Profile-backed semantic validation

PTSIP should be able to validate that:

- every `component_ref` resolves to the current effective Project Profile;
- the referenced component is valid for verification use, including the required `VERIFICATION` role where that is the contract;
- referenced execution targets and local dependency files exist where applicable;
- unknown modes and broken references fail closed;
- registry/schema compatibility is explicit rather than guessed from Tool version alone.

### 4.3 Deterministic selection resolver

The reusable resolver should support, at minimum:

- path normalization;
- changed-file selection;
- automatic and manual mode selection;
- `all` selection;
- dependency-sensitive multi-mode selection where explicitly declared;
- control-plane changes that conservatively select all applicable modes;
- stable machine-readable execution plans;
- no architecture reclassification during selection.

The exact control-plane path set must be configurable or adapter-owned rather than hard-coded to one Consumer Repository layout.

### 4.4 Local developer execution

The core reusable functionality should not require GitHub Actions merely to resolve or validate Test Modes.

A Consumer Repository should be able to perform locally equivalent operations, conceptually:

```text
validate registry against ptsip.yaml
resolve affected modes
emit execution plan
execute selected verification or hand the plan to a runner
```

Exact CLI names are an implementation decision for a later approved optional session.

### 4.5 GitHub Actions adapter/template

PTSIP may ship or generate a supported GitHub Actions integration based on the proven `tooling-test.yml` pattern:

```text
checkout
    ↓
validate Test Mode registry + Project Profile references
    ↓
resolve selected modes
    ↓
matrix execution
    ↓
record PTSIP provenance
```

The reference adapter should support self-hosted runners cleanly, while not making one repository's Windows runner labels a universal PTSIP requirement.

### 4.6 Provenance in development verification

Execution results should be able to expose provenance from the authoritative Project Profile, for example:

```text
mode id
component_ref
classification
roles
purpose
selected test targets
source revision / profile identity where useful
```

This lets developers and coding agents understand not only which tests ran but which declared verification responsibility justified that execution.

## 5. User-facing value

The capability should allow another PTSIP adopter to turn a validated Project Profile into a practical development control surface without inventing a repository-specific ownership router from scratch.

Expected benefits include:

- focused tests based on declared verification responsibility;
- reduced CI cost and feedback time compared with unconditional full-suite execution;
- explicit linkage from a test run back to the Project Profile responsibility it protects;
- fail-closed detection of stale or invalid Test Mode references after architecture changes;
- fewer ad-hoc path-based assumptions in coding-agent workflows;
- reusable local and CI development verification behavior;
- preservation of full qualification as a separate stronger release boundary.

## 6. Release/qualification firewall

This feature must preserve the distinction:

```text
Selective Test Mode success
    ≠ full repository qualification
```

A reusable Test Mode integration must not imply that a focused mode run is sufficient for a project's full release gate unless that project explicitly defines such a release contract through the appropriate authority.

For the PTSIP repository itself, existing exact-SHA/full qualification semantics remain independent from this optional productization effort.

## 7. Project Profile surface rule

The initial productization should prefer consuming existing Project Profile responsibility information rather than expanding `ptsip.yaml` with CI-specific configuration.

A new Project Profile field is justified only if the project itself must declare new durable truth, intent, authority, or analysis-contract information.

The following should remain Test Mode/execution concerns unless later architecture review proves otherwise:

```text
runner labels
Python/runtime setup
pytest CLI arguments
package install commands
timeouts
watch globs
CI matrix formatting
GitHub Actions implementation details
```

## 8. Packaging and distribution intent

For this to become a real PTSIP user capability rather than repository-local dogfood, a later optional implementation plan should decide how users receive it. Candidate deliverables include:

- packaged resolver/validator logic in the PTSIP distribution;
- versioned Test Mode registry schema;
- CLI entry points for validation/resolution;
- template/bootstrap assets for `.github/test-mode/`;
- supported GitHub Actions example or generator;
- adoption documentation showing how to connect existing `VERIFICATION` components to modes;
- migration guidance from repository-local resolvers such as the current `turbo-system` implementation.

Release packaging must be verified so a PyPI-installed PTSIP can actually access every asset promised by the public feature.

## 9. Candidate optional session sequence

This sequence is planning guidance, not implementation authorization.

```text
OP-TM-01 — Contract extraction
    define generic responsibility/execution boundary from PTSIP + turbo-system precedent

OP-TM-02 — Registry schema and semantic validator
    component_ref / VERIFICATION / target validation

OP-TM-03 — Generic resolver and machine execution plan
    changed paths, manual selection, control-plane behavior

OP-TM-04 — Local CLI / execution adapter
    validate + resolve without GitHub dependency

OP-TM-05 — GitHub Actions adapter and provenance
    reusable workflow/template with configurable runner/runtime surfaces

OP-TM-06 — External dogfood and distribution readiness
    verify against a Consumer Repository such as turbo-system and package promised assets
```

Each future session must remain independently classified `OPTIONAL / NON-BLOCKING` unless a Promotion Review explicitly changes its release relationship.

## 10. Acceptance criteria for eventual feature release

Before PTSIP advertises this as a supported user feature, the implementation should prove that a Consumer Repository can:

1. keep verification architecture authority in `ptsip.yaml`;
2. define Test Modes without duplicating Project Profile classification/role/purpose authority;
3. fail closed on unknown `component_ref`;
4. fail closed when the referenced responsibility is not eligible for verification;
5. deterministically resolve changed-file and manual selections;
6. conservatively handle control-plane changes;
7. emit stable machine-readable execution plans;
8. run the same resolver locally without requiring GitHub;
9. use a supported GitHub Actions adapter when desired;
10. report PTSIP-backed verification provenance;
11. preserve selective-vs-full qualification separation;
12. install the feature and its schemas/templates from the released PTSIP package;
13. demonstrate at least one external Consumer Repository integration using current Project Profile semantics.

## 11. Non-goals of this optional plan

This plan does not authorize:

- changing the current Project Profile schema merely to store CI convenience data;
- making GitHub Actions a universal PTSIP dependency;
- inferring architecture ownership from test paths or watch globs;
- replacing full release qualification with selective modes;
- copying `turbo-system` repository-specific mode ids, paths, Python versions, or runner labels into PTSIP defaults;
- implementing this feature before the applicable optional session is separately approved;
- making this feature a Tool 0.4.0 release blocker.

## 12. Release relationship

This feature is deliberately tracked under:

```text
planning/0.4.0-op/
```

because the existing 0.4.0 generic remediation CORE can release without it. Its value is substantial, and it is a strong candidate for formal PTSIP product support, but its development state must remain independent from the CORE release dependency graph unless explicitly promoted later.
