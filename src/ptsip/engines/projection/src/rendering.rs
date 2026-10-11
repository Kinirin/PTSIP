use crate::source::{self, Root};
use serde_json::{json, Map, Value};

pub fn render(input: &Value) -> Result<Value, String> {
    let mut root = Root::new(source::text(input, "root")?)?;
    let snapshot = root.file(&input["snapshot_ref"])?;
    if snapshot["semantic_owner"] != "CONTEXT" {
        return Err("registered Context snapshot required".into());
    }
    let original = root.bound(&snapshot["source_ref"])?;
    if snapshot["source_revision"] != snapshot["source_ref"]["sha256"]
        || snapshot["context_id"] != original["id"]
        || snapshot["content"] != original["content"]
        || snapshot["constraints"] != original["constraints"]
    {
        return Err("Context snapshot/source semantics drift".into());
    }
    source::inline(&snapshot["constraints"])?;
    jsonschema::validate(&snapshot["constraints"], &snapshot["content"])
        .map_err(|e| e.to_string())?;
    let definition = root.bound(&input["definition_ref"])?;
    source::schema("projection_definition", &definition)?;
    source::inline(&definition["output_schema"])?;
    let mut fields = Map::new();
    let mut lines = String::new();
    for field in definition["fields"]
        .as_array()
        .ok_or("projection fields required")?
    {
        let name = source::text(field, "name")?;
        let pointer = field["pointer"].as_str().ok_or("JSON pointer required")?;
        if !valid_pointer(pointer) || name.contains(['\r', '\n']) {
            return Err("invalid projection field or RFC 6901 pointer".into());
        }
        let value = snapshot["content"]
            .pointer(pointer)
            .ok_or("unknown projection pointer")?;
        if fields.insert(name.into(), value.clone()).is_some() {
            return Err("ambiguous output field".into());
        }
        lines.push_str(name);
        lines.push(':');
        lines.push_str(&serde_json::to_string(value).map_err(|e| e.to_string())?);
        lines.push('\n');
    }
    let content = if definition["format"] == "JSON_OBJECT" {
        Value::Object(fields)
    } else {
        Value::String(lines)
    };
    jsonschema::validate(&definition["output_schema"], &content)
        .map_err(|e| format!("Projection output schema: {e}"))?;
    root.finish()?;
    Ok(
        json!({"context_id":snapshot["context_id"],"source_revision":snapshot["source_revision"],"definition_revision":input["definition_ref"]["sha256"],"format":definition["format"],"content":content,"semantic_owner":"PROJECTION","source_semantics_owner":"CONTEXT"}),
    )
}

fn valid_pointer(pointer: &str) -> bool {
    if !pointer.is_empty() && !pointer.starts_with('/') {
        return false;
    }
    let mut chars = pointer.chars();
    while let Some(c) = chars.next() {
        if c == '~' && !matches!(chars.next(), Some('0' | '1')) {
            return false;
        }
    }
    true
}
