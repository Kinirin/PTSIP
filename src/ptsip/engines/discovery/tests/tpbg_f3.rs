use ptsip_engine_discovery::tpbg::{
    discover, discover_with_validation, Check, Contract, Origin, Report, Snapshot, Verdict,
};
use serde_json::{json, Value};
use std::fs;
use std::path::PathBuf;
use std::time::{SystemTime, UNIX_EPOCH};

const RA: &str = ".ptsip/profiles/responsibilities.yaml";
const RAC: &str = ".ptsip/profiles/assignment.json";
const AEC: &str = ".ptsip/profiles/exercise.json";
const FIRST: &str = "/responsibility_authority/assign~1a~0/RAC_id/0";
const SECOND: &str = "/responsibility_authority/assign-b/RAC_id/0";
const EXERCISE: &str = "/responsibility_authority/assign~1a~0/AEC_id/0";
const WRONG_NAME: &str = "/responsibility_authority/assign~1a~0/RAC_id/1";

struct Repo(PathBuf);

impl Repo {
    fn fixture() -> Self {
        let id = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        let root = std::env::temp_dir().join(format!("ptsip-tpbg-{}-{id}", std::process::id()));
        fs::create_dir(&root).unwrap();
        let repo = Self(root);
        fs::create_dir_all(repo.0.join(".ptsip/profiles/nested")).unwrap();
        repo.write(".ptsip/ptsip_config.json", "{}");
        repo.write(".ptsip/index.json", "invalid legacy index");
        repo.write(".ptsip/profiles/index.json", "invalid legacy index");
        repo.write(".ptsip/profiles/nested/ignore.yaml", "invalid nested input");
        repo.write(
            RA,
            concat!(
                "ptsip:\n  id: profiles-11111111111141118111111111111111\n",
                "profiles:\n  version: pp.3.00\nresponsibility_authority:\n",
                "  assign/a~:\n    RAC_id: [[rac-a, assignment-a], [rac-b, assignment-b]]\n",
                "    AEC_id: [[aec-a, exercise-a]]\n",
                "  assign-b:\n    RAC_id: [[rac-a, assignment-a]]\n",
                "    AEC_id: [[aec-a, exercise-a], [aec-b, exercise-b]]\n",
            ),
        );
        repo.write(RAC, r#"{"ptsip":{"id":"profiles-22222222222242228222222222222222"},"profiles":{"version":"pp.3.00"},"RAC_profile":[{"RAC_id":"rac-a","RAC_name":"assignment-a"},{"RAC_id":"rac-b","RAC_name":"different-name"}]}"#);
        repo.write(AEC, r#"{"ptsip":{"id":"profiles-33333333333343338333333333333333"},"profiles":{"version":"pp.3.00"},"AEC_profile":[{"AEC_id":"aec-a","AEC_name":"exercise-a"}]}"#);
        repo
    }

    fn write(&self, path: &str, contents: &str) {
        fs::write(self.0.join(path), contents).unwrap();
    }
}

impl Drop for Repo {
    fn drop(&mut self) {
        let _ = fs::remove_dir_all(&self.0);
    }
}

fn origin(file: &str, pointer: &str) -> Origin {
    Origin {
        file: file.into(),
        pointer: pointer.into(),
    }
}

fn proof(snapshot: &Snapshot, id: &str, pointer: &str, check: Check) -> Report {
    Report {
        id: id.into(),
        issuer: "fixture-validation".into(),
        subject: origin(RA, pointer),
        check,
        verdict: Verdict::Verified,
        sources: std::iter::once(snapshot.config.clone())
            .chain(snapshot.profiles.iter().map(|doc| doc.source.clone()))
            .collect(),
        contract_rev: Some("fixture-revision".into()),
    }
}

fn reports(snapshot: &Snapshot) -> Vec<Report> {
    vec![
        proof(snapshot, "fresh-source", "", Check::Freshness),
        proof(
            snapshot,
            "ra-contract",
            "",
            Check::Binding {
                contract: Some(Contract {
                    id: "fixture-ra-contract".into(),
                    rev: "fixture-revision".into(),
                    role: "fixture-ra-role".into(),
                    scope: "fixture-scope".into(),
                    schema_sha256: "a".repeat(64),
                }),
            },
        ),
        proof(snapshot, "reference-tuple", FIRST, Check::Reference),
        proof(
            snapshot,
            "first-relationship",
            FIRST,
            Check::Relation {
                target: Some(origin(RAC, "/RAC_profile/0")),
            },
        ),
        proof(
            snapshot,
            "second-relationship",
            SECOND,
            Check::Relation {
                target: Some(origin(RAC, "/RAC_profile/0")),
            },
        ),
        proof(
            snapshot,
            "exercise-relationship",
            EXERCISE,
            Check::Relation {
                target: Some(origin(AEC, "/AEC_profile/0")),
            },
        ),
    ]
}

fn rows<'a>(output: &'a Value, pointer: &str) -> &'a [Value] {
    output.pointer(pointer).unwrap().as_array().unwrap()
}

fn reference<'a>(output: &'a Value, pointer: &str) -> &'a Value {
    rows(output, "/references")
        .iter()
        .find(|row| row["origin"]["pointer"] == pointer)
        .unwrap()
}

