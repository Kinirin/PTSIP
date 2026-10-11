use serde_json::{json, Value};
use std::io::{self, Read};
const CONTRACT: &str = include_str!("../contract.json");
const TRANSPORT: &str = include_str!("../transport.json");
const MANIFEST: &str = include_str!("../manifest.json");
mod composition;
mod handoff;
fn evaluate(operation: &str, input: &Value) -> Result<Value, String> {
    if operation == "validation.verify-context-handoff" {
        return handoff::verify_context(input);
    }
    if operation == "validation.verify-handoff" {
        let contract: Value = serde_json::from_str(CONTRACT).map_err(|e| e.to_string())?;
        return handoff::verify(input, &contract);
    }
    if operation == "validation.compose" {
        return composition::validate(input);
    }
    fn closed_schema(v: &Value) -> bool {
        match v {
            Value::Object(o) => {
                !o.contains_key("$ref")
                    && !o.contains_key("$dynamicRef")
                    && o.values().all(closed_schema)
            }
            Value::Array(a) => a.iter().all(closed_schema),
            _ => true,
        }
    }
    if !closed_schema(&input["schema"]) {
        return Err(
            "external or unresolved schema references are not supported by this capability".into(),
        );
    }
    let validator = jsonschema::validator_for(&input["schema"]).map_err(|e| e.to_string())?;
    let diagnostics: Vec<String> = validator
        .iter_errors(&input["instance"])
        .map(|e| e.to_string())
        .collect();
    Ok(
        json!({"valid":diagnostics.is_empty(),"diagnostics":diagnostics,"ptsip_conformance_verdict":"NOT_ASSERTED"}),
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
        "validation.contract" => {
            "INDEPENDENT_INLINE_JSON_SCHEMA_2020_12_VALIDATION_NO_EXTERNAL_REFERENCE"
        }
        "validation.schema" => {
            "INDEPENDENT_INLINE_JSON_SCHEMA_2020_12_VALIDATION_NO_EXTERNAL_REFERENCE"
        }
        "validation.typed-io" => {
            "INDEPENDENT_INLINE_JSON_SCHEMA_2020_12_VALIDATION_NO_EXTERNAL_REFERENCE"
        }
        "validation.precondition" => {
            "INDEPENDENT_INLINE_JSON_SCHEMA_2020_12_VALIDATION_NO_EXTERNAL_REFERENCE"
        }
        "validation.case" => {
            "INDEPENDENT_INLINE_JSON_SCHEMA_2020_12_VALIDATION_NO_EXTERNAL_REFERENCE"
        }
        "validation.compose" => "EXPLICIT_OPTIONAL_INDEPENDENT_VALIDATION_CHECK_DAG_WITHOUT_IMPLICIT_VPMS_EQUIVALENCE",
        "validation.verify-handoff" => "VERIFY_CURRENT_VERSIONED_VPMS_RELATIONS_AND_FULL_RESPONSIBILITY_CLASSIFICATION_WITHOUT_PUBLIC_API_CUTOVER",
        "validation.verify-context-handoff" => "VERIFY_EXPLICIT_CONTEXT_SOURCE_AND_PROJECTION_OUTPUT_OWNERS_WITH_PRESERVED_PUBLIC_API",
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
    let args: Vec<String> = std::env::args().skip(1).collect();
    if args == ["--manifest"] {
        println!("{MANIFEST}");
        return;
    }
    let mut bytes = Vec::new();
    let result = if !args.is_empty() {
        Err("unsupported invocation arguments".into())
    } else {
        io::stdin()
            .take(1048577)
            .read_to_end(&mut bytes)
            .map_err(|e| e.to_string())
            .and_then(|_| {
                if bytes.len() > 1048576 {
                    return Err("request exceeds registered byte limit".into());
                }
                let request: Value = serde_json::from_slice(&bytes).map_err(|e| e.to_string())?;
                dispatch(&request)
            })
    };
    let manifest: Value = serde_json::from_str(MANIFEST).expect("compiled manifest must be valid");
    let contract: Value = serde_json::from_str(CONTRACT).expect("compiled contract must be valid");
    let (status, output, error) = match result {
        Ok(value) => ("PASS", value, Value::Null),
        Err(error) => ("FAIL", Value::Null, json!(error)),
    };
    println!(
        "{}",
        json!({"schema_version":"ptsip-engine-operation-result/v1","engine_id":manifest["engine_id"],"engine_version":manifest["implementation_version"],"contract_id":contract["contract_id"],"contract_version":contract["version"],"status":status,"output":output,"error":error,"normative_authority":false,"execution_authorized":false,"evidence_authentication":"NOT_PROVIDED_BY_RESULT"})
    );
    if status == "FAIL" {
        std::process::exit(2);
    }
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
    fn independent_capabilities_validate_without_conformance_claim() {
        for op in [
            "validation.contract",
            "validation.schema",
            "validation.typed-io",
            "validation.precondition",
            "validation.case",
        ] {
            let v = json!({"schema":{"type":"integer"},"instance":"bad"});
            let out = dispatch(&request(v, op)).unwrap();
            assert_eq!(out["valid"], false);
            assert_eq!(out["ptsip_conformance_verdict"], "NOT_ASSERTED");
        }
        let out = dispatch(&request(
            json!({"schema":{"type":"integer"},"instance":3}),
            "validation.schema",
        ))
        .unwrap();
        assert_eq!(out["valid"], true);
    }
    #[test]
    fn schema_reference_cannot_trigger_external_resolution() {
        assert!(dispatch(&request(
            json!({"schema":{"$ref":"https://example.invalid/schema"},"instance":{}}),
            "validation.schema"
        ))
        .is_err());
    }
}
