# PTSIP — Applicability small-unit handoff

> Non-normative local-agent prompt. This file does not register a Task Engine operation,
> authorize contract activation, or change effective policy.

## Entry and responsibility boundaries

Read `AGENTS.md`, then the current active branch/HEAD, and these existing sources:

- `.ptsip/contracts/policy/unified/authoring-decision.json` — previously approved structural work.
- `developer/policy/contracts/rbac-hybrid-conditions-direction.v1.json` — approved RBAC/ABAC design direction.
- `.ptsip/contracts/policy/unified/applicability-registry.json` — read-only applicability contract.
- `.ptsip/tasks/observation-learning/decision.json` and `observations.json` — observational evidence, **not policy**.
- `.ptsip/tasks/index.json` — currently reserved; do not claim a registered Task Engine.
- `.ptsip/contracts/compile-sealed-rbac/...` — inspect exact RA/RAC/AEC registration and permission flags before proposing any write.

Do **not** request reapproval of the already approved hybrid design direction.
Do **not** conflate approved design, GitHub push permission, implementation write permission,
and normative RA/RAC/AEC activation.

## Step 1 — Mechanical work (already scripted)

From the repository root on Windows 11, execute:

```powershell
python .ptsip/engine/developer/policy/applicability/run_local.py
```

This reuses the existing snapshot adapter and Go assessor, writes read-only results to
`.agent/policy-applicability/<full-HEAD>/` and test logs to the system temporary directory.
It validates the binding count and exact source revision, and runs the narrowly scoped
Go + Pytest checks. Do not rewrite this routine in an agent prompt.

Read `triage.json` and `assessment.json`. The triage groups are **mechanical counts,
not semantic ownership decisions**. Never infer contractual authority from family, path,
rule labels or a reverse `policy_basis` reference.

## Step 2 — One bounded agent task only

Choose **one** unresolved group and at most **three** representative rule references
from `triage.json`. Read those exact rules, contract items and revisions.

Complete only this small unit:

1. State the actual responsibility and evidence each rule declares.
2. Separate the already authorized implementation changes from new policy meaning.
3. If an existing, effective contract and authority explicitly permit a **non-normative
   tool/validator improvement**, implement that narrow improvement and its test.
4. Otherwise produce one precise, natural-language review question identifying the
   missing owner/condition/evidence; do not fabricate a binding.
5. Report the changed files, tests, and still-unresolved reasons. Do not expand to all
   2,001 unresolved cases automatically.

For old `src/policy/**` material, translate the current wording into ordinary Korean
with its real scenario before asking the user to confirm revised meaning; do not pursue
perfect historical migration. CASE-001..017 are observations, not automatically
generalized rules. Respect previous authorized CHANGE edits without treating unrelated
cases as approval.

## Write and validation boundaries

- Allowed: bounded implementation/test changes consistent with effective policy;
  non-normative derived analyses/candidates; approved candidate-contract preparation.
- Not granted by this prompt: changing `UNRESOLVED` to `DECLARED` or
  `NOT_APPLICABLE` without evidence, activating unregistered RA/RAC/AEC or Rego,
  policy lifecycle changes, or bypassing normative authorization gates.
- Keep existing read-only applicability behavior and its registry untouched.
- Do not create a branch; use the current active branch only. Do not bypass branch protection.
- Verify locally, then use the registered test mode and GitHub Actions after a permitted push.
- Keep long logs outside commits and remove temporary logs after delivering the result.

**Stop condition:** one mechanically verified group, one bounded improvement or a single
evidence-backed review question, and an exact test report. This is a *small work unit*,
not a replacement for the full automation project.
