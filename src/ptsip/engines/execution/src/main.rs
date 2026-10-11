use serde_json::{json, Value};
use std::io::{self, Read};
const CONTRACT: &str = include_str!("../contract.json");
const TRANSPORT: &str = include_str!("../transport.json");
const MANIFEST: &str = include_str!("../manifest.json");
mod authority;
mod effect;
mod invocation;
mod source;
fn evaluate(operation: &str, input: &Value) -> Result<Value, String> {
    if operation == "execution.invoke" {
        let manifest: Value = serde_json::from_str(MANIFEST).map_err(|e| e.to_string())?;
        let contract: Value = serde_json::from_str(CONTRACT).map_err(|e| e.to_string())?;
        return invocation::invoke(input, &manifest, &contract);
    }
    if operation == "execution.assess-stop" {
        if input["ack"]["correlation_id"] != input["result"]["correlation_id"] {
            return Err("stop correlation mismatch".into());
        }
        if input["ack"]["status"] == "REJECTED" && input["result"]["status"] == "STOP_CONFIRMED" {
            return Err("conflicting stop records".into());
        }
        return Ok(
            json!({"stop_confirmed":input["result"]["status"]=="STOP_CONFIRMED","original_operation_succeeded":false,"effects_reversed":false}),
        );
    }
    let attempt = &input["attempt"];
    let effects = input["effects"]
        .as_array()
        .ok_or("effect evidence required")?;
    if effects
        .iter()
        .any(|e| e["attempt_id"] != attempt["attempt_id"])
    {
        return Err("effect attempt correlation mismatch".into());
    }
    let refs = attempt["effect_evidence_refs"]
        .as_array()
        .ok_or("effect reference closure required")?;
    let mut observations = Vec::new();
    for record in effects {
        if !refs.iter().any(|r| r == &record["source_ref"]) {
            return Err("attempt/effect reference closure mismatch".into());
        }
        observations
            .push(effect::observation(input, record).unwrap_or_else(|_| "UNVERIFIED".into()));
    }
    let effect = if observations.is_empty() {
        "UNKNOWN"
    } else if observations.iter().any(|o| o == "OBSERVED_EFFECT") {
        "OBSERVED_EFFECT"
    } else if observations.iter().all(|o| o == "VERIFIED_NO_EFFECT") {
        "VERIFIED_NO_EFFECT"
    } else {
        "UNVERIFIED"
    };
    if attempt["cause"] == "RESOURCE_EXHAUSTION" && attempt["status"] != "FAILED" {
        return Err("resource exhaustion requires failed attempt".into());
    }
    Ok(
        json!({"attempt_id":attempt["attempt_id"],"attempt_status":attempt["status"],"effect_status":effect,"execution_authorized":false}),
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
"execution.invoke" => "CONSUME_CURRENT_CALLER_ADMITTED_LOCAL_AUTHORITY_AND_INVOKE_EXACT_BOUND_NATIVE_REQUEST_WITH_BOUNDED_IO_REVOCATION_AND_DIRECT_CHILD_STOP",

        "execution.assess-attempt" => {
            "VERIFY_CURRENT_REFERENCED_EFFECT_PROVENANCE_WITHOUT_INFERRING_ABSENCE_FROM_ATTEMPT_FAILURE"
        }
        "execution.assess-stop" => {
            "ACK_NOT_STOP_CONFIRMATION_AND_STOP_NOT_ORIGINAL_SUCCESS_OR_EFFECT_REVERSAL"
        }
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
    fn exhaustion_does_not_imply_no_effect() {
        let v = json!({"root":".","attempt":{"attempt_id":"a","status":"FAILED","cause":"RESOURCE_EXHAUSTION","effect_evidence_refs":[]},"effects":[]});
        assert_eq!(
            dispatch(&request(v, "execution.assess-attempt")).unwrap()["effect_status"],
            "UNKNOWN"
        );
    }
    #[test]
    fn ack_is_not_stop_or_success() {
        let v = json!({"ack":{"correlation_id":"s","status":"ACCEPTED","cause":"cancel"},"result":{"correlation_id":"s","attempt_id":"a","status":"PENDING","effect_evidence_refs":[]}});
        let out = dispatch(&request(v, "execution.assess-stop")).unwrap();
        assert_eq!(out["stop_confirmed"], false);
        assert_eq!(out["original_operation_succeeded"], false);
    }
}
