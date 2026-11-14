use crate::source::{self, Root};
use serde_json::{json, Value};

pub fn load(input: &Value) -> Result<Value, String> {
    let mut root = Root::new(source::text(input, "root")?)?;
    let value = root.bound(&input["source_ref"])?;
    source::schema("source", &value)?;
    source::inline(&value["constraints"])?;
    jsonschema::validate(&value["constraints"], &value["content"])
        .map_err(|e| format!("source constraints: {e}"))?;
    root.finish()?;
    Ok(
        json!({"context_id":value["id"],"content":value["content"],"source_revision":input["source_ref"]["sha256"],"source_ref":input["source_ref"],"constraints":value["constraints"],"semantic_owner":"CONTEXT"}),
    )
}

/// Order the explicitly selected graph without inspecting any other engine.
pub fn order_dependencies(input: &Value) -> Result<Value, String> {
    use std::collections::{BTreeMap, BTreeSet};

    let mut nodes = BTreeMap::new();
    for node in input["nodes"].as_array().ok_or("nodes required")? {
        let id = node.as_str().ok_or("node ID required")?;
        if nodes.insert(id, (0usize, Vec::new())).is_some() {
            return Err("duplicate dependency node".into());
        }
    }
    for edge in input["edges"].as_array().ok_or("edges required")? {
        let provider = source::text(edge, "provider")?;
        let consumer = source::text(edge, "consumer")?;
        nodes.get_mut(consumer).ok_or("unknown consumer node")?.0 += 1;
        nodes
            .get_mut(provider)
            .ok_or("unknown provider node")?
            .1
            .push(consumer);
    }
    let mut ready: BTreeSet<_> = nodes
        .iter()
        .filter_map(|(id, (count, _))| (*count == 0).then_some(*id))
        .collect();
    let mut ordered = Vec::with_capacity(nodes.len());
    while let Some(id) = ready.pop_first() {
        ordered.push(id);
        let (_, consumers) = nodes.remove(id).ok_or("unknown ready node")?;
        for consumer in consumers {
            let (count, _) = nodes.get_mut(consumer).ok_or("repeated dependency node")?;
            *count -= 1;
            if *count == 0 {
                ready.insert(consumer);
            }
        }
    }
    if !nodes.is_empty() {
        return Err("dependency cycle".into());
    }
    Ok(json!({"ordered_ids": ordered, "semantic_owner":"CONTEXT"}))
}