#[test]
fn discovers_external_repository_without_evidence_and_preserves_independent_lists() {
    let repo = Repo::fixture();
    let before = fs::read(repo.0.join(RA)).unwrap();
    let output = discover(&repo.0).unwrap();
    assert_eq!(rows(&output, "/bindings").len(), 3);
    let references = rows(&output, "/references");
    assert_eq!(
        references
            .iter()
            .map(|row| row["pair"][0].as_str().unwrap())
            .collect::<Vec<_>>(),
        ["rac-a", "aec-a", "aec-b", "rac-a", "rac-b", "aec-a"]
    );
    assert_eq!(
        reference(&output, FIRST)["pair"],
        json!(["rac-a", "assignment-a"])
    );
    assert!(references
        .iter()
        .all(|row| row["state"] == "unverified" && row["target"].is_null()));
    assert_eq!(
        output["evidence"],
        json!({"binding": [], "reference": [], "relation": [], "freshness": []})
    );
    assert!(rows(&output, "/graph/edges")
        .iter()
        .all(|edge| edge["kind"] != "resolves_to"));
    assert_eq!(before, fs::read(repo.0.join(RA)).unwrap());
}

#[test]
fn creates_distinct_occurrences_and_one_verified_target() {
    let repo = Repo::fixture();
    let output = discover_with_validation(&repo.0, reports).unwrap();
    let expected: Value = serde_json::from_str(include_str!("fixtures/tpbg-f3.json")).unwrap();
    assert_eq!(output, expected);
    let first = reference(&output, FIRST);
    let second = reference(&output, SECOND);
    assert_ne!(first["node"], second["node"]);
    assert_eq!(first["target"], second["target"]);
    assert_eq!(first["state"], "verified");
    assert_eq!(second["state"], "verified");
    assert_eq!(reference(&output, EXERCISE)["state"], "verified");
    assert_eq!(
        rows(&output, "/graph/nodes")
            .iter()
            .filter(|node| node["kind"] == "rac")
            .count(),
        1
    );
    assert_eq!(
        rows(&output, "/graph/edges")
            .iter()
            .filter(|edge| edge["kind"] == "resolves_to")
            .count(),
        3
    );
    for pointer in [
        "/evidence/binding",
        "/evidence/reference",
        "/evidence/relation",
        "/evidence/freshness",
    ] {
        assert!(!rows(&output, pointer).is_empty());
    }
    assert_eq!(
        output,
        discover_with_validation(&repo.0, |snapshot| {
            let mut reports = reports(snapshot);
            reports.reverse();
            reports
        })
        .unwrap()
    );
}

#[test]
fn ignores_stale_reports_and_inexact_or_rejected_relationships() {
    let repo = Repo::fixture();
    let output = discover_with_validation(&repo.0, |snapshot| {
        let mut stale = proof(
            snapshot,
            "stale",
            FIRST,
            Check::Relation {
                target: Some(origin(RAC, "/RAC_profile/0")),
            },
        );
        stale.sources[0].sha256 = "0".repeat(64);
        let wrong_name = proof(
            snapshot,
            "wrong-name",
            WRONG_NAME,
            Check::Relation {
                target: Some(origin(RAC, "/RAC_profile/1")),
            },
        );
        let mut rejected = proof(
            snapshot,
            "rejected",
            SECOND,
            Check::Relation { target: None },
        );
        rejected.verdict = Verdict::Rejected;
        vec![stale, wrong_name, rejected]
    })
    .unwrap();
    assert_eq!(rows(&output, "/evidence/relation").len(), 1);
    assert_eq!(reference(&output, FIRST)["state"], "unverified");
    assert_eq!(reference(&output, WRONG_NAME)["state"], "unverified");
    assert_eq!(reference(&output, SECOND)["state"], "rejected");
    assert!(rows(&output, "/references")
        .iter()
        .all(|row| row["target"].is_null()));
}

#[test]
fn keeps_competing_targets_as_evidence_without_choosing_an_edge() {
    let repo = Repo::fixture();
    let alternate = ".ptsip/profiles/alternate.json";
    repo.write(alternate, r#"{"profiles":{"version":"pp.3.00"},"RAC_profile":[{"RAC_id":"rac-a","RAC_name":"assignment-a"}]}"#);
    let output = discover_with_validation(&repo.0, |snapshot| {
        let mut reports = reports(snapshot);
        reports.push(proof(
            snapshot,
            "competing-target",
            FIRST,
            Check::Relation {
                target: Some(origin(alternate, "/RAC_profile/0")),
            },
        ));
        reports
    })
    .unwrap();
    let row = reference(&output, FIRST);
    assert_eq!(row["state"], "unverified");
    assert!(row["target"].is_null());
    assert!(rows(&output, "/evidence/relation")
        .iter()
        .any(|report| report["target"]["file"] == alternate));
    assert!(rows(&output, "/graph/edges")
        .iter()
        .all(|edge| edge["kind"] != "resolves_to" || edge["from"] != row["node"]));
}

#[test]
fn does_not_attach_evidence_if_snapshot_changes_during_validation() {
    let repo = Repo::fixture();
    let output = discover_with_validation(&repo.0, |snapshot| {
        repo.write(".ptsip/ptsip_config.json", "{\"changed\":true}");
        reports(snapshot)
    })
    .unwrap();
    assert!(rows(&output, "/evidence/relation").is_empty());
    assert_eq!(rows(&output, "/references").len(), 6);
}

#[test]
fn reports_normalization_failure_for_a_malformed_ra_tuple() {
    let repo = Repo::fixture();
    repo.write(RA, "profiles: {version: pp.3.00}\nresponsibility_authority: {bad: {RAC_id: [[rac-a]], AEC_id: [[aec-a, exercise-a]]}}\n");
    let error = discover(&repo.0).unwrap_err();
    assert_eq!(error.stage, "normalization");
    assert_eq!(error.path, RA);
    assert!(error
        .detail
        .contains("/responsibility_authority/bad/RAC_id/0"));
}
