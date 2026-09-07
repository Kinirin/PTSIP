# WU-13 — Dependency Evidence Reconciliation

**Status:** IMPLEMENTED; completion is conditional on the current commit's successful `self-hosted/tooling-test` verification
**Entry baseline:** `f2fbbb5175eafde0ff8a3b7c5e7ca31a8224189d`  
**Branch:** `dev/0.3.7-wu13-dependency-reconciliation`

## Purpose

Reduce large fail-closed dependency evidence sets into mechanically reconciled evidence before human review, without weakening PTSIP lifecycle rules or manufacturing architecture authority.

WU-13 is a Tool-side evidence/reconciliation improvement. It does not change the bound `0.3.7-draft` normative Specification in its initial tranche.

## Problem demonstrated by turbo-system

The consumer repository exposed several direct Product imports whose runtime distributions were already declared in a Product-owned runtime requirements manifest, but PTSIP still emitted unresolved `dependency-target:*` blockers because native Python declaration discovery only considered repository-root requirement files.

The same audit also exposed import/distribution naming mismatches such as:

- `PIL` -> `Pillow`
- `yaml` -> `PyYAML`

and showed that flat blocker output forces manual package-by-package review.

## Tranche 1 — Product runtime declaration reconciliation

Implement a deterministic reconciliation layer that:

1. inspects requirement manifests already assigned to declared components;
2. recognizes `requirements*.txt/.in` and names such as `runtime-requirements.txt`;
3. treats same-component declarations as external dependency evidence;
4. permits a Product component with `runtime_required: true` to use a Product-owned runtime requirements manifest as external dependency supply evidence;
5. supports only explicit conservative import/distribution aliases;
6. records reconciliation provenance and declaration paths;
7. suppresses only the unresolved dependency coverage gap proven by this declaration evidence;
8. leaves dynamic imports, unresolved relative imports, verification-purpose separation, and ambiguous namespace distributions unresolved for later tranches.

## Non-goals

WU-13 must not:

- infer lifecycle classification or project-owned relationships;
- convert dynamic imports into static architecture;
- treat arbitrary Product requirement files as global architecture authority;
- use the active Python environment as dependency authority;
- silently clear unresolved edges without provenance;
- weaken `PTSIP-DEP-001` or `PTSIP-EVD-003`;
- move the immutable Specification revision unless a later tranche identifies a genuine normative defect.

## Implemented follow-up tranches

- component/role-aware verification dependency separation;
- repository-local relative and namespace-package reconciliation;
- deterministic dynamic-import classification summaries;
- actionable buckets (`REPOSITORY_DEFECT`, `REVIEW_REQUIRED`, `EVIDENCE_INCOMPLETE`, `RESOLVER_LIMITATION`);
- concise human summary output while preserving full machine JSON;
- batch/component-level reconciliation reporting.

## Local-agent cost-control bootstrap

WU-13 uses the machine-readable operational policy:

```text
planning/0.3.7/WU-13-local-agent-cost-control.yaml
```

The local coding agent MUST use machine-first evidence handling before broad source reasoning. Until the planned Actionability Classifier and Review Pack surfaces exist, each AI review cycle is bounded to the configured item/context budget, full conformance JSON stays out of the prompt by default, and validation is batched by change/tranche rather than repeated per dependency.

Implementation priority is intentionally reordered so cost-control infrastructure comes before large-scale dependency cleanup:

```text
Actionability Classifier
    -> Review Pack
    -> concise summary
    -> incremental cache
    -> validation-plan generation
    -> remaining dependency reconciliation tranches
```

This policy is operational planning metadata only. It does not change PTSIP lifecycle authority, Specification semantics, or Consumer Repository architecture.

## Verification boundary

Focused tests must prove declaration matching, alias matching, and fail-closed behavior for an undeclared dependency. Full repository regression and exact-SHA self-hosted verification remain required before WU-13 completion.

## Machine-first implementation checkpoint

T1 through T10 are implemented at source commit `84427e75ab6a76fda6b32f0461c566f2dd99e3aa`. T11 consumer measurement below used those exact executable bytes. Subsequent documentation commits require their own exact-SHA verification before completion; a historical workflow success does not verify the current branch tip.

