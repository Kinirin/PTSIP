use crate::source::{hash, parse, text};
use regex::Regex;
use serde_json::{json, Value};
use std::collections::{BTreeMap, BTreeSet};

const TEMPLATE_CATALOG: &[u8] = include_bytes!("historical-profile-templates.json");
const V034: &str = "TOOL_0.3.5_PROFILE_0.3.4_DRAFT";
const V036: &str = "TOOL_0.3.6_PROFILE_0.3.6_DRAFT";
const COMPONENT: &str = "COMPONENT";
const ARTIFACT: &str = "ASSOCIATED_ARTIFACT";
const BOUNDARY: &str = "BOUNDARY";

pub const SOURCE: &str = "PROJECT_HISTORICAL_PROFILE_SOURCE_DECLARATIONS";
pub const COVERAGE: &str = "PROJECT_REPOSITORY_COVERAGE_AND_AMBIGUITY";
pub const COMPATIBILITY: &str = "PROJECT_TARGET_COMPATIBILITY_BY_OWNER_AND_PATH";

fn rows<'a>(data: &'a Value, key: &str) -> &'a [Value] {
    data[key].as_array().map(Vec::as_slice).unwrap_or_default()
}

fn strings(data: &[Value]) -> impl Iterator<Item = &str> {
    data.iter().filter_map(Value::as_str)
}

fn normalize(value: &str) -> String {
    value
        .trim()
        .replace('\\', "/")
        .trim_start_matches("./")
        .trim_matches('/')
        .into()
}

fn issue(code: &str, message: &str) -> Value {
    json!({"coverages":[], "relationship_semantics":[], "issues":[{"code":code,"message":message,"subject_id":null}]})
}

fn coverage_row(row: &Value, family: &str, kind: &str, origin: &str) -> Value {
    let from_template = origin == "TEMPLATE_EFFECTIVE";
    let classification = match (kind, from_template) {
        (COMPONENT, true) => &row["classification"],
        (COMPONENT, false) => &row["source_classification"],
        _ => &Value::Null,
    };
    json!({"declaration_id":row["id"],"kind":kind,"source_classification":classification,
        "include":rows(row,"include"),"exclude":rows(row,"exclude"),"purpose":row["purpose"],
        "source_pointer":if from_template { "/responsibility_map/template" } else { row["source_pointer"].as_str().unwrap_or_default() },
        "source_family":family,"origin":origin})
}

fn indexed(values: &[Value]) -> Result<BTreeMap<&str, &Value>, String> {
    let mut index = BTreeMap::new();
    for row in values {
        let id = text(row, "id")?;
        if index.insert(id, row).is_some() {
            return Err(format!("Duplicate declaration ID {id:?}"));
        }
    }
    Ok(index)
}

fn merged<'a>(
    base: &'a [Value],
    replacements: &'a [Value],
    removals: &[Value],
) -> Result<Vec<(&'a Value, &'static str)>, String> {
    let base = indexed(base)?;
    let overrides = indexed(replacements)?;
    let mut removed = BTreeSet::new();
    for id in strings(removals) {
        if !removed.insert(id) || !base.contains_key(id) || overrides.contains_key(id) {
            return Err(format!("Hybrid removal must be unique, name a template declaration, and have no matching override: {id:?}"));
        }
    }
    Ok(base
        .into_iter()
        .filter(|(id, _)| !removed.contains(id) && !overrides.contains_key(id))
        .map(|(_, row)| (row, "TEMPLATE_EFFECTIVE"))
        .chain(
            overrides
                .values()
                .copied()
                .map(|row| (row, "PROJECT_OVERRIDE")),
        )
        .collect())
}

