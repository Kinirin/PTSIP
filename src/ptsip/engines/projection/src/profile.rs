//! Profile metadata is a projection of explicitly selected identities.
//! Contract selection, lifecycle admission and identity validation have their own engines.
use serde_json::{json, Value};

const SCHEMA: &str = include_str!("schemas/profile.schema.json");

pub fn metadata(input: &Value) -> Result<Value, String> {
    let schema: Value = serde_json::from_str(SCHEMA).map_err(|error| error.to_string())?;
    jsonschema::validate(&schema["$defs"]["metadata_input"], input)
        .map_err(|error| format!("Profile metadata input: {error}"))?;
    let version = input["profile_contract_version"]
        .as_str()
        .ok_or("explicit Profile contract version required")?;
    let mut specification = json!({
        "source": input["specification_source"],
        "revision": input["specification_revision"],
    });
    let mut metadata = json!({"version": version});
    if version == "pp.1.01" {
        specification["family"] = input["specification_family"].clone();
    } else {
        metadata["revision"] = input
            .get("user_revision")
            .cloned()
            .unwrap_or_else(|| json!("Rev.0001"));
        metadata["profile_role"] = input
            .get("profile_role")
            .cloned()
            .unwrap_or_else(|| json!("PROJECT"));
    }
    metadata["specification"] = specification;

    // JSON flow notation is valid YAML. Quoting every string also prevents a
    // caller's source identity from injecting new YAML keys or documents.
    let serialized_header = format!(
        "{}\n",
        serde_json::to_string(&json!({"ptsip": metadata})).map_err(|error| error.to_string())?
    );
    let output = json!({
        "metadata": metadata,
        "serialized_header": serialized_header,
        "schema_ref": input["schema_ref"],
        "evidence_refs": input.get("evidence_refs").cloned().unwrap_or_else(|| json!([])),
    });
    jsonschema::validate(&schema["$defs"]["metadata_output"], &output)
        .map_err(|error| format!("Profile metadata output: {error}"))?;
    Ok(output)
}
