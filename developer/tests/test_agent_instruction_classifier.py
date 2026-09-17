from __future__ import annotations

from pathlib import Path

import yaml

from developer.automation.agent_instruction_classifier import classify_markdown, load_trial_policy


ROOT = Path(__file__).resolve().parents[2]


def _find(atoms, text):
    return next(atom for atom in atoms if atom.text == text)


def test_level1_policy_vocabulary_is_exact_and_unresolved_is_not_namespace(tmp_path) -> None:
    root = tmp_path
    policy = root / "developer" / "policy"
    policy.mkdir(parents=True)
    (policy / "MPD-0010.yaml").write_text(
        """schema_version: ptsip-developer-policy/v1
policy_class: PTSIP_DEVELOPER_POLICY
policy: {id: MPD-0010, title: trial, status: ACTIVE}
rules:
  agent_instruction_entry_taxonomy_trial:
    level_1_vocabulary: [APPLICABILITY, RULE, ACTION, EVIDENCE, OTHER]
    level_2_vocabulary:
      APPLICABILITY: [PATH, TASK, ENVIRONMENT, VERSION, VCS_CONTEXT]
      RULE: [MUTATION, DEPENDENCY, COMPATIBILITY, STYLE, SECURITY, RESOURCE]
      ACTION: [COMMAND, READ, WRITE, GENERATE, BUILD, INSTALL, PUBLISH]
      EVIDENCE: [TEST, BUILD_RESULT, STATIC_CHECK, ARTIFACT, REVIEW, STATUS]
      OTHER: [CONTEXT, GOAL]
    unresolved_is_namespace: false
""",
        encoding="utf-8",
    )
    payload = load_trial_policy(root)
    assert payload["level_1_vocabulary"] == ["APPLICABILITY", "RULE", "ACTION", "EVIDENCE", "OTHER"]
    assert payload["unresolved_is_namespace"] is False


def test_repository_trial_policy_and_resolver_binding_are_wired() -> None:
    payload = load_trial_policy(ROOT)
    assert payload["decision"] == "APPROVED"
    assert payload["stage"] == "REPOSITORY_TRIAL"
    assert payload["distribution"] == "FORBIDDEN"
    assert payload["product_integration"] == "NOT_AUTHORIZED"

    bindings = yaml.safe_load(
        (ROOT / "developer" / "policy" / "policy-resolver-bindings.yaml").read_text(encoding="utf-8")
    )
    refs = bindings["scope_bindings"]["developer/automation/agent_instruction_classifier.py"]["default_refs"]
    mpd10 = next(item for item in refs if item["policy_id"] == "MPD-0010")
    assert "agent_instruction_entry_taxonomy_trial" in mpd10["sections"]


def test_easy_multilabel_classification() -> None:
    atoms = classify_markdown("Before publishing, run pytest and report the status.\n")
    atom = atoms[0]
    assert atom.level1 == ("APPLICABILITY", "ACTION", "EVIDENCE")
    assert atom.level2["ACTION"] == ("COMMAND",)
    assert "TEST" in atom.level2["EVIDENCE"]
    assert "STATUS" in atom.level2["EVIDENCE"]


def test_goal_word_alone_does_not_create_other_goal() -> None:
    atoms = classify_markdown("Do not reconstruct goals from filenames.\n")
    atom = atoms[0]
    assert "RULE" in atom.level1
    assert atom.level2.get("OTHER", ()) == ()


def test_explicit_purpose_relation_is_other_goal() -> None:
    atoms = classify_markdown("The purpose of this component is to route instructions cheaply.\n")
    atom = atoms[0]
    assert "OTHER" in atom.level1
    assert atom.level2["OTHER"] == ("CONTEXT", "GOAL")


def test_other_is_positive_not_fallback() -> None:
    atom = classify_markdown("frobnicator quux zed\n")[0]
    assert atom.level1 == ()
    assert atom.unresolved is True


def test_parent_applicability_is_inherited_by_list_action() -> None:
    atoms = classify_markdown(
        "For SDK work:\n\n"
        "1. Read `docs/policy/sdk_governance.yaml`.\n"
        "2. Run `ptsip gate . --json`.\n"
    )
    read_atom = _find(atoms, "Read `docs/policy/sdk_governance.yaml`.")
    run_atom = _find(atoms, "Run `ptsip gate . --json`.")
    assert "APPLICABILITY" in read_atom.inherited_level1
    assert read_atom.level1 == ("APPLICABILITY", "ACTION")
    assert read_atom.level2["ACTION"] == ("READ",)
    assert "APPLICABILITY" in run_atom.inherited_level1
    assert "COMMAND" in run_atom.level2["ACTION"]


def test_descriptive_key_value_is_other_context() -> None:
    atom = classify_markdown("Architecture baseline: MVC plus EDA\n")[0]
    assert atom.level1 == ("OTHER",)
    assert atom.level2["OTHER"] == ("CONTEXT",)