fn materialize_template(
    input: &Value,
    semantics: &Value,
    coverages: &mut Vec<Value>,
    relations: &mut Vec<Value>,
) -> Result<(), String> {
    let catalog = parse(TEMPLATE_CATALOG)?;
    let selected = catalog
        .as_array()
        .ok_or("Embedded template catalog must be an array")?;
    let mut identities = BTreeSet::new();
    let mut template = None;
    for row in selected {
        let id = text(row, "id")?;
        let revision = text(row, "revision")?;
        let bytes = serde_json::to_vec(&row["map"]).map_err(|e| e.to_string())?;
        if !identities.insert((id, revision)) || revision != format!("sha256:{}", hash(&bytes)) {
            return Err(format!(
                "Embedded template revision is invalid: {id}@{revision}"
            ));
        }
        if semantics["template_id"] == id && semantics["template_revision"] == revision {
            template = Some(&row["map"]);
        }
    }
    let template = template.ok_or_else(|| {
        format!(
            "No admitted template matches {}@{}",
            semantics["template_id"], semantics["template_revision"]
        )
    })?;
    let hybrid = semantics["responsibility_map_mode"] == "hybrid";
    for (key, removal_key, kind) in [
        ("components", "remove_component_ids", COMPONENT),
        (
            "associated_artifacts",
            "remove_associated_artifact_ids",
            ARTIFACT,
        ),
        ("relationships", "remove_relationship_ids", ""),
    ] {
        let replacements = if hybrid { rows(input, key) } else { &[] };
        let removals = if hybrid {
            rows(semantics, removal_key)
        } else {
            &[]
        };
        for (row, origin) in merged(rows(template, key), replacements, removals)? {
            if key == "relationships" {
                relations.push(json!([row["from"], row["to"], row["type"]]));
            } else {
                coverages.push(coverage_row(row, V036, kind, origin));
            }
        }
    }
    Ok(())
}

pub fn project_source(input: &Value) -> Result<Value, String> {
    let family = text(input, "family")?;
    let semantics = &input["family_semantics"];
    let mut coverages = Vec::new();
    let mut relations = Vec::new();
    match (family, text(semantics, "kind")?) {
        (V034, "V034") => {
            coverages.extend(
                indexed(rows(input, "components"))?
                    .into_values()
                    .map(|row| coverage_row(row, family, COMPONENT, "SOURCE_COMPONENT")),
            );
            if semantics["declaration_form"] == "BOUNDARIES" {
                for boundary in rows(semantics, "boundaries") {
                    let class = text(boundary, "source_classification")?;
                    for (index, root) in strings(rows(boundary, "roots")).enumerate() {
                        let root = normalize(root);
                        coverages.push(json!({"declaration_id":format!("boundary:{class}:{index}:{root}"),"kind":BOUNDARY,
                            "source_classification":class,"include":[root],"exclude":[],"purpose":"historical_boundary_root",
                            "source_pointer":boundary["source_pointer"],"source_family":family,"origin":"SOURCE_BOUNDARY"}));
                    }
                }
            }
        }
        (V036, "V036") if semantics["responsibility_map_mode"] == "explicit" => {
            for (key, kind) in [
                ("components", COMPONENT),
                ("associated_artifacts", ARTIFACT),
            ] {
                coverages.extend(
                    indexed(rows(input, key))?
                        .into_values()
                        .map(|row| coverage_row(row, family, kind, "PROJECT_EXPLICIT")),
                );
            }
            relations.extend(
                indexed(rows(input, "relationships"))?
                    .into_values()
                    .map(|row| json!([row["from"], row["to"], row["type"]])),
            );
        }
        (V036, "V036") => {
            if let Err(error) =
                materialize_template(input, semantics, &mut coverages, &mut relations)
            {
                return Ok(issue("SOURCE_TEMPLATE_RESOLUTION_FAILED", &error));
            }
        }
        _ => {
            return Ok(issue(
                "SOURCE_SEMANTICS_MISMATCH",
                "Recorded source family and declaration semantics differ.",
            ))
        }
    }
    coverages.sort_by(|a, b| {
        (a["kind"].as_str(), a["declaration_id"].as_str())
            .cmp(&(b["kind"].as_str(), b["declaration_id"].as_str()))
    });
    relations.sort_by(|a, b| strings(a.as_array().unwrap()).cmp(strings(b.as_array().unwrap())));
    Ok(json!({"coverages":coverages,"relationship_semantics":relations,"issues":[]}))
}