| Commit | Implementation |
| --- | --- |
| `f9590a3` | Preserve declaration reconciliation through the public conformance engine. |
| `931a137` | Classify all dependency evidence into four advisory actionability buckets. |
| `bc5d872` | Add public bounded Review Packs and concise summaries. |
| `b40d7a4` | Cache source evidence in external Tool-owned state. |
| `6645712` | Generate proposed validation commands from declared verification ownership. |
| `f440916` | Separate verification purpose while retaining unresolved test evidence as blocking. |
| `7cb69c4` | Resolve tracked relative/local targets and literal or finite-loop dynamic imports. |
| `84427e7` | Preserve namespace ambiguity, relative dynamic semantics, cache identity and minimum review context. |

The frozen binding remains Tool `0.3.7`, Specification `0.3.7-draft @ 3c47816770d194ae42f98faedc911d980db0e62a`. No Specification, profile, lifecycle classification, consumer dependency declaration, or consumer source rewrite was made.

Public surfaces used during implementation:

```powershell
ptsip dependency analyze . --json
ptsip dependency review-pack . --max-items 8 --max-context-bytes 12000
ptsip dependency validation-plan . --changed tests/ptsip/conformance/test_dependency_local_resolution.py --changed tests/ptsip/conformance/test_dependency_cache.py --changed tests/ptsip/conformance/test_dependency_actionability.py --changed tests/ptsip/conformance/test_dependency_validation.py --changed tests/ptsip/cli/test_dependency_cli.py --json
```

Reports were redirected through explicit UTF-8 `Set-Content` into the external temporary directory; full dependency items were not sent to AI. The last validation plan selected the declared `ptsip-core-verification` and `repository-test-mode-control-plane` components. Its generated focused command was actually executed:

```powershell
python -m pytest tests/ptsip/cli/test_dependency_cli.py tests/ptsip/conformance/test_dependency_actionability.py tests/ptsip/conformance/test_dependency_cache.py tests/ptsip/conformance/test_dependency_local_resolution.py tests/ptsip/conformance/test_dependency_validation.py -q
```

Result: **28 passed**, exit `0`. The subsequent affected component batch was:

```powershell
python -m pytest tests/ptsip/conformance tests/ptsip/inspection tests/ptsip/cli -q --maxfail=10
```

Result: **112 passed**, exit `0`, 169.56 seconds. This includes existing scanner and public CLI compatibility tests as well as positive, negative, ambiguity, provenance, stale-cache and bounded-context contracts.

## turbo-system consumer measurement

Before and after strict runs used the same clean, stable consumer commit `053f3182e6c5bb840971f31f35ffaf61beb5c71c`, branch `fix/stale-import-residue-r6`. This was newer than the requested historical consumer baseline and was preserved. Baseline Tool source was `9598e43a3fb405448b133c66623bc4c3cfd5dcdb`; final Tool source was `84427e75ab6a76fda6b32f0461c566f2dd99e3aa`.

| Measurement | Before | After |
| --- | ---: | ---: |
| Observed dependency edges | 6,589 | 6,592 |
| All blocking evidence gaps | 413 | 55 |
| Runtime/unseparated dependency-target gaps | 394 | 24 |
| Separate verification dependency-target gaps | 0 | 12 |
| Combined dependency-target gaps | 394 | 36 |
| Strict exit code | 6 | 6 |
| Strict outcome | `INCOMPLETE` | `INCOMPLETE` |

Mechanical comparison by source path, line and edge type accounts for every original dependency-gap location: 194 were declaration-reconciled, 165 gained native target resolution, and 35 remained blocking. One additional blocking location was observed, giving 36 remaining dependency gaps and a net reduction of 358. The 12 verification gaps remain blocking and are not counted as resolved merely because their bucket changed. There were no unmatched original locations. The remaining 55 overall gaps preclude a consumer conformance claim.

The final advisory analysis classified **all 6,592 edges mechanically**:

