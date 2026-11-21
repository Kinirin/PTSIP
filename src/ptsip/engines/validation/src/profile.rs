use crate::source;
use serde_json::{json, Value};
use std::collections::BTreeSet;

const SCHEMA: &str = include_str!("schemas/profile.schema.json");
const STATE_SCHEMA: &str = include_str!("schemas/profile-state.schema.json");
const TRANSITION_CATALOG: &[u8] = include_bytes!("../profile-transitions.json");

pub(crate) fn check(schema: &str, definition: &str, value: &Value) -> Result<(), String> {
    let schema: Value = serde_json::from_str(schema).map_err(|e| e.to_string())?;
    let selected = schema["$defs"]
        .get(definition)
        .ok_or("native schema definition missing")?;
    jsonschema::validate(selected, value).map_err(|e| format!("{definition}: {e}"))
}

fn version(value: &str, filename: bool) -> Result<(u64, u64), String> {
    let text = value.trim();
    let rest = text
        .strip_prefix(if filename { "pp" } else { "pp." })
        .ok_or("Profile identity must use the pp namespace")?;
    let (major, minor) = rest
        .split_once('.')
        .ok_or("Profile identity requires two segments")?;
    if major.is_empty()
        || minor.is_empty()
        || !major.bytes().all(|c| c.is_ascii_digit())
        || !minor.bytes().all(|c| c.is_ascii_digit())
    {
        return Err("Profile identity segments must be decimal integers".into());
    }
    let major = major
        .parse::<u64>()
        .map_err(|_| "Profile major segment out of range")?;
    let minor = minor
        .parse::<u64>()
        .map_err(|_| "Profile minor segment out of range")?;
    let canonical = format!("pp{}{major}.{minor:02}", if filename { "" } else { "." });
    if text != canonical {
        return Err(format!(
            "Profile identity is not canonical; expected {canonical}"
        ));
    }
    Ok((major, minor))
}

fn revision(value: &str) -> Result<String, String> {
    let digits = value
        .strip_prefix("Rev.")
        .ok_or("Profile revision must use Rev.####")?;
    if digits.len() != 4 || !digits.bytes().all(|c| c.is_ascii_digit()) || digits == "0000" {
        return Err("Profile revision must be Rev.0001 through Rev.9999".into());
    }
    Ok(value.into())
}

/// Verify an explicit Discovery-selected contract; this function never selects a current version.
pub fn identity(input: &Value) -> Result<Value, String> {
    check(SCHEMA, "identity_input", input)?;
    let (major, minor) = version(source::text(input, "version")?, false)?;
    let canonical = format!("pp.{major}.{minor:02}");
    let filename = format!("pp{major}.{minor:02}");
    if let Some(token) = input.get("filename_token").and_then(Value::as_str) {
        if version(token, true)? != (major, minor) {
            return Err("Profile filename token disagrees with the declared version".into());
        }
    }
    let parsed_revision = input
        .get("revision")
        .and_then(Value::as_str)
        .map(revision)
        .transpose()?;
    let instance_revision = input
        .get("content")
        .and_then(Value::as_str)
        .map(|text| format!("sha256:{}", source::hash(text.as_bytes())));
    let selected = &input["selected_contract"];
    let selected_version = source::text(selected, "version")?;
    version(selected_version, false)?;
    let operations = selected["operations"]
        .as_array()
        .ok_or("selected operations required")?;
    let mut supported = canonical == selected_version && selected["allowed"] != false;
    if let Some(operation) = input.get("operation").and_then(Value::as_str) {
        supported &= operations.iter().any(|item| item == operation);
    }
    let lifecycle = source::text(selected, "lifecycle")?;
    if lifecycle == "LEGACY" && input["new_registration"] == true {
        supported = false;
    }
    let mut notifications = selected
        .get("notifications")
        .and_then(Value::as_array)
        .cloned()
        .unwrap_or_default();
    if lifecycle == "DEPRECATED"
        && !notifications
            .iter()
            .any(|n| n["code"] == "UPGRADE_REQUIRED" && n["mandatory"] == true)
    {
        notifications.push(json!({"code":"UPGRADE_REQUIRED", "message":"Support retirement is scheduled; transition to the current Profile contract is required.", "mandatory":true}));
    }
    let output = json!({
        "contract_version":canonical,
        "filename_token":filename,
        "parsed_revision":parsed_revision,
        "instance_revision":instance_revision,
        "profile_role":input.get("profile_role").cloned().unwrap_or(Value::Null),
        "supported_operations":operations,
        "source_ref":selected.get("source_ref").unwrap_or(&selected["schema_resource"]),
        "source_sha256":selected["source_sha256"],
        "status":if supported {"SUPPORTED"} else {"UNSUPPORTED"},
        "notifications":notifications
    });
    check(SCHEMA, "identity_output", &output)?;
    Ok(output)
}

