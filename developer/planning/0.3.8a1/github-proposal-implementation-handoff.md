# PTSIP Tool 0.3.8a1 — GitHub Proposal Implementation Handoff

## 1. Purpose

This handoff records the current implementation baseline for the remaining Tool 0.3.8a1 GitHub-coordination work and defines the exact next execution order.

The active runtime feature remains narrowly scoped to:

```text
explicit proposed nonexistent component
        ↓
stable proposal identity
        ↓
existing Decision Control Plane
        ↓
PROPOSAL_APPROVED local/application receipt
        ↓
no component materialization
no active components[] projection
```

This document is an execution handoff. It does not expand the Tool 0.3.8a1 release scope and does not claim completion of Tool 0.4.0 runtime architecture.

## 2. Exact baseline

Implementation baseline before this handoff document:

```text
branch: dev/0.3.8a1
HEAD:   e958bea9a153f85c72c9f527631c44a10f109c28
```

The current coding-agent entry path is machine-resolved through:

```text
Planning Entry Resolver
        ↓
developer/planning/0.3.8a1/emergency-implementation-overlay.yaml

Policy Resolver
        ↓
src/ptsip/app/github_authority.py + MODIFY
        ↓
MPD-0010 exact sections
PTSIP-AUT-001 .. PTSIP-AUT-007 exact projections
AST-validated implementation refs

Implementation Work Packet
        ↓
edit targets
read-only context
acceptance vectors
exact pytest nodes
edit budget
freshness guard
```

## 3. Developer automation currently available

### 3.1 Planning entry

`dev/0.3.8a1` resolves exactly to:

```text
developer/planning/0.3.8a1/emergency-implementation-overlay.yaml
role = EMERGENCY_RELEASE_OVERLAY
```

### 3.2 Policy resolution

For:

```text
scope     = src/ptsip/app/github_authority.py
operation = MODIFY
```

the resolver returns:

- MPD-0010:
  - `identity_and_resolution`
  - `canonical_and_runtime_projection`
- PTSIP-AUT-001 through PTSIP-AUT-007
- exact canonical Specification sections
- actual Git branch guard
- AST-validated implementation locations
- task test refs
- fail-closed constraints

The policy resolver remains a routing/projection surface. Canonical MPD and Specification records remain authority.

### 3.3 Implementation Work Packet

Developer-only surfaces:

```text
developer/automation/implementation_work_packet.py
developer/automation/implementation_workflows.yaml
developer/automation/implementation_workflows.schema.json
```

The current work packet separates:

#### Read-only context

```text
src/ptsip/app/github_authority.py::_global_decision_id
src/ptsip/app/github_authority.py::GithubControlPlaneClient.gate
src/ptsip/cli.py::resolve command branch
src/ptsip/proposed_component.py
```

#### Current edit targets

```text
src/ptsip/app/github_authority.py::_workflow_status
src/ptsip/app/github_authority.py::GithubControlPlaneClient.application
```

These edit targets are intentionally narrower than the read context.

## 4. Current local verification evidence

The current work packet was successfully prepared against exact HEAD:

```text
e958bea9a153f85c72c9f527631c44a10f109c28
```

Generated packet identity:

```text
iwp-9ee1e83f89ad5d75
```

### 4.1 Packet guard

Result:

```text
status: PASS
problems: []
branch expected/actual: dev/0.3.8a1 / dev/0.3.8a1
HEAD expected/actual: e958bea9... / e958bea9...
changed_paths: []
unexpected_changed_paths: []
context_changed: []
code_scope_violations: []
locations_need_refresh: false
```

This proves the packet was fresh and the local repository had no out-of-budget changes at the recorded baseline.

### 4.2 Baseline verification

Exact baseline nodes:

```text
tests/ptsip/test_proposed_component.py::
  test_cli_registers_and_approves_nonexistent_component_without_active_profile_projection

tests/ptsip/control_plane/test_github_authority.py::
  test_github_authority_uses_component_scope_not_local_clarification_id

tests/ptsip/control_plane/test_github_authority_reconciliation.py::
  test_application_receipt_is_explicitly_local_projection
```

Observed local result:

```text
3 passed / 0 failed
Python 3.14.3
pytest 8.4.2
```

