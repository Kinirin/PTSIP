use crate::migration::{merge_template_rows, resolve_historical_template};
use crate::source::{self, Root};
use jsonschema::error::{ValidationError, ValidationErrorKind};
use serde_json::{json, Value};
use std::collections::{BTreeMap, BTreeSet};

pub const ALGORITHM: &str = "PROJECT_PROFILE_PP200_TO_PP300_SEMANTIC_FIELDS";
const CATALOG: &str = "schemas/config.json";
const SOURCE_VERSION: &str = "pp.2.00";
const TARGET_VERSION: &str = "pp.3.00";

pub fn project(input: &Value) -> Result<Value, String> {
    let mut root = Root::new(source::text(input, "package_root")?)?;
    let (catalog, _) = root.read(CATALOG)?;
    let profiles = &catalog["storage"]["profile"];
    let source_schema = profile_schema(
        &mut root,
        &profiles["ptsip.project-profile.pp.2.00"],
        SOURCE_VERSION,
    )?;
    let target_schema = profile_schema(
        &mut root,
        &profiles["ptsip.project-profile"],
        TARGET_VERSION,
    )?;
    let result = project_fields(&input["profile"], &source_schema, &target_schema)?;
    root.finish()?;
    Ok(result)
}

fn profile_schema(root: &mut Root, registration: &Value, version: &str) -> Result<Value, String> {
    let path = source::text(registration, "path")?;
    let expected_path = match version {
        SOURCE_VERSION => "history/profile/project-profile-pp.2.00.schema.json",
        TARGET_VERSION => "storage/profile/project-profile.schema.json",
        _ => return Err(format!("unsupported Profile schema version {version}")),
    };
    if registration["version"] != version || path != expected_path {
        return Err(format!("invalid registered Profile resource for {version}"));
    }
    let path = format!("schemas/{path}");
    let (schema, digest) = root.read(&path)?;
    let bytes = root.bytes(&path, 2_097_152)?;
    let canonical = std::str::from_utf8(&bytes)
        .map_err(|error| error.to_string())?
        .replace("\r\n", "\n");
    if source::hash(&bytes) != digest
        || registration["schema_sha256"] != source::hash(canonical.as_bytes())
    {
        return Err(format!(
            "registered {version} Profile schema SHA-256 mismatch"
        ));
    }
    if schema["$id"] != registration["$id"]
        || !schema["$id"].is_string()
        || schema.pointer("/properties/ptsip/properties/version/const") != Some(&json!(version))
        || schema.get("$ref").is_some()
        || !local_references(&schema)
    {
        return Err(format!(
            "registered Profile schema identity or reference mismatch for {version}"
        ));
    }
    Ok(schema)
}

fn local_references(value: &Value) -> bool {
    match value {
        Value::Object(fields) => fields.iter().all(|(key, value)| {
            if key == "$ref" {
                value
                    .as_str()
                    .is_some_and(|reference| reference.starts_with("#/$defs/"))
            } else {
                key != "$dynamicRef" && key != "$recursiveRef" && local_references(value)
            }
        }),
        Value::Array(items) => items.iter().all(local_references),
        _ => true,
    }
}

