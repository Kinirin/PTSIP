//! Read-only TPBG F3 projection. A host must bind an admitted Validation source
//! separately before supplying evidence. Consumer files supply declarations only.

use serde_json::{json, Value};
use std::collections::BTreeMap;
use std::path::Path;
use std::sync::OnceLock;

mod source;

pub const SCHEMA: &str = include_str!("../../tpbg-f3.schema.json");

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Source {
    pub path: String,
    pub sha256: String,
}

impl Source {
    fn value(&self) -> Value {
        json!({"path": self.path, "sha256": self.sha256})
    }
}

pub struct Document {
    pub source: Source,
    pub value: Value,
}

pub struct Snapshot {
    pub config: Source,
    pub config_value: Value,
    pub profiles: Vec<Document>,
}

impl Snapshot {
    fn sources(&self) -> impl Iterator<Item = &Source> {
        std::iter::once(&self.config).chain(self.profiles.iter().map(|doc| &doc.source))
    }
}

#[derive(Debug)]
pub struct Error {
    pub stage: &'static str,
    pub path: String,
    pub detail: String,
}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}: {}: {}", self.stage, self.path, self.detail)
    }
}

impl std::error::Error for Error {}

#[derive(Clone, Debug, Eq, Ord, PartialEq, PartialOrd)]
pub struct Origin {
    pub file: String,
    pub pointer: String,
}

impl Origin {
    fn value(&self) -> Value {
        json!({"file": self.file, "pointer": self.pointer})
    }

    fn key(&self, kind: &str) -> String {
        format!("{kind}:{}:{}{}", self.file.len(), self.file, self.pointer)
    }

    fn error(&self, detail: &str) -> Error {
        Error {
            stage: "normalization",
            path: self.file.clone(),
            detail: format!("{}: {detail}", self.pointer),
        }
    }
}

#[derive(PartialEq, Eq)]
pub struct Contract {
    pub id: String,
    pub rev: String,
    pub role: String,
    pub scope: String,
    pub schema_sha256: String,
}

impl Contract {
    fn is_exact(&self) -> bool {
        [&self.id, &self.rev, &self.role, &self.scope]
            .into_iter()
            .all(|field| !field.is_empty())
            && self.schema_sha256.len() == 64
            && self
                .schema_sha256
                .bytes()
                .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
    }

    fn value(&self) -> Value {
        json!({"id": self.id, "rev": self.rev, "role": self.role, "scope": self.scope,
            "schema_sha256": self.schema_sha256})
    }
}

#[derive(Clone, Copy, PartialEq, Eq)]
pub enum Verdict {
    Verified,
    Rejected,
}

impl Verdict {
    fn label(self) -> &'static str {
        match self {
            Self::Verified => "verified",
            Self::Rejected => "rejected",
        }
    }
}

pub enum Check {
    Binding { contract: Option<Contract> },
    Reference,
    Relation { target: Option<Origin> },
    Freshness,
}

pub struct Report {
    pub id: String,
    pub issuer: String,
    pub subject: Origin,
    pub check: Check,
    pub verdict: Verdict,
    pub sources: Vec<Source>,
    pub contract_rev: Option<String>,
}

impl Report {
    fn value(&self, subject: &str) -> Value {
        let target = match &self.check {
            Check::Relation { target } => target.as_ref().map(Origin::value),
            _ => None,
        };
        json!({"id": self.id, "issuer": self.issuer, "subject": subject, "verdict": self.verdict.label(),
            "sources": self.sources.iter().map(Source::value).collect::<Vec<_>>(),
            "contract_rev": self.contract_rev, "target": target})
    }
}

/// Run discovery on a selected consumer repository and expose gaps in evidence.
pub fn discover(root: &Path) -> Result<Value, Error> {
    discover_with_validation(root, |_| Vec::new())
}