type Score = (bool, usize, usize, std::cmp::Reverse<usize>);

fn score(selector: &str) -> Score {
    let wildcards = selector
        .chars()
        .filter(|c| matches!(c, '*' | '?' | '['))
        .count();
    let literal = selector
        .chars()
        .filter(|c| !matches!(c, '*' | '?' | '[' | ']'))
        .count();
    (
        wildcards == 0,
        literal,
        selector.matches('/').count(),
        std::cmp::Reverse(wildcards),
    )
}

struct Selector {
    key: String,
    score: Score,
    expression: Option<Regex>,
}

impl Selector {
    fn compile(selector: &str) -> Result<Self, String> {
        let key = normalize(selector);
        let specificity = score(&key);
        if specificity.0 {
            return Ok(Self {
                key,
                score: specificity,
                expression: None,
            });
        }
        let chars: Vec<char> = key.chars().collect();
        let mut regex = String::from(r"\A");
        let mut index = 0;
        while index < chars.len() {
            match chars[index] {
                '*' if chars.get(index + 1) == Some(&'*') => {
                    regex.push_str(".*");
                    index += 1;
                }
                '*' => regex.push_str("[^/]*"),
                '?' => regex.push_str("[^/]"),
                '[' => {
                    if let Some(end) = chars[index + 1..]
                        .iter()
                        .position(|c| *c == ']')
                        .map(|offset| index + 1 + offset)
                    {
                        regex.push('[');
                        for c in &chars[index + 1..end] {
                            if matches!(c, '&' | '~' | '[') {
                                regex.push('\\');
                            }
                            regex.push(*c);
                        }
                        regex.push(']');
                        index = end;
                    } else {
                        regex.push_str(r"\[");
                    }
                }
                c => regex.push_str(&regex::escape(&c.to_string())),
            }
            index += 1;
        }
        regex.push_str(r"\z");
        let expression =
            Regex::new(&regex).map_err(|err| format!("Invalid selector {selector:?}: {err}"))?;
        Ok(Self {
            key,
            score: specificity,
            expression: Some(expression),
        })
    }

    fn matches(&self, path: &str) -> bool {
        self.expression
            .as_ref()
            .map_or(self.key == path, |expr| expr.is_match(path))
    }
}

struct Selectors {
    include: Vec<Selector>,
    exclude: Vec<Selector>,
    boundary: Option<String>,
}

impl Selectors {
    fn compile(row: &Value) -> Result<Self, String> {
        if row["kind"] == BOUNDARY {
            let root = strings(rows(row, "include")).next().unwrap_or_default();
            return Ok(Self {
                include: Vec::new(),
                exclude: Vec::new(),
                boundary: Some(normalize(root)),
            });
        }
        let include = strings(rows(row, "include"))
            .map(Selector::compile)
            .collect::<Result<_, _>>()?;
        let exclude = strings(rows(row, "exclude"))
            .map(Selector::compile)
            .collect::<Result<_, _>>()?;
        Ok(Self {
            include,
            exclude,
            boundary: None,
        })
    }

    fn coverage_match(&self, path: &str) -> Option<(Score, &str)> {
        if let Some(root) = &self.boundary {
            return (!root.is_empty()
                && (path == root
                    || path
                        .strip_prefix(root)
                        .is_some_and(|suffix| suffix.starts_with('/'))))
            .then(|| (score(&format!("{root}/**")), root.as_str()));
        }
        if self.exclude.iter().any(|selector| selector.matches(path)) {
            return None;
        }
        let selected = self
            .include
            .iter()
            .filter(|selector| selector.matches(path))
            .fold(None, |best: Option<&Selector>, selector| {
                if best.is_some_and(|prior| prior.score >= selector.score) {
                    best
                } else {
                    Some(selector)
                }
            });
        selected.map(|item| (item.score, item.key.as_str()))
    }
}