fn project_fields(
    profile: &Value,
    source_schema: &Value,
    target_schema: &Value,
) -> Result<Value, String> {
    let validator = jsonschema::validator_for(source_schema).map_err(|error| error.to_string())?;
    let mut diagnostics = Vec::new();
    let mode = profile["responsibility_map"]["mode"].as_str();
    let variant = source_schema["$defs"]["responsibilityMap"]["oneOf"]
        .as_array()
        .and_then(|alternatives| {
            alternatives.iter().position(|definition| {
                definition
                    .pointer("/properties/mode/const")
                    .and_then(Value::as_str)
                    == mode
                    && mode.is_some()
            })
        });
    for error in validator.iter_errors(profile) {
        report_source_error(&error, variant, &mut diagnostics);
    }
    if profile["ptsip"]["revision"].is_string() && profile["ptsip"]["revision"] != "Rev.0001" {
        diagnostics.push(issue("/ptsip/revision", "LINEAGE_UNMAPPED", "Projection supports the official base Rev.0001. This source revision requires an explicit lineage mapping."));
    }
    if !diagnostics.is_empty() {
        return Ok(unmapped(diagnostics));
    }
    let materialized = if matches!(mode, Some("template" | "hybrid")) {
        match materialize_profile(profile) {
            Ok(effective) => {
                if let Some(error) = validator.iter_errors(&effective).next() {
                    return Ok(unmapped(vec![issue(
                        "/responsibility_map",
                        "RESPONSIBILITY_MAP_UNMAPPED",
                        format!(
                            "Materialized Profile failed {} at effective path {}.",
                            error.kind().keyword(),
                            error.instance_path()
                        ),
                    )]));
                }
                Some(effective)
            }
            Err(diagnostic) => return Ok(unmapped(vec![diagnostic])),
        }
    } else {
        None
    };
    let effective = materialized.as_ref().unwrap_or(profile);
    let components = effective["components"]
        .as_array()
        .ok_or("explicit source Profile requires components")?;
    let mut owners = BTreeMap::new();
    for (index, component) in components.iter().enumerate() {
        let id = source::text(component, "id")?;
        if owners.insert(id, index).is_some() {
            diagnostics.push(issue(
                format!("/components/{index}/id"),
                "IDENTITY_AMBIGUOUS",
                "More than one component declares this ID.",
            ));
        }
    }
    check_reference_identity(effective, &owners, &mut diagnostics)?;
    if materialized.is_some() && !diagnostics.is_empty() {
        return Ok(unmapped(
            diagnostics
                .iter()
                .map(|diagnostic| {
                    issue(
                        "/responsibility_map",
                        diagnostic["code"]
                            .as_str()
                            .unwrap_or("RESPONSIBILITY_MAP_UNMAPPED"),
                        format!(
                            "{} Effective path: {}.",
                            diagnostic["reason"].as_str().unwrap_or_default(),
                            diagnostic["source_path"].as_str().unwrap_or_default()
                        ),
                    )
                })
                .collect(),
        ));
    }

    let policy = &effective["component_dependency_policy"];
    // An absent pp.2.00 policy applied no component-dependency restriction.
    // The other root policies remain independent and are copied verbatim.
    let default = policy["default"].as_str().unwrap_or("allow");
    let default_path = if policy.is_null() {
        "/component_dependency_policy"
    } else {
        "/component_dependency_policy/default"
    };
    let mut target = effective.clone();
    target["ptsip"]["version"] = json!(TARGET_VERSION);
    target["responsibility_map"]["mode"] = json!("DEFAULT");
    let mut changes = vec![
        change(
            "/ptsip/version",
            "/ptsip/version",
            &profile["ptsip"]["version"],
            &target["ptsip"]["version"],
        ),
        change(
            "/responsibility_map/mode",
            "/responsibility_map/mode",
            &profile["responsibility_map"]["mode"],
            &target["responsibility_map"]["mode"],
        ),
    ];
    for index in 0..components.len() {
        target["components"][index]["dependency"] = json!({"default": default});
        changes.push(change(
            default_path,
            format!("/components/{index}/dependency/default"),
            &policy["default"],
            &json!(default),
        ));
    }
    let mut pairs = BTreeMap::new();
    for list in ["allow", "deny"] {
        for (index, edge) in items(policy, list).iter().enumerate() {
            let path = format!("/component_dependency_policy/{list}/{index}");
            let from = source::text(edge, "from")?;
            let to = source::text(edge, "to")?;
            let Some(&owner) = owners.get(from) else {
                diagnostics.push(issue(
                    format!("{path}/from"),
                    "REFERENCE_UNRESOLVED",
                    "Dependency source does not name a declared component.",
                ));
                continue;
            };
            if !owners.contains_key(to) {
                diagnostics.push(issue(
                    format!("{path}/to"),
                    "REFERENCE_UNRESOLVED",
                    "Dependency target does not name a declared component.",
                ));
                continue;
            }
            if let Some(previous) = pairs.insert((from, to), list) {
                let (code, reason) = if previous == list {
                    (
                        "DEPENDENCY_RELATION_DUPLICATE",
                        "The same dependency pair is declared more than once.",
                    )
                } else {
                    (
                        "DEPENDENCY_RELATION_CONFLICT",
                        "The same dependency pair is declared in both allow and deny.",
                    )
                };
                diagnostics.push(issue(path, code, reason));
                continue;
            }
            let destination = target["components"][owner]["dependency"]
                .as_object_mut()
                .ok_or("projected dependency object required")?
                .entry(list)
                .or_insert_with(|| json!([]))
                .as_array_mut()
                .ok_or("projected dependency list required")?;
            let target_path = format!(
                "/components/{owner}/dependency/{list}/{}",
                destination.len()
            );
            destination.push(edge.clone());
            changes.push(change(path, target_path, edge, edge));
        }
    }
    if !diagnostics.is_empty() {
        return Ok(unmapped(diagnostics));
    }
    target
        .as_object_mut()
        .ok_or("projected Profile must be an object")?
        .remove("component_dependency_policy");
    if materialized.is_some() {
        // The pp.3.00 explicit map carries the resolved full rows. The source
        // map in the trace retains the exact template revision and overrides.
        for field in [
            "responsibility_map",
            "components",
            "associated_artifacts",
            "relationships",
        ] {
            if let Some(value) = target.get(field) {
                changes.push(change(
                    "/responsibility_map",
                    format!("/{field}"),
                    &profile["responsibility_map"],
                    value,
                ));
            }
        }
    }
    let validator = jsonschema::validator_for(target_schema).map_err(|error| error.to_string())?;
    if let Some(error) = validator.iter_errors(&target).next() {
        return Ok(unmapped(vec![issue(
            "",
            "TARGET_STRUCTURE_INVALID",
            format!(
                "Projected Profile failed {} at target path {}.",
                error.kind().keyword(),
                error.instance_path()
            ),
        )]));
    }
    Ok(json!({"status":"PROJECTED", "target_profile":target, "changes":changes, "unmapped":[]}))
}