This is baseline evidence only. It does not prove the missing GitHub proposal lifecycle is implemented.

## 5. Current incomplete state

The work packet intentionally reports:

```text
verification.status = REQUIRES_NEW_TESTS
```

The following required tests do not yet exist:

```text
tests/ptsip/control_plane/test_github_authority.py::
  test_github_proposal_resolution_returns_terminal_local_receipt

tests/ptsip/control_plane/test_github_authority.py::
  test_repeated_github_proposal_is_terminal_without_reapplication

tests/ptsip/control_plane/test_github_authority.py::
  test_normal_github_resolution_still_requires_local_application
```

Therefore the next implementation must not be considered complete until these acceptance tests exist and pass.

## 6. Required runtime semantics for the remaining implementation

### 6.1 Global authority remains global

For an explicit proposed component:

```text
PENDING
   ↓ valid resolution
RESOLVED
```

The GitHub global authority record must remain a global decision record.

Do not persist `PROPOSAL_APPROVED` as a replacement for the global `RESOLVED` winner state.

### 6.2 Proposal workflow status is terminal after resolution

For:

```text
record.status = RESOLVED
request.origin = EXPLICIT_PROPOSED_COMPONENT
```

the workflow result must be terminal:

```text
RESOLVED
```

It must not be converted to:

```text
RESOLVED_APPLICATION_REQUIRED
```

because the proposal approval does not require active Project Profile projection or physical materialization.

### 6.3 Application receipt

`GithubControlPlaneClient.application()` must support:

```text
status = PROPOSAL_APPROVED
```

as a local/application receipt while preserving global/local state separation.

Expected semantic boundary:

```text
global authority:
  status = RESOLVED
  winner answer unchanged

application receipt:
  status = PROPOSAL_APPROVED
  scope = LOCAL_PROJECTION
```

The receipt must not change the architecture winner.

### 6.4 Normal GitHub decisions must remain unchanged

For ordinary resolved decisions that are not explicit proposed components:

```text
RESOLVED
→ RESOLVED_APPLICATION_REQUIRED
```

must remain unchanged.

### 6.5 No materialization

Proposal approval must not:

- create the proposed component directory;
- create an active component declaration;
- project the proposal into `components[]`;
- fabricate project authority;
- weaken existing fail-closed coordination behavior.

## 7. Acceptance vectors

The current Work Packet defines four mandatory vectors.

### GITHUB_PROPOSAL_RESOLUTION

```text
precondition:
  request_origin = EXPLICIT_PROPOSED_COMPONENT
  global_decision_state = PENDING

action:
  RESOLVE_VALID_ANSWER

expected:
  global_decision_state = RESOLVED
  application_receipt_state = PROPOSAL_APPROVED
  application_receipt_scope = LOCAL_PROJECTION
  active_component_projection = false
  physical_materialization = false
```

### GITHUB_PROPOSAL_REPEAT

```text
precondition:
  request_origin = EXPLICIT_PROPOSED_COMPONENT
  global_decision_state = RESOLVED

action:
  GATE_SAME_SCOPE

expected:
  workflow_status = RESOLVED
  authority_write_required = false
  active_component_projection = false
```

### NORMAL_GITHUB_DECISION_UNCHANGED

```text
precondition:
  request_origin = NORMAL_DECISION
  global_decision_state = RESOLVED

expected:
  workflow_status = RESOLVED_APPLICATION_REQUIRED
```

### LOCAL_RECEIPT_DOES_NOT_CHANGE_WINNER

```text
precondition:
  request_origin = EXPLICIT_PROPOSED_COMPONENT
  global_decision_state = RESOLVED

action:
  RECORD_PROPOSAL_APPROVED_RECEIPT

expected:
  application_receipt_scope = LOCAL_PROJECTION
  global_answer_unchanged = true
```

## 8. Exact next work order

### Step 1 — Refresh before editing

If HEAD, branch, policy context, planning context, or relevant source files changed after the baseline, regenerate the packet.

```powershell
python -m developer.automation.implementation_work_packet prepare `
  --scope src/ptsip/app/github_authority.py `
  --operation MODIFY `
  --output .git/ptsip-iwp.json `
  --json
```