pub fn project_coverage(input: &Value) -> Result<Value, String> {
    let coverage = rows(input, "coverages");
    let selectors = coverage
        .iter()
        .map(Selectors::compile)
        .collect::<Result<Vec<_>, _>>()?;
    let paths = strings(rows(input, "paths")).collect::<BTreeSet<_>>();
    let mut hits = BTreeSet::new();
    let mut existing = Vec::new();
    let mut uncovered = Vec::new();
    let mut ambiguous = Vec::new();
    for path in paths {
        let normalized = normalize(path);
        let mut winners = Vec::new();
        for (row, selectors) in coverage.iter().zip(&selectors) {
            if let Some((score, key)) = selectors.coverage_match(&normalized) {
                hits.insert((text(row, "declaration_id")?, key.to_owned()));
                winners.push((
                    score,
                    text(row, "kind")?,
                    text(row, "declaration_id")?,
                    key,
                    row,
                ));
            }
        }
        if winners.is_empty() {
            uncovered.push(json!({"kind":"UNCOVERED_REPOSITORY_ELEMENT","path":path}));
            continue;
        }
        let kinds = winners.iter().map(|entry| entry.1).collect::<BTreeSet<_>>();
        if !(kinds.contains(COMPONENT) && kinds.contains(ARTIFACT)) {
            let best = winners.iter().map(|entry| entry.0).max().unwrap();
            winners.retain(|entry| entry.0 == best);
        }
        let unique: BTreeMap<_, _> = winners.iter().map(|entry| (entry.2, entry.4)).collect();
        if unique.len() > 1 {
            let mut rows: Vec<_> = unique.into_values().collect();
            rows.sort_by(|a, b| {
                (a["kind"].as_str(), a["declaration_id"].as_str())
                    .cmp(&(b["kind"].as_str(), b["declaration_id"].as_str()))
            });
            let selectors = winners.iter().map(|entry| entry.3).collect::<BTreeSet<_>>();
            ambiguous.push(json!({"kind":"AMBIGUOUS_SOURCE_ELEMENT","path":path,"coverages":rows,"selectors":selectors}));
        } else {
            let chosen = winners
                .into_iter()
                .min_by_key(|entry| (entry.1, entry.2, entry.3))
                .unwrap();
            existing.push(json!({"kind":"EXISTING_SOURCE_ELEMENT","path":path,"coverage":chosen.4,"selector":chosen.3}));
        }
    }
    let mut removed = Vec::new();
    for row in coverage {
        let id = text(row, "declaration_id")?;
        for key in strings(rows(row, "include")).map(normalize) {
            if !hits.contains(&(id, key.clone())) {
                removed.push(json!({"kind":"REMOVED_SOURCE_ELEMENT","element_id":format!("{id}:{key}"),"coverage":row,"selector":key}));
            }
        }
    }
    removed.sort_by(|a, b| a["element_id"].as_str().cmp(&b["element_id"].as_str()));
    Ok(json!({"existing":existing,"removed":removed,"uncovered":uncovered,"ambiguous":ambiguous}))
}

const REVIEW: &str = "TARGET_REVIEW_REQUIRED";
const TOOLCHAIN: &str = "HISTORICAL_TOOLCHAIN_AMBIGUITY";

fn compatible_result(path: &str, status: &str, lifecycle: Value) -> Value {
    json!({"path":path,"target_status":status,"lifecycle":lifecycle})
}

fn lifecycle(path: &str, kind: &str, source: &Value, target: &Value, rationale: &str) -> Value {
    json!({"subject_id":path,"kind":kind,"source_classification":source,"target_classification":target,"rationale":rationale})
}

