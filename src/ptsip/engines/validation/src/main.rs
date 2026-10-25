use serde_json::{json, Value};
use std::io::{self, Read};
const CONTRACT: &str = include_str!("../contract.json");
const TRANSPORT: &str = include_str!("../transport.json");
const MANIFEST: &str = include_str!("../manifest.json");
mod cases;
mod composition;
mod handoff;
mod retrieval;
mod schema;
#[cfg(test)]
mod schema_reference_tests;
mod source;
mod rbac;
fn evaluate(operation: &str, input: &Value) -> Result<Value, String> {
    if matches!(
        operation,
        "validation.prepare-cases" | "validation.select-cases" | "validation.prepare-composition"
    ) {
        return cases::prepare(operation, input);
    }
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
    let (diagnostics, evidence) = schema::validate_with_resolution(
        &input["schema"],
        &input["instance"],
        input.get("schema_resources"),
        input.get("schema_resolution"),
    )?;
    Ok(
        json!({"valid":diagnostics.is_empty(),"diagnostics":diagnostics,"schema_resources_used":evidence,"ptsip_conformance_verdict":"NOT_ASSERTED"}),
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
"validation.prepare-cases" => "VERIFY_CURRENT_CASE_CATALOG_REFERENCE_CLOSURE_AND_INDEPENDENT_INLINE_CASE_CONTRACTS",
"validation.select-cases" => "SELECT_EXPLICIT_CASE_IDS_OR_CURRENT_REGISTERED_SELECTOR_WITH_NO_EXECUTION",
"validation.prepare-composition" => "VERIFY_ALL_SELECTED_CASE_ADAPTERS_AND_CONTRACT_INPUTS_BEFORE_ANY_INVOCATION",

        "validation.contract" => {
            "INDEPENDENT_JSON_SCHEMA_VALIDATION_WITH_SCOPED_REFERENCE_RESOLUTION"
        }
        "validation.schema" => {
            "INDEPENDENT_JSON_SCHEMA_VALIDATION_WITH_SCOPED_REFERENCE_RESOLUTION"
        }
        "validation.typed-io" => {
            "INDEPENDENT_JSON_SCHEMA_VALIDATION_WITH_SCOPED_REFERENCE_RESOLUTION"
        }
        "validation.precondition" => {
            "INDEPENDENT_JSON_SCHEMA_VALIDATION_WITH_SCOPED_REFERENCE_RESOLUTION"
        }
        "validation.case" => {
            "INDEPENDENT_JSON_SCHEMA_VALIDATION_WITH_SCOPED_REFERENCE_RESOLUTION"
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
    if args == ["--help"] || args == ["-h"] {
        println!("PTSIP Validation Engine 0.3.4 / capability contract 4.0.1\nInput: one versioned JSON request on stdin.\nSupports internal $ref (JSON Pointer, anchors, recursive schemas), external $id references and $dynamicRef/$dynamicAnchor.\nschema_resources: explicit local resources.\nschema_resolution.directories: recursively discover JSON schemas inside explicitly permitted roots.\nschema_resolution.allow_network + allowed_origins: retrieve missing HTTP(S) references from exact permitted origins; redirects and ambient proxies disabled.\nOptional base_uri resolves relative references for anonymous input schemas.\nResults include schema_resources_used (URI, source, SHA256).\nLimits: 128 schema documents, 1 MiB per acquired document, 8 MiB total, 16384 schema nodes, 5s per HTTP request and 20s acquisition budget per schema evaluation.\nGuide: https://github.com/Kinirin/PTSIP/blob/main/docs/validation-schema-references.md");
        return;
    }
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

    #[test]
    fn registered_operations_and_composition_accept_explicit_schema_resources() {
        let resources = json!([{"$id":"urn:registered:string","type":"string"}]);
        for op in [
            "validation.contract",
            "validation.schema",
            "validation.typed-io",
            "validation.precondition",
            "validation.case",
        ] {
            let input = json!({"schema":{"$ref":"urn:registered:string"},"instance":"ok","schema_resources":resources});
            assert_eq!(dispatch(&request(input, op)).unwrap()["valid"], true);
        }
        let input = json!({"checks":[{"check_id":"a","operation_id":"validation.schema","schema":{"$ref":"urn:registered:string"},"instance":4,"depends_on":[]}],"schema_resources":resources});
        let output = dispatch(&request(input, "validation.compose")).unwrap();
        assert_eq!(output["valid_all"], false);
        assert_eq!(output["ptsip_conformance_verdict"], "NOT_ASSERTED");
    }
}
