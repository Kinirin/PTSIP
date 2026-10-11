use serde_json::{json, Value};
use std::io::{self, Read};
const CONTRACT: &str = include_str!("../contract.json");
const TRANSPORT: &str = include_str!("../transport.json");
const MANIFEST: &str = include_str!("../manifest.json");
mod native;
mod provider;
mod source;
fn evaluate(operation: &str, input: &Value) -> Result<Value, String> {
    let manifest: Value = serde_json::from_str(MANIFEST).map_err(|e| e.to_string())?;
    let contract: Value = serde_json::from_str(CONTRACT).map_err(|e| e.to_string())?;
    if operation == "storage.invoke" {
        return native::invoke(input, &manifest, &contract);
    }
    provider::mediate(input, &manifest, &contract)
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
"storage.invoke" => "MEDIATE_REGISTERED_PROVIDER_AND_CONSUME_CURRENT_CALLER_ADMITTED_AUTHORITY_BEFORE_ISOLATED_NATIVE_PROVIDER_INVOCATION",

      "storage.mediate" => "VERIFY_STORAGE_OWNER_PROVIDER_AND_CURRENT_TWO_SIDED_COMPATIBILITY_EVIDENCE_THEN_MEDIATION_PLAN",
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
                let request = source::parse(&bytes)?;
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
    fn compatible_flag_cannot_substitute_for_provider_evidence() {
        assert!(dispatch(&request(json!({"capability_id":"c","operation_id":"read","contract_version":"1","providers":[{"compatible":true}]}),"storage.mediate")).is_err());
    }
}
