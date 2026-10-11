use serde_json::{json, Value};
use std::io::{self, Read};
const CONTRACT: &str = include_str!("../contract.json");
const TRANSPORT: &str = include_str!("../transport.json");
const MANIFEST: &str = include_str!("../manifest.json");
fn evaluate(_operation: &str, input: &Value) -> Result<Value, String> {
    use std::collections::{BTreeMap, BTreeSet};
    let mut deps: BTreeMap<String, BTreeSet<String>> = BTreeMap::new();
    for node in input["nodes"].as_array().ok_or("nodes required")? {
        let id = node["id"]
            .as_str()
            .ok_or("node identity required")?
            .to_owned();
        let list = node["dependencies"]
            .as_array()
            .ok_or("dependencies required")?;
        let values: BTreeSet<String> = list
            .iter()
            .map(|v| v.as_str().unwrap_or("").to_owned())
            .collect();
        if values.len() != list.len() || deps.insert(id, values).is_some() {
            return Err("duplicate identity or dependency".into());
        }
    }
    if deps
        .values()
        .any(|ds| ds.iter().any(|id| !deps.contains_key(id)))
    {
        return Err("unknown dependency".into());
    }
    let mut ordered = Vec::new();
    while !deps.is_empty() {
        let ready = deps
            .iter()
            .find(|(_, d)| d.is_empty())
            .map(|(id, _)| id.clone())
            .ok_or("dependency cycle")?;
        deps.remove(&ready);
        for d in deps.values_mut() {
            d.remove(&ready);
        }
        ordered.push(ready);
    }
    Ok(json!({"ordered_ids":ordered,"execution_authorized":false}))
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
        "planning.order" => {
            "EXPLICIT_DEPENDENCY_DAG_LEXICAL_TOPOLOGICAL_ORDER_FAIL_UNKNOWN_DUPLICATE_OR_CYCLE"
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
    fn dag_order_cycles_unknown_and_duplicate_nodes() {
        let v = json!({"nodes":[{"id":"b","dependencies":["a"]},{"id":"a","dependencies":[]}]});
        assert_eq!(
            dispatch(&request(v, "planning.order")).unwrap()["ordered_ids"],
            json!(["a", "b"])
        );
        for v in [
            json!({"nodes":[{"id":"a","dependencies":["b"]},{"id":"b","dependencies":["a"]}]}),
            json!({"nodes":[{"id":"a","dependencies":["missing"]}]}),
            json!({"nodes":[{"id":"a","dependencies":[]},{"id":"a","dependencies":[]}]}),
        ] {
            assert!(dispatch(&request(v, "planning.order")).is_err());
        }
    }
}