/// The caller supplies a Validation adapter from its admitted engine binding.
/// Reports refer to the immutable snapshot passed to that adapter. Discovery
/// checks their correspondence to this snapshot and projects the given verdicts.
pub fn discover_with_validation(
    root: &Path,
    validate: impl FnOnce(&Snapshot) -> Vec<Report>,
) -> Result<Value, Error> {
    let snapshot = source::read(root)?;
    let mut projection = Projection::new(&snapshot)?;
    let mut reports = validate(&snapshot);
    if !reports.is_empty()
        && !source::read(root).is_ok_and(|current| current.sources().eq(snapshot.sources()))
    {
        reports.clear();
    }
    let output = projection.finish(&snapshot, reports);
    static VALIDATOR: OnceLock<jsonschema::Validator> = OnceLock::new();
    let validator = VALIDATOR.get_or_init(|| {
        let schema: Value = serde_json::from_str(SCHEMA).expect("embedded TPBG schema is JSON");
        jsonschema::validator_for(&schema).expect("embedded TPBG schema is valid")
    });
    validator.validate(&output).map_err(|error| Error {
        stage: "output",
        path: snapshot.config.path,
        detail: error.to_string(),
    })?;
    Ok(output)
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum Condition {
    Rac,
    Aec,
}

impl Condition {
    const ALL: [Self; 2] = [Self::Rac, Self::Aec];

    fn fields(self) -> (&'static str, &'static str, &'static str) {
        match self {
            Self::Rac => ("RAC_id", "RAC_name", "RAC_profile"),
            Self::Aec => ("AEC_id", "AEC_name", "AEC_profile"),
        }
    }

    fn kind(self) -> &'static str {
        match self {
            Self::Rac => "rac",
            Self::Aec => "aec",
        }
    }

    fn occurrence(self) -> &'static str {
        match self {
            Self::Rac => "rac_ref",
            Self::Aec => "aec_ref",
        }
    }
}

struct Reference {
    origin: Origin,
    ra: String,
    condition: Condition,
    pair: [String; 2],
}

struct Target {
    condition: Condition,
    pair: [String; 2],
}

#[derive(Default)]
struct Projection {
    nodes: BTreeMap<String, Value>,
    edges: Vec<Value>,
    references: Vec<Reference>,
    targets: BTreeMap<Origin, Target>,
}

fn field(value: &Value) -> Option<&str> {
    value.as_str().filter(|value| !value.is_empty())
}

fn pair(id: &Value, name: &Value) -> Option<[String; 2]> {
    Some([field(id)?.into(), field(name)?.into()])
}

impl Projection {
    fn new(snapshot: &Snapshot) -> Result<Self, Error> {
        let mut result = Self::default();
        for doc in &snapshot.profiles {
            let file = &doc.source.path;
            let profile = result.node(&Self::origin(file, String::new()), "profile");
            if doc.value.pointer("/profiles/version").and_then(field) == Some("pp.3.00") {
                result.ra(file, &profile, &doc.value)?;
                result.targets(file, &doc.value)?;
            }
        }
        Ok(result)
    }

    fn origin(file: &str, pointer: String) -> Origin {
        Origin {
            file: file.into(),
            pointer,
        }
    }

    fn node(&mut self, origin: &Origin, kind: &str) -> String {
        let id = origin.key(kind);
        self.nodes
            .entry(id.clone())
            .or_insert_with(|| json!({"id": id, "kind": kind, "origin": origin.value()}));
        id
    }

    fn edge(&mut self, from: &str, to: &str, kind: &str, evidence: Option<&str>) {
        self.edges
            .push(json!({"from": from, "to": to, "kind": kind, "evidence": evidence}));
    }

