//! Read-only exact lookup; the Validation engine owns relation validity.
use crate::profile;
use ptsip_engine_discovery::source::{self, Root};
use serde_json::{json, Value};
use std::collections::BTreeMap;

pub const ALGORITHM: &str = "DISCOVER_OPAQUE_RULE_IDS_AND_RESOLVE_EXACT_RECORD_WITHOUT_AUTHORITY";
const SCHEMA: &str = include_str!("schemas/rule-records.schema.json");
const RULE_PROFILE: &str = ".ptsip/profiles/rules.ptsip.yaml";

fn document(
    root: &Root,
    reference: &str,
    snapshots: &mut Vec<(String, String)>,
) -> Result<Value, String> {
    let raw = root.bytes(reference, 2_097_152)?;
    snapshots.push((reference.into(), source::hash(&raw)));
    source::parse_yaml(&raw)
}

fn linked_records(input: &Value) -> Result<(Vec<Value>, Option<String>), String> {
    let root = Root::new(source::text(input, "repository_root")?)?;
    let mut snapshots = Vec::new();
    let mut catalog = None;
    let main = match input.get("main_payload") {
        Some(payload) => payload.clone(),
        None => {
            catalog = profile::load_catalog(&root, None)?;
            let Some(selected) = catalog.as_ref() else {
                return Ok((Vec::new(), None));
            };
            snapshots.push((selected.reference.clone(), selected.digest.clone()));
            let reference = profile::selected_resource(selected, None)?;
            document(&root, &reference, &mut snapshots)?
        }
    };
    if !main.is_object() {
        return Err("main Profile must be an object".into());
    }
    let link = main
        .get("ptsip")
        .filter(|value| value.is_object())
        .and_then(|value| value.get("rule_profile"))
        .filter(|value| !value.is_null());
    let records = if let Some(link) = link {
        if link.as_str() != Some("rules.ptsip.yaml") {
            return Err("unregistered linked rule Profile resource".into());
        }
        if catalog.is_none() {
            catalog = profile::load_catalog(&root, None)?;
            if let Some(selected) = catalog.as_ref() {
                snapshots.push((selected.reference.clone(), selected.digest.clone()));
            }
        }
        let catalog = catalog
            .as_ref()
            .ok_or("linked rule Profile requires a registered local catalog")?;
        let matches: Vec<_> = catalog.payload["profiles"]
            .as_array()
            .ok_or("Profile catalog entries required")?
            .iter()
            .filter(|row| row["id"] == "rules" || row["resource"] == "rules.ptsip.yaml")
            .collect();
        if matches.len() != 1
            || matches[0]["id"] != "rules"
            || matches[0]["resource"] != "rules.ptsip.yaml"
        {
            return Err("linked rule Profile catalog registration missing or ambiguous".into());
        }
        let payload = document(&root, RULE_PROFILE, &mut snapshots)?;
        let definitions: Value = source::parse(SCHEMA.as_bytes())?;
        jsonschema::validate(&definitions["$defs"]["rule_profile"], &payload)
            .map_err(|error| format!("linked rule Profile wrapper: {error}"))?;
        payload["rules"]
            .as_array()
            .ok_or("rule Profile rules array required")?
            .clone()
    } else {
        Vec::new()
    };
    for (reference, expected) in snapshots {
        if source::hash(&root.bytes(&reference, 2_097_152)?) != expected {
            return Err("linked rule Profile source changed during discovery".into());
        }
    }
    Ok((records, link.map(|_| RULE_PROFILE.into())))
}

pub fn resolve(input: &Value) -> Result<Value, String> {
    let definitions: Value = source::parse(SCHEMA.as_bytes())?;
    jsonschema::validate(&definitions["$defs"]["input"], input).map_err(|e| e.to_string())?;
    let (records, source_ref) = match input.get("records") {
        Some(records) => (
            records
                .as_array()
                .ok_or("explicit rule records required")?
                .clone(),
            None,
        ),
        None => linked_records(input)?,
    };
    let mut by_id = BTreeMap::new();
    for record in &records {
        let id = source::text(record, "ID")?;
        if by_id.insert(id, record).is_some() {
            return Err("duplicate exact rule ID".into());
        }
    }
    let selected = input
        .get("rule_id")
        .filter(|value| !value.is_null())
        .map(|v| v.as_str().ok_or("rule_id must be a string"))
        .transpose()?;
    let record = selected.and_then(|id| by_id.get(id).copied());
    let status = match selected {
        None => "DISCOVERED",
        Some(_) if record.is_some() => "RESOLVED",
        _ => "UNRESOLVED",
    };
    Ok(
        json!({"rule_ids":by_id.keys().collect::<Vec<_>>(),"rule_id":selected,"record":record,"status":status,
        "records_sha256":source::hash(&serde_json::to_vec(&records).map_err(|e|e.to_string())?),"records":records,"source_ref":source_ref,
        "validation_verdict":"NOT_ASSERTED","normative_authority":false,"execution_authorized":false,"identity_rewritten":false}),
    )
}
