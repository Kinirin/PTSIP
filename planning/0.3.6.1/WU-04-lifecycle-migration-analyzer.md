# WU-04 — Lifecycle Migration Analyzer

> **Status:** PRE-CREATED / LOCKED  
> **Target Tool:** `0.3.6.1`  
> **Roadmap predecessor:** WU-03 — Tool 0.3.5 legacy reader  
> **Planning relocation baseline:** `3529a32c862e1d43a91f732ced36358b4e13e1d9`  
> **Entry baseline:** not assigned; capture fresh branch HEAD on actual entry  
> **Successor:** WU-05 — split/relationship/associated-artifact target proposals

## 0. Purpose

Analyze a legacy Tool `0.3.5` architecture together with current repository evidence and produce a reviewable migration analysis toward the canonical lifecycle model introduced by Tool `0.3.6` and used by Tool `0.3.6.1`.

WU-04 decides neither project intent nor writes. It identifies where direct preservation is possible, where lifecycle meaning changed, and where owner confirmation is required.

## 1. Inputs

```text
WU-03 legacy read model
+ WU-01/WU-02 normalized repository evidence
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

### WU-04A — migration comparison model
Define source state, observed state, and target-analysis findings without conflating them.

### WU-04B — legacy classification analysis
Classify migration questions produced by historical `PRODUCT | TOOLCHAIN | NEUTRAL_CONTRACT` versus the five-class lifecycle ontology.

### WU-04C — evidence correlation
Attach normalized evidence to findings while keeping evidence non-authoritative.

### WU-04D — ambiguity and rationale
Every non-trivial recommendation input must expose why review is needed.

### WU-04E — deterministic reports
Equivalent source/evidence snapshots must produce stable analysis output suitable for WU-05 proposal generation.

## 4. Non-goals

WU-04 does not construct the final target map, ask for confirmation, or write `ptsip.yaml`. Those responsibilities belong to WU-05 and WU-06.

## 5. Completion gate

WU-04 is complete when legacy/current/evidence states are compared deterministically, all ambiguous lifecycle changes remain explicit, no legacy vocabulary is silently canonicalized, every finding is provenance-backed, and the analysis contract is stable enough for WU-05 proposal generation.

## 6. Entry discipline

Pre-created roadmap document only. Actual entry requires WU-03 completion plus a fresh branch HEAD recorded here.