    fn ra(&mut self, file: &str, profile: &str, doc: &Value) -> Result<(), Error> {
        let Some(rows) = doc.get("responsibility_authority") else {
            return Ok(());
        };
        let rows = rows.as_object().ok_or_else(|| {
            Self::origin(file, "/responsibility_authority".into()).error("expected an RA mapping")
        })?;
        for (key, row) in rows {
            let pointer = format!(
                "/responsibility_authority/{}",
                key.replace('~', "~0").replace('/', "~1")
            );
            let ra = self.node(&Self::origin(file, pointer.clone()), "ra");
            self.edge(profile, &ra, "declares", None);
            for condition in Condition::ALL {
                let (id, _, _) = condition.fields();
                let list = row[id]
                    .as_array()
                    .filter(|list| !list.is_empty())
                    .ok_or_else(|| {
                        Self::origin(file, format!("{pointer}/{id}"))
                            .error("expected at least one condition reference")
                    })?;
                for (index, value) in list.iter().enumerate() {
                    let origin = Self::origin(file, format!("{pointer}/{id}/{index}"));
                    let Some([condition_id, name]) = value.as_array().map(Vec::as_slice) else {
                        return Err(origin.error("expected an ordered [ID, name] tuple"));
                    };
                    let pair = pair(condition_id, name).ok_or_else(|| {
                        origin.error("condition ID and name must be nonempty strings")
                    })?;
                    let occurrence = self.node(&origin, condition.occurrence());
                    self.edge(&ra, &occurrence, "references", None);
                    self.references.push(Reference {
                        origin,
                        ra: ra.clone(),
                        condition,
                        pair,
                    });
                }
            }
        }
        Ok(())
    }

    fn targets(&mut self, file: &str, doc: &Value) -> Result<(), Error> {
        for condition in Condition::ALL {
            let (id, name, root) = condition.fields();
            let Some(rows) = doc.get(root) else {
                continue;
            };
            let rows = rows.as_array().ok_or_else(|| {
                Self::origin(file, format!("/{root}")).error("expected a condition list")
            })?;
            for (index, row) in rows.iter().enumerate() {
                let origin = Self::origin(file, format!("/{root}/{index}"));
                let pair = pair(&row[id], &row[name]).ok_or_else(|| {
                    origin.error("condition ID and name must be nonempty strings")
                })?;
                self.targets.insert(origin, Target { condition, pair });
            }
        }
        Ok(())
    }

    fn finish(&mut self, snapshot: &Snapshot, reports: Vec<Report>) -> Value {
        let reports = Evidence::new(snapshot, reports);
        let mut output = EvidenceOutput::default();
        let bindings: Vec<_> = snapshot
            .profiles
            .iter()
            .map(|doc| output.binding(doc, &reports, &snapshot.config))
            .collect();
        let references: Vec<_> = std::mem::take(&mut self.references)
            .into_iter()
            .map(|reference| output.reference(self, reference, &reports, &snapshot.config))
            .collect();
        json!({
            "model": "TPBG F3",
            "sources": {"config": snapshot.config.value(),
                "profiles": snapshot.profiles.iter().map(|doc| doc.source.value()).collect::<Vec<_>>()},
            "bindings": bindings,
            "references": references,
            "graph": {"nodes": self.nodes.values().collect::<Vec<_>>(), "edges": self.edges},
            "evidence": output.value(),
        })
    }
}

struct Evidence(BTreeMap<Origin, Vec<Report>>);

impl Evidence {
    fn new(snapshot: &Snapshot, mut reports: Vec<Report>) -> Self {
        let digests: BTreeMap<_, _> = snapshot
            .sources()
            .map(|source| (&source.path, &source.sha256))
            .collect();
        let mut counts = BTreeMap::new();
        for report in &reports {
            *counts.entry(report.id.clone()).or_insert(0) += 1;
        }
        reports.sort_by(|a, b| a.id.cmp(&b.id));
        let mut by_subject = BTreeMap::<_, Vec<_>>::new();
        for mut report in reports {
            let includes_subject = report
                .sources
                .iter()
                .any(|source| source.path == report.subject.file);
            if report.id.is_empty()
                || report.issuer.is_empty()
                || counts[&report.id] != 1
                || !includes_subject
                || report.contract_rev.as_deref() == Some("")
                || !report.sources.iter().all(|s| {
                    digests
                        .get(&s.path)
                        .is_some_and(|digest| *digest == &s.sha256)
                })
            {
                continue;
            }
            report.sources.sort_by(|a, b| a.path.cmp(&b.path));
            report.sources.dedup();
            by_subject
                .entry(report.subject.clone())
                .or_default()
                .push(report);
        }
        Self(by_subject)
    }

    fn for_origin(&self, origin: &Origin) -> &[Report] {
        self.0.get(origin).map(Vec::as_slice).unwrap_or_default()
    }
}

#[derive(Default)]
struct EvidenceOutput {
    binding: Vec<Value>,
    reference: Vec<Value>,
    relation: Vec<Value>,
    freshness: Vec<Value>,
}