pub(crate) fn owned_transition_catalog() -> Result<Value, String> {
    let catalog = source::parse(TRANSITION_CATALOG)?;
    let mut schema = source::parse(STATE_SCHEMA.as_bytes())?;
    schema["$ref"] = json!("#/$defs/transition_catalog");
    jsonschema::validate(&schema, &catalog)
        .map_err(|error| format!("Profile transition catalog: {error}"))?;
    Ok(catalog)
}

/// Match a direct edge or a historical bridge from the engine catalog or explicit input.
pub fn transition(input: &Value) -> Result<Value, String> {
    check(SCHEMA, "transition_input", input)?;
    let owned_catalog;
    let catalog = if input.get("transitions").is_some() {
        input
    } else {
        owned_catalog = owned_transition_catalog()?;
        &owned_catalog
    };
    let source_version = source::text(input, "source_contract_version")?;
    let target_version = source::text(input, "target_contract_version")?;
    version(target_version, false)?;
    let mut edges = BTreeSet::new();
    let mut kind: Option<String> = None;
    for edge in catalog["transitions"]
        .as_array()
        .ok_or("transition registry required")?
    {
        let from = source::text(edge, "from")?;
        let to = source::text(edge, "to")?;
        version(from, false)?;
        version(to, false)?;
        if !edges.insert((from.to_string(), to.to_string())) {
            return Err("duplicate registered Profile transition".into());
        }
        if from == source_version && to == target_version {
            kind = Some(source::text(edge, "kind")?.into());
        }
    }
    let mut historic_keys = BTreeSet::new();
    let mut legacy_target_filename = Value::Null;
    let mut evidence_refs = Vec::new();
    if let Some(bridges) = catalog.get("historical_bridges").and_then(Value::as_array) {
        for bridge in bridges {
            let declared = source::text(bridge, "declared_version")?;
            let revision = source::text(bridge, "specification_revision")?;
            let target = source::text(bridge, "target_contract")?;
            version(target, false)?;
            if !historic_keys.insert((declared.to_string(), revision.to_string())) {
                return Err("duplicate historical Profile source binding".into());
            }
            if declared == source_version
                && target == target_version
                && input["source_revision"].as_str() == Some(revision)
            {
                if kind.is_some() {
                    return Err("ambiguous direct and historical Profile transition".into());
                }
                kind = Some(source::text(bridge, "transition_kind")?.into());
                legacy_target_filename = bridge
                    .get("legacy_target_filename")
                    .cloned()
                    .unwrap_or(Value::Null);
                evidence_refs.push(revision.to_string());
            }
        }
    }
    let declared = input.get("declared_transition").and_then(Value::as_str);
    let allowed = kind
        .as_deref()
        .is_some_and(|k| declared.is_none_or(|d| d == k));
    let output = json!({
        "allowed":allowed,
        "transition_kind":if allowed {kind} else {None},
        "evidence_refs":if allowed {evidence_refs} else {Vec::new()},
        "intermediate_materialization_allowed":false,
        "legacy_target_filename":if allowed {legacy_target_filename} else {Value::Null}
    });
    check(SCHEMA, "transition_output", &output)?;
    Ok(output)
}
