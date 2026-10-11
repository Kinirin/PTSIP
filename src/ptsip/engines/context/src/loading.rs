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
