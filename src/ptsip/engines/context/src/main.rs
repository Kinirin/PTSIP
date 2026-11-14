use serde_json::{json, Value};
const CONTRACT: &str = include_str!("../contract.json");
const TRANSPORT: &str = include_str!("../transport.json");
const MANIFEST: &str = include_str!("../manifest.json");
mod cli;
mod loading;
mod source;
fn evaluate(operation: &str, input: &Value) -> Result<Value, String> {
    if operation == "context.order-dependencies" {
        return loading::order_dependencies(input);
    }
    if operation == "context.load" {
        return loading::load(input);
    }
    Ok(
        json!({"context_id":input["context_id"],"content":input["content"],"source_revision":input["source_revision"],"semantic_owner":"CONTEXT"}),
    )
}
fn dispatch(request: &Value) -> Result<Value, String> {
    let contract: Value = serde_json::from_str(CONTRACT).map_err(|e| e.to_string())?;
    let manifest: Value = serde_json::from_str(MANIFEST).map_err(|e| e.to_string())?;
    let transport: Value = serde_json::from_str(TRANSPORT).map_err(|e| e.to_string())?;
    jsonschema::validate(&transport["request_schema"], request).map_err(|e| e.to_string())?;
    for (field, expected) in [
        ("engine_id", &manifest["engine_id"]),
        ("engine_version", &manifest["implementation_version"]),
        ("contract_id", &contract["contract_id"]),
        ("contract_version", &contract["version"]),
    ] {
        if &request[field] != expected {
            return Err(format!("current identity/version mismatch: {field}"));
        }
    }
    let operation = request["operation_id"]
        .as_str()
        .ok_or("operation identity required")?;
    let definition = contract["operations"]
        .as_array()
        .ok_or("closed contract operations required")?
        .iter()
        .find(|o| o["operation_id"] == operation)
        .ok_or("unsupported operation")?;
    jsonschema::validate(&definition["typed_input"], &request["input"])
        .map_err(|e| e.to_string())?;
    let algorithm = match operation {
        "context.load" => {
            "VERIFY_BOUND_JSON_SOURCE_AND_INLINE_CONSTRAINTS_THEN_PRESERVE_CONTEXT_CONTENT"
        }

        "context.order-dependencies" => "TOPOLOGICAL_ORDER_OF_EXPLICIT_DEPENDENCIES_WITH_LEXICOGRAPHIC_TIES",
        "context.snapshot" => "SNAPSHOT_EXPLICIT_SOURCE_WITHOUT_REINTERPRETATION",
        _ => return Err("unsupported registered adapter operation".into()),
    };
    if definition["machine_semantics"]["algorithm"] != algorithm {
        return Err("unsupported neutral module semantics".into());
    }
    let output = evaluate(operation, &request["input"])?;
    jsonschema::validate(&definition["typed_output"], &output)
        .map_err(|e| format!("output contract violation: {e}"))?;
    Ok(output)
}
fn main() {
    cli::run();
}
#[cfg(test)]
mod tests {
    use super::*;
    fn request(input: Value, operation: &str) -> Value {
        let manifest: Value = serde_json::from_str(MANIFEST).unwrap();
        let c: Value = serde_json::from_str(CONTRACT).unwrap();
        json!({"engine_id":manifest["engine_id"],"engine_version":manifest["implementation_version"],"contract_id":c["contract_id"],"contract_version":c["version"],"operation_id":operation,"input":input})
    }
    #[test]
    fn unknown_and_wrong_identity_fail_closed() {
        let r = request(json!({}), "UNKNOWN");
        assert!(dispatch(&r).is_err());
        let mut r = r;
        r["engine_version"] = json!("999.0.0");
        assert!(dispatch(&r).is_err());
    }
    #[test]
    fn invalid_input_and_unknown_fields_fail_closed() {
        let c: Value = serde_json::from_str(CONTRACT).unwrap();
        let op = c["operations"][0]["operation_id"].as_str().unwrap();
        let mut r = request(json!({}), op);
        assert!(dispatch(&r).is_err());
        r["grant_write"] = json!(true);
        assert!(dispatch(&r).is_err());
    }
    #[test]
    fn source_identity_and_content_preserved() {
        let v = json!({"context_id":"c","content":{"x":1},"source_revision":"r1"});
        let out = dispatch(&request(v.clone(), "context.snapshot")).unwrap();
        assert_eq!(out["content"], v["content"]);
        assert_eq!(out["semantic_owner"], "CONTEXT");
    }
}