### Step 2 — Add the three required acceptance tests

Add exactly the three required test nodes listed in section 5.

Prefer establishing the behavioral failure before changing runtime code.

### Step 3 — Implement only the current edit targets

Primary mutation targets:

```text
_workflow_status
GithubControlPlaneClient.application
```

Do not modify read-only context solely because it appears in the packet.

If implementation genuinely requires a broader mutation scope, stop and resolve/regenerate the task rather than silently broadening the edit budget.

### Step 4 — Run packet guard

```powershell
python -m developer.automation.implementation_work_packet check `
  --packet .git/ptsip-iwp.json `
  --json
```

Required result before verification/commit:

```text
status = PASS
unexpected_changed_paths = []
context_changed = []
code_scope_violations = []
```

If `locations_need_refresh = true` and another edit iteration is needed, regenerate the packet so AST locations are fresh.

### Step 5 — Focused verification

After all three required new tests exist:

```powershell
python -m developer.automation.implementation_work_packet verify `
  --packet .git/ptsip-iwp.json `
  --stage focused
```

Focused verification is intentionally blocked while required new tests are missing.

### Step 6 — Relevant regression

```powershell
python -m developer.automation.implementation_work_packet verify `
  --packet .git/ptsip-iwp.json `
  --stage regression
```

Regression targets:

```text
tests/ptsip/test_proposed_component.py
tests/ptsip/control_plane/test_github_authority.py
tests/ptsip/control_plane/test_github_authority_reconciliation.py
```

### Step 7 — Review implementation completeness

Confirm all acceptance vectors are represented by passing tests and no global/local authority boundary was weakened.

Only after this review should the emergency feature be reconsidered for broader release verification.

### Step 8 — Broader/full verification later

```powershell
python -m developer.automation.implementation_work_packet verify `
  --packet .git/ptsip-iwp.json `
  --stage full
```

Full regression is a broader completion/release boundary. Do not confuse the current baseline PASS with full release readiness.

## 9. Test Mode status

The current repository Test Mode registry does not register a `ptsip-core` lane.

Current Work Packet result:

```text
status: NOT_REGISTERED
component_ref: ptsip-core-verification
fallback: DEVELOPER_LOCAL_TASK_LANE
```

This is intentionally not changed by the present automation work.

Adding a repository-wide `ptsip-core` Test Mode would change the existing four-mode Phase 3 contract and requires an explicit policy/architecture decision rather than an implementation-agent inference.

## 10. Do not do

The next implementation agent must not:

- store `PROPOSAL_APPROVED` as the global authority winner state;
- treat proposal approval as active component materialization;
- modify `components[]` for the unmaterialized proposal;
- skip the three required new tests;
- broaden mutation scope outside the Work Packet without re-resolution;
- silently fall back to local authority when GitHub coordination fails;
- infer a new `ptsip-core` Test Mode without explicit authorization;
- claim release readiness from the 3-test baseline result.

## 11. Handoff completion criteria

This handoff can be considered implementation-complete only when all of the following are true:

- all three required GitHub proposal tests exist;
- Work Packet `check` reports `PASS`;
- focused verification passes;
- relevant regression verification passes;
- explicit proposal repetition returns terminal `RESOLVED`;
- ordinary GitHub resolution still returns `RESOLVED_APPLICATION_REQUIRED`;
- `PROPOSAL_APPROVED` is accepted only as the local/application receipt required by this feature;
- the global authority winner remains `RESOLVED` and unchanged by the receipt;
- no proposed component is materialized or projected into active components;
- no out-of-budget runtime mutation was required, or any required scope expansion was explicitly re-resolved.

## 12. Release boundary after implementation

Successful completion of the implementation steps above still does not by itself make Tool 0.3.8a1 release-ready.

The later release path remains:

```text
implementation completeness review
        ↓
relevant/full verification as required
        ↓
dev/0.3.8a1 → dev/0.4.0
        ↓
restore active 0.4.0 planning context
        ↓
dev/0.4.0 → main
        ↓
exact-main-SHA self-hosted verification
        ↓
release contract
        ↓
release dispatch
```

Do not skip the exact-SHA release boundary.
