# WU-08 — Lifecycle Migration Analyzer

> **Status:** PRE-CREATED / LOCKED  
> **Roadmap predecessor:** WU-07 — Tool 0.3.5 legacy reader  
> **Planning baseline:** `6aa3554cde4de2faa0e507fc5ddd3637174caa8c`  
> **Entry baseline:** not assigned; capture fresh branch HEAD on actual entry  
> **Successor:** WU-09 — split/relationship/associated-artifact target proposals

## 0. Purpose

Analyze a legacy Tool `0.3.5` architecture together with current repository evidence and produce a reviewable migration analysis toward the Tool `0.3.6` lifecycle model.

WU-08 decides neither project intent nor writes. It identifies where direct preservation is possible, where lifecycle meaning changed, and where owner confirmation is required.

## 1. Inputs

```text
WU-07 legacy read model
+ WU-05/WU-06 normalized repository evidence
+ Tool 0.3.6 canonical lifecycle ontology
    -> migration analysis
```

## 2. Required distinctions

The analyzer must distinguish:

- exact semantic preservation;
- terminology-only compatibility cases;
- ambiguous historical `TOOLCHAIN` cases;
- candidates that likely need lifecycle separation;
- missing relationships or associated-artifact representation;
- stale legacy declarations not supported by current evidence;
- new repository candidates absent from the historical profile.

No single automatic rule such as `TOOLCHAIN -> DEVELOPMENT_TOOLING` is authoritative.

## 3. Work tracks

### WU-08A — migration comparison model
Define source state, observed state, and target-analysis findings without conflating them.

### WU-08B — legacy classification analysis
Classify migration questions produced by historical `PRODUCT | TOOLCHAIN | NEUTRAL_CONTRACT` versus the five-class Tool 0.3.6 ontology.

### WU-08C — evidence correlation
Attach normalized evidence to findings while keeping evidence non-authoritative.

### WU-08D — ambiguity and rationale
Every non-trivial recommendation input must expose why review is needed.

### WU-08E — deterministic reports
Equivalent source/evidence snapshots must produce stable analysis output suitable for WU-09 proposal generation.

## 4. Non-goals

WU-08 does not construct the final target map, ask for confirmation, or write `ptsip.yaml`. Those responsibilities belong to WU-09 and WU-10.

## 5. Completion gate

WU-08 is complete when legacy/current/evidence states are compared deterministically, all ambiguous lifecycle changes remain explicit, no legacy vocabulary is silently canonicalized, every finding is provenance-backed, and the analysis contract is stable enough for WU-09 proposal generation.

## 6. Entry discipline

Pre-created roadmap document only. Actual entry requires WU-07 completion plus a fresh branch HEAD recorded here.
