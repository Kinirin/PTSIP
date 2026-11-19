use crate::profile::check;
use serde_json::{json, Value};
use std::collections::{BTreeMap, BTreeSet};

const SCHEMA: &str = include_str!("schemas/rule-records.schema.json");

fn diagnostic(diagnostics: &mut Vec<Value>, code: &str, id: &str, message: String) {
    diagnostics.push(json!({"code":code,"record_id":id,"message":message}));
}

/// Validate the current independent record wrapper and explicit graph only.
/// IDs and rule_value contents remain opaque; no legacy prefixes or depth are inferred.
pub fn validate(input: &Value) -> Result<Value, String> {
    check(SCHEMA, "input", input)?;
    let records = input["records"].as_array().ok_or("rule records required")?;
    let mut by_id = BTreeMap::new();
    let mut diagnostics = Vec::new();
    for record in records {
        let id = record["ID"].as_str().ok_or("rule ID required")?;
        if by_id.insert(id.to_string(), record).is_some() {
            diagnostic(
                &mut diagnostics,
                "DUPLICATE_ID",
                id,
                "An exact rule ID occurs more than once.".into(),
            );
        }
    }
    for (id, record) in &by_id {
        if let Some(parent) = record.get("parent_id").and_then(Value::as_str) {
            if !by_id.contains_key(parent) {
                diagnostic(
                    &mut diagnostics,
                    "MISSING_PARENT",
                    id,
                    format!("Parent ID does not resolve: {parent}"),
                );
            }
        }
        if let Some(pointer) = record
            .get("rule_value")
            .and_then(Value::as_object)
            .and_then(|v| v.get("root_id"))
        {
            if pointer.is_null() {
                continue;
            }
            match pointer.as_str().and_then(|target| by_id.get(target)) {
                Some(target) if target["parent_id"].as_str() == Some(id.as_str()) => {}
                _ => diagnostic(
                    &mut diagnostics,
                    "INVALID_CHILD_POINTER",
                    id,
                    "root_id must resolve exactly to a direct child record.".into(),
                ),
            }
        }
    }
    let mut reported_cycles = BTreeSet::new();
    let mut visited = BTreeSet::new();
    for id in by_id.keys() {
        if visited.contains(id) {
            continue;
        }
        let mut positions = BTreeMap::new();
        let mut path = Vec::new();
        let mut next = Some(id.as_str());
        while let Some(current) = next {
            if visited.contains(current) {
                break;
            }
            if let Some(position) = positions.get(current).copied() {
                let mut cycle: Vec<String> = path[position..].to_vec();
                cycle.sort();
                if reported_cycles.insert(cycle.clone()) {
                    diagnostic(
                        &mut diagnostics,
                        "PARENT_CYCLE",
                        &cycle[0],
                        format!("Parent relations form a cycle: {}", cycle.join(", ")),
                    );
                }
                break;
            }
            let Some(record) = by_id.get(current) else {
                break;
            };
            positions.insert(current.to_string(), path.len());
            path.push(current.to_string());
            next = record.get("parent_id").and_then(Value::as_str);
        }
        visited.extend(path);
    }
    let ids: Vec<_> = by_id.keys().cloned().collect();
    let output = json!({"valid":diagnostics.is_empty(),"diagnostics":diagnostics,"record_ids":ids,"records":records});
    check(SCHEMA, "output", &output)?;
    Ok(output)
}