impl EvidenceOutput {
    fn record<'a>(&mut self, report: &'a Report, subject: &str) -> &'a str {
        let bucket = match report.check {
            Check::Binding { .. } => &mut self.binding,
            Check::Reference => &mut self.reference,
            Check::Relation { .. } => &mut self.relation,
            Check::Freshness => &mut self.freshness,
        };
        bucket.push(report.value(subject));
        &report.id
    }

    fn binding(&mut self, doc: &Document, evidence: &Evidence, config: &Source) -> Value {
        let origin = Projection::origin(&doc.source.path, String::new());
        let node = origin.key("profile");
        let profile = doc.value.pointer("/ptsip/id").and_then(field);
        let version = doc
            .value
            .pointer("/profiles/version")
            .and_then(field)
            .or_else(|| doc.value.pointer("/ptsip/version").and_then(field));
        let mut contract = None;
        let mut rejected = false;
        let mut conflicting = false;
        let mut ids = Vec::new();
        for report in evidence.for_origin(&origin) {
            match &report.check {
                Check::Freshness => (),
                Check::Binding { contract: resolved } if report.sources.contains(config) => {
                    if report.verdict == Verdict::Rejected {
                        rejected = true;
                    } else {
                        let Some(resolved) = resolved.as_ref().filter(|c| c.is_exact()) else {
                            continue;
                        };
                        if profile.is_none() || report.contract_rev.as_ref() != Some(&resolved.rev)
                        {
                            continue;
                        }
                        conflicting |= contract.is_some_and(|previous| previous != resolved);
                        contract = Some(resolved);
                    }
                }
                _ => continue,
            }
            ids.push(self.record(report, &node));
        }
        if rejected || conflicting {
            contract = None;
        }
        let state = state(rejected, contract.is_some());
        json!({"node": node, "profile": profile, "version": version, "origin": origin.value(),
            "contract": contract.map(Contract::value), "state": state, "evidence": ids})
    }

    fn reference(
        &mut self,
        projection: &mut Projection,
        reference: Reference,
        evidence: &Evidence,
        config: &Source,
    ) -> Value {
        let node = reference.origin.key(reference.condition.occurrence());
        let mut matched = None;
        let mut rejected = false;
        let mut conflicting = false;
        let mut ids = Vec::new();
        for report in evidence.for_origin(&reference.origin) {
            match &report.check {
                Check::Freshness => (),
                Check::Reference => rejected |= report.verdict == Verdict::Rejected,
                Check::Relation { target } if report.sources.contains(config) => {
                    if report.verdict == Verdict::Rejected {
                        rejected = true;
                    } else {
                        let Some(target) = target.as_ref() else {
                            continue;
                        };
                        if report.contract_rev.is_none()
                            || !report
                                .sources
                                .iter()
                                .any(|source| source.path == target.file)
                            || !projection.targets.get(target).is_some_and(|actual| {
                                actual.condition == reference.condition
                                    && actual.pair == reference.pair
                            })
                        {
                            continue;
                        }
                        conflicting |= matched.is_some_and(|(previous, _)| previous != target);
                        if matched.is_none() {
                            matched = Some((target, report.id.as_str()));
                        }
                    }
                }
                _ => continue,
            }
            ids.push(self.record(report, &node));
        }
        if rejected || conflicting {
            matched = None;
        }
        let target = matched.map(|(origin, proof)| {
            let target = projection.node(origin, reference.condition.kind());
            projection.edge(&node, &target, "resolves_to", Some(proof));
            target
        });
        json!({"node": node, "ra": reference.ra, "kind": reference.condition.kind(), "pair": reference.pair,
            "origin": reference.origin.value(), "state": state(rejected, target.is_some()), "target": target, "evidence": ids})
    }

    fn value(self) -> Value {
        json!({"binding": self.binding, "reference": self.reference, "relation": self.relation, "freshness": self.freshness})
    }
}

fn state(rejected: bool, verified: bool) -> &'static str {
    if rejected {
        Verdict::Rejected.label()
    } else if verified {
        Verdict::Verified.label()
    } else {
        "unverified"
    }
}