fn materialize_profile(profile: &Value) -> Result<Value, Value> {
    let responsibility_map = &profile["responsibility_map"];
    let binding = &responsibility_map["template"];
    let template =
        resolve_historical_template(&binding["id"], &binding["revision"]).map_err(|reason| {
            issue(
                "/responsibility_map/template",
                "RESPONSIBILITY_MAP_UNMAPPED",
                reason,
            )
        })?;
    let collections = [
        ("components", "remove_component_ids"),
        ("associated_artifacts", "remove_associated_artifact_ids"),
        ("relationships", "remove_relationship_ids"),
    ];
    if template.as_object().is_none_or(|fields| {
        fields
            .keys()
            .any(|name| !collections.iter().any(|(key, _)| key == name))
    }) {
        return Err(issue(
            "/responsibility_map/template",
            "RESPONSIBILITY_MAP_UNMAPPED",
            "The pinned template contains an unhandled declaration.",
        ));
    }
    let mut result = profile.clone();
    result["responsibility_map"] = json!({"mode": "explicit"});
    let overrides = &responsibility_map["overrides"];
    for (collection, removal_field) in collections {
        if responsibility_map["mode"] == "hybrid" {
            let rows = merge_template_rows(
                items(&template, collection),
                items(overrides, collection),
                items(overrides, removal_field),
            )
            .map_err(|reason| {
                issue(
                    "/responsibility_map/overrides",
                    "RESPONSIBILITY_MAP_UNMAPPED",
                    reason,
                )
            })?;
            result[collection] =
                Value::Array(rows.into_iter().map(|(row, _)| row.clone()).collect());
        } else if let Some(rows) = template.get(collection) {
            result[collection] = rows.clone();
        }
    }
    Ok(result)
}