| Actionability | Count |
| --- | ---: |
| `AUTO_RESOLVED` | 5,513 |
| `REPOSITORY_DEFECT` | 0 |
| `REVIEW_REQUIRED` | 61 |
| `RESOLVER_LIMITATION` | 1,018 |

There are 561 verification-purpose dependency edges. Aggregate checks confirmed zero repository-defect proposals for the protected declared imports (`typer`, `openpyxl`, `piexif`, `urllib3`, `numpy`, `PIL`), verification-only edges, and resolved repository-local imports. No individual AUTO_RESOLVED or resolver-queue source review was performed.

| Python-source cache | Hits | Recomputed | Invalid | Write failures |
| --- | ---: | ---: | ---: | ---: |
| Final cold run | 0 | 982 | 0 | 0 |
| Final warm run | 982 | 0 | 0 | 0 |

Cache counts describe source files, not edges. Focused tests also prove that changing one of two ordinary source files reuses the other file, while manifest, profile, parser/platform or resolver identity changes invalidate affected cache evidence. Full repository snapshots are checked freshly even on cache hits. No consumer `.ptsip/` state directory is created.

### Bounded AI review record

Actual AI-reviewed unique items: **8**. All were selected `REVIEW_REQUIRED` items in an earlier eight-item, 3,000-byte Review Pack. They remain deferred for target/support/ownership evidence; no support contract or architecture decision was invented. Two lacked sufficient import context under that smaller cap, which motivated mandatory import-context deferral in the generator.

The final public generator selected the same eight evidence IDs with the normal 12,000-byte cap and deferred 53. All eight final items contain import context; the largest item is 8,924 serialized UTF-8 bytes and no item exceeds four source files. The expanded final pack was prepared for follow-up, not counted as a second AI review. Its `ai_reviewed_items: 0` correctly states that the Tool itself invokes no AI.

The reviewed set covers guarded optional/dynamic imports, runtime-selected plugin targets, an availability probe, a parameterized launcher, and a resolved local import needing ownership evidence. Their support and ownership questions affect consumer follow-up only; they do not authorize consumer changes or prevent the Tool from retaining fail-closed results.

## Full regression and exact-SHA evidence

The first full local command, `python -m pytest -q --maxfail=10`, at `7cb69c4d87600dd221466dbae330c49f863bffea` ended with **557 passed, 1 failed** (exit `1`). The failing console-script test selected the older global Tool `0.3.6` executable through PATH. With the dedicated Tool `0.3.7` virtual environment's Scripts directory prepended to PATH, that exact test passed in the 13-test correction batch (exit `0`). No test assertion or Product behavior was weakened to hide the environment mismatch.

The final completion gate is the existing `tooling-test.yml` workflow dispatched with `scope=full`, `mode=all` on the current WU-13 branch tip. It must verify the dispatched SHA, pass the complete regression and artifact checks, and record a successful `self-hosted/tooling-test` status for that SHA. This document does not substitute a prior local or remote success for that check. Release, publication, tagging and main merge remain outside this WU.

Bootstrap used a separate external virtual environment installed with the repository's documented editable development extras. The requested `requirements-agent.txt` installation exited `1` because that file does not exist in this repository. Public GitHub-backed gate verification returned `NO_DECISION_REQUIRED`, exit `0`, after selecting only actually discovered candidates.

The consumer's required Turbo SDK context-pack command succeeded at its exact clean HEAD. Its two generated report files were retained outside the consumer, and final consumer status was clean.

Local raw evidence is retained outside both repositories under `%TEMP%`:

- `ptsip-wu13-consumer-before-clean.json`
- `ptsip-wu13-consumer-final-cold.json`
- `ptsip-wu13-consumer-final-warm.json`
- `ptsip-wu13-consumer-final-conform.json`
- `ptsip-wu13-consumer-comparison.json`
- `ptsip-wu13-final-validation-plan.json`
- `ptsip-wu13-final-component-tests.txt`
- `ptsip-wu13-full-regression.txt` (initial failed environment run)

The final Review Pack is in existing external Tool state under `dependency-reviews/ea2e2ae2136330245901/92b9f62181bd735d98c8.json`. These local locators are handoff evidence, not durable CI records or architecture authority.