fn target_result(
    existing: &Value,
    target: &Value,
    owners: &[(&str, &Value, Selectors)],
    classifications: &BTreeSet<&str>,
) -> Result<Value, String> {
    let path = text(existing, "path")?;
    let normalized = normalize(path);
    let coverage = &existing["coverage"];
    let source = &coverage["source_classification"];
    let finding = |kind, dest: &Value, rationale| lifecycle(path, kind, source, dest, rationale);
    let review = |message| compatible_result(path, REVIEW, finding(REVIEW, &Value::Null, message));
    if target.is_null() {
        let result = if source == "TOOLCHAIN" {
            finding(TOOLCHAIN,&Value::Null,"Historical TOOLCHAIN is source vocabulary and cannot select one target lifecycle without project-owned resolution.")
        } else {
            Value::Null
        };
        return Ok(compatible_result(path, "NOT_EVALUATED", result));
    }
    let matches = owners
        .iter()
        .filter_map(|(kind, row, selectors)| {
            selectors
                .coverage_match(&normalized)
                .map(|(score, _)| (score, *kind, *row))
        })
        .collect::<Vec<_>>();
    let Some(best) = matches.iter().map(|entry| entry.0).max() else {
        return Ok(review(
            "No accepted target declaration provably covers this required source element.",
        ));
    };
    let kinds = matches.iter().map(|entry| entry.1).collect::<BTreeSet<_>>();
    let mut winners = BTreeMap::new();
    for (_, kind, row) in matches.into_iter().filter(|entry| entry.0 == best) {
        winners.entry((kind, text(row, "id")?)).or_insert(row);
    }
    if kinds.len() > 1 || winners.len() > 1 {
        return Ok(review(
            "Accepted target state has ambiguous coverage for this source obligation.",
        ));
    }
    let ((kind, id), row) = winners.into_iter().next().unwrap();
    let same_declaration = if coverage["declaration_id"] == id {
        "ALREADY_SATISFIED"
    } else {
        "COMPATIBLE_TARGET_STATE"
    };
    let dest = &row["classification"];
    if coverage["kind"] == ARTIFACT {
        if kind == ARTIFACT {
            return Ok(compatible_result(path, same_declaration, Value::Null));
        }
        let result = lifecycle(path,REVIEW,&Value::Null,dest,"A source associated-artifact obligation is covered by a different target declaration kind.");
        return Ok(compatible_result(path, REVIEW, result));
    }
    if kind != COMPONENT {
        return Ok(review("A source component/boundary obligation is covered by a target associated-artifact scope."));
    }
    if source == "TOOLCHAIN" {
        return Ok(compatible_result(path,REVIEW,finding(TOOLCHAIN,dest,"Historical TOOLCHAIN can correspond to multiple target lifecycles; no automatic conversion is authoritative.")));
    }
    let known = |value: &Value| value.as_str().is_some_and(|v| classifications.contains(v));
    let (status, kind, rationale) = if source == dest && known(source) {
        (
            same_declaration,
            "EXACT_SEMANTIC_PRESERVATION",
            "Source lifecycle classification is preserved by accepted target coverage.",
        )
    } else if known(source) && known(dest) {
        ("CONFLICTING_TARGET_STATE","POSSIBLE_LIFECYCLE_SEPARATION","Source and accepted target lifecycle classifications differ; owner review is required before treating the obligation as resolved.")
    } else {
        (
            REVIEW,
            REVIEW,
            "Lifecycle compatibility cannot be proven under the target Project Profile vocabulary.",
        )
    };
    Ok(compatible_result(
        path,
        status,
        finding(kind, dest, rationale),
    ))
}

pub fn project_compatibility(input: &Value) -> Result<Value, String> {
    let target = &input["target"];
    let mut owners = Vec::new();
    for (key, kind) in [
        ("components", COMPONENT),
        ("associated_artifacts", ARTIFACT),
    ] {
        for row in rows(target, key) {
            owners.push((kind, row, Selectors::compile(row)?));
        }
    }
    let classifications = strings(rows(input, "classifications")).collect::<BTreeSet<_>>();
    let mut results = Vec::new();
    for existing in rows(input, "existing") {
        results.push(target_result(existing, target, &owners, &classifications)?);
    }
    Ok(json!({"results":results}))
}