fn check_reference_identity(
    profile: &Value,
    owners: &BTreeMap<&str, usize>,
    diagnostics: &mut Vec<Value>,
) -> Result<(), String> {
    let mut endpoints: BTreeSet<&str> = owners.keys().copied().collect();
    for (index, artifact) in items(profile, "associated_artifacts").iter().enumerate() {
        let id = source::text(artifact, "id")?;
        let anchor = source::text(artifact, "anchor")?;
        if !endpoints.insert(id) {
            diagnostics.push(issue(
                format!("/associated_artifacts/{index}/id"),
                "IDENTITY_AMBIGUOUS",
                "Artifact ID does not resolve to a unique relationship endpoint.",
            ));
        }
        if !owners.contains_key(anchor) {
            diagnostics.push(issue(
                format!("/associated_artifacts/{index}/anchor"),
                "REFERENCE_UNRESOLVED",
                "Artifact anchor does not name a declared component.",
            ));
        }
    }
    let mut ids = BTreeSet::new();
    for (index, relationship) in items(profile, "relationships").iter().enumerate() {
        if !ids.insert(source::text(relationship, "id")?) {
            diagnostics.push(issue(
                format!("/relationships/{index}/id"),
                "IDENTITY_AMBIGUOUS",
                "Relationship ID is not unique.",
            ));
        }
        for endpoint in ["from", "to"] {
            if !endpoints.contains(source::text(relationship, endpoint)?) {
                diagnostics.push(issue(
                    format!("/relationships/{index}/{endpoint}"),
                    "REFERENCE_UNRESOLVED",
                    "Relationship endpoint is not declared by the source Profile.",
                ));
            }
        }
    }
    Ok(())
}

fn items<'a>(document: &'a Value, key: &str) -> &'a [Value] {
    document[key]
        .as_array()
        .map(Vec::as_slice)
        .unwrap_or_default()
}

fn report_source_error(
    error: &ValidationError<'_>,
    variant: Option<usize>,
    diagnostics: &mut Vec<Value>,
) {
    let path = error.instance_path().to_string();
    if let ValidationErrorKind::OneOfNotValid { context } = error.kind() {
        if let Some(selected) = variant
            .filter(|_| path == "/responsibility_map")
            .and_then(|index| context.get(index))
        {
            for detail in selected {
                report_source_error(detail, None, diagnostics);
            }
            return;
        }
    }
    match error.kind() {
        ValidationErrorKind::AdditionalProperties { unexpected } => {
            for field in unexpected {
                diagnostics.push(issue(
                    pointer(&path, field),
                    "UNSUPPORTED_SOURCE_FIELD",
                    "This field has no mapping in the registered pp.2.00 contract.",
                ));
            }
        }
        ValidationErrorKind::Required { property } => {
            let field = property.as_str().unwrap_or_default();
            let code = if path == "/component_dependency_policy" && field == "default" {
                "DEPENDENCY_DEFAULT_UNMAPPED"
            } else {
                "SOURCE_STRUCTURE_INVALID"
            };
            diagnostics.push(issue(
                pointer(&path, field),
                code,
                "Required pp.2.00 field is missing.",
            ));
        }
        kind => diagnostics.push(issue(
            path,
            "SOURCE_STRUCTURE_INVALID",
            format!(
                "Source field failed the registered pp.2.00 {} constraint.",
                kind.keyword()
            ),
        )),
    }
}

fn pointer(path: &str, key: &str) -> String {
    format!("{path}/{}", key.replace('~', "~0").replace('/', "~1"))
}

fn change(
    source_path: impl AsRef<str>,
    target_path: impl AsRef<str>,
    before: &Value,
    after: &Value,
) -> Value {
    json!({"source_path":source_path.as_ref(), "target_path":target_path.as_ref(), "before":before, "after":after})
}

fn issue(path: impl AsRef<str>, code: &str, reason: impl AsRef<str>) -> Value {
    json!({"source_path":path.as_ref(), "code":code, "reason":reason.as_ref()})
}

fn unmapped(diagnostics: Vec<Value>) -> Value {
    json!({"status":"UNMAPPED", "target_profile":null, "changes":[], "unmapped":diagnostics})
}
